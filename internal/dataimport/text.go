// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package dataimport

import (
	"encoding/csv"
	"errors"
	"io"
	"regexp"
	"strings"
)

var delimNames = map[rune]string{',': "comma", ';': "semicolon", '\t': "tab", '|': "pipe"}

// readDelimited reads CSV, TSV, semicolon- or pipe-separated text, or —
// when no separator fits — a report printed in fixed-width columns.
func readDelimited(text, ext string) *Book {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	best, bestScore := rune(0), 0.0
	for _, d := range []rune{',', ';', '\t', '|'} {
		rows, ok := parseDelimited(text, d, 200)
		if !ok {
			continue
		}
		if s := consistency(rows); s > bestScore {
			best, bestScore = d, s
		}
	}
	b := &Book{Kind: "csv"}
	switch ext {
	case ".tsv", ".tab":
		b.Format = "tab-separated text"
	case ".txt", ".prn", ".dat":
		b.Format = "text file"
		b.Kind = "txt"
	default:
		b.Format = "CSV file"
	}
	if best != 0 && bestScore >= 1.5 {
		rows, _ := parseDelimited(text, best, MaxRows+50)
		sh := Sheet{Name: "Data"}
		for _, r := range rows {
			if !sh.addRow(r) {
				break
			}
		}
		b.Sheets = []Sheet{sh}
		if best != ',' {
			b.Notes = append(b.Notes, "Columns are separated by "+delimNames[best]+"s.")
		}
		return b
	}
	// Fixed-width report: columns are separated by two or more spaces.
	b.Format, b.Kind = "text report", "txt"
	sh := Sheet{Name: "Data"}
	gap := regexp.MustCompile(`\s{2,}|\t`)
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if isRule(line) {
			continue
		}
		if !sh.addRow(gap.Split(strings.TrimSpace(line), -1)) {
			break
		}
	}
	b.Sheets = []Sheet{sh}
	b.Notes = append(b.Notes, "No separator was found; columns were split where two or more spaces separate the values. Check the matching carefully.")
	return b
}

// isRule recognises ruled lines such as "-----" or "=====" in printed
// reports.
func isRule(line string) bool {
	t := strings.TrimSpace(line)
	return t != "" && strings.Trim(t, "-=_+*| ") == ""
}

func parseDelimited(text string, d rune, limit int) ([][]string, bool) {
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = d
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	var rows [][]string
	for len(rows) < limit {
		rec, err := r.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			if len(rows) == 0 {
				return nil, false
			}
			break
		}
		// Keep blank lines as empty rows so that row numbers match the
		// file (the CSV reader skips them).
		if line, _ := r.FieldPos(0); line > 0 {
			for len(rows) < line-1 && len(rows) < limit {
				rows = append(rows, nil)
			}
		}
		if len(rec) == 1 && strings.TrimSpace(rec[0]) == "" {
			rec = nil
		}
		rows = append(rows, rec)
	}
	return rows, len(rows) > 0
}

// consistency scores how well a separator splits the rows: the share of
// rows having the most common field count, times that count.
func consistency(rows [][]string) float64 {
	counts := map[int]int{}
	total := 0
	for _, r := range rows {
		if len(r) > 0 {
			counts[len(r)]++
			total++
		}
	}
	mode, n := 0, 0
	for k, v := range counts {
		if v > n || (v == n && k > mode) {
			mode, n = k, v
		}
	}
	if mode < 2 {
		return 0
	}
	return float64(n) / float64(total) * float64(min(mode, 12))
}
