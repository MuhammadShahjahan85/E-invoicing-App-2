// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/validate"

	"github.com/shopspring/decimal"
)

// Totals are invoice totals in rupees.
type Totals struct {
	Gross         decimal.Decimal `json:"gross"`
	Discount      decimal.Decimal `json:"discount"`
	ValueExclST   decimal.Decimal `json:"valueExclST"`
	RetailValue   decimal.Decimal `json:"retailValue"`
	SalesTax      decimal.Decimal `json:"salesTax"`
	FurtherTax    decimal.Decimal `json:"furtherTax"`
	ExtraTax      decimal.Decimal `json:"extraTax"`
	FED           decimal.Decimal `json:"fed"`
	STWithheld    decimal.Decimal `json:"stWithheld"`
	TotalValue    decimal.Decimal `json:"totalValue"`
	AmountPayable decimal.Decimal `json:"amountPayable"`
}

// Invoice is a sale invoice or debit note.
type Invoice struct {
	ID                    int64                   `json:"id"`
	CompanyID             int64                   `json:"companyId"`
	Environment           domain.Environment      `json:"environment"`
	DocType               domain.DocType          `json:"docType"`
	InternalNo            string                  `json:"internalNo"`
	InvoiceDate           string                  `json:"invoiceDate"`
	Status                domain.InvoiceStatus    `json:"status"`
	CustomerID            *int64                  `json:"customerId"`
	SellerNTNCNIC         string                  `json:"sellerNtnCnic"`
	SellerName            string                  `json:"sellerName"`
	SellerProvince        string                  `json:"sellerProvince"`
	SellerAddress         string                  `json:"sellerAddress"`
	BuyerNTNCNIC          string                  `json:"buyerNtnCnic"`
	BuyerName             string                  `json:"buyerName"`
	BuyerProvince         string                  `json:"buyerProvince"`
	BuyerAddress          string                  `json:"buyerAddress"`
	BuyerRegistrationType domain.RegistrationType `json:"buyerRegistrationType"`
	WithholdingMode       string                  `json:"withholdingMode"`
	InvoiceRefNo          string                  `json:"invoiceRefNo"`
	RefInvoiceID          *int64                  `json:"refInvoiceId"`
	ScenarioID            string                  `json:"scenarioId"`
	ExternalRef           string                  `json:"externalRef"`
	Source                string                  `json:"source"`
	Notes                 string                  `json:"notes"`
	Totals                Totals                  `json:"totals"`
	FBRInvoiceNumber      string                  `json:"fbrInvoiceNumber"`
	FBRDated              string                  `json:"fbrDated"`
	FBRStatusCode         string                  `json:"fbrStatusCode"`
	FBRErrors             []fbr.ErrorItem         `json:"fbrErrors"`
	LastError             string                  `json:"lastError"`
	Validation            []validate.Issue        `json:"validation"`
	SubmitAttempts        int                     `json:"submitAttempts"`
	NextAttemptAt         string                  `json:"nextAttemptAt"`
	PayloadJSON           string                  `json:"-"`
	PayloadHash           string                  `json:"payloadHash"`
	SealHash              string                  `json:"sealHash"`
	PrevSealHash          string                  `json:"prevSealHash"`
	OfflineSince          string                  `json:"offlineSince"`
	PrintCount            int                     `json:"printCount"`
	CancelledAt           string                  `json:"cancelledAt"`
	CancelReason          string                  `json:"cancelReason"`
	CancelReference       string                  `json:"cancelReference"`
	CreatedBy             *int64                  `json:"createdBy"`
	UpdatedBy             *int64                  `json:"updatedBy"`
	CreatedAt             string                  `json:"createdAt"`
	UpdatedAt             string                  `json:"updatedAt"`
	SubmittedAt           string                  `json:"submittedAt"`
	AcceptedAt            string                  `json:"acceptedAt"`
	Items                 []*InvoiceItem          `json:"items,omitempty"`
}

