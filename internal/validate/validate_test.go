// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package validate

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/tax"
)

func base() *fbr.InvoicePayload {
	return &fbr.InvoicePayload{
		InvoiceType: "Sale Invoice", InvoiceDate: "2026-10-06",
		SellerNTNCNIC: "0786909", SellerBusinessName: "Seller", SellerProvince: "SINDH", SellerAddress: "Karachi",
		BuyerNTNCNIC: "2046004", BuyerBusinessName: "Buyer", BuyerProvince: "PUNJAB", BuyerAddress: "Lahore",
		BuyerRegistrationType: "Registered",
		Items: []fbr.ItemPayload{{
			HSCode: "0101.2100", ProductDescription: "Item", Rate: "18%", UoM: "Numbers, pieces, units",
			Quantity: fbr.Q(tax.MustD("1")), ValueSalesExcludingST: fbr.A(tax.MustD("1000")),
			SalesTaxApplicable: fbr.A(tax.MustD("180")), TotalValues: fbr.A(tax.MustD("1180")), SaleType: domain.STStandard,
		}},
	}
}

var today = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

func codes(r Result, sev Severity) string {
	var c []string
	for _, i := range r.Issues {
		if i.Severity == sev {
			c = append(c, i.Field+":"+i.Code)
		}
	}
	return strings.Join(c, ",")
}

func TestValidPayload(t *testing.T) {
	r := Payload(base(), Context{Env: domain.EnvProduction, Today: today})
	if r.HasErrors() {
		t.Fatalf("unexpected errors: %v", r.Errors())
	}
	if len(r.Warnings()) != 0 {
		t.Fatalf("unexpected warnings: %v", r.Warnings())
	}
}

func TestHeaderErrors(t *testing.T) {
	p := base()
	p.InvoiceType = "Credit Note"
	p.InvoiceDate = "06/10/2026"
	p.BuyerNTNCNIC = "12345"
	p.BuyerRegistrationType = "registered"
	p.BuyerProvince = ""
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today})
	got := codes(r, SevError)
	for _, want := range []string{"invoiceType:0003", "invoiceDate:0005", "buyerNTNCNIC:0002", "buyerRegistrationType:0053", "buyerProvince:0074"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
}

func TestFutureDateAndSandboxScenario(t *testing.T) {
	p := base()
	p.InvoiceDate = "2026-10-08"
	r := Payload(p, Context{Env: domain.EnvSandbox, Today: today})
	got := codes(r, SevError)
	if !strings.Contains(got, "invoiceDate:0043") || !strings.Contains(got, "scenarioId:") {
		t.Errorf("got %s", got)
	}
}

func TestRegisteredBuyerNeedsNTN(t *testing.T) {
	p := base()
	p.BuyerNTNCNIC = ""
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevError), "buyerNTNCNIC:0009") {
		t.Error("expected 0009")
	}
	p.BuyerRegistrationType = "Unregistered"
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if r.HasErrors() {
		t.Errorf("unregistered walk-in buyer without CNIC should be allowed: %v", r.Errors())
	}
	if strings.Contains(codes(r, SevWarning), "buyerNTNCNIC:") {
		t.Error("small supply by a non-manufacturer needs no CNIC warning")
	}
	// Section 23(1)(b): a manufacturer or importer must record the
	// unregistered buyer's CNIC/NTN whatever the value.
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today, SellerActivities: []string{"Importer"}})
	if r.HasErrors() || !strings.Contains(codes(r, SevWarning), "buyerNTNCNIC:") {
		t.Errorf("expected a section 23(1)(b) warning, got %s", codes(r, SevWarning))
	}
	for _, dummy := range []string{"0000000000000", "1000000000000", "1111111"} {
		p.BuyerNTNCNIC = dummy
		r = Payload(p, Context{Env: domain.EnvProduction, Today: today, SellerActivities: []string{"Manufacturer"}})
		if !strings.Contains(codes(r, SevWarning), "buyerNTNCNIC:") {
			t.Errorf("placeholder %s must not satisfy section 23(1)(b)", dummy)
		}
	}
	p.BuyerNTNCNIC = "4210112345671"
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today, SellerActivities: []string{"Manufacturer"}})
	if strings.Contains(codes(r, SevWarning), "buyerNTNCNIC:") {
		t.Error("no warning once the CNIC is recorded")
	}
}

