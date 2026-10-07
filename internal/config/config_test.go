// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package config

import (
	"os"
	"path/filepath"
	"testing"

	"einvoicing/internal/brand"
)

func TestProgramDataDirKeepsLegacyInstallation(t *testing.T) {
	pd := t.TempDir()
	cur := filepath.Join(pd, brand.WindowsServiceName)
	legacy := filepath.Join(pd, brand.LegacyWindowsServiceName)

	// Fresh computer: the new folder name.
	if got := programDataDir(pd); got != cur {
		t.Fatalf("fresh install: got %s, want %s", got, cur)
	}

	// Upgrade from a build released under the old name: keep its data.
	if err := os.MkdirAll(legacy, 0o750); err != nil {
		t.Fatal(err)
	}
	if got := programDataDir(pd); got != cur {
		t.Fatalf("empty legacy folder must not be used: got %s", got)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.json"), []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	if got := programDataDir(pd); got != legacy {
		t.Fatalf("upgrade: got %s, want %s", got, legacy)
	}

	// Once the new folder exists it always wins.
	if err := os.MkdirAll(cur, 0o750); err != nil {
		t.Fatal(err)
	}
	if got := programDataDir(pd); got != cur {
		t.Fatalf("both present: got %s, want %s", got, cur)
	}
}
