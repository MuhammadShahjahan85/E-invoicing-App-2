// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package printing

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"einvoicing/internal/domain"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"
)

func sampleInvoice() (*store.Company, *store.Invoice) {
	c := &store.Company{Name: "Seller (Pvt) Ltd", NTNCNIC: "1234567", Province: "SINDH", Address: "Karachi", PrintSettings: store.DefaultPrintSettings()}
	inv := &store.Invoice{
		Environment: domain.EnvProduction, DocType: domain.DocSaleInvoice, InternalNo: "INV-2627-000001", InvoiceDate: "2026-10-06",
		Status: domain.StatusAccepted, SellerName: c.Name, SellerNTNCNIC: c.NTNCNIC, SellerProvince: c.Province, SellerAddress: c.Address,
		BuyerName: "Buyer", BuyerNTNCNIC: "2046004", BuyerProvince: "PUNJAB", BuyerRegistrationType: domain.Registered,
		FBRInvoiceNumber: "1234567DI1759740000000",
		Items: []*store.InvoiceItem{{LineNo: 1, HSCode: "0101.2100", Description: "Item", UoM: "Numbers, pieces, units", Quantity: tax.MustD("10"),
			UnitPrice: tax.MustD("100"), ValueExclST: tax.MustD("1000"), SaleType: domain.STStandard, Rate: "18%", SalesTax: tax.MustD("180"),
			TotalValue: tax.MustD("1180")}},
	}
	inv.Totals.ValueExclST, inv.Totals.SalesTax, inv.Totals.TotalValue, inv.Totals.AmountPayable = tax.MustD("1000"), tax.MustD("180"), tax.MustD("1180"), tax.MustD("1180")
	return c, inv
}

