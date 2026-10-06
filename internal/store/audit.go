package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"einvoicing/internal/security"
)

// AuditEntry is an append-only, hash-chained audit record.
type AuditEntry struct {
	ID        int64  `json:"id"`
	TS        string `json:"ts"`
	UserID    *int64 `json:"userId"`
	Username  string `json:"username"`
	CompanyID *int64 `json:"companyId"`
	Action    string `json:"action"`
	Entity    string `json:"entity"`
	EntityID  string `json:"entityId"`
	Details   string `json:"details"`
	IP        string `json:"ip"`
	PrevHash  string `json:"prevHash"`
	Hash      string `json:"hash"`
}

var auditMu sync.Mutex

func auditPayload(e *AuditEntry) string {
	uid, cid := "", ""
	if e.UserID != nil {
		uid = fmt.Sprint(*e.UserID)
	}
	if e.CompanyID != nil {
		cid = fmt.Sprint(*e.CompanyID)
	}
	return strings.Join([]string{e.TS, uid, e.Username, cid, e.Action, e.Entity, e.EntityID, e.Details, e.IP}, "\x1f")
}

// AppendAudit writes an audit record linked to the previous one.
func (s *Store) AppendAudit(ctx context.Context, e *AuditEntry) error {
	auditMu.Lock()
	defer auditMu.Unlock()
	return s.Tx(ctx, func(q Querier) error {
		var prev string
		err := q.QueryRowContext(ctx, `SELECT hash FROM audit_log ORDER BY id DESC LIMIT 1`).Scan(&prev)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		e.TS = now()
		e.PrevHash = prev
		e.Hash = security.ChainHash(prev, auditPayload(e))
		res, err := q.ExecContext(ctx, `INSERT INTO audit_log(ts, user_id, username, company_id, action, entity, entity_id, details, ip, prev_hash, hash)
			VALUES(?,?,?,?,?,?,?,?,?,?,?)`, e.TS, nullInt(e.UserID), e.Username, nullInt(e.CompanyID), e.Action, e.Entity, e.EntityID,
			e.Details, e.IP, e.PrevHash, e.Hash)
		if err != nil {
			return err
		}
		e.ID, _ = res.LastInsertId()
		return nil
	})
}

// AuditFilter filters the audit log.
type AuditFilter struct {
	CompanyID int64
	Entity    string
	EntityID  string
	Action    string
	From, To  string
	Limit     int
	Offset    int
}

// ListAudit returns audit records, newest first.
func (s *Store) ListAudit(ctx context.Context, f AuditFilter) ([]*AuditEntry, int, error) {
	where := []string{"1=1"}
	var args []any
	if f.CompanyID > 0 {
		where = append(where, "company_id=?")
		args = append(args, f.CompanyID)
	}
	if f.Entity != "" {
		where = append(where, "entity=?")
		args = append(args, f.Entity)
	}
	if f.EntityID != "" {
		where = append(where, "entity_id=?")
		args = append(args, f.EntityID)
	}
	if f.Action != "" {
		where = append(where, "action LIKE ?")
		args = append(args, f.Action+"%")
	}
	if f.From != "" {
		where = append(where, "ts>=?")
		args = append(args, f.From)
	}
	if f.To != "" {
		where = append(where, "ts<=?")
		args = append(args, f.To+"T23:59:59Z")
	}
	w := strings.Join(where, " AND ")
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log WHERE `+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if f.Limit <= 0 || f.Limit > 1000 {
		f.Limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, ts, user_id, username, company_id, action, entity, entity_id, details, ip, prev_hash, hash
		FROM audit_log WHERE `+w+` ORDER BY id DESC LIMIT ? OFFSET ?`, append(args, f.Limit, f.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*AuditEntry
	for rows.Next() {
		var e AuditEntry
		var uid, cid sql.NullInt64
		if err := rows.Scan(&e.ID, &e.TS, &uid, &e.Username, &cid, &e.Action, &e.Entity, &e.EntityID, &e.Details, &e.IP, &e.PrevHash, &e.Hash); err != nil {
			return nil, 0, err
		}
		e.UserID, e.CompanyID = intPtr(uid), intPtr(cid)
		out = append(out, &e)
	}
	return out, total, rows.Err()
}

// VerifyAuditChain recomputes the hash chain. It returns the id of the first
// broken record (0 when the chain is intact) and the number checked.
func (s *Store) VerifyAuditChain(ctx context.Context) (brokenAt int64, checked int, err error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, ts, user_id, username, company_id, action, entity, entity_id, details, ip, prev_hash, hash
		FROM audit_log ORDER BY id`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	prev := ""
	for rows.Next() {
		var e AuditEntry
		var uid, cid sql.NullInt64
		if err := rows.Scan(&e.ID, &e.TS, &uid, &e.Username, &cid, &e.Action, &e.Entity, &e.EntityID, &e.Details, &e.IP, &e.PrevHash, &e.Hash); err != nil {
			return 0, checked, err
		}
		e.UserID, e.CompanyID = intPtr(uid), intPtr(cid)
		checked++
		if e.PrevHash != prev || security.ChainHash(prev, auditPayload(&e)) != e.Hash {
			return e.ID, checked, nil
		}
		prev = e.Hash
	}
	return 0, checked, rows.Err()
}

// FBRCall is a logged exchange with FBR (token never stored).
type FBRCall struct {
	ID           int64  `json:"id"`
	CompanyID    int64  `json:"companyId"`
	InvoiceID    *int64 `json:"invoiceId"`
	Environment  string `json:"environment"`
	Operation    string `json:"operation"`
	Method       string `json:"method"`
	URL          string `json:"url"`
	RequestBody  string `json:"requestBody"`
	ResponseBody string `json:"responseBody"`
	HTTPStatus   int    `json:"httpStatus"`
	DurationMS   int64  `json:"durationMs"`
	ErrorKind    string `json:"errorKind"`
	Error        string `json:"error"`
	CreatedAt    string `json:"createdAt"`
}

// LogFBRCall stores an exchange.
func (s *Store) LogFBRCall(ctx context.Context, c *FBRCall) error {
	res, err := s.DB.ExecContext(ctx, `INSERT INTO fbr_calls(company_id, invoice_id, environment, operation, method, url, request_body,
		response_body, http_status, duration_ms, error_kind, error, created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		c.CompanyID, nullInt(c.InvoiceID), c.Environment, c.Operation, c.Method, c.URL, c.RequestBody, c.ResponseBody, c.HTTPStatus,
		c.DurationMS, c.ErrorKind, c.Error, now())
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

// ListFBRCalls lists exchanges for a company or invoice.
func (s *Store) ListFBRCalls(ctx context.Context, companyID int64, invoiceID int64, limit, offset int) ([]*FBRCall, int, error) {
	where := "company_id=?"
	args := []any{companyID}
	if invoiceID > 0 {
		where += " AND invoice_id=?"
		args = append(args, invoiceID)
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM fbr_calls WHERE `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `SELECT id, company_id, invoice_id, environment, operation, method, url, request_body, response_body,
		http_status, duration_ms, error_kind, error, created_at FROM fbr_calls WHERE `+where+` ORDER BY id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []*FBRCall
	for rows.Next() {
		var c FBRCall
		var inv sql.NullInt64
		if err := rows.Scan(&c.ID, &c.CompanyID, &inv, &c.Environment, &c.Operation, &c.Method, &c.URL, &c.RequestBody, &c.ResponseBody,
			&c.HTTPStatus, &c.DurationMS, &c.ErrorKind, &c.Error, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		c.InvoiceID = intPtr(inv)
		out = append(out, &c)
	}
	return out, total, rows.Err()
}
