// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package printing

import (
	"embed"
	"encoding/base32"
	"encoding/base64"
	"html/template"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"einvoicing/internal/brand"
	"einvoicing/internal/domain"
	"einvoicing/internal/store"
	"einvoicing/internal/tax"

	"github.com/shopspring/decimal"
)

//go:embed templates/*.html
var tplFS embed.FS

// DefaultFBRLogo is the "FBR Digital Invoicing System" logo published in
// section 6 of PRAL's Technical Specification for DI API v1.12, which must be
// printed on every invoice. It is used unless a logo has been uploaded.
//
//go:embed assets/fbr-di-logo.jpg
var DefaultFBRLogo []byte

// DefaultFBRLogoMime is the media type of DefaultFBRLogo.
const DefaultFBRLogoMime = "image/jpeg"

var templates = template.Must(template.New("").ParseFS(tplFS, "templates/*.html"))

// Options controls rendering.
type Options struct {
	Format      string // "a4" (default) or "thermal"
	AutoPrint   bool
	ShowToolbar bool
	Duplicate   bool   // print "DUPLICATE" copy label
	Nonce       string // CSP nonce for the page's only script
	CompanyLogo []byte // raw image
	CompanyMime string
	FBRLogo     []byte // FBR DI logo (uploaded, or DefaultFBRLogo)
	FBRLogoMime string
	// SigningKey is the fingerprint of the key the invoice is signed with.
	SigningKey string
	Now        time.Time
}

type line struct {
	No                                            int
	Description, HSCode, SaleType, SRO, UoM, Rate string
	Qty, UnitPrice, Discount, Value, Retail       string
	SalesTax, FurtherTax, ExtraTax, FED, Total    string
	// FEDNote lists the federal excise duty particulars of rule
	// 150R(13)(aa)-(ff).
	FEDNote string
}

type totals struct {
	Value, Retail, SalesTax, FurtherTax, ExtraTax, FED, Total, Withheld, Payable, Discount string
}

type rateRow struct{ SaleType, Rate, Value, SalesTax string }

type cols struct {
	Discount, Retail, Further, Extra, FED, Withheld bool
	// FEDInValue: FED is part of the value of supply (section 2(46)), so it
	// is shown for information and not added to the total again.
	FEDInValue bool
}

type view struct {
	Title, InvoiceDate, Watermark, WatermarkSize, Notice, CopyLabel, AmountWords, PrintedAt, ProductName, Developer, ShortSeal string
	FBRNumber, Nonce                                                                                                           string
	// Particulars of rule 150R(13): tax period, software registration
	// number and the digital signature (rule 150R(4)(b)).
	TaxPeriod, SoftwareRegNo, Signature, SigningKey, OfflineIssued string
	// AdvanceNote explains an advance receipt invoice (section 23(1)).
	AdvanceNote                         string
	Company                             *store.Company
	Invoice                             *store.Invoice
	Settings                            store.PrintSettings
	CompanyLogo, FBRLogo                template.URL
	QR                                  template.HTML
	Lines                               []line
	Rates                               []rateRow
	T                                   totals
	Cols                                cols
	IsDebitNote, AutoPrint, ShowToolbar bool
	Sheets                              []sheet
}

// sheet is one printed copy of the A4 invoice; CopyLabel shadows the view's.
type sheet struct {
	*view
	CopyLabel string
}

// copyNames label the copies when more than one is printed per invoice.
var copyNames = []string{"BUYER'S COPY", "SELLER'S COPY", "OFFICE COPY", "COPY 4", "COPY 5"}

