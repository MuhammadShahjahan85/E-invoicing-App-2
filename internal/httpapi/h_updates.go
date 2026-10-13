// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"net/http"
	"os"
	"path/filepath"

	"einvoicing/internal/brand"
	"einvoicing/internal/service"
)

// --- Software updates ---

func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	st, err := s.Svc.UpdateStatus(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, st)
}

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	st, err := s.Svc.CheckForUpdate(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, st)
}

func (s *Server) handleUpdateStart(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in struct {
		Install bool `json:"install"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	st, err := s.Svc.StartUpdate(r.Context(), rc.Actor, in.Install)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, st)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in service.UpdateSettings
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	st, err := s.Svc.SaveUpdateSettings(r.Context(), rc.Actor, in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, st)
}

// handleUpdateFile hands the downloaded, verified installer to an
// administrator who installs it by hand.
func (s *Server) handleUpdateFile(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	p, err := s.Svc.VerifiedUpdateFile()
	if err != nil {
		s.fail(w, err)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		s.fail(w, err)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		s.fail(w, err)
		return
	}
	s.Svc.Audit(r.Context(), rc.Actor, 0, "system.update.download", "system", filepath.Base(p), nil)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(p)+`"`)
	http.ServeContent(w, r, filepath.Base(p), st.ModTime(), f)
}

// handleWhatsNew tells every user what the update installed last brought.
func (s *Server) handleWhatsNew(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	writeJSON(w, 200, map[string]any{"version": brand.Version, "update": s.Svc.WhatsNew(r.Context())})
}
