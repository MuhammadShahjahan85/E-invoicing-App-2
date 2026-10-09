// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package dataimport

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"einvoicing/internal/domain"

	"github.com/shopspring/decimal"
)

var (
	numClean  = regexp.MustCompile(`(?i)^(rs\.?|pkr|₨)\s*|\s*(rs\.?|pkr|₨|/-)$`)
	leadNum   = regexp.MustCompile(`^-?[0-9][0-9,]*(\.[0-9]+)?`)
	sciNotion = regexp.MustCompile(`^\d(\.\d+)?[eE]\+?\d+$`)
)

// Number reads an amount or quantity as written in Pakistani files:
// "1,23,456.50", "Rs. 1,000/-", "PKR 2,500", "(500)" for a negative amount.
func Number(s string) (decimal.Decimal, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return decimal.Zero, false
	}
	neg := false
	if strings.HasPrefix(t, "(") && strings.HasSuffix(t, ")") {
		neg, t = true, strings.TrimSpace(t[1:len(t)-1])
	}
	for i := 0; i < 2; i++ {
		t = strings.TrimSpace(numClean.ReplaceAllString(t, ""))
	}
	if strings.HasSuffix(t, "-") && !strings.HasPrefix(t, "-") {
		neg, t = true, strings.TrimSuffix(t, "-")
	}
	t = strings.ReplaceAll(strings.ReplaceAll(t, ",", ""), " ", "")
	d, err := decimal.NewFromString(t)
	if err != nil {
		return decimal.Zero, false
	}
	if neg {
		d = d.Neg()
	}
	return d, true
}

// Quantity reads a quantity that may carry its unit ("50 KG", "10 pcs").
func Quantity(s string) (decimal.Decimal, string, bool) {
	if d, ok := Number(s); ok {
		return d, "", true
	}
	t := strings.TrimSpace(s)
	m := leadNum.FindString(t)
	if m == "" {
		return decimal.Zero, "", false
	}
	d, ok := Number(m)
	return d, strings.TrimSpace(t[len(m):]), ok
}

// IsScientific recognises numbers Excel has turned into scientific
// notation (a CNIC shown as 3.52021E+12 has lost its digits).
func IsScientific(s string) bool { return sciNotion.MatchString(strings.TrimSpace(s)) }

// Rate normalises a sales tax rate: 18 → "18%", 0.18 → "18%", "18.00 %"
// → "18%". Text rates ("Exempt", "Rs.60/kg") are kept as written.
func Rate(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	pct := strings.HasSuffix(t, "%")
	d, ok := Number(strings.TrimSpace(strings.TrimSuffix(t, "%")))
	if !ok {
		return t
	}
	if !pct && d.IsPositive() && d.LessThan(decimal.NewFromInt(1)) {
		// A spreadsheet percentage cell holds 0.18 for 18%.
		d = d.Mul(decimal.NewFromInt(100))
	}
	return d.Round(4).String() + "%"
}

var dateLayouts = []string{
	"2006-01-02", "2006/01/02", "2006.01.02", "20060102",
	"02/01/2006", "2/1/2006", "02-01-2006", "2-1-2006", "02.01.2006", "2.1.2006",
	"02/01/06", "2/1/06", "02-01-06", "2-1-06",
	"02-Jan-2006", "2-Jan-2006", "02-Jan-06", "2-Jan-06", "02 Jan 2006", "2 Jan 2006", "02 January 2006", "2 January 2006",
	"Jan 2, 2006", "January 2, 2006", "Jan 02, 2006", "02-January-2006", "2006-Jan-02",
	"Monday, 2 January 2006", "Mon, 02 Jan 2006",
}

var dateTimeTail = regexp.MustCompile(`[ T]\d{1,2}:\d{2}(:\d{2})?(\.\d+)?\s*(am|pm|AM|PM)?(Z|[+-]\d{2}:?\d{2})?$`)

// Date reads a date written the Pakistani way (day first), in ISO form,
// with month names, with a time, or as an Excel serial number.
func Date(s string) (string, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", false
	}
	t = strings.TrimSpace(dateTimeTail.ReplaceAllString(t, ""))
	for _, l := range dateLayouts {
		if d, err := time.Parse(l, t); err == nil && d.Year() > 1990 && d.Year() < 2100 {
			return d.Format("2006-01-02"), true
		}
	}
	// Month-first dates (01/31/2026) when the day cannot be first.
	for _, l := range []string{"01/02/2006", "1/2/2006", "01-02-2006", "1/2/06"} {
		if d, err := time.Parse(l, t); err == nil && d.Year() > 1990 && d.Year() < 2100 {
			return d.Format("2006-01-02"), true
		}
	}
	// Excel serial date (1 = 1900-01-01).
	if n, err := strconv.ParseFloat(t, 64); err == nil && n > 32874 && n < 73051 {
		d := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).Add(time.Duration(n*24) * time.Hour)
		return d.Format("2006-01-02"), true
	}
	return "", false
}

