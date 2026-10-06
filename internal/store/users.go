package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Roles. Permissions are enforced in the HTTP layer (see httpapi/perms.go).
const (
	RoleAdmin      = "admin"      // everything, including users, licence, backups
	RoleManager    = "manager"    // company settings, masters, invoices, reports
	RoleAccountant = "accountant" // masters, invoices (create/submit/debit notes), reports
	RoleOperator   = "operator"   // create and submit invoices, view masters
	RoleAuditor    = "auditor"    // read-only, including audit log
)

// ValidRole reports whether r is a known role.
func ValidRole(r string) bool {
	switch r {
	case RoleAdmin, RoleManager, RoleAccountant, RoleOperator, RoleAuditor:
		return true
	}
	return false
}

// User is an application user.
type User struct {
	ID                 int64   `json:"id"`
	Username           string  `json:"username"`
	FullName           string  `json:"fullName"`
	Email              string  `json:"email"`
	PasswordHash       string  `json:"-"`
	Role               string  `json:"role"`
	AllCompanies       bool    `json:"allCompanies"`
	CompanyIDs         []int64 `json:"companyIds"`
	Active             bool    `json:"active"`
	MustChangePassword bool    `json:"mustChangePassword"`
	FailedLogins       int     `json:"-"`
	LockedUntil        string  `json:"lockedUntil"`
	LastLoginAt        string  `json:"lastLoginAt"`
	CreatedAt          string  `json:"createdAt"`
}

// CanAccessCompany reports whether the user may work with a company.
func (u *User) CanAccessCompany(id int64) bool {
	if u.AllCompanies || u.Role == RoleAdmin {
		return true
	}
	for _, c := range u.CompanyIDs {
		if c == id {
			return true
		}
	}
	return false
}

const userCols = `id, username, full_name, email, password_hash, role, all_companies, active, must_change_password,
	failed_logins, locked_until, last_login_at, created_at`

func (s *Store) scanUser(ctx context.Context, row interface{ Scan(...any) error }) (*User, error) {
	var u User
	var all, active, must int
	err := row.Scan(&u.ID, &u.Username, &u.FullName, &u.Email, &u.PasswordHash, &u.Role, &all, &active, &must,
		&u.FailedLogins, &u.LockedUntil, &u.LastLoginAt, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.AllCompanies, u.Active, u.MustChangePassword = all == 1, active == 1, must == 1
	return &u, nil
}

func (s *Store) loadUserCompanies(ctx context.Context, u *User) error {
	rows, err := s.DB.QueryContext(ctx, `SELECT company_id FROM user_companies WHERE user_id=? ORDER BY company_id`, u.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	u.CompanyIDs = []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		u.CompanyIDs = append(u.CompanyIDs, id)
	}
	return rows.Err()
}

// GetUser loads a user by id.
func (s *Store) GetUser(ctx context.Context, id int64) (*User, error) {
	u, err := s.scanUser(ctx, s.DB.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE id=?`, id))
	if err != nil {
		return nil, err
	}
	return u, s.loadUserCompanies(ctx, u)
}

// GetUserByUsername loads a user by username (case-insensitive).
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	u, err := s.scanUser(ctx, s.DB.QueryRowContext(ctx, `SELECT `+userCols+` FROM users WHERE username=?`, username))
	if err != nil {
		return nil, err
	}
	return u, s.loadUserCompanies(ctx, u)
}

// ListUsers returns all users.
func (s *Store) ListUsers(ctx context.Context) ([]*User, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT `+userCols+` FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	var out []*User
	for rows.Next() {
		u, err := s.scanUser(ctx, rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, u)
	}
	rows.Close()
	for _, u := range out {
		if err := s.loadUserCompanies(ctx, u); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// CountUsers returns the number of users.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

// CreateUser inserts a user.
func (s *Store) CreateUser(ctx context.Context, u *User) (int64, error) {
	t := now()
	var id int64
	err := s.Tx(ctx, func(q Querier) error {
		res, err := q.ExecContext(ctx, `INSERT INTO users(username, full_name, email, password_hash, role, all_companies, active,
			must_change_password, created_at, updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			u.Username, u.FullName, u.Email, u.PasswordHash, u.Role, b2i(u.AllCompanies), b2i(u.Active), b2i(u.MustChangePassword), t, t)
		if isUnique(err) {
			return ErrConflict
		}
		if err != nil {
			return err
		}
		id, _ = res.LastInsertId()
		return setUserCompanies(ctx, q, id, u.CompanyIDs)
	})
	return id, err
}

func setUserCompanies(ctx context.Context, q Querier, userID int64, ids []int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM user_companies WHERE user_id=?`, userID); err != nil {
		return err
	}
	for _, cid := range ids {
		if _, err := q.ExecContext(ctx, `INSERT OR IGNORE INTO user_companies(user_id, company_id) VALUES(?,?)`, userID, cid); err != nil {
			return err
		}
	}
	return nil
}

// UpdateUser saves profile, role, status and company access.
func (s *Store) UpdateUser(ctx context.Context, u *User) error {
	return s.Tx(ctx, func(q Querier) error {
		if _, err := q.ExecContext(ctx, `UPDATE users SET full_name=?, email=?, role=?, all_companies=?, active=?, updated_at=? WHERE id=?`,
			u.FullName, u.Email, u.Role, b2i(u.AllCompanies), b2i(u.Active), now(), u.ID); err != nil {
			return err
		}
		return setUserCompanies(ctx, q, u.ID, u.CompanyIDs)
	})
}

// SetPassword changes a password hash.
func (s *Store) SetPassword(ctx context.Context, userID int64, hash string, mustChange bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET password_hash=?, must_change_password=?, failed_logins=0, locked_until='', updated_at=? WHERE id=?`,
		hash, b2i(mustChange), now(), userID)
	return err
}

