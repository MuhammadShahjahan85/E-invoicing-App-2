// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package printing

import (
	"fmt"
	"io"
	"strings"

	"einvoicing/internal/brand"
	"einvoicing/internal/pdf"
	"einvoicing/internal/store"
)

// RenderPDF writes the invoice as an A4 PDF carrying the same particulars
// as the printed invoice: seller and buyer, FBR invoice number with the
// prescribed QR code and Digital Invoicing logo, lines, taxes, totals,
// amount in words, the digital signature and the integrity seal. Each copy
// set in the print settings starts on a new page.
func RenderPDF(w io.Writer, c *store.Company, inv *store.Invoice, o Options) error {
	v := prepare(c, inv, &o)
	d := pdf.New(false, pdf.Meta{Title: titleCase(v.Title) + " " + inv.InternalNo, Subject: "Sales tax invoice " + inv.InternalNo,
		Company: inv.SellerName, Plain: true, Watermark: v.Watermark, Generated: o.Now, Product: brand.ProductName, Developer: brand.Developer})
	for i, sh := range v.Sheets {
		if i > 0 {
			d.NewPage()
		}
		invoiceSheet(d, v, sh.CopyLabel, o)
	}
	return d.Output(w)
}

func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

func invoiceSheet(d *pdf.Doc, v *view, copyLabel string, o Options) {
	inv, c := v.Invoice, v.Company
	x0, W := d.Left(), d.Width()
	y := d.Y()

	// Letterhead: logo and seller on the left, FBR block on the right.
	sellerX := x0
	if lw, _ := d.Image("company-logo", o.CompanyLogo, o.CompanyMime, x0, y, 34, 20, false); lw > 0 {
		sellerX += lw + 4
	}
	fbrW := 0.0
	if v.FBRNumber != "" {
		fbrW = 72
	}
	sw := W - (sellerX - x0) - fbrW - 4
	yy := y
	d.SetFont(pdf.Title, 14.5)
	d.TextColor(pdf.Ink)
	for _, l := range d.Wrap(inv.SellerName, sw) {
		d.Text(sellerX, yy, sw, 6.4, l, "L")
		yy += 6.4
	}
	d.SetFont(pdf.Regular, 8.4)
	d.TextColor(pdf.Text2)
	addr := inv.SellerAddress
	if c.City != "" {
		addr += ", " + c.City
	}
	lines := d.Wrap(addr, sw)
	lines = append(lines, inv.SellerProvince)
	for _, l := range lines {
		d.Text(sellerX, yy, sw, pdf.LineH(8.4), l, "L")
		yy += pdf.LineH(8.4)
	}
	ids := "NTN/CNIC: " + inv.SellerNTNCNIC
	if c.STRN != "" {
		ids += "    STRN: " + c.STRN
	}
	d.SetFont(pdf.SemiBold, 8.4)
	d.TextColor(pdf.Ink)
	d.Text(sellerX, yy, sw, pdf.LineH(8.4), ids, "L")
	yy += pdf.LineH(8.4)
	if contact := strings.TrimSpace(c.Phone + "   " + c.Email); contact != "" {
		d.SetFont(pdf.Regular, 7.6)
		d.TextColor(pdf.Text3)
		d.Text(sellerX, yy, sw, pdf.LineH(7.6), contact, "L")
		yy += pdf.LineH(7.6)
	}
	bottom := yy
	if v.FBRNumber != "" {
		qx := x0 + W - 25.4
		bx := qx - 4 - 40
		by := y
		if lw, lh := d.Image("fbr-di-logo", o.FBRLogo, o.FBRLogoMime, bx+3, by, 34, 15, false); lw > 0 {
			by += lh + 1.5
		} else {
			d.DrawColor(pdf.GreenDark)
			d.Box(bx+3, by, 34, 9, 1.2, "D")
			d.SetFont(pdf.Bold, 6.6)
			d.TextColor(pdf.GreenDark)
			d.Text(bx, by+1.2, 40, 3.2, "FBR DIGITAL", "C")
			d.Text(bx, by+4.4, 40, 3.2, "INVOICING SYSTEM", "C")
			by += 10.5
		}
		d.SetFont(pdf.Regular, 6.8)
		d.TextColor(pdf.Text2)
		d.Text(bx, by, 40, 3.2, "FBR Invoice No.", "C")
		d.SetFont(pdf.Bold, 7.4)
		d.TextColor(pdf.Ink)
		by += 3.4
		for _, l := range splitEvery(v.FBRNumber, 40, d) {
			d.Text(bx, by, 40, 3.6, l, "C")
			by += 3.6
		}
		if q, err := newQR(v.FBRNumber); err == nil {
			d.QR(q.Bitmap(), qx, y, 25.4)
		}
		bottom = max(bottom, by, y+25.4)
	}
	y = bottom + 3
	d.Rule(x0, x0+W, y, 0.6, pdf.Navy)
	y += 4

	// Title and particulars.
	meta := [][2]string{{"Invoice No.", inv.InternalNo}, {"Date", v.InvoiceDate}}
	if v.TaxPeriod != "" {
		meta = append(meta, [2]string{"Tax period", v.TaxPeriod})
	}
	if inv.FBRDated != "" {
		meta = append(meta, [2]string{"FBR date/time", inv.FBRDated})
	}
	if inv.InvoiceRefNo != "" {
		k := "Reference"
		if v.IsDebitNote {
			k = "Against FBR inv."
		}
		meta = append(meta, [2]string{k, inv.InvoiceRefNo})
	}
	if inv.ExternalRef != "" {
		meta = append(meta, [2]string{"Your ref.", inv.ExternalRef})
	}
	if inv.AdvanceRef != "" {
		meta = append(meta, [2]string{"Advance adjusted", inv.AdvanceRef})
	}
	mw := 82.0
	my := y
	for _, kv := range meta {
		d.SetFont(pdf.Regular, 8.2)
		d.TextColor(pdf.Text2)
		d.Text(x0+W-mw, my, 30, 4.4, kv[0], "L")
		d.SetFont(pdf.SemiBold, 8.2)
		d.TextColor(pdf.Ink)
		d.Text(x0+W-mw+30, my, mw-30, 4.4, kv[1], "R")
		my += 4.6
	}
	d.SetFont(pdf.Title, 15)
	d.TextColor(pdf.Navy)
	d.Text(x0, y, W-mw-4, 7, v.Title, "L")
	if copyLabel != "" {
		d.SetFont(pdf.Bold, 6.8)
		lw := d.StringWidth(copyLabel) + 5
		d.DrawColor(pdf.Text2)
		d.Box(x0, y+8.6, lw, 4.6, 1.2, "D")
		d.TextColor(pdf.Text2)
		d.Text(x0, y+8.9, lw, 4, copyLabel, "C")
	}
	y = max(my, y+15) + 2

	// Notices.
	note := func(text string, c, bg pdf.RGB, dashed bool) {
		if text == "" {
			return
		}
		d.SetFont(pdf.SemiBold, 7.8)
		ls := d.Wrap(text, W-8)
		h := float64(len(ls))*pdf.LineH(7.8) + 4
		if dashed {
			d.Dashed(x0, y, W, h, c)
		} else {
			d.FillColor(bg)
			d.DrawColor(c)
			d.Box(x0, y, W, h, 1.4, "FD")
		}
		d.TextColor(c)
		ty := y + 2
		for _, l := range ls {
			d.Text(x0+4, ty, W-8, pdf.LineH(7.8), l, "L")
			ty += pdf.LineH(7.8)
		}
		y += h + 2.5
	}
	note(v.AdvanceNote, pdf.Text2, pdf.Surface2, true)
	note(v.Notice, pdf.Red, pdf.RedSoft, false)
	note(v.OfflineIssued, pdf.Text2, pdf.Surface2, true)

	// Buyer and supply details.
	half := (W - 4) / 2
	buyer := []kvLine{{"", inv.BuyerName, true}, {"", inv.BuyerAddress, false}, {"", inv.BuyerProvince, false},
		{"NTN/CNIC", dash(inv.BuyerNTNCNIC), false}, {"Status", string(inv.BuyerRegistrationType), false}}
	software := v.ProductName
	if v.SoftwareRegNo != "" {
		software += " · Reg. No. " + v.SoftwareRegNo
	}
	supply := []kvLine{{"Origination of supply", inv.SellerProvince, false}, {"Destination of supply", inv.BuyerProvince, false},
		{"Document type", string(inv.DocType), false}, {"Invoicing software", software, false}}
	if inv.WithholdingMode != "" {
		supply = append(supply, kvLine{"", "Buyer is a sales tax withholding agent", true})
	}
	if inv.ScenarioID != "" {
		supply = append(supply, kvLine{"Sandbox scenario", inv.ScenarioID, false})
	}
	h1 := partyHeight(d, buyer, half)
	h2 := partyHeight(d, supply, half)
	ph := max(h1, h2)
	party(d, x0, y, half, ph, "Buyer", buyer)
	party(d, x0+half+4, y, half, ph, "Supply details", supply)
	d.SetY(y + ph + 4)

	// Lines.
	g := pdf.Grid{Size: 7.8}
	add := func(title string, right bool) { g.Cols = append(g.Cols, pdf.Col{Title: title, Right: right}) }
	add("#", true)
	g.Cols = append(g.Cols, pdf.Col{Title: "Description / HS code", Sub: true, Min: 46})
	add("Qty", true)
	add("UoM", false)
	if v.Settings.ShowUnitPrice {
		add("Unit price", true)
	}
	if v.Cols.Discount {
		add("Discount", true)
	}
	if v.Cols.FEDInValue {
		add("Value excl. ST (incl. FED)", true)
	} else {
		add("Value excl. ST", true)
	}
	if v.Cols.Retail {
		add("Retail price excl. ST", true)
	}
	add("Rate", false)
	add("Sales tax", true)
	if v.Cols.Further {
		add("Further tax", true)
	}
	if v.Cols.Extra {
		add("Extra tax", true)
	}
	if v.Cols.FED {
		add("FED", true)
	}
	add("Total", true)
	for _, l := range v.Lines {
		desc := l.Description + "\nHS " + l.HSCode + " · " + l.SaleType
		if l.SRO != "" {
			desc += " · " + l.SRO
		}
		if l.FEDNote != "" {
			desc += "\n" + l.FEDNote
		}
		row := []string{fmt.Sprint(l.No), desc, l.Qty, l.UoM}
		if v.Settings.ShowUnitPrice {
			row = append(row, l.UnitPrice)
		}
		if v.Cols.Discount {
			row = append(row, l.Discount)
		}
		row = append(row, l.Value)
		if v.Cols.Retail {
			row = append(row, l.Retail)
		}
		row = append(row, l.Rate, l.SalesTax)
		if v.Cols.Further {
			row = append(row, l.FurtherTax)
		}
		if v.Cols.Extra {
			row = append(row, l.ExtraTax)
		}
		if v.Cols.FED {
			row = append(row, l.FED)
		}
		row = append(row, l.Total)
		g.Rows = append(g.Rows, row)
	}
	d.Table(g)

	// Totals on the right; rate summary, amount in words and notes on the
	// left.
	totals := [][3]string{{"Value excluding sales tax", v.T.Value, ""}}
	if v.Cols.Retail {
		totals = append(totals, [3]string{"Retail price excl. sales tax (Third Schedule)", v.T.Retail, ""})
	}
	fedLabel := "Federal excise duty in sales tax mode"
	if v.Cols.FEDInValue {
		fedLabel = "Federal excise duty (included in the value above)"
	}
	totals = append(totals, [3]string{"Sales tax", v.T.SalesTax, ""}, [3]string{"Further tax", v.T.FurtherTax, ""},
		[3]string{"Extra tax", v.T.ExtraTax, ""}, [3]string{fedLabel, v.T.FED, ""}, [3]string{"Total discount", v.T.Discount, ""},
		[3]string{"Value inclusive of taxes", v.T.Total, "grand"}, [3]string{"Less: sales tax withheld at source", "(" + v.T.Withheld + ")", ""})
	if v.Cols.Withheld {
		totals = append(totals, [3]string{"Amount payable", v.T.Payable, "grand"})
	}
	tw := 84.0
	need := float64(len(totals))*5.6 + 8
	d.EnsureSpace(need)
	y = d.Y()
	tx := x0 + W - tw
	ty := y
	for _, t := range totals {
		grand := t[2] == "grand"
		h := 5.4
		if grand {
			h = 7
			d.FillColor(pdf.Soft)
			d.F().Rect(tx, ty, tw, h, "F")
			d.Rule(tx, tx+tw, ty, 0.5, pdf.Navy)
			d.Rule(tx, tx+tw, ty+h, 0.5, pdf.Navy)
			d.SetFont(pdf.Bold, 9)
			d.TextColor(pdf.Ink)
		} else {
			d.SetFont(pdf.Regular, 7.8)
			d.TextColor(pdf.Text2)
		}
		d.Text(tx+2, ty+(h-4)/2, tw-30, 4, t[0], "L")
		if !grand {
			d.SetFont(pdf.SemiBold, 7.8)
			d.TextColor(pdf.Ink)
			d.Rule(tx, tx+tw, ty+h, 0.15, pdf.Border)
		}
		d.Text(tx+tw-34, ty+(h-4)/2, 32, 4, t[1], "R")
		ty += h
	}
	lw := W - tw - 6
	ly := y
	if len(v.Rates) > 1 {
		rg := pdf.Grid{Size: 7, Light: true, X: x0, W: lw, Cols: []pdf.Col{{Title: "Sale type"}, {Title: "Rate"}, {Title: "Value", Right: true}, {Title: "Sales tax", Right: true}}}
		for _, r := range v.Rates {
			rg.Rows = append(rg.Rows, []string{r.SaleType, r.Rate, r.Value, r.SalesTax})
		}
		d.SetY(ly)
		d.Table(rg)
		ly = d.Y()
	}
	if v.Settings.ShowAmountWords && v.AmountWords != "" {
		d.SetFont(pdf.SemiBold, 8.2)
		d.TextColor(pdf.Ink)
		for _, l := range d.Wrap(v.AmountWords, lw) {
			d.Text(x0, ly, lw, pdf.LineH(8.2), l, "L")
			ly += pdf.LineH(8.2)
		}
		ly += 2
	}
	if inv.Notes != "" {
		d.SetFont(pdf.Regular, 7.8)
		d.TextColor(pdf.Text2)
		for _, l := range d.Wrap("Notes: "+inv.Notes, lw) {
			d.Text(x0, ly, lw, pdf.LineH(7.8), l, "L")
			ly += pdf.LineH(7.8)
		}
	}
	d.SetY(max(ty, ly) + 5)

	// Terms, footer, signature.
	small := func(text string, c pdf.RGB) {
		if text == "" {
			return
		}
		d.SetFont(pdf.Regular, 7.2)
		d.TextColor(c)
		for _, l := range d.Wrap(text, W) {
			d.EnsureSpace(pdf.LineH(7.2))
			d.Text(x0, d.Y(), W, pdf.LineH(7.2), l, "L")
			d.SetY(d.Y() + pdf.LineH(7.2))
		}
		d.SetY(d.Y() + 1.5)
	}
	small(v.Settings.TermsText, pdf.Text2)
	if v.Settings.ShowSignature {
		d.EnsureSpace(20)
		sy := d.Y() + 12
		d.Rule(x0+W-62, x0+W, sy, 0.3, pdf.Ink)
		d.SetFont(pdf.Regular, 7.6)
		d.TextColor(pdf.Text2)
		d.Text(x0+W-62, sy+1, 62, 4, "Authorised signature", "C")
		d.SetY(sy + 7)
	}
	small(v.Settings.FooterText, pdf.Text2)
	var trail []string
	if v.Signature != "" {
		s := "Digital signature: " + v.Signature
		if v.SigningKey != "" {
			s += " (Ed25519 key " + v.SigningKey + ")"
		}
		trail = append(trail, s)
	}
	if v.ShortSeal != "" {
		trail = append(trail, "Integrity seal: "+v.ShortSeal)
	}
	trail = append(trail, "Printed "+v.PrintedAt, v.ProductName+" by "+v.Developer)
	small(strings.Join(trail, " · "), pdf.Text3)
}

