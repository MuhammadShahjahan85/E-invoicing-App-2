package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/printing"
	"einvoicing/internal/security"
	"einvoicing/internal/service"
	"einvoicing/internal/store"

	"github.com/shopspring/decimal"
)

// invoiceRequest accepts either the product's own invoice format or a raw
// FBR DI payload ("fbrPayload") from an ERP that already produces one.
type invoiceRequest struct {
	service.InvoiceInput
	FBRPayload *fbr.InvoicePayload `json:"fbrPayload"`
}

// inputFromFBR converts a DI payload into an InvoiceInput, keeping every
// amount the ERP supplied.
func inputFromFBR(p *fbr.InvoicePayload) service.InvoiceInput {
	in := service.InvoiceInput{DocType: p.InvoiceType, InvoiceDate: p.InvoiceDate, InvoiceRefNo: p.InvoiceRefNo, ScenarioID: p.ScenarioID,
		Buyer: &service.BuyerInput{NTNCNIC: p.BuyerNTNCNIC, Name: p.BuyerBusinessName, Province: p.BuyerProvince, Address: p.BuyerAddress,
			RegistrationType: p.BuyerRegistrationType}}
	for _, it := range p.Items {
		v, st, ft, fed, wh := it.ValueSalesExcludingST.Value, it.SalesTaxApplicable.Value, it.FurtherTax.Value, it.FEDPayable.Value, it.SalesTaxWithheldAtSource.Value
		item := service.ItemInput{HSCode: it.HSCode, Description: it.ProductDescription, UoM: it.UoM, Quantity: it.Quantity.Value,
			Value: &v, SaleType: it.SaleType, Rate: it.Rate, SalesTax: &st, FurtherTax: &ft, FED: &fed, STWithheld: &wh,
			DiscountAmount: it.Discount.Value, FurtherTaxMode: "no", SROScheduleNo: it.SROScheduleNo, SROItemSerialNo: it.SROItemSerialNo}
		if !it.ExtraTax.Empty {
			et := it.ExtraTax.Value
			item.ExtraTax = &et
		}
		if it.FixedNotifiedValueOrRetailPrice.Value.IsPositive() {
			rv := it.FixedNotifiedValueOrRetailPrice.Value
			item.RetailValue = &rv
		}
		in.Items = append(in.Items, item)
	}
	return in
}

func (s *Server) readInvoiceInput(r *http.Request, rc *reqCtx) (*service.InvoiceInput, error) {
	var req invoiceRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	in := req.InvoiceInput
	if req.FBRPayload != nil {
		conv := inputFromFBR(req.FBRPayload)
		conv.ExternalRef, conv.Submit, conv.Environment, conv.CustomerID = in.ExternalRef, in.Submit, in.Environment, in.CustomerID
		in = conv
	}
	// The environment is chosen by the company setting (Settings → FBR integration),
	// which requires company.write and the go-live checks. A request may name it
	// only to confirm it, never to post to another environment.
	if in.Environment != "" {
		if env := s.companyEnv(r); in.Environment != env {
			return nil, service.Invalid("environment %q does not match the company's working environment (%s); change it under Settings → FBR integration", in.Environment, env)
		}
	}
	in.Source = "ui"
	if rc.APIKey != nil {
		in.Source = "api"
	}
	return &in, nil
}

func (s *Server) handleListInvoices(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	q := r.URL.Query()
	f := store.InvoiceFilter{Environment: domain.Environment(q.Get("env")), DocType: domain.DocType(q.Get("docType")), From: q.Get("from"), To: q.Get("to"),
		Q: q.Get("q"), Limit: qInt(r, "limit", 50), Offset: qInt(r, "offset", 0)}
	if st := q.Get("status"); st != "" {
		for _, x := range strings.Split(st, ",") {
			f.Status = append(f.Status, domain.InvoiceStatus(strings.TrimSpace(x)))
		}
	}
	if f.Environment == "" {
		if c, err := s.Svc.Store.GetCompany(r.Context(), cid(r)); err == nil {
			f.Environment = c.Environment
		}
	}
	list, total, err := s.Svc.Store.ListInvoices(r.Context(), cid(r), f)
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.Invoice{}
	}
	writeJSON(w, 200, map[string]any{"items": list, "total": total})
}

