// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

// Package tax implements the sales tax computation rules applied to every
// invoice line before it is reported to FBR Digital Invoicing.
//
// All arithmetic uses exact decimals. Monetary amounts are rounded half-up
// (away from zero) to 2 decimal places per component per line, and invoice
// totals are the sum of the rounded line amounts — the same convention FBR
// uses when it re-computes salesTaxApplicable from valueSalesExcludingST and
// the rate during validation.
package tax

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Zero is a convenience zero value.
var Zero = decimal.Zero

// Hundred is 100 as a decimal.
var Hundred = decimal.NewFromInt(100)

// R2 rounds a monetary amount to 2 decimal places (half away from zero).
func R2(d decimal.Decimal) decimal.Decimal { return d.Round(2) }

// R4 rounds a quantity to 4 decimal places, the precision accepted by DI.
func R4(d decimal.Decimal) decimal.Decimal { return d.Round(4) }

// D parses a decimal from a string, returning zero for blank input.
func D(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return Zero, nil
	}
	return decimal.NewFromString(s)
}

// MustD parses a decimal and panics on error (tests and constants only).
func MustD(s string) decimal.Decimal {
	d, err := D(s)
	if err != nil {
		panic(err)
	}
	return d
}

// F converts a float to a decimal (used for JSON numbers from FBR samples).
func F(f float64) decimal.Decimal { return decimal.NewFromFloat(f) }

// Ptr returns a pointer to a decimal.
func Ptr(d decimal.Decimal) *decimal.Decimal { return &d }

// FormatAmount renders an amount with thousands separators and 2 decimals,
// e.g. 1234567.5 -> "1,234,567.50".
func FormatAmount(d decimal.Decimal) string {
	return groupThousands(d.StringFixed(2))
}

// FormatQty renders a quantity with up to 4 decimals and no trailing zeros.
func FormatQty(d decimal.Decimal) string {
	s := R4(d).String()
	return groupThousands(s)
}

func groupThousands(s string) string {
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	var b strings.Builder
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	out := b.String() + frac
	if neg {
		out = "-" + out
	}
	return out
}

// PercentOf returns base * pct / 100 (unrounded).
func PercentOf(base, pct decimal.Decimal) decimal.Decimal {
	return base.Mul(pct).Div(Hundred)
}

// Sum adds decimals.
func Sum(ds ...decimal.Decimal) decimal.Decimal {
	t := Zero
	for _, d := range ds {
		t = t.Add(d)
	}
	return t
}

// Money is a JSON friendly wrapper used in API responses where a plain
// number with exactly two decimals is wanted.
type Money struct{ decimal.Decimal }

// MarshalJSON renders the amount as a JSON number with two decimals.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.StringFixed(2)), nil
}

// String implements fmt.Stringer.
func (m Money) String() string { return fmt.Sprintf("%s", m.StringFixed(2)) }
