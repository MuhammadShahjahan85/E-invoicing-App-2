// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package dataimport

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/ledongthuc/pdf"
)

// A PDF has no tables, only text placed on pages. The reader rebuilds the
// table: characters on the same baseline form a line and close characters
// form words; the columns are the bands of the page that the words of the
// table's lines occupy, separated by the white space that runs down the
// table; the heading line names them.

type pdfWord struct {
	x0, x1 float64
	s      string
}

type pdfLine struct {
	page  int
	y     float64
	h     float64
	words []pdfWord
}

func (l pdfLine) text() string {
	parts := make([]string, len(l.words))
	for i, w := range l.words {
		parts[i] = w.s
	}
	return strings.Join(parts, " ")
}

func readPDF(data []byte) (*Book, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("cannot read the PDF (%v); if it is password-protected, remove the password and try again", err)
	}
	b := &Book{Format: "PDF document", Kind: "pdf"}
	var lines []pdfLine
	for i := 1; i <= min(r.NumPage(), 300); i++ {
		lines = append(lines, pageLines(r.Page(i), i)...)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("this PDF has no text that can be read (it may be a scanned image); export the report from your software as Excel or CSV instead")
	}
	b.Sheets = []Sheet{pdfTable(lines)}
	b.Notes = append(b.Notes, "The table was rebuilt from the position of the text in the PDF; check the column matching and the preview carefully.")
	return finish(b)
}

func pageLines(p pdf.Page, n int) (out []pdfLine) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	if p.V.IsNull() {
		return nil
	}
	texts := p.Content().Text
	if len(texts) == 0 {
		return nil
	}
	// Watermarks and titles are set much larger than the table: keep the
	// characters near the page's usual size.
	sizes := make([]float64, 0, len(texts))
	for _, t := range texts {
		if strings.TrimSpace(t.S) != "" {
			sizes = append(sizes, t.FontSize)
		}
	}
	sort.Float64s(sizes)
	if len(sizes) == 0 {
		return nil
	}
	median := sizes[len(sizes)/2]
	kept := texts[:0]
	for _, t := range texts {
		if t.S != "" && t.FontSize <= median*1.8 {
			kept = append(kept, t)
		}
	}
	texts = kept
	sort.SliceStable(texts, func(i, j int) bool {
		if math.Abs(texts[i].Y-texts[j].Y) > 0.01 {
			return texts[i].Y > texts[j].Y
		}
		return texts[i].X < texts[j].X
	})
	var groups [][]pdf.Text
	for _, t := range texts {
		if k := len(groups); k > 0 {
			ref := groups[k-1][0]
			if math.Abs(ref.Y-t.Y) <= math.Max(1.2, 0.3*math.Max(ref.FontSize, t.FontSize)) {
				groups[k-1] = append(groups[k-1], t)
				continue
			}
		}
		groups = append(groups, []pdf.Text{t})
	}
	for _, g := range groups {
		sort.SliceStable(g, func(i, j int) bool { return g[i].X < g[j].X })
		line := pdfLine{page: n, y: g[0].Y}
		var cur *pdfWord
		prevEnd, prevX, effX := 0.0, math.Inf(-1), 0.0
		for _, t := range g {
			fs := math.Max(t.FontSize, 4)
			line.h = math.Max(line.h, fs)
			// Some fonts give no glyph widths: all characters of a string
			// then report the string's start. Estimate their places.
			w := t.W
			if w <= 0 {
				w = 0.52 * fs
			}
			x := t.X
			if t.W <= 0 && math.Abs(t.X-prevX) < 0.01 {
				x = effX
			} else {
				prevX = t.X
			}
			effX = x + w
			if t.S == " " || strings.TrimSpace(t.S) == "" {
				if cur != nil {
					line.words = append(line.words, *cur)
					cur = nil
				}
				prevEnd = x + w
				continue
			}
			if cur != nil && x-prevEnd > 0.2*fs {
				line.words = append(line.words, *cur)
				cur = nil
			}
			if cur == nil {
				cur = &pdfWord{x0: x}
			}
			cur.s += t.S
			prevEnd = x + w
			cur.x1 = prevEnd
		}
		if cur != nil {
			line.words = append(line.words, *cur)
		}
		if len(line.words) > 0 {
			out = append(out, line)
		}
	}
	return out
}

var pdfFooter = regexp.MustCompile(`(?i)\bpage\s*\d+\s*(of|/)\s*\d+|\bgenerated\b.*\b(by|on)\b|\bprinted on\b`)

// segment splits a heading line into cells: words far apart are always
// separate; words close together are split where that makes known
// headings ("Inv # Date Customer Name" → "Inv #", "Date", "Customer Name").
func segment(words []pdfWord, h float64) [][]pdfWord {
	var parts [][]pdfWord
	start := 0
	for i := 1; i <= len(words); i++ {
		if i == len(words) || words[i].x0-words[i-1].x1 > 0.55*h {
			parts = append(parts, words[start:i])
			start = i
		}
	}
	var out [][]pdfWord
	for _, p := range parts {
		out = append(out, splitKnown(p)...)
	}
	return out
}

