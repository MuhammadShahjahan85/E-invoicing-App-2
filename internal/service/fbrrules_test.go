// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"regexp"
	"strings"
	"testing"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/printing"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"
)

// pkTime returns a fixed time in Pakistan.
func pkTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.ParseInLocation("2006-01-02 15:04", s, PKT)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func (f *fixture) setClock(t time.Time) { f.svc.Now = func() time.Time { return t } }

func (f *fixture) accepted(t *testing.T, cust *store.Customer, prod *store.Product) *store.Invoice {
	t.Helper()
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{CustomerID: &cust.ID,
		Items: []ItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("40")}}, Submit: true})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Status != domain.StatusAccepted {
		t.Fatalf("status %s: %s %+v", inv.Status, inv.LastError, inv.FBRErrors)
	}
	return inv
}

func TestInvoiceDigitalSignature(t *testing.T) {
	f := setup(t)
	inv := f.accepted(t, f.customer(t), f.product(t))
	if inv.Signature == "" || !f.svc.SignatureValid(f.ctx, inv) {
		t.Fatalf("accepted invoice is not signed: %q", inv.Signature)
	}
	// The signature covers the FBR invoice number and the seal.
	for _, forge := range []func(*store.Invoice){
		func(i *store.Invoice) { i.FBRInvoiceNumber += "0" },
		func(i *store.Invoice) { i.SealHash = strings.Repeat("0", len(i.SealHash)) },
	} {
		forged := *inv
		forge(&forged)
		if f.svc.SignatureValid(f.ctx, &forged) {
			t.Fatal("signature still valid after the invoice was changed")
		}
	}

	// The key is kept (encrypted) and reloaded after a restart.
	f.svc.signKey = nil
	if !f.svc.SignatureValid(f.ctx, inv) {
		t.Fatal("signature not valid after reloading the key")
	}
	k, err := f.svc.SigningKey(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if k.Algorithm != "Ed25519" || !regexp.MustCompile(`^[0-9A-F]{4}(-[0-9A-F]{4}){3}$`).MatchString(k.Fingerprint) {
		t.Fatalf("key info %+v", k)
	}
	block, _ := pem.Decode([]byte(k.PublicKey))
	if block == nil {
		t.Fatal("public key is not PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	sig, _ := base64.StdEncoding.DecodeString(inv.Signature)
	if !ed25519.Verify(pub.(ed25519.PublicKey), signatureMessage(inv), sig) {
		t.Fatal("published public key does not verify the invoice")
	}

	// The printed invoice carries the signature.
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	var buf bytes.Buffer
	if err := printing.Render(&buf, c, inv, printing.Options{SigningKey: k.Fingerprint}); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); !strings.Contains(out, printing.ShortSignature(inv.Signature)) || !strings.Contains(out, k.Fingerprint) {
		t.Fatal("printed invoice does not show the digital signature")
	}

	// A recorded signature cannot be changed, and a forged one is caught.
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `UPDATE invoices SET signature='x' WHERE id=?`, inv.ID); err == nil {
		t.Fatal("signature was changed")
	}
	if rep, err := f.svc.CheckIntegrity(f.ctx); err != nil || !rep.OK() {
		t.Fatalf("integrity %+v %v", rep, err)
	}
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `DROP TRIGGER trg_invoices_immutable`); err != nil {
		t.Fatal(err)
	}
	fake := base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `UPDATE invoices SET signature=? WHERE id=?`, fake, inv.ID); err != nil {
		t.Fatal(err)
	}
	rep, err := f.svc.CheckIntegrity(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rep.OK() || !strings.Contains(strings.Join(rep.Problems, ";"), "digital signature does not match") {
		t.Fatalf("forged signature not reported: %+v", rep)
	}
}

