// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package db

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
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

func TestInspectAndRestore(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	d, err := Open(filepath.Join(dir, "live.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	now := Now()
	if _, err := d.ExecContext(ctx, `INSERT INTO companies(name, ntn_cnic, province, address, created_at, updated_at) VALUES('Before','0786909','SINDH','Karachi',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(dir, "backup.db")
	if err := Backup(ctx, d, backup); err != nil {
		t.Fatal(err)
	}
	if v, err := Inspect(ctx, backup); err != nil || v != LatestMigration() {
		t.Fatalf("inspect backup: %d %v (latest %d)", v, err, LatestMigration())
	}
	if _, err := d.ExecContext(ctx, `UPDATE companies SET name='After'`); err != nil {
		t.Fatal(err)
	}
	if _, err := d.ExecContext(ctx, `INSERT INTO companies(name, ntn_cnic, province, address, created_at, updated_at) VALUES('New','1234567','PUNJAB','Lahore',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if err := Restore(ctx, d, backup); err != nil {
		t.Fatal(err)
	}
	var names string
	if err := d.QueryRowContext(ctx, `SELECT group_concat(name) FROM companies`).Scan(&names); err != nil || names != "Before" {
		t.Fatalf("after restore: %q %v", names, err)
	}
	// The live database keeps working, and the backup file is unchanged.
	if _, err := d.ExecContext(ctx, `UPDATE companies SET name='Again'`); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(ctx, backup); err != nil {
		t.Fatal(err)
	}

	// Files that are not backups of this software.
	junk := filepath.Join(dir, "junk.db")
	if err := os.WriteFile(junk, []byte("this is not a database, just text that is long enough to be read"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Inspect(ctx, junk); err == nil {
		t.Fatal("text file accepted")
	}
	other := filepath.Join(dir, "other.db")
	o, err := sql.Open("sqlite", "file:"+filepath.ToSlash(other))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.Exec(`CREATE TABLE notes(x TEXT)`); err != nil {
		t.Fatal(err)
	}
	o.Close()
	if _, err := Inspect(ctx, other); err == nil || !strings.Contains(err.Error(), "not a backup") {
		t.Fatalf("other database: %v", err)
	}
}