func (s *Server) handleCreateInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	in, err := s.readInvoiceInput(r, rc)
	if err != nil {
		s.fail(w, err)
		return
	}
	inv, err := s.Svc.CreateInvoice(r.Context(), rc.Actor, cid(r), in)
	if err != nil {
		// If the invoice was saved but submission was blocked, return it with the issues.
		if se, ok := service.IsSubmissionError(err); ok && in.ExternalRef != "" {
			if saved, e2 := s.Svc.Store.GetInvoiceByExternalRef(r.Context(), cid(r), envOr(in.Environment, s.companyEnv(r)), in.ExternalRef); e2 == nil {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "saved as draft; validation errors prevented submission", "issues": se.Issues, "invoice": saved})
				return
			}
		}
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, inv)
}

func envOr(e, def domain.Environment) domain.Environment {
	if e.Valid() {
		return e
	}
	return def
}

func (s *Server) companyEnv(r *http.Request) domain.Environment {
	if c, err := s.Svc.Store.GetCompany(r.Context(), cid(r)); err == nil {
		return c.Environment
	}
	return domain.EnvSimulator
}

func (s *Server) handleComputeInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	in, err := s.readInvoiceInput(r, rc)
	if err != nil {
		s.fail(w, err)
		return
	}
	inv, err := s.Svc.Compute(r.Context(), cid(r), in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, inv)
}

func (s *Server) handleGetInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	inv, err := s.Svc.Store.GetInvoice(r.Context(), cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	resp := map[string]any{"invoice": inv, "sealValid": inv.SealHash == "" || service.VerifySeal(inv)}
	if inv.RefInvoiceID != nil {
		if o, err := s.Svc.Store.GetInvoice(r.Context(), cid(r), *inv.RefInvoiceID); err == nil {
			resp["original"] = map[string]any{"id": o.ID, "internalNo": o.InternalNo, "fbrInvoiceNumber": o.FBRInvoiceNumber}
		}
	}
	if inv.DocType == domain.DocSaleInvoice && inv.FBRInvoiceNumber != "" {
		v, t, _ := s.Svc.Store.SumDebitNotes(r.Context(), cid(r), inv.FBRInvoiceNumber)
		resp["debitNotes"] = map[string]decimal.Decimal{"value": v, "salesTax": t}
	}
	writeJSON(w, 200, resp)
}

func (s *Server) handleInvoiceByRef(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	env := envOr(domain.Environment(r.URL.Query().Get("env")), s.companyEnv(r))
	inv, err := s.Svc.Store.GetInvoiceByExternalRef(r.Context(), cid(r), env, r.PathValue("ref"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, inv)
}

func (s *Server) handleUpdateInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	in, err := s.readInvoiceInput(r, rc)
	if err != nil {
		s.fail(w, err)
		return
	}
	inv, err := s.Svc.UpdateInvoice(r.Context(), rc.Actor, cid(r), id, in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, inv)
}

func (s *Server) handleDeleteInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Svc.DeleteInvoice(r.Context(), rc.Actor, cid(r), id); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) invoiceAction(w http.ResponseWriter, r *http.Request, fn func(id int64) (*store.Invoice, error)) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	inv, err := fn(id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, inv)
}

func (s *Server) handleValidateInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	s.invoiceAction(w, r, func(id int64) (*store.Invoice, error) {
		return s.Svc.ValidateWithFBR(r.Context(), rc.Actor, cid(r), id)
	})
}

func (s *Server) handleSubmitInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	s.invoiceAction(w, r, func(id int64) (*store.Invoice, error) {
		return s.Svc.Submit(r.Context(), rc.Actor, cid(r), id, service.SubmitOptions{SkipFBRValidation: r.URL.Query().Get("skipValidate") == "1"})
	})
}

func (s *Server) handleRetryInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	s.invoiceAction(w, r, func(id int64) (*store.Invoice, error) { return s.Svc.RetryNow(r.Context(), rc.Actor, cid(r), id) })
}

