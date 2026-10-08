// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/security"
	"einvoicing/internal/store"
	"einvoicing/internal/validate"
)

// SubmitOptions tunes a submission.
type SubmitOptions struct {
	// SkipFBRValidation posts without calling validateinvoicedata first.
	SkipFBRValidation bool
	// Background marks worker-initiated submissions.
	Background bool
}

// SubmissionError is returned when local validation blocks a submission.
type SubmissionError struct {
	Issues []validate.Issue
}

func (e *SubmissionError) Error() string {
	var msgs []string
	for _, i := range e.Issues {
		if i.Severity == validate.SevError {
			msgs = append(msgs, i.String())
		}
	}
	return "invoice has validation errors: " + strings.Join(msgs, "; ")
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := time.Duration(math.Pow(2, float64(attempt-1))) * 30 * time.Second
	if d > 30*time.Minute {
		d = 30 * time.Minute
	}
	return d
}

// Submit reports an invoice to FBR (postinvoicedata). It is safe to call
// repeatedly: accepted invoices are returned unchanged, invoices in flight or
// with an uncertain outcome are refused.
func (s *Service) Submit(ctx context.Context, a Actor, companyID, id int64, opts SubmitOptions) (*store.Invoice, error) {
	unlock := s.lockInvoice(id)
	defer unlock()

	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	inv, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	switch inv.Status {
	case domain.StatusAccepted, domain.StatusCancelled:
		return inv, nil
	case domain.StatusSubmitting:
		return nil, Invalid("invoice %s is already being submitted", inv.InternalNo)
	case domain.StatusUncertain:
		return nil, Invalid("the last submission of %s may have reached FBR; check IRIS and resolve it before resubmitting", inv.InternalNo)
	}

	// Pre-flight checks.
	var original *store.Invoice
	if inv.RefInvoiceID != nil {
		original, _ = s.Store.GetInvoice(ctx, companyID, *inv.RefInvoiceID)
	}
	res := s.ValidateLocal(ctx, c, inv, original)
	if res.HasErrors() {
		st := editableStatus(inv.Status)
		if inv.Status == domain.StatusQueued {
			// Make the row editable first so the issues are stored with it;
			// the operator needs them to correct an invoice already issued.
			_ = s.Store.SetStatus(ctx, inv.ID, domain.StatusRejected, "local validation failed", "")
			st = domain.StatusRejected
		}
		_ = s.Store.SetValidation(ctx, inv.ID, st, res.Issues, nil, "local validation failed")
		return nil, &SubmissionError{Issues: res.Issues}
	}
	if inv.Environment == domain.EnvProduction {
		// The licence governs issuing new invoices. A queued invoice was
		// already issued (it passed this check when first submitted), and the
		// law requires it to be reported, so it is never held back.
		if inv.Status != domain.StatusQueued {
			if err := s.Opts.License.CheckProduction(c.NTNCNIC); err != nil {
				return nil, Invalid("production submission blocked: %v", err)
			}
		}
		if !c.HasProductionToken {
			return nil, Invalid("no FBR production security token is configured for %s", c.Name)
		}
	}
	if inv.Environment == domain.EnvSandbox && !c.HasSandboxToken {
		return nil, Invalid("no FBR sandbox security token is configured for %s", c.Name)
	}

	payload := s.BuildPayload(c, inv)
	if inv.Environment != domain.EnvSandbox {
		payload.ScenarioID = ""
	}
	pj, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ph := security.SHA256Hex(string(pj))

	claimed, err := s.Store.ClaimForSubmission(ctx, inv.ID)
	if err != nil {
		return nil, err
	}
	if !claimed {
		return nil, Invalid("invoice %s is not in a submittable state", inv.InternalNo)
	}
	// From here the exchange with FBR and the recording of its outcome must
	// complete even if the caller goes away (browser closed, ERP timeout,
	// service stopping); the FBR client's own timeout bounds the calls.
	// Otherwise the invoice could stay SUBMITTING with FBR's answer lost.
	ctx = context.WithoutCancel(ctx)
	// The exact payload is stored before it is sent, so a crash mid-flight is
	// detected on restart and reconciled instead of silently re-posted.
	if err := s.Store.SetPayload(ctx, inv.ID, string(pj), ph); err != nil {
		_ = s.Store.SetStatus(ctx, inv.ID, domain.StatusQueued, "could not store payload: "+err.Error(), s.Now().UTC().Add(time.Minute).Format(time.RFC3339))
		return nil, err
	}
	inv.PayloadJSON, inv.PayloadHash = string(pj), ph

	cl, err := s.Client(ctx, c, inv.Environment, &inv.ID)
	if err != nil {
		_ = s.Store.SetStatus(ctx, inv.ID, domain.StatusRejected, err.Error(), "")
		return nil, err
	}

	attempt := inv.SubmitAttempts + 1
	if c.ValidateBeforePost && !opts.SkipFBRValidation {
		vresp, verr := cl.ValidateInvoice(ctx, &payload)
		if verr != nil {
			return s.handleTransportError(ctx, a, c, inv, verr, attempt, false)
		}
		if !vresp.IsValid() {
			return s.handleRejection(ctx, a, inv, vresp, "validateinvoicedata")
		}
	}

	resp, perr := cl.PostInvoice(ctx, &payload)
	if perr != nil {
		return s.handleTransportError(ctx, a, c, inv, perr, attempt, true)
	}
	if !resp.IsValid() {
		return s.handleRejection(ctx, a, inv, resp, "postinvoicedata")
	}
	if strings.TrimSpace(string(resp.InvoiceNumber)) == "" {
		_ = s.Store.SetStatus(ctx, inv.ID, domain.StatusUncertain, "FBR reported the invoice valid but returned no invoice number; verify on IRIS", "")
		s.Audit(ctx, a, companyID, "invoice.uncertain", "invoice", fmt.Sprint(inv.ID), "valid response without invoiceNumber")
		return s.Store.GetInvoice(ctx, companyID, inv.ID)
	}
	return s.accept(ctx, a, inv, string(resp.InvoiceNumber), string(resp.Dated), string(resp.ValidationResponse.StatusCode), resp.ValidationResponse.InvoiceStatuses)
}

