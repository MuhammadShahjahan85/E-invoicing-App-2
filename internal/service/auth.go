package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"einvoicing/internal/security"
	"einvoicing/internal/store"
)

// SessionTTL is the sliding session lifetime.
var SessionTTL = 12 * time.Hour

var (
	dummyOnce sync.Once
	dummyVal  string
)

// dummyHash is a real bcrypt hash compared against for unknown usernames so
// that response timing does not reveal which usernames exist.
func dummyHash() string {
	dummyOnce.Do(func() { dummyVal, _ = security.HashPassword(security.RandomToken(16)) })
	return dummyVal
}

// ErrAuth is returned for invalid credentials.
var ErrAuth = errors.New("invalid username or password (after 5 failed attempts an account is locked for 15 minutes)")

// ErrLocked is returned for locked accounts.
// It carries the same message as ErrAuth so that responses do not reveal
// which usernames exist or are locked.
var ErrLocked = errors.New("invalid username or password (after 5 failed attempts an account is locked for 15 minutes)")

// SetupInput is the first-run wizard payload.
type SetupInput struct {
	AdminUsername string        `json:"adminUsername"`
	AdminPassword string        `json:"adminPassword"`
	AdminFullName string        `json:"adminFullName"`
	Company       store.Company `json:"company"`
}

// NeedsSetup reports whether no user exists yet.
func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	n, err := s.Store.CountUsers(ctx)
	return n == 0, err
}

// Setup creates the first administrator and company.
func (s *Service) Setup(ctx context.Context, ip string, in SetupInput) error {
	need, err := s.NeedsSetup(ctx)
	if err != nil {
		return err
	}
	if !need {
		return Invalid("setup has already been completed")
	}
	in.AdminUsername = strings.TrimSpace(in.AdminUsername)
	if in.AdminUsername == "" {
		return Invalid("administrator username is required")
	}
	if err := security.PasswordPolicy(in.AdminPassword); err != nil {
		return Invalid("%v", err)
	}
	a := Actor{Username: in.AdminUsername, Role: store.RoleAdmin, IP: ip}
	comp := in.Company
	if _, err := s.SaveCompany(ctx, a, &comp); err != nil {
		return err
	}
	hash, err := security.HashPassword(in.AdminPassword)
	if err != nil {
		return err
	}
	u := &store.User{Username: in.AdminUsername, FullName: in.AdminFullName, PasswordHash: hash, Role: store.RoleAdmin, AllCompanies: true, Active: true}
	id, err := s.Store.CreateUser(ctx, u)
	if err != nil {
		return err
	}
	a.UserID = &id
	s.Audit(ctx, a, comp.ID, "system.setup", "user", fmt.Sprint(id), map[string]any{"admin": u.Username, "company": comp.Name})
	return nil
}

// LoginResult is returned on successful login.
type LoginResult struct {
	Token string
	CSRF  string
	User  *store.User
}

// Login verifies credentials and creates a session.
func (s *Service) Login(ctx context.Context, username, password, ip, ua string) (*LoginResult, error) {
	u, err := s.Store.GetUserByUsername(ctx, strings.TrimSpace(username))
	if errors.Is(err, store.ErrNotFound) {
		security.CheckPassword(dummyHash(), password) // equalise timing with existing users
		s.Audit(ctx, Actor{Username: username, IP: ip}, 0, "auth.login_failed", "user", "", "unknown user")
		return nil, ErrAuth
	}
	if err != nil {
		return nil, err
	}
	if u.LockedUntil != "" && store.ParseTime(u.LockedUntil).After(time.Now()) {
		security.CheckPassword(u.PasswordHash, password) // same timing as other failures
		s.Audit(ctx, Actor{UserID: &u.ID, Username: u.Username, IP: ip}, 0, "auth.login_failed", "user", fmt.Sprint(u.ID), "account locked")
		return nil, ErrLocked
	}
	if !u.Active || !security.CheckPassword(u.PasswordHash, password) {
		_ = s.Store.RecordLogin(ctx, u.ID, false)
		s.Audit(ctx, Actor{UserID: &u.ID, Username: u.Username, IP: ip}, 0, "auth.login_failed", "user", fmt.Sprint(u.ID), "bad password or inactive")
		return nil, ErrAuth
	}
	_ = s.Store.RecordLogin(ctx, u.ID, true)
	tok := security.RandomToken(32)
	csrf := security.RandomToken(24)
	if err := s.Store.CreateSession(ctx, security.SHA256Hex(tok), u.ID, csrf, SessionTTL, ip, truncate(ua, 300)); err != nil {
		return nil, err
	}
	s.Audit(ctx, Actor{UserID: &u.ID, Username: u.Username, IP: ip}, 0, "auth.login", "user", fmt.Sprint(u.ID), nil)
	return &LoginResult{Token: tok, CSRF: csrf, User: u}, nil
}