func (s *Server) handleResolveInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in service.ResolveInput
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	s.invoiceAction(w, r, func(id int64) (*store.Invoice, error) {
		return s.Svc.ResolveUncertain(r.Context(), rc.Actor, cid(r), id, in)
	})
}

func (s *Server) handleCancelInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in service.CancelInput
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	s.invoiceAction(w, r, func(id int64) (*store.Invoice, error) {
		return s.Svc.CancelInvoice(r.Context(), rc.Actor, cid(r), id, in)
	})
}

func (s *Server) handleDebitNote(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	s.invoiceAction(w, r, func(id int64) (*store.Invoice, error) {
		return s.Svc.NewDebitNoteDraft(r.Context(), rc.Actor, cid(r), id)
	})
}

func (s *Server) handleInvoicePayload(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	b, err := s.Svc.Payload(r.Context(), cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var out bytes.Buffer
	if json.Indent(&out, b, "", "  ") == nil {
		_, _ = w.Write(out.Bytes())
		return
	}
	_, _ = w.Write(b)
}

func (s *Server) handleInvoiceCalls(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	list, _, err := s.Svc.Store.ListFBRCalls(r.Context(), cid(r), id, 100, 0)
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.FBRCall{}
	}
	writeJSON(w, 200, list)
}

func (s *Server) fbrLogo(r *http.Request) ([]byte, string) {
	v, _ := s.Svc.Store.GetSetting(r.Context(), "fbr_logo")
	mime, _ := s.Svc.Store.GetSetting(r.Context(), "fbr_logo_mime")
	if v == "" {
		return nil, ""
	}
	b, err := decodeB64(v)
	if err != nil {
		return nil, ""
	}
	return b, mime
}

