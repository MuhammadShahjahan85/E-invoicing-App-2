package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"einvoicing/internal/domain"

	"github.com/shopspring/decimal"
)

// Customer is a buyer.
type Customer struct {
	ID               int64                   `json:"id"`
	CompanyID        int64                   `json:"companyId"`
	Code             string                  `json:"code"`
	Name             string                  `json:"name"`
	NTNCNIC          string                  `json:"ntnCnic"`
	STRN             string                  `json:"strn"`
	RegistrationType domain.RegistrationType `json:"registrationType"`
	Province         string                  `json:"province"`
	Address          string                  `json:"address"`
	City             string                  `json:"city"`
	Phone            string                  `json:"phone"`
	Email            string                  `json:"email"`
	// WithholdingMode: "" (not a withholding agent), "fraction" (1/5th),
	// "full" — see tax.WithholdingMode.
	WithholdingMode string `json:"withholdingMode"`
	StatlStatus     string `json:"statlStatus"`
	FBRRegType      string `json:"fbrRegType"`
	StatusCheckedAt string `json:"statusCheckedAt"`
	Notes           string `json:"notes"`
	Active          bool   `json:"active"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

const customerCols = `id, company_id, code, name, ntn_cnic, strn, registration_type, province, address, city, phone, email,
	withholding_mode, statl_status, fbr_reg_type, status_checked_at, notes, active, created_at, updated_at`

func scanCustomer(row interface{ Scan(...any) error }) (*Customer, error) {
	var c Customer
	var rt string
	var active int
	err := row.Scan(&c.ID, &c.CompanyID, &c.Code, &c.Name, &c.NTNCNIC, &c.STRN, &rt, &c.Province, &c.Address, &c.City, &c.Phone, &c.Email,
		&c.WithholdingMode, &c.StatlStatus, &c.FBRRegType, &c.StatusCheckedAt, &c.Notes, &active, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.RegistrationType = domain.RegistrationType(rt)
	c.Active = active == 1
	return &c, nil
}

// ListParams filters list queries.
type ListParams struct {
	Q          string
	Limit      int
	Offset     int
	OnlyActive bool
}

func (p *ListParams) norm() {
	if p.Limit <= 0 || p.Limit > 1000 {
		p.Limit = 100
	}
	if p.Offset < 0 {
		p.Offset = 0
	}
}

// ListCustomers lists a company's customers.
func (s *Store) ListCustomers(ctx context.Context, companyID int64, p ListParams) ([]*Customer, int, error) {
	p.norm()
	where := `company_id=?`
	args := []any{companyID}
	if p.OnlyActive {
		where += ` AND active=1`
	}
	if q := strings.TrimSpace(p.Q); q != "" {
		where += ` AND (name LIKE ? OR ntn_cnic LIKE ? OR code LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM customers WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+customerCols+` FROM customers WHERE `+where+` ORDER BY name LIMIT ? OFFSET ?`,
		append(args, p.Limit, p.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Customer
	for rows.Next() {
		c, err := scanCustomer(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, c)
	}
	return out, total, rows.Err()
}

// GetCustomer loads a customer of a company.
func (s *Store) GetCustomer(ctx context.Context, companyID, id int64) (*Customer, error) {
	return scanCustomer(s.DB.QueryRowContext(ctx, `SELECT `+customerCols+` FROM customers WHERE id=? AND company_id=?`, id, companyID))
}

// FindCustomerByRegNo finds a customer by NTN/CNIC.
func (s *Store) FindCustomerByRegNo(ctx context.Context, companyID int64, regNo string) (*Customer, error) {
	return scanCustomer(s.DB.QueryRowContext(ctx, `SELECT `+customerCols+` FROM customers WHERE company_id=? AND ntn_cnic=? ORDER BY id LIMIT 1`, companyID, regNo))
}

// SaveCustomer inserts or updates a customer.
func (s *Store) SaveCustomer(ctx context.Context, c *Customer) error {
	t := now()
	if c.ID == 0 {
		res, err := s.DB.ExecContext(ctx, `INSERT INTO customers(company_id, code, name, ntn_cnic, strn, registration_type, province, address,
			city, phone, email, withholding_mode, statl_status, fbr_reg_type, status_checked_at, notes, active, created_at, updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			c.CompanyID, c.Code, c.Name, c.NTNCNIC, c.STRN, string(c.RegistrationType), c.Province, c.Address, c.City, c.Phone, c.Email,
			c.WithholdingMode, c.StatlStatus, c.FBRRegType, c.StatusCheckedAt, c.Notes, b2i(c.Active), t, t)
		if err != nil {
			return err
		}
		c.ID, _ = res.LastInsertId()
		c.CreatedAt, c.UpdatedAt = t, t
		return nil
	}
	res, err := s.DB.ExecContext(ctx, `UPDATE customers SET code=?, name=?, ntn_cnic=?, strn=?, registration_type=?, province=?, address=?,
		city=?, phone=?, email=?, withholding_mode=?, statl_status=?, fbr_reg_type=?, status_checked_at=?, notes=?, active=?, updated_at=?
		WHERE id=? AND company_id=?`,
		c.Code, c.Name, c.NTNCNIC, c.STRN, string(c.RegistrationType), c.Province, c.Address, c.City, c.Phone, c.Email,
		c.WithholdingMode, c.StatlStatus, c.FBRRegType, c.StatusCheckedAt, c.Notes, b2i(c.Active), t, c.ID, c.CompanyID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	c.UpdatedAt = t
	return nil
}

// Product is an item or service master record carrying FBR classification.
type Product struct {
	ID              int64           `json:"id"`
	CompanyID       int64           `json:"companyId"`
	Code            string          `json:"code"`
	Description     string          `json:"description"`
	HSCode          string          `json:"hsCode"`
	UoM             string          `json:"uom"`
	SaleType        string          `json:"saleType"`
	Rate            string          `json:"rate"`
	SROScheduleNo   string          `json:"sroScheduleNo"`
	SROItemSerialNo string          `json:"sroItemSerialNo"`
	UnitPrice       decimal.Decimal `json:"unitPrice"`
	RetailPrice     decimal.Decimal `json:"retailPrice"`
	// FurtherTaxMode: "auto" (sale type default), "yes", "no".
	FurtherTaxMode string          `json:"furtherTaxMode"`
	ExtraTaxRate   decimal.Decimal `json:"extraTaxRate"`
	FEDRate        decimal.Decimal `json:"fedRate"`
	Active         bool            `json:"active"`
	CreatedAt      string          `json:"createdAt"`
	UpdatedAt      string          `json:"updatedAt"`
}

const productCols = `id, company_id, code, description, hs_code, uom, sale_type, rate, sro_schedule_no, sro_item_serial_no,
	unit_price, retail_price, further_tax_mode, extra_tax_rate, fed_rate, active, created_at, updated_at`

func scanProduct(row interface{ Scan(...any) error }) (*Product, error) {
	var p Product
	var up, rp, etr, fed string
	var active int
	err := row.Scan(&p.ID, &p.CompanyID, &p.Code, &p.Description, &p.HSCode, &p.UoM, &p.SaleType, &p.Rate, &p.SROScheduleNo, &p.SROItemSerialNo,
		&up, &rp, &p.FurtherTaxMode, &etr, &fed, &active, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.UnitPrice, p.RetailPrice, p.ExtraTaxRate, p.FEDRate = dec(up), dec(rp), dec(etr), dec(fed)
	p.Active = active == 1
	return &p, nil
}

// ListProducts lists a company's products.
func (s *Store) ListProducts(ctx context.Context, companyID int64, p ListParams) ([]*Product, int, error) {
	p.norm()
	where := `company_id=?`
	args := []any{companyID}
	if p.OnlyActive {
		where += ` AND active=1`
	}
	if q := strings.TrimSpace(p.Q); q != "" {
		where += ` AND (description LIKE ? OR code LIKE ? OR hs_code LIKE ?)`
		like := "%" + q + "%"
		args = append(args, like, like, like)
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM products WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+productCols+` FROM products WHERE `+where+` ORDER BY description LIMIT ? OFFSET ?`,
		append(args, p.Limit, p.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*Product
	for rows.Next() {
		pr, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, pr)
	}
	return out, total, rows.Err()
}

// GetProduct loads a product of a company.
func (s *Store) GetProduct(ctx context.Context, companyID, id int64) (*Product, error) {
	return scanProduct(s.DB.QueryRowContext(ctx, `SELECT `+productCols+` FROM products WHERE id=? AND company_id=?`, id, companyID))
}

// FindProductByCode finds a product by its code.
func (s *Store) FindProductByCode(ctx context.Context, companyID int64, code string) (*Product, error) {
	return scanProduct(s.DB.QueryRowContext(ctx, `SELECT `+productCols+` FROM products WHERE company_id=? AND code=? AND code<>'' ORDER BY id LIMIT 1`, companyID, code))
}

// SaveProduct inserts or updates a product.
func (s *Store) SaveProduct(ctx context.Context, p *Product) error {
	t := now()
	if p.FurtherTaxMode == "" {
		p.FurtherTaxMode = "auto"
	}
	if p.ID == 0 {
		res, err := s.DB.ExecContext(ctx, `INSERT INTO products(company_id, code, description, hs_code, uom, sale_type, rate, sro_schedule_no,
			sro_item_serial_no, unit_price, retail_price, further_tax_mode, extra_tax_rate, fed_rate, active, created_at, updated_at)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			p.CompanyID, p.Code, p.Description, p.HSCode, p.UoM, p.SaleType, p.Rate, p.SROScheduleNo, p.SROItemSerialNo,
			p.UnitPrice.String(), p.RetailPrice.String(), p.FurtherTaxMode, p.ExtraTaxRate.String(), p.FEDRate.String(), b2i(p.Active), t, t)
		if err != nil {
			return err
		}
		p.ID, _ = res.LastInsertId()
		p.CreatedAt, p.UpdatedAt = t, t
		return nil
	}
	res, err := s.DB.ExecContext(ctx, `UPDATE products SET code=?, description=?, hs_code=?, uom=?, sale_type=?, rate=?, sro_schedule_no=?,
		sro_item_serial_no=?, unit_price=?, retail_price=?, further_tax_mode=?, extra_tax_rate=?, fed_rate=?, active=?, updated_at=?
		WHERE id=? AND company_id=?`,
		p.Code, p.Description, p.HSCode, p.UoM, p.SaleType, p.Rate, p.SROScheduleNo, p.SROItemSerialNo,
		p.UnitPrice.String(), p.RetailPrice.String(), p.FurtherTaxMode, p.ExtraTaxRate.String(), p.FEDRate.String(), b2i(p.Active), t, p.ID, p.CompanyID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	p.UpdatedAt = t
	return nil
}
