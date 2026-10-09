// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"einvoicing/internal/db"
	"einvoicing/internal/domain"
)

// WorkerOptions configures background processing.
type WorkerOptions struct {
	Interval        time.Duration // queue poll interval
	BackupHour      int           // local hour for the daily backup (-1 disables)
	BackupRetention int           // number of automatic backups kept
}

// RunWorker processes the submission queue, reconciles interrupted
// submissions, detects outages and takes daily backups until ctx ends.
func (s *Service) RunWorker(ctx context.Context, o WorkerOptions) {
	if o.Interval <= 0 {
		o.Interval = 20 * time.Second
	}
	// At startup nothing of this process is in flight, so every submission
	// still marked SUBMITTING was interrupted by the previous run.
	s.recoverStuck(ctx, s.Now().UTC().Add(time.Second))
	s.detectUnexpectedStop(ctx)
	defer s.MarkCleanShutdown()
	t := time.NewTicker(o.Interval)
	defer t.Stop()
	lastBackupDay := ""
	var lastIntegrity time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		s.heartbeat(ctx)
		if time.Since(lastIntegrity) >= 24*time.Hour {
			lastIntegrity = time.Now()
			if rep, err := s.CheckIntegrity(ctx); err != nil {
				s.Log.Error("integrity check failed", "err", err)
			} else if !rep.OK() {
				s.Log.Error("integrity check found problems", "auditBrokenAt", rep.AuditBrokenAt, "problems", len(rep.Problems))
			}
		}
		s.recoverStuck(ctx, s.Now().UTC().Add(-s.stuckAfter()))
		s.requeueRecovered(ctx)
		s.processQueue(ctx)
		s.checkIncidents(ctx)
		s.signPending(ctx)
		s.runClosings(ctx)
		// E-mails can take a while (slow mail server); never hold up the queue.
		go s.runNotifications(ctx)
		_ = s.Store.PurgeExpiredSessions(ctx)
		if o.BackupHour >= 0 {
			now := s.Now().In(PKT)
			day := now.Format("2006-01-02")
			if now.Hour() == o.BackupHour && day != lastBackupDay {
				lastBackupDay = day
				if _, err := s.Backup(ctx, System, "auto"); err != nil {
					s.Log.Error("automatic backup failed", "err", err)
				} else {
					s.pruneBackups(o.BackupRetention)
				}
			}
		}
	}
}

// stuckAfter is how long a submission may stay SUBMITTING before it is
// considered interrupted: validation and posting each take at most the FBR
// timeout, plus a margin.
func (s *Service) stuckAfter() time.Duration {
	t := s.Opts.HTTPTimeout
	if t <= 0 {
		t = 30 * time.Second
	}
	return 2*t + time.Minute
}

// recoverStuck releases submissions interrupted before their outcome was
// recorded: never sent -> queued again; possibly sent -> needs
// reconciliation against IRIS (never re-posted blindly).
func (s *Service) recoverStuck(ctx context.Context, before time.Time) {
	if n, err := s.Store.RecoverStuckSubmissions(ctx, before.Format(time.RFC3339)); err == nil && n > 0 {
		s.Log.Warn("interrupted submissions released", "count", n)
		s.Audit(ctx, System, 0, "worker.recovered", "invoice", "", fmt.Sprintf("%d interrupted submissions", n))
	}
}

func (s *Service) processQueue(ctx context.Context) {
	ids, err := s.Store.DueForSubmission(ctx, s.Now().UTC().Format(time.RFC3339), 25)
	if err != nil {
		s.Log.Error("queue scan failed", "err", err)
		return
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		inv, err := s.Store.GetInvoiceAnyCompany(ctx, id)
		if err != nil {
			continue
		}
		if _, err := s.Submit(ctx, System, inv.CompanyID, id, SubmitOptions{Background: true}); err != nil {
			s.Log.Warn("queued submission failed", "invoice", id, "err", err)
			// Refused before reaching FBR (e.g. the token was removed): try
			// again later instead of retrying on every tick, so this invoice
			// cannot hold up the rest of the queue.
			if cur, gerr := s.Store.GetInvoiceAnyCompany(ctx, id); gerr == nil && cur.Status == domain.StatusQueued {
				next := s.Now().UTC().Add(15 * time.Minute).Format(time.RFC3339)
				_ = s.Store.SetStatus(ctx, id, domain.StatusQueued, "Not submitted: "+err.Error(), next)
			}
		}
	}
}

// BackupInfo describes a backup file.
type BackupInfo struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	Created string `json:"created"`
}

// Backup writes a consistent copy of the database to the backup directory.
// The master key (needed to decrypt FBR tokens) is deliberately NOT included;
// keep a separate secure copy of master.key.
func (s *Service) Backup(ctx context.Context, a Actor, kind string) (*BackupInfo, error) {
	dir := s.Opts.BackupDir
	if dir == "" {
		dir = filepath.Join(s.Opts.DataDir, "backups")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("einvoice-%s-%s.db", kind, s.Now().In(PKT).Format("20060102-150405"))
	path := filepath.Join(dir, name)
	if err := db.Backup(ctx, s.Store.DB, path); err != nil {
		return nil, err
	}
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	s.Audit(ctx, a, 0, "system.backup", "backup", name, map[string]any{"size": st.Size()})
	return &BackupInfo{Name: name, Size: st.Size(), Created: st.ModTime().UTC().Format(time.RFC3339)}, nil
}

// ListBackups lists backup files, newest first.
func (s *Service) ListBackups() ([]BackupInfo, error) {
	dir := s.BackupDir()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	var out []BackupInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, BackupInfo{Name: e.Name(), Size: info.Size(), Created: info.ModTime().UTC().Format(time.RFC3339)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created > out[j].Created })
	return out, nil
}

// BackupDir returns the backup directory.
func (s *Service) BackupDir() string {
	if s.Opts.BackupDir != "" {
		return s.Opts.BackupDir
	}
	return filepath.Join(s.Opts.DataDir, "backups")
}

// BackupPath validates a backup name and returns its path.
func (s *Service) BackupPath(name string) (string, error) {
	if name != filepath.Base(name) || !strings.HasPrefix(name, "einvoice-") || !strings.HasSuffix(name, ".db") {
		return "", Invalid("invalid backup name")
	}
	p := filepath.Join(s.BackupDir(), name)
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

func (s *Service) pruneBackups(keep int) {
	if keep <= 0 {
		keep = 30
	}
	list, err := s.ListBackups()
	if err != nil {
		return
	}
	n := 0
	for _, b := range list {
		if !strings.HasPrefix(b.Name, "einvoice-auto-") {
			continue
		}
		n++
		if n > keep {
			_ = os.Remove(filepath.Join(s.BackupDir(), b.Name))
		}
	}
}