func TestReducedRateRules(t *testing.T) {
	p := base()
	it := &p.Items[0]
	it.SaleType = domain.STReduced
	it.Rate = "1%"
	it.SalesTaxApplicable = fbr.A(tax.MustD("10"))
	it.ExtraTax = fbr.A(tax.MustD("0"))
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today})
	got := codes(r, SevError)
	for _, want := range []string{"sroScheduleNo:0077", "sroItemSerialNo:0078", "extraTax:0091"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
	it.SROScheduleNo, it.SROItemSerialNo, it.ExtraTax = "EIGHTH SCHEDULE Table 1", "82", fbr.EmptyAmount()
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if r.HasErrors() {
		t.Errorf("unexpected %v", r.Errors())
	}
}

func TestExemptRate(t *testing.T) {
	p := base()
	it := &p.Items[0]
	it.SaleType = domain.STExempt
	it.Rate = "0%"
	it.SalesTaxApplicable = fbr.A(tax.Zero)
	it.SROScheduleNo, it.SROItemSerialNo = "6th Schd Table I", "100"
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevError), "rate:0046") {
		t.Errorf("expected rate error, got %s", codes(r, SevError))
	}
}

func TestTaxMismatchWarns(t *testing.T) {
	p := base()
	p.Items[0].SalesTaxApplicable = fbr.A(tax.MustD("170"))
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevWarning), "salesTaxApplicable:0104") {
		t.Errorf("expected mismatch warning, got %s", codes(r, SevWarning))
	}
}

func TestThirdScheduleNeedsRetailPrice(t *testing.T) {
	p := base()
	it := &p.Items[0]
	it.SaleType = domain.STThirdSchedule
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevError), "fixedNotifiedValueOrRetailPrice:0090") {
		t.Errorf("got %s", codes(r, SevError))
	}
}

func TestDebitNote(t *testing.T) {
	p := base()
	p.InvoiceType = "Debit Note"
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevError), "invoiceRefNo:0026") {
		t.Errorf("got %s", codes(r, SevError))
	}
	p.InvoiceRefNo = "0786909DI1747119701593"
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today, Original: &OriginalInvoice{
		FBRNumber: p.InvoiceRefNo, Date: "2026-10-07", ValueExclST: tax.MustD("500"), SalesTax: tax.MustD("90"), BuyerNTNCNIC: "2046004"}})
	if !strings.Contains(codes(r, SevError), "invoiceDate:0035") || !strings.Contains(codes(r, SevWarning), "items:0036") {
		t.Errorf("errors %s warnings %s", codes(r, SevError), codes(r, SevWarning))
	}
}

func TestNormalizeRegNo(t *testing.T) {
	cases := map[string]string{"1234567-8": "1234567", "42101-1234567-1": "4210112345671", " 0786909 ": "0786909"}
	for in, want := range cases {
		if got, _ := NormalizeRegNo(in); got != want {
			t.Errorf("NormalizeRegNo(%q) = %q, want %q", in, got, want)
		}
	}
	if !ValidRegNoFormat("0786909") || !ValidRegNoFormat("4210112345671") || ValidRegNoFormat("12345678") || ValidRegNoFormat("12345a7") {
		t.Error("ValidRegNoFormat")
	}
}

func TestCatalogue(t *testing.T) {
	// DI API v1.12 lists 86 sales error codes (section 7) and 22 purchase
	// error codes (section 8).
	sales, purchase := 0, 0
	for _, e := range Catalogue() {
		switch e.Section {
		case "sales":
			sales++
		case "purchase":
			purchase++
		default:
			t.Errorf("%s: section %q", e.Code, e.Section)
		}
		if e.Title == "" || e.Detail == "" || e.Fix == "" {
			t.Errorf("%s: incomplete entry %+v", e.Code, e)
		}
	}
	if sales != 86 || purchase != 22 {
		t.Errorf("sales %d purchase %d", sales, purchase)
	}
	// Spot checks against the specification's wording.
	for code, want := range map[string]string{
		"0022": "Please provide ST withheld at Source or STS Withheld",
		"0026": "Invoice Reference No. is required.",
		"0034": "{0} only allowed within {1} days of invoice date of the original invoice",
		"0079": "If Value of Sales Excl. ST greater than {0}. Rate {1} not allowed.",
		"0401": "The provided seller NTN/CNIC does not have a valid or authorized access token",
		"0177": "Please provide Further Tax",
	} {
		if e, ok := Lookup(code); !ok || e.Title != want {
			t.Errorf("%s: %q", code, e.Title)
		}
	}
}

