// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
)

// register is a sales register in a business's own layout: a title, its
// own headings, day-first dates, "Rs" amounts, a unit FBR spells
// differently, no province and no sale type, and a totals line.
var register = [][]string{
	{"Indus Steel & Trading (Pvt) Ltd — Sales Register, October 2026"},
	{},
	{"Inv #", "Date", "Customer Name", "NTN/CNIC", "Address", "Item Description", "HS Code", "Unit", "Qty", "Rate", "Amount", "GST %", "GST", "Total"},
	{"INV-1001", "01/10/2026", "Punjab Builders Co.", "2046004", "Raiwind Road, Lahore", "Cement bag 50kg", "2523.2900", "Bags", "100", "Rs 1,200.00", "120,000.00", "18%", "21,600.00", "141,600.00"},
	{"INV-1001", "01/10/2026", "Punjab Builders Co.", "2046004", "Raiwind Road, Lahore", "Cement bag 50kg", "2523.2900", "Bags", "10", "1,200", "12,000.00", "18%", "2,160.00", "14,160.00"},
	{"INV-1002", "05/10/2026", "Walk-in Retail Buyer", "", "Saddar, Karachi", "Cement bag 50kg", "2523.2900", "Cartons", "1", "1,200.00", "1,200.00", "18%", "216.00", "1,416.00"},
	{"Grand Total", "", "", "", "", "", "", "", "", "", "133,200.00", "", "23,976.00", "157,176.00"},
}

func registerCSV() []byte {
	var b bytes.Buffer
	_ = csv.NewWriter(&b).WriteAll(register)
	return b.Bytes()
}

func TestSmartImport(t *testing.T) {
	f := setup(t)
	data := registerCSV()
	a, err := f.svc.AnalyzeImport(f.ctx, f.cid, "register.csv", data, -1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.HeaderRow != 2 || a.Rows != 3 {
		t.Fatalf("header row %d, rows %d", a.HeaderRow, a.Rows)
	}
	want := map[string]int{"invoice_ref": 0, "invoice_date": 1, "buyer_name": 2, "buyer_ntn_cnic": 3, "buyer_address": 4, "description": 5,
		"hs_code": 6, "uom": 7, "quantity": 8, "unit_price": 9, "value_excl_st": 10, "rate": 11, "sales_tax": 12, "total_value": 13}
	for k, v := range want {
		if got, ok := a.Mapping[k]; !ok || got != v {
			t.Errorf("mapping %s = %d (%v), want %d", k, got, ok, v)
		}
	}
	missing := map[string]MissingInfo{}
	for _, m := range a.Missing {
		missing[m.Field+"/"+m.Input] = m
	}
	prov, ok := missing["buyer_province/province"]
	if !ok || !strings.Contains(prov.Message, "worked out from the city in the address for 3 lines") {
		t.Errorf("province prompt: %+v", prov)
	}
	if st, ok := missing["sale_type/saletype"]; !ok || st.Suggest != "Goods at standard rate (default)" {
		t.Errorf("sale type prompt: %+v", st)
	}
	units, ok := missing["uom/valuemap"]
	if !ok || len(units.Values) != 1 || units.Values[0].Value != "Cartons" || units.Values[0].Lines != 1 {
		t.Errorf("unit translation prompt: %+v", units)
	}
	if _, ok := missing["buyer_registration_type/regrule"]; !ok {
		t.Error("registration rule prompt missing")
	}

	opts := ImportOptions{Sheet: a.Sheet, Mapping: a.Mapping, ProvinceFromAddress: true,
		Defaults: map[string]string{"sale_type": "Goods at standard rate (default)", "buyer_province": "PUNJAB"},
		ValueMap: map[string]map[string]string{"uom": {"Cartons": "Packs"}}}
	sum, err := f.svc.SmartImport(f.ctx, f.admin, f.cid, "register.csv", data, opts, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Invoices != 2 || len(sum.Results) != 2 {
		t.Fatalf("summary %+v", sum)
	}
	r0, r1 := sum.Results[0], sum.Results[1]
	if r0.Ref != "INV-1001" || r0.Lines != 2 || r0.Rows[0] != 4 || r0.Rows[1] != 5 || r0.Date != "2026-10-01" || r0.Total.String() != "155760" || len(r0.Warnings) != 0 {
		t.Fatalf("first invoice %+v", r0)
	}
	// The walk-in buyer is unregistered: further tax makes the computed
	// total differ from the file's, which is pointed out.
	if r1.Ref != "INV-1002" || r1.Status == "error" || len(r1.Warnings) != 1 {
		t.Fatalf("second invoice %+v", r1)
	}

	// Importing creates drafts; importing the same file again skips them.
	sum, err = f.svc.SmartImport(f.ctx, f.admin, f.cid, "register.csv", data, opts, false, false)
	if err != nil || sum.OK != 2 {
		t.Fatalf("import: %v %+v", err, sum)
	}
	inv, err := f.svc.Store.GetInvoice(f.ctx, f.cid, sum.Results[0].InvoiceID)
	if err != nil || inv.BuyerProvince != "PUNJAB" || inv.BuyerRegistrationType != "Registered" || len(inv.Items) != 2 || inv.Items[0].UoM != "Bag" {
		t.Fatalf("invoice %+v", inv)
	}
	inv2, _ := f.svc.Store.GetInvoice(f.ctx, f.cid, sum.Results[1].InvoiceID)
	if inv2.BuyerProvince != "SINDH" || inv2.Items[0].UoM != "Packs" || inv2.BuyerRegistrationType != "Unregistered" {
		t.Fatalf("second invoice %+v", inv2)
	}
	again, err := f.svc.SmartImport(f.ctx, f.admin, f.cid, "register.csv", data, opts, false, false)
	if err != nil || len(again.Results) != 2 || again.Results[0].InvoiceID != sum.Results[0].InvoiceID || again.Results[1].InvoiceID != sum.Results[1].InvoiceID {
		t.Fatalf("re-importing must return the invoices already created, not duplicates: %v %+v", err, again)
	}

	// Without invoice numbers, lines are grouped by buyer and date.
	noRef := ImportOptions{Sheet: a.Sheet, Mapping: map[string]int{}, Grouping: "buyer-date", ProvinceFromAddress: true,
		Defaults: map[string]string{"sale_type": "Goods at standard rate (default)", "uom": "Bag"}}
	for k, v := range a.Mapping {
		if k != "invoice_ref" && k != "uom" {
			noRef.Mapping[k] = v
		}
	}
	sum, err = f.svc.SmartImport(f.ctx, f.admin, f.cid, "register.csv", data, noRef, true, false)
	if err != nil || sum.Invoices != 2 || !strings.HasPrefix(sum.Results[0].Ref, "IMP-") || sum.Results[0].Lines != 2 {
		t.Fatalf("grouping: %v %+v", err, sum)
	}
}
