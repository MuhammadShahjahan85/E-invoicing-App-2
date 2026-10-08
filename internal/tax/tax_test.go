// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package tax

import (
	"testing"

	"einvoicing/internal/domain"

	"github.com/shopspring/decimal"
)

func eq(t *testing.T, name string, got decimal.Decimal, want string) {
	t.Helper()
	if !got.Equal(MustD(want)) {
		t.Errorf("%s = %s, want %s", name, got.StringFixed(2), want)
	}
}

func TestParseRate(t *testing.T) {
	cases := []struct {
		in      string
		pct     string
		perUnit string
		exempt  bool
		valid   bool
	}{
		{"18%", "18", "0", false, true},
		{"1.43%", "1.43", "0", false, true},
		{"0%", "0", "0", false, true},
		{"Exempt", "0", "0", true, true},
		{"Rs.3", "0", "3", false, true},
		{"Rs.200", "0", "200", false, true},
		{"18% along with rupees 60 per kilogram", "18", "60", false, true},
		{"Rs. 1,500/MT", "0", "1500", false, true},
		{"17", "17", "0", false, true},
		{"", "0", "0", false, false},
		{"n/a", "0", "0", false, false},
	}
	for _, c := range cases {
		r := ParseRate(c.in)
		if r.Valid != c.valid || r.Exempt != c.exempt || !r.Percent.Equal(MustD(c.pct)) || !r.PerUnit.Equal(MustD(c.perUnit)) {
			t.Errorf("ParseRate(%q) = %+v", c.in, r)
		}
	}
	if !ParseRate("18%").IsStandardRate() || ParseRate("17%").IsStandardRate() {
		t.Error("IsStandardRate")
	}
}

// The FBR scenario samples double as arithmetic test vectors.
func TestScenarioArithmetic(t *testing.T) {
	cases := []struct {
		id      string
		wantTax string
	}{
		{"SN001", "180"}, {"SN002", "180"}, {"SN003", "36900"}, {"SN004", "31500"},
		{"SN005", "10"}, {"SN006", "0"}, {"SN007", "0"}, {"SN008", "180"},
		{"SN010", "17"}, {"SN012", "1.43"}, {"SN013", "50"}, {"SN014", "180"},
		{"SN015", "222.12"}, {"SN016", "5"}, {"SN017", "8"}, {"SN018", "80"},
		{"SN019", "5"}, {"SN020", "10"}, {"SN021", "36"}, {"SN022", "78"},
		{"SN023", "24600"}, {"SN024", "250"}, {"SN025", "0"}, {"SN026", "180"}, {"SN027", "18"},
	}
	for _, c := range cases {
		sc, ok := domain.LookupScenario(c.id)
		if !ok {
			t.Fatalf("missing scenario %s", c.id)
		}
		it := sc.Item
		in := LineInput{
			Quantity:        F(it.Quantity),
			Value:           Ptr(F(it.ValueSalesExcludingST)),
			SaleType:        it.SaleType,
			Rate:            it.Rate,
			BuyerRegistered: sc.BuyerRegistrationType == domain.Registered,
		}
		if it.FixedNotifiedValueOrRetailPrice > 0 {
			in.RetailValue = Ptr(F(it.FixedNotifiedValueOrRetailPrice))
		}
		res := ComputeLine(in)
		eq(t, c.id+" sales tax", res.SalesTax, c.wantTax)
	}
}

func TestStandardRateUnregisteredFurtherTax(t *testing.T) {
	res := ComputeLine(LineInput{
		Quantity: MustD("10"), UnitPrice: MustD("150"), DiscountAmount: MustD("100"),
		SaleType: domain.STStandard, Rate: "18%", BuyerRegistered: false,
	})
	eq(t, "gross", res.Gross, "1500")
	eq(t, "value", res.ValueExclST, "1400")
	eq(t, "sales tax", res.SalesTax, "252")
	eq(t, "further tax", res.FurtherTax, "56")
	eq(t, "total", res.TotalValue, "1708")

	reg := ComputeLine(LineInput{Quantity: MustD("10"), UnitPrice: MustD("150"), SaleType: domain.STStandard, Rate: "18%", BuyerRegistered: true})
	eq(t, "registered further tax", reg.FurtherTax, "0")
}