// Every code a local check reports must be one FBR documents, so the help
// text shown next to it is right.
func TestLocalChecksCiteDocumentedCodes(t *testing.T) {
	src, err := os.ReadFile("validate.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range regexp.MustCompile(`"(\d{4})", Sev`).FindAllStringSubmatch(string(src), -1) {
		if _, ok := Lookup(m[1]); !ok {
			t.Errorf("validate.go cites undocumented code %s", m[1])
		}
	}
	for _, m := range regexp.MustCompile(`neg\("\w+", [^,]+, "(\d{4})"\)`).FindAllStringSubmatch(string(src), -1) {
		if _, ok := Lookup(m[1]); !ok {
			t.Errorf("validate.go cites undocumented code %s", m[1])
		}
	}
}

func TestNoteWindowAndElectricityRate(t *testing.T) {
	p := base()
	p.InvoiceType = "Debit Note"
	p.InvoiceRefNo = "0786909DI1747119701593"
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today, Original: &OriginalInvoice{
		FBRNumber: p.InvoiceRefNo, Date: "2026-03-01", ValueExclST: tax.MustD("5000"), SalesTax: tax.MustD("900"), BuyerNTNCNIC: "2046004"}})
	if !strings.Contains(codes(r, SevError), "invoiceDate:0034") {
		t.Errorf("note older than 180 days: %s", codes(r, SevError))
	}
	p.InvoiceRefNo = "INV-12"
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevWarning), "invoiceRefNo:0057") {
		t.Errorf("reference format: %s", codes(r, SevWarning))
	}

	p = base()
	it := &p.Items[0]
	it.SaleType, it.Rate, it.HSCode, it.UoM = domain.STElectricityRetl, "5%", "2716.0000", "KWH"
	it.ValueSalesExcludingST, it.SalesTaxApplicable = fbr.A(tax.MustD("25000")), fbr.A(tax.MustD("1250"))
	it.SROScheduleNo, it.SROItemSerialNo = "1450(I)/2021", "4"
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevWarning), "rate:0079") {
		t.Errorf("5%% above Rs 20,000: %s", codes(r, SevWarning))
	}
	it.UoM = "Numbers, pieces, units"
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevWarning), "uoM:0096") {
		t.Errorf("electricity unit: %s", codes(r, SevWarning))
	}

	p = base()
	p.BuyerRegistrationType, p.BuyerNTNCNIC = "Unregistered", ""
	p.Items[0].SaleType = domain.STCottonGinners
	r = Payload(p, Context{Env: domain.EnvProduction, Today: today})
	if !strings.Contains(codes(r, SevError), "saleType:0100") {
		t.Errorf("cotton ginners to unregistered buyer: %s", codes(r, SevError))
	}
}

func TestRepeatedLinesWarned(t *testing.T) {
	p := base()
	other := p.Items[0]
	other.HSCode = "0101.2900"
	p.Items = append(p.Items, other, p.Items[0])
	p.Items[2].ProductDescription = "  item " // same product, spacing and case differ
	r := Payload(p, Context{Env: domain.EnvProduction, Today: today})
	var got []Issue
	for _, i := range r.Warnings() {
		if strings.Contains(i.Message, "repeats line") {
			got = append(got, i)
		}
	}
	if len(got) != 1 || got[0].Line != 3 || !strings.Contains(got[0].Message, "repeats line 1") {
		t.Fatalf("repeated-line warnings: %+v", got)
	}
	if r.HasErrors() {
		t.Fatalf("a repeated line is only a warning: %v", r.Errors())
	}
}
