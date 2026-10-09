// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"einvoicing/internal/dataimport"
	"einvoicing/internal/domain"
	"einvoicing/internal/store"

	"github.com/shopspring/decimal"
)

// The smart import reads a sales register in any layout: the file's
// columns are matched to invoice particulars (the user can change the
// matching), anything FBR needs that the file does not hold is asked for,
// and the rows then go through the same checks as the import template.

// ImportAnalysis describes an uploaded file and how its columns match.
type ImportAnalysis struct {
	FileName  string             `json:"fileName"`
	Format    string             `json:"format"`
	Kind      string             `json:"kind"`
	Notes     []string           `json:"notes"`
	Sheets    []ImportSheet      `json:"sheets"`
	Sheet     int                `json:"sheet"`
	HeaderRow int                `json:"headerRow"`
	Columns   []ImportColumn     `json:"columns"`
	Rows      int                `json:"rows"`
	Mapping   map[string]int     `json:"mapping"`
	Fields    []dataimport.Field `json:"fields"`
	Missing   []MissingInfo      `json:"missing"`
	Preview   [][]string         `json:"preview"`
	Lists     ImportLists        `json:"lists"`
}

// ImportSheet is a table found in the file.
type ImportSheet struct {
	Name string `json:"name"`
	Rows int    `json:"rows"`
}

// ImportColumn is a column of the chosen table.
type ImportColumn struct {
	Index   int      `json:"index"`
	Header  string   `json:"header"`
	Samples []string `json:"samples"`
	Field   string   `json:"field"`
	Score   float64  `json:"score"`
	By      string   `json:"by"`
}

// MissingInfo is something FBR needs that the file does not (fully)
// supply, with what the user can do about it.
type MissingInfo struct {
	Field   string `json:"field"`
	Label   string `json:"label"`
	Level   string `json:"level"` // required | recommended | fix
	Message string `json:"message"`
	// Blank counts rows without a value when the column is matched.
	Blank int `json:"blank"`
	// Input is what the user is asked for: "text", "date", "province",
	// "saletype", "uom", "rate", "hs", "grouping", "regrule", "valuemap"
	// (translate the file's values) or "" (only a warning).
	Input   string `json:"input"`
	Suggest string `json:"suggest"`
	// Values are the file's values FBR would not accept, for "valuemap".
	Values []ValueCount `json:"values,omitempty"`
}

// ValueCount is a value found in the file and how many lines use it.
type ValueCount struct {
	Value   string `json:"value"`
	Lines   int    `json:"lines"`
	Suggest string `json:"suggest"`
}

// ImportLists are the choices offered for missing values.
type ImportLists struct {
	Provinces []string `json:"provinces"`
	SaleTypes []string `json:"saleTypes"`
	UOMs      []string `json:"uoms"`
	Rates     []string `json:"rates"`
}

// ImportOptions are the user's decisions for a smart import.
type ImportOptions struct {
	Sheet     int            `json:"sheet"`
	HeaderRow *int           `json:"headerRow"`
	Mapping   map[string]int `json:"mapping"`
	// Defaults fill a field wherever the file leaves it blank (or has no
	// column for it).
	Defaults map[string]string `json:"defaults"`
	// Grouping decides which lines form one invoice when the file has no
	// invoice number: "row", "buyer-date" or "single".
	Grouping string `json:"grouping"`
	// ProvinceFromAddress works out a blank province from the address.
	ProvinceFromAddress bool `json:"provinceFromAddress"`
	// RegRule decides a blank registration type: "ntn" (registered when an
	// NTN/CNIC is given), or a fixed value.
	RegRule string `json:"regRule"`
	// ValueMap translates the file's values, per field (e.g. the unit
	// "Cartons" → "Packs").
	ValueMap map[string]map[string]string `json:"valueMap"`
}

var totalRow = regexp.MustCompile(`(?i)^\s*(grand\s+|sub\s*-?\s*|net\s+)?totals?\b`)