func dataURI(b []byte, mime string) template.URL {
	if len(b) == 0 {
		return ""
	}
	if mime == "" {
		mime = "image/png"
	}
	// Only image types are embedded (prevents script injection via data URIs).
	if !strings.HasPrefix(mime, "image/") {
		return ""
	}
	return template.URL("data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b))
}

func amt(d decimal.Decimal) string { return tax.FormatAmount(d) }

// Render writes the printable invoice HTML.
func Render(w io.Writer, c *store.Company, inv *store.Invoice, o Options) error {
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
	v := view{Company: c, Invoice: inv, Settings: c.PrintSettings, AutoPrint: o.AutoPrint, ShowToolbar: o.ShowToolbar, Nonce: o.Nonce,
		ProductName: brand.ProductName, Developer: brand.Developer, IsDebitNote: inv.DocType == domain.DocDebitNote}
	v.Title = "SALES TAX INVOICE"
	switch {
	case v.IsDebitNote:
		v.Title = "DEBIT NOTE"
	case inv.AdvanceReceipt:
		v.Title = "ADVANCE RECEIPT INVOICE"
		v.AdvanceNote = "Sales tax invoice issued on receipt of an advance payment, before the goods or services are supplied (section 23(1) read with section 2(44) of the Sales Tax Act, 1990). The advance is adjusted in the final invoice issued on supply."
	}
	if d, err := time.Parse("2006-01-02", inv.InvoiceDate); err == nil {
		v.InvoiceDate = d.Format("02-Jan-2006")
	} else {
		v.InvoiceDate = inv.InvoiceDate
	}
	loc, _ := time.LoadLocation("Asia/Karachi")
	if loc == nil {
		loc = time.FixedZone("PKT", 5*3600)
	}
	v.PrintedAt = o.Now.In(loc).Format("02-Jan-2006 15:04")
	if o.Duplicate || inv.PrintCount > 0 {
		v.CopyLabel = "DUPLICATE"
	}
	v.CompanyLogo = dataURI(o.CompanyLogo, o.CompanyMime)
	v.FBRLogo = dataURI(o.FBRLogo, o.FBRLogoMime)
	if d, err := time.Parse("2006-01-02", inv.InvoiceDate); err == nil {
		v.TaxPeriod = d.Format("January 2006")
	}
	v.SoftwareRegNo = c.SoftwareRegNo
	if inv.Signature != "" {
		v.Signature, v.SigningKey = ShortSignature(inv.Signature), o.SigningKey
	}

	reported := inv.Status == domain.StatusAccepted || inv.Status == domain.StatusCancelled
	if reported && inv.FBRInvoiceNumber != "" {
		v.FBRNumber = inv.FBRInvoiceNumber
		if svg, err := QRSVG(inv.FBRInvoiceNumber); err == nil {
			v.QR = template.HTML(svg) // generated by us, contains no user input other than the FBR number
		}
	}
	if reported && inv.OfflineSince != "" {
		// Rule 150XC: invoices generated while the system or the internet
		// was down are identified as issued in offline mode.
		v.OfflineIssued = "Issued in offline mode on " + shortTime(inv.OfflineSince, loc) + " and reported to FBR on " + inv.FBRDated + " (rule 150XC)."
	}
	switch {
	case inv.Status == domain.StatusCancelled:
		v.Watermark = "CANCELLED"
		v.Notice = "This invoice was cancelled on " + inv.CancelledAt + " — " + inv.CancelReason
	case inv.Status == domain.StatusQueued:
		// Issued while FBR was unreachable: a provisional copy may be handed
		// over, but the invoice must reach FBR within 24 hours of the
		// connection being restored, after which the reported copy is printed.
		v.Watermark = "PENDING FBR REPORTING"
		v.Notice = "Provisional copy issued while FBR Digital Invoicing was unreachable. It is reported to FBR automatically, within 24 hours of the connection being restored; the reported copy carries the FBR invoice number and QR code."
	case !reported && inv.OfflineSince != "":
		// Issued offline, but its upload has not been accepted yet (for
		// example FBR rejected it after the connection came back).
		v.Watermark = "PENDING FBR REPORTING"
		v.Notice = "Provisional copy issued while FBR Digital Invoicing was unreachable. It has not yet been accepted by FBR (status " + string(inv.Status) + ") and must be reported within 24 hours of the connection being restored; the reported copy carries the FBR invoice number and QR code."
	case !reported:
		v.Watermark = "DRAFT — NOT REPORTED TO FBR"
		v.Notice = "Not a valid sales tax invoice: it has not been reported to FBR Digital Invoicing (status " + string(inv.Status) + ")."
	case inv.Environment == domain.EnvSandbox:
		v.Watermark = "FBR SANDBOX — NOT A VALID TAX INVOICE"
	case inv.Environment == domain.EnvSimulator:
		v.Watermark = "TRAINING — NOT A VALID TAX INVOICE"
	}

	switch n := utf8.RuneCountInString(v.Watermark); {
	case n > 30:
		v.WatermarkSize = "wm-s"
	case n > 22:
		v.WatermarkSize = "wm-m"
	}

	style := tax.WordsSouthAsian
	if c.PrintSettings.WordsStyle == string(tax.WordsInternational) {
		style = tax.WordsInternational
	}
	payable := inv.Totals.AmountPayable
	v.AmountWords = tax.AmountInWords(payable, style)
	if len(inv.SealHash) >= 16 {
		v.ShortSeal = inv.SealHash[:16]
	}

	for _, it := range inv.Items {
		l := line{No: it.LineNo, Description: it.Description, HSCode: it.HSCode, SaleType: it.SaleType, UoM: it.UoM, Rate: it.Rate,
			Qty: tax.FormatQty(it.Quantity), UnitPrice: amt(it.UnitPrice), Discount: amt(it.Discount), Value: amt(it.ValueExclST),
			Retail: amt(it.RetailValue), SalesTax: amt(it.SalesTax), FurtherTax: amt(it.FurtherTax), ExtraTax: amt(it.ExtraTax),
			FED: amt(it.FED), Total: amt(it.TotalValue), FEDNote: fedNote(it)}
		if it.SROScheduleNo != "" {
			l.SRO = it.SROScheduleNo
			if it.SROItemSerialNo != "" {
				l.SRO += " S.No. " + it.SROItemSerialNo
			}
		}
		if it.UnitPrice.IsZero() && it.Quantity.IsPositive() {
			l.UnitPrice = amt(it.ValueExclST.Add(it.Discount).Div(it.Quantity))
		}
		v.Lines = append(v.Lines, l)
		v.Cols.Discount = v.Cols.Discount || (it.Discount.IsPositive() && c.PrintSettings.ShowDiscount)
		v.Cols.Retail = v.Cols.Retail || it.RetailValue.IsPositive()
		v.Cols.Further = v.Cols.Further || it.FurtherTax.IsPositive()
		v.Cols.Extra = v.Cols.Extra || it.ExtraTax.IsPositive()
		v.Cols.FED = v.Cols.FED || it.FED.IsPositive()
		v.Cols.Withheld = v.Cols.Withheld || it.STWithheld.IsPositive()
	}
	t := inv.Totals
	v.Cols.FEDInValue = t.FED.IsPositive() && t.TotalValue.Sub(tax.Sum(t.ValueExclST, t.SalesTax, t.FurtherTax, t.ExtraTax)).Abs().LessThan(decimal.NewFromFloat(0.01))
	v.T = totals{Value: amt(t.ValueExclST), Retail: amt(t.RetailValue), SalesTax: amt(t.SalesTax), FurtherTax: amt(t.FurtherTax),
		ExtraTax: amt(t.ExtraTax), FED: amt(t.FED), Total: amt(t.TotalValue), Withheld: amt(t.STWithheld), Payable: amt(t.AmountPayable),
		Discount: amt(t.Discount)}

	// Rate summary.
	idx := map[string]int{}
	type acc struct {
		st, rate   string
		val, stTax decimal.Decimal
	}
	var accs []*acc
	for _, it := range inv.Items {
		k := it.SaleType + "|" + it.Rate
		i, ok := idx[k]
		if !ok {
			i = len(accs)
			idx[k] = i
			accs = append(accs, &acc{st: it.SaleType, rate: it.Rate})
		}
		accs[i].val = accs[i].val.Add(it.ValueExclST)
		accs[i].stTax = accs[i].stTax.Add(it.SalesTax)
	}
	for _, a := range accs {
		v.Rates = append(v.Rates, rateRow{SaleType: a.st, Rate: a.rate, Value: amt(a.val), SalesTax: amt(a.stTax)})
	}

	copies := c.PrintSettings.Copies
	if copies < 1 {
		copies = 1
	}
	if copies > len(copyNames) {
		copies = len(copyNames)
	}
	if copies == 1 {
		v.Sheets = []sheet{{view: &v, CopyLabel: v.CopyLabel}}
	} else {
		for i := 0; i < copies; i++ {
			label := copyNames[i]
			if v.CopyLabel != "" {
				label = v.CopyLabel + " · " + label
			}
			v.Sheets = append(v.Sheets, sheet{view: &v, CopyLabel: label})
		}
	}

	name := "invoice_a4.html"
	if o.Format == "thermal" || (o.Format == "" && c.PrintSettings.PaperSize == "thermal80") {
		name = "invoice_thermal.html"
	}
	return templates.ExecuteTemplate(w, name, &v)
}

// fedNote lists a line's federal excise duty particulars (rule
// 150R(13)(aa)-(ff), added by SRO 1666(I)/2026): type, rate, price per unit,
// amount and the FED Schedule/SRO reference and serial number.
func fedNote(it *store.InvoiceItem) string {
	if !it.FED.IsPositive() && it.FEDType == "" && it.FEDSRO == "" {
		return ""
	}
	parts := []string{"FED"}
	if it.FEDType != "" {
		parts[0] += ": " + it.FEDType
	}
	rate := it.FEDRateText
	if rate == "" && it.FEDRate.IsPositive() {
		rate = it.FEDRate.String() + "%"
	}
	if rate != "" {
		parts = append(parts, "rate "+rate)
	}
	if it.FEDUnitPrice.IsPositive() {
		parts = append(parts, "price per unit "+amt(it.FEDUnitPrice))
	}
	if it.FED.IsPositive() {
		if it.SaleType == domain.STGoodsFED || it.SaleType == domain.STServicesFED {
			parts = append(parts, "amount "+amt(it.FED)+" in sales tax mode")
		} else {
			parts = append(parts, "amount "+amt(it.FED)+" payable otherwise than in sales tax mode")
		}
	}
	if it.FEDSRO != "" {
		ref := it.FEDSRO
		if it.FEDSROSerial != "" {
			ref += " S.No. " + it.FEDSROSerial
		}
		parts = append(parts, ref)
	}
	return strings.Join(parts, " · ")
}

// shortTime formats a stored RFC 3339 time in Pakistan time.
func shortTime(ts string, loc *time.Location) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.In(loc).Format("02-Jan-2006 15:04")
}

// ShortSignature is the printable form of a base64 digital signature: its
// first 20 base32 characters in groups of four.
func ShortSignature(sig string) string {
	raw, err := base64.StdEncoding.DecodeString(sig)
	if err != nil || len(raw) == 0 {
		return ""
	}
	b := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw)
	if len(b) > 20 {
		b = b[:20]
	}
	var parts []string
	for i := 0; i < len(b); i += 4 {
		parts = append(parts, b[i:min(i+4, len(b))])
	}
	return strings.Join(parts, "-")
}