func render(t *testing.T, c *store.Company, inv *store.Invoice, o Options) string {
	t.Helper()
	var b bytes.Buffer
	if err := Render(&b, c, inv, o); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestRenderA4(t *testing.T) {
	c, inv := sampleInvoice()
	out := render(t, c, inv, Options{})
	for _, want := range []string{"SALES TAX INVOICE", inv.FBRInvoiceNumber, `viewBox="0 0 25 25"`, "FBR DIGITAL", "One Thousand One Hundred Eighty"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out, "NOT A VALID TAX INVOICE") || strings.Contains(out, "DUPLICATE") {
		t.Error("unexpected watermark or duplicate label on a first production print")
	}
	if n := strings.Count(out, `class="sheet copy-sheet"`); n != 1 {
		t.Errorf("%d sheets, want 1", n)
	}

	// Reprints are labelled DUPLICATE.
	inv.PrintCount = 1
	if out := render(t, c, inv, Options{}); !strings.Contains(out, "DUPLICATE") {
		t.Error("reprint must be labelled DUPLICATE")
	}
}

func TestRenderCopies(t *testing.T) {
	c, inv := sampleInvoice()
	c.PrintSettings.Copies = 3
	out := render(t, c, inv, Options{})
	if n := strings.Count(out, `class="sheet copy-sheet"`); n != 3 {
		t.Fatalf("%d sheets, want 3", n)
	}
	for _, l := range []string{"BUYER&#39;S COPY", "SELLER&#39;S COPY", "OFFICE COPY"} {
		if !strings.Contains(out, l) {
			t.Errorf("missing copy label %q", l)
		}
	}
	// Each copy carries the FBR number and QR code.
	if n := strings.Count(out, inv.FBRInvoiceNumber); n < 3 {
		t.Errorf("FBR number printed %d times, want at least 3", n)
	}
}

func TestRenderWatermarks(t *testing.T) {
	c, inv := sampleInvoice()
	inv.Status = domain.StatusDraft
	inv.FBRInvoiceNumber = ""
	out := render(t, c, inv, Options{})
	if !strings.Contains(out, "DRAFT — NOT REPORTED TO FBR") || strings.Contains(out, `viewBox="0 0 25 25"`) {
		t.Error("draft must be watermarked and carry no QR code")
	}
	_, inv = sampleInvoice()
	inv.Status = domain.StatusQueued
	inv.FBRInvoiceNumber = ""
	if out := render(t, c, inv, Options{}); !strings.Contains(out, "PENDING FBR REPORTING") || !strings.Contains(out, "within 24 hours") || strings.Contains(out, `viewBox="0 0 25 25"`) {
		t.Error("queued invoice must print as a provisional copy without a QR code")
	}
	// Issued offline, upload rejected after recovery: still provisional.
	_, inv = sampleInvoice()
	inv.Status, inv.OfflineSince, inv.FBRInvoiceNumber = domain.StatusRejected, "2026-10-07T05:00:00Z", ""
	if out := render(t, c, inv, Options{}); !strings.Contains(out, "PENDING FBR REPORTING") || !strings.Contains(out, "REJECTED") {
		t.Error("rejected offline invoice must stay a provisional copy")
	}
	// Once accepted, the offline history no longer affects the print.
	_, inv = sampleInvoice()
	inv.OfflineSince = "2026-10-07T05:00:00Z"
	if out := render(t, c, inv, Options{}); strings.Contains(out, "PENDING FBR REPORTING") || !strings.Contains(out, `viewBox="0 0 25 25"`) {
		t.Error("accepted invoice must print normally")
	}
	_, inv = sampleInvoice()
	inv.Environment = domain.EnvSandbox
	if out := render(t, c, inv, Options{}); !strings.Contains(out, "FBR SANDBOX — NOT A VALID TAX INVOICE") {
		t.Error("sandbox invoice must be watermarked")
	}
	_, inv = sampleInvoice()
	if out := render(t, c, inv, Options{Format: "thermal"}); !strings.Contains(out, inv.FBRInvoiceNumber) || !strings.Contains(out, `viewBox="0 0 25 25"`) {
		t.Error("thermal receipt must carry the FBR number and QR code")
	}
}

func TestRenderPDF(t *testing.T) {
	c, inv := sampleInvoice()
	c.STRN, c.City, c.Phone, c.Email = "32-77-8761-234-56", "Karachi", "021-111-222-333", "accounts@seller.pk"
	c.SoftwareRegNo = "DI-SW-2026-0042"
	inv.FBRDated = "2026-10-06 11:42:10"
	inv.Notes = "Delivered at Port Qasim warehouse; payment within 30 days."
	inv.Signature = "c2lnbmF0dXJlLWJ5dGVzLWZvci10ZXN0aW5nLW9ubHktMDEyMzQ1Njc4OQ=="
	inv.SealHash = "9f2c4e1ab37d55aa0c1d2e3f4a5b6c7d"
	for i := 2; i <= 14; i++ {
		inv.Items = append(inv.Items, &store.InvoiceItem{LineNo: i, HSCode: "3102.1000", Description: "Urea fertilizer, 50 kg bags — Engro brand",
			UoM: "KG", Quantity: tax.MustD("50"), UnitPrice: tax.MustD("120"), ValueExclST: tax.MustD("6000"), SaleType: domain.STReduced,
			Rate: "5%", SalesTax: tax.MustD("300"), TotalValue: tax.MustD("6300"), SROScheduleNo: "Eighth Schedule", SROItemSerialNo: "12"})
	}
	var b bytes.Buffer
	if err := RenderPDF(&b, c, inv, Options{FBRLogo: DefaultFBRLogo, FBRLogoMime: DefaultFBRLogoMime, SigningKey: "AB12-CD34"}); err != nil {
		t.Fatal(err)
	}
	out := b.Bytes()
	if !bytes.HasPrefix(out, []byte("%PDF-")) || len(out) < 5000 {
		t.Fatalf("not a PDF (%d bytes)", len(out))
	}
	if p := os.Getenv("PDF_SAMPLE"); p != "" {
		_ = os.WriteFile(p, out, 0o644)
	}

	// Drafts carry the not-reported watermark; two copies make two pages.
	inv.Status, inv.FBRInvoiceNumber = domain.StatusDraft, ""
	c.PrintSettings.Copies = 2
	b.Reset()
	if err := RenderPDF(&b, c, inv, Options{}); err != nil {
		t.Fatal(err)
	}
	if n := bytes.Count(b.Bytes(), []byte("/Type /Page\n")) + bytes.Count(b.Bytes(), []byte("/Type /Page ")); n < 2 {
		t.Errorf("%d pages for two copies", n)
	}
}