func TestFurtherTaxNotOnExemptOrThirdSchedule(t *testing.T) {
	ex := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("1000"), SaleType: domain.STExempt, Rate: "Exempt"})
	eq(t, "exempt tax", ex.SalesTax, "0")
	eq(t, "exempt further", ex.FurtherTax, "0")
	// Printed price 120 includes sales tax: retail value 240 x 100/118.
	third := ComputeLine(LineInput{Quantity: MustD("2"), UnitPrice: MustD("80"), RetailPrice: MustD("120"), SaleType: domain.STThirdSchedule, Rate: "18%"})
	eq(t, "retail value", third.RetailValue, "203.39")
	eq(t, "3rd schedule tax", third.SalesTax, "36.61")
	eq(t, "3rd schedule further", third.FurtherTax, "0")
	eq(t, "3rd schedule value", third.ValueExclST, "160")
}

func TestReducedRateExtraTaxEmpty(t *testing.T) {
	res := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("1000"), SaleType: domain.STReduced, Rate: "1%", ExtraTaxAmount: Ptr(MustD("5")), BuyerRegistered: true})
	if !res.ExtraTaxEmpty || !res.ExtraTax.IsZero() {
		t.Fatalf("extra tax should be empty for reduced rate: %+v", res)
	}
	if len(res.Warnings) == 0 {
		t.Error("expected a warning about removed extra tax")
	}
}

// Integrators report FBR refusing even a numeric zero extraTax on exempt,
// zero-rated and cotton ginner lines, so those are sent empty as well.
func TestExtraTaxEmptyForExemptZeroRatedAndCottonGinners(t *testing.T) {
	for _, c := range []struct{ saleType, rate string }{
		{domain.STExempt, "Exempt"}, {domain.STZeroRated, "0%"}, {domain.STCottonGinners, "18%"},
	} {
		res := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("1000"), SaleType: c.saleType, Rate: c.rate, BuyerRegistered: true})
		if !res.ExtraTaxEmpty || !res.ExtraTax.IsZero() {
			t.Errorf("%s: extra tax should be sent empty: %+v", c.saleType, res)
		}
	}
	std := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("1000"), SaleType: domain.STStandard, Rate: "18%", BuyerRegistered: true})
	if std.ExtraTaxEmpty {
		t.Error("standard-rate lines keep a numeric extra tax")
	}
}

func TestWithholdingAndRounding(t *testing.T) {
	res := ComputeLine(LineInput{Quantity: MustD("3"), UnitPrice: MustD("333.33"), SaleType: domain.STStandard, Rate: "18%", BuyerRegistered: true,
		Withholding: WithholdFraction})
	eq(t, "value", res.ValueExclST, "999.99")
	eq(t, "tax", res.SalesTax, "180.00") // 179.9982 -> 180.00
	eq(t, "withheld", res.STWithheld, "36")
	full := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("100"), SaleType: domain.STStandard, Rate: "18%", BuyerRegistered: true, Withholding: WithholdFull})
	eq(t, "full withheld", full.STWithheld, "18")
}

func TestExternalSalesTaxMismatchWarns(t *testing.T) {
	res := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("1000"), SaleType: domain.STStandard, Rate: "18%", BuyerRegistered: true, SalesTaxAmount: Ptr(MustD("179"))})
	eq(t, "kept external", res.SalesTax, "179")
	eq(t, "computed", res.ComputedSalesTax, "180")
	if len(res.Warnings) == 0 {
		t.Error("expected mismatch warning")
	}
}

func TestSumLines(t *testing.T) {
	a := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("1000"), SaleType: domain.STStandard, Rate: "18%"})
	b := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("500"), SaleType: domain.STStandard, Rate: "18%"})
	c := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("200"), SaleType: domain.STExempt, Rate: "Exempt"})
	tot := SumLines([]LineResult{a, b, c}, []string{domain.STStandard, domain.STStandard, domain.STExempt})
	eq(t, "value", tot.ValueExclST, "1700")
	eq(t, "tax", tot.SalesTax, "270")
	eq(t, "further", tot.FurtherTax, "60")
	eq(t, "total", tot.TotalValue, "2030")
	if len(tot.ByRate) != 2 {
		t.Fatalf("byRate groups = %d", len(tot.ByRate))
	}
}

