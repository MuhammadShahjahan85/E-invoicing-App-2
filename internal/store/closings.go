// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"

	"einvoicing/internal/domain"

	"github.com/shopspring/decimal"
)

// Closing is a day, week or month closing (rule 150R(4)(f) of the Sales Tax
// Rules, 2006): a snapshot of the period's invoices taken when it ends.
type Closing struct {
	ID          int64              `json:"id"`
	CompanyID   int64              `json:"companyId"`
	Environment domain.Environment `json:"environment"`
	Kind        string             `json:"kind"` // day, week, month
	PeriodKey   string             `json:"periodKey"`
	PeriodStart string             `json:"periodStart"`
	PeriodEnd   string             `json:"periodEnd"`
	Summary     ClosingSummary     `json:"summary"`
	Hash        string             `json:"hash"`
	PrevHash    string             `json:"prevHash"`
	CreatedAt   string             `json:"createdAt"`
}

// ClosingTotals are the amounts of reported documents of one type.
type ClosingTotals struct {
	Count       int             `json:"count"`
	ValueExclST decimal.Decimal `json:"valueExclST"`
	SalesTax    decimal.Decimal `json:"salesTax"`
	FurtherTax  decimal.Decimal `json:"furtherTax"`
	ExtraTax    decimal.Decimal `json:"extraTax"`
	FED         decimal.Decimal `json:"fed"`
	STWithheld  decimal.Decimal `json:"stWithheld"`
	TotalValue  decimal.Decimal `json:"totalValue"`
}

// ClosingSummary counts the period's documents by status and totals the
// documents reported to FBR.
type ClosingSummary struct {
	Documents    int           `json:"documents"`
	Reported     int           `json:"reported"`
	Cancelled    int           `json:"cancelled"`
	Pending      int           `json:"pending"`      // queued or being sent
	Unreconciled int           `json:"unreconciled"` // outcome unknown
	Rejected     int           `json:"rejected"`
	Drafts       int           `json:"drafts"`
	Sales        ClosingTotals `json:"sales"`      // sale invoices reported
	DebitNotes   ClosingTotals `json:"debitNotes"` // debit notes reported
	FirstNo      string        `json:"firstNo"`    // first and last invoice
	LastNo       string        `json:"lastNo"`     // numbers reported
	FirstFBRNo   string        `json:"firstFbrNo"`
	LastFBRNo    string        `json:"lastFbrNo"`
	OfflineMode  int           `json:"offlineMode"` // issued in offline mode
}

// ClosingSummaryFor computes the summary of a company's invoices dated in
// [from, to] (YYYY-MM-DD) for an environment.
func (s *Store) ClosingSummaryFor(ctx context.Context, companyID int64, env domain.Environment, from, to string) (ClosingSummary, error) {
	var sum ClosingSummary
	rows, err := s.DB.QueryContext(ctx, `SELECT doc_type, status, COUNT(*), COALESCE(SUM(total_value_excl_st),0), COALESCE(SUM(total_sales_tax),0),
		COALESCE(SUM(total_further_tax),0), COALESCE(SUM(total_extra_tax),0), COALESCE(SUM(total_fed),0), COALESCE(SUM(total_st_withheld),0),
		COALESCE(SUM(total_value),0), COALESCE(SUM(offline_since<>''),0)
		FROM invoices WHERE company_id=? AND environment=? AND invoice_date>=? AND invoice_date<=? GROUP BY doc_type, status`,
		companyID, string(env), from, to)
	if err != nil {
		return sum, err
	}
	defer rows.Close()
	for rows.Next() {
		var dt, st string
		var n, offline int
		var v, stx, ft, et, fed, wh, tv int64
		if err := rows.Scan(&dt, &st, &n, &v, &stx, &ft, &et, &fed, &wh, &tv, &offline); err != nil {
			return sum, err
		}
		sum.Documents += n
		sum.OfflineMode += offline
		switch domain.InvoiceStatus(st) {
		case domain.StatusAccepted:
			sum.Reported += n
			t := &sum.Sales
			if domain.DocType(dt) == domain.DocDebitNote {
				t = &sum.DebitNotes
			}
			t.Count += n
			t.ValueExclST = t.ValueExclST.Add(Rupees(v))
			t.SalesTax = t.SalesTax.Add(Rupees(stx))
			t.FurtherTax = t.FurtherTax.Add(Rupees(ft))
			t.ExtraTax = t.ExtraTax.Add(Rupees(et))
			t.FED = t.FED.Add(Rupees(fed))
			t.STWithheld = t.STWithheld.Add(Rupees(wh))
			t.TotalValue = t.TotalValue.Add(Rupees(tv))
		case domain.StatusCancelled:
			sum.Cancelled += n
		case domain.StatusQueued, domain.StatusSubmitting:
			sum.Pending += n
		case domain.StatusUncertain:
			sum.Unreconciled += n
		case domain.StatusRejected:
			sum.Rejected += n
		default:
			sum.Drafts += n
		}
	}
	if err := rows.Err(); err != nil {
		return sum, err
	}
	first := s.DB.QueryRowContext(ctx, `SELECT internal_no, fbr_invoice_number FROM invoices WHERE company_id=? AND environment=? AND invoice_date>=? AND invoice_date<=?
		AND status='ACCEPTED' ORDER BY accepted_at, id LIMIT 1`, companyID, string(env), from, to)
	if err := first.Scan(&sum.FirstNo, &sum.FirstFBRNo); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return sum, err
	}
	last := s.DB.QueryRowContext(ctx, `SELECT internal_no, fbr_invoice_number FROM invoices WHERE company_id=? AND environment=? AND invoice_date>=? AND invoice_date<=?
		AND status='ACCEPTED' ORDER BY accepted_at DESC, id DESC LIMIT 1`, companyID, string(env), from, to)
	if err := last.Scan(&sum.LastNo, &sum.LastFBRNo); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return sum, err
	}
	return sum, nil
}

