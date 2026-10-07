// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/config"
	"einvoicing/internal/printing"
	"einvoicing/internal/security"
	"einvoicing/internal/service"
	"einvoicing/internal/store"
)

func decodeB64(s string) ([]byte, error) { return base64.StdEncoding.DecodeString(s) }

// --- users ---

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, err := s.Svc.Store.ListUsers(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.User{}
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleSaveUser(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in service.UserInput
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	var id int64
	if r.Method == http.MethodPut {
		var err error
		if id, err = pathID(r, "id"); err != nil {
			s.fail(w, err)
			return
		}
	} else {
		n, _ := s.Svc.Store.CountUsers(r.Context())
		if err := s.License.CheckLimits(0, n+1); err != nil {
			writeErr(w, http.StatusPaymentRequired, err.Error())
			return
		}
	}
	u, err := s.Svc.SaveUser(r.Context(), rc.Actor, id, in)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, u)
}

// --- API keys ---

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, err := s.Svc.Store.ListAPIKeys(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.APIKey{}
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in struct {
		Name string `json:"name"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	plain, k, err := s.Svc.CreateAPIKey(r.Context(), rc.Actor, cid(r), in.Name)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, map[string]any{"key": plain, "apiKey": k, "note": "Copy this key now; it will not be shown again."})
}

func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Svc.RevokeAPIKey(r.Context(), rc.Actor, cid(r), id); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}

// --- audit ---

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	q := r.URL.Query()
	f := store.AuditFilter{Entity: q.Get("entity"), EntityID: q.Get("entityId"), Action: q.Get("action"), From: q.Get("from"), To: q.Get("to"),
		Limit: qInt(r, "limit", 100), Offset: qInt(r, "offset", 0)}
	c := int64(qInt(r, "companyId", 0))
	// Users limited to some companies may only read those companies' entries;
	// the installation-wide trail (all companies and system events) is for
	// administrators and users with access to all companies.
	if c <= 0 && rc.User != nil && !rc.User.AllCompanies && rc.User.Role != store.RoleAdmin {
		writeErr(w, 403, "choose a company: your account can only view the audit trail of the companies assigned to it")
		return
	}
	if c > 0 {
		if rc.User != nil && !rc.User.CanAccessCompany(c) {
			writeErr(w, 403, "no access to this company")
			return
		}
		f.CompanyID = c
	}
	list, total, err := s.Svc.Store.ListAudit(r.Context(), f)
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.AuditEntry{}
	}
	writeJSON(w, 200, map[string]any{"items": list, "total": total})
}

func (s *Server) handleAuditVerify(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	// The full integrity check (audit chain plus invoice seals) opens a
	// tampering incident when it finds a problem (Rule 150R).
	rep, err := s.Svc.CheckIntegrity(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	resp := map[string]any{"intact": rep.AuditBrokenAt == 0, "brokenAt": rep.AuditBrokenAt, "checked": rep.AuditChecked}
	// Invoice details span every company, so only installation-wide users see them.
	if rc.User != nil && (rc.User.AllCompanies || rc.User.Role == store.RoleAdmin) {
		resp["invoicesChecked"] = rep.InvoicesChecked
		resp["problems"] = rep.Problems
	}
	writeJSON(w, 200, resp)
}

// --- backups ---

func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, err := s.Svc.ListBackups()
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": list, "dir": s.Svc.BackupDir()})
}

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	b, err := s.Svc.Backup(r.Context(), rc.Actor, "manual")
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, 201, b)
}

func (s *Server) handleDownloadBackup(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	p, err := s.Svc.BackupPath(r.PathValue("name"))
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
	s.Svc.Audit(r.Context(), rc.Actor, 0, "system.backup_download", "backup", r.PathValue("name"), nil)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, r.PathValue("name")))
	_, _ = io.Copy(w, f)
}

// --- licence ---

func (s *Server) handleGetLicense(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	writeJSON(w, 200, s.License.Status(r.Context()))
}

func (s *Server) handleInstallLicense(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	b, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		s.fail(w, err)
		return
	}
	st, err := s.License.Install(r.Context(), b)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	s.Svc.Audit(r.Context(), rc.Actor, 0, "system.license_installed", "license", st.LicenseID, map[string]any{"licensee": st.Licensee, "expires": st.ExpiresAt})
	writeJSON(w, 200, st)
}

// --- FBR DI logo ---

func (s *Server) handleGetFBRLogo(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	b, mime := s.fbrLogo(r)
	if len(b) == 0 {
		http.NotFound(w, r)
		return
	}
	serveImage(w, b, mime)
}

func (s *Server) handlePutFBRLogo(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	b, mime, err := readImage(r)
	if err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Svc.Store.SetSetting(r.Context(), "fbr_logo", base64.StdEncoding.EncodeToString(b)); err != nil {
		s.fail(w, err)
		return
	}
	if err := s.Svc.Store.SetSetting(r.Context(), "fbr_logo_mime", mime); err != nil {
		s.fail(w, err)
		return
	}
	s.Svc.Audit(r.Context(), rc.Actor, 0, "system.fbr_logo", "settings", "fbr_logo", map[string]any{"bytes": len(b)})
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleSystemInfo(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	writeJSON(w, 200, map[string]any{
		"product": brand.ProductName, "version": brand.Version, "buildDate": brand.BuildDate, "go": runtime.Version(),
		"os": runtime.GOOS + "/" + runtime.GOARCH, "dataDir": s.Svc.Opts.DataDir, "backupDir": s.Svc.BackupDir(),
		"fbrBaseUrl": s.Svc.Opts.Endpoints.BaseURL, "uptimeSeconds": int(time.Since(s.Started).Seconds()),
	})
}

// --- incidents ---

func (s *Server) handleListIncidents(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	list, err := s.Svc.Store.ListIncidents(r.Context(), cid(r), 500)
	if err != nil {
		s.fail(w, err)
		return
	}
	if list == nil {
		list = []*store.Incident{}
	}
	writeJSON(w, 200, list)
}

func (s *Server) handleSaveIncident(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	var in store.Incident
	if r.Method == http.MethodPut {
		id, err := pathID(r, "id")
		if err != nil {
			s.fail(w, err)
			return
		}
		cur, err := s.Svc.Store.GetIncident(r.Context(), cid(r), id)
		if err != nil {
			s.fail(w, err)
			return
		}
		in = *cur
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, err)
		return
	}
	in.CompanyID = cid(r)
	if r.Method == http.MethodPost {
		in.ID = 0
		in.AutoDetected = false
		in.CreatedBy = rc.Actor.UserID
		if in.StartedAt == "" {
			in.StartedAt = time.Now().UTC().Format(time.RFC3339)
		}
	} else {
		in.ID, _ = pathID(r, "id")
	}
	if in.Kind == "" || in.Description == "" {
		writeErr(w, 422, "kind and description are required")
		return
	}
	if err := s.Svc.Store.SaveIncident(r.Context(), &in); err != nil {
		s.fail(w, err)
		return
	}
	s.Svc.Audit(r.Context(), rc.Actor, in.CompanyID, "incident.save", "incident", fmt.Sprint(in.ID), in)
	writeJSON(w, 200, in)
}

var letterTpl = template.Must(template.New("letter").Parse(`<!doctype html><html><head><meta charset="utf-8"><title>Incident intimation</title>
<style>body{font-family:Georgia,"Times New Roman",serif;max-width:180mm;margin:20mm auto;font-size:12pt;line-height:1.5;color:#111}
td{padding:2px 8px 2px 0;vertical-align:top} .blank{display:inline-block;min-width:60mm;border-bottom:1px solid #000}
@media print{button{display:none}}</style></head><body>
<button id="print-btn" type="button">Print</button>
<script nonce="{{.Nonce}}">document.getElementById('print-btn').addEventListener('click', function () { window.print(); });</script>
<p>Date: {{.Today}}</p>
<p>To,<br>The Commissioner Inland Revenue,<br><span class="blank"></span> (Zone / RTO / LTO / CTO)<br>Federal Board of Revenue</p>
<p>Copy to: Chief (IR Operations), FBR, Islamabad; Digital Invoicing Help Desk, PRAL.</p>
<p><b>Subject: Intimation of operational failure / disruption of the electronic invoicing system under rule 150R of the Sales Tax Rules, 2006</b></p>
<p>Respected Sir/Madam,</p>
<p>In compliance with rule 150R of the Sales Tax Rules, 2006, we hereby report the following disruption of our electronic invoicing system integrated with the Board's Digital Invoicing System:</p>
<table>
<tr><td>Registered person</td><td><b>{{.Company.Name}}</b></td></tr>
<tr><td>NTN / CNIC</td><td>{{.Company.NTNCNIC}}{{if .Company.STRN}} &nbsp; STRN: {{.Company.STRN}}{{end}}</td></tr>
<tr><td>Address</td><td>{{.Company.Address}}{{if .Company.City}}, {{.Company.City}}{{end}}</td></tr>
<tr><td>Nature of incident</td><td>{{.Kind}}</td></tr>
<tr><td>Started</td><td>{{.Started}}</td></tr>
<tr><td>Ended</td><td>{{if .Ended}}{{.Ended}}{{else}}Continuing at the time of this letter{{end}}</td></tr>
<tr><td>Details</td><td>{{.Incident.Description}}</td></tr>
</table>
{{if .Invoices}}<p>Invoices issued during the period ({{len .Invoices}}):</p>
<table border="1" cellspacing="0" cellpadding="3" style="border-collapse:collapse;font-size:10pt">
<tr><th>Invoice No.</th><th>Date</th><th>Status</th><th>FBR Invoice No.</th></tr>
{{range .Invoices}}<tr><td>{{.InternalNo}}</td><td>{{.InvoiceDate}}</td><td>{{.Status}}</td><td>{{.FBRInvoiceNumber}}</td></tr>{{end}}
</table>{{end}}
<p>All invoices issued during the disruption were recorded in our system and are being / have been transmitted to the Board's computerized system upon restoration. We shall extend full cooperation for any verification.</p>
<p>Yours faithfully,</p>
<p><br><span class="blank"></span><br>Authorised signatory<br>{{.Company.Name}}<br>{{.Company.Phone}} {{.Company.Email}}</p>
</body></html>`))

var incidentKinds = map[string]string{
	"fbr_unreachable": "Loss of connectivity with FBR Digital Invoicing System",
	"auth_failure":    "Security token / authorisation failure with FBR Digital Invoicing System",
	"system_failure":  "Failure of electronic invoicing hardware or software",
	"power_failure":   "Power failure",
	"tampering":       "Suspected tampering with the electronic invoicing system",
	"other":           "Other operational disruption",
}

func fmtPKT(rfc string) string {
	t, err := time.Parse(time.RFC3339, rfc)
	if err != nil {
		return rfc
	}
	return t.In(service.PKT).Format("02-Jan-2006 15:04 (PKT)")
}

func (s *Server) handleIncidentLetter(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	id, err := pathID(r, "id")
	if err != nil {
		s.fail(w, err)
		return
	}
	inc, err := s.Svc.Store.GetIncident(r.Context(), cid(r), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	c, err := s.Svc.Store.GetCompany(r.Context(), cid(r))
	if err != nil {
		s.fail(w, err)
		return
	}
	invs, _ := s.Svc.Store.InvoicesCreatedBetween(r.Context(), c.ID, inc.StartedAt, inc.EndedAt)
	kind := incidentKinds[inc.Kind]
	if kind == "" {
		kind = inc.Kind
	}
	nonce := security.RandomToken(18)
	data := map[string]any{"Company": c, "Incident": inc, "Kind": kind, "Started": fmtPKT(inc.StartedAt), "Ended": "", "Nonce": nonce,
		"Today": time.Now().In(service.PKT).Format("02-Jan-2006"), "Invoices": invs}
	if inc.EndedAt != "" {
		data["Ended"] = fmtPKT(inc.EndedAt)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-"+nonce+"'")
	_ = letterTpl.Execute(w, data)
}

// --- mobile access ---

// handleCACert serves the installation's local CA certificate so that office
// PCs and phones can trust the server (and install the web app on phones).
func (s *Server) handleCACert(w http.ResponseWriter, r *http.Request) {
	b, err := os.ReadFile(config.CACertPath(s.Svc.Opts.DataDir))
	if err != nil {
		writeErr(w, http.StatusNotFound, "no local certificate authority: this server uses an externally issued certificate or HTTP")
		return
	}
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="veridian-einvoicing-ca.crt"`)
	_, _ = w.Write(b)
}

// handleLinkQR renders a QR code for a link on this server (used to open the
// app on a phone by scanning the screen).
func (s *Server) handleLinkQR(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	u, err := url.Parse(r.URL.Query().Get("url"))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		writeErr(w, http.StatusUnprocessableEntity, "url must be an http(s) address")
		return
	}
	svg, err := printing.LinkQRSVG(u.String())
	if err != nil {
		s.fail(w, service.Invalid("%v", err))
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	_, _ = w.Write([]byte(svg))
}

// handleAddresses lists URLs at which other devices on the network can open
// this server (the port and scheme of the current request on every
// non-loopback interface address), plus whether a local CA is available.
func (s *Server) handleAddresses(w http.ResponseWriter, r *http.Request, rc *reqCtx) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	_, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		port = ""
	}
	var urls []string
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			ipn, ok := a.(*net.IPNet)
			if !ok || ipn.IP.IsLoopback() || ipn.IP.IsLinkLocalUnicast() {
				continue
			}
			host := ipn.IP.String()
			if ipn.IP.To4() == nil {
				host = "[" + host + "]"
			}
			if port != "" {
				host += ":" + port
			}
			urls = append(urls, scheme+"://"+host+"/")
		}
	}
	if urls == nil {
		urls = []string{}
	}
	_, caErr := os.Stat(config.CACertPath(s.Svc.Opts.DataDir))
	writeJSON(w, 200, map[string]any{"urls": urls, "https": r.TLS != nil, "localCA": caErr == nil})
}
