// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package dataimport reads tabular data from the files businesses keep —
// Excel workbooks (.xlsx, .xls), OpenDocument spreadsheets, CSV, TSV and
// text exports, JSON and XML (including FBR Digital Invoicing payloads),
// Word and HTML tables and PDF reports — and recognises which column holds
// which invoice particular, so that a sales register in any layout can be
// imported.
package dataimport

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

// Limits protect the server from oversized or hostile files.
const (
	MaxBytes = 20 << 20
	MaxRows  = 20000
	MaxCols  = 200
	maxCell  = 2000
)

// Sheet is one table found in a file: a worksheet, a Word or HTML table,
// or the rows read from a JSON, XML or PDF document.
type Sheet struct {
	Name string
	Rows [][]string
}

// Book is everything read from a file.
type Book struct {
	Format string // human description, e.g. "Excel workbook (.xlsx)"
	Kind   string // short code: xlsx, xls, ods, csv, txt, json, xml, html, docx, pdf
	Sheets []Sheet
	Notes  []string
}

// ErrUnsupported reports a file this package cannot read.
var ErrUnsupported = errors.New("unsupported file")

// Read detects the format of data (by content first, then by name) and
// reads its tables.
func Read(filename string, data []byte) (book *Book, err error) {
	defer func() {
		// Readers of third-party formats can panic on damaged files; a bad
		// upload must never take the server down.
		if r := recover(); r != nil {
			book, err = nil, fmt.Errorf("the file could not be read — it may be damaged or password-protected (%v)", r)
		}
	}()
	if len(data) == 0 {
		return nil, fmt.Errorf("the file is empty")
	}
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("the file is larger than %d MB; split it into smaller files", MaxBytes>>20)
	}
	ext := strings.ToLower(filepath.Ext(filename))
	switch {
	case bytes.HasPrefix(data, []byte("PK\x03\x04")):
		return readZip(data, ext)
	case bytes.HasPrefix(data, []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}):
		if ext == ".doc" {
			return nil, fmt.Errorf("old Word documents (.doc) cannot be read; open the file in Word and save it as .docx, or copy the table into Excel")
		}
		return readXLS(data)
	case bytes.HasPrefix(data, []byte("%PDF")):
		return readPDF(data)
	}
	text, enc := decodeText(data)
	trimmed := strings.TrimSpace(text)
	switch {
	case strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "["):
		b, err := readJSON(trimmed)
		if err == nil || ext == ".json" {
			return b, err
		}
	case strings.HasPrefix(trimmed, "<"):
		low := strings.ToLower(trimmed[:min(len(trimmed), 4096)])
		switch {
		case strings.Contains(low, "urn:schemas-microsoft-com:office:spreadsheet"):
			return readSpreadsheetML(trimmed)
		case strings.Contains(low, "<html") || strings.Contains(low, "<table") || strings.Contains(low, "<!doctype html"):
			return readHTML(trimmed)
		default:
			b, err := readXML(trimmed)
			if err == nil || ext == ".xml" {
				return b, err
			}
		}
	}
	if !looksLikeText(text) {
		return nil, fmt.Errorf("%w: the file does not look like a spreadsheet, a text export or a document with tables", ErrUnsupported)
	}
	b := readDelimited(text, ext)
	if enc != "" {
		b.Notes = append(b.Notes, "Text was converted from "+enc+".")
	}
	return b, nil
}

// decodeText turns bytes into UTF-8: UTF-16 files (Excel's "Unicode
// text") and Windows-1252 files (older accounting software) are converted.
func decodeText(data []byte) (string, string) {
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xFE}):
		if s, err := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewDecoder().Bytes(data); err == nil {
			return string(s), "UTF-16"
		}
	case bytes.HasPrefix(data, []byte{0xFE, 0xFF}):
		if s, err := unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewDecoder().Bytes(data); err == nil {
			return string(s), "UTF-16"
		}
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if utf8.Valid(data) {
		return string(data), ""
	}
	if s, err := charmap.Windows1252.NewDecoder().Bytes(data); err == nil {
		return string(s), "Windows-1252"
	}
	return string(data), ""
}

// looksLikeText rejects binary files: text has few control characters.
func looksLikeText(s string) bool {
	if s == "" {
		return false
	}
	sample := s[:min(len(s), 8192)]
	bad := 0
	for _, r := range sample {
		if (r < 0x20 && r != '\n' && r != '\r' && r != '\t') || r == utf8.RuneError {
			bad++
		}
	}
	return bad*20 < len(sample)
}

// clip limits a cell's length and trims spaces.
func clip(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, " ", " "))
	if len(s) > maxCell {
		s = s[:maxCell]
	}
	return s
}

// addRow appends a row to the sheet, cleaning cells and enforcing limits.
// It reports false once the sheet is full.
func (s *Sheet) addRow(cells []string) bool {
	if len(s.Rows) >= MaxRows+50 {
		return false
	}
	if len(cells) > MaxCols {
		cells = cells[:MaxCols]
	}
	row := make([]string, len(cells))
	for i, c := range cells {
		row[i] = clip(c)
	}
	s.Rows = append(s.Rows, row)
	return true
}

// nonEmpty drops sheets without data.
func nonEmpty(sheets []Sheet) []Sheet {
	var out []Sheet
	for _, s := range sheets {
		n := 0
		for _, r := range s.Rows {
			for _, c := range r {
				if c != "" {
					n++
					break
				}
			}
		}
		if n > 0 {
			out = append(out, s)
		}
	}
	return out
}

func finish(b *Book) (*Book, error) {
	b.Sheets = nonEmpty(b.Sheets)
	if len(b.Sheets) == 0 {
		return nil, fmt.Errorf("no table or data rows were found in this %s", b.Format)
	}
	for _, s := range b.Sheets {
		if len(s.Rows) > MaxRows+1 {
			b.Notes = append(b.Notes, fmt.Sprintf("%s has more than %d rows; only the first %d are read. Split the file and import the rest separately.", s.Name, MaxRows, MaxRows))
		}
	}
	return b, nil
}