// InvoiceItem is one invoice line: inputs (as entered) plus computed amounts.
type InvoiceItem struct {
	ID                  int64            `json:"id"`
	InvoiceID           int64            `json:"invoiceId"`
	LineNo              int              `json:"lineNo"`
	ProductID           *int64           `json:"productId"`
	HSCode              string           `json:"hsCode"`
	Description         string           `json:"description"`
	UoM                 string           `json:"uom"`
	Quantity            decimal.Decimal  `json:"quantity"`
	UnitPrice           decimal.Decimal  `json:"unitPrice"`
	DiscountPercent     decimal.Decimal  `json:"discountPercent"`
	DiscountAmount      decimal.Decimal  `json:"discountAmount"`
	ValueOverride       *decimal.Decimal `json:"valueOverride"`
	SaleType            string           `json:"saleType"`
	Rate                string           `json:"rate"`
	RetailPrice         decimal.Decimal  `json:"retailPrice"`
	RetailValueOverride *decimal.Decimal `json:"retailValueOverride"`
	FurtherTaxMode      string           `json:"furtherTaxMode"`
	FurtherTaxOverride  *decimal.Decimal `json:"furtherTaxOverride"`
	ExtraTaxRate        decimal.Decimal  `json:"extraTaxRate"`
	ExtraTaxOverride    *decimal.Decimal `json:"extraTaxOverride"`
	FEDRate             decimal.Decimal  `json:"fedRate"`
	FEDOverride         *decimal.Decimal `json:"fedOverride"`
	WithholdingOverride *decimal.Decimal `json:"withholdingOverride"`
	SalesTaxOverride    *decimal.Decimal `json:"salesTaxOverride"`
	SROScheduleNo       string           `json:"sroScheduleNo"`
	SROItemSerialNo     string           `json:"sroItemSerialNo"`
	// Computed.
	Gross            decimal.Decimal `json:"gross"`
	Discount         decimal.Decimal `json:"discount"`
	ValueExclST      decimal.Decimal `json:"valueExclST"`
	RetailValue      decimal.Decimal `json:"retailValue"`
	SalesTax         decimal.Decimal `json:"salesTax"`
	FurtherTax       decimal.Decimal `json:"furtherTax"`
	ExtraTax         decimal.Decimal `json:"extraTax"`
	ExtraTaxEmpty    bool            `json:"extraTaxEmpty"`
	FED              decimal.Decimal `json:"fed"`
	STWithheld       decimal.Decimal `json:"stWithheld"`
	TotalValue       decimal.Decimal `json:"totalValue"`
	FBRItemInvoiceNo string          `json:"fbrItemInvoiceNo"`
	FBRStatusCode    string          `json:"fbrStatusCode"`
	FBRErrorCode     string          `json:"fbrErrorCode"`
	FBRError         string          `json:"fbrError"`
	Warnings         []string        `json:"warnings,omitempty"`
}

const invoiceCols = `id, company_id, environment, doc_type, internal_no, invoice_date, status, customer_id, seller_ntn_cnic, seller_name,
	seller_province, seller_address, buyer_ntn_cnic, buyer_name, buyer_province, buyer_address, buyer_registration_type, withholding_mode,
	invoice_ref_no, ref_invoice_id, scenario_id, external_ref, source, notes, total_gross, total_discount, total_value_excl_st,
	total_retail_value, total_sales_tax, total_further_tax, total_extra_tax, total_fed, total_st_withheld, total_value, amount_payable,
	fbr_invoice_number, fbr_dated, fbr_status_code, fbr_errors, last_error, validation_json, submit_attempts, next_attempt_at,
	payload_json, payload_hash, seal_hash, prev_seal_hash, print_count, cancelled_at, cancel_reason, cancel_reference,
	created_by, updated_by, created_at, updated_at, submitted_at, accepted_at, offline_since`