// splitKnown splits a run of words into headings by dynamic programming:
// the split whose pieces best name known fields wins.
func splitKnown(ws []pdfWord) [][]pdfWord {
	n := len(ws)
	if n <= 1 {
		return [][]pdfWord{ws}
	}
	text := func(a, b int) string {
		parts := make([]string, 0, b-a)
		for _, w := range ws[a:b] {
			parts = append(parts, w.s)
		}
		return strings.Join(parts, " ")
	}
	piece := func(a, b int) float64 {
		t := text(a, b)
		best := 0.0
		for _, f := range Fields {
			best = max(best, fieldScore(t, f))
		}
		if best < 0.7 {
			return -0.1 * float64(b-a)
		}
		return best
	}
	// Each extra piece costs 1, so a heading that is known as a whole
	// ("HS Code") is not split into two known words ("HS", "Code").
	score := make([]float64, n+1)
	cut := make([]int, n+1)
	for i := 1; i <= n; i++ {
		score[i] = math.Inf(-1)
		for j := max(0, i-5); j < i; j++ {
			s := score[j] + piece(j, i)
			if j > 0 {
				s--
			}
			if s > score[i] {
				score[i], cut[i] = s, j
			}
		}
	}
	// Keep the run whole unless splitting finds more known headings.
	if whole := piece(0, n); whole >= score[n] || score[n] <= 0 {
		return [][]pdfWord{ws}
	}
	var out [][]pdfWord
	for i := n; i > 0; i = cut[i] {
		out = append([][]pdfWord{ws[cut[i]:i]}, out...)
	}
	return out
}

func cellsText(cells [][]pdfWord) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		parts := make([]string, len(c))
		for j, w := range c {
			parts[j] = w.s
		}
		out[i] = strings.Join(parts, " ")
	}
	return out
}

type band struct{ x0, x1 float64 }

// bestScore is how well text names any field.
func bestScore(text string) float64 {
	best := 0.0
	for _, f := range Fields {
		best = max(best, fieldScore(text, f))
	}
	return best
}

// mergeHeading folds a line of wrapped headings (above or below the
// heading line) into the heading cells, when every piece of it lands on a
// heading and makes that heading at least as recognisable ("BUYER" over
// "NTN/CNIC" makes "BUYER NTN/CNIC"). A report title above the headings
// fails the test and is left alone.
func mergeHeading(head [][]pdfWord, header []string, l pdfLine, before bool) bool {
	segs := segment(l.words, l.h)
	if len(segs) == 0 || len(segs) > len(head) {
		return false
	}
	merged := append([]string(nil), header...)
	for _, sg := range segs {
		t := cellsText([][]pdfWord{sg})[0]
		if _, num := Number(t); num {
			return false
		}
		x0, x1 := sg[0].x0, sg[len(sg)-1].x1
		hi, ho := -1, -3.0
		for i, c := range head {
			if o := math.Min(x1, c[len(c)-1].x1) - math.Max(x0, c[0].x0); o > ho {
				hi, ho = i, o
			}
		}
		if hi < 0 {
			return false
		}
		combined := merged[hi] + " " + t
		if before {
			combined = t + " " + merged[hi]
		}
		if cs := bestScore(combined); cs < 0.7 || cs < bestScore(merged[hi]) {
			return false
		}
		merged[hi] = combined
	}
	copy(header, merged)
	return true
}

