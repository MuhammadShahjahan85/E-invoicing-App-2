// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"einvoicing/internal/db"
	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/fbrmock"
	"einvoicing/internal/security"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"
)

const ntn = "0786909"

type fixture struct {
	svc   *Service
	sim   *fbrmock.Server
	admin Actor
	cid   int64
	ctx   context.Context
}

func setup(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	vault, err := security.OpenVault(dir)
	if err != nil {
		t.Fatal(err)
	}
	sim := fbrmock.New()
	sim.AddToken("sandbox-token-123", ntn, true)
	sim.AddToken("production-token-123", ntn, false)
	sim.TimeoutDelay = 1500 * time.Millisecond
	ts := httptest.NewServer(sim.Handler())
	t.Cleanup(ts.Close)
	svc := New(store.New(d), vault, Options{Endpoints: fbr.Endpoints{BaseURL: ts.URL}, HTTPTimeout: 700 * time.Millisecond, DataDir: dir})
	ctx := context.Background()
	if err := svc.EnsureSeedData(ctx); err != nil {
		t.Fatal(err)
	}
	err = svc.Setup(ctx, "127.0.0.1", SetupInput{AdminUsername: "admin", AdminPassword: "Admin12345", AdminFullName: "Admin",
		Company: store.Company{Name: "Company 8", NTNCNIC: ntn, Province: "Sindh", Address: "Karachi",
			BusinessActivities: []string{"Manufacturer"}, Sectors: []string{"All Other Sectors"}, Environment: domain.EnvSimulator, ValidateBeforePost: true}})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := svc.Store.GetUserByUsername(ctx, "admin")
	cs, _ := svc.Store.ListCompanies(ctx)
	return &fixture{svc: svc, sim: sim, admin: Actor{UserID: &u.ID, Username: "admin", Role: store.RoleAdmin}, cid: cs[0].ID, ctx: ctx}
}