func scanInvoice(row interface{ Scan(...any) error }) (*Invoice, error) {
	var inv Invoice
	var env, dt, st, rt, fbrErrs, valJSON string
	var cust, ref, cb, ub sql.NullInt64
	var g, d, v, rv, stx, ft, et, fed, wh, tv, ap int64
	err := row.Scan(&inv.ID, &inv.CompanyID, &env, &dt, &inv.InternalNo, &inv.InvoiceDate, &st, &cust, &inv.SellerNTNCNIC, &inv.SellerName,
		&inv.SellerProvince, &inv.SellerAddress, &inv.BuyerNTNCNIC, &inv.BuyerName, &inv.BuyerProvince, &inv.BuyerAddress, &rt, &inv.WithholdingMode,
		&inv.InvoiceRefNo, &ref, &inv.ScenarioID, &inv.ExternalRef, &inv.Source, &inv.Notes, &g, &d, &v,
		&rv, &stx, &ft, &et, &fed, &wh, &tv, &ap,
		&inv.FBRInvoiceNumber, &inv.FBRDated, &inv.FBRStatusCode, &fbrErrs, &inv.LastError, &valJSON, &inv.SubmitAttempts, &inv.NextAttemptAt,
		&inv.PayloadJSON, &inv.PayloadHash, &inv.SealHash, &inv.PrevSealHash, &inv.PrintCount, &inv.CancelledAt, &inv.CancelReason, &inv.CancelReference,
		&cb, &ub, &inv.CreatedAt, &inv.UpdatedAt, &inv.SubmittedAt, &inv.AcceptedAt, &inv.OfflineSince)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	inv.Environment, inv.DocType, inv.Status, inv.BuyerRegistrationType = domain.Environment(env), domain.DocType(dt), domain.InvoiceStatus(st), domain.RegistrationType(rt)
	inv.CustomerID, inv.RefInvoiceID, inv.CreatedBy, inv.UpdatedBy = intPtr(cust), intPtr(ref), intPtr(cb), intPtr(ub)
	inv.Totals = Totals{Gross: Rupees(g), Discount: Rupees(d), ValueExclST: Rupees(v), RetailValue: Rupees(rv), SalesTax: Rupees(stx),
		FurtherTax: Rupees(ft), ExtraTax: Rupees(et), FED: Rupees(fed), STWithheld: Rupees(wh), TotalValue: Rupees(tv), AmountPayable: Rupees(ap)}
	if fbrErrs != "" {
		_ = json.Unmarshal([]byte(fbrErrs), &inv.FBRErrors)
	}
	if valJSON != "" {
		_ = json.Unmarshal([]byte(valJSON), &inv.Validation)
	}
	return &inv, nil
}

const itemCols = `id, invoice_id, line_no, product_id, hs_code, description, uom, quantity, unit_price, discount_percent, discount_amount,
	value_override, sale_type, rate, retail_price, retail_value_override, further_tax_mode, further_tax_override, extra_tax_rate,
	extra_tax_override, fed_rate, fed_override, withholding_override, sales_tax_override, sro_schedule_no, sro_item_serial_no,
	gross, discount, value_excl_st, retail_value, sales_tax, further_tax, extra_tax, extra_tax_empty, fed, st_withheld, total_value,
	fbr_item_invoice_no, fbr_status_code, fbr_error_code, fbr_error`

func scanItem(row interface{ Scan(...any) error }) (*InvoiceItem, error) {
	var it InvoiceItem
	var pid sql.NullInt64
	var qty, up, dp, da, vo, rp, rvo, fto, etr, eto, fedr, fedo, who, sto string
	var g, d, v, rv, st, ft, et, ete, fed, wh, tv int64
	err := row.Scan(&it.ID, &it.InvoiceID, &it.LineNo, &pid, &it.HSCode, &it.Description, &it.UoM, &qty, &up, &dp, &da,
		&vo, &it.SaleType, &it.Rate, &rp, &rvo, &it.FurtherTaxMode, &fto, &etr,
		&eto, &fedr, &fedo, &who, &sto, &it.SROScheduleNo, &it.SROItemSerialNo,
		&g, &d, &v, &rv, &st, &ft, &et, &ete, &fed, &wh, &tv,
		&it.FBRItemInvoiceNo, &it.FBRStatusCode, &it.FBRErrorCode, &it.FBRError)
	if err != nil {
		return nil, err
	}
	it.ProductID = intPtr(pid)
	it.Quantity, it.UnitPrice, it.DiscountPercent, it.DiscountAmount = dec(qty), dec(up), dec(dp), dec(da)
	it.ValueOverride, it.RetailPrice, it.RetailValueOverride = decPtr(vo), dec(rp), decPtr(rvo)
	it.FurtherTaxOverride, it.ExtraTaxRate, it.ExtraTaxOverride = decPtr(fto), dec(etr), decPtr(eto)
	it.FEDRate, it.FEDOverride, it.WithholdingOverride, it.SalesTaxOverride = dec(fedr), decPtr(fedo), decPtr(who), decPtr(sto)
	it.Gross, it.Discount, it.ValueExclST, it.RetailValue = Rupees(g), Rupees(d), Rupees(v), Rupees(rv)
	it.SalesTax, it.FurtherTax, it.ExtraTax, it.ExtraTaxEmpty = Rupees(st), Rupees(ft), Rupees(et), ete == 1
	it.FED, it.STWithheld, it.TotalValue = Rupees(fed), Rupees(wh), Rupees(tv)
	return &it, nil
}

