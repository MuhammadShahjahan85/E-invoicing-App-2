// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package dataimport

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"golang.org/x/text/encoding/unicode"
)

// mapped reads a file and returns, for the first sheet, the data rows
// keyed by the suggested fields.
func mapped(t *testing.T, name string, data []byte) (*Book, []map[string]string) {
	t.Helper()
	b, err := Read(name, data)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	sh := b.Sheets[0]
	hr := HeaderRow(sh.Rows)
	headers := sh.Rows[hr]
	samples := make([][]string, len(headers))
	for _, r := range sh.Rows[hr+1:] {
		for i := range headers {
			if i < len(r) {
				samples[i] = append(samples[i], r[i])
			}
		}
	}
	var out []map[string]string
	ms := Suggest(headers, samples)
	for _, r := range sh.Rows[hr+1:] {
		m := map[string]string{}
		for _, mt := range ms {
			if mt.Col < len(r) {
				m[mt.Field] = r[mt.Col]
			}
		}
		out = append(out, m)
	}
	return b, out
}

func csvBytes(rows [][]string, comma rune) []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Comma = comma
	_ = w.WriteAll(rows)
	return b.Bytes()
}

// checkRegister verifies the sales register was read and understood.
func checkRegister(t *testing.T, name string, rows []map[string]string) {
	t.Helper()
	if len(rows) < 4 {
		t.Fatalf("%s: %d rows", name, len(rows))
	}
	r := rows[0]
	want := map[string]string{"invoice_ref": "INV-1001", "buyer_name": "Punjab Builders Co.", "buyer_ntn_cnic": "2046004", "description": "Cement bag 50kg",
		"hs_code": "2523.2900", "uom": "Bag", "buyer_address": "Lahore"}
	for k, v := range want {
		if strings.TrimSpace(r[k]) != v {
			t.Errorf("%s: %s = %q, want %q (row %v)", name, k, r[k], v, r)
		}
	}
	if q, ok := Number(r["quantity"]); !ok || q.IntPart() != 100 {
		t.Errorf("%s: quantity %q", name, r["quantity"])
	}
	if p, ok := Number(r["unit_price"]); !ok || p.IntPart() != 1200 {
		t.Errorf("%s: unit price %q", name, r["unit_price"])
	}
	if v, ok := Number(r["value_excl_st"]); !ok || v.IntPart() != 120000 {
		t.Errorf("%s: value %q", name, r["value_excl_st"])
	}
	if Rate(r["rate"]) != "18%" {
		t.Errorf("%s: rate %q", name, r["rate"])
	}
	if d, ok := Date(r["invoice_date"]); !ok || d != "2026-10-01" {
		t.Errorf("%s: date %q → %q", name, r["invoice_date"], d)
	}
	if d, _ := Date(rows[3]["invoice_date"]); d != "2026-10-13" {
		t.Errorf("%s: day-first date %q → %q", name, rows[3]["invoice_date"], d)
	}
	if NTN(rows[3]["buyer_ntn_cnic"]) != "3520212345671" {
		t.Errorf("%s: CNIC %q", name, rows[3]["buyer_ntn_cnic"])
	}
}

func TestReadsEveryFormat(t *testing.T) {
	plain := csvBytes(salesRegister, ',')
	utf16, _ := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes(csvBytes(salesRegister, '\t'))
	files := map[string][]byte{
		"register.csv":      plain,
		"register-sc.csv":   csvBytes(salesRegister, ';'),
		"register.txt":      utf16,
		"register-pipe.dat": csvBytes(salesRegister, '|'),
	}
	for _, name := range []string{"sales.xlsx", "sales.xls", "sales.ods", "sales.pdf"} {
		b, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = b
	}
	files["register.html"] = []byte(htmlTable(salesRegister))
	files["register.xml"] = []byte(spreadsheetML(salesRegister))
	for name, data := range files {
		t.Run(name, func(t *testing.T) {
			b, rows := mapped(t, name, data)
			checkRegister(t, name+" ("+b.Format+")", rows)
		})
	}
}

