// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/store"
)

// Rule 150XA(c) of the Sales Tax Rules, 2006 requires any failure, disruption
// or tampering of the electronic invoicing system to be reported to the Board
// and the Commissioner within 24 hours. Besides FBR outages (health.go), the worker
// detects unexpected stops of this system (crash or power failure) and checks
// the audit chain and the seals of accepted invoices every day. Problems are
// recorded as auto-detected incidents so the 24-hour clock is visible.

const (
	settingHeartbeat     = "worker.heartbeat"
	settingCleanShutdown = "worker.clean_shutdown"
)

// productionCompanies lists the active companies that report to FBR
// production; the reporting duty applies to them.
func (s *Service) productionCompanies(ctx context.Context) []*store.Company {
	cs, err := s.Store.ListCompanies(ctx)
	if err != nil {
		return nil
	}
	var out []*store.Company
	for _, c := range cs {
		if c.Active && c.Environment == domain.EnvProduction {
			out = append(out, c)
		}
	}
	return out
}

func (s *Service) heartbeat(ctx context.Context) {
	_ = s.Store.SetSetting(ctx, settingHeartbeat, s.Now().UTC().Format(time.RFC3339))
}

// detectUnexpectedStop runs when the worker starts. If the system last
// stopped without an orderly shutdown (crash, power failure, killed process)
// and was down for longer than OutageThreshold, a closed system_failure
// incident is recorded for each production company.
func (s *Service) detectUnexpectedStop(ctx context.Context) {
	hb, _ := s.Store.GetSetting(ctx, settingHeartbeat)
	clean, _ := s.Store.GetSetting(ctx, settingCleanShutdown)
	_ = s.Store.SetSetting(ctx, settingCleanShutdown, "0")
	s.heartbeat(ctx)
	last := store.ParseTime(hb)
	now := s.Now()
	if last.IsZero() || clean == "1" || now.Sub(last) < OutageThreshold {
		return
	}
	desc := fmt.Sprintf("The electronic invoicing system stopped unexpectedly (crash or power failure) at about %s and was restarted at %s. Invoices could not be issued through the system in this period.",
		last.In(PKT).Format("02-Jan-2006 15:04"), now.In(PKT).Format("02-Jan-2006 15:04"))
	for _, c := range s.productionCompanies(ctx) {
		inc := &store.Incident{CompanyID: c.ID, Kind: "system_failure", Description: desc, AutoDetected: true,
			StartedAt: last.UTC().Format(time.RFC3339), EndedAt: now.UTC().Format(time.RFC3339)}
		if err := s.Store.SaveIncident(ctx, inc); err == nil {
			s.Audit(ctx, System, c.ID, "incident.opened", "incident", fmt.Sprint(inc.ID), desc)
		}
	}
	s.Log.Warn("unexpected stop detected", "lastHeartbeat", last, "restarted", now)
}

// MarkCleanShutdown records an orderly stop so the next start does not report
// a failure. The application calls it as soon as a stop is requested.
func (s *Service) MarkCleanShutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.heartbeat(ctx)
	_ = s.Store.SetSetting(ctx, settingCleanShutdown, "1")
}

// IntegrityReport is the outcome of a tamper check.
type IntegrityReport struct {
	AuditBrokenAt   int64    `json:"auditBrokenAt"`
	AuditChecked    int      `json:"auditChecked"`
	InvoicesChecked int      `json:"invoicesChecked"`
	Problems        []string `json:"problems"`
}

// OK reports whether nothing was found.
func (r *IntegrityReport) OK() bool { return r.AuditBrokenAt == 0 && len(r.Problems) == 0 }

// CheckIntegrity verifies the audit trail's hash chain and, for every
// company, the seal of each accepted invoice and the link to the invoice it
// was chained to, the digital signatures and the chain of closings. Any
// problem opens a tampering incident (rule 150XA(c)).
func (s *Service) CheckIntegrity(ctx context.Context) (*IntegrityReport, error) {
	rep := &IntegrityReport{}
	brokenAt, checked, err := s.Store.VerifyAuditChain(ctx)
	if err != nil {
		return nil, err
	}
	rep.AuditBrokenAt, rep.AuditChecked = brokenAt, checked
	cs, err := s.Store.ListCompanies(ctx)
	if err != nil {
		return nil, err
	}
	perCompany := map[int64][]string{}
	for _, c := range cs {
		invs, err := s.Store.SealedInvoices(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		seals := make(map[string]bool, len(invs))
		for _, inv := range invs {
			seals[inv.SealHash] = true
		}
		for _, inv := range invs {
			rep.InvoicesChecked++
			switch {
			case !VerifySeal(inv):
				perCompany[c.ID] = append(perCompany[c.ID], fmt.Sprintf("invoice %s (FBR %s): seal does not match its contents", inv.InternalNo, inv.FBRInvoiceNumber))
			case inv.PrevSealHash != "" && !seals[inv.PrevSealHash]:
				perCompany[c.ID] = append(perCompany[c.ID], fmt.Sprintf("invoice %s (FBR %s): the accepted invoice it is chained to is missing or altered", inv.InternalNo, inv.FBRInvoiceNumber))
			case inv.Signature != "" && !s.SignatureValid(ctx, inv):
				perCompany[c.ID] = append(perCompany[c.ID], fmt.Sprintf("invoice %s (FBR %s): digital signature does not match", inv.InternalNo, inv.FBRInvoiceNumber))
			}
		}
		for _, env := range []domain.Environment{domain.EnvProduction, domain.EnvSandbox, domain.EnvSimulator} {
			broken, _, err := s.Store.VerifyClosings(ctx, c.ID, env)
			if err != nil {
				return nil, err
			}
			if broken != 0 {
				perCompany[c.ID] = append(perCompany[c.ID], fmt.Sprintf("%s closings: closing %d does not match its hash chain (altered or deleted)", env, broken))
			}
		}
	}
	for cid, ps := range perCompany {
		rep.Problems = append(rep.Problems, ps...)
		s.openTampering(ctx, cid, "Integrity check: "+strings.Join(limit(ps, 20), "; "))
	}
	if p := s.signingKeyProblem(ctx); p != "" {
		rep.Problems = append(rep.Problems, p)
		for _, c := range cs {
			if c.Active {
				s.openTampering(ctx, c.ID, "Integrity check: "+p+".")
			}
		}
	}
	if brokenAt != 0 {
		desc := fmt.Sprintf("Integrity check: the audit trail's hash chain is broken at entry %d (entries altered or deleted).", brokenAt)
		for _, c := range cs {
			if c.Active {
				s.openTampering(ctx, c.ID, desc)
			}
		}
	}
	return rep, nil
}

func limit(ss []string, n int) []string {
	if len(ss) > n {
		return append(ss[:n:n], fmt.Sprintf("and %d more", len(ss)-n))
	}
	return ss
}

// openTampering opens an auto-detected tampering incident unless one is open.
func (s *Service) openTampering(ctx context.Context, companyID int64, desc string) {
	if open, _ := s.Store.OpenAutoIncident(ctx, companyID, "tampering"); open != nil {
		return
	}
	inc := &store.Incident{CompanyID: companyID, Kind: "tampering", Description: desc, AutoDetected: true,
		StartedAt: s.Now().UTC().Format(time.RFC3339)}
	if err := s.Store.SaveIncident(ctx, inc); err == nil {
		s.Audit(ctx, System, companyID, "incident.opened", "incident", fmt.Sprint(inc.ID), desc)
		s.Log.Error("tampering detected", "company", companyID, "detail", desc)
	}
}