func insertItems(ctx context.Context, q Querier, invoiceID int64, items []*InvoiceItem) error {
	for i, it := range items {
		it.InvoiceID = invoiceID
		it.LineNo = i + 1
		res, err := q.ExecContext(ctx, `INSERT INTO invoice_items(invoice_id, line_no, product_id, hs_code, description, uom, quantity, unit_price,
			discount_percent, discount_amount, value_override, sale_type, rate, retail_price, retail_value_override, further_tax_mode,
			further_tax_override, extra_tax_rate, extra_tax_override, fed_rate, fed_override, withholding_override, sales_tax_override,
			sro_schedule_no, sro_item_serial_no, gross, discount, value_excl_st, retail_value, sales_tax, further_tax, extra_tax,
			extra_tax_empty, fed, st_withheld, total_value, fbr_item_invoice_no, fbr_status_code, fbr_error_code, fbr_error)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			invoiceID, it.LineNo, nullInt(it.ProductID), it.HSCode, it.Description, it.UoM, it.Quantity.String(), it.UnitPrice.String(),
			it.DiscountPercent.String(), it.DiscountAmount.String(), ptrStr(it.ValueOverride), it.SaleType, it.Rate, it.RetailPrice.String(),
			ptrStr(it.RetailValueOverride), it.FurtherTaxMode, ptrStr(it.FurtherTaxOverride), it.ExtraTaxRate.String(), ptrStr(it.ExtraTaxOverride),
			it.FEDRate.String(), ptrStr(it.FEDOverride), ptrStr(it.WithholdingOverride), ptrStr(it.SalesTaxOverride),
			it.SROScheduleNo, it.SROItemSerialNo, Paisa(it.Gross), Paisa(it.Discount), Paisa(it.ValueExclST), Paisa(it.RetailValue),
			Paisa(it.SalesTax), Paisa(it.FurtherTax), Paisa(it.ExtraTax), b2i(it.ExtraTaxEmpty), Paisa(it.FED), Paisa(it.STWithheld),
			Paisa(it.TotalValue), it.FBRItemInvoiceNo, it.FBRStatusCode, it.FBRErrorCode, it.FBRError)
		if err != nil {
			return err
		}
		it.ID, _ = res.LastInsertId()
	}
	return nil
}

func valJSON(v []validate.Issue) string {
	if len(v) == 0 {
		return ""
	}
	return toJSON(v)
}

func errsJSON(v []fbr.ErrorItem) string {
	if len(v) == 0 {
		return ""
	}
	return toJSON(v)
}

// InsertInvoice inserts an invoice and its lines.
func InsertInvoice(ctx context.Context, q Querier, inv *Invoice) error {
	t := now()
	inv.CreatedAt, inv.UpdatedAt = t, t
	tt := inv.Totals
	res, err := q.ExecContext(ctx, `INSERT INTO invoices(company_id, environment, doc_type, internal_no, invoice_date, status, customer_id,
		seller_ntn_cnic, seller_name, seller_province, seller_address, buyer_ntn_cnic, buyer_name, buyer_province, buyer_address,
		buyer_registration_type, withholding_mode, invoice_ref_no, ref_invoice_id, scenario_id, external_ref, source, notes,
		total_gross, total_discount, total_value_excl_st, total_retail_value, total_sales_tax, total_further_tax, total_extra_tax,
		total_fed, total_st_withheld, total_value, amount_payable, validation_json, created_by, updated_by, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		inv.CompanyID, string(inv.Environment), string(inv.DocType), inv.InternalNo, inv.InvoiceDate, string(inv.Status), nullInt(inv.CustomerID),
		inv.SellerNTNCNIC, inv.SellerName, inv.SellerProvince, inv.SellerAddress, inv.BuyerNTNCNIC, inv.BuyerName, inv.BuyerProvince, inv.BuyerAddress,
		string(inv.BuyerRegistrationType), inv.WithholdingMode, inv.InvoiceRefNo, nullInt(inv.RefInvoiceID), inv.ScenarioID, inv.ExternalRef, inv.Source, inv.Notes,
		Paisa(tt.Gross), Paisa(tt.Discount), Paisa(tt.ValueExclST), Paisa(tt.RetailValue), Paisa(tt.SalesTax), Paisa(tt.FurtherTax), Paisa(tt.ExtraTax),
		Paisa(tt.FED), Paisa(tt.STWithheld), Paisa(tt.TotalValue), Paisa(tt.AmountPayable), valJSON(inv.Validation),
		nullInt(inv.CreatedBy), nullInt(inv.UpdatedBy), t, t)
	if isUnique(err) {
		return ErrConflict
	}
	if err != nil {
		return err
	}
	inv.ID, _ = res.LastInsertId()
	return insertItems(ctx, q, inv.ID, inv.Items)
}

// UpdateInvoiceContent replaces an editable invoice's header and lines.
func UpdateInvoiceContent(ctx context.Context, q Querier, inv *Invoice) error {
	tt := inv.Totals
	inv.UpdatedAt = now()
	res, err := q.ExecContext(ctx, `UPDATE invoices SET doc_type=?, invoice_date=?, status=?, customer_id=?, seller_ntn_cnic=?, seller_name=?,
		seller_province=?, seller_address=?, buyer_ntn_cnic=?, buyer_name=?, buyer_province=?, buyer_address=?, buyer_registration_type=?,
		withholding_mode=?, invoice_ref_no=?, ref_invoice_id=?, scenario_id=?, external_ref=?, notes=?, total_gross=?, total_discount=?,
		total_value_excl_st=?, total_retail_value=?, total_sales_tax=?, total_further_tax=?, total_extra_tax=?, total_fed=?,
		total_st_withheld=?, total_value=?, amount_payable=?, validation_json=?, fbr_errors=?, last_error=?, updated_by=?, updated_at=?
		WHERE id=? AND status IN ('DRAFT','VALIDATED','REJECTED')`,
		string(inv.DocType), inv.InvoiceDate, string(inv.Status), nullInt(inv.CustomerID), inv.SellerNTNCNIC, inv.SellerName,
		inv.SellerProvince, inv.SellerAddress, inv.BuyerNTNCNIC, inv.BuyerName, inv.BuyerProvince, inv.BuyerAddress, string(inv.BuyerRegistrationType),
		inv.WithholdingMode, inv.InvoiceRefNo, nullInt(inv.RefInvoiceID), inv.ScenarioID, inv.ExternalRef, inv.Notes, Paisa(tt.Gross), Paisa(tt.Discount),
		Paisa(tt.ValueExclST), Paisa(tt.RetailValue), Paisa(tt.SalesTax), Paisa(tt.FurtherTax), Paisa(tt.ExtraTax), Paisa(tt.FED),
		Paisa(tt.STWithheld), Paisa(tt.TotalValue), Paisa(tt.AmountPayable), valJSON(inv.Validation), errsJSON(inv.FBRErrors), inv.LastError,
		nullInt(inv.UpdatedBy), inv.UpdatedAt, inv.ID)
	if isUnique(err) {
		return ErrConflict
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("invoice %d is not editable", inv.ID)
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM invoice_items WHERE invoice_id=?`, inv.ID); err != nil {
		return err
	}
	return insertItems(ctx, q, inv.ID, inv.Items)
}

func loadItems(ctx context.Context, q Querier, inv *Invoice) error {
	rows, err := q.QueryContext(ctx, `SELECT `+itemCols+` FROM invoice_items WHERE invoice_id=? ORDER BY line_no`, inv.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	inv.Items = nil
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return err
		}
		inv.Items = append(inv.Items, it)
	}
	return rows.Err()
}

// GetInvoiceQ loads an invoice with lines using q (inside a transaction).
func GetInvoiceQ(ctx context.Context, q Querier, companyID, id int64) (*Invoice, error) {
	inv, err := scanInvoice(q.QueryRowContext(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE id=? AND company_id=?`, id, companyID))
	if err != nil {
		return nil, err
	}
	return inv, loadItems(ctx, q, inv)
}

// GetInvoice loads an invoice with its lines.
func (s *Store) GetInvoice(ctx context.Context, companyID, id int64) (*Invoice, error) {
	return GetInvoiceQ(ctx, s.DB, companyID, id)
}

// GetInvoiceAnyCompany loads an invoice by id regardless of company (worker use).
func (s *Store) GetInvoiceAnyCompany(ctx context.Context, id int64) (*Invoice, error) {
	inv, err := scanInvoice(s.DB.QueryRowContext(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	return inv, loadItems(ctx, s.DB, inv)
}

// GetInvoiceByExternalRef finds an invoice by the ERP's reference.
func (s *Store) GetInvoiceByExternalRef(ctx context.Context, companyID int64, env domain.Environment, ref string) (*Invoice, error) {
	inv, err := scanInvoice(s.DB.QueryRowContext(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE company_id=? AND environment=? AND external_ref=?`,
		companyID, string(env), ref))
	if err != nil {
		return nil, err
	}
	return inv, loadItems(ctx, s.DB, inv)
}

// GetInvoiceByFBRNumber finds an invoice by its FBR invoice number.
func (s *Store) GetInvoiceByFBRNumber(ctx context.Context, companyID int64, fbrNo string) (*Invoice, error) {
	inv, err := scanInvoice(s.DB.QueryRowContext(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE company_id=? AND fbr_invoice_number=?`, companyID, fbrNo))
	if err != nil {
		return nil, err
	}
	return inv, loadItems(ctx, s.DB, inv)
}

// InvoiceFilter filters ListInvoices.
type InvoiceFilter struct {
	Environment domain.Environment
	Status      []domain.InvoiceStatus
	DocType     domain.DocType
	From, To    string // YYYY-MM-DD inclusive
	Q           string
	CustomerID  int64
	Limit       int
	Offset      int
}

// ListInvoices returns invoice headers (without lines) and the total count.
func (s *Store) ListInvoices(ctx context.Context, companyID int64, f InvoiceFilter) ([]*Invoice, int, error) {
	where := []string{"company_id=?"}
	args := []any{companyID}
	if f.Environment != "" {
		where = append(where, "environment=?")
		args = append(args, string(f.Environment))
	}
	if len(f.Status) > 0 {
		ph := make([]string, len(f.Status))
		for i, st := range f.Status {
			ph[i] = "?"
			args = append(args, string(st))
		}
		where = append(where, "status IN ("+strings.Join(ph, ",")+")")
	}
	if f.DocType != "" {
		where = append(where, "doc_type=?")
		args = append(args, string(f.DocType))
	}
	if f.From != "" {
		where = append(where, "invoice_date>=?")
		args = append(args, f.From)
	}
	if f.To != "" {
		where = append(where, "invoice_date<=?")
		args = append(args, f.To)
	}
	if f.CustomerID > 0 {
		where = append(where, "customer_id=?")
		args = append(args, f.CustomerID)
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		like := "%" + q + "%"
		where = append(where, "(internal_no LIKE ? OR fbr_invoice_number LIKE ? OR buyer_name LIKE ? OR buyer_ntn_cnic LIKE ? OR external_ref LIKE ?)")
		args = append(args, like, like, like, like, like)
	}
	w := strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM invoices WHERE `+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 || f.Limit > 1000 {
		f.Limit = 50
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE `+w+` ORDER BY invoice_date DESC, id DESC LIMIT ? OFFSET ?`,
		append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, 0, err
		}
		inv.PayloadJSON = ""
		out = append(out, inv)
	}
	return out, total, rows.Err()
}

// NextNumber allocates the next number in a series (inside a transaction).
func NextNumber(ctx context.Context, q Querier, companyID int64, env domain.Environment, docType domain.DocType, fiscalYear string) (int64, error) {
	if _, err := q.ExecContext(ctx, `INSERT INTO number_series(company_id, environment, doc_type, fiscal_year, next_no) VALUES(?,?,?,?,1)
		ON CONFLICT(company_id, environment, doc_type, fiscal_year) DO NOTHING`, companyID, string(env), string(docType), fiscalYear); err != nil {
		return 0, err
	}
	var n int64
	if err := q.QueryRowContext(ctx, `SELECT next_no FROM number_series WHERE company_id=? AND environment=? AND doc_type=? AND fiscal_year=?`,
		companyID, string(env), string(docType), fiscalYear).Scan(&n); err != nil {
		return 0, err
	}
	if _, err := q.ExecContext(ctx, `UPDATE number_series SET next_no=next_no+1 WHERE company_id=? AND environment=? AND doc_type=? AND fiscal_year=?`,
		companyID, string(env), string(docType), fiscalYear); err != nil {
		return 0, err
	}
	return n, nil
}

// ClaimForSubmission atomically moves an invoice into SUBMITTING. It returns
// false when another process already holds it or its state forbids it.
func (s *Store) ClaimForSubmission(ctx context.Context, id int64) (bool, error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE invoices SET status='SUBMITTING', submit_attempts=submit_attempts+1, updated_at=?
		WHERE id=? AND status IN ('DRAFT','VALIDATED','REJECTED','QUEUED')`, now(), id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

// SetPayload stores the exact payload being submitted.
func (s *Store) SetPayload(ctx context.Context, id int64, payloadJSON, payloadHash string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE invoices SET payload_json=?, payload_hash=?, submitted_at=?, updated_at=? WHERE id=?`,
		payloadJSON, payloadHash, now(), now(), id)
	return err
}

