// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/store"
	"einvoicing/internal/update"
)

// Software updates. An administrator clicks Update now, or the installation
// does it at the chosen hour: the vendor's signed update file is read, the
// installer downloaded and checked against it, the database backed up and
// the installer started. The installer restarts the service on the new
// version, which then records the update as complete.

// Update modes.
const (
	UpdateAuto   = "auto"   // install new versions at the update hour
	UpdateNotify = "notify" // tell administrators, who click Update now
	UpdateOff    = "off"    // never look for updates
)

// UpdateSettings are the administrator's choices.
type UpdateSettings struct {
	Mode string `json:"mode"`
	Hour int    `json:"hour"` // Pakistan time, 0-23
	Feed string `json:"feed"` // update address; empty = the one built in
}

// UpdateResult records an installation attempt.
type UpdateResult struct {
	From    string `json:"from"`
	To      string `json:"to"`
	At      string `json:"at"`
	By      string `json:"by"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Backup  string `json:"backup,omitempty"`
	Notes   string `json:"notes,omitempty"`
}

// UpdateStatus is shown on the System page.
type UpdateStatus struct {
	Current     string           `json:"current"`
	BuildDate   string           `json:"buildDate"`
	Platform    string           `json:"platform"`
	Configured  bool             `json:"configured"` // the build can verify updates and knows where to look
	Settings    UpdateSettings   `json:"settings"`
	BuiltInFeed string           `json:"builtInFeed"`
	LastCheck   string           `json:"lastCheck,omitempty"`
	LastError   string           `json:"lastError,omitempty"`
	Available   *update.Manifest `json:"available,omitempty"`
	State       string           `json:"state"` // idle | downloading | ready | installing | failed
	Message     string           `json:"message,omitempty"`
	Downloaded  int64            `json:"downloaded"`
	Size        int64            `json:"size"`
	CanInstall  bool             `json:"canInstall"`
	ManualNote  string           `json:"manualNote,omitempty"` // how to install when the app cannot
	Blocked     string           `json:"blocked,omitempty"`    // the licence's support period has ended
	Ready       bool             `json:"ready"`                // the verified installer can be downloaded
	NextAuto    string           `json:"nextAuto,omitempty"`   // when the update installs by itself
	Pending     *UpdateResult    `json:"pending,omitempty"`
	Last        *UpdateResult    `json:"last,omitempty"`
}

// updateCheck is the outcome of the last check.
type updateCheck struct {
	At     string `json:"at"`
	Error  string `json:"error,omitempty"`
	Signed string `json:"signed,omitempty"` // the newer version's signed update file
}

// updater holds the state of updates in this process.
type updater struct {
	mu          sync.Mutex
	state       string
	message     string
	downloaded  int64
	size        int64
	ready       *update.Staged
	lastCheck   time.Time
	lastTick    time.Time
	lastAuto    string // Pakistan day of the last automatic installation
	started     time.Time
	ticking     bool
	client      *update.Client
	launch      func(update.Staged) error
	capability  func(dataDir string) (bool, string)
	launchDelay time.Duration
	// installerRunning reports a running Windows installer.
	installerRunning func() bool
}

func newUpdater() *updater {
	return &updater{state: "idle", started: time.Now(), client: update.NewClient(), launch: update.Launch,
		capability: update.Capability, launchDelay: 1500 * time.Millisecond, installerRunning: update.InstallerRunning}
}

func (u *updater) set(state, msg string) {
	u.mu.Lock()
	u.state, u.message = state, msg
	u.mu.Unlock()
}

func (u *updater) busy() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.state == "downloading" || u.state == "installing"
}

const (
	keyUpdSettings = "update.settings"
	keyUpdCheck    = "update.check"
	keyUpdPending  = "update.pending"
	keyUpdLast     = "update.last"
)

func (s *Service) getJSON(ctx context.Context, key string, v any) bool {
	raw, err := s.Store.GetSetting(ctx, key)
	return err == nil && raw != "" && json.Unmarshal([]byte(raw), v) == nil
}

func (s *Service) setJSON(ctx context.Context, key string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.Store.SetSetting(ctx, key, string(b))
}

func (s *Service) updateSettings(ctx context.Context) UpdateSettings {
	set := UpdateSettings{Mode: UpdateNotify, Hour: 2}
	s.getJSON(ctx, keyUpdSettings, &set)
	if set.Mode != UpdateAuto && set.Mode != UpdateOff {
		set.Mode = UpdateNotify
	}
	if set.Hour < 0 || set.Hour > 23 {
		set.Hour = 2
	}
	return set
}

func (s *Service) updateFeed(ctx context.Context) string {
	if f := s.updateSettings(ctx).Feed; f != "" {
		return f
	}
	return update.FeedURL
}

// SaveUpdateSettings stores the administrator's update choices.
func (s *Service) SaveUpdateSettings(ctx context.Context, a Actor, in UpdateSettings) (*UpdateStatus, error) {
	if in.Mode != UpdateAuto && in.Mode != UpdateNotify && in.Mode != UpdateOff {
		return nil, Invalid("unknown update mode %q", in.Mode)
	}
	if in.Hour < 0 || in.Hour > 23 {
		return nil, Invalid("the update hour must be between 0 and 23")
	}
	in.Feed = strings.TrimSpace(in.Feed)
	if in.Feed != "" {
		u, err := url.Parse(in.Feed)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return nil, Invalid("the update address must start with https://")
		}
	}
	if in.Feed == update.FeedURL {
		in.Feed = ""
	}
	old := s.updateSettings(ctx)
	if err := s.setJSON(ctx, keyUpdSettings, in); err != nil {
		return nil, err
	}
	if old.Feed != in.Feed {
		// The last answer came from the previous address.
		_ = s.Store.SetSetting(ctx, keyUpdCheck, "")
	}
	s.Audit(ctx, a, 0, "system.update.settings", "system", "updates", in)
	return s.UpdateStatus(ctx)
}

// availableUpdate returns the last check and the newer version it found.
func (s *Service) availableUpdate(ctx context.Context) (updateCheck, *update.Manifest) {
	var c updateCheck
	if !s.getJSON(ctx, keyUpdCheck, &c) || c.Signed == "" {
		return c, nil
	}
	pub, _ := update.PublicKey()
	m, err := update.Parse([]byte(c.Signed), pub)
	if err != nil || !update.Newer(m.Version, brand.Version) {
		return c, nil
	}
	return c, m
}

// CheckForUpdate asks the vendor's update address for a newer version. A
// failed check is reported in the status, not as an error.
func (s *Service) CheckForUpdate(ctx context.Context) (*UpdateStatus, error) {
	pub, err := update.PublicKey()
	if err != nil {
		return nil, err
	}
	feed := s.updateFeed(ctx)
	if pub == nil || feed == "" {
		return nil, Invalid("%s", update.ErrNotConfigured.Error())
	}
	cctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	m, raw, err := s.upd.client.FetchManifest(cctx, feed, pub)
	c := updateCheck{At: s.Now().UTC().Format(time.RFC3339)}
	switch {
	case errors.Is(err, update.ErrNoRelease):
		// Nothing published yet: this is the latest version.
	case err != nil:
		c.Error = err.Error()
		s.Log.Warn("software update check failed", "err", err)
	case update.Newer(m.Version, brand.Version):
		c.Signed = string(raw)
		s.Log.Info("software update available", "version", m.Version)
	}
	if err := s.setJSON(ctx, keyUpdCheck, c); err != nil {
		return nil, err
	}
	s.upd.mu.Lock()
	s.upd.lastCheck = time.Now()
	s.upd.mu.Unlock()
	return s.UpdateStatus(ctx)
}

// UpdateStatus reports the installed version, the update found by the last
// check and any update in progress.
func (s *Service) UpdateStatus(ctx context.Context) (*UpdateStatus, error) {
	s.reconcileUpdate(ctx)
	pub, _ := update.PublicKey()
	set := s.updateSettings(ctx)
	st := &UpdateStatus{Current: brand.Version, BuildDate: brand.BuildDate, Platform: update.Platform(), Settings: set,
		BuiltInFeed: update.FeedURL, Configured: pub != nil && s.updateFeed(ctx) != ""}
	c, m := s.availableUpdate(ctx)
	st.LastCheck, st.LastError, st.Available = c.At, c.Error, m
	st.CanInstall, st.ManualNote = s.upd.capability(s.Opts.DataDir)
	if m != nil {
		st.Blocked = s.supportBlock(ctx, m)
	}
	s.upd.mu.Lock()
	st.State, st.Message, st.Downloaded, st.Size = s.upd.state, s.upd.message, s.upd.downloaded, s.upd.size
	st.Ready = s.upd.ready != nil && m != nil && s.upd.ready.Manifest.Version == m.Version
	s.upd.mu.Unlock()
	var p, last UpdateResult
	if s.getJSON(ctx, keyUpdPending, &p) && p.To != "" {
		st.Pending = &p
	}
	if s.getJSON(ctx, keyUpdLast, &last) && last.To != "" {
		st.Last = &last
	}
	if m != nil && set.Mode != UpdateOff && (set.Mode == UpdateAuto || m.Mandatory) && st.CanInstall && st.Blocked == "" {
		now := s.Now().In(PKT)
		next := time.Date(now.Year(), now.Month(), now.Day(), set.Hour, 0, 0, 0, PKT)
		if !next.After(now) && now.Hour() != set.Hour {
			next = next.AddDate(0, 0, 1)
		}
		st.NextAuto = next.UTC().Format(time.RFC3339)
	}
	return st, nil
}

// supportBlock explains why an update may not be installed under the
// licence: it was released after the support and updates period ended.
func (s *Service) supportBlock(ctx context.Context, m *update.Manifest) string {
	if m.Free || m.Released == "" {
		return ""
	}
	lr, ok := s.Opts.License.(LicenseReporter)
	if !ok {
		return ""
	}
	lic := lr.Status(ctx)
	if lic.SupportUntil == "" || (lic.Mode != "licensed" && lic.Mode != "grace" && lic.Mode != "expired") || m.Released <= lic.SupportUntil {
		return ""
	}
	return fmt.Sprintf("Version %s was released on %s, after the support and updates period of your licence ended (%s). To install it, renew support with %s (%s).",
		m.Version, dayDate(m.Released), dayDate(lic.SupportUntil), brand.Vendor, brand.SupportContact)
}

// dayDate formats YYYY-MM-DD as e.g. "20 Oct 2026".
func dayDate(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return iso
	}
	return t.Format("2 Jan 2006")
}

// StartUpdate downloads the available update in the background and, with
// install set, backs up the database and starts the installer.
func (s *Service) StartUpdate(ctx context.Context, a Actor, install bool) (*UpdateStatus, error) {
	st, err := s.UpdateStatus(ctx)
	if err != nil {
		return nil, err
	}
	m := st.Available
	switch {
	case m == nil:
		return nil, Invalid("No newer version is available. Click Check for updates.")
	case st.Blocked != "":
		return nil, Invalid("%s", st.Blocked)
	case install && !st.CanInstall:
		return nil, Invalid("%s", st.ManualNote)
	case st.Pending != nil:
		return nil, Invalid("The update to version %s is being installed.", st.Pending.To)
	}
	f, ok := m.Files[update.Platform()]
	if !ok {
		return nil, Invalid("Version %s has no installer for this computer (%s).", m.Version, update.Platform())
	}
	c, _ := s.availableUpdate(ctx)
	s.upd.mu.Lock()
	if s.upd.state == "downloading" || s.upd.state == "installing" {
		s.upd.mu.Unlock()
		return nil, Invalid("An update is already in progress.")
	}
	s.upd.state, s.upd.message, s.upd.downloaded, s.upd.size = "downloading", "Downloading version "+m.Version, 0, f.Size
	s.upd.mu.Unlock()
	go s.runUpdate(context.Background(), a, m, f, []byte(c.Signed), install)
	return s.UpdateStatus(ctx)
}

func (s *Service) runUpdate(ctx context.Context, a Actor, m *update.Manifest, f update.File, signedFile []byte, install bool) {
	fail := func(msg string) {
		s.Log.Error("software update failed", "version", m.Version, "err", msg)
		s.upd.set("failed", msg)
		if install {
			s.recordUpdate(ctx, a, UpdateResult{From: brand.Version, To: m.Version, At: s.Now().UTC().Format(time.RFC3339), By: a.Username, Message: msg})
		}
	}
	defer func() {
		if r := recover(); r != nil {
			fail(fmt.Sprint("unexpected error: ", r))
		}
	}()
	dir := filepath.Join(s.Opts.DataDir, "updates", m.Version)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		fail("The update folder could not be created: " + err.Error())
		return
	}
	staged := update.Staged{Dir: dir, Path: filepath.Join(dir, f.FileName()), Manifest: m, File: f}
	if update.VerifyFile(staged.Path, f) != nil {
		dctx, cancel := context.WithTimeout(ctx, time.Hour)
		err := s.upd.client.Download(dctx, f, staged.Path, func(n int64) {
			s.upd.mu.Lock()
			s.upd.downloaded = n
			s.upd.mu.Unlock()
		})
		cancel()
		if err != nil {
			fail("The download failed: " + err.Error())
			return
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "update.json"), signedFile, 0o600); err != nil {
		fail("The update file could not be saved: " + err.Error())
		return
	}
	s.upd.mu.Lock()
	s.upd.downloaded, s.upd.ready = f.Size, &staged
	s.upd.mu.Unlock()
	if !install {
		s.upd.set("ready", "Version "+m.Version+" is downloaded and verified.")
		return
	}
	s.upd.set("installing", "Backing up the database")
	b, err := s.Backup(ctx, a, "pre-update")
	if err != nil {
		fail("The database backup before the update failed, so nothing was installed: " + err.Error())
		return
	}
	p := UpdateResult{From: brand.Version, To: m.Version, At: s.Now().UTC().Format(time.RFC3339), By: a.Username, Backup: b.Name, Notes: m.Notes}
	if err := s.setJSON(ctx, keyUpdPending, p); err != nil {
		fail(err.Error())
		return
	}
	s.Audit(ctx, a, 0, "system.update.start", "system", m.Version, map[string]any{"from": brand.Version, "to": m.Version, "backup": b.Name})
	s.upd.set("installing", "Installing version "+m.Version+". The software restarts by itself in about a minute.")
	time.Sleep(s.upd.launchDelay) // let the page show the step before the service stops
	if err := s.upd.launch(staged); err != nil {
		_ = s.Store.SetSetting(ctx, keyUpdPending, "")
		fail("The installer could not be started: " + err.Error())
		return
	}
	s.Log.Info("installer started", "version", m.Version)
}

func (s *Service) recordUpdate(ctx context.Context, a Actor, r UpdateResult) {
	_ = s.setJSON(ctx, keyUpdLast, r)
	action := "system.update.done"
	if !r.OK {
		action = "system.update.failed"
	}
	s.Audit(ctx, a, 0, action, "system", r.To, map[string]any{"from": r.From, "to": r.To, "message": r.Message})
}

// FinishUpdate runs at startup: on the version an update was installing,
// the update is recorded as complete and its files removed.
func (s *Service) FinishUpdate(ctx context.Context) {
	var p UpdateResult
	if !s.getJSON(ctx, keyUpdPending, &p) || p.To == "" {
		// An update recorded as failed may have completed after all (the
		// program was replaced, but the restart came later).
		if !s.getJSON(ctx, keyUpdLast, &p) || p.OK || p.To != brand.Version {
			return
		}
	}
	if update.CompareVersions(brand.Version, p.To) < 0 {
		s.reconcileUpdate(ctx)
		return
	}
	p.OK = true
	p.At = s.Now().UTC().Format(time.RFC3339)
	p.Message = fmt.Sprintf("Updated from version %s to %s.", p.From, brand.Version)
	_ = s.Store.SetSetting(ctx, keyUpdPending, "")
	s.recordUpdate(ctx, System, p)
	s.Log.Info("software updated", "from", p.From, "to", brand.Version)
	_ = os.RemoveAll(filepath.Join(s.Opts.DataDir, "updates"))
}

// reconcileUpdate records an update that did not complete: the Linux update
// helper reported a failure, or the old version still runs 15 minutes after
// the installer started (the installer starts the old service again when it
// fails).
func (s *Service) reconcileUpdate(ctx context.Context) {
	var p UpdateResult
	if !s.getJSON(ctx, keyUpdPending, &p) || p.To == "" || update.CompareVersions(brand.Version, p.To) >= 0 {
		return
	}
	reason := ""
	resultFile := filepath.Join(s.Opts.DataDir, "updates", update.ResultName)
	if b, err := os.ReadFile(resultFile); err == nil {
		var r struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(b, &r)
		reason = r.Message
		_ = os.Remove(resultFile)
	} else if t := store.ParseTime(p.At); !t.IsZero() && s.Now().Sub(t) > 15*time.Minute {
		reason = "the installer did not finish"
	} else if !t.IsZero() && s.upd.started.Truncate(time.Second).After(t) && !s.upd.installerRunning() {
		// This process started after the installer (in a later second: the
		// time is kept to the second), which has ended: it started the old
		// version again.
		reason = "the installer stopped before it had installed the new version"
	}
	if reason == "" {
		return
	}
	p.OK = false
	p.At = s.Now().UTC().Format(time.RFC3339)
	p.Message = fmt.Sprintf("The update to version %s did not complete (%s); version %s is still in use.", p.To, reason, brand.Version)
	if p.Backup != "" {
		p.Message += " The database was backed up first (" + p.Backup + ")."
	}
	_ = s.Store.SetSetting(ctx, keyUpdPending, "")
	s.recordUpdate(ctx, System, p)
	s.upd.set("failed", p.Message)
}

// WhatsNew returns the update installed last, for the "What's new" note.
func (s *Service) WhatsNew(ctx context.Context) *UpdateResult {
	var last UpdateResult
	if !s.getJSON(ctx, keyUpdLast, &last) || !last.OK || last.To != brand.Version {
		return nil
	}
	return &UpdateResult{From: last.From, To: last.To, At: last.At, OK: true, Notes: last.Notes}
}

// VerifiedUpdateFile returns the downloaded installer, checked again, so
// that an administrator can install it by hand.
func (s *Service) VerifiedUpdateFile() (string, error) {
	s.upd.mu.Lock()
	st := s.upd.ready
	s.upd.mu.Unlock()
	if st == nil {
		return "", Invalid("The update has not been downloaded yet.")
	}
	if err := update.VerifyFile(st.Path, st.File); err != nil {
		return "", err
	}
	return st.Path, nil
}

// updateTick runs in the background: it looks for updates every 6 hours
// (hourly around the update time) and installs them at the update time when
// the administrator chose automatic updates or the vendor marked the update
// as required.
func (s *Service) updateTick(ctx context.Context) {
	s.removeOldUpdateFiles(ctx)
	set := s.updateSettings(ctx)
	if pub, _ := update.PublicKey(); set.Mode == UpdateOff || pub == nil || s.updateFeed(ctx) == "" || s.upd.busy() {
		return
	}
	now := s.Now().In(PKT)
	s.upd.mu.Lock()
	last, started := s.upd.lastCheck, s.upd.started
	s.upd.mu.Unlock()
	if last.IsZero() {
		var c updateCheck
		if s.getJSON(ctx, keyUpdCheck, &c) {
			last = store.ParseTime(c.At)
		}
	}
	due := time.Since(last) >= 6*time.Hour || (now.Hour() == set.Hour && time.Since(last) >= 50*time.Minute)
	if due && time.Since(started) >= 2*time.Minute {
		if _, err := s.CheckForUpdate(ctx); err != nil {
			return
		}
	}
	st, err := s.UpdateStatus(ctx)
	if err != nil || st.Available == nil || !st.CanInstall || st.Blocked != "" || st.Pending != nil {
		return
	}
	if (set.Mode == UpdateAuto || st.Available.Mandatory) && now.Hour() == set.Hour {
		day := now.Format("2006-01-02")
		s.upd.mu.Lock()
		if s.upd.lastAuto == day {
			s.upd.mu.Unlock()
			return
		}
		s.upd.lastAuto = day
		s.upd.mu.Unlock()
		s.Log.Info("installing software update automatically", "version", st.Available.Version)
		if _, err := s.StartUpdate(ctx, System, true); err != nil {
			s.Log.Warn("automatic software update not started", "err", err)
		}
	}
}

// removeOldUpdateFiles deletes downloads of versions that are no longer
// newer than this one (the installer may still have been running when the
// new version first started).
func (s *Service) removeOldUpdateFiles(ctx context.Context) {
	var p UpdateResult
	if s.upd.busy() || (s.getJSON(ctx, keyUpdPending, &p) && p.To != "") {
		return
	}
	dir := filepath.Join(s.Opts.DataDir, "updates")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() && !update.Newer(e.Name(), brand.Version) {
			_ = os.RemoveAll(filepath.Join(dir, e.Name()))
		}
	}
}

// startUpdateTick runs updateTick at most once a minute, in the background.
func (s *Service) startUpdateTick(ctx context.Context) {
	s.upd.mu.Lock()
	if s.upd.ticking || time.Since(s.upd.lastTick) < time.Minute {
		s.upd.mu.Unlock()
		return
	}
	s.upd.ticking, s.upd.lastTick = true, time.Now()
	s.upd.mu.Unlock()
	go func() {
		defer func() {
			s.upd.mu.Lock()
			s.upd.ticking = false
			s.upd.mu.Unlock()
		}()
		s.updateTick(ctx)
	}()
}

// updateAlert tells administrators about a newer version.
func (s *Service) updateAlert(ctx context.Context) *Alert {
	if s.updateSettings(ctx).Mode == UpdateOff {
		return nil
	}
	_, m := s.availableUpdate(ctx)
	if m == nil {
		return nil
	}
	a := &Alert{ID: "update", Severity: "info", Title: "Version " + m.Version + " of the software is available", Link: "/settings/system"}
	if m.Mandatory {
		a.Severity, a.Title = "warning", "Required update: version "+m.Version
	}
	a.Detail = "See what's new and install it under Settings → System."
	if m.Released != "" {
		a.Detail = "Released " + dayDate(m.Released) + ". " + a.Detail
	}
	return a
}
