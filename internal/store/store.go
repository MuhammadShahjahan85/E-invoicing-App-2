// Package store contains the data access layer (models and repositories).
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"einvoicing/internal/db"

	"github.com/shopspring/decimal"
)

func init() {
	// API responses carry amounts as JSON numbers.
	decimal.MarshalJSONWithoutQuotes = true
}

// ErrNotFound is returned when a record does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned on unique constraint violations.
var ErrConflict = errors.New("already exists")

// Store wraps the database handle.
type Store struct {
	DB *sql.DB
}

// Querier is satisfied by *sql.DB and *sql.Tx.
type Querier = db.Querier

// New creates a store.
func New(d *sql.DB) *Store { return &Store{DB: d} }

// Tx runs fn inside a transaction.
func (s *Store) Tx(ctx context.Context, fn func(q Querier) error) error {
	return db.InTx(ctx, s.DB, func(tx *sql.Tx) error { return fn(tx) })
}

// Paisa converts a rupee amount to integer paisa (rounded half away from zero).
func Paisa(d decimal.Decimal) int64 { return d.Round(2).Shift(2).IntPart() }

// Rupees converts integer paisa to a decimal rupee amount.
func Rupees(p int64) decimal.Decimal { return decimal.New(p, -2) }

func dec(s string) decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero
	}
	return d
}

func decPtr(s string) *decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return nil
	}
	return &d
}

func ptrStr(d *decimal.Decimal) string {
	if d == nil {
		return ""
	}
	return d.String()
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func jsonList(s string) []string {
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	if out == nil {
		out = []string{}
	}
	return out
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func nullInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

func intPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func isUnique(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "unique constraint")
}

func now() string { return db.Now() }

// ParseTime parses a stored timestamp.
func ParseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// Settings ------------------------------------------------------------------

// GetSetting returns a global setting value ("" when missing).
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v string
	err := s.DB.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// SetSetting stores a global setting.
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.DB.ExecContext(ctx, `INSERT INTO settings(key, value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}
