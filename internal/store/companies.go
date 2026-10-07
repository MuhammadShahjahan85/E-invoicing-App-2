// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"einvoicing/internal/domain"

	"github.com/shopspring/decimal"
)

// PrintSettings controls the printed invoice layout for a company.
type PrintSettings struct {
	PaperSize       string `json:"paperSize"`  // "A4" or "thermal80"
	WordsStyle      string `json:"wordsStyle"` // "south_asian" or "international"
	ShowAmountWords bool   `json:"showAmountWords"`
	ShowSignature   bool   `json:"showSignature"`
	FooterText      string `json:"footerText"`
	TermsText       string `json:"termsText"`
	ShowUnitPrice   bool   `json:"showUnitPrice"`
	ShowHSCode      bool   `json:"showHsCode"`
	ShowDiscount    bool   `json:"showDiscount"`
	Copies          int    `json:"copies"`
}

// DefaultPrintSettings returns sensible defaults.
func DefaultPrintSettings() PrintSettings {
	return PrintSettings{PaperSize: "A4", WordsStyle: "south_asian", ShowAmountWords: true, ShowSignature: true,
		ShowUnitPrice: true, ShowHSCode: true, ShowDiscount: true, Copies: 1,
		FooterText: "This is a computer generated invoice reported to FBR Digital Invoicing System."}
}

// Default due days of the monthly sales tax return (of the month after the
// tax period).
const (
	DefaultReturnPaymentDay = 15
	DefaultReturnFilingDay  = 18
)

// Company is a seller (taxpayer) registered for sales tax. One installation
// can manage several companies (e.g. a group or a tax practitioner's clients).
type Company struct {
	ID                    int64              `json:"id"`
	Name                  string             `json:"name"`
	NTNCNIC               string             `json:"ntnCnic"`
	STRN                  string             `json:"strn"`
	Province              string             `json:"province"`
	ProvinceCode          int                `json:"provinceCode"`
	Address               string             `json:"address"`
	City                  string             `json:"city"`
	Phone                 string             `json:"phone"`
	Email                 string             `json:"email"`
	BusinessActivities    []string           `json:"businessActivities"`
	Sectors               []string           `json:"sectors"`
	AssignedScenarios     []string           `json:"assignedScenarios"`
	Environment           domain.Environment `json:"environment"`
	SandboxTokenEnc       string             `json:"-"`
	ProductionTokenEnc    string             `json:"-"`
	HasSandboxToken       bool               `json:"hasSandboxToken"`
	HasProductionToken    bool               `json:"hasProductionToken"`
	SandboxTokenExpiry    string             `json:"sandboxTokenExpiry"`
	ProductionTokenExpiry string             `json:"productionTokenExpiry"`
	InvoicePrefix         string             `json:"invoicePrefix"`
	DebitNotePrefix       string             `json:"debitNotePrefix"`
	FurtherTaxRate        decimal.Decimal    `json:"furtherTaxRate"`
	WithholdingFraction   decimal.Decimal    `json:"withholdingFraction"`
	SendInternalRef       bool               `json:"sendInternalRef"`
	ValidateBeforePost    bool               `json:"validateBeforePost"`
	// ReturnPaymentDay and ReturnFilingDay are the days of the month after a
	// tax period by which sales tax is paid and the return is filed.
	ReturnPaymentDay int `json:"returnPaymentDay"`
	ReturnFilingDay  int `json:"returnFilingDay"`
	HasLogo               bool               `json:"hasLogo"`
	LogoMime              string             `json:"-"`
	PrintSettings         PrintSettings      `json:"printSettings"`
	Active                bool               `json:"active"`
	CreatedAt             string             `json:"createdAt"`
	UpdatedAt             string             `json:"updatedAt"`
}

const companyCols = `id, name, ntn_cnic, strn, province, province_code, address, city, phone, email,
	business_activities, sectors, assigned_scenarios, environment, sandbox_token_enc, production_token_enc,
	sandbox_token_expiry, production_token_expiry, invoice_prefix, debit_note_prefix, further_tax_rate,
	withholding_fraction, send_internal_ref, validate_before_post, (logo IS NOT NULL AND length(logo) > 0), logo_mime,
	print_settings, active, created_at, updated_at, return_payment_day, return_filing_day`

func scanCompany(row interface{ Scan(...any) error }) (*Company, error) {
	var c Company
	var acts, secs, scen, env, ftr, wf, ps string
	var sendRef, valBefore, hasLogo, active int
	err := row.Scan(&c.ID, &c.Name, &c.NTNCNIC, &c.STRN, &c.Province, &c.ProvinceCode, &c.Address, &c.City, &c.Phone, &c.Email,
		&acts, &secs, &scen, &env, &c.SandboxTokenEnc, &c.ProductionTokenEnc, &c.SandboxTokenExpiry, &c.ProductionTokenExpiry,
		&c.InvoicePrefix, &c.DebitNotePrefix, &ftr, &wf, &sendRef, &valBefore, &hasLogo, &c.LogoMime, &ps, &active, &c.CreatedAt, &c.UpdatedAt,
		&c.ReturnPaymentDay, &c.ReturnFilingDay)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.BusinessActivities, c.Sectors, c.AssignedScenarios = jsonList(acts), jsonList(secs), jsonList(scen)
	c.Environment = domain.Environment(env)
	c.FurtherTaxRate, c.WithholdingFraction = dec(ftr), dec(wf)
	c.SendInternalRef, c.ValidateBeforePost, c.HasLogo, c.Active = sendRef == 1, valBefore == 1, hasLogo == 1, active == 1
	c.HasSandboxToken, c.HasProductionToken = c.SandboxTokenEnc != "", c.ProductionTokenEnc != ""
	c.PrintSettings = DefaultPrintSettings()
	_ = json.Unmarshal([]byte(ps), &c.PrintSettings)
	return &c, nil
}

