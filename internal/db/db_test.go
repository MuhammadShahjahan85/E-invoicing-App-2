// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigrateAndTriggers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	now := Now()
	mustExec := func(q string, args ...any) {
		t.Helper()
		if _, err := d.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	mustExec(`INSERT INTO companies(name, ntn_cnic, province, address, created_at, updated_at) VALUES('A','0786909','SINDH','Karachi',?,?)`, now, now)
	mustExec(`INSERT INTO invoices(company_id, environment, doc_type, internal_no, invoice_date, status, seller_ntn_cnic, seller_name, seller_province, seller_address, buyer_name, buyer_province, buyer_registration_type, created_at, updated_at)
	          VALUES(1,'production','Sale Invoice','INV-1','2026-10-06','DRAFT','0786909','A','SINDH','Karachi','B','PUNJAB','Unregistered',?,?)`, now, now)
	mustExec(`INSERT INTO invoice_items(invoice_id, line_no, hs_code, description, uom, quantity, unit_price, sale_type, rate) VALUES(1,1,'0101.2100','x','KG','1','100','Goods at standard rate (default)','18%')`)
	// Draft is editable.
	mustExec(`UPDATE invoices SET buyer_name='C' WHERE id=1`)
	// Accept it.
	mustExec(`UPDATE invoices SET status='ACCEPTED', fbr_invoice_number='0786909DI1' WHERE id=1`)

	_, err = d.ExecContext(ctx, `UPDATE invoices SET total_sales_tax=1 WHERE id=1`)
	if err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("expected immutability error, got %v", err)
	}
	_, err = d.ExecContext(ctx, `UPDATE invoice_items SET quantity='2' WHERE invoice_id=1`)
	if err == nil {
		t.Fatal("expected item immutability error")
	}
	_, err = d.ExecContext(ctx, `DELETE FROM invoices WHERE id=1`)
	if err == nil {
		t.Fatal("expected delete protection")
	}
	_, err = d.ExecContext(ctx, `UPDATE invoices SET status='DRAFT' WHERE id=1`)
	if err == nil {
		t.Fatal("accepted invoice must not return to draft")
	}
	// Allowed changes.
	mustExec(`UPDATE invoices SET print_count=print_count+1 WHERE id=1`)
	mustExec(`UPDATE invoices SET status='CANCELLED', cancelled_at=?, cancel_reason='error' WHERE id=1`, now)

	mustExec(`INSERT INTO audit_log(ts, action, prev_hash, hash) VALUES(?, 'x', '', 'h')`, now)
	if _, err := d.ExecContext(ctx, `DELETE FROM audit_log`); err == nil {
		t.Fatal("audit log must be append-only")
	}

	// Re-opening must not re-apply migrations.
	d.Close()
	d2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	var n int
	files, _ := fs.ReadDir(migrationsFS, "migrations")
	if err := d2.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil || n != len(files) || n < 2 {
		t.Fatalf("migrations count %d of %d %v", n, len(files), err)
	}
	// Upgrades add columns without touching the immutability guarantees.
	var offline string
	if err := d2.QueryRow(`SELECT offline_since FROM invoices LIMIT 1`).Scan(&offline); err != nil && err != sql.ErrNoRows {
		t.Fatalf("offline_since column: %v", err)
	}
	if err := Backup(ctx, d2, filepath.Join(t.TempDir(), "b.db")); err != nil {
		t.Fatalf("backup: %v", err)
	}
}