func editableStatus(st domain.InvoiceStatus) domain.InvoiceStatus {
	if st.Editable() {
		if st == domain.StatusValidated {
			return domain.StatusDraft
		}
		return st
	}
	return domain.StatusDraft
}

// accept seals and stores an FBR acceptance.
func (s *Service) accept(ctx context.Context, a Actor, inv *store.Invoice, fbrNo, dated, statusCode string, statuses []fbr.InvoiceStatus) (*store.Invoice, error) {
	inv.FBRInvoiceNumber, inv.FBRDated, inv.FBRStatusCode = fbrNo, dated, statusCode
	if inv.FBRStatusCode == "" {
		inv.FBRStatusCode = "00"
	}
	bySNo := map[int]fbr.InvoiceStatus{}
	for _, st := range statuses {
		n, _ := strconv.Atoi(string(st.ItemSNo))
		bySNo[n] = st
	}
	for _, it := range inv.Items {
		if st, ok := bySNo[it.LineNo]; ok {
			it.FBRItemInvoiceNo, it.FBRStatusCode = string(st.InvoiceNo), string(st.StatusCode)
		} else {
			it.FBRItemInvoiceNo, it.FBRStatusCode = fmt.Sprintf("%s-%d", fbrNo, it.LineNo), "00"
		}
	}
	err := s.Store.Tx(ctx, func(q store.Querier) error {
		prev, err := store.LastSealHash(ctx, q, inv.CompanyID)
		if err != nil {
			return err
		}
		inv.PrevSealHash = prev
		inv.SealHash = sealHash(prev, inv)
		return store.MarkAccepted(ctx, q, inv, inv.Items)
	})
	if err != nil {
		// FBR has accepted the invoice; never lose that fact.
		s.Log.Error("FBR accepted invoice but saving failed", "invoice", inv.ID, "fbr", fbrNo, "err", err)
		_ = s.Store.SetStatus(ctx, inv.ID, domain.StatusUncertain, "FBR accepted the invoice as "+fbrNo+" but saving failed: "+err.Error(), "")
		return nil, err
	}
	s.Audit(ctx, a, inv.CompanyID, "invoice.accepted", "invoice", fmt.Sprint(inv.ID), map[string]any{"no": inv.InternalNo, "fbr": fbrNo,
		"env": inv.Environment, "total": inv.Totals.TotalValue.StringFixed(2), "seal": inv.SealHash})
	return s.Store.GetInvoice(ctx, inv.CompanyID, inv.ID)
}

