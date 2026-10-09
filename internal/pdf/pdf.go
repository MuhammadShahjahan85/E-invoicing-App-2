// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package pdf renders reports, invoices and the monthly report pack as PDF
// documents in the Veridian house style: Inter for text, Plus Jakarta Sans
// for titles, a deep navy and viridian green palette, and page furniture
// (brand, company, page numbers) on every page.
package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/go-pdf/fpdf"
)

// The fonts are static instances of the Inter and Plus Jakarta Sans
// variable fonts (Latin and Latin Extended), licensed under the SIL Open
// Font License 1.1 (see fonts/*-OFL.txt).
var (
	//go:embed fonts/Inter-Regular.ttf
	interRegular []byte
	//go:embed fonts/Inter-SemiBold.ttf
	interSemiBold []byte
	//go:embed fonts/Inter-Bold.ttf
	interBold []byte
	//go:embed fonts/PlusJakartaSans-ExtraBold.ttf
	jakartaExtraBold []byte
)

// Font families registered in every document.
const (
	fText   = "inter"   // Inter Regular ("") and Bold ("B")
	fSemi   = "intersb" // Inter SemiBold
	fTitle  = "jakarta" // Plus Jakarta Sans ExtraBold
	ptPerMM = 72 / 25.4
)

// RGB is a colour.
type RGB struct{ R, G, B int }

// The palette follows the app's design tokens.
var (
	Navy      = RGB{35, 34, 94}
	NavyDeep  = RGB{26, 25, 72}
	NavyLight = RGB{59, 57, 166}
	Ink       = RGB{23, 23, 61}
	Text2     = RGB{86, 90, 124}
	Text3     = RGB{138, 142, 174}
	Border    = RGB{229, 231, 241}
	Border2   = RGB{210, 214, 230}
	Surface2  = RGB{248, 249, 253}
	Surface3  = RGB{239, 241, 248}
	Soft      = RGB{237, 237, 251}
	Green     = RGB{22, 163, 74}
	GreenDark = RGB{15, 122, 55}
	GreenSoft = RGB{230, 247, 236}
	Gold      = RGB{232, 178, 58}
	Amber     = RGB{161, 92, 0}
	AmberSoft = RGB{255, 245, 223}
	Red       = RGB{199, 39, 29}
	RedSoft   = RGB{253, 236, 235}
	Purple    = RGB{106, 69, 209}
	White     = RGB{255, 255, 255}
)

// Meta describes the document and the company it belongs to.
type Meta struct {
	Title       string // document title (also the PDF's metadata title)
	Subject     string
	Company     string // seller name, shown at the top right of each page
	CompanyLine string // e.g. "NTN 1234567 · STRN 32-77-8761-234-56"
	// Environment is "production", "sandbox" or "simulator"; anything but
	// production is watermarked so test figures cannot pass for real ones.
	Environment string
	Generated   time.Time
	Author      string // the user who produced the document
	Product     string // e.g. "Veridian E-invoicing Pakistan"
	Developer   string // e.g. "Veridian Partners Consultancy Private Limited"
	// Plain omits the report header (used for invoices, which carry the
	// seller's own letterhead).
	Plain bool
	// Watermark overrides the environment watermark (e.g. "CANCELLED").
	Watermark string
}

// Doc is a PDF being built.
type Doc struct {
	f              *fpdf.Fpdf
	m              Meta
	pw, ph         float64
	lm, rm, tm, bm float64
	// repeat redraws a table's header row after a page break.
	repeat func()
}

// PKT is Pakistan Standard Time.
var PKT = time.FixedZone("PKT", 5*3600)