// sheetData is the chosen table: its headings and data rows (with their
// row numbers in the file).
type sheetData struct {
	book    *dataimport.Book
	sheet   int
	header  int
	headers []string
	rows    [][]string
	rowNos  []int
}

func loadSheet(filename string, data []byte, sheet int, headerRow *int) (*sheetData, error) {
	book, err := dataimport.Read(filename, data)
	if err != nil {
		return nil, Invalid("%v", err)
	}
	sd := &sheetData{book: book, sheet: sheet}
	if sheet < 0 || sheet >= len(book.Sheets) {
		// The table that looks most like a sales register.
		sd.sheet, sd.header = 0, 0
		best := -1.0
		for i, sh := range book.Sheets {
			hr := dataimport.HeaderRow(sh.Rows)
			score := float64(len(sh.Rows)) / 1e6
			if hr < len(sh.Rows) {
				score += headingScore(sh.Rows[hr])
			}
			if score > best {
				sd.sheet, best = i, score
			}
		}
	}
	rows := book.Sheets[sd.sheet].Rows
	if headerRow != nil && *headerRow >= 0 && *headerRow < len(rows) {
		sd.header = *headerRow
	} else {
		sd.header = dataimport.HeaderRow(rows)
	}
	if sd.header >= len(rows) {
		return nil, Invalid("the table has no rows")
	}
	sd.headers = rows[sd.header]
	width := len(sd.headers)
	for _, r := range rows[sd.header+1:] {
		width = max(width, len(r))
	}
	for len(sd.headers) < width {
		sd.headers = append(sd.headers, "")
	}
	for i, h := range sd.headers {
		if strings.TrimSpace(h) == "" {
			sd.headers[i] = fmt.Sprintf("Column %s", colName(i))
		}
	}
	for i, r := range rows[sd.header+1:] {
		empty := true
		for _, c := range r {
			if strings.TrimSpace(c) != "" {
				empty = false
				break
			}
		}
		if empty || sameRow(r, rows[sd.header]) {
			continue
		}
		if len(sd.rows) >= dataimport.MaxRows {
			break
		}
		cells := make([]string, width)
		copy(cells, r)
		sd.rows = append(sd.rows, cells)
		sd.rowNos = append(sd.rowNos, sd.header+i+2)
	}
	return sd, nil
}

func headingScore(cells []string) float64 {
	n := 0.0
	for _, m := range dataimport.Suggest(cells, nil) {
		n += m.Score
	}
	return n
}

func sameRow(a, b []string) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