func (f *fixture) customer(t *testing.T) *store.Customer {
	c, err := f.svc.SaveCustomer(f.ctx, f.admin, &store.Customer{CompanyID: f.cid, Name: "FERTILIZER MANUFAC IRS NEW", NTNCNIC: "2046004",
		RegistrationType: domain.Registered, Province: "Punjab", Address: "Lahore"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f *fixture) product(t *testing.T) *store.Product {
	p, err := f.svc.SaveProduct(f.ctx, f.admin, &store.Product{CompanyID: f.cid, Code: "P1", Description: "Fertilizer bag", HSCode: "3104.2000",
		UoM: "KG", SaleType: domain.STStandard, Rate: "18%", UnitPrice: tax.MustD("250")})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInvoiceLifecycleSimulator(t *testing.T) {
	f := setup(t)
	cust, prod := f.customer(t), f.product(t)
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{CustomerID: &cust.ID,
		Items: []ItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("40"), DiscountAmount: tax.MustD("100")}}, Submit: true})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Status != domain.StatusAccepted {
		t.Fatalf("status %s: %s %+v", inv.Status, inv.LastError, inv.FBRErrors)
	}
	if !strings.HasPrefix(inv.FBRInvoiceNumber, ntn+"DI") || !strings.HasPrefix(inv.InternalNo, "TR-INV-") {
		t.Fatalf("numbers %s %s", inv.FBRInvoiceNumber, inv.InternalNo)
	}
	if !inv.Totals.SalesTax.Equal(tax.MustD("1782")) || !inv.Totals.ValueExclST.Equal(tax.MustD("9900")) {
		t.Fatalf("totals %+v", inv.Totals)
	}
	if !VerifySeal(inv) || inv.Items[0].FBRItemInvoiceNo != inv.FBRInvoiceNumber+"-1" {
		t.Fatal("seal or item number invalid")
	}
	// Accepted invoices are immutable and cannot be deleted.
	if _, err := f.svc.UpdateInvoice(f.ctx, f.admin, f.cid, inv.ID, &InvoiceInput{CustomerID: &cust.ID, Items: []ItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("1")}}}); err == nil {
		t.Fatal("expected edit to be refused")
	}
	if err := f.svc.DeleteInvoice(f.ctx, f.admin, f.cid, inv.ID); err == nil {
		t.Fatal("expected delete to be refused")
	}
	// Re-submitting is idempotent.
	again, err := f.svc.Submit(f.ctx, f.admin, f.cid, inv.ID, SubmitOptions{})
	if err != nil || again.FBRInvoiceNumber != inv.FBRInvoiceNumber {
		t.Fatalf("resubmit: %v", err)
	}
	// Debit note.
	dn, err := f.svc.NewDebitNoteDraft(f.ctx, f.admin, f.cid, inv.ID)
	if err != nil {
		t.Fatal(err)
	}
	if dn.DocType != domain.DocDebitNote || dn.InvoiceRefNo != inv.FBRInvoiceNumber || !strings.HasPrefix(dn.InternalNo, "TR-DN-") {
		t.Fatalf("debit note %+v", dn)
	}
	dn, err = f.svc.Submit(f.ctx, f.admin, f.cid, dn.ID, SubmitOptions{})
	if err != nil || dn.Status != domain.StatusAccepted {
		t.Fatalf("debit note submit: %v %s %+v", err, dn.LastError, dn.FBRErrors)
	}
	// Seal chain links the debit note to the invoice.
	if dn.PrevSealHash != inv.SealHash {
		t.Fatal("seal chain broken")
	}
	// Cancellation within 72 hours.
	if _, err := f.svc.CancelInvoice(f.ctx, f.admin, f.cid, dn.ID, CancelInput{}); err == nil {
		t.Fatal("reason required")
	}
	c, err := f.svc.CancelInvoice(f.ctx, f.admin, f.cid, dn.ID, CancelInput{Reason: "Entered twice"})
	if err != nil || c.Status != domain.StatusCancelled {
		t.Fatalf("cancel: %v", err)
	}
	// Outside the window Commissioner approval is required.
	f.svc.Now = func() time.Time { return time.Now().Add(80 * time.Hour) }
	if _, err := f.svc.CancelInvoice(f.ctx, f.admin, f.cid, inv.ID, CancelInput{Reason: "late"}); err == nil || !strings.Contains(err.Error(), "Commissioner") {
		t.Fatalf("expected commissioner approval error, got %v", err)
	}
	f.svc.Now = time.Now

	// Audit chain is intact.
	broken, n, err := f.svc.Store.VerifyAuditChain(f.ctx)
	if err != nil || broken != 0 || n < 5 {
		t.Fatalf("audit chain broken=%d n=%d err=%v", broken, n, err)
	}
}

func TestLocalValidationBlocksSubmission(t *testing.T) {
	f := setup(t)
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{
		Buyer: &BuyerInput{Name: "Walk-in", Province: "Sindh", RegistrationType: "Registered"},
		Items: []ItemInput{{HSCode: "0101.2100", Description: "x", UoM: "KG", Quantity: tax.MustD("1"), UnitPrice: tax.MustD("100"), SaleType: domain.STReduced, Rate: "1%"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Submit(f.ctx, f.admin, f.cid, inv.ID, SubmitOptions{})
	se, ok := IsSubmissionError(err)
	if !ok {
		t.Fatalf("expected submission error, got %v", err)
	}
	joined := se.Error()
	for _, want := range []string{"0009", "0077", "0078"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %s in %s", want, joined)
		}
	}
	got, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID)
	if got.Status != domain.StatusDraft || len(got.Validation) == 0 {
		t.Fatalf("status %s validation %d", got.Status, len(got.Validation))
	}
	// Draft can be deleted.
	if err := f.svc.DeleteInvoice(f.ctx, f.admin, f.cid, inv.ID); err != nil {
		t.Fatal(err)
	}
}

func sandboxCompany(t *testing.T, f *fixture) {
	t.Helper()
	if _, err := f.svc.SetToken(f.ctx, f.admin, f.cid, TokenInput{Environment: domain.EnvSandbox, Token: "sandbox-token-123", Expiry: "2030-12-31"}); err != nil {
		t.Fatal(err)
	}
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	c.Environment = domain.EnvSandbox
	if _, err := f.svc.SaveCompany(f.ctx, f.admin, c); err != nil {
		t.Fatal(err)
	}
}

func TestScenariosInSandbox(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	ov, err := f.svc.Scenarios(f.ctx, f.cid)
	if err != nil || ov.AssignedCount != 11 {
		t.Fatalf("assigned %d %v", ov.AssignedCount, err)
	}
	for _, sc := range ov.Scenarios {
		if !sc.Assigned {
			continue
		}
		run, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, sc.ID, "engine", domain.EnvSandbox)
		if err != nil {
			t.Fatal(err)
		}
		if run.Status != "passed" {
			t.Errorf("%s: %s (%s)", sc.ID, run.Status, run.Message)
			continue
		}
		if inv.ScenarioID != sc.ID || !strings.HasPrefix(inv.InternalNo, "SB-") {
			t.Errorf("%s: invoice %s scenario %s", sc.ID, inv.InternalNo, inv.ScenarioID)
		}
	}
	ov, _ = f.svc.Scenarios(f.ctx, f.cid)
	if !ov.ReadyForProduction {
		t.Fatalf("passed %d of %d", ov.PassedCount, ov.AssignedCount)
	}
	// Every one of the 28 scenarios can be generated and passes the simulator rules.
	for _, sc := range domain.Scenarios {
		run, _, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, sc.ID, "engine", domain.EnvSandbox)
		if err != nil || run.Status != "passed" {
			t.Errorf("%s: %v %s %s", sc.ID, err, run.Status, run.Message)
		}
	}
}

