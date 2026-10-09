// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package pdf

import (
	"math"
	"strings"
)

// Col describes a table column.
type Col struct {
	Title string
	Right bool    // right-aligned (amounts and counts never wrap)
	Min   float64 // minimum width in mm (0: automatic)
	Max   float64 // maximum width in mm before text wraps (0: no limit)
	// Sub renders the lines after the first of each cell (separated by
	// "\n") smaller and muted, e.g. HS code and sale type under an item's
	// description.
	Sub bool
}

// Grid is a table of preformatted cells.
type Grid struct {
	Cols   []Col
	Rows   [][]string
	Totals []string // optional totals row; same length as Cols
	// Size is the body font size in points (default 8, reduced
	// automatically to fit wide tables).
	Size float64
	// Light uses a pale header row instead of the navy one.
	Light bool
	// X and W place the table (default: between the margins).
	X, W float64
}

const (
	padX = 1.7
	padY = 1.35
)

// cellLines splits a cell into display lines: the first paragraph at the
// body size, further paragraphs (for Sub columns) at the small size.
type cellLine struct {
	text  string
	small bool
}

func (d *Doc) layoutCell(text string, w float64, col Col, size float64) ([]cellLine, float64) {
	text = Clean(text)
	if col.Right {
		d.font(fText, "", size)
		return []cellLine{{text: text}}, lineH(size)
	}
	paras := []string{text}
	if col.Sub {
		paras = strings.Split(text, "\n")
	}
	var out []cellLine
	h := 0.0
	for i, p := range paras {
		small := col.Sub && i > 0
		sz := size
		if small {
			sz = size - 1.4
		}
		d.font(fText, "", sz)
		for _, l := range d.wrap(p, w) {
			out = append(out, cellLine{text: l, small: small})
			h += lineH(sz)
		}
	}
	return out, h
}

// widths works out column widths for the usable width W at font size size.
// It reports false when the columns cannot fit at this size.
func (d *Doc) widths(g Grid, W, size float64) ([]float64, bool) {
	n := len(g.Cols)
	nat := make([]float64, n)
	minw := make([]float64, n)
	hsize := size - 1
	for i, c := range g.Cols {
		d.font(fSemi, "", hsize)
		head := strings.ToUpper(c.Title)
		longestWord := 0.0
		for _, w := range strings.Fields(head) {
			longestWord = math.Max(longestWord, d.f.GetStringWidth(w))
		}
		headW := d.f.GetStringWidth(head)
		d.font(fText, "", size)
		body := 0.0
		bodyWord := 0.0
		rows := g.Rows
		if g.Totals != nil {
			rows = append(append(make([][]string, 0, len(g.Rows)+1), g.Rows...), g.Totals)
		}
		for _, r := range rows {
			if i >= len(r) {
				continue
			}
			for j, p := range strings.Split(Clean(r[i]), "\n") {
				if j > 0 && c.Sub {
					d.font(fText, "", size-1.4)
				}
				body = math.Max(body, d.f.GetStringWidth(p))
				for _, w := range strings.Fields(p) {
					bodyWord = math.Max(bodyWord, d.f.GetStringWidth(w))
				}
				d.font(fText, "", size)
			}
		}
		nat[i] = math.Max(body, headW) + 2*padX + 0.4
		if c.Right {
			minw[i] = math.Max(body, longestWord) + 2*padX + 0.4
		} else {
			minw[i] = math.Max(math.Min(bodyWord, 36), longestWord) + 2*padX + 0.4
			minw[i] = math.Max(minw[i], 12)
		}
		if c.Min > 0 {
			minw[i] = math.Max(minw[i], c.Min)
			nat[i] = math.Max(nat[i], c.Min)
		}
		if c.Max > 0 && nat[i] > c.Max {
			nat[i] = math.Max(c.Max, minw[i])
		}
		nat[i] = math.Max(nat[i], minw[i])
	}
	sum := func(v []float64) float64 {
		s := 0.0
		for _, x := range v {
			s += x
		}
		return s
	}
	out := append([]float64(nil), nat...)
	total := sum(out)
	if total <= W {
		// Share the spare width among text columns (or all if none).
		extra := W - total
		var flex []int
		for i, c := range g.Cols {
			if !c.Right {
				flex = append(flex, i)
			}
		}
		if len(flex) == 0 {
			for i := range g.Cols {
				flex = append(flex, i)
			}
		}
		base := 0.0
		for _, i := range flex {
			base += out[i]
		}
		for _, i := range flex {
			out[i] += extra * out[i] / base
		}
		return out, true
	}
	if sum(minw) > W {
		return nil, false
	}
	// Shrink in two passes so that text keeps as much room as possible:
	// first let the headings of amount columns wrap, then shrink the text
	// columns towards their minimum, proportionally to how much each can
	// give.
	over := total - W
	for _, pass := range []bool{true, false} {
		slack := 0.0
		for i, c := range g.Cols {
			if c.Right == pass {
				slack += out[i] - minw[i]
			}
		}
		if slack <= 0 {
			continue
		}
		take := math.Min(over, slack)
		for i, c := range g.Cols {
			if c.Right == pass {
				out[i] -= take * (out[i] - minw[i]) / slack
			}
		}
		over -= take
		if over <= 0.001 {
			break
		}
	}
	return out, true
}