// colName is the spreadsheet letter of a column (0 → A, 26 → AA).
func colName(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// isTotalRow recognises the totals line at the foot of a register.
func isTotalRow(r []string) bool {
	for _, c := range r {
		if strings.TrimSpace(c) != "" {
			return totalRow.MatchString(c)
		}
	}
	return false
}

// AnalyzeImport reads an uploaded file and proposes how to import it.
// With a mapping (the user's own matching) the missing details are worked
// out for that matching instead of the suggested one.
func (s *Service) AnalyzeImport(ctx context.Context, companyID int64, filename string, data []byte, sheet int, headerRow *int, mapping map[string]int) (*ImportAnalysis, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	sd, err := loadSheet(filename, data, sheet, headerRow)
	if err != nil {
		return nil, err
	}
	a := &ImportAnalysis{FileName: filename, Format: sd.book.Format, Kind: sd.book.Kind, Notes: sd.book.Notes, Sheet: sd.sheet,
		HeaderRow: sd.header, Fields: dataimport.Fields, Mapping: map[string]int{}}
	for _, sh := range sd.book.Sheets {
		a.Sheets = append(a.Sheets, ImportSheet{Name: sh.Name, Rows: len(sh.Rows)})
	}
	samples := make([][]string, len(sd.headers))
	for _, r := range sd.rows {
		if isTotalRow(r) {
			continue
		}
		for i := range sd.headers {
			if v := strings.TrimSpace(r[i]); v != "" && len(samples[i]) < 40 {
				samples[i] = append(samples[i], v)
			}
		}
	}
	matches := dataimport.Suggest(sd.headers, samples)
	byCol := map[int]dataimport.Match{}
	if mapping != nil {
		for f, col := range mapping {
			if _, ok := dataimport.FieldByKey(f); ok && col >= 0 && col < len(sd.headers) {
				byCol[col] = dataimport.Match{Col: col, Field: f, Score: 1, By: "you"}
				a.Mapping[f] = col
			}
		}
	} else {
		for _, m := range matches {
			byCol[m.Col] = m
			a.Mapping[m.Field] = m.Col
		}
	}
	for i, h := range sd.headers {
		col := ImportColumn{Index: i, Header: h, Samples: samples[i][:min(len(samples[i]), 5)]}
		if m, ok := byCol[i]; ok {
			col.Field, col.Score, col.By = m.Field, m.Score, m.By
		}
		a.Columns = append(a.Columns, col)
	}
	for _, r := range sd.rows {
		if !isTotalRow(r) {
			a.Rows++
		}
	}
	for _, r := range sd.rows[:min(len(sd.rows), 8)] {
		a.Preview = append(a.Preview, r)
	}
	a.Lists = s.importLists(ctx, c)
	a.Missing = s.missingInfo(ctx, c, sd, a.Mapping)
	return a, nil
}

func (s *Service) importLists(ctx context.Context, c *store.Company) ImportLists {
	l := ImportLists{UOMs: s.UOMs(ctx, c.Environment), Rates: []string{"18%", "17%", "16%", "15%", "10%", "8%", "5%", "3%", "2%", "1%", "0%", "Exempt"}}
	for _, p := range s.Provinces(ctx, c.Environment) {
		l.Provinces = append(l.Provinces, p.Name)
	}
	for _, st := range s.SaleTypes(ctx, c.Environment) {
		l.SaleTypes = append(l.SaleTypes, st.Description)
	}
	return l
}

// missingInfo lists what FBR needs that the file does not supply.
func (s *Service) missingInfo(ctx context.Context, c *store.Company, sd *sheetData, mapping map[string]int) []MissingInfo {
	var out []MissingInfo
	mapped := func(f string) bool {
		i, ok := mapping[f]
		return ok && i >= 0 && i < len(sd.headers)
	}
	blank := func(f string) int {
		if !mapped(f) {
			return len(sd.rows)
		}
		n := 0
		for _, r := range sd.rows {
			if !isTotalRow(r) && strings.TrimSpace(r[mapping[f]]) == "" {
				n++
			}
		}
		return n
	}
	label := func(f string) string {
		fd, _ := dataimport.FieldByKey(f)
		return fd.Label
	}
	add := func(m MissingInfo) {
		if m.Label == "" {
			m.Label = label(m.Field)
		}
		out = append(out, m)
	}
	n := 0
	for _, r := range sd.rows {
		if !isTotalRow(r) {
			n++
		}
	}
	if n == 0 {
		return []MissingInfo{{Field: "rows", Label: "Data rows", Level: "fix", Message: "No data rows were found under the headings. Check the heading row and the table chosen."}}
	}

	// Which lines form one invoice.
	if !mapped("invoice_ref") {
		g := "row"
		if mapped("buyer_name") || mapped("buyer_ntn_cnic") {
			g = "buyer-date"
		}
		add(MissingInfo{Field: "invoice_ref", Level: "recommended", Input: "grouping", Suggest: g,
			Message: "Your file has no invoice number. Choose which lines make up one invoice."})
	}
	today := s.Today()
	if b := blank("invoice_date"); b > 0 {
		msg := "Your file has no invoice date. Enter the date of supply for these invoices."
		if mapped("invoice_date") {
			msg = fmt.Sprintf("%s have no date. Enter the date to use for them.", plural(b, "line", "lines"))
		}
		add(MissingInfo{Field: "invoice_date", Level: "required", Blank: b, Input: "date", Suggest: today, Message: msg})
	}
	if mapped("invoice_date") {
		bad, example := 0, ""
		for _, r := range sd.rows {
			v := strings.TrimSpace(r[mapping["invoice_date"]])
			if v != "" && !isTotalRow(r) {
				if _, ok := dataimport.Date(v); !ok {
					bad++
					example = v
				}
			}
		}
		if bad > 0 {
			add(MissingInfo{Field: "invoice_date", Level: "fix", Blank: bad,
				Message: fmt.Sprintf("%s could not be read as dates (for example “%s”). Use day/month/year or year-month-day.", plural(bad, "date", "dates"), example)})
		}
	}

	// What was sold.
	products, _, _ := s.Store.ListProducts(ctx, c.ID, store.ListParams{Limit: 5000})
	byDesc := map[string]*store.Product{}
	byCode := map[string]*store.Product{}
	for _, p := range products {
		byDesc[strings.ToLower(strings.TrimSpace(p.Description))] = p
		if p.Code != "" {
			byCode[strings.ToLower(p.Code)] = p
		}
	}
	known := 0 // lines whose product is in the master
	for _, r := range sd.rows {
		if isTotalRow(r) {
			continue
		}
		if mapped("product_code") && byCode[strings.ToLower(strings.TrimSpace(r[mapping["product_code"]]))] != nil {
			known++
		} else if mapped("description") && byDesc[strings.ToLower(strings.TrimSpace(r[mapping["description"]]))] != nil {
			known++
		}
	}
	fromMaster := ""
	if known > 0 {
		fromMaster = fmt.Sprintf(" %s match your Products master and take its details.", plural(known, "line", "lines"))
	}
	if !mapped("description") && !mapped("product_code") {
		add(MissingInfo{Field: "description", Level: "required", Blank: n, Input: "text", Suggest: "",
			Message: "Your file has no description of what was sold. Enter a description for all lines, or match the right column."})
	} else if b := blank("description"); mapped("description") && b > 0 && !mapped("product_code") {
		add(MissingInfo{Field: "description", Level: "required", Blank: b, Input: "text",
			Message: fmt.Sprintf("%s have no description. Enter one to use for them.", plural(b, "line", "lines"))})
	}
	need := func(field, input, suggest, what string) {
		if b := blank(field); b > 0 && b > known {
			rest := "all lines"
			if known > 0 {
				rest = "the other lines"
			}
			msg := fmt.Sprintf("Your file has no %s.%s Choose the %s to use for %s.", what, fromMaster, what, rest)
			if mapped(field) {
				msg = fmt.Sprintf("%s have no %s.%s Choose the %s to use for them.", plural(b, "line", "lines"), what, fromMaster, what)
			}
			add(MissingInfo{Field: field, Level: "required", Blank: b, Input: input, Suggest: suggest, Message: msg})
		}
	}
	need("hs_code", "hs", "", "HS code")
	need("uom", "uom", "Numbers, pieces, units", "unit of measure")
	need("sale_type", "saletype", string(domain.STStandard), "sale type")
	if !mapped("rate") && mapped("sales_tax") && mapped("value_excl_st") {
		add(MissingInfo{Field: "rate", Level: "recommended", Input: "rate", Suggest: "18%",
			Message: "Your file has no tax rate column; the rate of each line is worked out from its sales tax and value. Lines without both use the rate you choose."})
	} else {
		need("rate", "rate", "18%", "sales tax rate")
	}
	if !mapped("quantity") {
		add(MissingInfo{Field: "quantity", Level: "required", Blank: n, Input: "text", Suggest: "1",
			Message: "Your file has no quantity. Enter the quantity to use for every line (1 if each line is one supply)."})
	}
	if !mapped("unit_price") && !mapped("value_excl_st") {
		add(MissingInfo{Field: "unit_price", Level: "fix", Blank: n,
			Message: "Your file has no price or value of the goods. Match the column that holds the price or the value before sales tax."})
	}

	// Values FBR would not accept, to translate.
	lists := s.importLists(ctx, c)
	unknown := func(field, what string, known func(string) (string, bool)) {
		if !mapped(field) {
			return
		}
		counts := map[string]int{}
		var order []string
		for _, r := range sd.rows {
			v := strings.TrimSpace(r[mapping[field]])
			if v == "" || isTotalRow(r) {
				continue
			}
			if _, ok := known(v); ok {
				continue
			}
			if counts[v] == 0 {
				order = append(order, v)
			}
			counts[v]++
		}
		if len(order) == 0 {
			return
		}
		mi := MissingInfo{Field: field, Level: "fix", Input: "valuemap",
			Message: fmt.Sprintf("%s in your file %s not on FBR's list. Choose FBR's %s for each.", plural(len(order), what, what+"s"), map[bool]string{true: "is", false: "are"}[len(order) == 1], what)}
		for _, v := range order[:min(len(order), 50)] {
			mi.Values = append(mi.Values, ValueCount{Value: v, Lines: counts[v]})
			mi.Blank += counts[v]
		}
		add(mi)
	}
	unknown("uom", "unit", func(v string) (string, bool) { return dataimport.UOM(v, lists.UOMs) })
	unknown("sale_type", "sale type", func(v string) (string, bool) { return dataimport.SaleType(v, lists.SaleTypes) })
	unknown("buyer_province", "province", func(v string) (string, bool) { p := dataimport.Province(v); return p, p != "" })

	// The buyer.
	if b := blank("buyer_province"); b > 0 {
		msg := "Your file has no buyer province (destination of supply)."
		if mapped("buyer_province") {
			msg = fmt.Sprintf("%s have no buyer province.", plural(b, "line", "lines"))
		}
		if mapped("buyer_address") {
			found := 0
			for _, r := range sd.rows {
				if isTotalRow(r) || (mapped("buyer_province") && strings.TrimSpace(r[mapping["buyer_province"]]) != "") {
					continue
				}
				if dataimport.ProvinceFromAddress(r[mapping["buyer_address"]]) != "" {
					found++
				}
			}
			if found > 0 {
				msg += fmt.Sprintf(" It can be worked out from the city in the address for %s.", plural(min(found, b), "line", "lines"))
			}
		}
		add(MissingInfo{Field: "buyer_province", Level: "required", Blank: b, Input: "province", Suggest: c.Province,
			Message: msg + " Choose the province for the rest."})
	}
	if !mapped("buyer_registration_type") {
		add(MissingInfo{Field: "buyer_registration_type", Level: "recommended", Input: "regrule", Suggest: "ntn",
			Message: "Your file does not say which buyers are registered for sales tax. Buyers with an NTN/CNIC are treated as registered unless you choose otherwise; check doubtful buyers with “Verify a buyer”."})
	}
	if mapped("buyer_ntn_cnic") {
		bad, example := 0, ""
		for _, r := range sd.rows {
			if v := r[mapping["buyer_ntn_cnic"]]; dataimport.IsScientific(v) {
				bad++
				example = v
			}
		}
		if bad > 0 {
			add(MissingInfo{Field: "buyer_ntn_cnic", Level: "fix", Blank: bad,
				Message: fmt.Sprintf("%s were turned into numbers by Excel (for example %s) and have lost digits. Format the column as Text in Excel, re-type them and save again.", plural(bad, "NTN/CNIC", "NTNs/CNICs"), example)})
		}
	}
	return out
}

// SmartImport imports a file with the user's matching and defaults.
func (s *Service) SmartImport(ctx context.Context, a Actor, companyID int64, filename string, data []byte, o ImportOptions, preview, submit bool) (*ImportSummary, error) {
	c, err := s.companyFor(ctx, companyID)
	if err != nil {
		return nil, err
	}
	sd, err := loadSheet(filename, data, o.Sheet, o.HeaderRow)
	if err != nil {
		return nil, err
	}
	rows, err := s.mapRows(ctx, c, sd, o, data)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, Invalid("no data rows to import")
	}
	return s.Import(ctx, a, companyID, rows, preview, submit)
}

