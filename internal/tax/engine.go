// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

package tax

import (
	"fmt"

	"einvoicing/internal/domain"

	"github.com/shopspring/decimal"
)

// DefaultFurtherTaxRate is the further tax rate under section 3(1A) of the
// Sales Tax Act, 1990 on taxable supplies to persons who are not registered
// (4% of the value of supply, in addition to the normal rate). It can be
// changed per company if the Finance Act changes it.
var DefaultFurtherTaxRate = decimal.NewFromInt(4)

// DefaultWithholdingFraction is the share of sales tax a withholding agent
// withholds under the Sales Tax Special Procedure (Withholding) Rules, 2007
// in the general case (one-fifth).
var DefaultWithholdingFraction = decimal.NewFromFloat(0.2)

// WithholdingMode controls salesTaxWithheldAtSource for a line.
type WithholdingMode string

const (
	WithholdNone     WithholdingMode = ""         // buyer is not a withholding agent
	WithholdFraction WithholdingMode = "fraction" // fraction of sales tax (default 1/5)
	WithholdFull     WithholdingMode = "full"     // the whole sales tax
	WithholdAmount   WithholdingMode = "amount"   // explicit amount
)

// LineInput is everything the engine needs for one invoice line.
// Pointer fields are optional overrides; nil means "compute".
type LineInput struct {
	Quantity  decimal.Decimal
	UnitPrice decimal.Decimal // price per unit excluding sales tax
	// Value overrides Quantity*UnitPrice-discount as the value of supply
	// excluding sales tax (ERP pass-through).
	Value           *decimal.Decimal
	DiscountAmount  decimal.Decimal
	DiscountPercent decimal.Decimal

	SaleType string
	Rate     string // FBR rate description, e.g. "18%"

	// RetailPrice is the printed retail price per unit (Third Schedule).
	RetailPrice decimal.Decimal
	// RetailValue overrides Quantity*RetailPrice (fixedNotifiedValueOrRetailPrice).
	RetailValue *decimal.Decimal

	BuyerRegistered bool

	// FurtherTaxApplies overrides the sale type default (nil = default).
	FurtherTaxApplies *bool
	// FurtherTaxRate in percent; zero means DefaultFurtherTaxRate.
	FurtherTaxRate   decimal.Decimal
	FurtherTaxAmount *decimal.Decimal

	ExtraTaxRate   decimal.Decimal // percent of value
	ExtraTaxAmount *decimal.Decimal

	FEDRate   decimal.Decimal // percent of value (FED charged separately)
	FEDAmount *decimal.Decimal

	Withholding         WithholdingMode
	WithholdingFraction decimal.Decimal // used with WithholdFraction; zero = 1/5
	WithholdingAmount   *decimal.Decimal

	// SalesTaxAmount is an externally computed sales tax (ERP pass-through).
	// The engine keeps it but reports a warning if it differs from its own
	// computation by more than one paisa.
	SalesTaxAmount *decimal.Decimal
}

// LineResult is the computed line, ready to be mapped to the DI payload.
type LineResult struct {
	Quantity    decimal.Decimal `json:"quantity"`
	Gross       decimal.Decimal `json:"gross"`
	Discount    decimal.Decimal `json:"discount"`
	ValueExclST decimal.Decimal `json:"valueExclST"`
	RetailValue decimal.Decimal `json:"retailValue"`
	Rate        Rate            `json:"rate"`
	Basis       domain.TaxBasis `json:"basis"`
	TaxBase     decimal.Decimal `json:"taxBase"`
	SalesTax    decimal.Decimal `json:"salesTax"`
	// ComputedSalesTax is the engine's own figure (equals SalesTax unless an
	// external amount was supplied).
	ComputedSalesTax decimal.Decimal `json:"computedSalesTax"`
	FurtherTax       decimal.Decimal `json:"furtherTax"`
	ExtraTax         decimal.Decimal `json:"extraTax"`
	// ExtraTaxEmpty means the payload must carry extraTax as "" (FBR 0091).
	ExtraTaxEmpty bool            `json:"extraTaxEmpty"`
	FED           decimal.Decimal `json:"fed"`
	STWithheld    decimal.Decimal `json:"stWithheld"`
	// TotalValue = value excl. ST + sales tax + further tax + extra tax + FED.
	TotalValue decimal.Decimal `json:"totalValue"`
	Warnings   []string        `json:"warnings,omitempty"`
}