func TestClosingPeriods(t *testing.T) {
	thu := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	for kind, want := range map[string]string{ClosingDay: "2026-10-07", ClosingWeek: "2026-W40", ClosingMonth: "2026-09"} {
		if got := duePeriods(kind, thu, nil); len(got) != 1 || got[0].key != want {
			t.Errorf("%s: %+v, want %s", kind, got, want)
		}
	}
	if p := periodOf(ClosingWeek, thu); p.start != "2026-10-05" || p.end != "2026-10-11" || p.key != "2026-W41" {
		t.Errorf("week %+v", p)
	}
	// A week or month still running is never closed.
	mon := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	if got := duePeriods(ClosingWeek, mon, nil); got[0].key != "2026-W40" || got[0].end != "2026-10-04" {
		t.Errorf("monday %+v", got)
	}
	first := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	if got := duePeriods(ClosingMonth, first, nil); got[0].key != "2026-10" || got[0].end != "2026-10-31" {
		t.Errorf("first of month %+v", got)
	}
	// Catch-up after the server was off.
	got := duePeriods(ClosingDay, thu, &store.Closing{PeriodEnd: "2026-10-04"})
	if len(got) != 3 || got[0].key != "2026-10-05" || got[2].key != "2026-10-07" {
		t.Errorf("catch-up %+v", got)
	}
	if got := duePeriods(ClosingDay, thu, &store.Closing{PeriodEnd: "2026-10-07"}); len(got) != 0 {
		t.Errorf("already closed %+v", got)
	}
	// ISO week 53 and the turn of the year.
	if p := periodOf(ClosingWeek, time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)); p.key != "2026-W53" || p.start != "2026-12-28" {
		t.Errorf("year end %+v", p)
	}
}

func TestDayWeekMonthClosings(t *testing.T) {
	f := setup(t)
	f.setClock(pkTime(t, "2026-09-30 11:00")) // Wednesday
	inv := f.accepted(t, f.customer(t), f.product(t))
	if inv.InvoiceDate != "2026-09-30" {
		t.Fatalf("invoice date %s", inv.InvoiceDate)
	}

	f.setClock(pkTime(t, "2026-10-01 09:00"))
	n, err := f.svc.CloseDuePeriods(f.ctx)
	if err != nil || n != 3 {
		t.Fatalf("closings %d %v", n, err)
	}
	if n, _ := f.svc.CloseDuePeriods(f.ctx); n != 0 {
		t.Fatalf("closed twice: %d", n)
	}
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	list, err := f.svc.Closings(f.ctx, c, c.Environment, "", 50)
	if err != nil {
		t.Fatal(err)
	}
	if !list.ChainOK || list.Checked != 3 || len(list.Closings) != 3 {
		t.Fatalf("list %+v", list)
	}
	byKey := map[string]*store.Closing{}
	for _, cl := range list.Closings {
		byKey[cl.Kind+" "+cl.PeriodKey] = cl
	}
	day, week, month := byKey["day 2026-09-30"], byKey["week 2026-W39"], byKey["month 2026-09"]
	if day == nil || week == nil || month == nil {
		t.Fatalf("periods %v", byKey)
	}
	s := day.Summary
	if s.Documents != 1 || s.Reported != 1 || s.Sales.Count != 1 || !s.Sales.SalesTax.Equal(inv.Totals.SalesTax) ||
		s.FirstFBRNo != inv.FBRInvoiceNumber || s.LastNo != inv.InternalNo {
		t.Fatalf("day summary %+v", s)
	}
	if week.Summary.Documents != 0 || month.Summary.Reported != 1 || !month.Summary.Sales.ValueExclST.Equal(inv.Totals.ValueExclST) {
		t.Fatalf("week %+v month %+v", week.Summary, month.Summary)
	}

	// After five days off, the missed days and last week are closed.
	f.setClock(pkTime(t, "2026-10-06 09:00")) // Tuesday
	if n, err := f.svc.CloseDuePeriods(f.ctx); err != nil || n != 6 {
		t.Fatalf("catch-up %d %v", n, err)
	}
	days, _ := f.svc.Closings(f.ctx, c, c.Environment, ClosingDay, 50)
	if len(days.Closings) != 6 || days.Closings[0].PeriodKey != "2026-10-05" || !days.ChainOK || days.Checked != 9 {
		t.Fatalf("days %+v", days)
	}
	weeks, _ := f.svc.Closings(f.ctx, c, c.Environment, ClosingWeek, 50)
	if len(weeks.Closings) != 2 || weeks.Closings[0].PeriodKey != "2026-W40" || weeks.Closings[0].Summary.Reported != 1 {
		t.Fatalf("weeks %+v", weeks.Closings)
	}

	// Closings cannot be changed or deleted; tampering breaks the chain.
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `UPDATE closings SET summary_json='{}' WHERE id=?`, day.ID); err == nil {
		t.Fatal("closing was changed")
	}
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `DELETE FROM closings WHERE id=?`, day.ID); err == nil {
		t.Fatal("closing was deleted")
	}
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `DROP TRIGGER trg_closings_noupdate`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `UPDATE closings SET summary_json='{"documents":0}' WHERE id=?`, day.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := f.svc.Closings(f.ctx, c, c.Environment, "", 50); list.ChainOK || list.BrokenAt != day.ID {
		t.Fatalf("tampering not detected: %+v", list)
	}
	rep, err := f.svc.CheckIntegrity(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rep.OK() || !strings.Contains(strings.Join(rep.Problems, ";"), "closing") {
		t.Fatalf("integrity check missed the altered closing: %+v", rep)
	}
}