// pdfTable rebuilds a table from the lines of a PDF.
func pdfTable(lines []pdfLine) Sheet {
	sh := Sheet{Name: "PDF table"}
	// The heading is the line (near the top) that names the most fields.
	hi, best := -1, 0.0
	for i, l := range lines[:min(len(lines), 80)] {
		if s := headerScore(cellsText(segment(l.words, l.h))); s > best {
			hi, best = i, s
		}
	}
	if hi < 0 || best < 2 {
		hi = 0
		for i, l := range lines[:min(len(lines), 40)] {
			if len(l.words) > len(lines[hi].words) {
				hi = i
			}
		}
	}
	head := lines[hi]
	headCells := segment(head.words, head.h)
	headWords := map[string]bool{}
	for _, w := range head.words {
		headWords[strings.ToLower(w.s)] = true
	}
	isHeaderRepeat := func(l pdfLine) bool {
		hits := 0
		for _, w := range l.words {
			if headWords[strings.ToLower(w.s)] {
				hits++
			}
		}
		return hits*10 >= len(l.words)*8
	}
	// Body lines: below the heading, without page furniture or repeated
	// headings. Headings wrapped onto the next line are kept aside.
	header := cellsText(headCells)
	near := func(l pdfLine) bool { return l.page == head.page && math.Abs(l.y-head.y) < 1.7*head.h }
	if hi > 0 && near(lines[hi-1]) {
		mergeHeading(headCells, header, lines[hi-1], true)
	}
	skipNext := hi+1 < len(lines) && near(lines[hi+1]) && mergeHeading(headCells, header, lines[hi+1], false)
	var body []pdfLine
	for i := hi + 1; i < len(lines); i++ {
		l := lines[i]
		if l.page == head.page && l.y > head.y {
			continue
		}
		if (i == hi+1 && skipNext) || pdfFooter.MatchString(l.text()) || isHeaderRepeat(l) {
			continue
		}
		body = append(body, l)
	}
	// Columns: bands of the page covered by the body's words, split where
	// a vertical strip of white space runs through (nearly) every line.
	minX, maxX := math.Inf(1), math.Inf(-1)
	for _, l := range append([]pdfLine{head}, body...) {
		for _, w := range l.words {
			minX, maxX = math.Min(minX, w.x0), math.Max(maxX, w.x1)
		}
	}
	width := int(math.Ceil(maxX-minX)) + 2
	if width <= 0 || width > 5000 {
		return sh
	}
	occ := make([]int, width)
	for _, l := range body {
		seen := make([]bool, width)
		for _, w := range l.words {
			for x := int(w.x0 - minX); x <= int(w.x1-minX) && x < width; x++ {
				if x >= 0 && !seen[x] {
					seen[x] = true
					occ[x]++
				}
			}
		}
	}
	limit := len(body) / 20
	var bands []band
	in := false
	for x := 0; x < width; x++ {
		busy := occ[x] > limit
		switch {
		case busy && !in:
			bands = append(bands, band{x0: float64(x) + minX})
			in = true
		case !busy && in:
			bands[len(bands)-1].x1 = float64(x) + minX
			in = false
		}
	}
	if in {
		bands[len(bands)-1].x1 = maxX
	}
	// The heading cells define the columns: each band joins the heading
	// it lies under (a long name spills past its heading, and gaps between
	// its words open extra bands); bands under no heading join their left
	// neighbour when close, else become unnamed columns.
	if len(bands) == 0 {
		for _, c := range headCells {
			bands = append(bands, band{c[0].x0, c[len(c)-1].x1})
		}
	}
	overlap := func(a0, a1 float64, b band) float64 { return math.Min(a1, b.x1) - math.Max(a0, b.x0) }
	cols := make([]band, len(headCells))
	used := make([]bool, len(headCells))
	for i, c := range headCells {
		cols[i] = band{c[0].x0, c[len(c)-1].x1}
	}
	var loose []band
	for _, b := range bands {
		bi, bo := -1, 0.0
		for i, c := range headCells {
			if o := overlap(c[0].x0, c[len(c)-1].x1, b); o > bo {
				bi, bo = i, o
			}
		}
		if bi < 0 {
			loose = append(loose, b)
			continue
		}
		if !used[bi] {
			cols[bi], used[bi] = b, true
		} else {
			cols[bi].x0, cols[bi].x1 = math.Min(cols[bi].x0, b.x0), math.Max(cols[bi].x1, b.x1)
		}
	}
	for _, b := range loose {
		// The nearest column on the left, if the gap is only a word space.
		li := -1
		for i, c := range cols {
			if c.x1 <= b.x0+0.5 && (li < 0 || c.x1 > cols[li].x1) {
				li = i
			}
		}
		if li >= 0 && b.x0-cols[li].x1 < head.h {
			cols[li].x1 = math.Max(cols[li].x1, b.x1)
			continue
		}
		cols = append(cols, b)
		header = append(header, fmt.Sprintf("Column %d", len(cols)))
	}
	// Keep the columns in page order.
	order := make([]int, len(cols))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return cols[order[a]].x0 < cols[order[b]].x0 })
	sortedCols := make([]band, len(cols))
	sortedHeader := make([]string, len(cols))
	for i, o := range order {
		sortedCols[i], sortedHeader[i] = cols[o], header[o]
	}
	cols, header = sortedCols, sortedHeader
	colOf := func(w pdfWord) int {
		bi, bo := 0, math.Inf(-1)
		for i, c := range cols {
			if o := overlap(w.x0, w.x1, c); o > bo {
				bi, bo = i, o
			}
		}
		return bi
	}
	sh.addRow(header)
	rows := make([][]string, 0, len(body))
	for _, l := range body {
		cells := make([]string, len(cols))
		for _, w := range l.words {
			j := colOf(w)
			if cells[j] != "" {
				cells[j] += " "
			}
			cells[j] += w.s
		}
		rows = append(rows, cells)
	}
	// Numeric columns: most cells are numbers.
	numeric := make([]bool, len(cols))
	for j := range cols {
		n, num := 0, 0
		for _, r := range rows {
			if r[j] != "" {
				n++
				if _, ok := Number(r[j]); ok {
					num++
				}
			}
		}
		numeric[j] = n > 0 && num*10 >= n*7
	}
	// A line with text only in text columns and nothing in the first
	// column continues the row above (a wrapped description).
	var out [][]string
	for i, r := range rows {
		cont := len(out) > 0 && r[0] == "" && body[i].page == body[i-1].page
		for j, c := range r {
			if c != "" && numeric[j] {
				cont = false
			}
		}
		if cont {
			prev := out[len(out)-1]
			for j, c := range r {
				if c != "" {
					prev[j] = strings.TrimSpace(prev[j] + " " + c)
				}
			}
			continue
		}
		out = append(out, r)
	}
	for _, r := range out {
		if !sh.addRow(r) {
			break
		}
	}
	return sh
}