func (s *Server) handlePrint(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	ctx := r.Context()
	inv, err := s.Svc.Store.GetInvoice(ctx, cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	c, err := s.Svc.Store.GetCompany(ctx, cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	logo, mime, _ := s.Svc.Store.GetCompanyLogo(ctx, c.ID)
	fbrLogo, fbrMime := s.fbrLogo(r)
	q := r.URL.Query()
	nonce := security.RandomToken(18)
	var buf bytes.Buffer
	err = printing.Render(&buf, c, inv, printing.Options{Format: q.Get("format"), AutoPrint: q.Get("autoprint") == "1", ShowToolbar: q.Get("toolbar") != "0",
		CompanyLogo: logo, CompanyMime: mime, FBRLogo: fbrLogo, FBRLogoMime: fbrMime, Nonce: nonce})
	if err != nil {
		s.fail(w, err)
		return
	}
	if inv.Status == domain.StatusAccepted && q.Get("preview") != "1" {
		_ = s.Svc.Store.IncrementPrintCount(ctx, inv.ID)
		s.Svc.Audit(ctx, rc.Actor, c.ID, "invoice.print", "invoice", fmt.Sprint(inv.ID), map[string]any{"no": inv.InternalNo, "copy": inv.PrintCount + 1})
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src data:; style-src 'unsafe-inline'; script-src 'nonce-"+nonce+"'")
	_, _ = w.Write(buf.Bytes())
}

func (s *Server) qrContent(r *http.Request) (string, error) {
	id, err := pathID(r, "id")
	if err != nil {
		return "", err
	}
	inv, err := s.Svc.Store.GetInvoice(r.Context(), cid(r), id)
	if err != nil {
		return "", err
	}
	if inv.FBRInvoiceNumber == "" {
		return "", service.Invalid("the invoice has no FBR invoice number yet")
	}
	return inv.FBRInvoiceNumber, nil
}

func (s *Server) handleQRSVG(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	content, err := s.qrContent(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	svg, err := printing.QRSVG(content)
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write([]byte(svg))
}

func (s *Server) handleQRPNG(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	content, err := s.qrContent(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	png, err := printing.QRPNG(content, qInt(r, "scale", 12)) // scale is clamped to 1..40 px per module
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	_, _ = w.Write(png)
}

// --- import ---

func (s *Server) handleTemplateCSV(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	attachment(w, "invoice-import-template.csv", "text/csv; charset=utf-8", service.ImportTemplateCSV())
}

func (s *Server) handleTemplateXLSX(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	b, err := service.ImportTemplateXLSX()
	if err != nil {
		s.fail(w, err)
		return
	}
	attachment(w, "invoice-import-template.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", b)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	r.Body = http.MaxBytesReader(w, r.Body, 21<<20)
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeErr(w, http.StatusRequestEntityTooLarge, "the file is larger than 20 MB; split it into smaller files")
			return
		}
		writeErr(w, 400, "upload a CSV or XLSX file (multipart field 'file')")
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeErr(w, 400, "missing file")
		return
	}
	defer f.Close()
	rows, err := service.ReadRows(hdr.Filename, f)
	if err != nil {
		s.fail(w, service.Invalid("%v", err))
		return
	}
	preview := r.URL.Query().Get("preview") == "1"
	submit := r.URL.Query().Get("submit") == "1"
	sum, err := s.Svc.Import(r.Context(), rc.Actor, cid(r), rows, preview, submit)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, sum)
}

// --- scenarios ---

func (s *Server) handleScenarios(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	ov, err := s.Svc.Scenarios(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, ov)
}

func (s *Server) handleAssignScenarios(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in struct {
		ScenarioIDs []string `json:"scenarioIds"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Svc.SetAssignedScenarios(r.Context(), rc.Actor, cid(r), in.ScenarioIDs); err != nil {
		s.fail(w, err)
		return
	}
	s.handleScenarios(w, r, rc)
}

func (s *Server) handleRunScenario(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in struct {
		Mode        string             `json:"mode"`
		Environment domain.Environment `json:"environment"`
	}
	_ = decode(r, &in)
	run, inv, err := s.Svc.RunScenario(r.Context(), rc.Actor, cid(r), r.PathValue("sn"), in.Mode, in.Environment)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"run": run, "invoice": inv})
}

// --- reports ---

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	q := r.URL.Query()
	env := envOr(domain.Environment(q.Get("env")), s.companyEnv(r))
	f := store.ReportFilter{CompanyID: cid(r), Environment: env, From: q.Get("from"), To: q.Get("to")}
	ctx := r.Context()
	var data any
	var table *service.Table
	switch r.PathValue("kind") {
	case "register":
		rows, err := s.Svc.Store.SalesRegister(ctx, f)
		if err != nil {
			s.fail(w, err)
			return
		}
		if rows == nil {
			rows = []store.RegisterLine{}
		}
		data, table = rows, service.RegisterTable(rows)
	case "tax-summary":
		rows, err := s.Svc.Store.TaxSummary(ctx, f)
		if err != nil {
			s.fail(w, err)
			return
		}
		if rows == nil {
			rows = []store.TaxSummaryRow{}
		}
		data, table = rows, service.TaxSummaryTable(rows)
	case "monthly":
		rows, err := s.Svc.Store.MonthlySummary(ctx, f)
		if err != nil {
			s.fail(w, err)
			return
		}
		if rows == nil {
			rows = []store.PeriodRow{}
		}
		data, table = rows, service.MonthlyTable(rows)
	case "customers":
		rows, err := s.Svc.Store.CustomerSummary(ctx, f)
		if err != nil {
			s.fail(w, err)
			return
		}
		if rows == nil {
			rows = []store.CustomerRow{}
		}
		data, table = rows, service.CustomerTable(rows)
	default:
		writeErr(w, 404, "unknown report")
		return
	}
	name := fmt.Sprintf("%s-%s-%s-to-%s", r.PathValue("kind"), env, nz(f.From, "start"), nz(f.To, "today"))
	switch q.Get("format") {
	case "csv":
		attachment(w, name+".csv", "text/csv; charset=utf-8", table.CSV())
	case "xlsx":
		b, err := table.XLSX()
		if err != nil {
			s.fail(w, err)
			return
		}
		attachment(w, name+".xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", b)
	default:
		writeJSON(w, 200, data)
	}
}

func nz(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (s *Server) handleCalls(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, total, err := s.Svc.Store.ListFBRCalls(r.Context(), cid(r), 0, qInt(r, "limit", 100), qInt(r, "offset", 0))
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.FBRCall{}
	}
	writeJSON(w, 200, map[string]any{"items": list, "total": total})
}