// RecordLogin updates login bookkeeping. On failure the account is locked for
// 15 minutes after 5 consecutive failures.
func (s *Store) RecordLogin(ctx context.Context, userID int64, success bool) error {
	if success {
		_, err := s.DB.ExecContext(ctx, `UPDATE users SET failed_logins=0, locked_until='', last_login_at=? WHERE id=?`, now(), userID)
		return err
	}
	_, err := s.DB.ExecContext(ctx, `UPDATE users SET failed_logins=failed_logins+1,
		locked_until=CASE WHEN failed_logins+1 >= 5 THEN ? ELSE locked_until END WHERE id=?`,
		time.Now().UTC().Add(15*time.Minute).Format(time.RFC3339), userID)
	return err
}

// Session -------------------------------------------------------------------

// Session is a logged-in browser session.
type Session struct {
	TokenHash string
	UserID    int64
	CSRFToken string
	ExpiresAt string
}

// CreateSession stores a session.
func (s *Store) CreateSession(ctx context.Context, tokenHash string, userID int64, csrf string, ttl time.Duration, ip, ua string) error {
	t := time.Now().UTC()
	_, err := s.DB.ExecContext(ctx, `INSERT INTO sessions(token_hash, user_id, csrf_token, created_at, expires_at, last_seen_at, ip, user_agent)
		VALUES(?,?,?,?,?,?,?,?)`, tokenHash, userID, csrf, t.Format(time.RFC3339), t.Add(ttl).Format(time.RFC3339), t.Format(time.RFC3339), ip, ua)
	return err
}

// GetSession returns a non-expired session.
func (s *Store) GetSession(ctx context.Context, tokenHash string) (*Session, error) {
	var se Session
	err := s.DB.QueryRowContext(ctx, `SELECT token_hash, user_id, csrf_token, expires_at FROM sessions WHERE token_hash=?`, tokenHash).
		Scan(&se.TokenHash, &se.UserID, &se.CSRFToken, &se.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if ParseTime(se.ExpiresAt).Before(time.Now()) {
		_, _ = s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=?`, tokenHash)
		return nil, ErrNotFound
	}
	return &se, nil
}

// TouchSession extends a session's expiry (sliding expiration).
func (s *Store) TouchSession(ctx context.Context, tokenHash string, ttl time.Duration) error {
	t := time.Now().UTC()
	_, err := s.DB.ExecContext(ctx, `UPDATE sessions SET last_seen_at=?, expires_at=? WHERE token_hash=?`,
		t.Format(time.RFC3339), t.Add(ttl).Format(time.RFC3339), tokenHash)
	return err
}

// DeleteSession removes a session.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash=?`, tokenHash)
	return err
}

// DeleteUserSessions removes all sessions of a user.
func (s *Store) DeleteUserSessions(ctx context.Context, userID int64) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE user_id=?`, userID)
	return err
}

// PurgeExpiredSessions removes expired sessions.
func (s *Store) PurgeExpiredSessions(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, now())
	return err
}

// API keys ------------------------------------------------------------------

// APIKey authorises an external system (ERP/POS) to use the REST API for one company.
type APIKey struct {
	ID         int64  `json:"id"`
	CompanyID  int64  `json:"companyId"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	KeyHash    string `json:"-"`
	CreatedBy  *int64 `json:"createdBy"`
	CreatedAt  string `json:"createdAt"`
	LastUsedAt string `json:"lastUsedAt"`
	RevokedAt  string `json:"revokedAt"`
}

// CreateAPIKey stores a key (hash only).
func (s *Store) CreateAPIKey(ctx context.Context, k *APIKey) (int64, error) {
	res, err := s.DB.ExecContext(ctx, `INSERT INTO api_keys(company_id, name, prefix, key_hash, created_by, created_at) VALUES(?,?,?,?,?,?)`,
		k.CompanyID, k.Name, k.Prefix, k.KeyHash, nullInt(k.CreatedBy), now())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListAPIKeys lists a company's keys.
func (s *Store) ListAPIKeys(ctx context.Context, companyID int64) ([]*APIKey, error) {
	rows, err := s.DB.QueryContext(ctx, `SELECT id, company_id, name, prefix, key_hash, created_by, created_at, last_used_at, revoked_at
		FROM api_keys WHERE company_id=? ORDER BY id DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*APIKey
	for rows.Next() {
		var k APIKey
		var cb sql.NullInt64
		if err := rows.Scan(&k.ID, &k.CompanyID, &k.Name, &k.Prefix, &k.KeyHash, &cb, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		k.CreatedBy = intPtr(cb)
		out = append(out, &k)
	}
	return out, rows.Err()
}

// FindAPIKey returns an active key by hash.
func (s *Store) FindAPIKey(ctx context.Context, hash string) (*APIKey, error) {
	var k APIKey
	var cb sql.NullInt64
	err := s.DB.QueryRowContext(ctx, `SELECT id, company_id, name, prefix, key_hash, created_by, created_at, last_used_at, revoked_at
		FROM api_keys WHERE key_hash=? AND revoked_at=''`, hash).
		Scan(&k.ID, &k.CompanyID, &k.Name, &k.Prefix, &k.KeyHash, &cb, &k.CreatedAt, &k.LastUsedAt, &k.RevokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	k.CreatedBy = intPtr(cb)
	_, _ = s.DB.ExecContext(ctx, `UPDATE api_keys SET last_used_at=? WHERE id=?`, now(), k.ID)
	return &k, nil
}

// RevokeAPIKey revokes a key.
func (s *Store) RevokeAPIKey(ctx context.Context, companyID, id int64) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE api_keys SET revoked_at=? WHERE id=? AND company_id=? AND revoked_at=''`, now(), id, companyID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