func TestStockTransferNotes(t *testing.T) {
	f := setup(t)
	prod := f.product(t)
	in := TransferInput{ToName: "Own warehouse, Port Qasim", ToAddress: "Plot 7, Port Qasim, Karachi", VehicleNo: "jx-1234",
		DriverCNIC: "4210112345671", AuthorisedBy: "Store manager",
		Items: []TransferItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("500"), ValueAtCost: tax.MustD("100000")},
			{Description: "Empty bags", Quantity: tax.MustD("1000"), UoM: "Numbers, pieces, units", ValueAtCost: tax.MustD("2500.5")}}}
	if _, err := f.svc.CreateStockTransfer(f.ctx, f.admin, f.cid, in); err == nil || !strings.Contains(err.Error(), "sales tax invoice") {
		t.Fatalf("transfer to a warehouse with another STRN accepted: %v", err)
	}
	in.SameSTRN = true
	first, err := f.svc.CreateStockTransfer(f.ctx, f.admin, f.cid, in)
	if err != nil {
		t.Fatal(err)
	}
	if first.Number != "STN-000001" || first.Status != "DISPATCHED" || first.DriverCNIC != "42101-1234567-1" || first.VehicleNo != "JX-1234" ||
		first.FromName != "Company 8" || !first.TotalValue.Equal(tax.MustD("102500.5")) || len(first.Items) != 2 {
		t.Fatalf("note %+v", first)
	}
	if it := first.Items[0]; it.Description != "Fertilizer bag" || it.HSCode != "3104.2000" || it.UoM != "KG" {
		t.Fatalf("line from product %+v", it)
	}
	second, err := f.svc.CreateStockTransfer(f.ctx, f.admin, f.cid, in)
	if err != nil || second.Number != "STN-000002" {
		t.Fatalf("second note %v %v", second, err)
	}
	bad := in
	bad.DriverCNIC = "12345"
	if _, err := f.svc.CreateStockTransfer(f.ctx, f.admin, f.cid, bad); err == nil {
		t.Fatal("short CNIC accepted")
	}
	bad = in
	bad.Items = []TransferItemInput{{Description: "Bags", Quantity: tax.MustD("0")}}
	if _, err := f.svc.CreateStockTransfer(f.ctx, f.admin, f.cid, bad); err == nil {
		t.Fatal("zero quantity accepted")
	}

	got, err := f.svc.ReceiveStockTransfer(f.ctx, f.admin, f.cid, first.ID, "Warehouse in-charge", "2026-10-09T16:30")
	if err != nil || got.Status != "RECEIVED" || got.ReceivedBy != "Warehouse in-charge" {
		t.Fatalf("receive %+v %v", got, err)
	}
	if _, err := f.svc.ReceiveStockTransfer(f.ctx, f.admin, f.cid, first.ID, "Again", ""); err == nil {
		t.Fatal("received twice")
	}
	if _, err := f.svc.CancelStockTransfer(f.ctx, f.admin, f.cid, first.ID, "Error"); err == nil {
		t.Fatal("received note cancelled")
	}
	if got, err := f.svc.CancelStockTransfer(f.ctx, f.admin, f.cid, second.ID, "Duplicate note"); err != nil || got.Status != "CANCELLED" {
		t.Fatalf("cancel %+v %v", got, err)
	}
	list, total, err := f.svc.Store.ListStockTransfers(f.ctx, f.cid, store.TransferFilter{Status: "RECEIVED"})
	if err != nil || total != 1 || list[0].Number != "STN-000001" || list[0].ItemCount != 2 {
		t.Fatalf("register %+v %d %v", list, total, err)
	}
	if _, total, _ := f.svc.Store.ListStockTransfers(f.ctx, f.cid, store.TransferFilter{Q: "bags"}); total != 2 {
		t.Fatalf("search found %d", total)
	}
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `DELETE FROM stock_transfers WHERE id=?`, second.ID); err == nil {
		t.Fatal("note deleted from the register")
	}

	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	cancelled, err := f.svc.Store.GetStockTransfer(f.ctx, f.cid, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := printing.RenderStockTransfer(&buf, c, cancelled, printing.Options{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"STN-000002", "WAREHOUSE COPY", "NOT A TAXABLE SUPPLY", "CANCELLED", "42101-1234567-1", "102,500.50"} {
		if !strings.Contains(out, want) {
			t.Errorf("printed note lacks %q", want)
		}
	}
}

func TestReturnFilingExtension(t *testing.T) {
	f := setup(t)
	f.setClock(pkTime(t, "2026-10-09 10:00"))
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	before := f.svc.CompanyDeadlines(f.ctx, c)
	if len(before) != 2 || before[0].Period != "2026-09" || before[0].Kind != "payment" || before[0].Due != "2026-10-15" || before[1].Due != "2026-10-18" {
		t.Fatalf("deadlines %+v", before)
	}
	for _, bad := range []store.ReturnExtension{
		{Period: "2026-9", FilingDate: "2026-10-25", Reference: "x"},
		{Period: "2026-09", FilingDate: "2026-09-20", Reference: "x"},
		{Period: "2026-09", FilingDate: "2026-10-25"},
	} {
		if _, err := f.svc.SetReturnExtension(f.ctx, f.admin, f.cid, bad); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
	ext, err := f.svc.SetReturnExtension(f.ctx, f.admin, f.cid, store.ReturnExtension{Period: "2026-09", FilingDate: "2026-10-27",
		Reference: "FBR Notification C.No.3(4)ST-L&P/2026, 16 October 2026"})
	if err != nil || ext.FilingDate != "2026-10-27" {
		t.Fatalf("extension %+v %v", ext, err)
	}
	after := f.svc.CompanyDeadlines(f.ctx, c)
	var pay, file ReturnDeadline
	for _, d := range after {
		if d.Kind == "payment" {
			pay = d
		} else {
			file = d
		}
	}
	if pay.Due != "2026-10-15" || pay.OriginalDue != "" {
		t.Fatalf("payment date moved: %+v", pay)
	}
	if file.Due != "2026-10-27" || file.OriginalDue != "2026-10-18" || file.DaysLeft != 18 || !strings.Contains(file.Reference, "C.No.3(4)") {
		t.Fatalf("filing %+v", file)
	}
	// After the original filing date the extended return is still the one shown.
	f.setClock(pkTime(t, "2026-10-20 10:00"))
	if dls := f.svc.CompanyDeadlines(f.ctx, c); dls[0].Period != "2026-09" {
		t.Fatalf("after original date %+v", dls)
	}
	if err := f.svc.DeleteReturnExtension(f.ctx, f.admin, f.cid, "2026-09"); err != nil {
		t.Fatal(err)
	}
	if dls := f.svc.CompanyDeadlines(f.ctx, c); dls[0].Period != "2026-10" {
		t.Fatalf("extension not removed %+v", dls)
	}
}

func TestAnnexCReconciliation(t *testing.T) {
	f := setup(t)
	cust, prod := f.customer(t), f.product(t)
	a := f.accepted(t, cust, prod)
	b := f.accepted(t, cust, prod)
	if _, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{CustomerID: &cust.ID,
		Items: []ItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("1")}}}); err != nil {
		t.Fatal(err)
	}
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	rows, err := f.svc.Store.AnnexCReconciliation(f.ctx, store.ReportFilter{CompanyID: f.cid, Environment: c.Environment, From: a.InvoiceDate, To: b.InvoiceDate})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].FBRInvoiceNumber != a.FBRInvoiceNumber || rows[1].FBRInvoiceNumber != b.FBRInvoiceNumber ||
		rows[0].BuyerNTNCNIC != "2046004" || !rows[0].SalesTax.Equal(a.Totals.SalesTax) || rows[0].Status != "ACCEPTED" {
		t.Fatalf("rows %+v", rows)
	}
	tbl := AnnexCTable(rows)
	if len(tbl.Rows) != 2 || len(tbl.Headers) != len(tbl.Rows[0]) {
		t.Fatalf("table %+v", tbl)
	}
}