func (s *Service) handleRejection(ctx context.Context, a Actor, inv *store.Invoice, resp *fbr.InvoiceResponse, op string) (*store.Invoice, error) {
	inv.FBRErrors = resp.Errors()
	if resp.ValidationResponse != nil {
		inv.FBRStatusCode = string(resp.ValidationResponse.StatusCode)
	}
	var msgs []string
	for _, e := range inv.FBRErrors {
		msgs = append(msgs, e.String())
	}
	inv.LastError = "FBR rejected the invoice (" + op + "): " + strings.Join(msgs, "; ")
	byLine := map[int]fbr.InvoiceStatus{}
	if resp.ValidationResponse != nil {
		for _, st := range resp.ValidationResponse.InvoiceStatuses {
			n, _ := strconv.Atoi(string(st.ItemSNo))
			byLine[n] = st
		}
	}
	for _, it := range inv.Items {
		if st, ok := byLine[it.LineNo]; ok {
			it.FBRStatusCode, it.FBRErrorCode, it.FBRError = string(st.StatusCode), string(st.ErrorCode), string(st.Error)
		}
	}
	if err := s.Store.MarkRejected(ctx, inv); err != nil {
		return nil, err
	}
	s.Audit(ctx, a, inv.CompanyID, "invoice.rejected", "invoice", fmt.Sprint(inv.ID), map[string]any{"no": inv.InternalNo, "errors": inv.FBRErrors})
	return s.Store.GetInvoice(ctx, inv.CompanyID, inv.ID)
}

func (s *Service) handleTransportError(ctx context.Context, a Actor, c *store.Company, inv *store.Invoice, err error, attempt int, posting bool) (*store.Invoice, error) {
	kind := fbr.KindOf(err)
	now := s.Now().UTC()
	switch {
	case kind == fbr.ErrUncertain && posting:
		msg := "No definitive answer from FBR (" + err.Error() + "). The invoice may have been recorded. Check IRIS (Digital Invoicing) before resubmitting."
		_ = s.Store.SetStatus(ctx, inv.ID, domain.StatusUncertain, msg, "")
		s.Audit(ctx, a, inv.CompanyID, "invoice.uncertain", "invoice", fmt.Sprint(inv.ID), map[string]any{"no": inv.InternalNo, "error": err.Error()})
	case kind == fbr.ErrNotSent || kind == fbr.ErrUnavailable || (kind == fbr.ErrUncertain && !posting) || (kind == fbr.ErrDecode && !posting):
		next := now.Add(backoff(attempt)).Format(time.RFC3339)
		msg := "FBR could not be reached (" + err.Error() + "). Queued for automatic resubmission."
		_ = s.Store.SetStatus(ctx, inv.ID, domain.StatusQueued, msg, next)
		_ = s.Store.MarkOffline(ctx, inv.ID, now.Format(time.RFC3339))
		s.Audit(ctx, a, inv.CompanyID, "invoice.queued", "invoice", fmt.Sprint(inv.ID), map[string]any{"no": inv.InternalNo, "error": err.Error(), "next": next})
	case kind == fbr.ErrAuth:
		next := now.Add(15 * time.Minute).Format(time.RFC3339)
		msg := "FBR rejected the security token (" + err.Error() + "). Check that the " + inv.Environment.Label() + " token is current (a sandbox token works only in the sandbox and a production token only in production) and that this server's IP is whitelisted with PRAL. The invoice is queued."
		_ = s.Store.SetStatus(ctx, inv.ID, domain.StatusQueued, msg, next)
		_ = s.Store.MarkOffline(ctx, inv.ID, now.Format(time.RFC3339))
		s.Audit(ctx, a, inv.CompanyID, "invoice.auth_error", "invoice", fmt.Sprint(inv.ID), err.Error())
	default:
		msg := "FBR returned an error: " + err.Error()
		inv.LastError = msg
		inv.FBRErrors = []fbr.ErrorItem{{Message: err.Error()}}
		_ = s.Store.MarkRejected(ctx, inv)
		s.Audit(ctx, a, inv.CompanyID, "invoice.error", "invoice", fmt.Sprint(inv.ID), err.Error())
	}
	return s.Store.GetInvoice(ctx, inv.CompanyID, inv.ID)
}

