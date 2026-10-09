// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"bytes"
	"net/http"

	"einvoicing/internal/printing"
	"einvoicing/internal/security"
	"einvoicing/internal/service"
	"einvoicing/internal/store"
)

// --- Closings (rule 150R(4)(f)) ---

func (s *Server) handleClosings(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	list, err := s.Svc.Closings(r.Context(), c, envParam(r, c), r.URL.Query().Get("kind"), qInt(r, "limit", 60))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, list)
}

// --- Return filing extensions ---

func (s *Server) handleListExtensions(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, err := s.Svc.Store.ListReturnExtensions(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleSetExtension(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in store.ReturnExtension
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	in.Period = r.PathValue("period")
	out, err := s.Svc.SetReturnExtension(r.Context(), rc.Actor, cid(r), in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleDeleteExtension(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	if err := s.Svc.DeleteReturnExtension(r.Context(), rc.Actor, cid(r), r.PathValue("period")); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// --- Stock Transfer Notes (Sales Tax General Order 25 of 2026) ---

func (s *Server) handleListTransfers(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	q := r.URL.Query()
	list, total, err := s.Svc.Store.ListStockTransfers(r.Context(), cid(r), store.TransferFilter{From: q.Get("from"), To: q.Get("to"),
		Status: q.Get("status"), Q: q.Get("q"), Limit: qInt(r, "limit", 100), Offset: qInt(r, "offset", 0)})
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"transfers": list, "total": total})
}

func (s *Server) handleCreateTransfer(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in service.TransferInput
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	t, err := s.Svc.CreateStockTransfer(r.Context(), rc.Actor, cid(r), in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, t)
}

func (s *Server) handleGetTransfer(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	t, err := s.Svc.Store.GetStockTransfer(r.Context(), cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, t)
}

func (s *Server) handleReceiveTransfer(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	var in struct {
		ReceivedBy string `json:"receivedBy"`
		ReceivedAt string `json:"receivedAt"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	t, err := s.Svc.ReceiveStockTransfer(r.Context(), rc.Actor, cid(r), id, in.ReceivedBy, in.ReceivedAt)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, t)
}

func (s *Server) handleCancelTransfer(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	t, err := s.Svc.CancelStockTransfer(r.Context(), rc.Actor, cid(r), id, in.Reason)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, t)
}

// writePrintPage sends a server-rendered page that may run only its own
// nonce'd script and show only inline images.
func writePrintPage(w http.ResponseWriter, nonce string, body []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src data:; style-src 'unsafe-inline'; script-src 'nonce-"+nonce+"'")
	_, _ = w.Write(body)
}

func (s *Server) handlePrintTransfer(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	ctx := r.Context()
	t, err := s.Svc.Store.GetStockTransfer(ctx, cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	c, err := s.Svc.Store.GetCompany(ctx, cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	nonce := security.RandomToken(18)
	var buf bytes.Buffer
	if err := printing.RenderStockTransfer(&buf, c, t, printing.Options{Nonce: nonce, ShowToolbar: r.URL.Query().Get("toolbar") != "0"}); err != nil {
		s.fail(w, err)
		return
	}
	writePrintPage(w, nonce, buf.Bytes())
}

// --- "Integrated with FBR" signboard (rule 150R(11)) ---

func (s *Server) handleSignboard(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	logo, mime := s.fbrLogo(r)
	nonce := security.RandomToken(18)
	var buf bytes.Buffer
	if err := printing.RenderSignboard(&buf, c, r.URL.Query().Get("outlet"), printing.Options{Nonce: nonce, ShowToolbar: true,
		FBRLogo: logo, FBRLogoMime: mime}); err != nil {
		s.fail(w, err)
		return
	}
	writePrintPage(w, nonce, buf.Bytes())
}

// --- Invoice signing key (rule 150R(4)(b)) ---

func (s *Server) handleSigningKey(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	k, err := s.Svc.SigningKey(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	if r.URL.Query().Get("format") == "pem" {
		attachment(w, "invoice-signing-key-"+k.Fingerprint+".pem", "application/x-pem-file", []byte(k.PublicKey))
		return
	}
	writeJSON(w, 200, k)
}

// --- Public IP for PRAL's IP whitelisting ---

func (s *Server) handlePublicIP(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	ip, source, err := s.Svc.PublicIP(r.Context())
	if err != nil {
		writeErr(w, 502, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"ip": ip, "source": source})
}
