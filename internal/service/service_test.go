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
	got, err := f.svc.ResolveUncertain(f.ctx, f.admin, f.cid, inv.ID, ResolveInput{Action: "accepted", FBRInvoiceNumber: ntn + "DI1747119701593", Note: "found on IRIS"})
	if err != nil || got.Status != domain.StatusAccepted || !VerifySeal(got) {
		t.Fatalf("resolve: %v %+v", err, got)
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