func TestTimeoutBecomesUncertainAndIsReconciled(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	f.sim.InjectFault(fbrmock.FaultTimeout, 1)
	run, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, "SN001", "engine", domain.EnvSandbox)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Status != domain.StatusUncertain || run.Status != "failed" {
		t.Fatalf("status %s run %s", inv.Status, run.Status)
	}
	if f.sim.Count() != 1 {
		t.Fatalf("simulator should have recorded the invoice once, got %d", f.sim.Count())
	}
	// Submit is refused until reconciled.
	if _, err := f.svc.Submit(f.ctx, f.admin, f.cid, inv.ID, SubmitOptions{}); err == nil {
		t.Fatal("submit of uncertain invoice must be refused")
	}
	if _, err := f.svc.ResolveUncertain(f.ctx, f.admin, f.cid, inv.ID, ResolveInput{Action: "accepted", FBRInvoiceNumber: "1234567DI1700000000000"}); err == nil {
		t.Fatal("number of another seller must be rejected")
	}
	if _, err := f.svc.ResolveUncertain(f.ctx, f.admin, f.cid, inv.ID, ResolveInput{Action: "accepted", FBRInvoiceNumber: ntn + "DI1747119701593"}); err == nil {
		t.Fatal("the FBR date/time from IRIS must be required")
	}
	got, err := f.svc.ResolveUncertain(f.ctx, f.admin, f.cid, inv.ID, ResolveInput{Action: "accepted", FBRInvoiceNumber: ntn + "DI1747119701593",
		FBRDated: time.Now().In(PKT).Add(-96 * time.Hour).Format("2006-01-02 15:04:05"), Note: "found on IRIS"})
	if err != nil || got.Status != domain.StatusAccepted || !VerifySeal(got) {
		t.Fatalf("resolve: %v %+v", err, got)
	}
	// FBR issued it 96 hours ago: the 72-hour window has closed even though
	// the acceptance was only recorded now.
	if _, err := f.svc.CancelInvoice(f.ctx, f.admin, f.cid, inv.ID, CancelInput{Reason: "error", Reference: "IRIS-1"}); err == nil || !strings.Contains(err.Error(), "Commissioner") {
		t.Fatalf("cancellation after 72 hours from FBR's issue time must need approval, got %v", err)
	}
}

func TestOutageQueuesAndWorkerResubmits(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	f.sim.InjectFault(fbrmock.FaultUnavailable, 1)
	_, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, "SN002", "engine", domain.EnvSandbox)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Status != domain.StatusQueued || inv.NextAttemptAt == "" {
		t.Fatalf("status %s next %q", inv.Status, inv.NextAttemptAt)
	}
	// Not due yet.
	f.svc.processQueue(f.ctx)
	got, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID)
	if got.Status != domain.StatusQueued {
		t.Fatalf("status %s", got.Status)
	}
	f.svc.Now = func() time.Time { return time.Now().Add(5 * time.Minute) }
	f.svc.processQueue(f.ctx)
	f.svc.Now = time.Now
	got, _ = f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID)
	if got.Status != domain.StatusAccepted {
		t.Fatalf("after retry status %s %s", got.Status, got.LastError)
	}
}

