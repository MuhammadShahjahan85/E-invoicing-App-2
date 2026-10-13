// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Administrators restore backups (after typing RESTORE), change the backup
// schedule and read the server log; nobody else can.
func TestRestoreAndMaintenanceAPI(t *testing.T) {
	e := newEnv(t)
	st, b := e.adminA.do("POST", "/system/backups", nil)
	var bk struct {
		Name string `json:"name"`
	}
	if st != 201 || json.Unmarshal(b, &bk) != nil || bk.Name == "" {
		t.Fatalf("backup: %d %s", st, b)
	}
	if st, b := e.adminA.do("POST", fmt.Sprintf("/companies/%d/customers", e.compA),
		map[string]any{"name": "Added After Backup", "registrationType": "Unregistered", "province": "Punjab"}); st != 201 {
		t.Fatalf("customer: %d %s", st, b)
	}

	op := e.login("operatorA", "Initial123")
	for _, c := range []struct{ method, path string }{
		{"POST", "/system/backups/" + bk.Name + "/restore"}, {"POST", "/system/backups/upload"},
		{"GET", "/system/backups/settings"}, {"PUT", "/system/backups/settings"}, {"GET", "/system/logs"}, {"GET", "/system/logs/download"},
	} {
		if st, _ := op.do(c.method, c.path, map[string]any{"confirm": "RESTORE"}); st != 403 {
			t.Errorf("operator %s %s: got %d, want 403", c.method, c.path, st)
		}
	}

	if st, b := e.adminA.do("POST", "/system/backups/"+bk.Name+"/restore", map[string]any{}); st != 422 || !strings.Contains(string(b), "RESTORE") {
		t.Fatalf("restore without confirmation: %d %s", st, b)
	}
	st, b = e.adminA.do("POST", "/system/backups/"+bk.Name+"/restore", map[string]any{"confirm": "RESTORE"})
	if st != 200 || !strings.Contains(string(b), "einvoice-before-restore-") {
		t.Fatalf("restore: %d %s", st, b)
	}
	if st, b := e.adminA.do("GET", fmt.Sprintf("/companies/%d/customers", e.compA), nil); st != 200 || strings.Contains(string(b), "Added After Backup") {
		t.Fatalf("customers after restore: %d %s", st, b)
	}

	if st, b := e.adminA.do("GET", "/system/backups/settings", nil); st != 200 || !strings.Contains(string(b), `"retention":30`) {
		t.Fatalf("backup settings: %d %s", st, b)
	}
	if st, _ := e.adminA.do("PUT", "/system/backups/settings", map[string]any{"hour": 25, "retention": 30}); st != 422 {
		t.Fatalf("invalid backup hour: %d", st)
	}
	if st, b := e.adminA.do("PUT", "/system/backups/settings", map[string]any{"hour": 1, "retention": 14, "dir": ""}); st != 200 || !strings.Contains(string(b), `"hour":1`) {
		t.Fatalf("save backup settings: %d %s", st, b)
	}
	if st, b := e.adminA.do("GET", "/system/logs?lines=50", nil); st != 200 || !strings.Contains(string(b), `"path"`) {
		t.Fatalf("log: %d %s", st, b)
	}
	if st, b := e.adminA.do("GET", "/meta", nil); st != 200 || !strings.Contains(string(b), `"userGuide":false`) {
		t.Fatalf("meta: %d", st)
	}
	if st, _ := e.adminA.do("GET", "/help/guide", nil); st != 404 {
		t.Fatalf("user guide without the file: %d", st)
	}
}
