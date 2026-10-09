// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package dataimport

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
)

// readZip dispatches the zipped office formats.
func readZip(data []byte, ext string) (*Book, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("the file looks like a ZIP archive but cannot be opened: %v", err)
	}
	names := map[string]*zip.File{}
	for _, f := range zr.File {
		names[f.Name] = f
	}
	switch {
	case names["xl/workbook.xml"] != nil:
		return readXLSX(data, ext)
	case names["content.xml"] != nil && names["mimetype"] != nil:
		mt, _ := readZipFile(names["mimetype"], 256)
		if strings.Contains(string(mt), "spreadsheet") {
			return readODS(names["content.xml"])
		}
		return nil, fmt.Errorf("OpenDocument text and presentations cannot be imported; copy the table into a spreadsheet (LibreOffice Calc or Excel)")
	case names["word/document.xml"] != nil:
		return readDOCX(names["word/document.xml"])
	}
	return nil, fmt.Errorf("%w: a ZIP archive that is not an Excel, OpenDocument or Word file — extract it and upload the spreadsheet inside", ErrUnsupported)
}

// readZipFile reads a zip entry with a size cap (zip bombs).
func readZipFile(f *zip.File, limit int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, limit))
}

func readXLSX(data []byte, ext string) (*Book, error) {
	// Bound decompression so that a crafted workbook cannot exhaust memory or disk.
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: 128 << 20, UnzipXMLSizeLimit: 32 << 20})
	if err != nil {
		return nil, fmt.Errorf("cannot read the Excel file: %v", err)
	}
	defer f.Close()
	b := &Book{Format: "Excel workbook (" + strings.TrimPrefix(nz(ext, ".xlsx"), ".") + ")", Kind: "xlsx"}
	for _, name := range f.GetSheetList() {
		if v, _ := f.GetSheetVisible(name); !v {
			continue
		}
		// Raw values: dates arrive as serial numbers (unambiguous, unlike
		// the US-style text excelize would format them as) and long
		// numbers such as CNICs keep every digit.
		rows, err := f.Rows(name)
		if err != nil {
			continue
		}
		sh := Sheet{Name: name}
		for rows.Next() {
			cols, err := rows.Columns(excelize.Options{RawCellValue: true})
			if err != nil {
				break
			}
			if !sh.addRow(cols) {
				break
			}
		}
		_ = rows.Close()
		b.Sheets = append(b.Sheets, sh)
	}
	return finish(b)
}

func readXLS(data []byte) (*Book, error) {
	wb, err := xls.OpenReader(bytes.NewReader(data), "utf-8")
	if err != nil {
		return nil, fmt.Errorf("cannot read the Excel 97-2003 file (%v); open it in Excel and save it as .xlsx", err)
	}
	b := &Book{Format: "Excel 97-2003 workbook (xls)", Kind: "xls"}
	for i := 0; i < wb.NumSheets(); i++ {
		ws := wb.GetSheet(i)
		if ws == nil {
			continue
		}
		sh := Sheet{Name: ws.Name}
		for r := 0; r <= int(ws.MaxRow); r++ {
			row := xlsRow(ws, r)
			if row == nil {
				if !sh.addRow(nil) {
					break
				}
				continue
			}
			var cells []string
			for c := 0; c < row.LastCol() && c < MaxCols; c++ {
				cells = append(cells, row.Col(c))
			}
			if !sh.addRow(cells) {
				break
			}
		}
		b.Sheets = append(b.Sheets, sh)
	}
	return finish(b)
}

// xlsRow returns a row of an .xls sheet, or nil for a blank row (the
// library dereferences a nil row for those).
func xlsRow(ws *xls.WorkSheet, r int) (row *xls.Row) {
	defer func() {
		if recover() != nil {
			row = nil
		}
	}()
	return ws.Row(r)
}