// Table draws a data table: a navy header row repeated on every page,
// zebra rows and an optional totals row.
func (d *Doc) Table(g Grid) {
	if len(g.Cols) == 0 {
		return
	}
	f := d.f
	W := d.Width()
	if g.W > 0 {
		W = g.W
	}
	left := d.lm
	if g.X > 0 {
		left = g.X
	}
	size := g.Size
	if size == 0 {
		size = 8
	}
	var ws []float64
	for {
		var ok bool
		ws, ok = d.widths(g, W, size)
		if ok || size <= 5.6 {
			break
		}
		size -= 0.4
	}
	if ws == nil {
		// Even the smallest size does not fit: share the width equally.
		ws = make([]float64, len(g.Cols))
		for i := range ws {
			ws[i] = W / float64(len(g.Cols))
		}
	}
	hsize := size - 1

	drawHeader := func() {
		d.font(fSemi, "", hsize)
		lines := make([][]string, len(g.Cols))
		maxl := 1
		for i, c := range g.Cols {
			lines[i] = d.wrap(strings.ToUpper(Clean(c.Title)), ws[i]-2*padX)
			maxl = max(maxl, len(lines[i]))
		}
		lh := lineH(hsize)
		h := float64(maxl)*lh + 2*padY + 0.6
		y := f.GetY()
		if g.Light {
			d.fill(Surface3)
		} else {
			d.fill(Navy)
		}
		f.RoundedRect(left, y, W, h, 1.6, "12", "F")
		x := left
		for i, c := range g.Cols {
			align := "L"
			if c.Right {
				align = "R"
			}
			if g.Light {
				d.text(Text2)
			} else {
				d.text(White)
			}
			// Bottom-align the header text.
			yy := y + h - padY - float64(len(lines[i]))*lh
			for _, l := range lines[i] {
				f.SetXY(x+padX, yy)
				f.CellFormat(ws[i]-2*padX, lh, l, "", 0, align, false, 0, "")
				yy += lh
			}
			x += ws[i]
		}
		f.SetXY(left, y+h)
	}

	row := func(cells []string, zebra, totals bool) {
		type laid struct {
			lines []cellLine
			h     float64
		}
		cs := make([]laid, len(g.Cols))
		h := 0.0
		for i, c := range g.Cols {
			txt := ""
			if i < len(cells) {
				txt = cells[i]
			}
			ls, hh := d.layoutCell(txt, ws[i]-2*padX, c, size)
			cs[i] = laid{ls, hh}
			h = math.Max(h, hh)
		}
		h += 2 * padY
		if f.GetY()+h > d.Bottom() {
			d.NewPage()
			drawHeader()
		}
		y := f.GetY()
		switch {
		case totals:
			d.fill(Soft)
			f.Rect(left, y, W, h, "F")
			d.draw(Navy)
			f.SetLineWidth(0.4)
			f.Line(left, y, left+W, y)
		case zebra:
			d.fill(Surface2)
			f.Rect(left, y, W, h, "F")
		}
		x := left
		for i, c := range g.Cols {
			align := "L"
			if c.Right {
				align = "R"
			}
			yy := y + padY
			for _, l := range cs[i].lines {
				sz := size
				switch {
				case totals:
					d.font(fText, "B", sz)
					d.text(Ink)
				case l.small:
					sz = size - 1.4
					d.font(fText, "", sz)
					d.text(Text2)
				default:
					d.font(fText, "", sz)
					d.text(Ink)
				}
				f.SetXY(x+padX, yy)
				f.CellFormat(ws[i]-2*padX, lineH(sz), l.text, "", 0, align, false, 0, "")
				yy += lineH(sz)
			}
			x += ws[i]
		}
		if !totals {
			d.draw(Border)
			f.SetLineWidth(0.2)
			f.Line(left, y+h, left+W, y+h)
		}
		f.SetXY(left, y+h)
	}

	// Keep the header with at least one row.
	d.EnsureSpace(16)
	drawHeader()
	for i, r := range g.Rows {
		row(r, i%2 == 1, false)
	}
	if len(g.Rows) == 0 {
		d.font(fText, "", size)
		d.text(Text3)
		f.SetX(left)
		f.CellFormat(W, 9, "Nothing to show for this selection.", "", 1, "C", false, 0, "")
	}
	if g.Totals != nil && len(g.Rows) > 0 {
		row(g.Totals, false, true)
	}
	d.Space(5)
}
