package validate

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/tax"

	"github.com/shopspring/decimal"
)

// Severity of an issue. Errors block submission; warnings do not.
type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
)

// Issue is one finding. Line is 0 for header fields and 1-based for items.
type Issue struct {
	Line     int      `json:"line"`
	Field    string   `json:"field"`
	Code     string   `json:"code,omitempty"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
}

func (i Issue) String() string {
	loc := "Header"
	if i.Line > 0 {
		loc = fmt.Sprintf("Line %d", i.Line)
	}
	code := ""
	if i.Code != "" {
		code = " [" + i.Code + "]"
	}
	return fmt.Sprintf("%s %s%s: %s", loc, i.Field, code, i.Message)
}

// Result collects issues.
type Result struct {
	Issues []Issue `json:"issues"`
}

// HasErrors reports whether any blocking issue exists.
func (r Result) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Severity == SevError {
			return true
		}
	}
	return false
}

// Errors returns only blocking issues.
func (r Result) Errors() []Issue { return r.filter(SevError) }

// Warnings returns only warnings.
func (r Result) Warnings() []Issue { return r.filter(SevWarning) }

func (r Result) filter(s Severity) []Issue {
	var out []Issue
	for _, i := range r.Issues {
		if i.Severity == s {
			out = append(out, i)
		}
	}
	return out
}

func (r *Result) add(line int, field, code string, sev Severity, format string, args ...any) {
	r.Issues = append(r.Issues, Issue{Line: line, Field: field, Code: code, Severity: sev, Message: fmt.Sprintf(format, args...)})
}

// OriginalInvoice describes the invoice a debit note refers to.
type OriginalInvoice struct {
	FBRNumber    string
	Date         string // YYYY-MM-DD
	ValueExclST  decimal.Decimal
	SalesTax     decimal.Decimal
	BuyerNTNCNIC string
}

// Context supplies information the payload alone does not contain.
type Context struct {
	Env   domain.Environment
	Today time.Time
	// Known reference lists (from FBR sync). Nil maps disable the check.
	KnownUOMs      map[string]bool
	KnownSaleTypes map[string]bool
	KnownProvinces map[string]bool
	// Original is set when validating a debit note.
	Original *OriginalInvoice
	// MaxBackdateDays warns when the invoice date is older than this (0 = 7).
	MaxBackdateDays int
	// CNICThreshold warns when an unregistered buyer without NTN/CNIC
	// receives an invoice above this value (0 disables).
	CNICThreshold decimal.Decimal
}

var (
	reDigits   = regexp.MustCompile(`^\d+$`)
	reHSCode   = regexp.MustCompile(`^\d{4}\.\d{4}$`)
	reScenario = regexp.MustCompile(`^SN\d{3}$`)
	reDate     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reFBRInvNo = regexp.MustCompile(`^[0-9A-Za-z]{5,}DI[0-9]{6,}$`)
	reNTNCheck = regexp.MustCompile(`^(\d{7})-(\d)$`)
)

// NormalizeRegNo strips separators from an NTN/CNIC. An NTN written with its
// check digit ("1234567-8") is reduced to the 7-digit NTN DI expects; the
// second return value then explains the change.
func NormalizeRegNo(s string) (string, string) {
	t := strings.TrimSpace(s)
	if m := reNTNCheck.FindStringSubmatch(t); m != nil {
		return m[1], "NTN check digit removed (" + t + " → " + m[1] + ")"
	}
	return strings.NewReplacer("-", "", " ", "", ".", "").Replace(t), ""
}

// ValidRegNoFormat reports whether s is a 7- or 9-digit NTN or 13-digit CNIC.
func ValidRegNoFormat(s string) bool {
	if !reDigits.MatchString(s) {
		return false
	}
	switch len(s) {
	case 7, 9, 13:
		return true
	}
	return false
}

// Payload validates a DI payload.
func Payload(p *fbr.InvoicePayload, ctx Context) Result {
	var r Result
	if ctx.Today.IsZero() {
		ctx.Today = time.Now()
	}
	backdate := ctx.MaxBackdateDays
	if backdate <= 0 {
		backdate = 7
	}

	// --- Header ---
	dt := domain.DocType(p.InvoiceType)
	if p.InvoiceType == "" {
		r.add(0, "invoiceType", "0011", SevError, "Invoice type is required.")
	} else if !dt.Valid() {
		r.add(0, "invoiceType", "0003", SevError, "Invoice type %q is not valid; use 'Sale Invoice' or 'Debit Note'.", p.InvoiceType)
	}

	var invDate time.Time
	if p.InvoiceDate == "" {
		r.add(0, "invoiceDate", "0042", SevError, "Invoice date is required.")
	} else if !reDate.MatchString(p.InvoiceDate) {
		r.add(0, "invoiceDate", "0005", SevError, "Invoice date must be in YYYY-MM-DD format.")
	} else if d, err := time.Parse("2006-01-02", p.InvoiceDate); err != nil {
		r.add(0, "invoiceDate", "0043", SevError, "Invoice date %q is not a valid date.", p.InvoiceDate)
	} else {
		invDate = d
		today := time.Date(ctx.Today.Year(), ctx.Today.Month(), ctx.Today.Day(), 0, 0, 0, 0, time.UTC)
		if d.After(today) {
			r.add(0, "invoiceDate", "0043", SevError, "Invoice date %s is in the future.", p.InvoiceDate)
		} else if today.Sub(d) > time.Duration(backdate)*24*time.Hour {
			r.add(0, "invoiceDate", "", SevWarning, "Invoice is dated %s, more than %d days ago. DI requires real-time reporting at the time of supply; back-dated invoices may attract scrutiny.", p.InvoiceDate, backdate)
		}
		if d.Year() != today.Year() || d.Month() != today.Month() {
			if !d.After(today) {
				r.add(0, "invoiceDate", "", SevWarning, "Invoice belongs to a previous tax period (%s). Check whether that period's return has already been filed.", d.Format("Jan 2006"))
			}
		}
	}

	// Seller
	if p.SellerNTNCNIC == "" {
		r.add(0, "sellerNTNCNIC", "0001", SevError, "Seller NTN/CNIC is required (set it in Company settings).")
	} else if !ValidRegNoFormat(p.SellerNTNCNIC) {
		r.add(0, "sellerNTNCNIC", "0001", SevError, "Seller NTN/CNIC %q must be a 7- or 9-digit NTN or a 13-digit CNIC.", p.SellerNTNCNIC)
	}
	if strings.TrimSpace(p.SellerBusinessName) == "" {
		r.add(0, "sellerBusinessName", "", SevError, "Seller business name is required.")
	}
	if strings.TrimSpace(p.SellerProvince) == "" {
		r.add(0, "sellerProvince", "0073", SevError, "Seller province is required.")
	} else if ctx.KnownProvinces != nil && !ctx.KnownProvinces[strings.ToUpper(p.SellerProvince)] {
		r.add(0, "sellerProvince", "0073", SevWarning, "Seller province %q is not in FBR's province list.", p.SellerProvince)
	}
	if strings.TrimSpace(p.SellerAddress) == "" {
		r.add(0, "sellerAddress", "", SevError, "Seller address is required.")
	}

	// Buyer
	regType := domain.RegistrationType(p.BuyerRegistrationType)
	if p.BuyerRegistrationType == "" {
		r.add(0, "buyerRegistrationType", "0012", SevError, "Buyer registration type is required.")
	} else if !regType.Valid() {
		r.add(0, "buyerRegistrationType", "0012", SevError, "Buyer registration type %q is not valid; use 'Registered' or 'Unregistered'.", p.BuyerRegistrationType)
	}
	if p.BuyerNTNCNIC == "" {
		if regType == domain.Registered {
			r.add(0, "buyerNTNCNIC", "0009", SevError, "A registered buyer must have an NTN or CNIC.")
		}
	} else if !ValidRegNoFormat(p.BuyerNTNCNIC) {
		r.add(0, "buyerNTNCNIC", "0002", SevError, "Buyer NTN/CNIC %q must be a 7- or 9-digit NTN or a 13-digit CNIC (digits only).", p.BuyerNTNCNIC)
	}
	if p.BuyerNTNCNIC != "" && p.BuyerNTNCNIC == p.SellerNTNCNIC {
		r.add(0, "buyerNTNCNIC", "0058", SevError, "Buyer and seller have the same registration number (self-invoicing).")
	}
	if strings.TrimSpace(p.BuyerBusinessName) == "" {
		r.add(0, "buyerBusinessName", "0010", SevError, "Buyer name is required.")
	}
	if strings.TrimSpace(p.BuyerProvince) == "" {
		r.add(0, "buyerProvince", "0074", SevError, "Buyer province (destination of supply) is required.")
	} else if ctx.KnownProvinces != nil && !ctx.KnownProvinces[strings.ToUpper(p.BuyerProvince)] {
		r.add(0, "buyerProvince", "0074", SevWarning, "Buyer province %q is not in FBR's province list.", p.BuyerProvince)
	}
	if strings.TrimSpace(p.BuyerAddress) == "" {
		r.add(0, "buyerAddress", "", SevWarning, "Buyer address is blank; section 23 requires the recipient's address on a tax invoice.")
	}

	// Debit note reference
	if dt == domain.DocDebitNote {
		if strings.TrimSpace(p.InvoiceRefNo) == "" {
			r.add(0, "invoiceRefNo", "0041", SevError, "A debit note must reference the FBR invoice number of the original invoice.")
		} else if !reFBRInvNo.MatchString(p.InvoiceRefNo) {
			r.add(0, "invoiceRefNo", "0039", SevWarning, "%q does not look like an FBR DI invoice number (e.g. 7000007DI1747119701593).", p.InvoiceRefNo)
		}
		if o := ctx.Original; o != nil {
			if o.Date != "" && p.InvoiceDate != "" && p.InvoiceDate < o.Date {
				r.add(0, "invoiceDate", "0035", SevError, "Debit note date %s is earlier than the original invoice date %s.", p.InvoiceDate, o.Date)
			}
			if o.BuyerNTNCNIC != "" && p.BuyerNTNCNIC != o.BuyerNTNCNIC {
				r.add(0, "buyerNTNCNIC", "", SevWarning, "Buyer differs from the original invoice buyer (%s).", o.BuyerNTNCNIC)
			}
		}
	}

	// Scenario id (sandbox only)
	switch ctx.Env {
	case domain.EnvSandbox:
		if p.ScenarioID == "" {
			r.add(0, "scenarioId", "", SevError, "Sandbox invoices must carry a scenario id (SN001–SN028).")
		} else if !reScenario.MatchString(p.ScenarioID) {
			r.add(0, "scenarioId", "", SevError, "Scenario id %q is not valid (expected SN001–SN028).", p.ScenarioID)
		}
	case domain.EnvProduction:
		if p.ScenarioID != "" {
			r.add(0, "scenarioId", "", SevWarning, "Scenario id is a sandbox-only field and will not be sent to production.")
		}
	}

	// --- Items ---
	if len(p.Items) == 0 {
		r.add(0, "items", "", SevError, "At least one line item is required.")
	}
	totalValue, totalTax := decimal.Zero, decimal.Zero
	for i := range p.Items {
		validateItem(&r, i+1, &p.Items[i], regType, ctx)
		totalValue = totalValue.Add(p.Items[i].ValueSalesExcludingST.Value)
		totalTax = totalTax.Add(p.Items[i].SalesTaxApplicable.Value)
	}

	if dt == domain.DocDebitNote && ctx.Original != nil {
		if ctx.Original.ValueExclST.IsPositive() && totalValue.GreaterThan(ctx.Original.ValueExclST) {
			r.add(0, "items", "0036", SevWarning, "Debit note value %s exceeds the original invoice value %s.", totalValue.StringFixed(2), ctx.Original.ValueExclST.StringFixed(2))
		}
		if ctx.Original.SalesTax.IsPositive() && totalTax.GreaterThan(ctx.Original.SalesTax) {
			r.add(0, "items", "0067", SevWarning, "Debit note sales tax %s exceeds the original invoice sales tax %s.", totalTax.StringFixed(2), ctx.Original.SalesTax.StringFixed(2))
		}
	}

	if regType == domain.Unregistered && p.BuyerNTNCNIC == "" && ctx.CNICThreshold.IsPositive() && totalValue.GreaterThan(ctx.CNICThreshold) {
		r.add(0, "buyerNTNCNIC", "", SevWarning, "Supply of %s to an unregistered buyer without CNIC. Record the buyer's CNIC for supplies above %s.", totalValue.StringFixed(2), ctx.CNICThreshold.StringFixed(0))
	}
	_ = invDate
	return r
}

func validateItem(r *Result, n int, it *fbr.ItemPayload, regType domain.RegistrationType, ctx Context) {
	st, known := domain.LookupSaleType(it.SaleType)
	if strings.TrimSpace(it.SaleType) == "" {
		r.add(n, "saleType", "0013", SevError, "Sale type is required.")
	} else if ctx.KnownSaleTypes != nil && !ctx.KnownSaleTypes[it.SaleType] {
		r.add(n, "saleType", "0013", SevWarning, "Sale type %q is not in FBR's synced list; check spelling.", it.SaleType)
	} else if ctx.KnownSaleTypes == nil && !known {
		r.add(n, "saleType", "0013", SevWarning, "Sale type %q is not in the built-in catalogue; check spelling.", it.SaleType)
	}

	if strings.TrimSpace(it.HSCode) == "" {
		r.add(n, "hsCode", "0044", SevError, "HS code is required.")
	} else if !reHSCode.MatchString(it.HSCode) {
		r.add(n, "hsCode", "0052", SevWarning, "HS code %q is not in FBR's PCT format NNNN.NNNN (e.g. 0101.2100).", it.HSCode)
	}
	if strings.TrimSpace(it.ProductDescription) == "" {
		r.add(n, "productDescription", "0024", SevError, "Product description is required.")
	}
	if strings.TrimSpace(it.UoM) == "" {
		r.add(n, "uoM", "0023", SevError, "Unit of measure is required.")
	} else if ctx.KnownUOMs != nil && !ctx.KnownUOMs[it.UoM] {
		r.add(n, "uoM", "0023", SevWarning, "UoM %q is not in FBR's list (UoM values are case sensitive).", it.UoM)
	}
	if st.Name == domain.STPotassiumChlor && it.UoM != "" && it.UoM != "KG" {
		r.add(n, "uoM", "0165", SevError, "Potassium chlorate must be invoiced in KG.")
	}

	rate := tax.ParseRate(it.Rate)
	if strings.TrimSpace(it.Rate) == "" {
		r.add(n, "rate", "0046", SevError, "Rate is required.")
	} else if !rate.Valid {
		r.add(n, "rate", "0046", SevWarning, "Rate %q could not be interpreted; FBR expects the exact SaleTypeToRate description.", it.Rate)
	}
	if known && st.Exempt && !rate.Exempt {
		r.add(n, "rate", "0046", SevError, "Exempt goods must have the rate 'Exempt'.")
	}
	if known && !st.Exempt && rate.Exempt {
		r.add(n, "rate", "0046", SevError, "Rate 'Exempt' is only valid with the sale type 'Exempt goods'.")
	}
	if known && st.Name == domain.STStandard && rate.Valid && !rate.IsStandardRate() {
		r.add(n, "rate", "0046", SevWarning, "Standard-rate goods are normally taxed at 18%%; %q was given.", it.Rate)
	}

	neg := func(field string, a fbr.Amount, code string) {
		if a.Value.IsNegative() {
			r.add(n, field, code, SevError, "%s cannot be negative.", field)
		}
	}
	if it.Quantity.Value.IsNegative() {
		r.add(n, "quantity", "0022", SevError, "Quantity cannot be negative.")
	} else if it.Quantity.Value.IsZero() {
		r.add(n, "quantity", "0022", SevWarning, "Quantity is zero.")
	}
	neg("valueSalesExcludingST", it.ValueSalesExcludingST, "0021")
	neg("salesTaxApplicable", it.SalesTaxApplicable, "0027")
	neg("furtherTax", it.FurtherTax, "0028")
	neg("extraTax", it.ExtraTax, "0029")
	neg("fedPayable", it.FEDPayable, "0030")
	neg("discount", it.Discount, "0031")
	neg("salesTaxWithheldAtSource", it.SalesTaxWithheldAtSource, "0055")
	neg("fixedNotifiedValueOrRetailPrice", it.FixedNotifiedValueOrRetailPrice, "0026")

	// Sales tax consistency (FBR: "Provided sales tax amount does not match
	// the calculated sales tax amount").
	if rate.Valid {
		var base decimal.Decimal
		if known && st.Basis == domain.BasisRetailPrice {
			base = it.FixedNotifiedValueOrRetailPrice.Value
			if base.IsZero() {
				r.add(n, "fixedNotifiedValueOrRetailPrice", "0175", SevError, "Third Schedule goods require the printed retail price.")
			}
		} else {
			base = it.ValueSalesExcludingST.Value
		}
		expected := tax.R2(tax.PercentOf(base, rate.Percent).Add(it.Quantity.Value.Mul(rate.PerUnit)))
		if rate.Exempt {
			expected = decimal.Zero
		}
		diff := it.SalesTaxApplicable.Value.Sub(expected).Abs()
		if diff.GreaterThan(decimal.NewFromFloat(0.01)) {
			r.add(n, "salesTaxApplicable", "0027", SevWarning, "Sales tax %s does not match %s × %s = %s; FBR rejects mismatched amounts.",
				it.SalesTaxApplicable.Value.StringFixed(2), base.StringFixed(2), it.Rate, expected.StringFixed(2))
		}
	}

	if known && st.SRORequired {
		if strings.TrimSpace(it.SROScheduleNo) == "" {
			r.add(n, "sroScheduleNo", "0077", SevError, "SRO/Schedule number is required for %q.", st.Name)
		}
		if strings.TrimSpace(it.SROItemSerialNo) == "" {
			r.add(n, "sroItemSerialNo", "0078", SevError, "SRO item serial number is required for %q.", st.Name)
		}
	} else if known && st.SROTypical && strings.TrimSpace(it.SROScheduleNo) == "" {
		r.add(n, "sroScheduleNo", "0077", SevWarning, "An SRO/Schedule reference is normally required for %q.", st.Name)
	}
	if strings.TrimSpace(it.SROScheduleNo) != "" && strings.TrimSpace(it.SROItemSerialNo) == "" && !(known && st.SRORequired) {
		r.add(n, "sroItemSerialNo", "0078", SevWarning, "SRO/Schedule given without an item serial number.")
	}

	if known && st.ExtraTaxMustBeEmpty && !it.ExtraTax.Empty {
		r.add(n, "extraTax", "0091", SevError, "Extra tax must be empty (not 0) for reduced-rate goods.")
	}

	if regType == domain.Registered && it.FurtherTax.Value.IsPositive() {
		r.add(n, "furtherTax", "0028", SevWarning, "Further tax is charged only on supplies to unregistered buyers.")
	}
	if it.SalesTaxWithheldAtSource.Value.IsPositive() {
		if regType == domain.Unregistered {
			r.add(n, "salesTaxWithheldAtSource", "0070", SevWarning, "Sales tax withholding is not normally allowed for an unregistered buyer.")
		}
		if it.SalesTaxWithheldAtSource.Value.GreaterThan(it.SalesTaxApplicable.Value) {
			r.add(n, "salesTaxWithheldAtSource", "0055", SevError, "Sales tax withheld exceeds the sales tax on the line.")
		} else if !it.SalesTaxWithheldAtSource.Value.Equal(it.SalesTaxApplicable.Value) {
			r.add(n, "salesTaxWithheldAtSource", "0008", SevWarning, "FBR expects sales tax withheld at source to be zero or equal to the sales tax.")
		}
	}

	// totalValues consistency (informational).
	want := tax.R2(tax.Sum(it.ValueSalesExcludingST.Value, it.SalesTaxApplicable.Value, it.FurtherTax.Value, it.ExtraTax.Value, it.FEDPayable.Value))
	if it.TotalValues.Value.IsPositive() && it.TotalValues.Value.Sub(want).Abs().GreaterThan(decimal.NewFromInt(1)) {
		r.add(n, "totalValues", "0025", SevWarning, "Total value %s differs from value + taxes = %s.", it.TotalValues.Value.StringFixed(2), want.StringFixed(2))
	}
}