func TestAmountInWords(t *testing.T) {
	cases := []struct {
		amt   string
		style WordsStyle
		want  string
	}{
		{"123456.78", WordsSouthAsian, "Rupees One Lakh Twenty-Three Thousand Four Hundred Fifty-Six and Paisa Seventy-Eight Only"},
		{"12345678", WordsSouthAsian, "Rupees One Crore Twenty-Three Lakh Forty-Five Thousand Six Hundred Seventy-Eight Only"},
		{"1180", WordsSouthAsian, "Rupees One Thousand One Hundred Eighty Only"},
		{"0", WordsSouthAsian, "Rupees Zero Only"},
		{"1234567.5", WordsInternational, "Rupees One Million Two Hundred Thirty-Four Thousand Five Hundred Sixty-Seven and Paisa Fifty Only"},
	}
	for _, c := range cases {
		if got := AmountInWords(MustD(c.amt), c.style); got != c.want {
			t.Errorf("AmountInWords(%s) = %q, want %q", c.amt, got, c.want)
		}
	}
}

func TestFormatAmount(t *testing.T) {
	if got := FormatAmount(MustD("1234567.5")); got != "1,234,567.50" {
		t.Errorf("got %s", got)
	}
	if got := FormatAmount(MustD("-999")); got != "-999.00" {
		t.Errorf("got %s", got)
	}
}

// Third Schedule: the price printed on the pack includes sales tax (section
// 3(2)(a)); the tax is charged on the retail price excluding it (section
// 2(27)), i.e. printed price x rate / (100 + rate).
func TestThirdScheduleTaxInclusivePrintedPrice(t *testing.T) {
	res := ComputeLine(LineInput{Quantity: MustD("10"), UnitPrice: MustD("80"), RetailPrice: MustD("118"), SaleType: domain.STThirdSchedule, Rate: "18%"})
	eq(t, "printed retail value", res.PrintedRetailValue, "1180")
	eq(t, "retail value (FBR fixedNotifiedValueOrRetailPrice)", res.RetailValue, "1000")
	eq(t, "sales tax", res.SalesTax, "180")
	eq(t, "value", res.ValueExclST, "800")
	eq(t, "total", res.TotalValue, "980")
	// FBR's own retail value (ex sales tax) is used as given.
	given := ComputeLine(LineInput{Quantity: MustD("10"), UnitPrice: MustD("80"), RetailValue: Ptr(MustD("1000")), SaleType: domain.STThirdSchedule, Rate: "18%"})
	eq(t, "given retail value", given.RetailValue, "1000")
	eq(t, "given sales tax", given.SalesTax, "180")
}

// FED charged separately is part of the value of supply (section 2(46)(a)),
// so sales tax and further tax are charged on value + FED and FED is not
// added to the total a second time.
func TestFEDIncludedInValueOfSupply(t *testing.T) {
	res := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("1000"), FEDRate: MustD("20"), SaleType: domain.STStandard, Rate: "18%"})
	eq(t, "fed", res.FED, "200")
	eq(t, "value of supply", res.ValueExclST, "1200")
	eq(t, "sales tax", res.SalesTax, "216")
	eq(t, "further tax", res.FurtherTax, "48")
	eq(t, "total", res.TotalValue, "1464")

	// Specific FED (e.g. Rs per kg) given as an amount.
	kg := ComputeLine(LineInput{Quantity: MustD("100"), UnitPrice: MustD("10"), FEDAmount: Ptr(MustD("150")), SaleType: domain.STStandard, Rate: "18%", BuyerRegistered: true})
	eq(t, "kg value of supply", kg.ValueExclST, "1150")
	eq(t, "kg sales tax", kg.SalesTax, "207")
	eq(t, "kg total", kg.TotalValue, "1357")

	// FED in sales tax mode is itself the tax: it stays outside the value.
	st := ComputeLine(LineInput{Quantity: MustD("1"), UnitPrice: MustD("1000"), FEDRate: MustD("10"), SaleType: domain.STGoodsFED, Rate: "8%", BuyerRegistered: true})
	eq(t, "ST-mode value", st.ValueExclST, "1000")
	eq(t, "ST-mode tax", st.SalesTax, "80")
	eq(t, "ST-mode total", st.TotalValue, "1180")
}
