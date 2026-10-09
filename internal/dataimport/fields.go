// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package dataimport

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// Field is an invoice particular that a column can supply.
type Field struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Group string `json:"group"`
	// Need is "required" (FBR needs it on every line), "recommended" or
	// "optional".
	Need string `json:"need"`
	Help string `json:"help"`
	Kind string `json:"kind"`
	syn  []string
}

// Fields lists what an import can carry, in the order shown to the user.
// The keys are the columns of the import template.
var Fields = []Field{
	{Key: "invoice_ref", Label: "Invoice number", Group: "Invoice", Need: "recommended", Kind: "text",
		Help: "Your own invoice or voucher number. Lines with the same number become one invoice, and a number imported before is skipped.",
		syn: []string{"invoice ref", "invoice no", "invoice number", "invoice num", "inv no", "inv number", "inv num", "inv", "invoice", "invoice id", "invoice code",
			"bill no", "bill number", "voucher no", "voucher number", "vch no", "vr no", "document no", "document number", "doc no", "doc number",
			"reference", "ref", "ref no", "reference no", "reference number", "sales invoice no", "sale invoice no", "si no", "order no", "erp ref",
			"external ref", "your ref", "receipt no", "transaction no", "txn no"}},
	{Key: "invoice_date", Label: "Invoice date", Group: "Invoice", Need: "required", Kind: "date",
		Help: "The date of supply. Day-first dates (09/10/2026), ISO dates and Excel dates are understood.",
		syn: []string{"invoice date", "date", "inv date", "bill date", "voucher date", "vch date", "document date", "doc date", "posting date",
			"transaction date", "txn date", "sale date", "supply date", "dated", "date of supply", "date of invoice"}},
	{Key: "doc_type", Label: "Document type", Group: "Invoice", Need: "optional", Kind: "doctype",
		Help: "Sale Invoice or Debit Note. Blank means Sale Invoice.",
		syn:  []string{"doc type", "document type", "invoice type", "voucher type", "type of document"}},
	{Key: "original_fbr_invoice_no", Label: "Original FBR invoice number", Group: "Invoice", Need: "optional", Kind: "text",
		Help: "For debit notes: the FBR number of the invoice being adjusted.",
		syn: []string{"original fbr invoice no", "original fbr invoice number", "original invoice", "original invoice no", "against invoice",
			"ref fbr invoice no", "reference fbr invoice no", "invoice ref no", "fbr ref", "original fbr no"}},
	{Key: "scenario_id", Label: "Sandbox scenario", Group: "Invoice", Need: "optional", Kind: "text",
		Help: "Only for the FBR sandbox (SN001–SN028).", syn: []string{"scenario", "scenario id", "scenario no"}},
	{Key: "advance_receipt", Label: "Advance receipt invoice", Group: "Invoice", Need: "optional", Kind: "yesno",
		Help: "Yes for an invoice issued on receipt of an advance (section 23(1)).", syn: []string{"advance receipt", "advance receipt invoice", "is advance"}},
	{Key: "advance_ref", Label: "Advance invoices adjusted", Group: "Invoice", Need: "optional", Kind: "text",
		Help: "On a final invoice, the FBR numbers of the advance receipt invoices it settles.", syn: []string{"advance ref", "advance reference", "advance invoice no"}},

	{Key: "buyer_name", Label: "Buyer name", Group: "Buyer", Need: "recommended", Kind: "text",
		Help: "The buyer's name as registered. Blank means a walk-in customer.",
		syn: []string{"buyer name", "buyer", "customer", "customer name", "client", "client name", "party", "party name", "account", "account name",
			"account title", "sold to", "bill to", "billed to", "consignee", "buyer business name", "name of buyer", "purchaser", "debtor", "name"}},
	{Key: "buyer_ntn_cnic", Label: "Buyer NTN or CNIC", Group: "Buyer", Need: "recommended", Kind: "ntn",
		Help: "7-digit NTN or 13-digit CNIC. Required for registered buyers.",
		syn: []string{"buyer ntn", "buyer ntn cnic", "buyer cnic", "ntn", "ntn cnic", "cnic", "ntn no", "ntn number", "customer ntn", "party ntn",
			"buyer ntncnic", "ntncnic", "national tax number", "registration no", "registration number", "reg no", "buyer registration no", "tax id", "ntn cnic no"}},
	{Key: "buyer_registration_type", Label: "Buyer registration type", Group: "Buyer", Need: "recommended", Kind: "regtype",
		Help: "Registered or Unregistered. Blank: registered when an NTN/CNIC is known to the customer master, otherwise unregistered.",
		syn: []string{"buyer registration type", "registration type", "registration status", "buyer type", "customer type", "registered unregistered",
			"reg type", "filer status", "tax status", "buyer status", "registered"}},
	{Key: "buyer_province", Label: "Buyer province", Group: "Buyer", Need: "required", Kind: "province",
		Help: "The province of the buyer (destination of supply).",
		syn: []string{"buyer province", "province", "destination province", "destination", "province of buyer", "state", "region",
			"place of supply", "destination of supply", "customer province", "province name"}},
	{Key: "buyer_address", Label: "Buyer address", Group: "Buyer", Need: "recommended", Kind: "text",
		Help: "The buyer's address. A city in the address also tells the province.",
		syn:  []string{"buyer address", "address", "customer address", "billing address", "party address", "client address", "delivery address", "city", "location"}},
	{Key: "withholding_mode", Label: "Withholding agent", Group: "Buyer", Need: "optional", Kind: "text",
		Help: "Blank, fraction (one-fifth) or full, when the buyer withholds sales tax.", syn: []string{"withholding mode", "withholding agent", "wht agent", "withholding"}},

	{Key: "product_code", Label: "Product code", Group: "Item", Need: "optional", Kind: "text",
		Help: "A code from your Products master; its HS code, unit, sale type and rate are used for anything the file leaves blank.",
		syn:  []string{"product code", "item code", "sku", "code", "item id", "product id", "part no", "part number", "article no", "stock code", "material code", "item no"}},
	{Key: "description", Label: "Description", Group: "Item", Need: "required", Kind: "text",
		Help: "What was sold. Matching names in your Products master fill in the HS code, unit, sale type and rate.",
		syn: []string{"description", "product description", "item description", "item", "item name", "product", "product name", "particulars",
			"goods", "details", "detail", "narration", "description of goods", "commodity", "service", "services", "goods description", "item details"}},
	{Key: "hs_code", Label: "HS code", Group: "Item", Need: "required", Kind: "hs",
		Help: "The PCT/HS code of the goods or service (e.g. 2523.2900).",
		syn:  []string{"hs code", "hs", "hs no", "pct", "pct code", "pct heading", "hscode", "tariff code", "harmonized code", "h s code", "hs pct", "pct hs code"}},
	{Key: "uom", Label: "Unit of measure", Group: "Item", Need: "required", Kind: "uom",
		Help: "FBR's unit for the HS code, e.g. KG or Numbers, pieces, units.",
		syn:  []string{"uom", "unit", "units", "unit of measure", "unit of measurement", "measurement unit", "u m", "uo m", "measure", "unit name"}},
	{Key: "quantity", Label: "Quantity", Group: "Item", Need: "required", Kind: "number",
		Help: "How many units were sold.", syn: []string{"quantity", "qty", "qnty", "qty sold", "no of units", "nos", "pcs", "quantity sold", "quantities", "qty pcs"}},
	{Key: "unit_price", Label: "Unit price", Group: "Item", Need: "required", Kind: "number",
		Help: "Price per unit excluding sales tax. Not needed when the value of the line is given.",
		syn:  []string{"unit price", "price", "rate per unit", "unit rate", "price per unit", "sale price", "selling price", "unit cost", "rate"}},
	{Key: "discount", Label: "Discount", Group: "Item", Need: "optional", Kind: "number",
		Help: "Discount amount on the line.", syn: []string{"discount", "disc", "discount amount", "disc amount", "trade discount", "discount rs"}},
	{Key: "value_excl_st", Label: "Value excluding sales tax", Group: "Item", Need: "optional", Kind: "number",
		Help: "Value of the line before sales tax; checked against quantity × price.",
		syn: []string{"value excl st", "value excluding sales tax", "value exclusive of sales tax", "amount", "value", "net amount", "taxable value",
			"taxable amount", "value of supply", "value of goods", "gross amount", "amount excl tax", "amount excl gst", "value sales excluding st",
			"sale value", "excl value", "amount before tax", "value excluding st", "line amount"}},
	{Key: "retail_price", Label: "Retail price", Group: "Item", Need: "optional", Kind: "number",
		Help: "Printed retail price for Third Schedule goods.",
		syn:  []string{"retail price", "mrp", "printed retail price", "fixed notified value or retail price", "notified value", "retail value", "max retail price"}},
	{Key: "sale_type", Label: "Sale type", Group: "Item", Need: "required", Kind: "saletype",
		Help: "FBR's sale type, e.g. Goods at standard rate (default).", syn: []string{"sale type", "type of sale", "transaction type", "sales type", "nature of supply"}},
	{Key: "rate", Label: "Sales tax rate", Group: "Item", Need: "required", Kind: "rate",
		Help: "e.g. 18%, 5% or Exempt.",
		syn: []string{"tax rate", "st rate", "sales tax rate", "gst rate", "gst %", "st %", "tax %", "rate of tax", "rate %", "sales tax %",
			"gst percent", "percentage", "gst rate %", "rate of sales tax", "rate"}},
	{Key: "sro_schedule_no", Label: "SRO / schedule", Group: "Item", Need: "optional", Kind: "text",
		Help: "SRO or schedule for reduced-rate and exempt supplies.", syn: []string{"sro schedule no", "sro", "sro no", "schedule", "schedule no", "sro schedule"}},
	{Key: "sro_item_serial_no", Label: "SRO item serial", Group: "Item", Need: "optional", Kind: "text",
		Help: "Serial number of the item in the SRO or schedule.", syn: []string{"sro item serial no", "sro serial", "item serial no", "sro serial no", "serial no of schedule"}},

	{Key: "sales_tax", Label: "Sales tax", Group: "Tax", Need: "optional", Kind: "number",
		Help: "Sales tax on the line; checked against the rate.",
		syn:  []string{"sales tax", "gst", "st", "sales tax amount", "gst amount", "st amount", "tax amount", "tax", "sales tax applicable", "output tax", "gst rs"}},
	{Key: "further_tax", Label: "Further tax", Group: "Tax", Need: "optional", Kind: "number", syn: []string{"further tax", "ft", "further tax amount"}},
	{Key: "extra_tax", Label: "Extra tax", Group: "Tax", Need: "optional", Kind: "number", syn: []string{"extra tax", "extra tax amount"}},
	{Key: "fed", Label: "FED", Group: "Tax", Need: "optional", Kind: "number", syn: []string{"fed", "federal excise duty", "excise duty", "fed payable", "fed amount"}},
	{Key: "st_withheld", Label: "Sales tax withheld", Group: "Tax", Need: "optional", Kind: "number",
		syn: []string{"st withheld", "sales tax withheld", "withheld", "wht", "sales tax withheld at source", "withholding tax", "st wht"}},
	{Key: "total_value", Label: "Total (for checking)", Group: "Tax", Need: "optional", Kind: "number",
		Help: "The line or invoice total in your file; it is compared with the computed total.",
		syn: []string{"total", "total value", "total amount", "grand total", "invoice total", "amount incl tax", "value including tax", "total values",
			"net total", "gross total", "inclusive value", "amount inclusive", "total rs", "total incl gst"}},
	{Key: "fed_type", Label: "FED type", Group: "Tax", Need: "optional", Kind: "text", syn: []string{"fed type"}},
	{Key: "fed_rate_text", Label: "FED rate", Group: "Tax", Need: "optional", Kind: "text", syn: []string{"fed rate", "fed rate text"}},
	{Key: "fed_unit_price", Label: "FED price per unit", Group: "Tax", Need: "optional", Kind: "number", syn: []string{"fed unit price", "fed price per unit"}},
	{Key: "fed_sro", Label: "FED SRO / schedule", Group: "Tax", Need: "optional", Kind: "text", syn: []string{"fed sro", "fed schedule"}},
	{Key: "fed_sro_serial", Label: "FED SRO serial", Group: "Tax", Need: "optional", Kind: "text", syn: []string{"fed sro serial", "fed serial"}},
}