type kvLine struct {
	k, v string
	bold bool
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// partyHeight measures a party box exactly as party draws it.
func partyHeight(d *pdf.Doc, rows []kvLine, w float64) float64 {
	h := 7.6
	for _, r := range rows {
		if r.v == "" {
			continue
		}
		if r.k != "" {
			d.SetFont(pdf.Regular, 8.2)
			kw := d.StringWidth(r.k+": ") + 0.4
			d.SetFont(pdf.SemiBold, 8.2)
			h += float64(len(d.Wrap(r.v, w-8-kw))) * pdf.LineH(8.2)
			continue
		}
		if r.bold {
			d.SetFont(pdf.Bold, 9)
		} else {
			d.SetFont(pdf.Regular, 8.2)
		}
		h += float64(len(d.Wrap(r.v, w-8))) * pdf.LineH(8.2)
	}
	return h + 3
}

func party(d *pdf.Doc, x, y, w, h float64, title string, rows []kvLine) {
	d.FillColor(pdf.Surface2)
	d.DrawColor(pdf.Border2)
	d.Box(x, y, w, h, 2, "FD")
	d.SetFont(pdf.SemiBold, 6.8)
	d.TextColor(pdf.Text3)
	d.Text(x+4, y+3, w-8, 3.4, strings.ToUpper(title), "L")
	ty := y + 7.6
	for _, r := range rows {
		if r.v == "" {
			continue
		}
		if r.k != "" {
			d.SetFont(pdf.Regular, 8.2)
			d.TextColor(pdf.Text2)
			kw := d.StringWidth(r.k+": ") + 0.4
			d.Text(x+4, ty, kw, pdf.LineH(8.2), r.k+": ", "L")
			d.SetFont(pdf.SemiBold, 8.2)
			d.TextColor(pdf.Ink)
			for _, l := range d.Wrap(r.v, w-8-kw) {
				d.Text(x+4+kw, ty, w-8-kw, pdf.LineH(8.2), l, "L")
				ty += pdf.LineH(8.2)
			}
			continue
		}
		if r.bold {
			d.SetFont(pdf.Bold, 9)
			d.TextColor(pdf.Ink)
		} else {
			d.SetFont(pdf.Regular, 8.2)
			d.TextColor(pdf.Text2)
		}
		for _, l := range d.Wrap(r.v, w-8) {
			d.Text(x+4, ty, w-8, pdf.LineH(8.2), l, "L")
			ty += pdf.LineH(8.2)
		}
	}
}

// splitEvery breaks a long unspaced value (an FBR invoice number) into
// lines that fit width w at the current font.
func splitEvery(s string, w float64, d *pdf.Doc) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if d.StringWidth(cur+string(r)) > w-1 && cur != "" {
			out = append(out, cur)
			cur = ""
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
