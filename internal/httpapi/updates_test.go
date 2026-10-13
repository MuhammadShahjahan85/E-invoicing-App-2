// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

// Only administrators see and control software updates; everyone gets the
// "What's new" note.
func TestSoftwareUpdateAPI(t *testing.T) {
	e := newEnv(t)
	op := e.login("operatorA", "Initial123")
	for _, c := range []struct{ method, path string }{
		{"GET", "/system/updates"}, {"POST", "/system/updates/check"}, {"POST", "/system/updates/start"},
		{"PUT", "/system/updates/settings"}, {"GET", "/system/updates/file"},
	} {
		if st, _ := op.do(c.method, c.path, map[string]any{}); st != 403 {
			t.Errorf("operator %s %s: got %d, want 403", c.method, c.path, st)
		}
	}
	if st, b := op.do("GET", "/system/whats-new", nil); st != 403 && st != 200 {
		t.Errorf("what's new for an operator: %d %s", st, b)
	}

	st, b := e.adminA.do("GET", "/system/updates", nil)
	var us struct {
		Current    string `json:"current"`
		Configured bool   `json:"configured"`
		State      string `json:"state"`
		Settings   struct {
			Mode string `json:"mode"`
			Hour int    `json:"hour"`
		} `json:"settings"`
	}
	if st != 200 || json.Unmarshal(b, &us) != nil || us.Current == "" || us.Configured || us.State != "idle" || us.Settings.Mode != "notify" || us.Settings.Hour != 2 {
		t.Fatalf("status: %d %s", st, b)
	}
	// This test build has no update key: checking explains why.
	if st, b := e.adminA.do("POST", "/system/updates/check", nil); st != 422 || !strings.Contains(string(b), "not set up") {
		t.Fatalf("check without a key: %d %s", st, b)
	}
	if st, b := e.adminA.do("POST", "/system/updates/start", map[string]any{"install": true}); st != 422 || !strings.Contains(string(b), "No newer version") {
		t.Fatalf("install without an update: %d %s", st, b)
	}
	if st, _ := e.adminA.do("GET", "/system/updates/file", nil); st != 422 {
		t.Fatalf("file before a download: %d", st)
	}
	if st, b := e.adminA.do("PUT", "/system/updates/settings", map[string]any{"mode": "auto", "hour": 23, "feed": ""}); st != 200 || !strings.Contains(string(b), `"mode":"auto"`) {
		t.Fatalf("save settings: %d %s", st, b)
	}
	if st, _ := e.adminA.do("PUT", "/system/updates/settings", map[string]any{"mode": "auto", "hour": 2, "feed": "http://example.pk/u.json"}); st != 422 {
		t.Fatalf("plain http update address: %d", st)
	}
	if st, b := e.adminA.do("GET", "/system/whats-new", nil); st != 200 || !strings.Contains(string(b), `"update":null`) {
		t.Fatalf("what's new: %d %s", st, b)
	}
}
