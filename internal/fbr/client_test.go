// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

package fbr_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/fbrmock"
	"einvoicing/internal/tax"
)

const sellerNTN = "0786909"

func samplePayload() *fbr.InvoicePayload {
	return &fbr.InvoicePayload{
		InvoiceType: "Sale Invoice", InvoiceDate: time.Now().Format("2006-01-02"),
		SellerNTNCNIC: sellerNTN, SellerBusinessName: "Company 8", SellerProvince: "SINDH", SellerAddress: "Karachi",
		BuyerNTNCNIC: "2046004", BuyerBusinessName: "FERTILIZER MANUFAC IRS NEW", BuyerProvince: "SINDH", BuyerAddress: "Karachi",
		BuyerRegistrationType: "Registered", ScenarioID: "SN001",
		Items: []fbr.ItemPayload{{
			HSCode: "0101.2100", ProductDescription: "test", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: fbr.Q(tax.MustD("1")), ValueSalesExcludingST: fbr.A(tax.MustD("1000")),
			SalesTaxApplicable: fbr.A(tax.MustD("180")), TotalValues: fbr.A(tax.MustD("1180")),
			SaleType: domain.STStandard,
		}},
	}
}

func newPair(t *testing.T, timeout time.Duration) (*fbrmock.Server, *fbr.Client, func()) {
	t.Helper()
	sim := fbrmock.New()
	sim.AddToken("sb-token", sellerNTN, true)
	sim.AddToken("prod-token", sellerNTN, false)
	ts := httptest.NewServer(sim.Handler())
	c := fbr.New(domain.EnvSandbox, "sb-token", fbr.Endpoints{BaseURL: ts.URL}, fbr.NewHTTPClient(timeout))
	return sim, c, ts.Close
}

func TestPostInvoiceSuccess(t *testing.T) {
	sim, c, done := newPair(t, 5*time.Second)
	defer done()
	var logs []fbr.CallLog
	c.OnCall = func(l fbr.CallLog) { logs = append(logs, l) }
	resp, err := c.PostInvoice(context.Background(), samplePayload())
	if err != nil {
		t.Fatal(err)
	}
	if !resp.IsValid() {
		t.Fatalf("expected valid, got %+v", resp.Errors())
	}
	if !strings.HasPrefix(string(resp.InvoiceNumber), sellerNTN+"DI") {
		t.Errorf("invoice number %q", resp.InvoiceNumber)
	}
	if got := string(resp.ValidationResponse.InvoiceStatuses[0].InvoiceNo); got != string(resp.InvoiceNumber)+"-1" {
		t.Errorf("item invoice no %q", got)
	}
	if sim.Count() != 1 {
		t.Errorf("recorded %d", sim.Count())
	}
	if len(logs) != 1 || logs[0].HTTPStatus != 200 || strings.Contains(string(logs[0].RequestBody), "sb-token") {
		t.Errorf("bad call log %+v", logs)
	}
}

func TestValidateDoesNotRecord(t *testing.T) {
	sim, c, done := newPair(t, 5*time.Second)
	defer done()
	resp, err := c.ValidateInvoice(context.Background(), samplePayload())
	if err != nil || !resp.IsValid() {
		t.Fatalf("validate: %v %+v", err, resp)
	}
	if resp.InvoiceNumber != "" || sim.Count() != 0 {
		t.Error("validate must not issue a number or record the invoice")
	}
}

func TestRejectionIsNotAnError(t *testing.T) {
	_, c, done := newPair(t, 5*time.Second)
	defer done()
	p := samplePayload()
	p.Items[0].SalesTaxApplicable = fbr.A(tax.MustD("170"))
	resp, err := c.PostInvoice(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if resp.IsValid() {
		t.Fatal("expected rejection")
	}
	errs := resp.Errors()
	if len(errs) != 1 || errs[0].Item != 1 || errs[0].Code != "0102" {
		t.Fatalf("errors %+v", errs)
	}
}

func TestHeaderErrorWrongSeller(t *testing.T) {
	_, c, done := newPair(t, 5*time.Second)
	defer done()
	p := samplePayload()
	p.SellerNTNCNIC = "1234567"
	resp, err := c.PostInvoice(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if resp.IsValid() || resp.Errors()[0].Code != "0401" {
		t.Fatalf("expected 0401, got %+v", resp.Errors())
	}
}

func TestScenarioIdStrippedInProduction(t *testing.T) {
	var seen map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&seen)
		_, _ = w.Write([]byte(`{"invoiceNumber":"0786909DI1","dated":"2025-01-01 10:00:00","validationResponse":{"statusCode":"00","status":"Valid","error":"","invoiceStatuses":[{"itemSNo":1,"statusCode":"00","status":"Valid","invoiceNo":"0786909DI1-1","errorCode":"","error":""},]}}`))
	}))
	defer ts.Close()
	c := fbr.New(domain.EnvProduction, "t", fbr.Endpoints{BaseURL: ts.URL}, nil)
	resp, err := c.PostInvoice(context.Background(), samplePayload())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := seen["scenarioId"]; ok {
		t.Error("scenarioId must not be sent to production")
	}
	if !resp.IsValid() || resp.ValidationResponse.InvoiceStatuses[0].ItemSNo != "1" {
		t.Errorf("lenient decode failed: %+v", resp)
	}
}

func TestUnauthorized(t *testing.T) {
	_, c, done := newPair(t, 5*time.Second)
	defer done()
	c.Token = "wrong"
	_, err := c.PostInvoice(context.Background(), samplePayload())
	if fbr.KindOf(err) != fbr.ErrAuth {
		t.Fatalf("expected auth error, got %v", err)
	}
}

