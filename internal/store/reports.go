package store

import (
	"context"
	"encoding/json"
	"sort"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"

	"github.com/shopspring/decimal"
)

// ReportFilter scopes report queries.
type ReportFilter struct {
	CompanyID   int64
	Environment domain.Environment
	From, To    string
}

// RegisterLine is one invoice line in the sales register (the same
// particulars as the DI payload / Annexure-C of the sales tax return).
type RegisterLine struct {
	InvoiceID         int64           `json:"invoiceId"`
	DocType           string          `json:"docType"`
	InternalNo        string          `json:"internalNo"`
	FBRInvoiceNumber  string          `json:"fbrInvoiceNumber"`
	InvoiceDate       string          `json:"invoiceDate"`
	InvoiceRefNo      string          `json:"invoiceRefNo"`
	Status            string          `json:"status"`
	BuyerNTNCNIC      string          `json:"buyerNtnCnic"`
	BuyerName         string          `json:"buyerName"`
	BuyerRegistration string          `json:"buyerRegistrationType"`
	SellerProvince    string          `json:"sellerProvince"`
	BuyerProvince     string          `json:"buyerProvince"`
	LineNo            int             `json:"lineNo"`
	HSCode            string          `json:"hsCode"`
	Description       string          `json:"description"`
	SaleType          string          `json:"saleType"`
	Rate              string          `json:"rate"`
	Quantity          decimal.Decimal `json:"quantity"`
	UoM               string          `json:"uom"`
	ValueExclST       decimal.Decimal `json:"valueExclST"`
	RetailValue       decimal.Decimal `json:"retailValue"`
	SalesTax          decimal.Decimal `json:"salesTax"`
	ExtraTax          decimal.Decimal `json:"extraTax"`
	FurtherTax        decimal.Decimal `json:"furtherTax"`
	FED               decimal.Decimal `json:"fed"`
	STWithheld        decimal.Decimal `json:"stWithheld"`
	Discount          decimal.Decimal `json:"discount"`
	TotalValue        decimal.Decimal `json:"totalValue"`
	SROScheduleNo     string          `json:"sroScheduleNo"`
	SROItemSerialNo   string          `json:"sroItemSerialNo"`
}

func reportWhere(f ReportFilter) (string, []any) {
	w := `i.company_id=? AND i.environment=? AND i.status='ACCEPTED'`
	args := []any{f.CompanyID, string(f.Environment)}
	if f.From != "" {
		w += ` AND i.invoice_date>=?`
		args = append(args, f.From)
	}
	if f.To != "" {
		w += ` AND i.invoice_date<=?`
		args = append(args, f.To)
	}
	return w, args
}

