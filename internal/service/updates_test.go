// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/license"
	"einvoicing/internal/store"
	"einvoicing/internal/update"
)

// updateServer publishes signed update files like the vendor's release page.
type updateServer struct {
	t         *testing.T
	priv      ed25519.PrivateKey
	srv       *httptest.Server
	mu        sync.Mutex
	feed      []byte
	installer []byte
}

func newUpdateServer(t *testing.T) *updateServer {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	u := &updateServer{t: t, priv: priv, installer: bytes.Repeat([]byte("new version "), 4000)}
	u.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.mu.Lock()
		defer u.mu.Unlock()
		switch {
		case r.URL.Path == "/latest/update.json" && u.feed != nil:
			w.Write(u.feed)
		case r.URL.Path == "/setup.exe":
			w.Write(u.installer)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(u.srv.Close)
	oldKey, oldFeed := update.PublicKeyB64, update.FeedURL
	update.PublicKeyB64 = base64.StdEncoding.EncodeToString(pub)
	update.FeedURL = u.srv.URL + "/latest/update.json"
	t.Cleanup(func() { update.PublicKeyB64, update.FeedURL = oldKey, oldFeed })
	return u
}

func (u *updateServer) publish(version, released string, mod func(*update.Manifest)) *update.Manifest {
	sum := sha256.Sum256(u.installer)
	m := update.Manifest{Product: brand.ShortName, Version: version, Released: released, Notes: "What's new in " + version,
		Files: map[string]update.File{update.Platform(): {URL: u.srv.URL + "/setup.exe", SHA256: hex.EncodeToString(sum[:]), Size: int64(len(u.installer))}}}
	if mod != nil {
		mod(&m)
	}
	b, err := update.Sign(m, u.priv)
	if err != nil {
		u.t.Fatal(err)
	}
	u.mu.Lock()
	u.feed = b
	u.mu.Unlock()
	return &m
}

// prepare lets the fixture's service download from the test server and
// records the installers it would start.
func (u *updateServer) prepare(f *fixture) chan update.Staged {
	launched := make(chan update.Staged, 4)
	f.svc.upd.client.HTTP.Transport = u.srv.Client().Transport
	f.svc.upd.capability = func(string) (bool, string) { return true, "" }
	f.svc.upd.launch = func(st update.Staged) error { launched <- st; return nil }
	f.svc.upd.launchDelay = 0
	f.svc.upd.installerRunning = func() bool { return false }
	return launched
}

func waitLaunch(t *testing.T, f *fixture, launched chan update.Staged) update.Staged {
	t.Helper()
	select {
	case st := <-launched:
		return st
	case <-time.After(10 * time.Second):
		st, _ := f.svc.UpdateStatus(f.ctx)
		t.Fatalf("installer not started: %+v", st)
	}
	return update.Staged{}
}

func waitIdle(t *testing.T, f *fixture) *UpdateStatus {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		st, err := f.svc.UpdateStatus(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		if st.State != "downloading" && (st.State != "installing" || st.Pending != nil) || time.Now().After(deadline) {
			return st
		}
		time.Sleep(20 * time.Millisecond)
	}
}

type fakeLicense struct{ st license.Status }

func (fakeLicense) CheckProduction(string) error            { return nil }
func (l fakeLicense) Status(context.Context) license.Status { return l.st }

func TestSoftwareUpdate(t *testing.T) {
	f := setup(t)
	u := newUpdateServer(t)
	launched := u.prepare(f)

	st, err := f.svc.UpdateStatus(f.ctx)
	if err != nil || !st.Configured || st.Available != nil || st.Settings.Mode != UpdateNotify || st.Current != brand.Version {
		t.Fatalf("initial status: %+v %v", st, err)
	}
	// Nothing published yet, then only the version already installed.
	if st, err = f.svc.CheckForUpdate(f.ctx); err != nil || st.Available != nil || st.LastError != "" || st.LastCheck == "" {
		t.Fatalf("nothing published: %+v %v", st, err)
	}
	u.publish(brand.Version, "2026-10-01", nil)
	if st, _ = f.svc.CheckForUpdate(f.ctx); st.Available != nil {
		t.Fatal("the installed version was offered as an update")
	}
	if _, err := f.svc.StartUpdate(f.ctx, f.admin, true); !IsValidation(err) {
		t.Fatalf("update without a newer version: %v", err)
	}

	// A newer version is found and shown to administrators.
	u.publish("99.0.0", "2026-10-20", nil)
	if st, _ = f.svc.CheckForUpdate(f.ctx); st.Available == nil || st.Available.Version != "99.0.0" || st.Available.Notes != "What's new in 99.0.0" {
		t.Fatalf("newer version not found: %+v", st)
	}
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	found := false
	for _, a := range f.svc.Alerts(f.ctx, c, c.Environment, true) {
		found = found || (a.ID == "update" && strings.Contains(a.Title, "99.0.0"))
	}
	for _, a := range f.svc.Alerts(f.ctx, c, c.Environment, false) {
		if a.ID == "update" {
			t.Fatal("update alert shown to a user who cannot install it")
		}
	}
	if !found {
		t.Fatal("no update alert for administrators")
	}

	// Update now: download, verify, back up, start the installer.
	if _, err := f.svc.StartUpdate(f.ctx, f.admin, true); err != nil {
		t.Fatal(err)
	}
	staged := waitLaunch(t, f, launched)
	got, err := os.ReadFile(staged.Path)
	if err != nil || !bytes.Equal(got, u.installer) || staged.Manifest.Version != "99.0.0" {
		t.Fatalf("staged installer: %v", err)
	}
	if _, err := os.Stat(filepath.Join(staged.Dir, "update.json")); err != nil {
		t.Fatal("signed update file not kept next to the installer")
	}
	st = waitIdle(t, f)
	if st.Pending == nil || st.Pending.To != "99.0.0" || !strings.HasPrefix(st.Pending.Backup, "einvoice-pre-update-") {
		t.Fatalf("pending update: %+v", st.Pending)
	}
	if _, err := os.Stat(filepath.Join(f.svc.BackupDir(), st.Pending.Backup)); err != nil {
		t.Fatal("no backup before the update")
	}
	if _, err := f.svc.StartUpdate(f.ctx, f.admin, true); !IsValidation(err) {
		t.Fatalf("second update while one is installing: %v", err)
	}

	// The service starts again on the new version.
	old := brand.Version
	brand.Version = "99.0.0"
	f.svc.FinishUpdate(f.ctx)
	st, _ = f.svc.UpdateStatus(f.ctx)
	wn := f.svc.WhatsNew(f.ctx)
	brand.Version = old
	if st.Pending != nil || st.Last == nil || !st.Last.OK || st.Last.From != old || st.Available != nil {
		t.Fatalf("after the update: %+v", st)
	}
	if wn == nil || wn.Notes != "What's new in 99.0.0" {
		t.Fatalf("what's new: %+v", wn)
	}
	if _, err := os.Stat(filepath.Join(f.svc.Opts.DataDir, "updates")); !os.IsNotExist(err) {
		t.Fatal("update files left after a successful update")
	}

	// An update recorded as failed that completed later (restarted by hand).
	_ = f.svc.setJSON(f.ctx, keyUpdLast, UpdateResult{From: old, To: "99.0.1", Message: "the service could not be restarted"})
	brand.Version = "99.0.1"
	f.svc.FinishUpdate(f.ctx)
	brand.Version = old
	var last UpdateResult
	if !f.svc.getJSON(f.ctx, keyUpdLast, &last) || !last.OK || last.To != "99.0.1" {
		t.Fatalf("update completed later: %+v", last)
	}
	entries, _, _ := f.svc.Store.ListAudit(f.ctx, store.AuditFilter{Action: "system.update"})
	actions := map[string]bool{}
	for _, e := range entries {
		actions[e.Action] = true
	}
	if !actions["system.update.start"] || !actions["system.update.done"] {
		t.Fatalf("audit trail: %v", actions)
	}
}

func TestSoftwareUpdateFailures(t *testing.T) {
	f := setup(t)
	u := newUpdateServer(t)
	launched := u.prepare(f)

	// A file not signed with the vendor's key is refused.
	m := u.publish("99.0.0", "2026-10-20", nil)
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	forged, _ := update.Sign(*m, other)
	u.mu.Lock()
	u.feed = forged
	u.mu.Unlock()
	st, err := f.svc.CheckForUpdate(f.ctx)
	if err != nil || st.Available != nil || !strings.Contains(st.LastError, "signature") {
		t.Fatalf("forged update file: %+v %v", st, err)
	}

	// An installer that does not match the signed file is never started.
	u.publish("99.0.0", "2026-10-20", func(m *update.Manifest) {
		fl := m.Files[update.Platform()]
		fl.SHA256 = strings.Repeat("0", 64)
		m.Files[update.Platform()] = fl
	})
	if _, err := f.svc.CheckForUpdate(f.ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.StartUpdate(f.ctx, f.admin, true); err != nil {
		t.Fatal(err)
	}
	st = waitIdle(t, f)
	if st.State != "failed" || !strings.Contains(st.Message, "SHA-256") || st.Last == nil || st.Last.OK || st.Pending != nil {
		t.Fatalf("altered installer: %+v", st)
	}
	select {
	case <-launched:
		t.Fatal("altered installer started")
	default:
	}

	// Download only, for installing by hand.
	u.publish("99.0.0", "2026-10-20", nil)
	f.svc.CheckForUpdate(f.ctx)
	if _, err := f.svc.StartUpdate(f.ctx, f.admin, false); err != nil {
		t.Fatal(err)
	}
	if st = waitIdle(t, f); st.State != "ready" || !st.Ready {
		t.Fatalf("download only: %+v", st)
	}
	if p, err := f.svc.VerifiedUpdateFile(); err != nil || !strings.HasSuffix(p, "setup.exe") {
		t.Fatalf("verified file: %q %v", p, err)
	}

	// An installer that stops without installing: still the old version
	// 15 minutes later, so the attempt is recorded as failed.
	if _, err := f.svc.StartUpdate(f.ctx, f.admin, true); err != nil {
		t.Fatal(err)
	}
	waitLaunch(t, f, launched)
	waitIdle(t, f)
	f.svc.Now = func() time.Time { return time.Now().Add(20 * time.Minute) }
	st, _ = f.svc.UpdateStatus(f.ctx)
	f.svc.Now = time.Now
	if st.Pending != nil || st.Last == nil || st.Last.OK || !strings.Contains(st.Last.Message, "did not complete") || st.State != "failed" {
		t.Fatalf("installer that did not finish: %+v", st)
	}

	// The installer ended and the old version started again (Windows).
	if _, err := f.svc.StartUpdate(f.ctx, f.admin, true); err != nil {
		t.Fatal(err)
	}
	waitLaunch(t, f, launched)
	waitIdle(t, f)
	running := true
	f.svc.upd.installerRunning = func() bool { return running }
	f.svc.upd.started = time.Now().Add(time.Minute) // a process started after the installer
	if st, _ = f.svc.UpdateStatus(f.ctx); st.Pending == nil {
		t.Fatal("reported as failed while the installer still runs")
	}
	running = false
	if st, _ = f.svc.UpdateStatus(f.ctx); st.Pending != nil || st.Last == nil || st.Last.OK || !strings.Contains(st.Last.Message, "stopped before") {
		t.Fatalf("installer that stopped: %+v", st.Last)
	}
	f.svc.upd.started = time.Now().Add(-time.Hour)

	// The support period of the licence ended before the release.
	f.svc.Opts.License = fakeLicense{license.Status{Mode: "licensed", SupportUntil: "2026-06-30"}}
	if st, _ = f.svc.UpdateStatus(f.ctx); !strings.Contains(st.Blocked, "renew support") || st.NextAuto != "" {
		t.Fatalf("support ended: %+v", st)
	}
	if _, err := f.svc.StartUpdate(f.ctx, f.admin, true); !IsValidation(err) {
		t.Fatalf("update after the support period: %v", err)
	}
	u.publish("99.0.0", "2026-10-20", func(m *update.Manifest) { m.Free = true })
	if st, _ = f.svc.CheckForUpdate(f.ctx); st.Blocked != "" {
		t.Fatalf("free update blocked: %s", st.Blocked)
	}
	f.svc.Opts.License = fakeLicense{license.Status{Mode: "licensed", SupportUntil: "2027-06-30"}}
	u.publish("99.0.0", "2026-10-20", nil)
	if st, _ = f.svc.CheckForUpdate(f.ctx); st.Blocked != "" {
		t.Fatalf("update within the support period blocked: %s", st.Blocked)
	}

	// Settings are validated.
	for _, in := range []UpdateSettings{{Mode: "sometimes", Hour: 2}, {Mode: UpdateAuto, Hour: 24}, {Mode: UpdateAuto, Hour: 2, Feed: "http://updates.example.pk/update.json"}} {
		if _, err := f.svc.SaveUpdateSettings(f.ctx, f.admin, in); !IsValidation(err) {
			t.Errorf("settings %+v accepted", in)
		}
	}
	// Another update address: the previous answer no longer counts.
	if st, err = f.svc.SaveUpdateSettings(f.ctx, f.admin, UpdateSettings{Mode: UpdateNotify, Hour: 3, Feed: u.srv.URL + "/other.json"}); err != nil || st.Available != nil {
		t.Fatalf("new address: %+v %v", st, err)
	}
	if st, _ = f.svc.CheckForUpdate(f.ctx); st.Available != nil || st.LastError != "" {
		t.Fatalf("empty address: %+v", st)
	}
}

func TestAutomaticUpdate(t *testing.T) {
	f := setup(t)
	u := newUpdateServer(t)
	launched := u.prepare(f)
	f.svc.upd.started = time.Now().Add(-time.Hour)
	u.publish("99.0.0", "2026-10-20", nil)
	at := func(hour int) {
		f.svc.Now = func() time.Time { return time.Date(2026, 10, 21, hour, 30, 0, 0, PKT) }
	}
	t.Cleanup(func() { f.svc.Now = time.Now })

	// "Tell me": found, never installed by itself.
	at(2)
	f.svc.updateTick(f.ctx)
	st, _ := f.svc.UpdateStatus(f.ctx)
	if st.Available == nil || st.NextAuto != "" {
		t.Fatalf("notify mode: %+v", st)
	}
	select {
	case <-launched:
		t.Fatal("installed in notify mode")
	case <-time.After(200 * time.Millisecond):
	}

	// Automatic: installed at the chosen hour only.
	if _, err := f.svc.SaveUpdateSettings(f.ctx, f.admin, UpdateSettings{Mode: UpdateAuto, Hour: 3}); err != nil {
		t.Fatal(err)
	}
	if st, _ = f.svc.UpdateStatus(f.ctx); st.NextAuto != "2026-10-20T22:00:00Z" {
		t.Fatalf("next automatic installation: %q", st.NextAuto)
	}
	f.svc.updateTick(f.ctx)
	select {
	case <-launched:
		t.Fatal("installed outside the update hour")
	case <-time.After(200 * time.Millisecond):
	}
	at(3)
	f.svc.updateTick(f.ctx)
	select {
	case st := <-launched:
		if st.Manifest.Version != "99.0.0" {
			t.Fatalf("installed %s", st.Manifest.Version)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("not installed at the update hour")
	}
	waitIdle(t, f)

	// "Off": no checks at all.
	if _, err := f.svc.SaveUpdateSettings(f.ctx, f.admin, UpdateSettings{Mode: UpdateOff, Hour: 3}); err != nil {
		t.Fatal(err)
	}
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	for _, a := range f.svc.Alerts(f.ctx, c, c.Environment, true) {
		if a.ID == "update" {
			t.Fatal("update alert with updates turned off")
		}
	}
}

func TestMandatoryUpdate(t *testing.T) {
	f := setup(t)
	u := newUpdateServer(t)
	launched := u.prepare(f)
	f.svc.upd.started = time.Now().Add(-time.Hour)
	u.publish("99.0.0", "2026-10-20", func(m *update.Manifest) { m.Mandatory = true })
	f.svc.Now = func() time.Time { return time.Date(2026, 10, 21, 2, 5, 0, 0, PKT) }
	t.Cleanup(func() { f.svc.Now = time.Now })
	f.svc.updateTick(f.ctx)
	select {
	case <-launched:
	case <-time.After(10 * time.Second):
		t.Fatal("required update not installed at the update hour")
	}
	waitIdle(t, f)
	c, _ := f.svc.Store.GetCompany(f.ctx, f.cid)
	for _, a := range f.svc.Alerts(f.ctx, c, c.Environment, true) {
		if a.ID == "update" && a.Severity != "warning" {
			t.Fatalf("required update alert: %+v", a)
		}
	}
}

func TestOldUpdateFilesRemoved(t *testing.T) {
	f := setup(t)
	dir := filepath.Join(f.svc.Opts.DataDir, "updates")
	for _, v := range []string{brand.Version, "0.0.1", "99.0.0"} {
		if err := os.MkdirAll(filepath.Join(dir, v), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	f.svc.updateTick(f.ctx)
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "99.0.0" {
		t.Fatalf("left in the updates folder: %v", entries)
	}
}

func TestUpdatesNotConfigured(t *testing.T) {
	f := setup(t)
	oldKey, oldFeed := update.PublicKeyB64, update.FeedURL
	update.PublicKeyB64, update.FeedURL = "", ""
	t.Cleanup(func() { update.PublicKeyB64, update.FeedURL = oldKey, oldFeed })
	st, err := f.svc.UpdateStatus(f.ctx)
	if err != nil || st.Configured {
		t.Fatalf("status: %+v %v", st, err)
	}
	if _, err := f.svc.CheckForUpdate(f.ctx); !IsValidation(err) {
		t.Fatalf("check without a key: %v", err)
	}
	f.svc.updateTick(f.ctx) // must not panic or call out
}