func TestTimeoutAfterRecordingIsUncertain(t *testing.T) {
	sim, c, done := newPair(t, 300*time.Millisecond)
	defer done()
	sim.TimeoutDelay = 1 * time.Second
	sim.InjectFault(fbrmock.FaultTimeout, 1)
	_, err := c.PostInvoice(context.Background(), samplePayload())
	if fbr.KindOf(err) != fbr.ErrUncertain {
		t.Fatalf("expected uncertain, got %v", err)
	}
	if sim.Count() != 1 {
		t.Fatal("the simulator recorded the invoice; the client must treat the outcome as uncertain")
	}
}

func TestServiceUnavailableIsRetryable(t *testing.T) {
	sim, c, done := newPair(t, 5*time.Second)
	defer done()
	sim.InjectFault(fbrmock.FaultUnavailable, 1)
	_, err := c.PostInvoice(context.Background(), samplePayload())
	var ce *fbr.CallError
	if fbr.KindOf(err) != fbr.ErrUnavailable {
		t.Fatalf("expected unavailable, got %v", err)
	}
	if ok := asCallError(err, &ce); !ok || !ce.Retryable() {
		t.Error("503 must be retryable")
	}
	if sim.Count() != 0 {
		t.Error("nothing should be recorded")
	}
}

func TestDroppedConnectionIsUncertain(t *testing.T) {
	sim, c, done := newPair(t, 5*time.Second)
	defer done()
	sim.InjectFault(fbrmock.FaultDrop, 1)
	_, err := c.PostInvoice(context.Background(), samplePayload())
	if fbr.KindOf(err) != fbr.ErrUncertain {
		t.Fatalf("expected uncertain, got %v", err)
	}
}

func TestConnectionRefusedIsNotSent(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	url := ts.URL
	ts.Close()
	c := fbr.New(domain.EnvSandbox, "t", fbr.Endpoints{BaseURL: url}, fbr.NewHTTPClient(2*time.Second))
	_, err := c.PostInvoice(context.Background(), samplePayload())
	if fbr.KindOf(err) != fbr.ErrNotSent {
		t.Fatalf("expected not_sent, got %v", err)
	}
}

func TestReferenceAPIs(t *testing.T) {
	_, c, done := newPair(t, 5*time.Second)
	defer done()
	ctx := context.Background()
	prov, err := c.Provinces(ctx)
	if err != nil || len(prov) == 0 {
		t.Fatalf("provinces %v %v", prov, err)
	}
	tt, err := c.TransTypes(ctx)
	if err != nil || len(tt) < 20 {
		t.Fatalf("transtypes %d %v", len(tt), err)
	}
	var stdID int
	for _, x := range tt {
		if x.Description == domain.STReduced {
			stdID = x.ID
		}
	}
	rates, err := c.SaleTypeToRate(ctx, time.Now(), stdID, 8)
	if err != nil || len(rates) == 0 || rates[0].Description != "1%" {
		t.Fatalf("rates %+v %v", rates, err)
	}
	scheds, err := c.SROSchedule(ctx, rates[0].ID, time.Now(), 8)
	if err != nil || len(scheds) == 0 {
		t.Fatalf("schedules %+v %v", scheds, err)
	}
	items, err := c.SROItems(ctx, time.Now(), scheds[0].ID)
	if err != nil || len(items) == 0 {
		t.Fatalf("sro items %+v %v", items, err)
	}
	uoms, err := c.HSUOM(ctx, "5904.9000", 3)
	if err != nil || len(uoms) != 1 || uoms[0].Description != "Square Metre" {
		t.Fatalf("hs uom %+v %v", uoms, err)
	}
	st, err := c.STATL(ctx, "2046004", time.Now())
	if err != nil || !st.Active {
		t.Fatalf("statl %+v %v", st, err)
	}
	st2, err := c.STATL(ctx, "1000000000000", time.Now())
	if err != nil || st2.Active {
		t.Fatalf("statl inactive %+v %v", st2, err)
	}
	rt, err := c.RegType(ctx, "2046004")
	if err != nil || !rt.Registered {
		t.Fatalf("regtype %+v %v", rt, err)
	}
	hs, err := c.ItemDescCodes(ctx)
	if err != nil || len(hs) == 0 {
		t.Fatalf("hs codes %v", err)
	}
}

func TestAmountJSON(t *testing.T) {
	it := fbr.ItemPayload{ExtraTax: fbr.EmptyAmount(), SalesTaxApplicable: fbr.A(tax.MustD("180")), Quantity: fbr.Q(tax.MustD("1.5"))}
	b, _ := json.Marshal(it)
	s := string(b)
	if !strings.Contains(s, `"extraTax":""`) || !strings.Contains(s, `"salesTaxApplicable":180.00`) || !strings.Contains(s, `"quantity":1.5000`) {
		t.Fatalf("unexpected json %s", s)
	}
	var back fbr.ItemPayload
	if err := json.Unmarshal([]byte(`{"extraTax":"","furtherTax":"12.5","quantity":"2","discount":null}`), &back); err != nil {
		t.Fatal(err)
	}
	if !back.ExtraTax.Empty || !back.FurtherTax.Value.Equal(tax.MustD("12.5")) || !back.Quantity.Value.Equal(tax.MustD("2")) {
		t.Fatalf("decode %+v", back)
	}
}

func asCallError(err error, target **fbr.CallError) bool {
	ce, ok := err.(*fbr.CallError)
	if ok {
		*target = ce
	}
	return ok
}