func TestSigningKeyFaults(t *testing.T) {
	f := setup(t)
	inv := f.accepted(t, f.customer(t), f.product(t))
	goodPub, _ := f.svc.Store.GetSetting(f.ctx, settingSigningPub)

	// A public key swapped in the database is caught.
	other, _, _ := ed25519.GenerateKey(nil)
	if err := f.svc.Store.SetSetting(f.ctx, settingSigningPub, base64.StdEncoding.EncodeToString(other)); err != nil {
		t.Fatal(err)
	}
	rep, err := f.svc.CheckIntegrity(f.ctx)
	if err != nil || rep.OK() || !strings.Contains(strings.Join(rep.Problems, ";"), "does not match this installation's private key") {
		t.Fatalf("swapped public key not reported: %+v %v", rep, err)
	}

	// If the vault cannot open the private key (master.key lost), earlier
	// signatures are still checked with the stored public key and the fault
	// is reported.
	_ = f.svc.Store.SetSetting(f.ctx, settingSigningPub, goodPub)
	_ = f.svc.Store.SetSetting(f.ctx, settingSigningKey, "damaged")
	f.svc.signKey = nil
	if !f.svc.SignatureValid(f.ctx, inv) {
		t.Fatal("signature not verifiable with the stored public key")
	}
	rep, err = f.svc.CheckIntegrity(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(rep.Problems, ";")
	if !strings.Contains(joined, "cannot be opened") || strings.Contains(joined, "digital signature does not match") {
		t.Fatalf("problems %v", rep.Problems)
	}
}

func TestFEDParticularsAndAdvanceReceipts(t *testing.T) {
	f := setup(t)
	cust := f.customer(t)
	prod, err := f.svc.SaveProduct(f.ctx, f.admin, &store.Product{CompanyID: f.cid, Code: "CEM", Description: "Cement bag 50 kg", HSCode: "2523.2900",
		UoM: "KG", SaleType: domain.STStandard, Rate: "18%", UnitPrice: tax.MustD("20"), FEDRate: tax.MustD("10"),
		FEDType: "Ad valorem", FEDSRO: "First Schedule, Federal Excise Act 2005", FEDSROSerial: "Table I, S.No. 9"})
	if err != nil {
		t.Fatal(err)
	}
	inv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{CustomerID: &cust.ID, AdvanceRef: "  FBR 0786909DI0000000000001  ",
		Items: []ItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("1000"), FEDUnitPrice: tax.MustD("20")}}, Submit: true})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Status != domain.StatusAccepted {
		t.Fatalf("status %s: %s %+v", inv.Status, inv.LastError, inv.FBRErrors)
	}
	it := inv.Items[0]
	if it.FEDType != "Ad valorem" || it.FEDSROSerial != "Table I, S.No. 9" || !it.FED.Equal(tax.MustD("2000")) || inv.AdvanceRef != "FBR 0786909DI0000000000001" {
		t.Fatalf("line %+v ref %q", it, inv.AdvanceRef)
	}
	for _, is := range inv.Validation {
		if is.Field == "fedType" {
			t.Fatalf("complete FED particulars flagged: %+v", is)
		}
	}
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	var buf bytes.Buffer
	if err := printing.Render(&buf, c, inv, printing.Options{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"FED: Ad valorem · rate 10% · price per unit 20.00 · amount 2,000.00 payable otherwise than in sales tax mode",
		"First Schedule, Federal Excise Act 2005 S.No. Table I, S.No. 9", "Advance adjusted", "Federal excise duty (included in the value above)"} {
		if !strings.Contains(out, want) {
			t.Errorf("printed invoice lacks %q", want)
		}
	}
	if _, err := f.svc.Store.DB.ExecContext(f.ctx, `UPDATE invoices SET advance_ref='' WHERE id=?`, inv.ID); err == nil {
		t.Fatal("advance reference of an accepted invoice changed")
	}

	// Missing FED particulars are flagged.
	draft, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{CustomerID: &cust.ID,
		Items: []ItemInput{{Description: "Cement", HSCode: "2523.2900", UoM: "KG", Quantity: tax.MustD("10"), UnitPrice: tax.MustD("20"),
			SaleType: domain.STStandard, Rate: "18%", FEDRate: tax.MustD("10")}}})
	if err != nil {
		t.Fatal(err)
	}
	flagged := false
	for _, is := range draft.Validation {
		flagged = flagged || (is.Field == "fedType" && strings.Contains(is.Message, "150R(13)"))
	}
	if !flagged {
		t.Fatalf("missing FED particulars not flagged: %+v", draft.Validation)
	}

	// Advance receipt invoice.
	adv, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{CustomerID: &cust.ID, AdvanceReceipt: true, AdvanceRef: "ignored",
		Items: []ItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("100")}}, Submit: true})
	if err != nil || adv.Status != domain.StatusAccepted || !adv.AdvanceReceipt || adv.AdvanceRef != "" {
		t.Fatalf("advance %+v %v", adv, err)
	}
	buf.Reset()
	if err := printing.Render(&buf, c, adv, printing.Options{}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "ADVANCE RECEIPT INVOICE") || !strings.Contains(buf.String(), "section 23(1)") {
		t.Fatal("advance receipt invoice not titled as such")
	}
	if _, err := f.svc.CreateInvoice(f.ctx, f.admin, f.cid, &InvoiceInput{DocType: string(domain.DocDebitNote), RefInvoiceID: &adv.ID, AdvanceReceipt: true,
		CustomerID: &cust.ID, Items: []ItemInput{{ProductID: &prod.ID, Quantity: tax.MustD("1")}}}); err == nil {
		t.Fatal("debit note accepted as an advance receipt invoice")
	}
}

