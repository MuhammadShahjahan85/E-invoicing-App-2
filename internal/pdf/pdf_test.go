// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package pdf

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"
)

func TestClean(t *testing.T) {
	for in, want := range map[string]string{
		"Cement\tbag 50kg": "Cement bag 50kg",
		"Rs 1,000 — ₨ ok":  "Rs 1,000 — ₨ ok",
		"اردو":             "????",
		"emoji 🙂":          "emoji ?",
		"line1\nline2":     "line1\nline2",
		"ctl\x07x":         "ctlx",
		"Şükran Çelik":     "Şükran Çelik",
	} {
		if got := Clean(in); got != want {
			t.Errorf("Clean(%q) = %q, want %q", in, got, want)
		}
	}
}

// pages counts the pages of a PDF.
func pages(b []byte) int {
	return len(regexp.MustCompile(`/Type /Page\b`).FindAll(b, -1))
}

func TestDocumentBuildingBlocks(t *testing.T) {
	d := New(false, Meta{Title: "Sales tax report pack", Company: "Indus Steel & Trading (Pvt) Ltd", CompanyLine: "NTN 1234567 · STRN 32-77-8761-234-56",
		Environment: "simulator", Generated: time.Date(2026, 10, 9, 9, 30, 0, 0, PKT), Author: "Ayesha Khan", Developer: "Veridian Partners Consultancy Private Limited"})
	d.Heading("Sales tax return · September 2026", "Sales tax report pack", "Everything reported to FBR in the period, ready to match with Annexure-C before filing.")
	d.Tiles([]Tile{{Label: "Value of supplies", Value: "Rs 12,345,678.00", Sub: "42 sale invoices", Hero: true},
		{Label: "Sales tax", Value: "Rs 2,222,222.04", Sub: "at 18% and reduced rates"}, {Label: "Further tax", Value: "Rs 0.00", Sub: "unregistered buyers"},
		{Label: "Needs attention", Value: "0", Sub: "every invoice with FBR", Tone: "good"}})
	d.Section("Books, FBR and Annexure-C", "Each side ties when nothing is left to resolve.")
	d.TiePanel([3]string{"BOOKS", "FBR", "ANNEX-C"}, [3]TieRow{{OK: true, Title: "Books = FBR", Detail: "Every issued invoice sent to FBR", Value: "0.00"},
		{OK: false, Title: "FBR = Annexure-C", Detail: "2 documents rejected or uncertain", Value: "12,000.00"},
		{OK: true, Title: "Books = Annexure-C", Detail: "No unissued drafts dated in the period", Value: "0.00"}})
	d.KV([][2]string{{"Tax period", "September 2026"}, {"Payment due", "Thu, 15 Oct 2026"}, {"Return due", "Sun, 18 Oct 2026 (extended by FBR from 18 Oct)"}})
	d.Checks([]Check{{OK: true, Title: "Every issued invoice is reported to FBR"}, {OK: false, Title: "No rejected invoices left uncorrected",
		Detail: "2 invoices were rejected by FBR. Correct and resubmit them so that they appear in Annexure-C."}})
	g := Grid{Cols: []Col{{Title: "FBR invoice no."}, {Title: "Date"}, {Title: "Buyer name", Max: 60}, {Title: "Description", Sub: true, Max: 70},
		{Title: "Value excl. ST", Right: true}, {Title: "Sales tax", Right: true}, {Title: "Total", Right: true}}}
	for i := 0; i < 90; i++ {
		g.Rows = append(g.Rows, []string{fmt.Sprintf("1234567DI17915480%05d", i), "09-Oct-2026", "Fertilizer Manufacturers & Importers of Pakistan (Private) Limited",
			"Urea fertilizer 50 kg bags\nHS 3102.1000 · Goods at standard rate (default) · 18%", "1,000,000.00", "180,000.00", "1,180,000.00"})
	}
	g.Totals = []string{"Total", "", "", "", "90,000,000.00", "16,200,000.00", "106,200,000.00"}
	d.Table(g)
	d.SignOff([]string{"Prepared by", "Reviewed by", "Approved by"})
	b, err := d.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) || pages(b) < 3 {
		t.Fatalf("pdf of %d bytes, %d pages", len(b), pages(b))
	}
	if out := os.Getenv("PDF_SAMPLE"); out != "" {
		_ = os.WriteFile(out, b, 0o644)
	}
}

func TestWideTableFits(t *testing.T) {
	d := New(true, Meta{Title: "Register", Environment: "production"})
	var cols []Col
	row := []string{}
	for i := 0; i < 26; i++ {
		cols = append(cols, Col{Title: fmt.Sprintf("Column heading %d", i), Right: i%2 == 0})
		row = append(row, "1,234,567.89")
	}
	d.Table(Grid{Cols: cols, Rows: [][]string{row, row}})
	if _, err := d.Bytes(); err != nil {
		t.Fatal(err)
	}
}