// ListCompanies returns all companies.
func (s *Store) ListCompanies(ctx context.Context) ([]*Company, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+companyCols+` FROM companies ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Company
	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCompany loads a company.
func (s *Store) GetCompany(ctx context.Context, id int64) (*Company, error) {
	return scanCompany(s.DB.QueryRowContext(ctx, `SELECT `+companyCols+` FROM companies WHERE id=?`, id))
}

// CountCompanies returns the number of companies.
func (s *Store) CountCompanies(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM companies`).Scan(&n)
	return n, err
}

// CreateCompany inserts a company and returns its id.
func (s *Store) CreateCompany(ctx context.Context, c *Company) (int64, error) {
	t := now()
	if c.Environment == "" {
		c.Environment = domain.EnvSimulator
	}
	if c.InvoicePrefix == "" {
		c.InvoicePrefix = "INV"
	}
	if c.DebitNotePrefix == "" {
		c.DebitNotePrefix = "DN"
	}
	if c.FurtherTaxRate.IsZero() {
		c.FurtherTaxRate = decimal.NewFromInt(4)
	}
	if c.WithholdingFraction.IsZero() {
		c.WithholdingFraction = decimal.NewFromFloat(0.2)
	}
	if c.PrintSettings.PaperSize == "" {
		c.PrintSettings = DefaultPrintSettings()
	}
	if c.ReturnPaymentDay == 0 {
		c.ReturnPaymentDay = DefaultReturnPaymentDay
	}
	if c.ReturnFilingDay == 0 {
		c.ReturnFilingDay = DefaultReturnFilingDay
	}
	res, err := s.DB.ExecContext(ctx, `INSERT INTO companies(name, ntn_cnic, strn, province, province_code, address, city, phone, email,
		business_activities, sectors, assigned_scenarios, environment, invoice_prefix, debit_note_prefix, further_tax_rate,
		withholding_fraction, send_internal_ref, validate_before_post, print_settings, active, created_at, updated_at,
		return_payment_day, return_filing_day)
		VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,1,?,?,?,?)`,
		c.Name, c.NTNCNIC, c.STRN, c.Province, c.ProvinceCode, c.Address, c.City, c.Phone, c.Email,
		toJSON(nz(c.BusinessActivities)), toJSON(nz(c.Sectors)), toJSON(nz(c.AssignedScenarios)), string(c.Environment),
		c.InvoicePrefix, c.DebitNotePrefix, c.FurtherTaxRate.String(), c.WithholdingFraction.String(),
		b2i(c.SendInternalRef), b2i(c.ValidateBeforePost), toJSON(c.PrintSettings), t, t,
		c.ReturnPaymentDay, c.ReturnFilingDay)
	if isUnique(err) {
		return 0, ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func nz(a []string) []string {
	if a == nil {
		return []string{}
	}
	return a
}

// UpdateCompany saves editable company fields (not tokens or logo).
func (s *Store) UpdateCompany(ctx context.Context, c *Company) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE companies SET name=?, ntn_cnic=?, strn=?, province=?, province_code=?, address=?, city=?,
		phone=?, email=?, business_activities=?, sectors=?, assigned_scenarios=?, environment=?, invoice_prefix=?, debit_note_prefix=?,
		further_tax_rate=?, withholding_fraction=?, send_internal_ref=?, validate_before_post=?, print_settings=?, active=?,
		sandbox_token_expiry=?, production_token_expiry=?, return_payment_day=?, return_filing_day=?, updated_at=? WHERE id=?`,
		c.Name, c.NTNCNIC, c.STRN, c.Province, c.ProvinceCode, c.Address, c.City, c.Phone, c.Email,
		toJSON(nz(c.BusinessActivities)), toJSON(nz(c.Sectors)), toJSON(nz(c.AssignedScenarios)), string(c.Environment),
		c.InvoicePrefix, c.DebitNotePrefix, c.FurtherTaxRate.String(), c.WithholdingFraction.String(),
		b2i(c.SendInternalRef), b2i(c.ValidateBeforePost), toJSON(c.PrintSettings), b2i(c.Active),
		c.SandboxTokenExpiry, c.ProductionTokenExpiry, c.ReturnPaymentDay, c.ReturnFilingDay, now(), c.ID)
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

// SetCompanyTokens stores encrypted tokens. Empty strings leave a token unchanged
// unless clear is true for that environment.
func (s *Store) SetCompanyToken(ctx context.Context, id int64, env domain.Environment, enc string) error {
	col := "sandbox_token_enc"
	if env == domain.EnvProduction {
		col = "production_token_enc"
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE companies SET `+col+`=?, updated_at=? WHERE id=?`, enc, now(), id)
	return err
}

// SetCompanyLogo stores the company logo image.
func (s *Store) SetCompanyLogo(ctx context.Context, id int64, data []byte, mime string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE companies SET logo=?, logo_mime=?, updated_at=? WHERE id=?`, data, mime, now(), id)
	return err
}

// GetCompanyLogo returns the logo bytes and mime type.
func (s *Store) GetCompanyLogo(ctx context.Context, id int64) ([]byte, string, error) {
	var b []byte
	var mime string
	err := s.DB.QueryRowContext(ctx, `SELECT logo, logo_mime FROM companies WHERE id=?`, id).Scan(&b, &mime)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	return b, mime, err
}
