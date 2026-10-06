// Package db opens the embedded SQLite database and applies migrations.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Querier is satisfied by *sql.DB and *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Open opens (creating if needed) the database file and migrates it.
// SQLite runs in WAL mode with a single connection, which serialises access
// and avoids "database is locked" errors for the small concurrent user
// counts of an on-premise installation.
func Open(path string) (*sql.DB, error) {
	dsn := dsnFor(path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)
	d.SetConnMaxLifetime(0)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := d.PingContext(ctx); err != nil {
		d.Close()
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	if err := Migrate(ctx, d); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func dsnFor(path string) string {
	if path == ":memory:" {
		return "file::memory:?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)"
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	q := url.Values{}
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "synchronous(NORMAL)")
	q.Add("_pragma", "busy_timeout(10000)")
	return "file:" + filepath.ToSlash(abs) + "?" + q.Encode()
}

// Migrate applies pending migrations in order, each in its own transaction.
func Migrate(ctx context.Context, d *sql.DB) error {
	if _, err := d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	applied := map[int]bool{}
	rows, err := d.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		applied[v] = true
	}
	rows.Close()

	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	type mig struct {
		version int
		name    string
	}
	var migs []mig
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		v, err := strconv.Atoi(strings.SplitN(name, "_", 2)[0])
		if err != nil {
			return fmt.Errorf("bad migration name %s", name)
		}
		migs = append(migs, mig{v, name})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })
	for _, m := range migs {
		if applied[m.version] {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + m.name)
		if err != nil {
			return err
		}
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", m.name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version, name, applied_at) VALUES(?,?,?)`, m.version, m.name, Now()); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Now returns the current UTC time in the storage format.
func Now() string { return time.Now().UTC().Format(time.RFC3339) }

// Backup writes a consistent snapshot of the database to dest using
// VACUUM INTO (safe while the application is running).
func Backup(ctx context.Context, d *sql.DB, dest string) error {
	_, err := d.ExecContext(ctx, `VACUUM INTO ?`, dest)
	return err
}

// InTx runs fn in a transaction.
func InTx(ctx context.Context, d *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
