// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/service"
	"einvoicing/internal/store"
)

// Alert is a notification shown under the bell in the header. Alerts are
// computed from the current state on every request, so they disappear as
// soon as their cause is dealt with.
type Alert struct {
	ID       string `json:"id"`
	Severity string `json:"severity"` // error | warning | info
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Link     string `json:"link,omitempty"`
}

// alertsFor computes the alerts for a company and environment.
func (s *Server) alertsFor(r *http.Request, rc *reqCtx, c *store.Company, env domain.Environment) []Alert {
	ctx := r.Context()
	now := s.Svc.Now()
	out := []Alert{}
	add := func(id, sev, title, detail, link string) {
		out = append(out, Alert{ID: id, Severity: sev, Title: title, Detail: detail, Link: link})
	}

	if env != domain.EnvSimulator {
		if conn := s.Svc.Connection(c.ID, env); !conn.Healthy {
			title := "FBR Digital Invoicing is not reachable"
			if conn.AuthFailure {
				title = "FBR rejected the security token"
			}
			add("connection", "error", title, "Failing since "+shortTime(conn.FailingSince)+". Invoices are queued and sent automatically "+
				"once the connection is restored. "+conn.LastError, "/settings/fbr")
		}
	}
	if d, err := s.Svc.Store.GetDashboard(ctx, c.ID, env, now.In(service.PKT).Format("2006-01-02"), now.In(service.PKT).Format("2006-01")+"-01"); err == nil {
		counts := map[string]int{}
		for _, sc := range d.StatusCounts {
			counts[sc.Status] = sc.Count
		}
		if d.PendingUpload > 0 {
			sev := "warning"
			if t, err := time.Parse(time.RFC3339, d.OldestPending); err == nil && now.Sub(t) > 24*time.Hour {
				sev = "error"
			}
			add("pending", sev, fmt.Sprintf("%d invoice(s) not yet reported to FBR", d.PendingUpload),
				"Oldest issued "+shortTime(d.OldestPending)+". Invoices issued while FBR was unreachable must be uploaded within 24 hours of "+
					"the connection being restored.", "/invoices?status=QUEUED,REJECTED,UNCERTAIN")
		}
		if n := counts[string(domain.StatusRejected)]; n > 0 {
			add("rejected", "error", fmt.Sprintf("%d invoice(s) rejected by FBR", n),
				"Correct the errors shown on each invoice and resubmit it. Rejected invoices are not tax invoices until FBR accepts them.",
				"/invoices?status=REJECTED")
		}
		if n := counts[string(domain.StatusUncertain)]; n > 0 {
			add("uncertain", "warning", fmt.Sprintf("%d submission(s) need reconciliation", n),
				"FBR's response was lost. Check the invoice on IRIS and record the outcome before resubmitting, to avoid reporting it twice.",
				"/invoices?status=UNCERTAIN")
		}
		if n := counts[string(domain.StatusDraft)] + counts[string(domain.StatusValidated)]; n > 0 {
			add("drafts", "info", fmt.Sprintf("%d draft invoice(s) not yet issued", n),
				"Drafts are not reported to FBR. Issue them when the supply is made, or delete them.", "/invoices?status=DRAFT,VALIDATED")
		}
	}
	if _, unreported, err := s.Svc.Store.IncidentCounts(ctx, c.ID); err == nil && unreported > 0 {
		add("incidents", "warning", fmt.Sprintf("%d incident(s) not reported to FBR", unreported),
			"Rule 150R requires operational failures to be reported to FBR within 24 hours. Record the report reference in the incident register.",
			"/incidents")
	}
	if env == domain.EnvProduction && c.ProductionTokenExpiry != "" {
		if exp, err := time.Parse("2006-01-02", c.ProductionTokenExpiry); err == nil && time.Until(exp) < 60*24*time.Hour {
			add("token", "warning", "Production security token expires on "+c.ProductionTokenExpiry,
				"Generate a new token on IRIS and save it under FBR integration before it expires.", "/settings/fbr")
		}
	}
	for _, dl := range s.Svc.CompanyDeadlines(c) {
		if dl.DaysLeft < 0 || dl.DaysLeft > 5 {
			continue
		}
		sev := "info"
		if dl.DaysLeft <= 2 {
			sev = "warning"
		}
		what := "Pay sales tax"
		if dl.Kind == "filing" {
			what = "File the sales tax return"
		}
		when := fmt.Sprintf("in %d days", dl.DaysLeft)
		switch dl.DaysLeft {
		case 0:
			when = "today"
		case 1:
			when = "tomorrow"
		}
		add("deadline-"+dl.Kind, sev, fmt.Sprintf("%s for %s %s", what, dl.PeriodLabel, when),
			"Due "+dl.Due+". Check FBR's website for any extension. Review the period before filing.", "/compliance")
	}
	if env != domain.EnvSimulator {
		if last, err := s.Svc.Store.LastReferenceSync(ctx, env); err == nil {
			if last == "" {
				add("refsync", "info", "FBR reference data not downloaded yet",
					"Download FBR's lists of sale types, units, HS codes and SROs so that invoices are checked against FBR's current data.", "/library?tab=sync")
			} else if t := store.ParseTime(last); !t.IsZero() && now.Sub(t) > 14*24*time.Hour {
				add("refsync", "info", "FBR reference data is more than 14 days old",
					"Last downloaded "+shortTime(last)+". Refresh it to pick up new sale types, rates and SROs.", "/library?tab=sync")
			}
		}
	}
	if env == domain.EnvSandbox {
		if sc, err := s.Svc.Scenarios(ctx, c.ID); err == nil && sc.AssignedCount > 0 && !sc.ReadyForProduction {
			add("scenarios", "info", fmt.Sprintf("Sandbox scenarios passed: %d of %d", sc.PassedCount, sc.AssignedCount),
				"Pass every scenario assigned on IRIS to obtain the production token.", "/scenarios")
		}
	}
	lic := s.License.Status(ctx)
	switch lic.Mode {
	case "trial", "grace":
		add("license", "warning", "Licence: "+lic.Message, "", "/settings/system")
	case "trial_expired", "expired", "invalid":
		add("license", "error", "Licence: "+lic.Message, "", "/settings/system")
	case "licensed":
		if lic.DaysLeft > 0 && lic.DaysLeft <= 30 {
			add("license", "warning", fmt.Sprintf("Licence expires in %d days", lic.DaysLeft), "Contact the vendor to renew it.", "/settings/system")
		}
	}
	if allowed(rc, PermSystem) {
		if b, err := s.Svc.ListBackups(); err == nil {
			if len(b) == 0 {
				add("backup", "warning", "No database backup yet", "Create a backup now and keep a copy of master.key in a safe place.", "/settings/system")
			} else if t, err := time.Parse(time.RFC3339, b[0].Created); err == nil && now.Sub(t) > 48*time.Hour {
				add("backup", "warning", "Last backup is more than 2 days old", "Last backup "+shortTime(b[0].Created)+
					". Check that the server runs at the scheduled backup hour.", "/settings/system")
			}
		}
	}
	return out
}