func TestRejectionFromFBR(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{ScenarioID: "SN001",
		Buyer: &BuyerInput{NTNCNIC: "2046004", Name: "Buyer", Province: "Punjab", Address: "Lahore", RegistrationType: "Registered"},
		Items: []ItemInput{{HSCode: "0101.2100", Description: "x", UoM: "Numbers, pieces, units", Quantity: tax.MustD("1"), UnitPrice: tax.MustD("100"),
			SaleType: domain.STStandard, Rate: "17%"}}})
	if err != nil {
		t.Fatal(err)
	}
	inv, err = f.svc.Submit(f.ctx, f.admin, f.cid, inv.ID, SubmitOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Status != domain.StatusRejected || len(inv.FBRErrors) == 0 || inv.Items[0].FBRErrorCode != "0046" {
		t.Fatalf("status %s errors %+v item %+v", inv.Status, inv.FBRErrors, inv.Items[0])
	}
	// Fix and resubmit.
	inv, err = f.svc.UpdateInvoice(f.ctx, f.admin, f.cid, inv.ID, &InvoiceInput{ScenarioID: "SN001",
		Buyer: &BuyerInput{NTNCNIC: "2046004", Name: "Buyer", Province: "Punjab", Address: "Lahore", RegistrationType: "Registered"},
		Items: []ItemInput{{HSCode: "0101.2100", Description: "x", UoM: "Numbers, pieces, units", Quantity: tax.MustD("1"), UnitPrice: tax.MustD("100"),
			SaleType: domain.STStandard, Rate: "18%"}}, Submit: true})
	if err != nil || inv.Status != domain.StatusAccepted {
		t.Fatalf("resubmit: %v %s", err, inv.Status)
	}
}

func TestProductionRequiresToken(t *testing.T) {
	f := setup(t)
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	c.Environment = domain.EnvProduction
	if _, err := f.svc.SaveCompany(f.ctx, f.admin, c); err == nil || !strings.Contains(err.Error(), "production security token") {
		t.Fatalf("switching to production without a token must fail, got %v", err)
	}
	if _, err := f.svc.SetToken(f.ctx, f.admin, f.cid, TokenInput{Environment: domain.EnvProduction, Token: "production-token-123"}); err != nil {
		t.Fatal(err)
	}
	c, _ = f.svc.Store.GetCompany(f.ctx, f.cid)
	c.Environment = domain.EnvProduction
	if _, err := f.svc.SaveCompany(f.ctx, f.admin, c); err != nil {
		t.Fatal(err)
	}
	// A token cleared after go-live blocks submission until it is replaced.
	if _, err := f.svc.SetToken(f.ctx, f.admin, f.cid, TokenInput{Environment: domain.EnvProduction, Clear: true}); err != nil {
		t.Fatal(err)
	}
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{
		Buyer: &BuyerInput{Name: "Walk-in customer", Province: "Sindh", RegistrationType: "Unregistered"},
		Items: []ItemInput{{HSCode: "0101.2100", Description: "x", UoM: "KG", Quantity: tax.MustD("1"), UnitPrice: tax.MustD("100"), SaleType: domain.STStandard}}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(inv.InternalNo, "INV-") || !inv.Totals.FurtherTax.Equal(tax.MustD("4")) {
		t.Fatalf("no %s further %s", inv.InternalNo, inv.Totals.FurtherTax)
	}
	if _, err := f.svc.Submit(f.ctx, f.admin, f.cid, inv.ID, SubmitOptions{}); err == nil || !strings.Contains(err.Error(), "production security token") {
		t.Fatalf("expected token error, got %v", err)
	}
	if _, err := f.svc.SetToken(f.ctx, f.admin, f.cid, TokenInput{Environment: domain.EnvProduction, Token: "production-token-123"}); err != nil {
		t.Fatal(err)
	}
	inv, err = f.svc.Submit(f.ctx, f.admin, f.cid, inv.ID, SubmitOptions{})
	if err != nil || inv.Status != domain.StatusAccepted {
		t.Fatalf("production submit: %v %s %s", err, inv.Status, inv.LastError)
	}
	// The stored payload has no scenarioId and the token is never logged.
	if strings.Contains(inv.PayloadJSON, "scenarioId") {
		t.Error("production payload must not contain scenarioId")
	}
	calls, _, _ := f.svc.Store.ListFBRCalls(f.ctx, f.cid, inv.ID, 10, 0)
	for _, c := range calls {
		if strings.Contains(c.RequestBody+c.URL, "production-token-123") {
			t.Error("token leaked into call log")
		}
	}
	// Reports.
	reg, err := f.svc.Store.SalesRegister(f.ctx, store.ReportFilter{CompanyID: f.cid, Environment: domain.EnvProduction})
	if err != nil || len(reg) != 1 || !reg[0].FurtherTax.Equal(tax.MustD("4")) {
		t.Fatalf("register %+v %v", reg, err)
	}
}

func TestExternalRefIdempotency(t *testing.T) {
	f := setup(t)
	in := func() *InvoiceInput {
		return &InvoiceInput{ExternalRef: "ERP-1001", Source: "api",
			Buyer: &BuyerInput{Name: "Walk-in", Province: "Sindh", RegistrationType: "Unregistered"},
			Items: []ItemInput{{HSCode: "0101.2100", Description: "x", UoM: "KG", Quantity: tax.MustD("2"), UnitPrice: tax.MustD("50"), SaleType: domain.STStandard}}}
	}
	a, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, in())
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, in())
	if err != nil || a.ID != b.ID {
		t.Fatalf("duplicate created: %d vs %d (%v)", a.ID, b.ID, err)
	}
}

