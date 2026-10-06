package printing

import (
	"bytes"
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
	inv.Environment = domain.EnvSandbox
	if out := render(t, c, inv, Options{}); !strings.Contains(out, "FBR SANDBOX — NOT A VALID TAX INVOICE") {
		t.Error("sandbox invoice must be watermarked")
	}
	_, inv = sampleInvoice()
	if out := render(t, c, inv, Options{Format: "thermal"}); !strings.Contains(out, inv.FBRInvoiceNumber) || !strings.Contains(out, `viewBox="0 0 25 25"`) {
		t.Error("thermal receipt must carry the FBR number and QR code")
	}
}