func init() {
	// Synonyms are compared in the same plain form as headings.
	for i := range Fields {
		for j, s := range Fields[i].syn {
			Fields[i].syn[j] = norm(s)
		}
	}
}

// FieldByKey finds a field.
func FieldByKey(key string) (Field, bool) {
	for _, f := range Fields {
		if f.Key == key {
			return f, true
		}
	}
	return Field{}, false
}

// diKeys maps the FBR Digital Invoicing JSON names (normalised) to fields.
var diKeys = map[string]string{
	"invoicetype": "doc_type", "invoicedate": "invoice_date", "buyerntncnic": "buyer_ntn_cnic", "buyerbusinessname": "buyer_name",
	"buyerprovince": "buyer_province", "buyeraddress": "buyer_address", "buyerregistrationtype": "buyer_registration_type",
	"invoicerefno": "original_fbr_invoice_no", "scenarioid": "scenario_id", "hscode": "hs_code", "productdescription": "description",
	"rate": "rate", "uom": "uom", "quantity": "quantity", "totalvalues": "total_value", "valuesalesexcludingst": "value_excl_st",
	"fixednotifiedvalueorretailprice": "retail_price", "salestaxapplicable": "sales_tax", "salestaxwithheldatsource": "st_withheld",
	"extratax": "extra_tax", "furthertax": "further_tax", "sroscheduleno": "sro_schedule_no", "fedpayable": "fed", "discount": "discount",
	"saletype": "sale_type", "sroitemserialno": "sro_item_serial_no",
}

