// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"errors"
	"strings"
	"testing"

	"einvoicing/internal/store"
)

func TestDeleteUser(t *testing.T) {
	f := setup(t)
	clerk, err := f.svc.SaveUser(f.ctx, f.admin, 0, UserInput{Username: "clerk", FullName: "Counter clerk", Role: store.RoleOperator,
		Password: "Clerk12345", AllCompanies: false, CompanyIDs: []int64{f.cid}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Login(f.ctx, "clerk", "Clerk12345", "127.0.0.1", "test"); err != nil {
		t.Fatal(err)
	}

	// Nobody can delete their own account.
	if err := f.svc.DeleteUser(f.ctx, f.admin, *f.admin.UserID); !IsValidation(err) {
		t.Fatalf("self delete: %v", err)
	}

	if err := f.svc.DeleteUser(f.ctx, f.admin, clerk.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Store.GetUser(f.ctx, clerk.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("deleted user still there: %v", err)
	}
	if _, err := f.svc.Login(f.ctx, "clerk", "Clerk12345", "127.0.0.1", "test"); err == nil {
		t.Fatal("a deleted user could log in")
	}
	var sessions int
	if err := f.svc.Store.DB.QueryRowContext(f.ctx, `SELECT COUNT(*) FROM sessions WHERE user_id=?`, clerk.ID).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatalf("sessions left: %d %v", sessions, err)
	}
	if err := f.svc.DeleteUser(f.ctx, f.admin, clerk.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
	entries, _, err := f.svc.Store.ListAudit(f.ctx, store.AuditFilter{Action: "user.delete"})
	if err != nil || len(entries) != 1 || !strings.Contains(entries[0].Details, `"clerk"`) || entries[0].Username != "admin" {
		t.Fatalf("audit: %v %+v", err, entries)
	}
	if broken, _, err := f.svc.Store.VerifyAuditChain(f.ctx); err != nil || broken != 0 {
		t.Fatalf("audit chain broken at %d: %v", broken, err)
	}

	// The last active administrator cannot be deleted, by anyone.
	cli := Actor{Username: "cli", Role: store.RoleAdmin}
	if err := f.svc.DeleteUser(f.ctx, cli, *f.admin.UserID); !IsValidation(err) {
		t.Fatalf("last admin deleted: %v", err)
	}
	// With a second administrator, either can delete the other.
	admin2, err := f.svc.SaveUser(f.ctx, f.admin, 0, UserInput{Username: "admin2", Role: store.RoleAdmin, Password: "Admin212345", AllCompanies: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeleteUser(f.ctx, Actor{UserID: &admin2.ID, Username: "admin2", Role: store.RoleAdmin}, *f.admin.UserID); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeleteUser(f.ctx, cli, admin2.ID); !IsValidation(err) {
		t.Fatalf("new last admin deleted: %v", err)
	}
	if n, _ := f.svc.Store.CountUsers(f.ctx); n != 1 {
		t.Fatalf("users left: %d", n)
	}
}
