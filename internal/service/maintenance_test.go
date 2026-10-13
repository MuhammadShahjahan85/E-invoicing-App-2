// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"einvoicing/internal/store"
)

func TestRestoreBackup(t *testing.T) {
	f := setup(t)
	cust := f.customer(t)
	b, err := f.svc.Backup(f.ctx, f.admin, "manual")
	if err != nil {
		t.Fatal(err)
	}
	// Changes after the backup.
	cust.Name = "Renamed after the backup"
	if _, err := f.svc.SaveCustomer(f.ctx, f.admin, cust); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.SaveUser(f.ctx, f.admin, 0, UserInput{Username: "late", Role: store.RoleOperator, Password: "Late123456", AllCompanies: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.signingKey(f.ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := f.svc.RestoreBackup(f.ctx, f.admin, "../einvoice.db"); !IsValidation(err) {
		t.Fatalf("path outside the backup folder: %v", err)
	}
	safety, err := f.svc.RestoreBackup(f.ctx, f.admin, b.Name)
	if err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.Store.GetCustomer(f.ctx, f.cid, cust.ID)
	if err != nil || got.Name == "Renamed after the backup" {
		t.Fatalf("customer after restore: %+v %v", got, err)
	}
	if _, err := f.svc.Store.GetUserByUsername(f.ctx, "late"); err == nil {
		t.Fatal("user created after the backup still there")
	}
	if f.svc.signKey != nil {
		t.Fatal("signing key of the previous data kept")
	}
	// The data before the restore was saved, and can itself be restored.
	if !strings.HasPrefix(safety.Name, "einvoice-before-restore-") {
		t.Fatalf("safety backup: %s", safety.Name)
	}
	if _, err := f.svc.RestoreBackup(f.ctx, f.admin, safety.Name); err != nil {
		t.Fatal(err)
	}
	if got, _ = f.svc.Store.GetCustomer(f.ctx, f.cid, cust.ID); got.Name != "Renamed after the backup" {
		t.Fatalf("after restoring the safety backup: %q", got.Name)
	}
	entries, _, _ := f.svc.Store.ListAudit(f.ctx, store.AuditFilter{Action: "system.restore"})
	if len(entries) == 0 {
		t.Fatal("restore not in the audit trail")
	}
	if broken, _, err := f.svc.Store.VerifyAuditChain(f.ctx); err != nil || broken != 0 {
		t.Fatalf("audit chain after restores: %d %v", broken, err)
	}

	// Uploaded files are checked before they are kept.
	if _, err := f.svc.SaveUploadedBackup(f.ctx, f.admin, strings.NewReader("not a database")); !IsValidation(err) {
		t.Fatalf("junk upload: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(f.svc.BackupDir(), b.Name))
	if err != nil {
		t.Fatal(err)
	}
	up, err := f.svc.SaveUploadedBackup(f.ctx, f.admin, bytes.NewReader(raw))
	if err != nil || !strings.HasPrefix(up.Name, "einvoice-uploaded-") {
		t.Fatalf("upload: %+v %v", up, err)
	}
	list, _ := f.svc.ListBackups()
	for _, x := range list {
		if strings.HasSuffix(x.Name, ".part") {
			t.Fatalf("partial upload listed: %s", x.Name)
		}
	}
	if _, err := f.svc.RestoreBackup(f.ctx, f.admin, up.Name); err != nil {
		t.Fatal(err)
	}
}

func TestBackupSettings(t *testing.T) {
	f := setup(t)
	bs := f.svc.GetBackupSettings(f.ctx)
	if bs.Hour != 23 || bs.Retention != 30 || bs.Dir != "" || bs.DefaultDir == "" {
		t.Fatalf("defaults: %+v", bs)
	}
	for _, in := range []BackupSettings{{Hour: 24, Retention: 30}, {Hour: 2, Retention: 0}, {Hour: 2, Retention: 30, Dir: "relative/folder"}} {
		if _, err := f.svc.SaveBackupSettings(f.ctx, f.admin, in); !IsValidation(err) {
			t.Errorf("%+v accepted", in)
		}
	}
	other := filepath.Join(t.TempDir(), "elsewhere")
	bs, err := f.svc.SaveBackupSettings(f.ctx, f.admin, BackupSettings{Hour: -1, Retention: 7, Dir: other})
	if err != nil || bs.Hour != -1 || bs.Retention != 7 || bs.Dir != other {
		t.Fatalf("save: %+v %v", bs, err)
	}
	b, err := f.svc.Backup(f.ctx, f.admin, "manual")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(other, b.Name)); err != nil || f.svc.BackupDir() != other {
		t.Fatalf("backup not in the chosen folder: %v", err)
	}
	// Back to the default folder.
	if bs, err = f.svc.SaveBackupSettings(f.ctx, f.admin, BackupSettings{Hour: 1, Retention: 7, Dir: bs.DefaultDir}); err != nil || bs.Dir != "" {
		t.Fatalf("default folder: %+v %v", bs, err)
	}
}

func TestLogTail(t *testing.T) {
	f := setup(t)
	if text, err := f.svc.LogTail(10); err != nil || text != "" {
		t.Fatalf("no log yet: %q %v", text, err)
	}
	if err := os.MkdirAll(filepath.Dir(f.svc.LogPath()), 0o750); err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	for i := 1; i <= 50; i++ {
		b.WriteString("line ")
		b.WriteString(strings.Repeat("x", i%7))
		b.WriteString("\n")
	}
	b.WriteString("last line\n")
	if err := os.WriteFile(f.svc.LogPath(), []byte(b.String()), 0o640); err != nil {
		t.Fatal(err)
	}
	text, err := f.svc.LogTail(3)
	if err != nil || len(strings.Split(text, "\n")) != 3 || !strings.HasSuffix(text, "last line") {
		t.Fatalf("tail: %q %v", text, err)
	}
}