func shortTime(iso string) string {
	t := store.ParseTime(iso)
	if t.IsZero() {
		return iso
	}
	return t.In(service.PKT).Format("02 Jan 2006 15:04")
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"alerts": s.alertsFor(r, rc, c, envParam(r, c))})
}

// handleSearch searches invoices, customers, products and FBR's HS codes.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	ctx := r.Context()
	c, err := s.Svc.Store.GetCompany(ctx, cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	out := map[string]any{"q": q, "invoices": []*store.Invoice{}, "customers": []*store.Customer{}, "products": []*store.Product{}, "hsCodes": []store.HSCode{}}
	if len([]rune(q)) < 2 {
		writeJSON(w, 200, out)
		return
	}
	limit := qInt(r, "limit", 6)
	if inv, _, err := s.Svc.Store.ListInvoices(ctx, c.ID, store.InvoiceFilter{Environment: envParam(r, c), Q: q, Limit: limit}); err == nil && inv != nil {
		out["invoices"] = inv
	}
	if cu, _, err := s.Svc.Store.ListCustomers(ctx, c.ID, store.ListParams{Q: q, Limit: limit}); err == nil && cu != nil {
		out["customers"] = cu
	}
	if pr, _, err := s.Svc.Store.ListProducts(ctx, c.ID, store.ListParams{Q: q, Limit: limit}); err == nil && pr != nil {
		out["products"] = pr
	}
	if hs, err := s.Svc.Store.SearchHSCodes(ctx, q, limit); err == nil && hs != nil {
		out["hsCodes"] = hs
	}
	writeJSON(w, 200, out)
}

// handleCompliance returns the review of a tax period (?period=YYYY-MM).
func (s *Server) handleCompliance(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	rev, err := s.Svc.ReviewPeriod(r.Context(), c, envParam(r, c), r.URL.Query().Get("period"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, rev)
}

func (s *Server) handleDuplicateInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	inv, err := s.Svc.DuplicateInvoice(r.Context(), rc.Actor, cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, inv)
}
