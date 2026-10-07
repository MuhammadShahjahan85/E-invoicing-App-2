// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

package tax

import (
	"strings"

	"github.com/shopspring/decimal"
)

// WordsStyle selects the numbering system for amounts in words.
type WordsStyle string

const (
	// WordsSouthAsian uses crore/lakh grouping, customary on Pakistani invoices.
	WordsSouthAsian WordsStyle = "south_asian"
	// WordsInternational uses billion/million grouping.
	WordsInternational WordsStyle = "international"
)

var ones = []string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine",
	"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"}
var tens = []string{"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"}

func below100(n int64) string {
	if n < 20 {
		return ones[n]
	}
	t := tens[n/10]
	if n%10 != 0 {
		return t + "-" + ones[n%10]
	}
	return t
}

func below1000(n int64) string {
	var parts []string
	if n >= 100 {
		parts = append(parts, ones[n/100]+" Hundred")
		n %= 100
	}
	if n > 0 {
		parts = append(parts, below100(n))
	}
	return strings.Join(parts, " ")
}

func intWords(n int64, style WordsStyle) string {
	if n == 0 {
		return "Zero"
	}
	var parts []string
	if style == WordsInternational {
		units := []struct {
			v    int64
			name string
		}{{1_000_000_000_000, "Trillion"}, {1_000_000_000, "Billion"}, {1_000_000, "Million"}, {1_000, "Thousand"}}
		for _, u := range units {
			if n >= u.v {
				parts = append(parts, intWords(n/u.v, style)+" "+u.name)
				n %= u.v
			}
		}
	} else {
		if n >= 10_000_000 {
			parts = append(parts, intWords(n/10_000_000, style)+" Crore")
			n %= 10_000_000
		}
		if n >= 100_000 {
			parts = append(parts, below100(n/100_000)+" Lakh")
			n %= 100_000
		}
		if n >= 1_000 {
			parts = append(parts, below100(n/1_000)+" Thousand")
			n %= 1_000
		}
	}
	if n > 0 {
		parts = append(parts, below1000(n))
	}
	return strings.Join(parts, " ")
}

// AmountInWords renders a rupee amount, e.g.
// 123456.78 -> "Rupees One Lakh Twenty-Three Thousand Four Hundred Fifty-Six and Paisa Seventy-Eight Only".
func AmountInWords(amount decimal.Decimal, style WordsStyle) string {
	a := R2(amount)
	neg := a.IsNegative()
	a = a.Abs()
	rupees := a.Truncate(0)
	paisa := a.Sub(rupees).Mul(Hundred).Round(0).IntPart()
	s := "Rupees " + intWords(rupees.IntPart(), style)
	if paisa > 0 {
		s += " and Paisa " + below100(paisa)
	}
	s += " Only"
	if neg {
		s = "Minus " + s
	}
	return s
}