func TestImportAdvanceReceiptAndFEDColumns(t *testing.T) {
	f := setup(t)
	csvText := strings.Join(ImportColumns, ",") + "\n" +
		// An advance receipt invoice for a cement buyer, with FED particulars.
		"ADV-1,2026-10-09,Sale Invoice,,,2046004,FERTILIZER MANUFAC IRS NEW,Punjab,Lahore,Registered,," +
		",2523.2900,Cement,KG,1000,20,0,,Goods at standard rate (default),18%,,,," +
		",,,,," +
		"yes,,Specific (per unit),Rs 2 per kg,,First Schedule FED Act 2005,Table I S.No. 9\n" +
		"BAD-1,2026-10-09,Sale Invoice,,,2046004,X,Punjab,Lahore,Registered,," +
		",2523.2900,Cement,KG,1,20,0,,Goods at standard rate (default),18%,,,," +
		",,,,," +
		"maybe,,,,,,\n"
	rows, err := ReadRows("invoices.csv", strings.NewReader(csvText))
	if err != nil {
		t.Fatal(err)
	}
	f.setClock(pkTime(t, "2026-10-09 12:00"))
	sum, err := f.svc.Import(f.ctx, f.admin, f.cid, rows, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Invoices != 2 || sum.OK != 1 || sum.Failed != 1 || !strings.Contains(strings.Join(sum.Results[1].Errors, ";"), "advance_receipt") {
		t.Fatalf("summary %+v", sum)
	}
	inv, err := f.svc.Store.GetInvoice(f.ctx, f.cid, sum.Results[0].InvoiceID)
	if err != nil {
		t.Fatal(err)
	}
	it := inv.Items[0]
	if !inv.AdvanceReceipt || it.FEDType != "Specific (per unit)" || it.FEDRateText != "Rs 2 per kg" || it.FEDSROSerial != "Table I S.No. 9" {
		t.Fatalf("imported %+v %+v", inv, it)
	}
}
