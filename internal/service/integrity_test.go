// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"strings"
	"testing"
	"time"

	"einvoicing/internal/domain"
)

func incidentsOfKind(t *testing.T, f *fixture, kind string) int {
	t.Helper()
	list, err := f.svc.Store.ListIncidents(f.ctx, f.cid, 0)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, i := range list {
		if i.Kind == kind && i.AutoDetected {
			n++
		}
	}
	return n
}

// Altering an accepted invoice directly in the database (bypassing the
// application) is found by the integrity check and opens a tampering
// incident for Rule 150R reporting.
func TestIntegrityCheckDetectsTampering(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	for _, sc := range []string{"SN001", "SN002"} {
		if _, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, sc, "engine", domain.EnvSandbox); err != nil || inv.Status != domain.StatusAccepted {
			t.Fatalf("%s: %v", sc, err)
		}
	}
	rep, err := f.svc.CheckIntegrity(f.ctx)
	if err != nil || !rep.OK() || rep.InvoicesChecked != 2 {
		t.Fatalf("clean database reported problems: %+v %v", rep, err)
	}
	if n := incidentsOfKind(t, f, "tampering"); n != 0 {
		t.Fatalf("no incident expected, got %d", n)
	}

	// Someone with access to the database file removes the protection and
	// lowers the sales tax of an accepted invoice.
	db := f.svc.Store.DB
	if _, err := db.ExecContext(f.ctx, `DROP TRIGGER trg_invoices_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(f.ctx, `UPDATE invoices SET total_sales_tax = total_sales_tax - 100 WHERE id = (SELECT MIN(id) FROM invoices WHERE seal_hash <> '')`); err != nil {
		t.Fatal(err)
	}
	rep, err = f.svc.CheckIntegrity(f.ctx)
	if err != nil || rep.OK() || !strings.Contains(strings.Join(rep.Problems, " "), "seal does not match") {
		t.Fatalf("tampering not detected: %+v %v", rep, err)
	}
	if n := incidentsOfKind(t, f, "tampering"); n != 1 {
		t.Fatalf("expected one tampering incident, got %d", n)
	}
	// A second run does not open a duplicate while the first is open.
	if _, err := f.svc.CheckIntegrity(f.ctx); err != nil {
		t.Fatal(err)
	}
	if n := incidentsOfKind(t, f, "tampering"); n != 1 {
		t.Fatalf("duplicate tampering incident: %d", n)
	}
}

// An unexpected stop (no orderly shutdown) of a production installation is
// recorded as a system failure incident; an orderly stop is not.
func TestUnexpectedStopIsRecorded(t *testing.T) {
	f := setup(t)
	if _, err := f.svc.SetToken(f.ctx, f.admin, f.cid, TokenInput{Environment: domain.EnvProduction, Token: "production-token-123"}); err != nil {
		t.Fatal(err)
	}
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	c.Environment = domain.EnvProduction
	if _, err := f.svc.SaveCompany(f.ctx, f.admin, c); err != nil {
		t.Fatal(err)
	}
	stopped := time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339)

	// Orderly shutdown three hours ago: nothing to report.
	_ = f.svc.Store.SetSetting(f.ctx, settingHeartbeat, stopped)
	_ = f.svc.Store.SetSetting(f.ctx, settingCleanShutdown, "1")
	f.svc.detectUnexpectedStop(f.ctx)
	if n := incidentsOfKind(t, f, "system_failure"); n != 0 {
		t.Fatalf("orderly stop reported as failure (%d)", n)
	}

	// Power failure: the last heartbeat is three hours old and the stop was not orderly.
	_ = f.svc.Store.SetSetting(f.ctx, settingHeartbeat, stopped)
	f.svc.detectUnexpectedStop(f.ctx) // the previous start left clean_shutdown=0
	list, _ := f.svc.Store.ListIncidents(f.ctx, f.cid, 0)
	if n := incidentsOfKind(t, f, "system_failure"); n != 1 {
		t.Fatalf("expected one system failure incident, got %d", n)
	}
	for _, i := range list {
		if i.Kind == "system_failure" && (i.StartedAt != stopped || i.EndedAt == "" || i.ReportedAt != "") {
			t.Fatalf("incident %+v", i)
		}
	}

	// A quick restart (heartbeat a minute ago) is not an outage.
	_ = f.svc.Store.SetSetting(f.ctx, settingHeartbeat, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339))
	f.svc.detectUnexpectedStop(f.ctx)
	if n := incidentsOfKind(t, f, "system_failure"); n != 1 {
		t.Fatalf("short restart must not be reported, got %d", n)
	}
}
