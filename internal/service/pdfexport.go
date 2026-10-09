// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"fmt"
	"regexp"
	"strings"

	"einvoicing/internal/domain"
	"einvoicing/internal/pdf"
	"einvoicing/internal/tax"

	"github.com/shopspring/decimal"
)

var numeric = regexp.MustCompile(`^-?[0-9][0-9,]*(\.[0-9]+)?$`)

// countHeaders are integer columns that add up in a totals row (line
// numbers and the like do not).
var countHeaders = map[string]bool{"Invoices": true, "Lines": true, "Sale Invoices": true, "Debit Notes": true}

// Grid converts the table for the PDF renderer: amounts are grouped in
// thousands and right-aligned, and amount columns are totalled.
func (t *Table) Grid() pdf.Grid {
	cols := t.PDFCols
	if cols == nil {
		for i := range t.Headers {
			cols = append(cols, i)
		}
	}
	g := pdf.Grid{}
	sums := make([]decimal.Decimal, len(cols))
	counts := make([]int, len(cols))
	isAmount := make([]bool, len(cols))
	isCount := make([]bool, len(cols))
	for j, c := range cols {
		h := t.Headers[c]
		col := pdf.Col{Title: h}
		right := len(t.Rows) > 0
		for _, r := range t.Rows {
			switch x := r[c].(type) {
			case decimal.Decimal:
				isAmount[j] = true
			case int, int64:
			case string:
				// Quantities are exported as text; keep them aligned too.
				if !strings.Contains(h, "Quantity") || (x != "" && !numeric.MatchString(x)) {
					right = false
				}
			default:
				right = false
			}
		}
		col.Right = right
		isCount[j] = right && !isAmount[j] && countHeaders[h]
		lh := strings.ToLower(h)
		if !right && (strings.Contains(lh, "name") || strings.Contains(lh, "description")) {
			col.Max = 70
		}
		g.Cols = append(g.Cols, col)
	}
	// Debit notes reduce the seller's sales, so a table mixing them with
	// sale invoices is totalled net of them.
	docCol := -1
	for i, h := range t.Headers {
		if h == "Doc Type" {
			docCol = i
		}
	}
	netted := false
	for _, r := range t.Rows {
		row := make([]string, len(cols))
		debit := docCol >= 0 && r[docCol] == string(domain.DocDebitNote)
		netted = netted || debit
		for j, c := range cols {
			switch x := r[c].(type) {
			case decimal.Decimal:
				row[j] = tax.FormatAmount(x)
				if debit {
					sums[j] = sums[j].Sub(x)
				} else {
					sums[j] = sums[j].Add(x)
				}
			case int:
				row[j] = fmt.Sprint(x)
				counts[j] += x
			case int64:
				row[j] = fmt.Sprint(x)
				counts[j] += int(x)
			case nil:
			default:
				row[j] = fmt.Sprint(x)
			}
		}
		g.Rows = append(g.Rows, row)
	}
	any := false
	tot := make([]string, len(cols))
	for j := range cols {
		switch {
		case isAmount[j]:
			tot[j], any = tax.FormatAmount(sums[j]), true
		case isCount[j]:
			tot[j], any = fmt.Sprint(counts[j]), true
		}
	}
	if any {
		if tot[0] == "" {
			tot[0] = "Total"
			if netted {
				tot[0] = "Net of debit notes"
			}
		}
		g.Totals = tot
	}
	return g
}

// PDF renders the table as a landscape A4 document.
func (t *Table) PDF(m pdf.Meta, eyebrow, sub string) ([]byte, error) {
	if m.Title == "" {
		m.Title = t.Title
	}
	d := pdf.New(true, m)
	d.Heading(eyebrow, t.Title, sub)
	d.Table(t.Grid())
	return d.Bytes()
}