// ComputeLine applies the sales tax rules to one line.
func ComputeLine(in LineInput) LineResult {
	res := LineResult{Quantity: R4(in.Quantity)}
	st, known := domain.LookupSaleType(in.SaleType)
	if !known {
		st = domain.SaleType{Name: in.SaleType, Basis: domain.BasisValue}
		if in.SaleType != "" {
			res.warn("Sale type %q is not in the built-in catalogue; it will be sent as entered.", in.SaleType)
		}
	}
	res.Basis = st.Basis

	rateDesc := in.Rate
	if rateDesc == "" && st.DefaultRate != "" {
		rateDesc = st.DefaultRate
	}
	res.Rate = ParseRate(rateDesc)
	if st.Exempt && !res.Rate.Exempt {
		res.warn("Sale type %q requires the rate 'Exempt'; %q was given.", st.Name, rateDesc)
	}
	if !res.Rate.Valid {
		res.warn("Rate %q could not be interpreted; sales tax computed as zero.", rateDesc)
	}

	// Value of supply (section 2(46)): quantity x price less trade discount.
	res.Gross = R2(res.Quantity.Mul(in.UnitPrice))
	disc := R2(in.DiscountAmount)
	if disc.IsZero() && in.DiscountPercent.IsPositive() {
		disc = R2(PercentOf(res.Gross, in.DiscountPercent))
	}
	res.Discount = disc
	if in.Value != nil {
		res.ValueExclST = R2(*in.Value)
		if res.Gross.IsZero() {
			res.Gross = R2(res.ValueExclST.Add(disc))
		}
	} else {
		res.ValueExclST = R2(res.Gross.Sub(disc))
	}
	if res.ValueExclST.IsNegative() {
		res.warn("Discount exceeds the line value; value of supply is negative.")
	}

	// Retail price / notified value (fixedNotifiedValueOrRetailPrice).
	if in.RetailValue != nil {
		res.RetailValue = R2(*in.RetailValue)
	} else if in.RetailPrice.IsPositive() {
		res.RetailValue = R2(res.Quantity.Mul(in.RetailPrice))
	}

	// Sales tax.
	switch {
	case res.Rate.Exempt:
		res.TaxBase = res.ValueExclST
		res.ComputedSalesTax = Zero
	case st.Basis == domain.BasisRetailPrice:
		res.TaxBase = res.RetailValue
		if res.RetailValue.IsZero() {
			res.warn("Third Schedule goods are taxed on the printed retail price; enter the retail price.")
		}
		res.ComputedSalesTax = R2(PercentOf(res.RetailValue, res.Rate.Percent).Add(res.Quantity.Mul(res.Rate.PerUnit)))
	default:
		res.TaxBase = res.ValueExclST
		res.ComputedSalesTax = R2(PercentOf(res.ValueExclST, res.Rate.Percent).Add(res.Quantity.Mul(res.Rate.PerUnit)))
	}
	res.SalesTax = res.ComputedSalesTax
	if in.SalesTaxAmount != nil {
		given := R2(*in.SalesTaxAmount)
		if given.Sub(res.ComputedSalesTax).Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
			res.warn("Sales tax %s differs from the computed %s (base %s at %s). FBR rejects lines where the sales tax does not match the value and rate.",
				given.StringFixed(2), res.ComputedSalesTax.StringFixed(2), res.TaxBase.StringFixed(2), res.Rate.Raw)
		}
		res.SalesTax = given
	}

	// Further tax, section 3(1A): only on supplies to unregistered persons.
	applies := st.FurtherTaxDefault && !in.BuyerRegistered
	if in.FurtherTaxApplies != nil {
		applies = *in.FurtherTaxApplies && !in.BuyerRegistered
		if *in.FurtherTaxApplies && in.BuyerRegistered {
			res.warn("Further tax is only charged on supplies to unregistered buyers; it was not applied.")
		}
	}
	if res.Rate.IsZeroRate() {
		applies = false
	}
	switch {
	case in.FurtherTaxAmount != nil:
		res.FurtherTax = R2(*in.FurtherTaxAmount)
		if in.BuyerRegistered && res.FurtherTax.IsPositive() {
			res.warn("Further tax entered for a registered buyer; FBR may reject it.")
		}
	case applies:
		rate := in.FurtherTaxRate
		if rate.IsZero() {
			rate = DefaultFurtherTaxRate
		}
		res.FurtherTax = R2(PercentOf(res.ValueExclST, rate))
	default:
		res.FurtherTax = Zero
	}

	// Extra tax (section 3(5)); must be empty for reduced-rate goods.
	if in.ExtraTaxAmount != nil {
		res.ExtraTax = R2(*in.ExtraTaxAmount)
	} else if in.ExtraTaxRate.IsPositive() {
		res.ExtraTax = R2(PercentOf(res.ValueExclST, in.ExtraTaxRate))
	}
	if st.ExtraTaxMustBeEmpty {
		if res.ExtraTax.IsPositive() {
			res.warn("Extra tax cannot be charged on reduced-rate goods (FBR error 0091); it was removed.")
		}
		res.ExtraTax = Zero
		res.ExtraTaxEmpty = true
	}

	// Federal excise duty charged separately from sales tax.
	if in.FEDAmount != nil {
		res.FED = R2(*in.FEDAmount)
	} else if in.FEDRate.IsPositive() {
		res.FED = R2(PercentOf(res.ValueExclST, in.FEDRate))
	}

	// Sales tax withheld at source by a withholding agent buyer.
	switch in.Withholding {
	case WithholdFraction:
		f := in.WithholdingFraction
		if f.IsZero() {
			f = DefaultWithholdingFraction
		}
		res.STWithheld = R2(res.SalesTax.Mul(f))
	case WithholdFull:
		res.STWithheld = res.SalesTax
	case WithholdAmount:
		if in.WithholdingAmount != nil {
			res.STWithheld = R2(*in.WithholdingAmount)
		}
	}
	if res.STWithheld.GreaterThan(res.SalesTax) {
		res.warn("Sales tax withheld (%s) exceeds sales tax (%s).", res.STWithheld.StringFixed(2), res.SalesTax.StringFixed(2))
	}
	if res.STWithheld.IsPositive() && !res.STWithheld.Equal(res.SalesTax) {
		res.warn("FBR validation (error 0008) expects sales tax withheld at source to be either zero or equal to the sales tax; confirm the withholding treatment before submitting.")
	}

	res.TotalValue = R2(Sum(res.ValueExclST, res.SalesTax, res.FurtherTax, res.ExtraTax, res.FED))
	return res
}

