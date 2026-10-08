// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package service implements the business processes: invoice lifecycle and
// FBR submission, reference data, masters, scenario testing, import,
// reporting, incidents, backups and authentication.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/fbr"
	"einvoicing/internal/fbrmock"
	"einvoicing/internal/security"
	"einvoicing/internal/store"
)

// PKT is Pakistan Standard Time, used for business dates (invoice dates,
// tax periods) regardless of the server's own time zone.
var PKT = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Karachi"); err == nil {
		return loc
	}
	return time.FixedZone("PKT", 5*3600)
}()

// LicenseChecker decides whether production submissions are permitted.
type LicenseChecker interface {
	CheckProduction(sellerNTN string) error
}

type allowAll struct{}

func (allowAll) CheckProduction(string) error { return nil }

// Options configures the service.
type Options struct {
	Endpoints     fbr.Endpoints
	HTTPTimeout   time.Duration
	DataDir       string
	BackupDir     string
	CNICThreshold string // warn for unregistered buyers without CNIC above this value
	License       LicenseChecker
	Logger        *slog.Logger
}

// Service is the application core.
type Service struct {
	Store     *store.Store
	Vault     *security.Vault
	Opts      Options
	Simulator *fbrmock.Server
	Log       *slog.Logger
	Now       func() time.Time
	// HTTPClient used for FBR calls (overridable in tests).
	HTTPClient *http.Client

	submitMu sync.Map // per-invoice locks
	health   *healthTracker

	notifyMu      sync.Mutex // one notification run at a time
	lastNotifyRun time.Time
}

// New builds the service.
func New(st *store.Store, vault *security.Vault, opts Options) *Service {
	if opts.License == nil {
		opts.License = allowAll{}
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	sim := fbrmock.New()
	sim.AcceptAnyToken = true
	s := &Service{
		Store: st, Vault: vault, Opts: opts, Simulator: sim, Log: opts.Logger, Now: time.Now,
		HTTPClient: fbr.NewHTTPClient(opts.HTTPTimeout),
		health:     newHealthTracker(),
	}
	return s
}

// Actor identifies who performs an action (for permissions and audit).
type Actor struct {
	UserID   *int64
	Username string
	Role     string
	IP       string
	APIKeyID *int64
}

// System is the actor for background jobs.
var System = Actor{Username: "system", Role: store.RoleAdmin}

// ErrValidation wraps user-facing validation problems.
type ErrValidation struct{ Msg string }

func (e *ErrValidation) Error() string { return e.Msg }

// Invalid builds an ErrValidation.
func Invalid(format string, args ...any) error {
	return &ErrValidation{Msg: fmt.Sprintf(format, args...)}
}

// IsValidation reports whether err is a user-facing validation error.
func IsValidation(err error) bool {
	var v *ErrValidation
	return errors.As(err, &v)
}

// Audit writes an audit entry; failures are logged, not returned.
func (s *Service) Audit(ctx context.Context, a Actor, companyID int64, action, entity, entityID string, details any) {
	var cid *int64
	if companyID > 0 {
		cid = &companyID
	}
	d := ""
	switch v := details.(type) {
	case nil:
	case string:
		d = v
	default:
		b, _ := json.Marshal(v)
		d = string(b)
	}
	e := &store.AuditEntry{UserID: a.UserID, Username: a.Username, CompanyID: cid, Action: action, Entity: entity, EntityID: entityID, Details: d, IP: a.IP}
	if err := s.Store.AppendAudit(ctx, e); err != nil {
		s.Log.Error("audit write failed", "err", err, "action", action)
	}
}

// Today returns today's business date (PKT) as YYYY-MM-DD.
func (s *Service) Today() string { return s.Now().In(PKT).Format("2006-01-02") }

// FiscalYear returns Pakistan's fiscal year code for a date (July–June),
// e.g. 2026-10-06 -> "2627", 2027-03-01 -> "2627".
func FiscalYear(date string) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		t = time.Now().In(PKT)
	}
	y := t.Year()
	if t.Month() < time.July {
		y--
	}
	return fmt.Sprintf("%02d%02d", y%100, (y+1)%100)
}

// token returns the decrypted FBR token for an environment.
func (s *Service) token(c *store.Company, env domain.Environment) (string, error) {
	switch env {
	case domain.EnvSimulator:
		return "SIMULATOR", nil
	case domain.EnvSandbox:
		return s.Vault.Decrypt(c.SandboxTokenEnc)
	case domain.EnvProduction:
		return s.Vault.Decrypt(c.ProductionTokenEnc)
	}
	return "", fmt.Errorf("unknown environment %q", env)
}

// Client builds an FBR client for a company and environment. Every exchange
// is written to the FBR call log (token excluded).
func (s *Service) Client(ctx context.Context, c *store.Company, env domain.Environment, invoiceID *int64) (*fbr.Client, error) {
	tok, err := s.token(c, env)
	if err != nil {
		return nil, err
	}
	hc := s.HTTPClient
	ep := s.Opts.Endpoints
	clientEnv := env
	if env == domain.EnvSimulator {
		hc = &http.Client{Transport: s.Simulator.Transport(), Timeout: 30 * time.Second}
		ep = fbr.Endpoints{BaseURL: "http://simulator.local"}
		clientEnv = domain.EnvProduction // behaves like production (no scenarioId)
	}
	cl := fbr.New(clientEnv, tok, ep, hc)
	companyID := c.ID
	cl.OnCall = func(l fbr.CallLog) {
		rec := &store.FBRCall{CompanyID: companyID, InvoiceID: invoiceID, Environment: string(env), Operation: l.Operation,
			Method: l.Method, URL: l.URL, RequestBody: string(l.RequestBody), ResponseBody: truncate(string(l.ResponseBody), 200000),
			HTTPStatus: l.HTTPStatus, DurationMS: l.Duration.Milliseconds(), ErrorKind: string(l.ErrKind)}
		if l.Err != nil {
			rec.Error = l.Err.Error()
		}
		// Use a detached context so the log survives request cancellation.
		lctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Store.LogFBRCall(lctx, rec); err != nil {
			s.Log.Error("log fbr call", "err", err)
		}
		s.health.observe(companyID, env, l)
	}
	return cl, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…[truncated]"
}

// lockInvoice serialises work on one invoice within this process.
func (s *Service) lockInvoice(id int64) func() {
	v, _ := s.submitMu.LoadOrStore(id, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}

// companyFor loads a company and checks it is usable.
func (s *Service) companyFor(ctx context.Context, id int64) (*store.Company, error) {
	c, err := s.Store.GetCompany(ctx, id)
	if err != nil {
		return nil, err
	}
	if !c.Active {
		return nil, Invalid("company %s is inactive", c.Name)
	}
	return c, nil
}

func cleanText(s string) string { return strings.Join(strings.Fields(s), " ") }