func htmlTable(rows [][]string) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><body><h2>Report</h2><table>")
	for _, r := range rows {
		b.WriteString("<tr>")
		for _, c := range r {
			b.WriteString("<td>" + c + "</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</table></body></html>")
	return b.String()
}

func spreadsheetML(rows [][]string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><Workbook xmlns="urn:schemas-microsoft-com:office:spreadsheet" xmlns:ss="urn:schemas-microsoft-com:office:spreadsheet"><Worksheet ss:Name="Sales"><Table>`)
	for _, r := range rows {
		b.WriteString("<Row>")
		for _, c := range r {
			b.WriteString(`<Cell><Data ss:Type="String">` + strings.ReplaceAll(c, "&", "&amp;") + `</Data></Cell>`)
		}
		b.WriteString("</Row>")
	}
	b.WriteString("</Table></Worksheet></Workbook>")
	return b.String()
}

func TestWordTables(t *testing.T) {
	data, err := os.ReadFile("testdata/table.docx")
	if err != nil {
		t.Fatal(err)
	}
	b, rows := mapped(t, "table.docx", data)
	if b.Kind != "docx" || len(rows) != 2 || rows[0]["invoice_ref"] != "INV-1001" || rows[1]["buyer_name"] != "Walk-in Retail Buyer" || rows[0]["rate"] != "18%" {
		t.Fatalf("%s: %+v", b.Format, rows)
	}
}

func TestFBRDigitalInvoicingJSON(t *testing.T) {
	payload := `[{"invoiceType":"Sale Invoice","invoiceDate":"2026-10-05","sellerNTNCNIC":"1234567","sellerBusinessName":"Seller",
	"buyerNTNCNIC":"2046004","buyerBusinessName":"Punjab Builders Co.","buyerProvince":"Punjab","buyerAddress":"Lahore","buyerRegistrationType":"Registered",
	"invoiceRefNo":"","items":[
	 {"hsCode":"2523.2900","productDescription":"Cement bag 50kg","rate":"18%","uoM":"Numbers, pieces, units","quantity":100,"totalValues":141600,
	  "valueSalesExcludingST":120000,"fixedNotifiedValueOrRetailPrice":0,"salesTaxApplicable":21600,"salesTaxWithheldAtSource":0,"extraTax":"",
	  "furtherTax":0,"sroScheduleNo":"","fedPayable":0,"discount":0,"saleType":"Goods at standard rate (default)","sroItemSerialNo":""},
	 {"hsCode":"7214.2000","productDescription":"Steel bar","rate":"18%","uoM":"KG","quantity":500,"totalValues":147500,"valueSalesExcludingST":125000,
	  "fixedNotifiedValueOrRetailPrice":0,"salesTaxApplicable":22500,"salesTaxWithheldAtSource":0,"extraTax":"","furtherTax":0,"sroScheduleNo":"",
	  "fedPayable":0,"discount":0,"saleType":"Goods at standard rate (default)","sroItemSerialNo":""}]}]`
	b, rows := mapped(t, "payload.json", []byte(payload))
	if b.Format != "FBR Digital Invoicing JSON" || len(rows) != 2 {
		t.Fatalf("%s: %d rows %+v", b.Format, len(rows), rows)
	}
	r := rows[1]
	for k, v := range map[string]string{"description": "Steel bar", "hs_code": "7214.2000", "uom": "KG", "quantity": "500", "value_excl_st": "125000",
		"sales_tax": "22500", "buyer_name": "Punjab Builders Co.", "buyer_province": "Punjab", "invoice_date": "2026-10-05", "doc_type": "Sale Invoice",
		"sale_type": "Goods at standard rate (default)", "rate": "18%", "total_value": "147500"} {
		if r[k] != v {
			t.Errorf("%s = %q, want %q", k, r[k], v)
		}
	}
}

func TestGenericXML(t *testing.T) {
	doc := `<?xml version="1.0"?><SalesBook><Voucher VoucherNo="V-1" Date="2026-10-02"><Party>Mehran Traders</Party><PartyNTN>3520212345671</PartyNTN>
	<Lines><Line><Item>Steel bar</Item><HSCode>7214.2000</HSCode><Qty>40</Qty><Price>250</Price></Line>
	<Line><Item>Cement</Item><HSCode>2523.2900</HSCode><Qty>2</Qty><Price>1200</Price></Line></Lines></Voucher>
	<Voucher VoucherNo="V-2" Date="2026-10-03"><Party>ABC</Party><PartyNTN>2046004</PartyNTN><Lines><Line><Item>Cement</Item><HSCode>2523.2900</HSCode><Qty>1</Qty><Price>1200</Price></Line></Lines></Voucher></SalesBook>`
	_, rows := mapped(t, "sales.xml", []byte(doc))
	if len(rows) != 3 || rows[1]["description"] != "Cement" || rows[1]["invoice_ref"] != "V-1" || rows[2]["buyer_name"] != "ABC" ||
		rows[0]["quantity"] != "40" || rows[0]["unit_price"] != "250" || rows[0]["invoice_date"] != "2026-10-02" || rows[0]["hs_code"] != "7214.2000" {
		t.Fatalf("rows %+v", rows)
	}
}

func TestOurOwnPDFReport(t *testing.T) {
	data, err := os.ReadFile("testdata/register.pdf")
	if err != nil {
		t.Fatal(err)
	}
	_, rows := mapped(t, "register.pdf", data)
	if len(rows) < 4 {
		t.Fatalf("%d rows: %+v", len(rows), rows)
	}
	if rows[0]["invoice_ref"] != "TR-INV-2627-000001" || rows[0]["hs_code"] != "2523.2900" || !strings.HasPrefix(rows[0]["buyer_name"], "Punjab Builders") {
		t.Fatalf("first row %+v", rows[0])
	}
}

func TestRejectsJunk(t *testing.T) {
	if _, err := Read("x.bin", []byte{0, 1, 2, 3, 4, 5, 0, 0, 0, 9, 8, 7, 0, 0}); err == nil {
		t.Fatal("binary junk must be refused")
	}
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	w, _ := zw.Create("readme.txt")
	_, _ = w.Write([]byte("hello"))
	_ = zw.Close()
	if _, err := Read("x.zip", zb.Bytes()); err == nil {
		t.Fatal("a plain zip must be refused")
	}
	if _, err := Read("empty.csv", nil); err == nil {
		t.Fatal("an empty file must be refused")
	}
}

func TestNormalisers(t *testing.T) {
	for in, want := range map[string]string{"Rs. 1,23,456.50/-": "123456.5", "(500)": "-500", "PKR 2,500": "2500", "1,000-": "-1000"} {
		if d, ok := Number(in); !ok || d.String() != want {
			t.Errorf("Number(%q) = %s %v", in, d, ok)
		}
	}
	for in, want := range map[string]string{"18": "18%", "0.18": "18%", "18.00 %": "18%", "Exempt": "Exempt", "5%": "5%"} {
		if got := Rate(in); got != want {
			t.Errorf("Rate(%q) = %q", in, got)
		}
	}
	for in, want := range map[string]string{"09/10/2026": "2026-10-09", "2026-10-09T10:30:00Z": "2026-10-09", "9-Oct-26": "2026-10-09",
		"46304": "2026-10-09", "10/31/2026": "2026-10-31", "October 9, 2026": "2026-10-09"} {
		if got, ok := Date(in); !ok || got != want {
			t.Errorf("Date(%q) = %q %v", in, got, ok)
		}
	}
	for in, want := range map[string]string{"Plot 4, Sundar Industrial Estate, Raiwind Road, Lahore": "PUNJAB", "SITE Area, Karachi": "SINDH",
		"Hayatabad, Peshawar": "KHYBER PAKHTUNKHWA", "Blue Area Islamabad": "CAPITAL TERRITORY", "Main road, Rahim Yar Khan": "PUNJAB", "KPK": "KHYBER PAKHTUNKHWA", "Somewhere": ""} {
		if got := ProvinceFromAddress(in); got != want {
			t.Errorf("ProvinceFromAddress(%q) = %q, want %q", in, got, want)
		}
	}
	if NTN("1234567-8") != "1234567" || NTN("35202-1234567-1") != "3520212345671" || !IsScientific("3.52021E+12") {
		t.Error("NTN cleaning")
	}
	if RegType("Non-Filer") != "Unregistered" || RegType("Filer") != "Registered" || RegType("maybe") != "" {
		t.Error("registration types")
	}
}
