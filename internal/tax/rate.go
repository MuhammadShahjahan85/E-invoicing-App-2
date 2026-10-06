package tax

import (
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

// Rate is a parsed FBR rate description. FBR returns rates from the
// SaleTypeToRate reference API as free text (ratE_DESC) and expects exactly
// the same text back in the "rate" field of each invoice line. Examples:
//
//	"18%"                                    -> 18% ad valorem
//	"1.43%"                                  -> 1.43% ad valorem
//	"Exempt"                                 -> exempt, no tax
//	"Rs.3"                                   -> Rs.3 per unit (specific rate)
//	"18% along with rupees 60 per kilogram"  -> 18% + Rs.60 per unit (KG)
//	"Rs.1500/MT"                             -> Rs.1500 per unit (MT)
type Rate struct {
	// Raw is the exact description sent to FBR.
	Raw string `json:"raw"`
	// Percent is the ad valorem component (e.g. 18 for "18%").
	Percent decimal.Decimal `json:"percent"`
	// PerUnit is the specific (fixed) rupee amount per unit of quantity.
	PerUnit decimal.Decimal `json:"perUnit"`
	// Exempt is true for "Exempt".
	Exempt bool `json:"exempt"`
	// Valid is false when the text could not be interpreted.
	Valid bool `json:"valid"`
}

var (
	rePercent = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*%`)
	// "Rs.3", "Rs 3", "Rs. 1,500", "rupees 60", "PKR 25"
	reRupees = regexp.MustCompile(`(?i)(?:rs\.?|rupees?|pkr)\s*([\d,]+(?:\.\d+)?)`)
)

// ParseRate interprets an FBR rate description.
func ParseRate(desc string) Rate {
	raw := strings.TrimSpace(desc)
	r := Rate{Raw: raw, Percent: Zero, PerUnit: Zero}
	if raw == "" {
		return r
	}
	low := strings.ToLower(raw)
	if strings.HasPrefix(low, "exempt") {
		r.Exempt = true
		r.Valid = true
		return r
	}
	if m := rePercent.FindStringSubmatch(raw); m != nil {
		if p, err := decimal.NewFromString(m[1]); err == nil {
			r.Percent = p
			r.Valid = true
		}
	}
	if m := reRupees.FindStringSubmatch(raw); m != nil {
		if v, err := decimal.NewFromString(strings.ReplaceAll(m[1], ",", "")); err == nil {
			r.PerUnit = v
			r.Valid = true
		}
	}
	if !r.Valid {
		// A bare number such as "18" is treated as a percentage.
		if p, err := decimal.NewFromString(raw); err == nil {
			r.Percent = p
			r.Valid = true
		}
	}
	return r
}

// FormatPercentRate renders a percentage the way FBR writes it: "18%",
// "1.43%", "0%" (no trailing zeros).
func FormatPercentRate(pct decimal.Decimal) string {
	return pct.String() + "%"
}

// IsStandardRate reports whether the rate is the standard 18%.
func (r Rate) IsStandardRate() bool {
	return !r.Exempt && r.PerUnit.IsZero() && r.Percent.Equal(decimal.NewFromInt(18))
}

// IsZeroRate reports whether the rate charges no tax (0% or exempt).
func (r Rate) IsZeroRate() bool {
	return r.Exempt || (r.Percent.IsZero() && r.PerUnit.IsZero())
}
