// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"fmt"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/license"
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

// LicenseReporter describes the installed licence (implemented by
// license.Manager).
type LicenseReporter interface {
	Status(ctx context.Context) license.Status
}

// Alerts computes the notifications for a company and environment. They are
// worked out from the current state, so each disappears once its cause is
// dealt with. includeSystem adds installation-wide items (backups) meant
// for administrators.
func (s *Service) Alerts(ctx context.Context, c *store.Company, env domain.Environment, includeSystem bool) []Alert {
	return append(s.companyAlerts(ctx, c, env), s.systemAlerts(ctx, includeSystem)...)
}

// companyAlerts are the alerts about one company's invoicing.
func (s *Service) companyAlerts(ctx context.Context, c *store.Company, env domain.Environment) []Alert {
	now := s.Now()
	out := []Alert{}
	add := func(id, sev, title, detail, link string) {
		out = append(out, Alert{ID: id, Severity: sev, Title: title, Detail: detail, Link: link})
	}

	if env != domain.EnvSimulator {
		if conn := s.Connection(c.ID, env); !conn.Healthy {
			title := "FBR Digital Invoicing is not reachable"
			if conn.AuthFailure {
				title = "FBR rejected the security token"
			}
			add("connection", "error", title, "Failing since "+shortTime(conn.FailingSince)+". Invoices are queued and sent automatically "+
				"once the connection is restored. "+conn.LastError, "/settings/fbr")
		}
	}
	if d, err := s.Store.GetDashboard(ctx, c.ID, env, now.In(PKT).Format("2006-01-02"), now.In(PKT).Format("2006-01")+"-01"); err == nil {
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
	if _, unreported, err := s.Store.IncidentCounts(ctx, c.ID); err == nil && unreported > 0 {
		add("incidents", "warning", fmt.Sprintf("%d incident(s) not reported to FBR", unreported),
			"Rule 150XA(c) requires operational failures, disruptions and tampering to be reported to FBR and the Commissioner within 24 hours. Record the report reference in the incident register.",
			"/incidents")
	}
	if env == domain.EnvProduction && c.ProductionTokenExpiry != "" {
		if exp, err := time.Parse("2006-01-02", c.ProductionTokenExpiry); err == nil && time.Until(exp) < 60*24*time.Hour {
			add("token", "warning", "Production security token expires on "+c.ProductionTokenExpiry,
				"Generate a new token on IRIS and save it under FBR integration before it expires.", "/settings/fbr")
		}
	}
	for _, dl := range s.CompanyDeadlines(ctx, c) {
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
		if last, err := s.Store.LastReferenceSync(ctx, env); err == nil {
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
		if sc, err := s.Scenarios(ctx, c.ID); err == nil && sc.AssignedCount > 0 && !sc.ReadyForProduction {
			add("scenarios", "info", fmt.Sprintf("Sandbox scenarios passed: %d of %d", sc.PassedCount, sc.AssignedCount),
				"Pass every scenario assigned on IRIS to obtain the production token.", "/scenarios")
		}
	}
	return out
}

// systemAlerts are installation-wide: the licence and, for administrators,
// backups.
func (s *Service) systemAlerts(ctx context.Context, includeBackups bool) []Alert {
	now := s.Now()
	out := []Alert{}
	add := func(id, sev, title, detail, link string) {
		out = append(out, Alert{ID: id, Severity: sev, Title: title, Detail: detail, Link: link})
	}
	if includeBackups {
		if b, err := s.ListBackups(); err == nil {
			if len(b) == 0 {
				add("backup", "warning", "No database backup yet", "Create a backup now and keep a copy of master.key in a safe place.", "/settings/system")
			} else if t, err := time.Parse(time.RFC3339, b[0].Created); err == nil && now.Sub(t) > 48*time.Hour {
				add("backup", "warning", "Last backup is more than 2 days old", "Last backup "+shortTime(b[0].Created)+
					". Check that the server runs at the scheduled backup hour.", "/settings/system")
			}
		}
	}
	if lr, ok := s.Opts.License.(LicenseReporter); ok {
		lic := lr.Status(ctx)
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
	}
	return out
}

func shortTime(iso string) string {
	t := store.ParseTime(iso)
	if t.IsZero() {
		return iso
	}
	return t.In(PKT).Format("02 Jan 2006 15:04")
}