// RegType reads a registration status: "Registered", "Yes", "Filer",
// "Active" → Registered; "Unregistered", "No", "Non-filer", "Walk-in"
// → Unregistered. It returns "" when the value says neither.
func RegType(s string) string {
	switch strings.ToLower(strings.Join(strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(s)), " ")) {
	case "registered", "reg", "r", "yes", "y", "filer", "active", "atl", "active taxpayer", "registered buyer":
		return string(domain.Registered)
	case "unregistered", "un registered", "unreg", "u", "no", "n", "non filer", "nonfiler", "inactive", "walk in", "walkin", "unregistered buyer", "end consumer", "consumer":
		return string(domain.Unregistered)
	}
	return ""
}

// NTN cleans an NTN or CNIC: "1234567-8" (NTN with its check digit) →
// "1234567", "35202-1234567-1" → "3520212345671".
func NTN(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if m := regexp.MustCompile(`^(\d{7})-\d$`).FindStringSubmatch(t); m != nil {
		return m[1]
	}
	var b strings.Builder
	for _, r := range t {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return t
	}
	return b.String()
}

// Province recognises a province (or territory) name in any common
// spelling; "" when the value is not one.
func Province(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	p := domain.NormalizeProvince(t)
	for _, q := range domain.Provinces {
		if q.Name == p {
			return p
		}
	}
	return ""
}

// cityProvince maps Pakistan's main cities and towns to their province, to
// work out the destination of supply from an address.
var cityProvince = map[string]string{}

func init() {
	add := func(prov string, cities ...string) {
		for _, c := range cities {
			cityProvince[c] = prov
		}
	}
	add("PUNJAB", "lahore", "faisalabad", "rawalpindi", "gujranwala", "multan", "sialkot", "bahawalpur", "sargodha", "sheikhupura", "jhang",
		"rahim yar khan", "gujrat", "kasur", "sahiwal", "okara", "wah cantt", "dera ghazi khan", "chiniot", "kamoke", "mandi bahauddin",
		"jhelum", "sadiqabad", "khanewal", "hafizabad", "muzaffargarh", "khanpur", "attock", "chakwal", "vehari", "mianwali", "bahawalnagar",
		"narowal", "pakpattan", "lodhran", "toba tek singh", "taxila", "murree", "raiwind", "daska", "wazirabad", "burewala", "jaranwala")
	add("SINDH", "karachi", "hyderabad", "sukkur", "larkana", "nawabshah", "shaheed benazirabad", "mirpur khas", "jacobabad", "shikarpur",
		"khairpur", "dadu", "thatta", "badin", "kotri", "tando adam", "tando allahyar", "umerkot", "ghotki", "sanghar", "port qasim", "nooriabad")
	add("KHYBER PAKHTUNKHWA", "peshawar", "mardan", "mingora", "swat", "kohat", "abbottabad", "dera ismail khan", "mansehra", "nowshera",
		"charsadda", "swabi", "bannu", "haripur", "chitral", "hangu", "karak", "timergara", "battagram", "lakki marwat", "hattar", "gadoon")
	add("BALOCHISTAN", "quetta", "gwadar", "turbat", "khuzdar", "hub", "chaman", "sibi", "zhob", "loralai", "dera murad jamali", "uthal", "lasbela")
	add("CAPITAL TERRITORY", "islamabad")
	add("AZAD JAMMU AND KASHMIR", "muzaffarabad", "mirpur", "kotli", "bhimber", "bagh", "rawalakot")
	add("GILGIT BALTISTAN", "gilgit", "skardu", "hunza", "chilas")
}

var wordRe = regexp.MustCompile(`[a-z]+`)

// ProvinceFromAddress works out the province from a city or province named
// in an address; "" when none is found.
func ProvinceFromAddress(addr string) string {
	if p := Province(addr); p != "" {
		return p
	}
	words := wordRe.FindAllString(strings.ToLower(addr), -1)
	// Later words (the city usually ends an address) win; check pairs and
	// triples for names such as "rahim yar khan".
	for i := len(words) - 1; i >= 0; i-- {
		for n := 3; n >= 1; n-- {
			if i-n+1 < 0 {
				continue
			}
			cand := strings.Join(words[i-n+1:i+1], " ")
			if p, ok := cityProvince[cand]; ok {
				return p
			}
			if p := Province(cand); p != "" && n <= 2 {
				return p
			}
		}
	}
	return ""
}

