package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"einvoicing/internal/db"
	"einvoicing/internal/domain"
	"einvoicing/internal/license"
	"einvoicing/internal/security"
	"einvoicing/internal/service"
	"einvoicing/internal/store"
)

type env struct {
	t      *testing.T
	svc    *service.Service
	srv    *httptest.Server
	compA  int64
	compB  int64
	adminA *client
}

type client struct {
	t    *testing.T
	base string
	hc   *http.Client
	csrf string
}

func newEnv(t *testing.T) *env {
	t.Helper()
	dir := t.TempDir()
	d, err := db.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	vault, err := security.OpenVault(dir)
	if err != nil {
		t.Fatal(err)
	}
	st := store.New(d)
	lic, err := license.NewManager(st) // developer build: no licence checks
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.New(st, vault, service.Options{DataDir: dir, License: lic, Logger: log})
	ctx := context.Background()
	if err := svc.EnsureSeedData(ctx); err != nil {
		t.Fatal(err)
	}
	if err := svc.Setup(ctx, "127.0.0.1", service.SetupInput{AdminUsername: "admin", AdminPassword: "Admin12345", AdminFullName: "Admin",
		Company: store.Company{Name: "Alpha Traders", NTNCNIC: "1234567", Province: "Sindh", Address: "Karachi",
			BusinessActivities: []string{"Wholesaler"}, Sectors: []string{"All Other Sectors"}, Environment: domain.EnvSimulator}}); err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, svc: svc}
	cs, _ := st.ListCompanies(ctx)
	e.compA = cs[0].ID
	u, _ := st.GetUserByUsername(ctx, "admin")
	uid := u.ID
	admin := service.Actor{UserID: &uid, Username: "admin", Role: store.RoleAdmin}
	b, err := svc.SaveCompany(ctx, admin, &store.Company{Name: "Bravo Industries", NTNCNIC: "7654321", Province: "Punjab", Address: "Lahore",
		Environment: domain.EnvSimulator})
	if err != nil {
		t.Fatal(err)
	}
	e.compB = b.ID
	for _, nu := range []service.UserInput{
		{Username: "auditorA", Role: store.RoleAuditor, Password: "Initial123", CompanyIDs: []int64{e.compA}},
		{Username: "operatorA", Role: store.RoleOperator, Password: "Initial123", CompanyIDs: []int64{e.compA}},
	} {
		if _, err := svc.SaveUser(ctx, admin, 0, nu); err != nil {
			t.Fatal(err)
		}
	}
	e.srv = httptest.NewServer(New(svc, lic, nil, log, false).Handler())
	t.Cleanup(e.srv.Close)
	e.adminA = e.login("admin", "Admin12345")
	return e
}

func (e *env) login(user, pw string) *client {
	e.t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &client{t: e.t, base: e.srv.URL + "/api/v1", hc: &http.Client{Jar: jar}}
	status, body := c.do("POST", "/auth/login", map[string]string{"username": user, "password": pw})
	if status != 200 {
		e.t.Fatalf("login %s: %d %s", user, status, body)
	}
	var me struct {
		CSRF string `json:"csrf"`
		User struct {
			MustChangePassword bool `json:"mustChangePassword"`
		} `json:"user"`
	}
	_ = json.Unmarshal(body, &me)
	c.csrf = me.CSRF
	if me.User.MustChangePassword {
		if st, b := c.do("POST", "/auth/password", map[string]string{"current": pw, "new": pw + "x1"}); st != 200 {
			e.t.Fatalf("password change %s: %d %s", user, st, b)
		}
	}
	return c
}

func (c *client) do(method, path string, body any) (int, []byte) {
	c.t.Helper()
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+path, rd)
	req.Header.Set("Content-Type", "application/json")
	if c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, b
}

func (c *client) get(path string) (*http.Response, []byte) {
	c.t.Helper()
	resp, err := c.hc.Get(c.base + path)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b
}

func sampleInvoice(extra map[string]any) map[string]any {
	inv := map[string]any{
		"buyer": map[string]any{"ntnCnic": "2046004", "name": "Buyer", "province": "PUNJAB", "address": "Lahore", "registrationType": "Registered"},
		"items": []map[string]any{{"hsCode": "0101.2100", "description": "Item", "uom": "Numbers, pieces, units", "quantity": 1, "unitPrice": 100,
			"saleType": "Goods at standard rate (default)", "rate": "18%"}},
		"submit": true,
	}
	for k, v := range extra {
		inv[k] = v
	}
	return inv
}

// Users restricted to some companies must not read other companies' audit entries.
func TestAuditTrailIsScopedToUserCompanies(t *testing.T) {
	e := newEnv(t)
	if st, b := e.adminA.do("POST", fmt.Sprintf("/companies/%d/customers", e.compB),
		map[string]any{"name": "Confidential Buyer of Bravo", "registrationType": "Unregistered", "province": "Punjab"}); st != 201 {
		t.Fatalf("customer: %d %s", st, b)
	}
	aud := e.login("auditorA", "Initial123")
	if st, _ := aud.do("GET", "/audit?limit=200", nil); st != 403 {
		t.Fatalf("installation-wide audit trail for a restricted auditor: got %d, want 403", st)
	}
	if st, _ := aud.do("GET", fmt.Sprintf("/audit?companyId=%d", e.compB), nil); st != 403 {
		t.Fatalf("other company's audit trail: got %d, want 403", st)
	}
	st, b := aud.do("GET", fmt.Sprintf("/audit?companyId=%d&limit=200", e.compA), nil)
	if st != 200 || strings.Contains(string(b), "Bravo") {
		t.Fatalf("own company audit trail: %d, leaked=%v", st, strings.Contains(string(b), "Bravo"))
	}
	if st, b := e.adminA.do("GET", "/audit?limit=200", nil); st != 200 || !strings.Contains(string(b), "Bravo") {
		t.Fatalf("administrator global audit trail: %d", st)
	}
}