func TestFiscalYear(t *testing.T) {
	cases := map[string]string{"2026-10-06": "2627", "2027-03-01": "2627", "2026-06-30": "2526", "2026-07-01": "2627"}
	for d, want := range cases {
		if got := FiscalYear(d); got != want {
			t.Errorf("FiscalYear(%s) = %s, want %s", d, got, want)
		}
	}
}

func TestCancelThroughFBRAPI(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	_, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, "SN001", "engine", domain.EnvSandbox)
	if err != nil || inv.Status != domain.StatusAccepted {
		t.Fatalf("scenario: %v", err)
	}
	if _, err := f.svc.CancelInvoice(f.ctx, f.admin, f.cid, inv.ID, CancelInput{Reason: "Issued in error", UseAPI: true}); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("expected a not-configured error, got %v", err)
	}
	f.svc.Opts.Endpoints.CancelSandboxPath = "/di_data/v1/di/cancelinvoicedata_sb"

	// FBR refuses (its clock says the 72 hours have passed): nothing is recorded.
	f.sim.Now = func() time.Time { return time.Now().Add(80 * time.Hour) }
	if _, err := f.svc.CancelInvoice(f.ctx, f.admin, f.cid, inv.ID, CancelInput{Reason: "Issued in error", UseAPI: true}); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("expected FBR refusal, got %v", err)
	}
	if got, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID); got.Status != domain.StatusAccepted {
		t.Fatalf("refused cancellation must not change the invoice, status %s", got.Status)
	}

	f.sim.Now = time.Now
	got, err := f.svc.CancelInvoice(f.ctx, f.admin, f.cid, inv.ID, CancelInput{Reason: "Issued in error", UseAPI: true})
	if err != nil || got.Status != domain.StatusCancelled || !f.sim.Cancelled(inv.FBRInvoiceNumber) || !strings.Contains(got.CancelReference, "FBR API") {
		t.Fatalf("cancel via API: %v %+v", err, got)
	}
}

func TestCancelOutcome(t *testing.T) {
	cases := []struct {
		raw       string
		ok, known bool
	}{
		{`{"statusCode":"00","status":"Cancelled"}`, true, true},
		{`{"validationResponse":{"statusCode":"01","error":"not allowed"}}`, false, true},
		{`{"status code":"00"}`, true, true},
		{`{"message":"done"}`, false, false},
		{`<html>gateway</html>`, false, false},
	}
	for _, c := range cases {
		ok, known, _ := fbr.CancelOutcome([]byte(c.raw))
		if ok != c.ok || known != c.known {
			t.Errorf("%s: ok=%v known=%v", c.raw, ok, known)
		}
	}
}