// ValidateWithFBR runs local validation and FBR's validateinvoicedata
// without recording the invoice.
func (s *Service) ValidateWithFBR(ctx context.Context, a Actor, companyID, id int64) (*store.Invoice, error) {
	unlock := s.lockInvoice(id)
	defer unlock()
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	inv, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if !inv.Status.Editable() {
		return nil, Invalid("invoice %s is %s", inv.InternalNo, inv.Status)
	}
	var original *store.Invoice
	if inv.RefInvoiceID != nil {
		original, _ = s.Store.GetInvoice(ctx, companyID, *inv.RefInvoiceID)
	}
	res := s.ValidateLocal(ctx, c, inv, original)
	if res.HasErrors() {
		_ = s.Store.SetValidation(ctx, id, domain.StatusDraft, res.Issues, nil, "local validation failed")
		return s.Store.GetInvoice(ctx, companyID, id)
	}
	cl, err := s.Client(ctx, c, inv.Environment, &inv.ID)
	if err != nil {
		return nil, err
	}
	p := s.BuildPayload(c, inv)
	resp, err := cl.ValidateInvoice(ctx, &p)
	if err != nil {
		_ = s.Store.SetValidation(ctx, id, domain.StatusDraft, res.Issues, nil, "FBR validation unavailable: "+err.Error())
		return nil, Invalid("FBR validation service unavailable: %v", err)
	}
	if resp.IsValid() {
		_ = s.Store.SetValidation(ctx, id, domain.StatusValidated, res.Issues, nil, "")
		s.Audit(ctx, a, companyID, "invoice.validated", "invoice", fmt.Sprint(id), inv.InternalNo)
	} else {
		errs := resp.Errors()
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.String())
		}
		_ = s.Store.SetValidation(ctx, id, domain.StatusDraft, res.Issues, errs, "FBR validation failed: "+strings.Join(msgs, "; "))
	}
	return s.Store.GetInvoice(ctx, companyID, id)
}

// ResolveInput reconciles an UNCERTAIN invoice after checking IRIS.
type ResolveInput struct {
	// Action: "accepted" (IRIS shows the invoice), "retry" (IRIS does not
	// show it; resubmit automatically) or "draft" (return to draft).
	Action           string `json:"action"`
	FBRInvoiceNumber string `json:"fbrInvoiceNumber"`
	FBRDated         string `json:"fbrDated"`
	Note             string `json:"note"`
}

var reFBRNo = regexp.MustCompile(`^(\d{7}|\d{9}|\d{13})DI\d{6,}$`)

// ResolveUncertain applies the operator's reconciliation.
func (s *Service) ResolveUncertain(ctx context.Context, a Actor, companyID, id int64, in ResolveInput) (*store.Invoice, error) {
	unlock := s.lockInvoice(id)
	defer unlock()
	inv, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != domain.StatusUncertain {
		return nil, Invalid("invoice %s is not awaiting reconciliation", inv.InternalNo)
	}
	switch in.Action {
	case "accepted":
		no := strings.TrimSpace(in.FBRInvoiceNumber)
		if !reFBRNo.MatchString(no) || !strings.HasPrefix(no, inv.SellerNTNCNIC+"DI") {
			return nil, Invalid("%q is not a valid FBR invoice number for seller %s", no, inv.SellerNTNCNIC)
		}
		// FBR's issue time decides the 72-hour cancellation window, so it
		// is taken from IRIS rather than assumed to be now.
		issued, ok := ParseFBRDated(in.FBRDated)
		if !ok {
			return nil, Invalid("enter the FBR date and time shown on IRIS for this invoice (YYYY-MM-DD HH:MM:SS)")
		}
		if issued.After(s.Now().Add(10 * time.Minute)) {
			return nil, Invalid("the FBR date and time cannot be in the future")
		}
		dated := issued.In(PKT).Format("2006-01-02 15:04:05")
		s.Audit(ctx, a, companyID, "invoice.reconciled", "invoice", fmt.Sprint(id), map[string]any{"action": "accepted", "fbr": no, "note": in.Note})
		return s.accept(ctx, a, inv, no, dated, "00", nil)
	case "retry":
		if err := s.Store.SetStatus(ctx, id, domain.StatusQueued, "Operator confirmed the invoice is not on IRIS; resubmitting. "+in.Note, ""); err != nil {
			return nil, err
		}
	case "draft":
		if err := s.Store.SetStatus(ctx, id, domain.StatusDraft, "Operator confirmed the invoice is not on IRIS. "+in.Note, ""); err != nil {
			return nil, err
		}
	default:
		return nil, Invalid("unknown action %q", in.Action)
	}
	s.Audit(ctx, a, companyID, "invoice.reconciled", "invoice", fmt.Sprint(id), map[string]any{"action": in.Action, "note": in.Note})
	return s.Store.GetInvoice(ctx, companyID, id)
}

// RetryNow moves a queued invoice to immediate resubmission.
func (s *Service) RetryNow(ctx context.Context, a Actor, companyID, id int64) (*store.Invoice, error) {
	inv, err := s.Store.GetInvoice(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != domain.StatusQueued {
		return nil, Invalid("invoice is not queued")
	}
	return s.Submit(ctx, a, companyID, id, SubmitOptions{})
}

// IsSubmissionError reports whether err is a local validation block.
func IsSubmissionError(err error) (*SubmissionError, bool) {
	var se *SubmissionError
	ok := errors.As(err, &se)
	return se, ok
}
