// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

// Reference data cache ----------------------------------------------------------

// RefEntry is a cached reference API response.
type RefEntry struct {
	Key       string `json:"key"`
	Kind      string `json:"kind"`
	Data      string `json:"data"`
	Source    string `json:"source"`
	FetchedAt string `json:"fetchedAt"`
}

// GetRef returns a cached reference response.
func (s *Store) GetRef(ctx context.Context, key string) (*RefEntry, error) {
	var e RefEntry
	err := s.DB.QueryRowContext(ctx, `SELECT cache_key, kind, data, source, fetched_at FROM ref_cache WHERE cache_key=?`, key).
		Scan(&e.Key, &e.Kind, &e.Data, &e.Source, &e.FetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}

// PutRef stores a reference response.
func (s *Store) PutRef(ctx context.Context, key, kind, data, source string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO ref_cache(cache_key, kind, data, source, fetched_at) VALUES(?,?,?,?,?)
		ON CONFLICT(cache_key) DO UPDATE SET kind=excluded.kind, data=excluded.data, source=excluded.source, fetched_at=excluded.fetched_at`,
		key, kind, data, source, now())
	return err
}

// RefStatus lists cached keys (for the sync status screen).
func (s *Store) RefStatus(ctx context.Context) ([]RefEntry, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT cache_key, kind, '', source, fetched_at FROM ref_cache ORDER BY cache_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []RefEntry
	for rows.Next() {
		var e RefEntry
		if err := rows.Scan(&e.Key, &e.Kind, &e.Data, &e.Source, &e.FetchedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// HSCode is a searchable HS (PCT) code.
type HSCode struct {
	Code        string `json:"code"`
	Description string `json:"description"`
	// Context is the tariff heading and groups above the code, e.g.
	// "Live bovine animals › Cattle".
	Context string `json:"context,omitempty"`
	// InFBR is set when the code is on the list downloaded from FBR's
	// Digital Invoicing API.
	InFBR bool `json:"inFbr"`
}

// LoadTariffHSCodes stores the built-in Pakistan Customs Tariff codes. The
// descriptions of the tariff are kept; whether a code is on FBR's list is
// left as it is.
func (s *Store) LoadTariffHSCodes(ctx context.Context, codes []HSCode) error {
	return s.Tx(ctx, func(q Querier) error {
		if _, err := q.ExecContext(ctx, `DELETE FROM ref_hs_codes WHERE source='seed'`); err != nil {
			return err
		}
		for _, c := range codes {
			if _, err := q.ExecContext(ctx, `INSERT INTO ref_hs_codes(code, description, context, source, in_fbr) VALUES(?,?,?,'pct',0)
				ON CONFLICT(code) DO UPDATE SET description=excluded.description, context=excluded.context, source='pct'`,
				c.Code, c.Description, c.Context); err != nil {
				return err
			}
		}
		return nil
	})
}

// ReplaceHSCodes records the list downloaded from FBR (source "fbr"): its
// codes are marked as on FBR's list, codes FBR added are stored with FBR's
// description, and the tariff's own descriptions are kept.
func (s *Store) ReplaceHSCodes(ctx context.Context, codes []HSCode, source string) error {
	return s.Tx(ctx, func(q Querier) error {
		if source == "fbr" {
			if _, err := q.ExecContext(ctx, `UPDATE ref_hs_codes SET in_fbr=0`); err != nil {
				return err
			}
		}
		inFBR := 0
		if source == "fbr" {
			inFBR = 1
		}
		for _, c := range codes {
			if _, err := q.ExecContext(ctx, `INSERT INTO ref_hs_codes(code, description, source, in_fbr) VALUES(?,?,?,?)
				ON CONFLICT(code) DO UPDATE SET in_fbr=excluded.in_fbr,
					description=CASE WHEN ref_hs_codes.source='pct' THEN ref_hs_codes.description ELSE excluded.description END`,
				c.Code, c.Description, source, inFBR); err != nil {
				return err
			}
		}
		return nil
	})
}

// SearchHSCodes finds codes by number ("7214", "7214.99" or "721499") or
// by words in the description and the heading above it ("steel bars").
func (s *Store) SearchHSCodes(ctx context.Context, q string, limit int) ([]HSCode, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q = strings.TrimSpace(q)
	var where []string
	var args []any
	digits := strings.ReplaceAll(q, ".", "")
	isNumber := q != "" && strings.Trim(digits, "0123456789") == ""
	prefix := q
	if isNumber && len(digits) > 4 && !strings.Contains(q, ".") {
		prefix = digits[:4] + "." + digits[4:]
	}
	if isNumber {
		where = append(where, `code LIKE ?`)
		args = append(args, prefix+"%")
	} else {
		for _, w := range strings.Fields(q) {
			where = append(where, `(description LIKE ? OR context LIKE ?)`)
			args = append(args, "%"+w+"%", "%"+w+"%")
		}
	}
	if len(where) == 0 {
		return []HSCode{}, nil
	}
	// Codes whose own description matches come before those matched by
	// their heading only.
	first := "%" + q + "%"
	args = append(args, first, limit)
	rows, err := s.DB.QueryContext(ctx, `SELECT code, description, context, in_fbr FROM ref_hs_codes
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY CASE WHEN description LIKE ? THEN 0 ELSE 1 END, code LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HSCode{}
	for rows.Next() {
		var c HSCode
		if err := rows.Scan(&c.Code, &c.Description, &c.Context, &c.InFBR); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetHSCode returns one stored code.
func (s *Store) GetHSCode(ctx context.Context, code string) (*HSCode, error) {
	var c HSCode
	err := s.DB.QueryRowContext(ctx, `SELECT code, description, context, in_fbr FROM ref_hs_codes WHERE code=?`, strings.TrimSpace(code)).
		Scan(&c.Code, &c.Description, &c.Context, &c.InFBR)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &c, err
}

// CountHSCodes returns the number of stored HS codes.
func (s *Store) CountHSCodes(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM ref_hs_codes`).Scan(&n)
	return n, err
}

// Scenario runs -------------------------------------------------------------------

// ScenarioRun records a sandbox scenario attempt.
type ScenarioRun struct {
	ID               int64  `json:"id"`
	CompanyID        int64  `json:"companyId"`
	ScenarioID       string `json:"scenarioId"`
	InvoiceID        *int64 `json:"invoiceId"`
	Mode             string `json:"mode"`
	Status           string `json:"status"` // passed | failed | error
	FBRInvoiceNumber string `json:"fbrInvoiceNumber"`
	Message          string `json:"message"`
	RunAt            string `json:"runAt"`
	RunBy            *int64 `json:"runBy"`
}

// AddScenarioRun stores a run.
func (s *Store) AddScenarioRun(ctx context.Context, r *ScenarioRun) error {
	r.RunAt = now()
	res, err := s.DB.ExecContext(ctx, `INSERT INTO scenario_runs(company_id, scenario_id, invoice_id, mode, status, fbr_invoice_number, message, run_at, run_by)
		VALUES(?,?,?,?,?,?,?,?,?)`, r.CompanyID, r.ScenarioID, nullInt(r.InvoiceID), r.Mode, r.Status, r.FBRInvoiceNumber, r.Message, r.RunAt, nullInt(r.RunBy))
	if err != nil {
		return err
	}
	r.ID, _ = res.LastInsertId()
	return nil
}

// LatestScenarioRuns returns the most recent run per scenario and whether
// each scenario has ever passed.
func (s *Store) LatestScenarioRuns(ctx context.Context, companyID int64) (map[string]*ScenarioRun, map[string]bool, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, company_id, scenario_id, invoice_id, mode, status, fbr_invoice_number, message, run_at, run_by
		FROM scenario_runs WHERE company_id=? ORDER BY id`, companyID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	latest := map[string]*ScenarioRun{}
	passed := map[string]bool{}
	for rows.Next() {
		var r ScenarioRun
		var inv, by sql.NullInt64
		if err := rows.Scan(&r.ID, &r.CompanyID, &r.ScenarioID, &inv, &r.Mode, &r.Status, &r.FBRInvoiceNumber, &r.Message, &r.RunAt, &by); err != nil {
			return nil, nil, err
		}
		r.InvoiceID, r.RunBy = intPtr(inv), intPtr(by)
		rr := r
		latest[r.ScenarioID] = &rr
		if r.Status == "passed" {
			passed[r.ScenarioID] = true
		}
	}
	return latest, passed, rows.Err()
}

// Incidents ------------------------------------------------------------------------

// Incident is an operational failure of the e-invoicing system. Rule
// 150XA(c) requires the integrated person to report any operational failure, damage,
// disruption or tampering to the Board and the Commissioner within 24 hours.
type Incident struct {
	ID              int64  `json:"id"`
	CompanyID       int64  `json:"companyId"`
	Kind            string `json:"kind"` // fbr_unreachable | auth_failure | system_failure | power_failure | tampering | other
	Description     string `json:"description"`
	StartedAt       string `json:"startedAt"`
	EndedAt         string `json:"endedAt"`
	AutoDetected    bool   `json:"autoDetected"`
	ReportedAt      string `json:"reportedAt"`
	ReportReference string `json:"reportReference"`
	CreatedBy       *int64 `json:"createdBy"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

const incidentCols = `id, company_id, kind, description, started_at, ended_at, auto_detected, reported_at, report_reference, created_by, created_at, updated_at`

func scanIncident(row interface{ Scan(...any) error }) (*Incident, error) {
	var i Incident
	var auto int
	var by sql.NullInt64
	err := row.Scan(&i.ID, &i.CompanyID, &i.Kind, &i.Description, &i.StartedAt, &i.EndedAt, &auto, &i.ReportedAt, &i.ReportReference, &by, &i.CreatedAt, &i.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	i.AutoDetected, i.CreatedBy = auto == 1, intPtr(by)
	return &i, nil
}

// SaveIncident inserts or updates an incident.
func (s *Store) SaveIncident(ctx context.Context, i *Incident) error {
	t := now()
	if i.ID == 0 {
		res, err := s.DB.ExecContext(ctx, `INSERT INTO incidents(company_id, kind, description, started_at, ended_at, auto_detected, reported_at,
			report_reference, created_by, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
			i.CompanyID, i.Kind, i.Description, i.StartedAt, i.EndedAt, b2i(i.AutoDetected), i.ReportedAt, i.ReportReference, nullInt(i.CreatedBy), t, t)
		if err != nil {
			return err
		}
		i.ID, _ = res.LastInsertId()
		i.CreatedAt, i.UpdatedAt = t, t
		return nil
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE incidents SET kind=?, description=?, started_at=?, ended_at=?, reported_at=?, report_reference=?, updated_at=?
		WHERE id=? AND company_id=?`, i.Kind, i.Description, i.StartedAt, i.EndedAt, i.ReportedAt, i.ReportReference, t, i.ID, i.CompanyID)
	return err
}

// GetIncident loads an incident.
func (s *Store) GetIncident(ctx context.Context, companyID, id int64) (*Incident, error) {
	return scanIncident(s.DB.QueryRowContext(ctx, `SELECT `+incidentCols+` FROM incidents WHERE id=? AND company_id=?`, id, companyID))
}

// OpenAutoIncident returns the open auto-detected incident of a kind, if any.
func (s *Store) OpenAutoIncident(ctx context.Context, companyID int64, kind string) (*Incident, error) {
	return scanIncident(s.DB.QueryRowContext(ctx, `SELECT `+incidentCols+` FROM incidents WHERE company_id=? AND kind=? AND auto_detected=1 AND ended_at=''
		ORDER BY id DESC LIMIT 1`, companyID, kind))
}

// ListIncidents lists a company's incidents.
func (s *Store) ListIncidents(ctx context.Context, companyID int64, limit int) ([]*Incident, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT `+incidentCols+` FROM incidents WHERE company_id=? ORDER BY started_at DESC LIMIT ?`, companyID, limit)
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

// InvoiceBrief is a short invoice reference used in letters and lists.
type InvoiceBrief struct {
	ID               int64  `json:"id"`
	InternalNo       string `json:"internalNo"`
	InvoiceDate      string `json:"invoiceDate"`
	Status           string `json:"status"`
	FBRInvoiceNumber string `json:"fbrInvoiceNumber"`
	AcceptedAt       string `json:"acceptedAt"`
}

// InvoicesCreatedBetween lists invoices created in a time window (RFC 3339).
func (s *Store) InvoicesCreatedBetween(ctx context.Context, companyID int64, from, to string) ([]InvoiceBrief, error) {
	if to == "" {
		to = now()
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, internal_no, invoice_date, status, fbr_invoice_number, accepted_at FROM invoices
		WHERE company_id=? AND environment='production' AND created_at>=? AND created_at<=? ORDER BY id`, companyID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []InvoiceBrief
	for rows.Next() {
		var b InvoiceBrief
		if err := rows.Scan(&b.ID, &b.InternalNo, &b.InvoiceDate, &b.Status, &b.FBRInvoiceNumber, &b.AcceptedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
