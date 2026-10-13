// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/db"
)

// Maintenance an administrator does from the System page: restoring a
// backup, backup settings, the server log and the user guide.

const keyBackupSettings = "backup.settings"

// BackupSettings are when and where automatic backups are made. They
// override the backupHour, backupRetention and backupDir of config.json.
type BackupSettings struct {
	Hour       int    `json:"hour"`      // Pakistan time, 0-23; -1 = no automatic backups
	Retention  int    `json:"retention"` // automatic backups kept
	Dir        string `json:"dir"`       // empty = the default folder
	DefaultDir string `json:"defaultDir"`
}

func (s *Service) defaultBackupDir() string {
	if s.Opts.BackupDir != "" {
		return s.Opts.BackupDir
	}
	return filepath.Join(s.Opts.DataDir, "backups")
}

// GetBackupSettings returns the backup schedule and folder in use.
func (s *Service) GetBackupSettings(ctx context.Context) BackupSettings {
	b := BackupSettings{Hour: int(s.backupHour.Load()), Retention: int(s.backupKeep.Load())}
	s.getJSON(ctx, keyBackupSettings, &b)
	if b.Retention <= 0 {
		b.Retention = 30
	}
	b.DefaultDir = s.defaultBackupDir()
	return b
}

// SaveBackupSettings changes when and where automatic backups are made.
func (s *Service) SaveBackupSettings(ctx context.Context, a Actor, in BackupSettings) (BackupSettings, error) {
	if in.Hour < -1 || in.Hour > 23 {
		return BackupSettings{}, Invalid("the backup hour must be between 0 and 23, or off")
	}
	if in.Retention < 1 || in.Retention > 3650 {
		return BackupSettings{}, Invalid("keep between 1 and 3650 automatic backups")
	}
	in.Dir = strings.TrimSpace(in.Dir)
	if in.Dir == s.defaultBackupDir() {
		in.Dir = ""
	}
	if in.Dir != "" {
		if !filepath.IsAbs(in.Dir) {
			return BackupSettings{}, Invalid(`the backup folder must be a full path, e.g. D:\Backups\E-invoicing or \\server\share\E-invoicing`)
		}
		if err := writable(in.Dir); err != nil {
			return BackupSettings{}, Invalid("the software cannot write to %s: %v", in.Dir, err)
		}
	}
	in.DefaultDir = ""
	if err := s.setJSON(ctx, keyBackupSettings, in); err != nil {
		return BackupSettings{}, err
	}
	s.Audit(ctx, a, 0, "system.backup_settings", "system", "backups", in)
	return s.GetBackupSettings(ctx), nil
}

// writable checks that the service can create files in dir.
func writable(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".write-test-*")
	if err != nil {
		return err
	}
	name := f.Name()
	f.Close()
	return os.Remove(name)
}

// RestoreBackup replaces all data with that of a backup. The current data
// is backed up first; the backup made is returned. Everybody has to sign in
// again, as the sign-ins are those of the backup.
func (s *Service) RestoreBackup(ctx context.Context, a Actor, name string) (*BackupInfo, error) {
	path, err := s.BackupPath(name)
	if err != nil {
		return nil, err
	}
	v, err := db.Inspect(ctx, path)
	if err != nil {
		return nil, Invalid("%s cannot be restored: %v", name, err)
	}
	if v > db.LatestMigration() {
		return nil, Invalid("%s was made by a newer version of the software; update the software first", name)
	}
	s.restoreMu.Lock()
	defer s.restoreMu.Unlock()
	safety, err := s.Backup(ctx, a, "before-restore")
	if err != nil {
		return nil, fmt.Errorf("the current data could not be backed up first, so nothing was restored: %w", err)
	}
	if err := db.Restore(ctx, s.Store.DB, path); err != nil {
		return nil, fmt.Errorf("the backup could not be restored (%w); the data before the attempt is in %s", err, safety.Name)
	}
	// Forget what was read from the previous data.
	s.signMu.Lock()
	s.signKey = nil
	s.signMu.Unlock()
	if err := s.EnsureSeedData(ctx); err != nil {
		s.Log.Warn("reference data after restore", "err", err)
	}
	s.Audit(ctx, a, 0, "system.restore", "backup", name, map[string]any{"safetyBackup": safety.Name})
	s.Log.Warn("database restored from a backup", "backup", name, "safetyBackup", safety.Name, "by", a.Username)
	return safety, nil
}

// SaveUploadedBackup checks an uploaded backup file and stores it in the
// backup folder, from where it can be restored.
func (s *Service) SaveUploadedBackup(ctx context.Context, a Actor, r io.Reader) (*BackupInfo, error) {
	dir := s.BackupDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	name := uniqueName(dir, "einvoice-uploaded-"+s.Now().In(PKT).Format("20060102-150405"), ".db")
	path := filepath.Join(dir, name)
	tmp := path + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return nil, fmt.Errorf("the upload was interrupted: %w", err)
	}
	if _, err := db.Inspect(ctx, tmp); err != nil {
		os.Remove(tmp)
		return nil, Invalid("the file cannot be used: %v", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return nil, err
	}
	s.Audit(ctx, a, 0, "system.backup_upload", "backup", name, map[string]any{"size": n})
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return &BackupInfo{Name: name, Size: st.Size(), Created: st.ModTime().UTC().Format(time.RFC3339)}, nil
}

// uniqueName returns base+ext, or base-2+ext and so on when taken.
func uniqueName(dir, base, ext string) string {
	name := base + ext
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			return name
		}
		name = fmt.Sprintf("%s-%d%s", base, i, ext)
	}
}

// LogPath is the server's log file.
func (s *Service) LogPath() string {
	return filepath.Join(s.Opts.DataDir, "logs", "einvoice.log")
}

// LogTail returns up to the last n lines of the server log.
func (s *Service) LogTail(n int) (string, error) {
	if n <= 0 || n > 5000 {
		n = 300
	}
	f, err := os.Open(s.LogPath())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	const window = 1 << 20
	start := st.Size() - window
	if start < 0 {
		start = 0
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", err
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	if start > 0 {
		// Drop the partial first line.
		if i := bytes.IndexByte(b, '\n'); i >= 0 {
			b = b[i+1:]
		}
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n"), nil
}

// UserGuidePath finds the printable user guide installed with the program
// (in its docs folder); "" when it is not there.
func UserGuidePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	p := filepath.Join(filepath.Dir(exe), "docs", brand.UserGuideFile)
	if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
		return p
	}
	return ""
}
