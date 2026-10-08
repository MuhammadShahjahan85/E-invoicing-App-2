// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"net/http"
	"strings"

	"einvoicing/internal/service"
	"einvoicing/internal/store"
)

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	alerts := s.Svc.Alerts(r.Context(), c, envParam(r, c), allowed(rc, PermSystem))
	if alerts == nil {
		alerts = []service.Alert{}
	}
	writeJSON(w, 200, map[string]any{"alerts": alerts})
}

// handleSearch searches invoices, customers, products and FBR's HS codes.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	ctx := r.Context()
	c, err := s.Svc.Store.GetCompany(ctx, cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	out := map[string]any{"q": q, "invoices": []*store.Invoice{}, "customers": []*store.Customer{}, "products": []*store.Product{}, "hsCodes": []store.HSCode{}}
	if len([]rune(q)) < 2 {
		writeJSON(w, 200, out)
		return
	}
	limit := qInt(r, "limit", 6)
	if inv, _, err := s.Svc.Store.ListInvoices(ctx, c.ID, store.InvoiceFilter{Environment: envParam(r, c), Q: q, Limit: limit}); err == nil && inv != nil {
		out["invoices"] = inv
	}
	if cu, _, err := s.Svc.Store.ListCustomers(ctx, c.ID, store.ListParams{Q: q, Limit: limit}); err == nil && cu != nil {
		out["customers"] = cu
	}
	if pr, _, err := s.Svc.Store.ListProducts(ctx, c.ID, store.ListParams{Q: q, Limit: limit}); err == nil && pr != nil {
		out["products"] = pr
	}
	if hs, err := s.Svc.Store.SearchHSCodes(ctx, q, limit); err == nil && hs != nil {
		out["hsCodes"] = hs
	}
	writeJSON(w, 200, out)
}

// handleCompliance returns the review of a tax period (?period=YYYY-MM).
func (s *Server) handleCompliance(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	rev, err := s.Svc.ReviewPeriod(r.Context(), c, envParam(r, c), r.URL.Query().Get("period"))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, rev)
}

func (s *Server) handleDuplicateInvoice(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	inv, err := s.Svc.DuplicateInvoice(r.Context(), rc.Actor, cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, inv)
}