// New starts an A4 document.
func New(landscape bool, m Meta) *Doc {
	orient := "P"
	if landscape {
		orient = "L"
	}
	f := fpdf.New(orient, "mm", "A4", "")
	f.AddUTF8FontFromBytes(fText, "", interRegular)
	f.AddUTF8FontFromBytes(fText, "B", interBold)
	f.AddUTF8FontFromBytes(fSemi, "", interSemiBold)
	f.AddUTF8FontFromBytes(fTitle, "", jakartaExtraBold)
	if m.Generated.IsZero() {
		m.Generated = time.Now()
	}
	if m.Product == "" {
		m.Product = "Veridian E-invoicing Pakistan"
	}
	d := &Doc{f: f, m: m, lm: 14, rm: 14, tm: 14, bm: 16}
	d.pw, d.ph = f.GetPageSize()
	if !m.Plain {
		d.tm = 27
	}
	f.SetMargins(d.lm, d.tm, d.rm)
	f.SetCellMargin(0)
	f.SetAutoPageBreak(false, d.bm)
	f.SetTitle(m.Title, true)
	f.SetSubject(m.Subject, true)
	f.SetAuthor(m.Author, true)
	f.SetCreator(m.Product, true)
	f.SetProducer(m.Product, true)
	f.SetCreationDate(m.Generated)
	f.AliasNbPages("{nb}")
	f.SetHeaderFuncMode(d.header, true)
	f.SetFooterFunc(d.footer)
	f.AddPage()
	return d
}

// F exposes the underlying fpdf document for custom drawing.
func (d *Doc) F() *fpdf.Fpdf { return d.f }

// Width is the usable width between the margins.
func (d *Doc) Width() float64 { return d.pw - d.lm - d.rm }

// Left is the left margin.
func (d *Doc) Left() float64 { return d.lm }

// Bottom is the lowest y content may reach on a page.
func (d *Doc) Bottom() float64 { return d.ph - d.bm }

