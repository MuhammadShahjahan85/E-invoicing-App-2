// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"errors"
)

// ReturnExtension is a return filing date extended by FBR for a tax period.
type ReturnExtension struct {
	Period     string `json:"period"`     // YYYY-MM
	FilingDate string `json:"filingDate"` // YYYY-MM-DD
	Reference  string `json:"reference"`  // FBR order or circular
	CreatedAt  string `json:"createdAt"`
}

// SetReturnExtension records (or replaces) the extension of a period.
func (s *Store) SetReturnExtension(ctx context.Context, companyID int64, e ReturnExtension, userID *int64) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO return_extensions(company_id, period, filing_date, reference, created_by, created_at)
		VALUES(?,?,?,?,?,?) ON CONFLICT(company_id, period) DO UPDATE SET filing_date=excluded.filing_date, reference=excluded.reference,
		created_by=excluded.created_by, created_at=excluded.created_at`,
		companyID, e.Period, e.FilingDate, e.Reference, nullInt(userID), now())
	return err
}

// DeleteReturnExtension removes the extension of a period.
func (s *Store) DeleteReturnExtension(ctx context.Context, companyID int64, period string) error {
	res, err := s.DB.ExecContext(ctx, `DELETE FROM return_extensions WHERE company_id=? AND period=?`, companyID, period)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ReturnExtensionFor returns the extension of a period, or nil.
func (s *Store) ReturnExtensionFor(ctx context.Context, companyID int64, period string) (*ReturnExtension, error) {
	var e ReturnExtension
	err := s.DB.QueryRowContext(ctx, `SELECT period, filing_date, reference, created_at FROM return_extensions WHERE company_id=? AND period=?`,
		companyID, period).Scan(&e.Period, &e.FilingDate, &e.Reference, &e.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// ListReturnExtensions returns a company's extensions, newest period first.
func (s *Store) ListReturnExtensions(ctx context.Context, companyID int64) ([]ReturnExtension, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT period, filing_date, reference, created_at FROM return_extensions WHERE company_id=? ORDER BY period DESC LIMIT 36`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ReturnExtension{}
	for rows.Next() {
		var e ReturnExtension
		if err := rows.Scan(&e.Period, &e.FilingDate, &e.Reference, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