var camel = regexp.MustCompile(`([a-z0-9])([A-Z])`)

// norm lowers a heading to plain words: "Buyer NTN/CNIC #" → "buyer ntn cnic".
func norm(s string) string {
	s = camel.ReplaceAllString(strings.TrimSpace(s), "$1 $2")
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '%':
			b.WriteRune(r)
		default:
			b.WriteByte(' ')
		}
	}
	words := strings.Fields(b.String())
	// Currency and filler words say nothing about the column.
	out := words[:0]
	for _, w := range words {
		switch w {
		case "rs", "pkr", "rupees", "in", "the", "of", "per", "amt":
			if len(words) > 1 {
				if w == "amt" {
					out = append(out, "amount")
				}
				continue
			}
		}
		out = append(out, w)
	}
	return strings.Join(out, " ")
}

// fieldScore says how well a heading names a field (0–1).
func fieldScore(header string, f Field) float64 {
	h := norm(header)
	if h == "" {
		return 0
	}
	compact := strings.ReplaceAll(h, " ", "")
	if k, ok := diKeys[compact]; ok && k == f.Key {
		return 1
	}
	if h == strings.ReplaceAll(f.Key, "_", " ") {
		return 1
	}
	best := 0.0
	for i, s := range f.syn {
		switch {
		case h == s:
			return 0.98 - float64(i)*0.0005
		case strings.ReplaceAll(s, " ", "") == compact:
			best = max(best, 0.95)
		case len(s) > 2 && containsPhrase(h, s):
			// "customer name (as per NTN)" contains "customer name".
			best = max(best, 0.6+0.3*float64(len(s))/float64(len(h)))
		}
	}
	return best
}