// DocType reads a document type.
func DocType(s string) string {
	l := strings.ToLower(strings.TrimSpace(s))
	switch {
	case l == "":
		return ""
	case strings.Contains(l, "debit") || l == "dn":
		return string(domain.DocDebitNote)
	case strings.Contains(l, "sale") || strings.Contains(l, "invoice") || l == "si" || l == "inv":
		return string(domain.DocSaleInvoice)
	}
	return strings.TrimSpace(s)
}

// YesNo reads yes/no values.
func YesNo(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "no", "n", "0", "false", "f":
		return false, true
	case "yes", "y", "1", "true", "t":
		return true, true
	}
	return false, false
}

// uomSynonyms map common ways of writing a unit to FBR's unit names.
var uomSynonyms = map[string]string{
	"pc": "Numbers, pieces, units", "piece": "Numbers, pieces, units", "pieces": "Numbers, pieces, units", "nos": "Numbers, pieces, units",
	"no.": "Numbers, pieces, units", "number": "Numbers, pieces, units", "numbers": "Numbers, pieces, units", "unit": "Numbers, pieces, units",
	"units": "Numbers, pieces, units", "each": "Numbers, pieces, units", "ea": "Numbers, pieces, units", "qty": "Numbers, pieces, units",
	"kgs": "KG", "kilo": "KG", "kilos": "KG", "kilograms": "Kilogram", "ton": "MT", "tons": "MT", "tonne": "MT", "tonnes": "MT",
	"metric ton": "MT", "metric tons": "MT", "ltr": "Liter", "ltrs": "Liter", "litre": "Liter", "litres": "Liter", "liters": "Liter", "l": "Liter",
	"g": "Gram", "gm": "Gram", "gms": "Gram", "grams": "Gram", "doz": "Dozen", "dz": "Dozen", "dozens": "Dozen", "m": "Meter", "mtr": "Meter",
	"mtrs": "Meter", "metre": "Meter", "meters": "Meter", "metres": "Meter", "sqm": "Square Metre", "sq m": "Square Metre", "sq.m": "Square Metre",
	"square meter": "Square Metre", "sqft": "Square Foot", "sq ft": "Square Foot", "sq.ft": "Square Foot", "square feet": "Square Foot",
	"sq yd": "SqY", "sq yards": "SqY", "cubic meter": "Cubic Metre", "cbm": "Cubic Metre", "m3": "Cubic Metre", "pair": "Pair", "pairs": "Pair",
	"pack": "Packs", "packet": "Packs", "packets": "Packs", "pkt": "Packs", "pkts": "Packs", "sets": "SET", "bags": "Bag", "bag": "Bag",
	"barrel": "Barrels", "bbl": "Barrels", "lb": "Pound", "lbs": "Pound", "pounds": "Pound", "gallons": "Gallon", "gal": "Gallon", "kwh": "KWH",
}

// UOM matches a unit to FBR's list (exact spelling matters to FBR); ok is
// false when the unit is not recognised.
func UOM(s string, list []string) (string, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", false
	}
	for _, u := range list {
		if strings.EqualFold(u, t) {
			return u, true
		}
	}
	if c, ok := uomSynonyms[strings.ToLower(t)]; ok {
		for _, u := range list {
			if u == c {
				return u, true
			}
		}
	}
	return "", false
}

// SaleType matches a sale type to FBR's list, accepting short forms such
// as "standard", "reduced", "exempt", "zero rated", "3rd schedule".
func SaleType(s string, list []string) (string, bool) {
	t := strings.TrimSpace(s)
	if t == "" {
		return "", false
	}
	for _, st := range list {
		if strings.EqualFold(st, t) {
			return st, true
		}
	}
	if st, ok := domain.LookupSaleType(t); ok {
		return st.Name, true
	}
	l := strings.ToLower(t)
	short := map[string]string{"standard": domain.STStandard, "standard rate": domain.STStandard, "normal": domain.STStandard,
		"taxable": domain.STStandard, "reduced": domain.STReduced, "reduced rate": domain.STReduced, "exempt": domain.STExempt,
		"exempted": domain.STExempt, "zero": domain.STZeroRated, "zero rated": domain.STZeroRated, "zero-rated": domain.STZeroRated,
		"zero rate": domain.STZeroRated, "3rd schedule": domain.STThirdSchedule, "third schedule": domain.STThirdSchedule,
		"retail price": domain.STThirdSchedule, "services": domain.STServices, "service": domain.STServices}
	if c, ok := short[l]; ok {
		return c, true
	}
	return "", false
}