// closingHash chains a closing to the previous one of the same company and
// environment.
func closingHash(c *Closing) (string, error) {
	body, err := json.Marshal(struct {
		CompanyID   int64          `json:"c"`
		Environment string         `json:"e"`
		Kind        string         `json:"k"`
		PeriodKey   string         `json:"p"`
		PeriodStart string         `json:"s"`
		PeriodEnd   string         `json:"t"`
		Summary     ClosingSummary `json:"m"`
		CreatedAt   string         `json:"at"`
	}{c.CompanyID, string(c.Environment), c.Kind, c.PeriodKey, c.PeriodStart, c.PeriodEnd, c.Summary, c.CreatedAt})
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(append([]byte(c.PrevHash+"|"), body...))
	return hex.EncodeToString(h[:]), nil
}

// InsertClosing records a closing, chained to the last closing of the
// company and environment. It returns ErrConflict if the period is closed.
func (s *Store) InsertClosing(ctx context.Context, c *Closing) error {
	return s.Tx(ctx, func(q Querier) error {
		var prev string
		err := q.QueryRowContext(ctx, `SELECT hash FROM closings WHERE company_id=? AND environment=? ORDER BY id DESC LIMIT 1`,
			c.CompanyID, string(c.Environment)).Scan(&prev)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		c.PrevHash, c.CreatedAt = prev, now()
		if c.Hash, err = closingHash(c); err != nil {
			return err
		}
		res, err := q.ExecContext(ctx, `INSERT INTO closings(company_id, environment, kind, period_key, period_start, period_end, summary_json, hash, prev_hash, created_at)
			VALUES(?,?,?,?,?,?,?,?,?,?)`, c.CompanyID, string(c.Environment), c.Kind, c.PeriodKey, c.PeriodStart, c.PeriodEnd,
			toJSON(c.Summary), c.Hash, c.PrevHash, c.CreatedAt)
		if isUnique(err) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		c.ID, err = res.LastInsertId()
		return err
	})
}

const closingCols = `id, company_id, environment, kind, period_key, period_start, period_end, summary_json, hash, prev_hash, created_at`

func scanClosing(row interface{ Scan(...any) error }) (*Closing, error) {
	var c Closing
	var env, summary string
	if err := row.Scan(&c.ID, &c.CompanyID, &env, &c.Kind, &c.PeriodKey, &c.PeriodStart, &c.PeriodEnd, &summary, &c.Hash, &c.PrevHash, &c.CreatedAt); err != nil {
		return nil, err
	}
	c.Environment = domain.Environment(env)
	_ = json.Unmarshal([]byte(summary), &c.Summary)
	return &c, nil
}

// ListClosings returns a company's closings of one kind (all kinds when
// kind is empty), newest first.
func (s *Store) ListClosings(ctx context.Context, companyID int64, env domain.Environment, kind string, limit int) ([]*Closing, error) {
	if limit <= 0 || limit > 400 {
		limit = 60
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+closingCols+` FROM closings WHERE company_id=? AND environment=? AND (?='' OR kind=?)
		ORDER BY period_start DESC, id DESC LIMIT ?`, companyID, string(env), kind, kind, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Closing
	for rows.Next() {
		c, err := scanClosing(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// LastClosing returns the latest closing of a kind, or nil.
func (s *Store) LastClosing(ctx context.Context, companyID int64, env domain.Environment, kind string) (*Closing, error) {
	c, err := scanClosing(s.DB.QueryRowContext(ctx, `SELECT `+closingCols+` FROM closings WHERE company_id=? AND environment=? AND kind=?
		ORDER BY period_start DESC, id DESC LIMIT 1`, companyID, string(env), kind))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// VerifyClosings recomputes the hash chain of a company's closings in an
// environment. It returns the id of the first closing that does not match
// (0 when the chain is intact) and the number checked.
func (s *Store) VerifyClosings(ctx context.Context, companyID int64, env domain.Environment) (int64, int, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+closingCols+` FROM closings WHERE company_id=? AND environment=? ORDER BY id`, companyID, string(env))
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	prev, n := "", 0
	for rows.Next() {
		c, err := scanClosing(rows)
		if err != nil {
			return 0, n, err
		}
		n++
		want, err := closingHash(c)
		if err != nil {
			return 0, n, err
		}
		if c.PrevHash != prev || c.Hash != want {
			return c.ID, n, nil
		}
		prev = c.Hash
	}
	return 0, n, rows.Err()
}