func containsPhrase(h, phrase string) bool {
	return strings.HasPrefix(h, phrase+" ") || strings.HasSuffix(h, " "+phrase) || strings.Contains(h, " "+phrase+" ")
}

// headerScore rates a row as the heading row of a table.
func headerScore(cells []string) float64 {
	used := map[string]bool{}
	total := 0.0
	for _, c := range cells {
		if c == "" {
			continue
		}
		bestKey, best := "", 0.0
		for _, f := range Fields {
			if used[f.Key] {
				continue
			}
			if s := fieldScore(c, f); s > best {
				bestKey, best = f.Key, s
			}
		}
		if best >= 0.7 {
			used[bestKey] = true
			total += best
		}
	}
	return total
}

// HeaderRow guesses which of the first rows holds the column headings
// (reports often start with a title, the company name or a blank line).
func HeaderRow(rows [][]string) int {
	bestI, best := -1, 0.0
	for i, r := range rows[:min(len(rows), 30)] {
		if s := headerScore(r); s > best {
			bestI, best = i, s
		}
	}
	if bestI >= 0 && best >= 1.9 {
		return bestI
	}
	for i, r := range rows[:min(len(rows), 30)] {
		text := 0
		for _, c := range r {
			if c != "" {
				if _, isNum := Number(c); !isNum {
					text++
				}
			}
		}
		if text >= 2 {
			return i
		}
	}
	return 0
}

// Match is a suggested use for a column.
type Match struct {
	Col   int     `json:"col"`
	Field string  `json:"field"`
	Score float64 `json:"score"`
	By    string  `json:"by"` // "heading" or "values"
}