// Bytes returns the finished PDF.
func (d *Doc) Bytes() ([]byte, error) {
	var b bytes.Buffer
	if err := d.Output(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Output writes the finished PDF.
func (d *Doc) Output(w io.Writer) error {
	if err := d.f.Error(); err != nil {
		return err
	}
	return d.f.Output(w)
}

// --- colours and fonts ---

func (d *Doc) text(c RGB) { d.f.SetTextColor(c.R, c.G, c.B) }
func (d *Doc) fill(c RGB) { d.f.SetFillColor(c.R, c.G, c.B) }
func (d *Doc) draw(c RGB) { d.f.SetDrawColor(c.R, c.G, c.B) }
func (d *Doc) font(fam, style string, size float64) {
	d.f.SetFont(fam, style, size)
}

// lineH is the line height for a font size in points.
func lineH(size float64) float64 { return size / ptPerMM * 1.32 }

// Clean makes text safe for the embedded fonts: tabs and odd spaces become
// spaces and characters the fonts do not cover are replaced, so that
// nothing is silently dropped or measured wrongly.
func Clean(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			b.WriteRune(r)
		case r == '\t' || r == '\r' || r == ' ' || r == ' ' || r == ' ':
			b.WriteByte(' ')
		case r < 0x20 || (r >= 0x7f && r < 0xa0):
			// control characters are dropped
		case covered(r):
			b.WriteRune(r)
		case unicode.Is(unicode.Mn, r):
			// unsupported combining marks are dropped
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

// covered reports whether the embedded fonts have a glyph for r (the
// Latin and Latin Extended subsets of Inter).
func covered(r rune) bool {
	for _, g := range coverage {
		if r >= g[0] && r <= g[1] {
			return true
		}
		if r < g[0] {
			return false
		}
	}
	return false
}

// coverage lists the code point ranges of the embedded fonts, sorted.
var coverage = [][2]rune{
	{0x20, 0x7e}, {0xa0, 0x2ba}, {0x2bb, 0x2c5}, {0x2c6, 0x2cc}, {0x2ce, 0x2d7}, {0x2da, 0x2da}, {0x2dc, 0x2ff},
	{0x304, 0x304}, {0x308, 0x308}, {0x329, 0x329}, {0x1d00, 0x1dbf}, {0x1e00, 0x1e9f}, {0x1ef2, 0x1eff},
	{0x2000, 0x206f}, {0x20a0, 0x20ab}, {0x20ac, 0x20ac}, {0x20ad, 0x20c0}, {0x2113, 0x2113}, {0x2122, 0x2122},
	{0x2191, 0x2191}, {0x2193, 0x2193}, {0x2212, 0x2212}, {0x2215, 0x2215}, {0x2c60, 0x2c7f}, {0xa720, 0xa7ff},
}

// --- page furniture ---

// Mark draws the Veridian logo mark (green tile, white check-mark V, gold
// seal) with its top-left corner at x, y.
func (d *Doc) Mark(x, y, s float64) {
	f := d.f
	f.ClipRoundedRect(x, y, s, s, s*0.22, false)
	f.LinearGradient(x, y, s, s, 0x2b, 0xb0, 0x83, 0x0a, 0x3a, 0x31, 0, 1, 1, 0)
	f.ClipEnd()
	p := func(px, py float64) (float64, float64) { return x + px/100*s, y + py/100*s }
	d.draw(White)
	f.SetLineWidth(s * 0.11)
	f.SetLineCapStyle("round")
	f.SetLineJoinStyle("round")
	f.MoveTo(p(26, 35))
	f.LineTo(p(46, 70))
	f.LineTo(p(72, 30))
	f.DrawPath("D")
	f.SetLineCapStyle("butt")
	f.SetLineJoinStyle("miter")
	cx, cy := p(80.5, 18.5)
	d.fill(RGB{0xe3, 0xb3, 0x41})
	f.Circle(cx, cy, s*0.065, "F")
	f.SetLineWidth(0.2)
}

// flowLine draws the navy-to-green line used under headers.
func (d *Doc) flowLine(x, y, w, h float64) {
	gap := 1.2
	segs := []struct {
		frac float64
		c    RGB
	}{{0.62, Navy}, {0.14, RGB{27, 143, 76}}, {0.12, RGB{34, 179, 94}}, {0.12, RGB{95, 217, 143}}}
	avail := w - gap*float64(len(segs)-1)
	for _, s := range segs {
		sw := avail * s.frac
		d.fill(s.c)
		d.f.Rect(x, y, sw, h, "F")
		x += sw + gap
	}
}

func (d *Doc) header() {
	f := d.f
	if d.m.Plain {
		return
	}
	y := 10.0
	d.Mark(d.lm, y, 8)
	d.font(fTitle, "", 11.5)
	d.text(Ink)
	f.SetXY(d.lm+10.5, y+0.2)
	f.CellFormat(40, 4.6, "Veridian", "", 0, "L", false, 0, "")
	d.font(fText, "B", 5.6)
	d.text(GreenDark)
	f.SetXY(d.lm+10.5, y+4.9)
	f.CellFormat(60, 3, "E-INVOICING  PAKISTAN", "", 0, "L", false, 0, "")
	// Company at the top right.
	right := d.pw - d.rm
	d.font(fSemi, "", 9)
	d.text(Ink)
	f.SetXY(right-140, y+0.2)
	f.CellFormat(140, 4.4, Clean(d.m.Company), "", 0, "R", false, 0, "")
	d.font(fText, "", 7.2)
	d.text(Text2)
	f.SetXY(right-140, y+4.7)
	f.CellFormat(140, 3.6, Clean(d.m.CompanyLine), "", 0, "R", false, 0, "")
	d.flowLine(d.lm, 21, d.Width(), 0.9)
	f.SetXY(d.lm, d.tm)
}

func (d *Doc) footer() {
	f := d.f
	// The watermark is laid over the finished page, faintly, so that
	// opaque panels and table rows cannot hide it.
	d.watermark()
	y := d.ph - 11
	d.draw(Border)
	f.SetLineWidth(0.25)
	f.Line(d.lm, y, d.pw-d.rm, y)
	d.font(fText, "", 6.8)
	d.text(Text3)
	gen := "Generated " + d.m.Generated.In(PKT).Format("02 Jan 2006, 15:04") + " PKT"
	if d.m.Author != "" {
		gen += " by " + d.m.Author
	}
	// "{nb}" is replaced by the page count when the document is closed.
	right := Clean(fmt.Sprintf("%s  ·  Page %d of {nb}", gen, f.PageNo()))
	rw := f.GetStringWidth(right) + 1
	left := d.m.Product
	if d.m.Developer != "" && f.GetStringWidth(left+" · developed by "+d.m.Developer) < d.Width()-rw-6 {
		left += " · developed by " + d.m.Developer
	}
	f.SetXY(d.lm, y+1.6)
	f.CellFormat(d.Width()-rw-4, 3.5, Clean(left), "", 0, "L", false, 0, "")
	f.SetXY(d.pw-d.rm-rw, y+1.6)
	f.CellFormat(rw, 3.5, right, "", 0, "R", false, 0, "")
}

// watermark marks every page of a document that must not pass for a real
// tax record.
func (d *Doc) watermark() {
	text, col := d.m.Watermark, Red
	if text == "" {
		switch d.m.Environment {
		case "simulator":
			text, col = "TRAINING SIMULATOR — NOT REPORTED TO FBR", Purple
		case "sandbox":
			text, col = "FBR SANDBOX — TEST DATA", Amber
		default:
			return
		}
	}
	f := d.f
	text = Clean(text)
	// Fit the text within the page width (a negative x would be taken as
	// measured from the right edge).
	size := 44.0
	d.font(fTitle, "", size)
	w := f.GetStringWidth(text)
	if limit := d.pw - 24; w > limit {
		size *= limit / w
		d.font(fTitle, "", size)
		w = f.GetStringWidth(text)
	}
	cx, cy := d.pw/2, d.ph/2
	f.SetAlpha(0.08, "Normal")
	d.text(col)
	f.TransformBegin()
	f.TransformRotate(28, cx, cy)
	f.SetXY(cx-w/2, cy-6)
	f.CellFormat(w, 12, text, "", 0, "C", false, 0, "")
	f.TransformEnd()
	f.SetAlpha(1, "Normal")
}

// --- flow helpers ---

// Y is the current vertical position.
func (d *Doc) Y() float64 { return d.f.GetY() }

// Space adds vertical space.
func (d *Doc) Space(h float64) { d.f.SetY(d.f.GetY() + h) }

// EnsureSpace starts a new page unless h millimetres remain on this one.
func (d *Doc) EnsureSpace(h float64) {
	if d.f.GetY()+h > d.Bottom() {
		d.NewPage()
	}
}

// NewPage starts a new page.
func (d *Doc) NewPage() {
	d.f.AddPage()
	d.f.SetXY(d.lm, d.tm)
}

// wrap splits text into lines that fit width w at the current font.
func (d *Doc) wrap(text string, w float64) []string {
	if text == "" {
		return []string{""}
	}
	var out []string
	for _, para := range strings.Split(text, "\n") {
		if para == "" {
			out = append(out, "")
			continue
		}
		lines := d.f.SplitText(para, w)
		if len(lines) == 0 {
			lines = []string{""}
		}
		out = append(out, lines...)
	}
	return out
}

// Heading is the title block of a document: a green eyebrow, a large
// title and a muted subtitle.
func (d *Doc) Heading(eyebrow, title, sub string) {
	f := d.f
	x := d.lm
	if eyebrow != "" {
		d.font(fText, "B", 7.4)
		d.text(GreenDark)
		f.SetX(x)
		f.CellFormat(d.Width(), 4, Clean(strings.ToUpper(eyebrow)), "", 1, "L", false, 0, "")
		d.Space(0.8)
	}
	d.font(fTitle, "", 19)
	d.text(Ink)
	for _, l := range d.wrap(Clean(title), d.Width()) {
		f.SetX(x)
		f.CellFormat(d.Width(), 8.2, l, "", 1, "L", false, 0, "")
	}
	if sub != "" {
		d.font(fText, "", 9)
		d.text(Text2)
		for _, l := range d.wrap(Clean(sub), d.Width()) {
			f.SetX(x)
			f.CellFormat(d.Width(), lineH(9), l, "", 1, "L", false, 0, "")
		}
	}
	d.Space(4)
}

// Section is a section heading with a short green accent bar.
func (d *Doc) Section(title, sub string) {
	// Keep the heading with the start of its content.
	need := 40.0
	d.EnsureSpace(need)
	f := d.f
	y := f.GetY() + 2
	d.fill(Green)
	f.RoundedRect(d.lm, y+0.6, 1.2, 4.6, 0.6, "1234", "F")
	d.font(fTitle, "", 11.5)
	d.text(Ink)
	f.SetXY(d.lm+3.6, y)
	f.CellFormat(d.Width()-3.6, 5.8, Clean(title), "", 1, "L", false, 0, "")
	if sub != "" {
		d.font(fText, "", 8)
		d.text(Text2)
		for _, l := range d.wrap(Clean(sub), d.Width()-3.6) {
			f.SetX(d.lm + 3.6)
			f.CellFormat(d.Width()-3.6, lineH(8), l, "", 1, "L", false, 0, "")
		}
	}
	d.Space(2.6)
}

// Para writes a paragraph.
func (d *Doc) Para(text string, size float64, c RGB) {
	if size == 0 {
		size = 8.6
	}
	f := d.f
	d.font(fText, "", size)
	d.text(c)
	lh := lineH(size)
	for _, l := range d.wrap(Clean(text), d.Width()) {
		d.EnsureSpace(lh)
		f.SetX(d.lm)
		f.CellFormat(d.Width(), lh, l, "", 1, "L", false, 0, "")
	}
	d.Space(1.5)
}

// Tile is a summary figure.
type Tile struct {
	Label, Value, Sub string
	Hero              bool // navy tile with white text
	Tone              string
}

// Tiles draws a row of summary tiles.
func (d *Doc) Tiles(tiles []Tile) {
	if len(tiles) == 0 {
		return
	}
	f := d.f
	gap := 3.5
	h := 21.0
	d.EnsureSpace(h + 4)
	w := (d.Width() - gap*float64(len(tiles)-1)) / float64(len(tiles))
	y := f.GetY()
	for i, t := range tiles {
		x := d.lm + float64(i)*(w+gap)
		label, value, sub := Text2, Ink, Text3
		if t.Hero {
			f.ClipRoundedRect(x, y, w, h, 2.6, false)
			f.LinearGradient(x, y, w, h, 61, 60, 168, 26, 25, 72, 1, 1, 0, 0)
			f.ClipEnd()
			label, value, sub = RGB{214, 216, 250}, White, RGB{190, 193, 236}
		} else {
			d.fill(White)
			d.draw(Border)
			f.SetLineWidth(0.3)
			f.RoundedRect(x, y, w, h, 2.6, "1234", "FD")
		}
		switch t.Tone {
		case "good":
			value = GreenDark
		case "warn":
			value = Amber
		case "bad":
			value = Red
		}
		d.font(fSemi, "", 7.2)
		d.text(label)
		f.SetXY(x+3.6, y+3)
		f.CellFormat(w-7.2, 3.8, Clean(t.Label), "", 0, "L", false, 0, "")
		size := 13.5
		d.font(fTitle, "", size)
		for size > 8 && f.GetStringWidth(t.Value) > w-7.2 {
			size -= 0.5
			d.font(fTitle, "", size)
		}
		d.text(value)
		f.SetXY(x+3.6, y+7.6)
		f.CellFormat(w-7.2, 6.4, Clean(t.Value), "", 0, "L", false, 0, "")
		d.font(fText, "", 6.8)
		d.text(sub)
		f.SetXY(x+3.6, y+15)
		f.CellFormat(w-7.2, 3.4, Clean(t.Sub), "", 0, "L", false, 0, "")
	}
	f.SetXY(d.lm, y+h+4)
}

// KV draws label/value pairs in two columns of a light panel.
func (d *Doc) KV(rows [][2]string) {
	if len(rows) == 0 {
		return
	}
	f := d.f
	lw := 46.0
	vw := d.Width() - lw - 8
	size := 8.4
	lh := lineH(size)
	type kv struct {
		k  string
		vs []string
	}
	var items []kv
	total := 4.0
	d.font(fText, "", size)
	for _, r := range rows {
		vs := d.wrap(Clean(r[1]), vw)
		items = append(items, kv{Clean(r[0]), vs})
		total += float64(len(vs))*lh + 1.6
	}
	d.EnsureSpace(total)
	y := f.GetY()
	d.fill(Surface2)
	d.draw(Border)
	f.SetLineWidth(0.25)
	f.RoundedRect(d.lm, y, d.Width(), total, 2.2, "1234", "FD")
	y += 2.4
	for _, it := range items {
		d.font(fSemi, "", size)
		d.text(Text2)
		f.SetXY(d.lm+4, y)
		f.CellFormat(lw, lh, it.k, "", 0, "L", false, 0, "")
		d.font(fText, "", size)
		d.text(Ink)
		for i, v := range it.vs {
			f.SetXY(d.lm+4+lw, y+float64(i)*lh)
			f.CellFormat(vw, lh, v, "", 0, "L", false, 0, "")
		}
		y += float64(len(it.vs))*lh + 1.6
	}
	f.SetXY(d.lm, y+3)
}

// Check is one item of a checklist.
type Check struct {
	OK            bool
	Title, Detail string
}

// Checks draws a checklist with green ticks and amber marks.
func (d *Doc) Checks(items []Check) {
	f := d.f
	tw := d.Width() - 10
	for _, c := range items {
		d.font(fText, "", 7.8)
		details := []string{}
		if c.Detail != "" {
			details = d.wrap(Clean(c.Detail), tw)
		}
		d.font(fSemi, "", 9)
		titles := d.wrap(Clean(c.Title), tw)
		h := float64(len(titles))*lineH(9) + float64(len(details))*lineH(7.8) + 3.6
		d.EnsureSpace(h)
		y := f.GetY()
		cx, cy := d.lm+3, y+3.2
		if c.OK {
			d.fill(Green)
			f.Circle(cx, cy, 2.3, "F")
			d.draw(White)
			f.SetLineWidth(0.55)
			f.SetLineCapStyle("round")
			f.SetLineJoinStyle("round")
			f.MoveTo(cx-1.1, cy+0.05)
			f.LineTo(cx-0.3, cy+0.85)
			f.LineTo(cx+1.2, cy-0.8)
			f.DrawPath("D")
		} else {
			d.fill(Gold)
			f.Circle(cx, cy, 2.3, "F")
			d.draw(White)
			f.SetLineWidth(0.6)
			f.SetLineCapStyle("round")
			f.Line(cx, cy-1.2, cx, cy+0.25)
			f.Line(cx, cy+1.15, cx, cy+1.2)
		}
		f.SetLineCapStyle("butt")
		f.SetLineJoinStyle("miter")
		f.SetLineWidth(0.2)
		yy := y + 0.6
		d.font(fSemi, "", 9)
		d.text(Ink)
		for _, l := range titles {
			f.SetXY(d.lm+8, yy)
			f.CellFormat(tw, lineH(9), l, "", 0, "L", false, 0, "")
			yy += lineH(9)
		}
		d.font(fText, "", 7.8)
		d.text(Text2)
		for _, l := range details {
			f.SetXY(d.lm+8, yy)
			f.CellFormat(tw, lineH(7.8), l, "", 0, "L", false, 0, "")
			yy += lineH(7.8)
		}
		d.draw(Border)
		f.SetLineWidth(0.2)
		f.Line(d.lm+8, y+h-0.6, d.pw-d.rm, y+h-0.6)
		f.SetXY(d.lm, y+h)
	}
	d.Space(3)
}

// SignOff draws signature lines, e.g. "Prepared by", "Reviewed by".
func (d *Doc) SignOff(labels []string) {
	if len(labels) == 0 {
		return
	}
	f := d.f
	d.EnsureSpace(24)
	d.Space(12)
	gap := 8.0
	w := (d.Width() - gap*float64(len(labels)-1)) / float64(len(labels))
	y := f.GetY()
	for i, l := range labels {
		x := d.lm + float64(i)*(w+gap)
		d.draw(Ink)
		f.SetLineWidth(0.3)
		f.Line(x, y, x+w, y)
		d.font(fSemi, "", 8)
		d.text(Ink)
		f.SetXY(x, y+1.2)
		f.CellFormat(w, 4, Clean(l), "", 0, "L", false, 0, "")
		d.font(fText, "", 7)
		d.text(Text3)
		f.SetXY(x, y+5.2)
		f.CellFormat(w, 3.4, "Name, signature and date", "", 0, "L", false, 0, "")
	}
	f.SetXY(d.lm, y+11)
}

// Tie draws the three-way tie diagram: three nodes joined by edges that
// are green when that pair agrees and amber when it does not. edges are
// top–left, top–right and left–right.
func (d *Doc) Tie(x, y, w float64, nodes [3]string, edges [3]bool) float64 {
	f := d.f
	r := w * 0.13
	top := [2]float64{x + w/2, y + r}
	left := [2]float64{x + r, y + w*0.74}
	right := [2]float64{x + w - r, y + w*0.74}
	pairs := [3][2][2]float64{{top, left}, {top, right}, {left, right}}
	for i, p := range pairs {
		if edges[i] {
			d.draw(Green)
			f.SetDashPattern(nil, 0)
		} else {
			d.draw(RGB{242, 169, 27})
			f.SetDashPattern([]float64{1.6, 1.2}, 0)
		}
		f.SetLineWidth(0.9)
		f.Line(p[0][0], p[0][1], p[1][0], p[1][1])
	}
	f.SetDashPattern(nil, 0)
	for i, c := range [][2]float64{top, left, right} {
		d.fill(Surface3)
		d.draw(Border2)
		f.SetLineWidth(0.35)
		f.Circle(c[0], c[1], r, "FD")
		size := 8.0
		if len(nodes[i]) > 5 {
			size = 6.6
		}
		d.font(fTitle, "", size)
		d.text(Ink)
		f.SetXY(c[0]-r, c[1]-2.2)
		f.CellFormat(2*r, 4.4, Clean(nodes[i]), "", 0, "C", false, 0, "")
	}
	for i, p := range pairs {
		mx, my := (p[0][0]+p[1][0])/2, (p[0][1]+p[1][1])/2
		mr := r * 0.36
		d.draw(White)
		f.SetLineWidth(0.8)
		if edges[i] {
			d.fill(Green)
		} else {
			d.fill(RGB{242, 169, 27})
		}
		f.Circle(mx, my, mr, "FD")
		f.SetLineCapStyle("round")
		f.SetLineJoinStyle("round")
		f.SetLineWidth(0.55)
		if edges[i] {
			f.MoveTo(mx-mr*0.45, my+mr*0.02)
			f.LineTo(mx-mr*0.12, my+mr*0.36)
			f.LineTo(mx+mr*0.5, my-mr*0.34)
			f.DrawPath("D")
		} else {
			f.Line(mx, my-mr*0.5, mx, my+mr*0.08)
			f.Line(mx, my+mr*0.45, mx, my+mr*0.5)
		}
		f.SetLineCapStyle("butt")
		f.SetLineJoinStyle("miter")
	}
	f.SetLineWidth(0.2)
	return w*0.74 + r
}

// TieRow is one side of the tie, listed beside the diagram.
type TieRow struct {
	OK                   bool
	Title, Detail, Value string
}

// TiePanel draws the tie diagram with its three sides listed beside it.
func (d *Doc) TiePanel(nodes [3]string, rows [3]TieRow) {
	f := d.f
	h := 62.0
	d.EnsureSpace(h + 4)
	y := f.GetY()
	d.fill(White)
	d.draw(Border)
	f.SetLineWidth(0.3)
	f.RoundedRect(d.lm, y, d.Width(), h, 2.6, "1234", "FD")
	d.Tie(d.lm+6, y+4, 62, nodes, [3]bool{rows[0].OK, rows[1].OK, rows[2].OK})
	lx := d.lm + 80
	lw := d.Width() - 80 - 5
	ry := y + 6.5
	for i, r := range rows {
		cx, cy := lx+2.6, ry+3.2
		d.draw(Green)
		if !r.OK {
			d.draw(Gold)
		}
		f.SetLineWidth(0.4)
		f.Circle(cx, cy, 2.6, "D")
		f.SetLineCapStyle("round")
		f.SetLineJoinStyle("round")
		if r.OK {
			f.MoveTo(cx-1.1, cy+0.05)
			f.LineTo(cx-0.3, cy+0.85)
			f.LineTo(cx+1.2, cy-0.8)
			f.DrawPath("D")
		} else {
			f.Line(cx, cy-1.3, cx, cy+0.3)
			f.Line(cx, cy+1.2, cx, cy+1.25)
		}
		f.SetLineCapStyle("butt")
		f.SetLineJoinStyle("miter")
		d.font(fSemi, "", 9)
		d.text(Ink)
		f.SetXY(lx+8, ry)
		f.CellFormat(lw-45, 4.4, Clean(r.Title), "", 0, "L", false, 0, "")
		d.font(fText, "", 7.6)
		d.text(Text2)
		f.SetXY(lx+8, ry+4.6)
		f.CellFormat(lw-45, 3.6, Clean(r.Detail), "", 0, "L", false, 0, "")
		d.font(fTitle, "", 10)
		d.text(Ink)
		if !r.OK {
			d.text(Amber)
		}
		f.SetXY(lx+lw-40, ry+1.2)
		f.CellFormat(40, 5, Clean(r.Value), "", 0, "R", false, 0, "")
		if i < 2 {
			d.draw(Border)
			f.SetLineWidth(0.2)
			f.Line(lx, ry+11.6, lx+lw, ry+11.6)
		}
		ry += 17
	}
	f.SetLineWidth(0.2)
	f.SetXY(d.lm, y+h+5)
}

// Font kinds for SetFont.
const (
	Regular  = "regular"
	Bold     = "bold"
	SemiBold = "semibold"
	Title    = "title"
)

// SetFont selects one of the embedded fonts.
func (d *Doc) SetFont(kind string, size float64) {
	switch kind {
	case Bold:
		d.font(fText, "B", size)
	case SemiBold:
		d.font(fSemi, "", size)
	case Title:
		d.font(fTitle, "", size)
	default:
		d.font(fText, "", size)
	}
}

// TextColor, FillColor and DrawColor set the current colours.
func (d *Doc) TextColor(c RGB) { d.text(c) }
func (d *Doc) FillColor(c RGB) { d.fill(c) }
func (d *Doc) DrawColor(c RGB) { d.draw(c) }

// Wrap splits text into lines that fit width w at the current font.
func (d *Doc) Wrap(text string, w float64) []string { return d.wrap(Clean(text), w) }

// LineH is the line height for a font size in points.
func LineH(size float64) float64 { return lineH(size) }

// Text writes one line of text in a box of width w at x, y.
func (d *Doc) Text(x, y, w, h float64, text, align string) {
	d.f.SetXY(x, y)
	d.f.CellFormat(w, h, Clean(text), "", 0, align, false, 0, "")
}

// StringWidth measures text at the current font.
func (d *Doc) StringWidth(text string) float64 { return d.f.GetStringWidth(Clean(text)) }

// Image places a PNG, JPEG or GIF image scaled to fit within maxW × maxH
// with its top-left corner at x, y (right-aligned to x+maxW when right is
// set). It returns the size drawn, or zeros when the image cannot be used.
func (d *Doc) Image(name string, data []byte, mime string, x, y, maxW, maxH float64, right bool) (float64, float64) {
	typ := ""
	switch mime {
	case "image/png":
		typ = "PNG"
	case "image/jpeg", "image/jpg":
		typ = "JPG"
	case "image/gif":
		typ = "GIF"
	}
	if typ == "" || len(data) == 0 {
		return 0, 0
	}
	opt := fpdf.ImageOptions{ImageType: typ}
	info := d.f.RegisterImageOptionsReader(name, opt, bytes.NewReader(data))
	if info == nil || d.f.Err() || info.Width() <= 0 || info.Height() <= 0 {
		// An unreadable image must not spoil the document.
		d.f.ClearError()
		return 0, 0
	}
	w, h := maxW, maxW*info.Height()/info.Width()
	if h > maxH {
		h, w = maxH, maxH*info.Width()/info.Height()
	}
	if right {
		x += maxW - w
	}
	d.f.ImageOptions(name, x, y, w, h, false, opt, 0, "")
	return w, h
}

// QR draws a square QR code of side s from a module bitmap (true = dark),
// merging runs of dark modules so viewers show no hairlines.
func (d *Doc) QR(bitmap [][]bool, x, y, s float64) {
	n := len(bitmap)
	if n == 0 {
		return
	}
	m := s / float64(n)
	d.fill(White)
	d.f.Rect(x, y, s, s, "F")
	d.fill(RGB{0, 0, 0})
	for r, row := range bitmap {
		for c := 0; c < len(row); {
			if !row[c] {
				c++
				continue
			}
			start := c
			for c < len(row) && row[c] {
				c++
			}
			d.f.Rect(x+float64(start)*m, y+float64(r)*m, float64(c-start)*m+0.01, m+0.01, "F")
		}
	}
}

// Box draws a rounded panel; style is "D", "F" or "FD".
func (d *Doc) Box(x, y, w, h, r float64, style string) {
	d.f.SetLineWidth(0.3)
	d.f.RoundedRect(x, y, w, h, r, "1234", style)
}

// Rule draws a horizontal line.
func (d *Doc) Rule(x1, x2, y, width float64, c RGB) {
	d.draw(c)
	d.f.SetLineWidth(width)
	d.f.Line(x1, y, x2, y)
	d.f.SetLineWidth(0.2)
}

// Dashed draws a dashed rounded panel outline.
func (d *Doc) Dashed(x, y, w, h float64, c RGB) {
	d.draw(c)
	d.f.SetLineWidth(0.3)
	d.f.SetDashPattern([]float64{1.2, 0.9}, 0)
	d.f.RoundedRect(x, y, w, h, 1.4, "1234", "D")
	d.f.SetDashPattern(nil, 0)
	d.f.SetLineWidth(0.2)
}

// SetY moves to a vertical position at the left margin.
func (d *Doc) SetY(y float64) { d.f.SetXY(d.lm, y) }
