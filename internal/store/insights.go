// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package store

import (
	"context"

	"einvoicing/internal/domain"

	"github.com/shopspring/decimal"
)

// ItemRow aggregates accepted sale invoice lines by HS code.
type ItemRow struct {
	HSCode      string          `json:"hsCode"`
	Description string          `json:"description"`
	Lines       int             `json:"lines"`
	ValueExclST decimal.Decimal `json:"valueExclST"`
	SalesTax    decimal.Decimal `json:"salesTax"`
}

// TopItems returns the HS codes with the highest value of accepted sale
// invoices in the period.
func (s *Store) TopItems(ctx context.Context, f ReportFilter, limit int) ([]ItemRow, error) {
	w, args := reportWhere(f)
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT l.hs_code, MAX(l.description), COUNT(*), SUM(l.value_excl_st), SUM(l.sales_tax)
		FROM invoice_items l JOIN invoices i ON i.id=l.invoice_id
		WHERE `+w+` AND i.doc_type='Sale Invoice' GROUP BY l.hs_code ORDER BY SUM(l.value_excl_st) DESC LIMIT ?`, append(args, limit)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ItemRow
	for rows.Next() {
		var r ItemRow
		var v, st int64
		if err := rows.Scan(&r.HSCode, &r.Description, &r.Lines, &v, &st); err != nil {
			return nil, err
		}
		r.ValueExclST, r.SalesTax = Rupees(v), Rupees(st)
		out = append(out, r)
	}
	return out, rows.Err()
}

// PeriodStatus counts a company's documents dated in a period, by status
// and document type, in one environment.
type PeriodStatus struct {
	Status  string `json:"status"`
	DocType string `json:"docType"`
	Count   int    `json:"count"`
}

// PeriodStatusCounts counts documents dated from..to (inclusive) by status.
func (s *Store) PeriodStatusCounts(ctx context.Context, companyID int64, env domain.Environment, from, to string) ([]PeriodStatus, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT status, doc_type, COUNT(*) FROM invoices
		WHERE company_id=? AND environment=? AND invoice_date>=? AND invoice_date<=? GROUP BY status, doc_type ORDER BY status, doc_type`,
		companyID, string(env), from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PeriodStatus
	for rows.Next() {
		var p PeriodStatus
		if err := rows.Scan(&p.Status, &p.DocType, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// StatusTotal is the number and value of a company's documents in one
// status.
type StatusTotal struct {
	Count    int
	Value    decimal.Decimal // value excluding sales tax
	SalesTax decimal.Decimal
}

// PeriodStatusTotals totals the documents dated from..to (inclusive) by
// status, in one environment.
func (s *Store) PeriodStatusTotals(ctx context.Context, companyID int64, env domain.Environment, from, to string) (map[string]StatusTotal, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT status, COUNT(*), COALESCE(SUM(total_value_excl_st),0), COALESCE(SUM(total_sales_tax),0) FROM invoices
		WHERE company_id=? AND environment=? AND invoice_date>=? AND invoice_date<=? GROUP BY status`, companyID, string(env), from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]StatusTotal{}
	for rows.Next() {
		var status string
		var n int
		var v, st int64
		if err := rows.Scan(&status, &n, &v, &st); err != nil {
			return nil, err
		}
		out[status] = StatusTotal{Count: n, Value: Rupees(v), SalesTax: Rupees(st)}
	}
	return out, rows.Err()
}

// IncidentsBetween returns a company's incidents that started before `to`
// and had not ended before `from` (RFC 3339 timestamps).
func (s *Store) IncidentsBetween(ctx context.Context, companyID int64, from, to string) ([]*Incident, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+incidentCols+` FROM incidents WHERE company_id=? AND started_at<? AND (ended_at='' OR ended_at>=?)
		ORDER BY started_at`, companyID, to, from)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Incident
	for rows.Next() {
		i, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// LastReferenceSync returns when FBR reference lists were last downloaded
// for an environment ("" when they never were).
func (s *Store) LastReferenceSync(ctx context.Context, env domain.Environment) (string, error) {
	var t string
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(fetched_at),'') FROM ref_cache WHERE source='fbr' AND cache_key IN (?,?,?,?,?)`,
		string(env)+":provinces", string(env)+":doctypecode", string(env)+":transtypecode", string(env)+":uom", string(env)+":sroitemcode").Scan(&t)
	return t, err
}

// HSCodeSource reports where the stored HS code list came from ("fbr" when
// it was downloaded from FBR, otherwise "seed") and how many codes it holds.
func (s *Store) HSCodeSource(ctx context.Context) (string, int, error) {
	var fromFBR, n int
	err := s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN source='fbr' THEN 1 ELSE 0 END),0), COUNT(*) FROM ref_hs_codes`).Scan(&fromFBR, &n)
	if fromFBR > 0 {
		return "fbr", n, err
	}
	return "seed", n, err
}

// IncidentCounts returns how many of a company's incidents are still open
// and how many have not been reported to FBR.
func (s *Store) IncidentCounts(ctx context.Context, companyID int64) (open, unreported int, err error) {
	err = s.DB.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN ended_at='' THEN 1 ELSE 0 END),0),
		COALESCE(SUM(CASE WHEN reported_at='' THEN 1 ELSE 0 END),0) FROM incidents WHERE company_id=?`, companyID).Scan(&open, &unreported)
	return open, unreported, err
}
