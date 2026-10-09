// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package dataimport

import (
	"os"
	"testing"

	"github.com/xuri/excelize/v2"
)

// salesRegister is a typical Pakistani sales register: a title, a blank
// line, headings in the business's own words, day-first dates, amounts
// with thousands separators and a totals row.
var salesRegister = [][]string{
	{"Indus Steel & Trading (Pvt) Ltd"},
	{"Sales Register for October 2026"},
	{},
	{"Inv #", "Date", "Customer Name", "NTN/CNIC", "City", "Item Description", "HS Code", "Unit", "Qty", "Rate", "Amount", "GST %", "GST", "Total"},
	{"INV-1001", "01/10/2026", "Punjab Builders Co.", "2046004", "Lahore", "Cement bag 50kg", "2523.2900", "Bag", "100", "1,200.00", "120,000.00", "18%", "21,600.00", "141,600.00"},
	{"INV-1001", "01/10/2026", "Punjab Builders Co.", "2046004", "Lahore", "Steel bar 12mm", "7214.2000", "KG", "500", "250", "125,000.00", "18%", "22,500.00", "147,500.00"},
	{"INV-1002", "05/10/2026", "Walk-in Retail Buyer", "", "Karachi", "Cement bag 50kg", "2523.2900", "Bag", "1", "1,200.00", "1,200.00", "18%", "216.00", "1,416.00"},
	{"INV-1003", "13/10/2026", "Mehran Traders", "35202-1234567-1", "Hyderabad", "Steel bar 12mm", "7214.2000", "KG", "40", "250", "10,000.00", "18%", "1,800.00", "11,800.00"},
	{"Total", "", "", "", "", "", "", "", "", "", "256,200.00", "", "46,116.00", "302,316.00"},
}

// TestMakeFixtures writes testdata/sales.xlsx when FIXTURES=1; the .xls,
// .ods, .docx and .pdf fixtures were converted from it with LibreOffice.
func TestMakeFixtures(t *testing.T) {
	if os.Getenv("FIXTURES") != "1" {
		t.Skip("set FIXTURES=1 to regenerate the fixtures")
	}
	f := excelize.NewFile()
	defer f.Close()
	sh := "Sales"
	f.SetSheetName("Sheet1", sh)
	for r, row := range salesRegister {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			_ = f.SetCellValue(sh, cell, v)
		}
	}
	// Columns wide enough for their contents, as in a real register
	// (narrow columns would clip the text in the PDF export).
	for c, w := range []float64{11, 11, 22, 17, 11, 18, 11, 6, 6, 10, 12, 7, 11, 12} {
		col, _ := excelize.ColumnNumberToName(c + 1)
		_ = f.SetColWidth(sh, col, col, w)
	}
	// Printed landscape, one page wide, like a register exported to PDF.
	land, one, fit := "landscape", 1, true
	_ = f.SetPageLayout(sh, &excelize.PageLayoutOptions{Orientation: &land, FitToWidth: &one, FitToHeight: &one})
	_ = f.SetSheetProps(sh, &excelize.SheetPropsOptions{FitToPage: &fit})
	if err := f.SaveAs("testdata/sales.xlsx"); err != nil {
		t.Fatal(err)
	}
}