// Invoices issued while FBR was unreachable are resubmitted as soon as a call
// succeeds again, without waiting for their back-off (24-hour upload rule).
func TestRecoveryResubmitsQueuedAtOnce(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	f.sim.InjectFault(fbrmock.FaultUnavailable, 2)
	for _, sc := range []string{"SN001", "SN002"} {
		_, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, sc, "engine", domain.EnvSandbox)
		if err != nil || inv.Status != domain.StatusQueued {
			t.Fatalf("%s: %v", sc, err)
		}
	}
	day := time.Now().Format("2006-01-02")
	d, err := f.svc.Store.GetDashboard(f.ctx, f.cid, domain.EnvSandbox, day, day[:8]+"01")
	if err != nil || d.PendingUpload != 2 || d.OldestPending == "" {
		t.Fatalf("dashboard pending %d %q %v", d.PendingUpload, d.OldestPending, err)
	}
	// Nothing is due yet.
	f.svc.requeueRecovered(f.ctx)
	f.svc.processQueue(f.ctx)
	if d, _ := f.svc.Store.GetDashboard(f.ctx, f.cid, domain.EnvSandbox, day, day[:8]+"01"); d.PendingUpload != 2 {
		t.Fatalf("queued invoices must wait for their back-off, pending %d", d.PendingUpload)
	}
	// The connection recovers (any successful call), so the queue is released at once.
	if _, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, "SN001", "engine", domain.EnvSandbox); err != nil || inv.Status != domain.StatusAccepted {
		t.Fatalf("recovery call: %v", err)
	}
	f.svc.requeueRecovered(f.ctx)
	f.svc.processQueue(f.ctx)
	if d, _ := f.svc.Store.GetDashboard(f.ctx, f.cid, domain.EnvSandbox, day, day[:8]+"01"); d.PendingUpload != 0 {
		t.Fatalf("queued invoices should have been resubmitted after recovery, pending %d", d.PendingUpload)
	}
}

// A buyer identified only by NTN takes its registration type from the
// customer master, so a registered customer is not charged further tax.
func TestBuyerMatchedByNTNUsesCustomerMaster(t *testing.T) {
	f := setup(t)
	f.customer(t) // 2046004, Registered, Punjab
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{
		Buyer: &BuyerInput{NTNCNIC: "2046004"},
		Items: []ItemInput{{HSCode: "3104.2000", Description: "Fertilizer", UoM: "KG", Quantity: tax.MustD("100"), UnitPrice: tax.MustD("100"),
			SaleType: domain.STStandard, Rate: "18%"}}})
	if err != nil {
		t.Fatal(err)
	}
	if inv.CustomerID == nil || inv.BuyerRegistrationType != domain.Registered || inv.BuyerName != "FERTILIZER MANUFAC IRS NEW" || inv.BuyerProvince == "" {
		t.Fatalf("buyer particulars not taken from the master: %+v", inv)
	}
	if !inv.Totals.FurtherTax.IsZero() || inv.Totals.SalesTax.String() != "1800" {
		t.Fatalf("further tax %s sales tax %s", inv.Totals.FurtherTax, inv.Totals.SalesTax)
	}
}

// A queued invoice that cannot be sent for a reason other than the network
// (here the token was removed) is deferred rather than retried on every
// worker tick, so it cannot hold up the queue.
func TestRefusedQueuedInvoiceIsDeferred(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	f.sim.InjectFault(fbrmock.FaultUnavailable, 1)
	_, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, "SN001", "engine", domain.EnvSandbox)
	if err != nil || inv.Status != domain.StatusQueued {
		t.Fatalf("queue: %v", err)
	}
	if _, err := f.svc.SetToken(f.ctx, f.admin, f.cid, TokenInput{Environment: domain.EnvSandbox, Clear: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Store.RequeueNow(f.ctx, f.cid, domain.EnvSandbox); err != nil {
		t.Fatal(err)
	}
	f.svc.processQueue(f.ctx)
	got, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID)
	next := store.ParseTime(got.NextAttemptAt)
	if got.Status != domain.StatusQueued || time.Until(next) < 10*time.Minute || !strings.Contains(got.LastError, "token") {
		t.Fatalf("status %s next %q error %q", got.Status, got.NextAttemptAt, got.LastError)
	}
}

