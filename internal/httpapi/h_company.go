package httpapi

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"einvoicing/internal/domain"
	"einvoicing/internal/service"
	"einvoicing/internal/store"
)

func envParam(r *http.Request, c *store.Company) domain.Environment {
	if e := domain.Environment(r.URL.Query().Get("env")); e.Valid() {
		return e
	}
	if c != nil {
		return c.Environment
	}
	return domain.EnvSimulator
}

func (s *Server) handleListCompanies(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, err := s.Svc.Store.ListCompanies(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	out := []*store.Company{}
	for _, c := range list {
		if rc.APIKey != nil && rc.APIKey.CompanyID != c.ID {
			continue
		}
		if rc.User != nil && !rc.User.CanAccessCompany(c.ID) {
			continue
		}
		out = append(out, c)
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleCreateCompany(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var c store.Company
	if err := decode(r, &c); err != nil {
		s.fail(w, err)
		return
	}
	n, _ := s.Svc.Store.CountCompanies(r.Context())
	if err := s.License.CheckLimits(n+1, 0); err != nil {
		writeErr(w, http.StatusPaymentRequired, err.Error())
		return
	}
	c.ID = 0
	out, err := s.Svc.SaveCompany(r.Context(), rc.Actor, &c)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, out)
}

func (s *Server) handleGetCompany(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, c)
}

func (s *Server) handleUpdateCompany(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	cur, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	in := *cur
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	in.ID = cur.ID
	in.SandboxTokenEnc, in.ProductionTokenEnc = cur.SandboxTokenEnc, cur.ProductionTokenEnc
	out, err := s.Svc.SaveCompany(r.Context(), rc.Actor, &in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleSetToken(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in service.TokenInput
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	out, err := s.Svc.SetToken(r.Context(), rc.Actor, cid(r), in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleTestConnection(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	res, err := s.Svc.TestConnection(r.Context(), rc.Actor, c.ID, envParam(r, c))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleSyncReference(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	rep, err := s.Svc.SyncReference(r.Context(), rc.Actor, c.ID, envParam(r, c))
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, rep)
}

func (s *Server) handleGetLogo(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	b, mime, err := s.Svc.Store.GetCompanyLogo(r.Context(), cid(r))
	if err != nil || len(b) == 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

func readImage(r *http.Request) ([]byte, string, error) {
	mime := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0]))
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp", "image/svg+xml":
	default:
		return nil, "", service.Invalid("upload a PNG, JPEG, GIF, WebP or SVG image")
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 2<<20+1))
	if err != nil {
		return nil, "", err
	}
	if len(b) > 2<<20 {
		return nil, "", service.Invalid("image is larger than 2 MB")
	}
	if mime == "image/svg+xml" && strings.Contains(strings.ToLower(string(b)), "<script") {
		return nil, "", service.Invalid("SVG images must not contain scripts")
	}
	return b, mime, nil
}

func (s *Server) handlePutLogo(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	b, mime, err := readImage(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Svc.Store.SetCompanyLogo(r.Context(), cid(r), b, mime); err != nil {
		s.fail(w, err)
		return
	}
	s.Svc.Audit(r.Context(), rc.Actor, cid(r), "company.logo", "company", strconv.FormatInt(cid(r), 10), map[string]any{"bytes": len(b)})
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleDeleteLogo(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	if err := s.Svc.Store.SetCompanyLogo(r.Context(), cid(r), nil, ""); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	env := envParam(r, c)
	now := time.Now().In(service.PKT)
	d, err := s.Svc.Store.GetDashboard(r.Context(), c.ID, env, now.Format("2006-01-02"), now.Format("2006-01")+"-01")
	if err != nil {
		s.fail(w, err)
		return
	}
	scen, _ := s.Svc.Scenarios(r.Context(), c.ID)
	incidents, _ := s.Svc.Store.ListIncidents(r.Context(), c.ID, 5)
	openIncidents := 0
	unreported := 0
	for _, i := range incidents {
		if i.EndedAt == "" {
			openIncidents++
		}
		if i.ReportedAt == "" {
			unreported++
		}
	}
	tokenWarn := ""
	if env == domain.EnvProduction && c.ProductionTokenExpiry != "" {
		if exp, err := time.Parse("2006-01-02", c.ProductionTokenExpiry); err == nil && time.Until(exp) < 60*24*time.Hour {
			tokenWarn = "The production security token expires on " + c.ProductionTokenExpiry + ". Renew it on IRIS."
		}
	}
	writeJSON(w, 200, map[string]any{
		"environment": env, "stats": d, "connection": s.Svc.Connection(c.ID, env), "scenarios": scen,
		"openIncidents": openIncidents, "unreportedIncidents": unreported, "tokenWarning": tokenWarn,
		"license": s.License.Status(r.Context()),
	})
}

// --- customers ---

func listParams(r *http.Request) store.ListParams {
	q := r.URL.Query()
	return store.ListParams{Q: q.Get("q"), Limit: qInt(r, "limit", 100), Offset: qInt(r, "offset", 0), OnlyActive: q.Get("active") == "1"}
}

func (s *Server) handleListCustomers(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, total, err := s.Svc.Store.ListCustomers(r.Context(), cid(r), listParams(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.Customer{}
	}
	writeJSON(w, 200, map[string]any{"items": list, "total": total})
}

func (s *Server) handleGetCustomer(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	c, err := s.Svc.Store.GetCustomer(r.Context(), cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, c)
}

func (s *Server) handleSaveCustomer(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var c store.Customer
	if r.Method == http.MethodPut {
		id, err := pathID(r, "id")
		if err != nil {
			s.fail(w, err)
			return
		}
		cur, err := s.Svc.Store.GetCustomer(r.Context(), cid(r), id)
		if err != nil {
			s.fail(w, err)
			return
		}
		c = *cur
	}
	if err := decode(r, &c); err != nil {
		s.fail(w, err)
		return
	}
	c.CompanyID = cid(r)
	if r.Method == http.MethodPost {
		c.ID = 0
	} else {
		c.ID, _ = pathID(r, "id")
	}
	out, err := s.Svc.SaveCustomer(r.Context(), rc.Actor, &c)
	if err != nil {
		s.fail(w, err)
		return
	}
	status := 200
	if r.Method == http.MethodPost {
		status = 201
	}
	writeJSON(w, status, out)
}

func (s *Server) handleCheckCustomer(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	res, err := s.Svc.CheckBuyer(r.Context(), rc.Actor, c.ID, envParam(r, c), "", id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, res)
}

func (s *Server) handleBuyerCheck(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in struct {
		RegNo string `json:"regNo"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	res, err := s.Svc.CheckBuyer(r.Context(), rc.Actor, c.ID, envParam(r, c), strings.TrimSpace(in.RegNo), 0)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, res)
}

// --- products ---

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, total, err := s.Svc.Store.ListProducts(r.Context(), cid(r), listParams(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.Product{}
	}
	writeJSON(w, 200, map[string]any{"items": list, "total": total})
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	p, err := s.Svc.Store.GetProduct(r.Context(), cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, p)
}

func (s *Server) handleSaveProduct(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var p store.Product
	if r.Method == http.MethodPut {
		id, err := pathID(r, "id")
		if err != nil {
			s.fail(w, err)
			return
		}
		cur, err := s.Svc.Store.GetProduct(r.Context(), cid(r), id)
		if err != nil {
			s.fail(w, err)
			return
		}
		p = *cur
	}
	if err := decode(r, &p); err != nil {
		s.fail(w, err)
		return
	}
	p.CompanyID = cid(r)
	if r.Method == http.MethodPost {
		p.ID = 0
	} else {
		p.ID, _ = pathID(r, "id")
	}
	out, err := s.Svc.SaveProduct(r.Context(), rc.Actor, &p)
	if err != nil {
		s.fail(w, err)
		return
	}
	status := 200
	if r.Method == http.MethodPost {
		status = 201
	}
	writeJSON(w, status, out)
}

// --- reference data ---

func (s *Server) handleRef(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	ctx := r.Context()
	c, err := s.Svc.Store.GetCompany(ctx, cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	env := envParam(r, c)
	q := r.URL.Query()
	switch r.PathValue("kind") {
	case "provinces":
		writeJSON(w, 200, s.Svc.Provinces(ctx, env))
	case "sale-types":
		writeJSON(w, 200, s.Svc.SaleTypes(ctx, env))
	case "uoms":
		writeJSON(w, 200, s.Svc.UOMs(ctx, env))
	case "hs-codes":
		list, err := s.Svc.Store.SearchHSCodes(ctx, q.Get("q"), qInt(r, "limit", 50))
		if err != nil {
			s.fail(w, err)
			return
		}
		if list == nil {
			list = []store.HSCode{}
		}
		writeJSON(w, 200, list)
	case "rates":
		list, err := s.Svc.Rates(ctx, c.ID, env, q.Get("saleType"), q.Get("date"))
		if err != nil {
			s.fail(w, err)
			return
		}
		writeJSON(w, 200, list)
	case "sro-schedules":
		list, err := s.Svc.SROSchedules(ctx, c.ID, env, qInt(r, "rateId", 0), q.Get("date"))
		if err != nil {
			s.fail(w, err)
			return
		}
		writeJSON(w, 200, list)
	case "sro-items":
		list, err := s.Svc.SROItems(ctx, c.ID, env, qInt(r, "sroId", 0), q.Get("date"))
		if err != nil {
			s.fail(w, err)
			return
		}
		writeJSON(w, 200, list)
	case "hs-uom":
		list, err := s.Svc.HSUOM(ctx, c.ID, env, q.Get("hsCode"))
		if err != nil {
			s.fail(w, err)
			return
		}
		writeJSON(w, 200, list)
	case "status":
		st, err := s.Svc.Store.RefStatus(ctx)
		if err != nil {
			s.fail(w, err)
			return
		}
		n, _ := s.Svc.Store.CountHSCodes(ctx)
		writeJSON(w, 200, map[string]any{"entries": st, "hsCodes": n})
	default:
		writeErr(w, 404, "unknown reference list")
	}
}
