// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestDashboardSearchAlertsAndCompliance(t *testing.T) {
	e := newEnv(t)
	base := fmt.Sprintf("/companies/%d", e.compA)
	st, b := e.adminA.do("POST", base+"/invoices", sampleInvoice(nil))
	if st != 201 {
		t.Fatalf("invoice: %d %s", st, b)
	}
	var inv struct {
		ID         int64  `json:"id"`
		InternalNo string `json:"internalNo"`
	}
	_ = json.Unmarshal(b, &inv)

	// Dashboard: deadlines, recent invoices and (for report users) the trend.
	st, b = e.adminA.do("GET", base+"/dashboard", nil)
	if st != 200 {
		t.Fatalf("dashboard: %d %s", st, b)
	}
	var dash struct {
		Deadlines []struct{ Kind, Due string } `json:"deadlines"`
		Recent    []struct{ ID int64 }         `json:"recent"`
		Trend     []struct {
			Period       string `json:"period"`
			SaleInvoices int    `json:"saleInvoices"`
		} `json:"trend"`
		TopBuyers []struct{ BuyerName string } `json:"topBuyers"`
		TopItems  []struct{ HSCode string }    `json:"topItems"`
		Reference struct {
			HSCodes  int    `json:"hsCodes"`
			HSSource string `json:"hsSource"`
		} `json:"reference"`
	}
	if err := json.Unmarshal(b, &dash); err != nil {
		t.Fatal(err)
	}
	if len(dash.Deadlines) != 2 || len(dash.Recent) != 1 || len(dash.Trend) != 12 || dash.Trend[11].SaleInvoices != 1 ||
		len(dash.TopBuyers) != 1 || len(dash.TopItems) != 1 || dash.TopItems[0].HSCode != "0101.2100" ||
		dash.Reference.HSCodes == 0 || dash.Reference.HSSource != "seed" {
		t.Fatalf("dashboard %s", b)
	}
	// Operators may see the dashboard but not sales by month, buyer or item.
	op := e.login("operatorA", "Initial123")
	st, b = op.do("GET", base+"/dashboard", nil)
	if st != 200 || strings.Contains(string(b), `"trend"`) || strings.Contains(string(b), `"topBuyers"`) {
		t.Fatalf("operator dashboard: %d %s", st, b)
	}
	if st, _ := op.do("GET", base+"/compliance", nil); st != 403 {
		t.Fatalf("operator compliance review: got %d, want 403", st)
	}

	// Search finds the invoice by number and the HS code by description.
	st, b = e.adminA.do("GET", base+"/search?q="+inv.InternalNo, nil)
	if st != 200 || !strings.Contains(string(b), inv.InternalNo) {
		t.Fatalf("search invoice: %d %s", st, b)
	}
	st, b = e.adminA.do("GET", base+"/search?q=0101", nil)
	if st != 200 || !strings.Contains(string(b), `"hsCodes":[{"code":"0101`) {
		t.Fatalf("search HS code: %d %s", st, b)
	}
	if st, b = e.adminA.do("GET", base+"/search?q=x", nil); st != 200 || !strings.Contains(string(b), `"invoices":[]`) {
		t.Fatalf("short query: %d %s", st, b)
	}
	// Another company's data is not searchable.
	if st, _ := op.do("GET", fmt.Sprintf("/companies/%d/search?q=Buyer", e.compB), nil); st != 403 {
		t.Fatalf("other company search: got %d, want 403", st)
	}

	// Duplicate creates a draft; alerts then report the unissued draft.
	st, b = e.adminA.do("POST", fmt.Sprintf("%s/invoices/%d/duplicate", base, inv.ID), nil)
	if st != 201 || !strings.Contains(string(b), `"status":"DRAFT"`) {
		t.Fatalf("duplicate: %d %s", st, b)
	}
	st, b = e.adminA.do("GET", base+"/alerts", nil)
	if st != 200 || !strings.Contains(string(b), `"id":"drafts"`) || !strings.Contains(string(b), `"id":"backup"`) {
		t.Fatalf("alerts: %d %s", st, b)
	}
	// Only system administrators are told about backups.
	if st, b = op.do("GET", base+"/alerts", nil); st != 200 || strings.Contains(string(b), `"id":"backup"`) {
		t.Fatalf("operator alerts: %d %s", st, b)
	}

	// Period review for the return.
	st, b = e.adminA.do("GET", base+"/compliance", nil)
	if st != 200 || !strings.Contains(string(b), `"checks":[`) || !strings.Contains(string(b), `"deadlines":[`) {
		t.Fatalf("compliance: %d %s", st, b)
	}
	if st, _ = e.adminA.do("GET", base+"/compliance?period=bad", nil); st != 422 {
		t.Fatalf("bad period: got %d, want 422", st)
	}

	// FBR document types are always available.
	if st, b = e.adminA.do("GET", base+"/ref/doc-types", nil); st != 200 || !strings.Contains(string(b), "Debit Note") {
		t.Fatalf("doc types: %d %s", st, b)
	}
}