// An invoice issued while FBR was unreachable stays in the "not yet reported"
// count (24-hour upload rule) when its upload is rejected after recovery,
// until FBR accepts it.
func TestOfflineInvoiceTrackedUntilAccepted(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{ScenarioID: "SN001",
		Buyer: &BuyerInput{NTNCNIC: "2046004", Name: "Buyer", Province: "Punjab", Address: "Lahore", RegistrationType: "Registered"},
		Items: []ItemInput{{HSCode: "0101.2100", Description: "x", UoM: "Numbers, pieces, units", Quantity: tax.MustD("1"), UnitPrice: tax.MustD("100"),
			SaleType: domain.STStandard, Rate: "17%"}}})
	if err != nil {
		t.Fatal(err)
	}
	f.sim.InjectFault(fbrmock.FaultUnavailable, 1)
	if inv, err = f.svc.Submit(f.ctx, f.admin, f.cid, inv.ID, SubmitOptions{}); err != nil || inv.Status != domain.StatusQueued || inv.OfflineSince == "" {
		t.Fatalf("queue: %v %s %q", err, inv.Status, inv.OfflineSince)
	}
	_, _ = f.svc.Store.RequeueNow(f.ctx, f.cid, domain.EnvSandbox)
	f.svc.processQueue(f.ctx) // FBR is back but rejects the rate (0046)
	got, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID)
	if got.Status != domain.StatusRejected {
		t.Fatalf("expected rejection after recovery, got %s", got.Status)
	}
	day := time.Now().Format("2006-01-02")
	d, _ := f.svc.Store.GetDashboard(f.ctx, f.cid, domain.EnvSandbox, day, day[:8]+"01")
	if d.PendingUpload != 1 || d.OldestPending != got.OfflineSince {
		t.Fatalf("rejected offline invoice must stay pending: %d %q", d.PendingUpload, d.OldestPending)
	}
}

// Sample mode reproduces FBR's published sample amounts, including a sample
// with FED (SN005), although the engine adds FED to the value of supply.
func TestScenarioSampleModeKeepsFBRValues(t *testing.T) {
	f := setup(t)
	sc, ok := domain.LookupScenario("SN005")
	if !ok {
		t.Fatal("SN005 missing")
	}
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	in := ScenarioInvoiceInput(sc, c, domain.EnvSimulator, "sample", time.Now().In(PKT).Format("2006-01-02"))
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, in)
	if err != nil {
		t.Fatal(err)
	}
	p := f.svc.BuildPayload(c, inv)
	it := p.Items[0]
	if !it.ValueSalesExcludingST.Value.Equal(tax.F(sc.Item.ValueSalesExcludingST)) || !it.FEDPayable.Value.Equal(tax.F(sc.Item.FEDPayable)) ||
		!it.SalesTaxApplicable.Value.Equal(tax.F(sc.Item.SalesTaxApplicable)) {
		t.Fatalf("sample not reproduced: value %s fed %s tax %s", it.ValueSalesExcludingST.Value, it.FEDPayable.Value, it.SalesTaxApplicable.Value)
	}
}

func TestEqualNumberPrefixesRefused(t *testing.T) {
	f := setup(t)
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	c.InvoicePrefix, c.DebitNotePrefix = "INV", "inv"
	if _, err := f.svc.SaveCompany(f.ctx, f.admin, c); err == nil || !strings.Contains(err.Error(), "prefixes must be different") {
		t.Fatalf("equal prefixes accepted: %v", err)
	}
}