// SalesRegister returns accepted invoice lines in the period.
func (s *Store) SalesRegister(ctx context.Context, f ReportFilter) ([]RegisterLine, error) {
	w, args := reportWhere(f)
	rows, err := s.DB.QueryContext(ctx, `SELECT i.id, i.doc_type, i.internal_no, i.fbr_invoice_number, i.invoice_date, i.invoice_ref_no, i.status,
		i.buyer_ntn_cnic, i.buyer_name, i.buyer_registration_type, i.seller_province, i.buyer_province,
		it.line_no, it.hs_code, it.description, it.sale_type, it.rate, it.quantity, it.uom, it.value_excl_st, it.retail_value,
		it.sales_tax, it.extra_tax, it.further_tax, it.fed, it.st_withheld, it.discount, it.total_value, it.sro_schedule_no, it.sro_item_serial_no
		FROM invoices i JOIN invoice_items it ON it.invoice_id=i.id WHERE `+w+` ORDER BY i.invoice_date, i.id, it.line_no`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RegisterLine
	for rows.Next() {
		var l RegisterLine
		var qty string
		var v, rv, st, et, ft, fed, wh, d, tv int64
		if err := rows.Scan(&l.InvoiceID, &l.DocType, &l.InternalNo, &l.FBRInvoiceNumber, &l.InvoiceDate, &l.InvoiceRefNo, &l.Status,
			&l.BuyerNTNCNIC, &l.BuyerName, &l.BuyerRegistration, &l.SellerProvince, &l.BuyerProvince,
			&l.LineNo, &l.HSCode, &l.Description, &l.SaleType, &l.Rate, &qty, &l.UoM, &v, &rv, &st, &et, &ft, &fed, &wh, &d, &tv,
			&l.SROScheduleNo, &l.SROItemSerialNo); err != nil {
			return nil, err
		}
		l.Quantity = dec(qty)
		l.ValueExclST, l.RetailValue, l.SalesTax, l.ExtraTax = Rupees(v), Rupees(rv), Rupees(st), Rupees(et)
		l.FurtherTax, l.FED, l.STWithheld, l.Discount, l.TotalValue = Rupees(ft), Rupees(fed), Rupees(wh), Rupees(d), Rupees(tv)
		out = append(out, l)
	}
	return out, rows.Err()
}

// TaxSummaryRow aggregates by document type, sale type and rate.
type TaxSummaryRow struct {
	DocType     string          `json:"docType"`
	SaleType    string          `json:"saleType"`
	Rate        string          `json:"rate"`
	Invoices    int             `json:"invoices"`
	Lines       int             `json:"lines"`
	ValueExclST decimal.Decimal `json:"valueExclST"`
	RetailValue decimal.Decimal `json:"retailValue"`
	SalesTax    decimal.Decimal `json:"salesTax"`
	FurtherTax  decimal.Decimal `json:"furtherTax"`
	ExtraTax    decimal.Decimal `json:"extraTax"`
	FED         decimal.Decimal `json:"fed"`
	STWithheld  decimal.Decimal `json:"stWithheld"`
}

// TaxSummary groups accepted lines by document type, sale type and rate.
func (s *Store) TaxSummary(ctx context.Context, f ReportFilter) ([]TaxSummaryRow, error) {
	w, args := reportWhere(f)
	rows, err := s.DB.QueryContext(ctx, `SELECT i.doc_type, it.sale_type, it.rate, COUNT(DISTINCT i.id), COUNT(*),
		SUM(it.value_excl_st), SUM(it.retail_value), SUM(it.sales_tax), SUM(it.further_tax), SUM(it.extra_tax), SUM(it.fed), SUM(it.st_withheld)
		FROM invoices i JOIN invoice_items it ON it.invoice_id=i.id WHERE `+w+`
		GROUP BY i.doc_type, it.sale_type, it.rate ORDER BY i.doc_type DESC, it.sale_type, it.rate`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TaxSummaryRow
	for rows.Next() {
		var r TaxSummaryRow
		var v, rv, st, ft, et, fed, wh int64
		if err := rows.Scan(&r.DocType, &r.SaleType, &r.Rate, &r.Invoices, &r.Lines, &v, &rv, &st, &ft, &et, &fed, &wh); err != nil {
			return nil, err
		}
		r.ValueExclST, r.RetailValue, r.SalesTax, r.FurtherTax = Rupees(v), Rupees(rv), Rupees(st), Rupees(ft)
		r.ExtraTax, r.FED, r.STWithheld = Rupees(et), Rupees(fed), Rupees(wh)
		out = append(out, r)
	}
	return out, rows.Err()
}

// PeriodRow aggregates by month.
type PeriodRow struct {
	Period          string          `json:"period"` // YYYY-MM
	SaleInvoices    int             `json:"saleInvoices"`
	DebitNotes      int             `json:"debitNotes"`
	ValueExclST     decimal.Decimal `json:"valueExclST"`
	SalesTax        decimal.Decimal `json:"salesTax"`
	FurtherTax      decimal.Decimal `json:"furtherTax"`
	DebitValue      decimal.Decimal `json:"debitNoteValue"`
	DebitSalesTax   decimal.Decimal `json:"debitNoteSalesTax"`
	STWithheld      decimal.Decimal `json:"stWithheld"`
}

// MonthlySummary aggregates accepted documents per calendar month (tax period).
func (s *Store) MonthlySummary(ctx context.Context, f ReportFilter) ([]PeriodRow, error) {
	w, args := reportWhere(f)
	rows, err := s.DB.QueryContext(ctx, `SELECT substr(i.invoice_date,1,7) AS p,
		SUM(CASE WHEN i.doc_type='Sale Invoice' THEN 1 ELSE 0 END), SUM(CASE WHEN i.doc_type='Debit Note' THEN 1 ELSE 0 END),
		SUM(CASE WHEN i.doc_type='Sale Invoice' THEN i.total_value_excl_st ELSE 0 END),
		SUM(CASE WHEN i.doc_type='Sale Invoice' THEN i.total_sales_tax ELSE 0 END),
		SUM(CASE WHEN i.doc_type='Sale Invoice' THEN i.total_further_tax ELSE 0 END),
		SUM(CASE WHEN i.doc_type='Debit Note' THEN i.total_value_excl_st ELSE 0 END),
		SUM(CASE WHEN i.doc_type='Debit Note' THEN i.total_sales_tax ELSE 0 END),
		SUM(i.total_st_withheld)
		FROM invoices i WHERE `+w+` GROUP BY p ORDER BY p`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PeriodRow
	for rows.Next() {
		var r PeriodRow
		var v, st, ft, dv, dst, wh int64
		if err := rows.Scan(&r.Period, &r.SaleInvoices, &r.DebitNotes, &v, &st, &ft, &dv, &dst, &wh); err != nil {
			return nil, err
		}
		r.ValueExclST, r.SalesTax, r.FurtherTax, r.DebitValue, r.DebitSalesTax, r.STWithheld = Rupees(v), Rupees(st), Rupees(ft), Rupees(dv), Rupees(dst), Rupees(wh)
		out = append(out, r)
	}
	return out, rows.Err()
}

// CustomerRow aggregates by buyer.
type CustomerRow struct {
	BuyerNTNCNIC      string          `json:"buyerNtnCnic"`
	BuyerName         string          `json:"buyerName"`
	BuyerRegistration string          `json:"buyerRegistrationType"`
	Invoices          int             `json:"invoices"`
	ValueExclST       decimal.Decimal `json:"valueExclST"`
	SalesTax          decimal.Decimal `json:"salesTax"`
	FurtherTax        decimal.Decimal `json:"furtherTax"`
	STWithheld        decimal.Decimal `json:"stWithheld"`
	TotalValue        decimal.Decimal `json:"totalValue"`
}

// CustomerSummary aggregates accepted sale invoices per buyer.
func (s *Store) CustomerSummary(ctx context.Context, f ReportFilter) ([]CustomerRow, error) {
	w, args := reportWhere(f)
	rows, err := s.DB.QueryContext(ctx, `SELECT i.buyer_ntn_cnic, MAX(i.buyer_name), MAX(i.buyer_registration_type), COUNT(*),
		SUM(i.total_value_excl_st), SUM(i.total_sales_tax), SUM(i.total_further_tax), SUM(i.total_st_withheld), SUM(i.total_value)
		FROM invoices i WHERE `+w+` AND i.doc_type='Sale Invoice' GROUP BY i.buyer_ntn_cnic, CASE WHEN i.buyer_ntn_cnic='' THEN i.buyer_name ELSE '' END
		ORDER BY SUM(i.total_value) DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CustomerRow
	for rows.Next() {
		var r CustomerRow
		var v, st, ft, wh, tv int64
		if err := rows.Scan(&r.BuyerNTNCNIC, &r.BuyerName, &r.BuyerRegistration, &r.Invoices, &v, &st, &ft, &wh, &tv); err != nil {
			return nil, err
		}
		r.ValueExclST, r.SalesTax, r.FurtherTax, r.STWithheld, r.TotalValue = Rupees(v), Rupees(st), Rupees(ft), Rupees(wh), Rupees(tv)
		out = append(out, r)
	}
	return out, rows.Err()
}

// StatusCount is the number of invoices per status.
type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// Dashboard summarises the current state for a company and environment.
type Dashboard struct {
	StatusCounts   []StatusCount   `json:"statusCounts"`
	TodayCount     int             `json:"todayCount"`
	TodayValue     decimal.Decimal `json:"todayValue"`
	TodaySalesTax  decimal.Decimal `json:"todaySalesTax"`
	MonthCount     int             `json:"monthCount"`
	MonthValue     decimal.Decimal `json:"monthValue"`
	MonthSalesTax  decimal.Decimal `json:"monthSalesTax"`
	NeedsAttention int             `json:"needsAttention"`
	TopErrors      []ErrorCount    `json:"topErrors"`
}

// ErrorCount counts an FBR error code.
type ErrorCount struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// GetDashboard computes dashboard figures.
func (s *Store) GetDashboard(ctx context.Context, companyID int64, env domain.Environment, today, monthStart string) (*Dashboard, error) {
	d := &Dashboard{}
	rows, err := s.DB.QueryContext(ctx, `SELECT status, COUNT(*) FROM invoices WHERE company_id=? AND environment=? GROUP BY status ORDER BY status`, companyID, string(env))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sc StatusCount
		if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
			rows.Close()
			return nil, err
		}
		d.StatusCounts = append(d.StatusCounts, sc)
		switch domain.InvoiceStatus(sc.Status) {
		case domain.StatusRejected, domain.StatusUncertain, domain.StatusQueued:
			d.NeedsAttention += sc.Count
		}
	}
	rows.Close()
	var tv, tst, mv, mst int64
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(total_value_excl_st),0), COALESCE(SUM(total_sales_tax),0) FROM invoices
		WHERE company_id=? AND environment=? AND status='ACCEPTED' AND doc_type='Sale Invoice' AND invoice_date=?`, companyID, string(env), today).Scan(&d.TodayCount, &tv, &tst); err != nil {
		return nil, err
	}
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(total_value_excl_st),0), COALESCE(SUM(total_sales_tax),0) FROM invoices
		WHERE company_id=? AND environment=? AND status='ACCEPTED' AND doc_type='Sale Invoice' AND invoice_date>=?`, companyID, string(env), monthStart).Scan(&d.MonthCount, &mv, &mst); err != nil {
		return nil, err
	}
	d.TodayValue, d.TodaySalesTax, d.MonthValue, d.MonthSalesTax = Rupees(tv), Rupees(tst), Rupees(mv), Rupees(mst)

	// Top FBR error codes over rejected invoices.
	erows, err := s.DB.QueryContext(ctx, `SELECT fbr_errors FROM invoices WHERE company_id=? AND environment=? AND fbr_errors<>'' ORDER BY id DESC LIMIT 500`, companyID, string(env))
	if err != nil {
		return nil, err
	}
	counts := map[string]*ErrorCount{}
	for erows.Next() {
		var js string
		if err := erows.Scan(&js); err != nil {
			erows.Close()
			return nil, err
		}
		var items []fbr.ErrorItem
		_ = json.Unmarshal([]byte(js), &items)
		for _, e := range items {
			k := e.Code
			if k == "" {
				k = e.Message
			}
			if c, ok := counts[k]; ok {
				c.Count++
			} else {
				counts[k] = &ErrorCount{Code: e.Code, Message: e.Message, Count: 1}
			}
		}
	}
	erows.Close()
	for _, c := range counts {
		d.TopErrors = append(d.TopErrors, *c)
	}
	sort.Slice(d.TopErrors, func(i, j int) bool { return d.TopErrors[i].Count > d.TopErrors[j].Count })
	if len(d.TopErrors) > 5 {
		d.TopErrors = d.TopErrors[:5]
	}
	return d, nil
}