// Invoices always use the company's working environment.
func TestInvoiceEnvironmentCannotBeOverridden(t *testing.T) {
	e := newEnv(t)
	op := e.login("operatorA", "Initial123")
	path := fmt.Sprintf("/companies/%d/invoices", e.compA)
	if st, b := op.do("POST", path, sampleInvoice(map[string]any{"environment": "production"})); st != 422 {
		t.Fatalf("production override: got %d %s, want 422", st, b)
	}
	st, b := op.do("POST", path, sampleInvoice(map[string]any{"environment": "simulator"}))
	if st != 201 || !strings.Contains(string(b), `"status":"ACCEPTED"`) {
		t.Fatalf("matching environment: %d %s", st, b)
	}
}

// Large QR scales are clamped so a request cannot exhaust memory.
func TestQRPNGScaleIsBounded(t *testing.T) {
	e := newEnv(t)
	st, b := e.adminA.do("POST", fmt.Sprintf("/companies/%d/invoices", e.compA), sampleInvoice(nil))
	if st != 201 {
		t.Fatalf("invoice: %d %s", st, b)
	}
	var inv struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(b, &inv)
	resp, body := e.adminA.get(fmt.Sprintf("/companies/%d/invoices/%d/qr.png?scale=100000", e.compA, inv.ID))
	if resp.StatusCode != 200 {
		t.Fatalf("qr.png: %d", resp.StatusCode)
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(body))
	if err != nil || cfg.Width > 25*40 {
		t.Fatalf("png %dx%d err=%v; want at most 1000 px", cfg.Width, cfg.Height, err)
	}
}

// The print page allows only its own nonce'd script.
func TestPrintPageUsesNonceCSP(t *testing.T) {
	e := newEnv(t)
	st, b := e.adminA.do("POST", fmt.Sprintf("/companies/%d/invoices", e.compA), sampleInvoice(nil))
	if st != 201 {
		t.Fatalf("invoice: %d %s", st, b)
	}
	var inv struct {
		ID int64 `json:"id"`
	}
	_ = json.Unmarshal(b, &inv)
	resp, body := e.adminA.get(fmt.Sprintf("/companies/%d/invoices/%d/print?autoprint=1&preview=1", e.compA, inv.ID))
	csp := resp.Header.Get("Content-Security-Policy")
	scriptSrc := ""
	for _, d := range strings.Split(csp, ";") {
		if d = strings.TrimSpace(d); strings.HasPrefix(d, "script-src") {
			scriptSrc = d
		}
	}
	if !strings.HasPrefix(scriptSrc, "script-src 'nonce-") || strings.Contains(scriptSrc, "unsafe-inline") {
		t.Fatalf("print CSP script-src: %q", scriptSrc)
	}
	nonce := csp[strings.Index(csp, "'nonce-")+7:]
	nonce = nonce[:strings.Index(nonce, "'")]
	if !strings.Contains(string(body), `nonce="`+nonce+`"`) || strings.Contains(string(body), "onclick=") {
		t.Fatal("print page script must carry the CSP nonce and no inline handlers")
	}
}

// Changing one's password signs out every other session of that user.
func TestPasswordChangeRevokesOtherSessions(t *testing.T) {
	e := newEnv(t)
	other := e.login("admin", "Admin12345")
	if st, b := e.adminA.do("POST", "/auth/password", map[string]string{"current": "Admin12345", "new": "Changed123"}); st != 200 {
		t.Fatalf("change: %d %s", st, b)
	}
	if st, _ := other.do("GET", "/auth/me", nil); st != 401 {
		t.Fatalf("other session after password change: got %d, want 401", st)
	}
	if st, _ := e.adminA.do("GET", "/auth/me", nil); st != 200 {
		t.Fatalf("current session after password change: got %d, want 200", st)
	}
}

// Login failures do not reveal whether a username exists or is locked.
func TestLoginFailuresAreIndistinguishable(t *testing.T) {
	e := newEnv(t)
	c := &client{t: t, base: e.srv.URL + "/api/v1", hc: &http.Client{}}
	_, unknown := c.do("POST", "/auth/login", map[string]string{"username": "nobody", "password": "x"})
	_, wrong := c.do("POST", "/auth/login", map[string]string{"username": "operatorA", "password": "x"})
	for i := 0; i < 5; i++ {
		c.do("POST", "/auth/login", map[string]string{"username": "operatorA", "password": "x"})
	}
	st, locked := c.do("POST", "/auth/login", map[string]string{"username": "operatorA", "password": "Initial123"})
	if st != 401 || string(unknown) != string(wrong) || string(wrong) != string(locked) {
		t.Fatalf("responses differ:\n unknown=%s\n wrong=%s\n locked(%d)=%s", unknown, wrong, st, locked)
	}
}

// Oversized imports are rejected before they are buffered.
func TestImportRejectsOversizedUpload(t *testing.T) {
	e := newEnv(t)
	var buf bytes.Buffer
	buf.WriteString("--b\r\nContent-Disposition: form-data; name=\"file\"; filename=\"big.csv\"\r\nContent-Type: text/csv\r\n\r\n")
	buf.Write(bytes.Repeat([]byte("x"), 22<<20))
	buf.WriteString("\r\n--b--\r\n")
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/companies/%d/import?preview=1", e.srv.URL, e.compA), &buf)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=b")
	req.Header.Set("X-CSRF-Token", e.adminA.csrf)
	resp, err := e.adminA.hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized import: got %d, want 413", resp.StatusCode)
	}
}