// Suggest proposes a field for each column from its heading and, where
// the heading says nothing, from the values in it.
func Suggest(headers []string, samples [][]string) []Match {
	type cand struct {
		col   int
		field string
		score float64
		by    string
	}
	var cands []cand
	for i, h := range headers {
		var vals []string
		if i < len(samples) {
			vals = samples[i]
		}
		for _, f := range Fields {
			s := fieldScore(h, f)
			if s < 0.6 {
				continue
			}
			s = adjustByValues(norm(h), f.Key, vals, s)
			if s >= 0.6 {
				cands = append(cands, cand{i, f.Key, s, "heading"})
			}
		}
		if k, s := byValues(vals); k != "" {
			cands = append(cands, cand{i, k, s, "values"})
		}
	}
	sort.SliceStable(cands, func(a, b int) bool {
		if cands[a].score != cands[b].score {
			return cands[a].score > cands[b].score
		}
		return cands[a].col < cands[b].col
	})
	colUsed := map[int]bool{}
	fieldUsed := map[string]bool{}
	var out []Match
	for _, c := range cands {
		if colUsed[c.col] || fieldUsed[c.field] {
			continue
		}
		colUsed[c.col], fieldUsed[c.field] = true, true
		out = append(out, Match{Col: c.col, Field: c.field, Score: c.score, By: c.by})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Col < out[b].Col })
	return out
}

// adjustByValues settles ambiguous headings from their values: in
// Pakistan "Rate" usually means the unit price, unless the values are
// percentages; "GST" may be the tax amount or the tax rate.
func adjustByValues(h, key string, vals []string, s float64) float64 {
	pct := share(vals, func(v string) bool { return strings.HasSuffix(strings.TrimSpace(v), "%") })
	switch h {
	case "rate":
		if key == "rate" {
			if pct >= 0.6 {
				return 0.97
			}
			return 0.55
		}
		if key == "unit_price" && pct >= 0.6 {
			return 0.5
		}
	case "gst", "tax", "st", "sales tax":
		if key == "sales_tax" && pct >= 0.6 {
			return 0.5
		}
		if key == "rate" && pct >= 0.6 {
			return 0.96
		}
	case "type":
		return 0
	case "name":
		if key == "buyer_name" {
			return 0.62
		}
	case "city", "location":
		if key == "buyer_address" {
			return 0.62
		}
	}
	return s
}

var (
	hsRe   = regexp.MustCompile(`^\d{4}\.\d{2,4}$|^\d{8}$`)
	ntnRe  = regexp.MustCompile(`^\d{7}(-\d)?$|^\d{13}$|^\d{5}-\d{7}-\d$|^\d{9}$`)
	pctRe  = regexp.MustCompile(`^\d{1,2}(\.\d+)?\s*%$`)
	saleRe = regexp.MustCompile(`(?i)(standard rate|reduced rate|exempt|zero.?rate|third schedule|goods at|services)`)
)

// byValues recognises a column without a useful heading from its values.
func byValues(vals []string) (string, float64) {
	if len(vals) == 0 {
		return "", 0
	}
	check := func(f func(string) bool) bool { return share(vals, f) >= 0.7 }
	switch {
	case check(func(v string) bool { return hsRe.MatchString(v) }):
		return "hs_code", 0.66
	case check(func(v string) bool { return ntnRe.MatchString(strings.TrimSpace(v)) }):
		return "buyer_ntn_cnic", 0.62
	case check(func(v string) bool { _, ok := Date(v); return ok && !isPlainNumber(v) }):
		return "invoice_date", 0.64
	case check(func(v string) bool { return pctRe.MatchString(strings.TrimSpace(v)) }):
		return "rate", 0.63
	case check(func(v string) bool { return Province(v) != "" }):
		return "buyer_province", 0.65
	case check(func(v string) bool { return RegType(v) != "" && !isPlainNumber(v) }):
		return "buyer_registration_type", 0.61
	case check(func(v string) bool { return saleRe.MatchString(v) }):
		return "sale_type", 0.6
	case check(func(v string) bool {
		l := strings.ToLower(v)
		return strings.Contains(l, "sale invoice") || strings.Contains(l, "debit note")
	}):
		return "doc_type", 0.62
	}
	return "", 0
}

func isPlainNumber(v string) bool {
	_, ok := Number(v)
	return ok && !strings.ContainsAny(v, "/-")
}

// share is the fraction of non-empty values satisfying f.
func share(vals []string, f func(string) bool) float64 {
	n, ok := 0, 0
	for _, v := range vals {
		if strings.TrimSpace(v) == "" {
			continue
		}
		n++
		if f(v) {
			ok++
		}
	}
	if n == 0 {
		return 0
	}
	return float64(ok) / float64(n)
}
