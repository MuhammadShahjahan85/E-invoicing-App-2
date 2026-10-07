// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"net/http"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/domain"
	"einvoicing/internal/service"
	"einvoicing/internal/store"
	"einvoicing/internal/validate"
)

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	need, err := s.Svc.NeedsSetup(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	lic := s.License.Status(r.Context())
	writeJSON(w, 200, map[string]any{
		"product": brand.ProductName, "vendor": brand.Vendor, "version": brand.Version, "needsSetup": need,
		"copyright": brand.Copyright, "developedBy": brand.DevelopedBy,
		"license": map[string]any{"mode": lic.Mode, "message": lic.Message}, "serverTime": time.Now().UTC().Format(time.RFC3339),
	})
}

func (s *Server) handleSetup(w http.ResponseWriter, r *http.Request) {
	var in service.SetupInput
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Svc.Setup(r.Context(), clientIP(r), in); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"ok": true})
}

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, Secure: s.Secure,
		SameSite: http.SameSiteStrictMode, MaxAge: maxAge})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if !s.limiter.allow(clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "too many login attempts; wait a minute")
		return
	}
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	res, err := s.Svc.Login(r.Context(), in.Username, in.Password, clientIP(r), r.UserAgent())
	if err != nil {
		s.fail(w, err)
		return
	}
	s.setSessionCookie(w, res.Token, int(service.SessionTTL.Seconds()))
	writeJSON(w, 200, map[string]any{"user": res.User, "csrf": res.CSRF, "permissions": PermissionsFor(res.User.Role)})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	if rc.Token != "" {
		_ = s.Svc.Logout(r.Context(), rc.Actor, rc.Token)
	}
	s.setSessionCookie(w, "", -1)
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	if rc.User == nil {
		writeJSON(w, 200, map[string]any{"apiKey": rc.APIKey})
		return
	}
	writeJSON(w, 200, map[string]any{"user": rc.User, "csrf": rc.CSRF, "permissions": PermissionsFor(rc.User.Role)})
}

func (s *Server) handlePassword(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	if rc.User == nil {
		writeErr(w, 400, "not a user session")
		return
	}
	var in struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Svc.ChangePassword(r.Context(), rc.Actor, rc.User.ID, in.Current, in.New, rc.Token); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// handleMeta returns static catalogues used by the UI.
func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	envs := []map[string]string{}
	for _, e := range []domain.Environment{domain.EnvSimulator, domain.EnvSandbox, domain.EnvProduction} {
		envs = append(envs, map[string]string{"value": string(e), "label": e.Label()})
	}
	writeJSON(w, 200, map[string]any{
		"saleTypes":          domain.SaleTypes,
		"businessActivities": domain.BusinessActivities,
		"sectors":            domain.Sectors,
		"scenarios":          domain.Scenarios,
		"environments":       envs,
		"docTypes":           []domain.DocType{domain.DocSaleInvoice, domain.DocDebitNote},
		"registrationTypes":  []domain.RegistrationType{domain.Registered, domain.Unregistered},
		"provinces":          domain.Provinces,
		"uoms":               domain.UOMs,
		"errorCatalogue":     validate.Catalogue(),
		"roles":              []string{store.RoleAdmin, store.RoleManager, store.RoleAccountant, store.RoleOperator, store.RoleAuditor},
		"product":            brand.ProductName,
		"vendor":             brand.Vendor,
		"support":            brand.SupportContact,
		"copyright":          brand.Copyright,
		"developedBy":        brand.DevelopedBy,
		"version":            brand.Version,
		"cancelWindowHours":  int(service.CancelWindow.Hours()),
		// The FBR cancellation API is opt-in (fbr.endpoints.cancelPath /
		// cancelSandboxPath in config.json) until PRAL publishes its format.
		"cancelApi": map[string]bool{
			string(domain.EnvProduction): s.Svc.Opts.Endpoints.CancelPath != "",
			string(domain.EnvSandbox):    s.Svc.Opts.Endpoints.CancelSandboxPath != "",
		},
	})
}