// MarkAccepted records FBR's acceptance and the tamper-evident seal. Lines
// are written first: once the header is ACCEPTED the immutability triggers
// forbid any further change to the invoice and its lines.
func MarkAccepted(ctx context.Context, q Querier, inv *Invoice, items []*InvoiceItem) error {
	for _, it := range items {
		if _, err := q.ExecContext(ctx, `UPDATE invoice_items SET fbr_item_invoice_no=?, fbr_status_code=?, fbr_error_code='', fbr_error='' WHERE id=?`,
			it.FBRItemInvoiceNo, it.FBRStatusCode, it.ID); err != nil {
			return err
		}
	}
	t := now()
	res, err := q.ExecContext(ctx, `UPDATE invoices SET status='ACCEPTED', fbr_invoice_number=?, fbr_dated=?, fbr_status_code=?, fbr_errors='',
		last_error='', seal_hash=?, prev_seal_hash=?, accepted_at=?, next_attempt_at='', updated_at=? WHERE id=? AND status IN ('SUBMITTING','UNCERTAIN')`,
		inv.FBRInvoiceNumber, inv.FBRDated, inv.FBRStatusCode, inv.SealHash, inv.PrevSealHash, t, t, inv.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return fmt.Errorf("invoice %d was not awaiting an FBR result", inv.ID)
	}
	inv.AcceptedAt = t
	return nil
}