// SessionUser resolves a session token.
func (s *Service) SessionUser(ctx context.Context, token string) (*store.User, *store.Session, error) {
	if token == "" {
		return nil, nil, store.ErrNotFound
	}
	h := security.SHA256Hex(token)
	se, err := s.Store.GetSession(ctx, h)
	if err != nil {
		return nil, nil, err
	}
	u, err := s.Store.GetUser(ctx, se.UserID)
	if err != nil {
		return nil, nil, err
	}
	if !u.Active {
		_ = s.Store.DeleteSession(ctx, h)
		return nil, nil, store.ErrNotFound
	}
	_ = s.Store.TouchSession(ctx, h, SessionTTL)
	return u, se, nil
}

// Logout ends a session.
func (s *Service) Logout(ctx context.Context, a Actor, token string) error {
	s.Audit(ctx, a, 0, "auth.logout", "user", "", nil)
	return s.Store.DeleteSession(ctx, security.SHA256Hex(token))
}

// ChangePassword changes the current user's password.
// keepToken is the caller's own session token, which stays valid; every other
// session of the user is revoked.
func (s *Service) ChangePassword(ctx context.Context, a Actor, userID int64, oldPw, newPw, keepToken string) error {
	u, err := s.Store.GetUser(ctx, userID)
	if err != nil {
		return err
	}
	if !security.CheckPassword(u.PasswordHash, oldPw) {
		return Invalid("current password is incorrect")
	}
	if err := security.PasswordPolicy(newPw); err != nil {
		return Invalid("%v", err)
	}
	hash, err := security.HashPassword(newPw)
	if err != nil {
		return err
	}
	if err := s.Store.SetPassword(ctx, userID, hash, false); err != nil {
		return err
	}
	if err := s.Store.DeleteOtherSessions(ctx, userID, security.SHA256Hex(keepToken)); err != nil {
		return err
	}
	s.Audit(ctx, a, 0, "user.password_changed", "user", fmt.Sprint(userID), nil)
	return nil
}

// UserInput creates or updates a user.
type UserInput struct {
	Username     string  `json:"username"`
	FullName     string  `json:"fullName"`
	Email        string  `json:"email"`
	Role         string  `json:"role"`
	Password     string  `json:"password"`
	AllCompanies bool    `json:"allCompanies"`
	CompanyIDs   []int64 `json:"companyIds"`
	Active       *bool   `json:"active"`
}