func (r *LineResult) warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// Totals aggregates computed lines.
type Totals struct {
	Gross       decimal.Decimal `json:"gross"`
	Discount    decimal.Decimal `json:"discount"`
	ValueExclST decimal.Decimal `json:"valueExclST"`
	RetailValue decimal.Decimal `json:"retailValue"`
	SalesTax    decimal.Decimal `json:"salesTax"`
	FurtherTax  decimal.Decimal `json:"furtherTax"`
	ExtraTax    decimal.Decimal `json:"extraTax"`
	FED         decimal.Decimal `json:"fed"`
	STWithheld  decimal.Decimal `json:"stWithheld"`
	TotalValue  decimal.Decimal `json:"totalValue"`
	// AmountPayable is what the buyer pays the supplier: total value less
	// sales tax withheld (which the withholding agent deposits directly).
	AmountPayable decimal.Decimal `json:"amountPayable"`
	ByRate        []RateSummary   `json:"byRate"`
}

// RateSummary groups lines by sale type and rate (useful for the monthly
// sales tax return and printed invoice footers).
type RateSummary struct {
	SaleType    string          `json:"saleType"`
	Rate        string          `json:"rate"`
	ValueExclST decimal.Decimal `json:"valueExclST"`
	SalesTax    decimal.Decimal `json:"salesTax"`
	FurtherTax  decimal.Decimal `json:"furtherTax"`
}

// SumLines totals computed lines. saleTypes must be parallel to lines.
func SumLines(lines []LineResult, saleTypes []string) Totals {
	t := Totals{}
	idx := map[string]int{}
	for i, l := range lines {
		t.Gross = t.Gross.Add(l.Gross)
		t.Discount = t.Discount.Add(l.Discount)
		t.ValueExclST = t.ValueExclST.Add(l.ValueExclST)
		t.RetailValue = t.RetailValue.Add(l.RetailValue)
		t.SalesTax = t.SalesTax.Add(l.SalesTax)
		t.FurtherTax = t.FurtherTax.Add(l.FurtherTax)
		t.ExtraTax = t.ExtraTax.Add(l.ExtraTax)
		t.FED = t.FED.Add(l.FED)
		t.STWithheld = t.STWithheld.Add(l.STWithheld)
		t.TotalValue = t.TotalValue.Add(l.TotalValue)
		st := ""
		if i < len(saleTypes) {
			st = saleTypes[i]
		}
		key := st + "\x00" + l.Rate.Raw
		j, ok := idx[key]
		if !ok {
			j = len(t.ByRate)
			idx[key] = j
			t.ByRate = append(t.ByRate, RateSummary{SaleType: st, Rate: l.Rate.Raw})
		}
		rs := &t.ByRate[j]
		rs.ValueExclST = rs.ValueExclST.Add(l.ValueExclST)
		rs.SalesTax = rs.SalesTax.Add(l.SalesTax)
		rs.FurtherTax = rs.FurtherTax.Add(l.FurtherTax)
	}
	t.AmountPayable = t.TotalValue.Sub(t.STWithheld)
	return t
}