// MarkRejected records FBR validation errors (invoice becomes editable).
func (s *Store) MarkRejected(ctx context.Context, inv *Invoice) error {
	return s.Tx(ctx, func(q Querier) error {
		_, err := q.ExecContext(ctx, `UPDATE invoices SET status='REJECTED', fbr_status_code=?, fbr_errors=?, last_error=?, next_attempt_at='', updated_at=? WHERE id=?`,
			inv.FBRStatusCode, errsJSON(inv.FBRErrors), inv.LastError, now(), inv.ID)
		if err != nil {
			return err
		}
		for _, it := range inv.Items {
			if _, err := q.ExecContext(ctx, `UPDATE invoice_items SET fbr_status_code=?, fbr_error_code=?, fbr_error=? WHERE id=?`,
				it.FBRStatusCode, it.FBRErrorCode, it.FBRError, it.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// SetStatus changes the status with an error note (queue/uncertain/validated).
func (s *Store) SetStatus(ctx context.Context, id int64, st domain.InvoiceStatus, lastError, nextAttemptAt string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE invoices SET status=?, last_error=?, next_attempt_at=?, updated_at=? WHERE id=?`,
		string(st), lastError, nextAttemptAt, now(), id)
	return err
}

// SetValidation stores the latest validation outcome on an editable invoice.
func (s *Store) SetValidation(ctx context.Context, id int64, st domain.InvoiceStatus, issues []validate.Issue, fbrErrs []fbr.ErrorItem, lastError string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE invoices SET status=?, validation_json=?, fbr_errors=?, last_error=?, updated_at=?
		WHERE id=? AND status IN ('DRAFT','VALIDATED','REJECTED')`,
		string(st), valJSON(issues), errsJSON(fbrErrs), lastError, now(), id)
	return err
}

// DueForSubmission returns queued invoices whose next attempt is due.
func (s *Store) DueForSubmission(ctx context.Context, nowRFC string, limit int) ([]int64, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id FROM invoices WHERE status='QUEUED' AND (next_attempt_at='' OR next_attempt_at<=?)
		ORDER BY next_attempt_at, id LIMIT ?`, nowRFC, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// RecoverStuckSubmissions turns SUBMITTING rows left by a crash into
// UNCERTAIN (they may or may not have reached FBR).
func (s *Store) RecoverStuckSubmissions(ctx context.Context, olderThan string) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE invoices SET status=CASE WHEN payload_json='' THEN 'QUEUED' ELSE 'UNCERTAIN' END,
		last_error=CASE WHEN payload_json='' THEN 'Submission interrupted before it was sent to FBR; queued for resubmission.'
			ELSE 'Submission interrupted before FBR''s answer was recorded (application stopped or connection lost). Verify on IRIS before resubmitting.' END,
		next_attempt_at='', updated_at=?
		WHERE status='SUBMITTING' AND updated_at<?`, now(), olderThan)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// LastSealHash returns the most recent seal hash of a company (hash chain).
func LastSealHash(ctx context.Context, q Querier, companyID int64) (string, error) {
	var h string
	err := q.QueryRowContext(ctx, `SELECT seal_hash FROM invoices WHERE company_id=? AND seal_hash<>'' ORDER BY accepted_at DESC, id DESC LIMIT 1`, companyID).Scan(&h)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return h, err
}

// DeleteInvoice deletes a never-reported invoice.
func (s *Store) DeleteInvoice(ctx context.Context, companyID, id int64) error {
	return s.Tx(ctx, func(q Querier) error {
		if _, err := q.ExecContext(ctx, `DELETE FROM invoice_items WHERE invoice_id=? AND invoice_id IN (SELECT id FROM invoices WHERE company_id=?)`, id, companyID); err != nil {
			return err
		}
		res, err := q.ExecContext(ctx, `DELETE FROM invoices WHERE id=? AND company_id=?`, id, companyID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// IncrementPrintCount records a print.
func (s *Store) IncrementPrintCount(ctx context.Context, id int64) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE invoices SET print_count=print_count+1 WHERE id=?`, id)
	return err
}

// MarkCancelled records an FBR cancellation.
func (s *Store) MarkCancelled(ctx context.Context, id int64, reason, reference string) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE invoices SET status='CANCELLED', cancelled_at=?, cancel_reason=?, cancel_reference=?, updated_at=?
		WHERE id=? AND status='ACCEPTED'`, now(), reason, reference, now(), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("only accepted invoices can be cancelled")
	}
	return nil
}

// SumDebitNotes returns the totals of accepted debit notes against an FBR number.
func (s *Store) SumDebitNotes(ctx context.Context, companyID int64, fbrNo string) (decimal.Decimal, decimal.Decimal, error) {
	var v, t int64
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(total_value_excl_st),0), COALESCE(SUM(total_sales_tax),0) FROM invoices
		WHERE company_id=? AND doc_type='Debit Note' AND invoice_ref_no=? AND status='ACCEPTED'`, companyID, fbrNo).Scan(&v, &t)
	return Rupees(v), Rupees(t), err
}

// RequeueNow makes every queued invoice of a company and environment due
// immediately (used when the connection to FBR is restored, so invoices issued
// during the outage are reported well within FBR's 24-hour window).
func (s *Store) RequeueNow(ctx context.Context, companyID int64, env domain.Environment) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE invoices SET next_attempt_at='' WHERE company_id=? AND environment=? AND status='QUEUED'`,
		companyID, string(env))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// SealedInvoices returns the header rows (without items) of every invoice of a
// company that carries a seal, i.e. every invoice FBR accepted, for the
// integrity check.
func (s *Store) SealedInvoices(ctx context.Context, companyID int64) ([]*Invoice, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE company_id=? AND seal_hash<>'' ORDER BY id`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Invoice
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

// MarkOffline records when an invoice could first not be reported because
// FBR (or its security token) was unavailable; later calls keep that time.
func (s *Store) MarkOffline(ctx context.Context, id int64, at string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE invoices SET offline_since=? WHERE id=? AND offline_since=''`, at, id)
	return err
}

// RequeueOldest makes the oldest queued invoice of a company and environment
// due immediately (a probe whether posting to FBR works again).
func (s *Store) RequeueOldest(ctx context.Context, companyID int64, env domain.Environment) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `UPDATE invoices SET next_attempt_at='' WHERE id=(SELECT id FROM invoices
		WHERE company_id=? AND environment=? AND status='QUEUED' ORDER BY created_at, id LIMIT 1)`, companyID, string(env))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
