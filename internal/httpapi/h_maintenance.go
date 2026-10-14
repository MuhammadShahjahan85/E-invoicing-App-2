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

// --- Restore, backup settings, server log, user guide ---

func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in struct {
		Confirm string `json:"confirm"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	if in.Confirm != "RESTORE" {
		s.fail(w, service.Invalid(`type RESTORE to confirm`))
		return
	}
	dir := s.Svc.BackupDir()
	safety, err := s.Svc.RestoreBackup(r.Context(), rc.Actor, r.PathValue("name"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"restored": r.PathValue("name"), "safetyBackup": safety.Name, "dir": dir})
}

func (s *Server) handleUploadBackup(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	b, err := s.Svc.SaveUploadedBackup(r.Context(), rc.Actor, http.MaxBytesReader(w, r.Body, 4<<30))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, b)
}

func (s *Server) handleGetBackupSettings(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	writeJSON(w, 200, s.Svc.GetBackupSettings(r.Context()))
}

func (s *Server) handleSaveBackupSettings(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in service.BackupSettings
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.Svc.SaveBackupSettings(r.Context(), rc.Actor, in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleLogTail(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	text, err := s.Svc.LogTail(qInt(r, "lines", 300))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"text": text, "path": s.Svc.LogPath()})
}

func (s *Server) handleLogDownload(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	f, err := os.Open(s.Svc.LogPath())
	if err != nil {
		s.fail(w, service.Invalid("there is no log file yet"))
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="einvoice.log"`)
	http.ServeContent(w, r, "einvoice.log", st.ModTime(), f)
}

// handleUserGuide serves the printable user guide installed with the program.
func (s *Server) handleUserGuide(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	p := service.UserGuidePath()
	if p == "" {
		writeErr(w, http.StatusNotFound, "the user guide is not installed on this server")
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
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="`+filepath.Base(brand.UserGuideFile)+`"`)
	http.ServeContent(w, r, brand.UserGuideFile, st.ModTime(), f)
}
