// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

// Package httpapi exposes the REST API used by the web UI and by external
// ERP/POS systems, and serves the embedded single-page application.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"einvoicing/internal/license"
	"einvoicing/internal/service"
	"einvoicing/internal/store"
)

// Server holds dependencies.
type Server struct {
	Svc     *service.Service
	License *license.Manager
	UI      fs.FS // built web UI (may be nil)
	Log     *slog.Logger
	Secure  bool // cookies marked Secure (HTTPS)
	Started time.Time
	limiter *rateLimiter
}

// New creates the HTTP server.
func New(svc *service.Service, lic *license.Manager, ui fs.FS, log *slog.Logger, secure bool) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{Svc: svc, License: lic, UI: ui, Log: log, Secure: secure, Started: time.Now(), limiter: newRateLimiter(10, time.Minute)}
}

const cookieName = "einv_session"

type ctxKey int

const reqCtxKey ctxKey = 1

// reqCtx carries the authenticated principal.
type reqCtx struct {
	User   *store.User
	APIKey *store.APIKey
	CSRF   string
	Token  string
	Actor  service.Actor
}

func getReq(r *http.Request) *reqCtx {
	rc, _ := r.Context().Value(reqCtxKey).(*reqCtx)
	return rc
}

// Handler builds the root handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	if s.UI != nil {
		mux.Handle("/", s.spa())
	}
	return s.recoverer(s.securityHeaders(s.logRequests(mux)))
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// --- middleware ---

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			s.Log.Info("http", "method", r.Method, "path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds(), "ip", clientIP(r))
		}
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.Log.Error("panic", "err", v, "stack", string(debug.Stack()))
				writeErr(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		if h.Get("Content-Security-Policy") == "" {
			h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; frame-ancestors 'self'; base-uri 'self'; form-action 'self'")
		}
		if s.Secure {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}

// authenticate resolves the session cookie or API key. Missing auth is not
// an error here; perm() enforces it.
func (s *Server) authenticate(r *http.Request) (*reqCtx, error) {
	ctx := r.Context()
	if key := apiKeyFrom(r); key != "" {
		k, err := s.Svc.APIKeyAuth(ctx, key)
		if err != nil {
			return nil, errUnauthorized
		}
		id := k.ID
		return &reqCtx{APIKey: k, Actor: service.Actor{Username: "api:" + k.Name, Role: "api", IP: clientIP(r), APIKeyID: &id}}, nil
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return nil, nil
	}
	u, se, err := s.Svc.SessionUser(ctx, c.Value)
	if err != nil {
		return nil, nil
	}
	uid := u.ID
	return &reqCtx{User: u, CSRF: se.CSRFToken, Token: c.Value, Actor: service.Actor{UserID: &uid, Username: u.Username, Role: u.Role, IP: clientIP(r)}}, nil
}

func apiKeyFrom(r *http.Request) string {
	if k := strings.TrimSpace(r.Header.Get("X-API-Key")); k != "" {
		return k
	}
	if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer eik_") {
		return strings.TrimSpace(a[7:])
	}
	return ""
}

var errUnauthorized = errors.New("unauthorized")

// handler signature for authenticated endpoints.
type authedFunc func(w http.ResponseWriter, r *http.Request, rc *reqCtx)

// perm wraps a handler with authentication, permission and CSRF checks.
func (s *Server) perm(p Perm, h authedFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rc, err := s.authenticate(r)
		if err != nil || rc == nil {
			writeErr(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if rc.User != nil && r.Method != http.MethodGet && r.Method != http.MethodHead {
			if tok := r.Header.Get("X-CSRF-Token"); tok == "" || tok != rc.CSRF {
				writeErr(w, http.StatusForbidden, "missing or invalid CSRF token")
				return
			}
		}
		if !allowed(rc, p) {
			writeErr(w, http.StatusForbidden, "your role does not permit this action")
			return
		}
		if rc.User != nil && rc.User.MustChangePassword && p != PermSelf {
			writeErr(w, http.StatusPreconditionRequired, "password change required")
			return
		}
		// Company scoping.
		if cidStr := r.PathValue("cid"); cidStr != "" {
			cid, err := strconv.ParseInt(cidStr, 10, 64)
			if err != nil {
				writeErr(w, http.StatusBadRequest, "invalid company id")
				return
			}
			if rc.APIKey != nil && rc.APIKey.CompanyID != cid {
				writeErr(w, http.StatusForbidden, "API key is not valid for this company")
				return
			}
			if rc.User != nil && !rc.User.CanAccessCompany(cid) {
				writeErr(w, http.StatusForbidden, "no access to this company")
				return
			}
		}
		ctx := context.WithValue(r.Context(), reqCtxKey, rc)
		h(w, r.WithContext(ctx), rc)
	}
}

// --- responses ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": msg})
}

// fail maps service errors to HTTP responses.
func (s *Server) fail(w http.ResponseWriter, err error) {
	if se, ok := service.IsSubmissionError(err); ok {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "The invoice has validation errors and was not sent to FBR.", "issues": se.Issues})
		return
	}
	switch {
	case service.IsValidation(err):
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, "already exists")
	case errors.Is(err, service.ErrAuth), errors.Is(err, service.ErrLocked):
		writeErr(w, http.StatusUnauthorized, err.Error())
	default:
		s.Log.Error("request failed", "err", err)
		writeErr(w, http.StatusInternalServerError, "internal error: "+err.Error())
	}
}

func decode(r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 8<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		return service.Invalid("invalid JSON body: %v", err)
	}
	return nil
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, service.Invalid("invalid %s", name)
	}
	return id, nil
}

func cid(r *http.Request) int64 {
	id, _ := strconv.ParseInt(r.PathValue("cid"), 10, 64)
	return id
}

func qInt(r *http.Request, name string, def int) int {
	if v, err := strconv.Atoi(r.URL.Query().Get(name)); err == nil {
		return v
	}
	return def
}

// --- rate limiting (login) ---

type rateLimiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string][]time.Time
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{max: max, window: window, hits: map[string][]time.Time{}}
}

func (l *rateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	h := l.hits[key]
	n := 0
	for _, t := range h {
		if now.Sub(t) < l.window {
			h[n] = t
			n++
		}
	}
	h = h[:n]
	if len(h) >= l.max {
		l.hits[key] = h
		return false
	}
	l.hits[key] = append(h, now)
	return true
}

// --- SPA ---

func init() {
	// Go's built-in table lacks the web app manifest type on some systems.
	_ = mime.AddExtensionType(".webmanifest", "application/manifest+json")
}

func (s *Server) spa() http.Handler {
	files := http.FileServer(http.FS(s.UI))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeErr(w, http.StatusNotFound, "no such endpoint")
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(s.UI, p); err != nil {
			// Client-side route: serve index.html.
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			w.Header().Set("Cache-Control", "no-cache")
			files.ServeHTTP(w, r2)
			return
		}
		if strings.HasPrefix(p, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

func attachment(w http.ResponseWriter, name, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}