// The caller going away (browser closed, ERP timeout) during the exchange
// with FBR must not leave the invoice stuck in SUBMITTING with FBR's answer
// lost: the outcome is still recorded.
func TestCancelledCallerStillRecordsOutcome(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	c.ValidateBeforePost = false
	if _, err := f.svc.SaveCompany(f.ctx, f.admin, c); err != nil {
		t.Fatal(err)
	}
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, ScenarioInvoiceInput(mustScenario(t, "SN001"), c, domain.EnvSandbox, "engine", time.Now().In(PKT).Format("2006-01-02")))
	if err != nil {
		t.Fatal(err)
	}
	f.sim.InjectFault(fbrmock.FaultTimeout, 1) // FBR records it, then answers too late
	ctx, cancel := context.WithCancel(f.ctx)
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	_, _ = f.svc.Submit(ctx, f.admin, f.cid, inv.ID, SubmitOptions{})
	got, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID)
	if got.Status == domain.StatusSubmitting || got.Status != domain.StatusUncertain {
		t.Fatalf("status %s after the caller went away (want UNCERTAIN, FBR recorded it)", got.Status)
	}
}

func mustScenario(t *testing.T, id string) domain.Scenario {
	t.Helper()
	sc, ok := domain.LookupScenario(id)
	if !ok {
		t.Fatalf("scenario %s missing", id)
	}
	return sc
}

// Interrupted submissions are released by the worker while it runs, not
// only at startup, and at startup regardless of age.
func TestStuckSubmissionsReleased(t *testing.T) {
	f := setup(t)
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{
		Buyer: &BuyerInput{NTNCNIC: "2046004", Name: "Buyer", Province: "Punjab", Address: "Lahore", RegistrationType: "Registered"},
		Items: []ItemInput{{HSCode: "3104.2000", Description: "x", UoM: "KG", Quantity: tax.MustD("1"), UnitPrice: tax.MustD("100"), SaleType: domain.STStandard, Rate: "18%"}}})
	if err != nil {
		t.Fatal(err)
	}
	set := func(age time.Duration, payload string) {
		_, err := f.svc.Store.DB.ExecContext(f.ctx, `UPDATE invoices SET status='SUBMITTING', payload_json=?, updated_at=? WHERE id=?`,
			payload, time.Now().UTC().Add(-age).Format(time.RFC3339), inv.ID)
		if err != nil {
			t.Fatal(err)
		}
	}
	status := func() domain.InvoiceStatus {
		got, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID)
		return got.Status
	}
	// Still within a possible in-flight exchange: left alone.
	set(30*time.Second, "{}")
	f.svc.recoverStuck(f.ctx, f.svc.Now().UTC().Add(-f.svc.stuckAfter()))
	if status() != domain.StatusSubmitting {
		t.Fatalf("in-flight submission released too early: %s", status())
	}
	// Older than any exchange can take: released for reconciliation.
	set(f.svc.stuckAfter()+time.Minute, "{}")
	f.svc.recoverStuck(f.ctx, f.svc.Now().UTC().Add(-f.svc.stuckAfter()))
	if status() != domain.StatusUncertain {
		t.Fatalf("stuck submission not released: %s", status())
	}
	// At startup even a fresh one is released; never sent -> queued again.
	set(5*time.Second, "")
	f.svc.recoverStuck(f.ctx, f.svc.Now().UTC().Add(time.Second))
	if status() != domain.StatusQueued {
		t.Fatalf("startup recovery: %s", status())
	}
}

// A queued invoice that fails local validation when retried keeps the
// validation issues so the operator knows what to correct.
func TestQueuedInvoiceRejectedLocallyKeepsIssues(t *testing.T) {
	f := setup(t)
	sandboxCompany(t, f)
	f.sim.InjectFault(fbrmock.FaultUnavailable, 1)
	_, inv, err := f.svc.RunScenario(f.ctx, f.admin, f.cid, "SN001", "engine", domain.EnvSandbox)
	if err != nil || inv.Status != domain.StatusQueued {
		t.Fatalf("queue: %v", err)
	}
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `UPDATE invoice_items SET hs_code='' WHERE invoice_id=?`, inv.ID); err != nil {
		t.Fatal(err)
	}
	_, _ = f.svc.Store.RequeueNow(f.ctx, f.cid, domain.EnvSandbox)
	f.svc.processQueue(f.ctx)
	got, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, inv.ID)
	if got.Status != domain.StatusRejected || len(got.Validation) == 0 {
		t.Fatalf("status %s issues %v", got.Status, got.Validation)
	}
}