// readODS reads an OpenDocument spreadsheet (LibreOffice, WPS).
func readODS(content *zip.File) (*Book, error) {
	data, err := readZipFile(content, 64<<20)
	if err != nil {
		return nil, err
	}
	b := &Book{Format: "OpenDocument spreadsheet (ods)", Kind: "ods"}
	dec := xml.NewDecoder(bytes.NewReader(data))
	var sh *Sheet
	var row []string
	var cell strings.Builder
	inCell, cellRepeat, rowRepeat := false, 1, 1
	value := ""
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "table":
				if t.Name.Space == "urn:oasis:names:tc:opendocument:xmlns:table:1.0" {
					b.Sheets = append(b.Sheets, Sheet{Name: attr(t, "name")})
					sh = &b.Sheets[len(b.Sheets)-1]
				}
			case "table-row":
				row = nil
				rowRepeat = atoiDef(attr(t, "number-rows-repeated"), 1)
			case "table-cell", "covered-table-cell":
				inCell = true
				cell.Reset()
				cellRepeat = atoiDef(attr(t, "number-columns-repeated"), 1)
				// Typed values are more exact than the displayed text.
				value = ""
				switch attr(t, "value-type") {
				case "float", "percentage", "currency":
					value = attr(t, "value")
				case "date":
					value = attr(t, "date-value")
				}
			case "p":
				if inCell && cell.Len() > 0 {
					cell.WriteByte(' ')
				}
			case "s":
				if inCell {
					cell.WriteByte(' ')
				}
			}
		case xml.CharData:
			if inCell {
				cell.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "table-cell", "covered-table-cell":
				inCell = false
				v := cell.String()
				if value != "" {
					v = value
				}
				// Trailing empty cells are often "repeated" thousands of times.
				if v == "" && cellRepeat > 1 {
					cellRepeat = min(cellRepeat, MaxCols-len(row))
				}
				for i := 0; i < cellRepeat && len(row) < MaxCols; i++ {
					row = append(row, v)
				}
			case "table-row":
				if sh == nil {
					continue
				}
				empty := true
				for _, c := range row {
					if strings.TrimSpace(c) != "" {
						empty = false
						break
					}
				}
				if empty {
					// Blank rows are "repeated" to the end of the sheet.
					rowRepeat = 1
				}
				rowRepeat = min(rowRepeat, 1000)
				for i := 0; i < rowRepeat; i++ {
					if !sh.addRow(row) {
						break
					}
				}
			case "table":
				sh = nil
			}
		}
	}
	return finish(b)
}

// readSpreadsheetML reads the Excel 2003 XML format still produced by
// some accounting packages.
func readSpreadsheetML(text string) (*Book, error) {
	b := &Book{Format: "Excel 2003 XML spreadsheet", Kind: "xml"}
	dec := xml.NewDecoder(strings.NewReader(text))
	dec.Strict = false
	var sh *Sheet
	var row []string
	var cell strings.Builder
	inData := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "Worksheet":
				b.Sheets = append(b.Sheets, Sheet{Name: attr(t, "Name")})
				sh = &b.Sheets[len(b.Sheets)-1]
			case "Row":
				row = nil
			case "Cell":
				if idx := atoiDef(attr(t, "Index"), 0); idx > len(row)+1 && idx <= MaxCols {
					for len(row) < idx-1 {
						row = append(row, "")
					}
				}
				cell.Reset()
			case "Data":
				inData = true
			}
		case xml.CharData:
			if inData {
				cell.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "Data":
				inData = false
			case "Cell":
				row = append(row, cell.String())
			case "Row":
				if sh != nil && !sh.addRow(row) {
					sh = nil
				}
			}
		}
	}
	return finish(b)
}

// readDOCX reads the tables of a Word document.
func readDOCX(doc *zip.File) (*Book, error) {
	data, err := readZipFile(doc, 64<<20)
	if err != nil {
		return nil, err
	}
	b := &Book{Format: "Word document (docx)", Kind: "docx"}
	dec := xml.NewDecoder(bytes.NewReader(data))
	depth := 0 // table nesting: only top-level tables are read
	var sh *Sheet
	var row []string
	var cell strings.Builder
	span := 1
	inText := false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "tbl":
				depth++
				if depth == 1 {
					b.Sheets = append(b.Sheets, Sheet{Name: fmt.Sprintf("Table %d", len(b.Sheets)+1)})
					sh = &b.Sheets[len(b.Sheets)-1]
				}
			case "tr":
				if depth == 1 {
					row = nil
				}
			case "tc":
				if depth == 1 {
					cell.Reset()
					span = 1
				}
			case "gridSpan":
				if depth == 1 {
					span = atoiDef(attr(t, "val"), 1)
				}
			case "t":
				inText = true
			case "tab":
				cell.WriteByte(' ')
			case "p":
				if depth >= 1 && cell.Len() > 0 {
					cell.WriteByte(' ')
				}
			}
		case xml.CharData:
			if inText && depth >= 1 {
				cell.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "t":
				inText = false
			case "tc":
				if depth == 1 {
					row = append(row, cell.String())
					for i := 1; i < span && len(row) < MaxCols; i++ {
						row = append(row, "")
					}
				}
			case "tr":
				if depth == 1 && sh != nil {
					sh.addRow(row)
				}
			case "tbl":
				depth--
			}
		}
	}
	if len(b.Sheets) == 0 {
		return nil, fmt.Errorf("the Word document has no tables; put the invoice lines in a table, or save them as Excel or CSV")
	}
	return finish(b)
}

func attr(t xml.StartElement, local string) string {
	for _, a := range t.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func atoiDef(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return def
	}
	return n
}

func nz(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
