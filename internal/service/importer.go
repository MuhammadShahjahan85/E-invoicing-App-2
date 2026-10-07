package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/store"
	"einvoicing/internal/validate"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

// ImportColumns is the import template, one row per invoice line. Rows that
// share an invoice_ref form one invoice.
var ImportColumns = []string{
	"invoice_ref", "invoice_date", "doc_type", "original_fbr_invoice_no", "scenario_id",
	"buyer_ntn_cnic", "buyer_name", "buyer_province", "buyer_address", "buyer_registration_type", "withholding_mode",
	"product_code", "hs_code", "description", "uom", "quantity", "unit_price", "discount", "value_excl_st",
	"sale_type", "rate", "retail_price", "sro_schedule_no", "sro_item_serial_no",
	"sales_tax", "further_tax", "extra_tax", "fed", "st_withheld",
}

// ImportTemplateCSV returns a CSV template with an example row.
func ImportTemplateCSV() []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write(ImportColumns)
	_ = w.Write([]string{"ERP-1001", time.Now().In(PKT).Format("2006-01-02"), "Sale Invoice", "", "",
		"2046004", "ABC Traders", "Punjab", "Lahore", "Registered", "",
		"", "0101.2100", "Example product", "Numbers, pieces, units", "10", "150", "0", "",
		"Goods at standard rate (default)", "18%", "", "", "",
		"", "", "", "", ""})
	w.Flush()
	return b.Bytes()
}

// ImportTemplateXLSX returns an Excel template with instructions.
func ImportTemplateXLSX() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sh := "Invoices"
	f.SetSheetName("Sheet1", sh)
	for i, c := range ImportColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sh, cell, c)
	}
	example := []any{"ERP-1001", time.Now().In(PKT).Format("2006-01-02"), "Sale Invoice", "", "", "2046004", "ABC Traders", "Punjab", "Lahore", "Registered", "",
		"", "0101.2100", "Example product", "Numbers, pieces, units", 10, 150, 0, "", "Goods at standard rate (default)", "18%", "", "", "", "", "", "", "", ""}
	for i, v := range example {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(sh, cell, v)
	}
	help := "Help"
	_, _ = f.NewSheet(help)
	lines := []string{
		"One row per invoice line. Rows with the same invoice_ref become one invoice (invoice_ref is stored as the external reference and prevents duplicates).",
		"invoice_date: YYYY-MM-DD or DD/MM/YYYY. doc_type: 'Sale Invoice' or 'Debit Note' (debit notes need original_fbr_invoice_no).",
		"buyer_registration_type: Registered / Unregistered. Registered buyers need buyer_ntn_cnic (7/9-digit NTN or 13-digit CNIC).",
		"Either product_code (from the Products master) or hs_code + description + uom + sale_type + rate must be given.",
		"value_excl_st, sales_tax, further_tax, extra_tax, fed and st_withheld are optional overrides; leave blank to let the tax engine compute them.",
		"withholding_mode: blank, 'fraction' (1/5th) or 'full'. scenario_id is only used in the FBR sandbox (SN001–SN028).",
	}
	for i, l := range lines {
		_ = f.SetCellValue(help, fmt.Sprintf("A%d", i+1), l)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ImportResult reports one imported (or previewed) invoice.
type ImportResult struct {
	Ref       string           `json:"ref"`
	Rows      []int            `json:"rows"`
	InvoiceID int64            `json:"invoiceId,omitempty"`
	Status    string           `json:"status"`
	Errors    []string         `json:"errors,omitempty"`
	Issues    []validate.Issue `json:"issues,omitempty"`
	Total     decimal.Decimal  `json:"total"`
}

// ImportSummary summarises an import.
type ImportSummary struct {
	Invoices int            `json:"invoices"`
	OK       int            `json:"ok"`
	Failed   int            `json:"failed"`
	Results  []ImportResult `json:"results"`
	Preview  bool           `json:"preview"`
}

// ReadRows parses a CSV or XLSX upload into header-keyed rows.
func ReadRows(filename string, r io.Reader) ([]map[string]string, error) {
	data, err := io.ReadAll(io.LimitReader(r, 20<<20))
	if err != nil {
		return nil, err
	}
	var table [][]string
	if strings.HasSuffix(strings.ToLower(filename), ".xlsx") {
		// Bound decompression so that a crafted workbook cannot exhaust memory or disk.
		f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: 128 << 20, UnzipXMLSizeLimit: 32 << 20})
		if err != nil {
			return nil, fmt.Errorf("cannot read Excel file: %w", err)
		}
		defer f.Close()
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("the workbook has no sheets")
		}
		table, err = f.GetRows(sheets[0])
		if err != nil {
			return nil, err
		}
	} else {
		data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
		cr := csv.NewReader(bytes.NewReader(data))
		cr.FieldsPerRecord = -1
		cr.TrimLeadingSpace = true
		table, err = cr.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("cannot read CSV: %w", err)
		}
	}
	if len(table) < 2 {
		return nil, fmt.Errorf("the file has no data rows")
	}
	header := make([]string, len(table[0]))
	for i, h := range table[0] {
		header[i] = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(h, " ", "_")))
	}
	var rows []map[string]string
	for _, rec := range table[1:] {
		m := map[string]string{}
		empty := true
		for i, v := range rec {
			if i < len(header) {
				m[header[i]] = strings.TrimSpace(v)
				if m[header[i]] != "" {
					empty = false
				}
			}
		}
		if !empty {
			rows = append(rows, m)
		}
	}
	return rows, nil
}

func parseImportDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	for _, layout := range []string{"2006-01-02", "02/01/2006", "2/1/2006", "02-01-2006", "02-Jan-2006", "2-Jan-2006", "01-02-06", "2006/01/02", "02.01.2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}
	// Excel serial date.
	if n, err := strconv.ParseFloat(s, 64); err == nil && n > 20000 && n < 80000 {
		t := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC).Add(time.Duration(n*24) * time.Hour)
		return t.Format("2006-01-02"), nil
	}
	return "", fmt.Errorf("unrecognised date %q", s)
}

func optDec(s string) (*decimal.Decimal, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	if s == "" {
		return nil, nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return nil, fmt.Errorf("%q is not a number", s)
	}
	return &d, nil
}

func reqDec(s string) (decimal.Decimal, error) {
	d, err := optDec(s)
	if err != nil || d == nil {
		return decimal.Zero, err
	}
	return *d, nil
}

// rowsToInputs groups rows into invoice inputs.
func rowsToInputs(rows []map[string]string, env domain.Environment) ([]*InvoiceInput, [][]int, []string, [][]string) {
	order := []string{}
	byRef := map[string]*InvoiceInput{}
	rowsByRef := map[string][]int{}
	errsByRef := map[string][]string{}
	for i, r := range rows {
		rowNo := i + 2 // header is row 1
		ref := r["invoice_ref"]
		if ref == "" {
			ref = fmt.Sprintf("row-%d", rowNo)
			errsByRef[ref] = append(errsByRef[ref], fmt.Sprintf("row %d: invoice_ref is required", rowNo))
		}
		in, ok := byRef[ref]
		if !ok {
			date, err := parseImportDate(r["invoice_date"])
			if err != nil {
				errsByRef[ref] = append(errsByRef[ref], fmt.Sprintf("row %d: %v", rowNo, err))
			}
			in = &InvoiceInput{Environment: env, DocType: firstNonEmpty(r["doc_type"], string(domain.DocSaleInvoice)), InvoiceDate: date,
				InvoiceRefNo: r["original_fbr_invoice_no"], ScenarioID: r["scenario_id"], ExternalRef: ref, Source: "import",
				Buyer: &BuyerInput{NTNCNIC: r["buyer_ntn_cnic"], Name: r["buyer_name"], Province: r["buyer_province"],
					Address: r["buyer_address"], RegistrationType: firstNonEmpty(r["buyer_registration_type"], "Unregistered")}}
			if wm := r["withholding_mode"]; wm != "" {
				in.WithholdingMode = &wm
			}
			byRef[ref] = in
			order = append(order, ref)
		}
		rowsByRef[ref] = append(rowsByRef[ref], rowNo)
		item := ItemInput{ProductCode: r["product_code"], HSCode: r["hs_code"], Description: r["description"], UoM: r["uom"],
			SaleType: r["sale_type"], Rate: r["rate"], SROScheduleNo: r["sro_schedule_no"], SROItemSerialNo: r["sro_item_serial_no"]}
		var err error
		bad := func(field string, e error) {
			if e != nil {
				errsByRef[ref] = append(errsByRef[ref], fmt.Sprintf("row %d %s: %v", rowNo, field, e))
			}
		}
		item.Quantity, err = reqDec(r["quantity"])
		bad("quantity", err)
		item.UnitPrice, err = reqDec(r["unit_price"])
		bad("unit_price", err)
		item.DiscountAmount, err = reqDec(r["discount"])
		bad("discount", err)
		item.RetailPrice, err = reqDec(r["retail_price"])
		bad("retail_price", err)
		item.Value, err = optDec(r["value_excl_st"])
		bad("value_excl_st", err)
		item.SalesTax, err = optDec(r["sales_tax"])
		bad("sales_tax", err)
		item.FurtherTax, err = optDec(r["further_tax"])
		bad("further_tax", err)
		item.ExtraTax, err = optDec(r["extra_tax"])
		bad("extra_tax", err)
		item.FED, err = optDec(r["fed"])
		bad("fed", err)
		item.STWithheld, err = optDec(r["st_withheld"])
		bad("st_withheld", err)
		in.Items = append(in.Items, item)
	}
	var ins []*InvoiceInput
	var rowGroups [][]int
	var refs []string
	var errs [][]string
	for _, ref := range order {
		ins = append(ins, byRef[ref])
		rowGroups = append(rowGroups, rowsByRef[ref])
		refs = append(refs, ref)
		errs = append(errs, errsByRef[ref])
	}
	return ins, rowGroups, refs, errs
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Import creates invoices from parsed rows. With preview=true nothing is
// saved; each invoice is computed and validated only.
func (s *Service) Import(ctx context.Context, a Actor, companyID int64, rows []map[string]string, preview, submit bool) (*ImportSummary, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	ins, groups, refs, errs := rowsToInputs(rows, c.Environment)
	sum := &ImportSummary{Preview: preview}
	for i, in := range ins {
		res := ImportResult{Ref: refs[i], Rows: groups[i]}
		if len(errs[i]) > 0 {
			res.Status, res.Errors = "error", errs[i]
			sum.Results = append(sum.Results, res)
			sum.Failed++
			continue
		}
		if preview {
			inv, vr, err := s.Build(ctx, c, in)
			if err != nil {
				res.Status, res.Errors = "error", []string{err.Error()}
				sum.Failed++
			} else {
				res.Total, res.Issues = inv.Totals.TotalValue, vr.Issues
				res.Status = "ok"
				if vr.HasErrors() {
					res.Status = "invalid"
					sum.Failed++
				} else {
					sum.OK++
				}
			}
			sum.Results = append(sum.Results, res)
			continue
		}
		in.Submit = submit
		inv, err := s.CreateInvoice(ctx, a, companyID, in)
		if err != nil {
			res.Status = "error"
			if se, ok := IsSubmissionError(err); ok {
				res.Issues = se.Issues
			}
			res.Errors = []string{err.Error()}
			sum.Failed++
			if inv == nil {
				if existing, e2 := s.Store.GetInvoiceByExternalRef(ctx, companyID, c.Environment, in.ExternalRef); e2 == nil {
					res.InvoiceID = existing.ID
				}
			}
		} else {
			res.InvoiceID, res.Status, res.Total, res.Issues = inv.ID, string(inv.Status), inv.Totals.TotalValue, inv.Validation
			if inv.Status == domain.StatusRejected || inv.Status == domain.StatusUncertain {
				sum.Failed++
			} else {
				sum.OK++
			}
		}
		sum.Results = append(sum.Results, res)
	}
	sum.Invoices = len(ins)
	if !preview {
		s.Audit(ctx, a, companyID, "invoice.import", "invoice", "", map[string]any{"invoices": sum.Invoices, "ok": sum.OK, "failed": sum.Failed, "submit": submit})
	}
	return sum, nil
}

// Table is a generic tabular export.
type Table struct {
	Title   string
	Headers []string
	Rows    [][]any
}

// CSV renders the table as CSV (UTF-8 with BOM so Excel opens it correctly).
func (t *Table) CSV() []byte {
	var b bytes.Buffer
	b.WriteString("\xef\xbb\xbf")
	w := csv.NewWriter(&b)
	_ = w.Write(t.Headers)
	for _, r := range t.Rows {
		rec := make([]string, len(r))
		for i, v := range r {
			switch x := v.(type) {
			case decimal.Decimal:
				rec[i] = x.StringFixed(2)
			default:
				rec[i] = fmt.Sprint(x)
			}
		}
		_ = w.Write(rec)
	}
	w.Flush()
	return b.Bytes()
}

// XLSX renders the table as an Excel workbook.
func (t *Table) XLSX() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sh := "Report"
	f.SetSheetName("Sheet1", sh)
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}, Fill: excelize.Fill{Type: "pattern", Color: []string{"#E8E8E8"}, Pattern: 1}})
	numFmt := "#,##0.00"
	money, _ := f.NewStyle(&excelize.Style{CustomNumFmt: &numFmt})
	for i, h := range t.Headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sh, cell, h)
		_ = f.SetCellStyle(sh, cell, cell, bold)
	}
	for r, row := range t.Rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			switch x := v.(type) {
			case decimal.Decimal:
				fv, _ := x.Float64()
				_ = f.SetCellValue(sh, cell, fv)
				_ = f.SetCellStyle(sh, cell, cell, money)
			default:
				_ = f.SetCellValue(sh, cell, x)
			}
		}
	}
	_ = f.SetPanes(sh, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// RegisterTable builds the sales register export (Annexure-C style columns).
func RegisterTable(lines []store.RegisterLine) *Table {
	t := &Table{Title: "Sales register", Headers: []string{"Doc Type", "Invoice No", "FBR Invoice No", "Date", "Ref FBR Invoice No",
		"Buyer NTN/CNIC", "Buyer Name", "Buyer Type", "Origin Province", "Destination Province", "Line", "HS Code", "Description",
		"Sale Type", "Rate", "Quantity", "UoM", "Value excl ST", "Fixed/Retail Value", "Sales Tax", "Extra Tax", "Further Tax", "FED",
		"ST Withheld", "Discount", "Total Value", "SRO/Schedule", "SRO Item S.No"}}
	for _, l := range lines {
		t.Rows = append(t.Rows, []any{l.DocType, l.InternalNo, l.FBRInvoiceNumber, l.InvoiceDate, l.InvoiceRefNo, l.BuyerNTNCNIC, l.BuyerName,
			l.BuyerRegistration, l.SellerProvince, l.BuyerProvince, l.LineNo, l.HSCode, l.Description, l.SaleType, l.Rate, l.Quantity.String(),
			l.UoM, l.ValueExclST, l.RetailValue, l.SalesTax, l.ExtraTax, l.FurtherTax, l.FED, l.STWithheld, l.Discount, l.TotalValue,
			l.SROScheduleNo, l.SROItemSerialNo})
	}
	return t
}

// TaxSummaryTable builds the tax summary export.
func TaxSummaryTable(rows []store.TaxSummaryRow) *Table {
	t := &Table{Title: "Tax summary", Headers: []string{"Doc Type", "Sale Type", "Rate", "Invoices", "Lines", "Value excl ST", "Retail Value",
		"Sales Tax", "Further Tax", "Extra Tax", "FED", "ST Withheld"}}
	for _, r := range rows {
		t.Rows = append(t.Rows, []any{r.DocType, r.SaleType, r.Rate, r.Invoices, r.Lines, r.ValueExclST, r.RetailValue, r.SalesTax,
			r.FurtherTax, r.ExtraTax, r.FED, r.STWithheld})
	}
	return t
}

// MonthlyTable builds the monthly summary export.
func MonthlyTable(rows []store.PeriodRow) *Table {
	t := &Table{Title: "Monthly summary", Headers: []string{"Tax Period", "Sale Invoices", "Debit Notes", "Value excl ST", "Sales Tax",
		"Further Tax", "Debit Note Value", "Debit Note Sales Tax", "ST Withheld"}}
	for _, r := range rows {
		t.Rows = append(t.Rows, []any{r.Period, r.SaleInvoices, r.DebitNotes, r.ValueExclST, r.SalesTax, r.FurtherTax, r.DebitValue, r.DebitSalesTax, r.STWithheld})
	}
	return t
}

// CustomerTable builds the buyer-wise export.
func CustomerTable(rows []store.CustomerRow) *Table {
	t := &Table{Title: "Buyer-wise summary", Headers: []string{"Buyer NTN/CNIC", "Buyer Name", "Buyer Type", "Invoices", "Value excl ST",
		"Sales Tax", "Further Tax", "ST Withheld", "Total Value"}}
	for _, r := range rows {
		t.Rows = append(t.Rows, []any{r.BuyerNTNCNIC, r.BuyerName, r.BuyerRegistration, r.Invoices, r.ValueExclST, r.SalesTax, r.FurtherTax, r.STWithheld, r.TotalValue})
	}
	return t
}