var standardRates = []float64{18, 17, 16, 15, 10, 8, 5, 3, 2, 1, 0.5, 0}

// mapRows turns the file's rows into import rows keyed by field.
func (s *Service) mapRows(ctx context.Context, c *store.Company, sd *sheetData, o ImportOptions, data []byte) ([]map[string]string, error) {
	col := func(f string) int {
		if i, ok := o.Mapping[f]; ok && i >= 0 && i < len(sd.headers) {
			return i
		}
		return -1
	}
	products, _, _ := s.Store.ListProducts(ctx, c.ID, store.ListParams{Limit: 5000})
	byDesc := map[string]*store.Product{}
	for _, p := range products {
		if p.Code != "" {
			byDesc[strings.ToLower(strings.TrimSpace(p.Description))] = p
		}
	}
	customers, _, _ := s.Store.ListCustomers(ctx, c.ID, store.ListParams{Limit: 5000})
	custByName := map[string]*store.Customer{}
	for _, cu := range customers {
		if cu.NTNCNIC != "" {
			custByName[strings.ToLower(strings.TrimSpace(cu.Name))] = cu
		}
	}
	lists := s.importLists(ctx, c)
	sum := sha256.Sum256(data)
	fileTag := strings.ToUpper(hex.EncodeToString(sum[:])[:8])
	var out []map[string]string
	for i, r := range sd.rows {
		if isTotalRow(r) {
			continue
		}
		m := map[string]string{"_row": fmt.Sprint(sd.rowNos[i])}
		for _, f := range dataimport.Fields {
			if j := col(f.Key); j >= 0 {
				m[f.Key] = strings.TrimSpace(r[j])
				if to, ok := o.ValueMap[f.Key][m[f.Key]]; ok && m[f.Key] != "" {
					m[f.Key] = strings.TrimSpace(to)
				}
			}
		}
		// Defaults fill blanks; the province and the rate are first worked
		// out from the address and from the tax and value.
		late := map[string]bool{"buyer_province": true, "rate": true, "buyer_registration_type": true}
		for k, v := range o.Defaults {
			if !late[k] && strings.TrimSpace(m[k]) == "" && strings.TrimSpace(v) != "" {
				m[k] = strings.TrimSpace(v)
			}
		}
		// Normalise what spreadsheets and accounting exports write.
		if v := m["invoice_date"]; v != "" {
			if d, ok := dataimport.Date(v); ok {
				m["invoice_date"] = d
			}
		}
		for _, k := range []string{"unit_price", "discount", "value_excl_st", "retail_price", "sales_tax", "further_tax", "extra_tax", "fed", "st_withheld", "fed_unit_price", "total_value"} {
			if v := m[k]; v != "" {
				if d, ok := dataimport.Number(v); ok {
					m[k] = d.String()
				}
			}
		}
		if v := m["quantity"]; v != "" {
			if q, unit, ok := dataimport.Quantity(v); ok {
				m["quantity"] = q.String()
				if unit != "" && m["uom"] == "" {
					m["uom"] = unit
				}
			}
		}
		if v := m["rate"]; v != "" {
			m["rate"] = dataimport.Rate(v)
		} else if m["sales_tax"] != "" && m["value_excl_st"] != "" {
			// The rate implied by the tax and the value.
			st, _ := decimal.NewFromString(m["sales_tax"])
			val, _ := decimal.NewFromString(m["value_excl_st"])
			if val.IsPositive() {
				pct, _ := st.Div(val).Mul(decimal.NewFromInt(100)).Float64()
				for _, sr := range standardRates {
					if pct > sr-0.06 && pct < sr+0.06 {
						m["rate"] = decimal.NewFromFloat(sr).String() + "%"
						break
					}
				}
			}
		}
		if v := m["doc_type"]; v != "" {
			m["doc_type"] = dataimport.DocType(v)
		}
		if u, ok := dataimport.UOM(m["uom"], lists.UOMs); ok {
			m["uom"] = u
		}
		if st, ok := dataimport.SaleType(m["sale_type"], lists.SaleTypes); ok {
			m["sale_type"] = st
		}
		if v := m["buyer_ntn_cnic"]; v != "" {
			m["buyer_ntn_cnic"] = dataimport.NTN(v)
		}
		if m["buyer_ntn_cnic"] == "" && m["buyer_name"] != "" {
			if cu := custByName[strings.ToLower(m["buyer_name"])]; cu != nil {
				m["buyer_ntn_cnic"] = cu.NTNCNIC
			}
		}
		if v := m["buyer_province"]; v != "" {
			if p := dataimport.Province(v); p != "" {
				m["buyer_province"] = p
			}
		}
		if m["buyer_province"] == "" && o.ProvinceFromAddress {
			m["buyer_province"] = dataimport.ProvinceFromAddress(m["buyer_address"])
		}
		for _, k := range []string{"buyer_province", "rate"} {
			if m[k] == "" {
				m[k] = strings.TrimSpace(o.Defaults[k])
			}
		}
		if v := m["buyer_registration_type"]; v != "" {
			if rt := dataimport.RegType(v); rt != "" {
				m["buyer_registration_type"] = rt
			}
		} else {
			switch o.RegRule {
			case "", "ntn":
				// Left blank, the customer master decides for known buyers.
				if m["buyer_ntn_cnic"] != "" && custByName[strings.ToLower(m["buyer_name"])] == nil {
					m["buyer_registration_type"] = string(domain.Registered)
				}
			default:
				m["buyer_registration_type"] = o.RegRule
			}
		}
		if v := m["advance_receipt"]; v != "" {
			if yes, ok := dataimport.YesNo(v); ok {
				m["advance_receipt"] = map[bool]string{true: "yes", false: "no"}[yes]
			}
		}
		// A description matching the Products master brings its details.
		if m["product_code"] == "" && m["hs_code"] == "" {
			if p := byDesc[strings.ToLower(m["description"])]; p != nil {
				m["product_code"] = p.Code
			}
		}
		if m["quantity"] == "" && m["value_excl_st"] != "" {
			m["quantity"] = "1"
		}
		if m["unit_price"] == "" && m["value_excl_st"] != "" {
			if q, err := decimal.NewFromString(m["quantity"]); err == nil && q.IsPositive() {
				v, _ := decimal.NewFromString(m["value_excl_st"])
				d, _ := decimal.NewFromString(nzs(m["discount"], "0"))
				m["unit_price"] = v.Add(d).Div(q).Round(4).String()
			}
		}
		// Lines without an invoice number are grouped as the user chose.
		if m["invoice_ref"] == "" {
			switch o.Grouping {
			case "single":
				m["invoice_ref"] = "IMP-" + fileTag
			case "buyer-date":
				buyer := nzs(m["buyer_ntn_cnic"], strings.ToUpper(m["buyer_name"]))
				key := sha256.Sum256([]byte(m["invoice_date"] + "|" + buyer))
				m["invoice_ref"] = "IMP-" + fileTag + "-" + strings.ToUpper(hex.EncodeToString(key[:])[:6])
			default:
				m["invoice_ref"] = fmt.Sprintf("IMP-%s-R%s", fileTag, m["_row"])
			}
		}
		out = append(out, m)
	}
	return out, nil
}

func nzs(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
}