// SaveUser creates (id==0) or updates a user.
func (s *Service) SaveUser(ctx context.Context, a Actor, id int64, in UserInput) (*store.User, error) {
	if !store.ValidRole(in.Role) {
		return nil, Invalid("unknown role %q", in.Role)
	}
	if id == 0 {
		in.Username = strings.TrimSpace(in.Username)
		if in.Username == "" {
			return nil, Invalid("username is required")
		}
		if err := security.PasswordPolicy(in.Password); err != nil {
			return nil, Invalid("%v", err)
		}
		hash, err := security.HashPassword(in.Password)
		if err != nil {
			return nil, err
		}
		u := &store.User{Username: in.Username, FullName: in.FullName, Email: in.Email, PasswordHash: hash, Role: in.Role,
			AllCompanies: in.AllCompanies, CompanyIDs: in.CompanyIDs, Active: true, MustChangePassword: true}
		nid, err := s.Store.CreateUser(ctx, u)
		if errors.Is(err, store.ErrConflict) {
			return nil, Invalid("username %q is taken", in.Username)
		}
		if err != nil {
			return nil, err
		}
		s.Audit(ctx, a, 0, "user.create", "user", fmt.Sprint(nid), map[string]any{"username": u.Username, "role": u.Role})
		return s.Store.GetUser(ctx, nid)
	}
	u, err := s.Store.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Password != "" {
		if err := security.PasswordPolicy(in.Password); err != nil {
			return nil, Invalid("%v", err)
		}
	}
	if a.UserID != nil && *a.UserID == id && in.Role != store.RoleAdmin && u.Role == store.RoleAdmin {
		return nil, Invalid("you cannot remove your own administrator role")
	}
	u.FullName, u.Email, u.Role, u.AllCompanies, u.CompanyIDs = in.FullName, in.Email, in.Role, in.AllCompanies, in.CompanyIDs
	if in.Active != nil {
		if a.UserID != nil && *a.UserID == id && !*in.Active {
			return nil, Invalid("you cannot deactivate yourself")
		}
		u.Active = *in.Active
	}
	if err := s.Store.UpdateUser(ctx, u); err != nil {
		return nil, err
	}
	if !u.Active {
		_ = s.Store.DeleteUserSessions(ctx, id)
	}
	if in.Password != "" {
		hash, err := security.HashPassword(in.Password)
		if err != nil {
			return nil, err
		}
		if err := s.Store.SetPassword(ctx, id, hash, true); err != nil {
			return nil, err
		}
		_ = s.Store.DeleteUserSessions(ctx, id)
	}
	s.Audit(ctx, a, 0, "user.update", "user", fmt.Sprint(id), map[string]any{"role": u.Role, "active": u.Active, "passwordReset": in.Password != ""})
	return s.Store.GetUser(ctx, id)
}

// ResetAdminPassword is used by the command-line recovery tool.
func (s *Service) ResetAdminPassword(ctx context.Context, username, newPw string) error {
	u, err := s.Store.GetUserByUsername(ctx, username)
	if err != nil {
		return err
	}
	if err := security.PasswordPolicy(newPw); err != nil {
		return err
	}
	hash, err := security.HashPassword(newPw)
	if err != nil {
		return err
	}
	if err := s.Store.SetPassword(ctx, u.ID, hash, true); err != nil {
		return err
	}
	_ = s.Store.DeleteUserSessions(ctx, u.ID)
	s.Audit(ctx, Actor{Username: "cli", Role: store.RoleAdmin}, 0, "user.password_reset_cli", "user", fmt.Sprint(u.ID), nil)
	return nil
}

// CreateAPIKey issues a key for ERP/POS integration. The plain key is
// returned once and only its hash is stored.
func (s *Service) CreateAPIKey(ctx context.Context, a Actor, companyID int64, name string) (string, *store.APIKey, error) {
	if strings.TrimSpace(name) == "" {
		return "", nil, Invalid("a name for the key is required (e.g. 'SAP B1 connector')")
	}
	plain := "eik_" + security.RandomToken(30)
	k := &store.APIKey{CompanyID: companyID, Name: strings.TrimSpace(name), Prefix: plain[:12], KeyHash: security.SHA256Hex(plain), CreatedBy: a.UserID}
	id, err := s.Store.CreateAPIKey(ctx, k)
	if err != nil {
		return "", nil, err
	}
	k.ID = id
	s.Audit(ctx, a, companyID, "apikey.create", "api_key", fmt.Sprint(id), map[string]any{"name": k.Name, "prefix": k.Prefix})
	return plain, k, nil
}

// APIKeyAuth resolves an API key.
func (s *Service) APIKeyAuth(ctx context.Context, plain string) (*store.APIKey, error) {
	if !strings.HasPrefix(plain, "eik_") {
		return nil, store.ErrNotFound
	}
	return s.Store.FindAPIKey(ctx, security.SHA256Hex(plain))
}

// RevokeAPIKey revokes a key.
func (s *Service) RevokeAPIKey(ctx context.Context, a Actor, companyID, id int64) error {
	if err := s.Store.RevokeAPIKey(ctx, companyID, id); err != nil {
		return err
	}
	s.Audit(ctx, a, companyID, "apikey.revoke", "api_key", fmt.Sprint(id), nil)
	return nil
}
