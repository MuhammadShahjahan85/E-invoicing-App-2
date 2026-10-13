// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build linux

package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"einvoicing/internal/brand"
)

// HelperUnit is the systemd path unit that runs "einvoice update apply" as
// root when the service asks for an update (packaging/linux).
var HelperUnit = "/etc/systemd/system/einvoice-update.path"

// Capability reports whether this installation can install updates itself
// and, if not, how to install them.
func Capability(dataDir string) (bool, string) {
	for _, p := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(p); err == nil {
			return false, "The software runs in a container: install updates by pulling the new image and recreating the container (see the installation guide)."
		}
	}
	if os.Getenv("INVOCATION_ID") == "" {
		return false, "The software is not running as the einvoice system service, so it cannot restart itself: download the update below and install it with sudo sh install.sh ./einvoice."
	}
	unit, err := os.ReadFile(HelperUnit)
	if err != nil || !bytes.Contains(unit, []byte(filepath.Join(dataDir, "updates", RequestName))) {
		return false, "The update helper is not installed on this server. Install this update once by hand (download it below, then run sudo sh install.sh ./einvoice); later updates then install from the app."
	}
	return true, ""
}

// Launch asks the update helper to install the staged update.
func Launch(st Staged) error {
	if err := VerifyFile(st.Path, st.File); err != nil {
		return err
	}
	req, _ := json.Marshal(map[string]string{"version": st.Manifest.Version})
	return os.WriteFile(filepath.Join(filepath.Dir(st.Dir), RequestName), req, 0o600)
}

// Apply installs the update the service asked for. The update helper runs
// it as root; everything in the data folder is untrusted here, so the
// update file must carry the vendor's signature and the installer must
// match it. Only the program file is replaced; then the service restarts.
func Apply(dataDir string, logf func(format string, args ...any)) (err error) {
	dir := filepath.Join(dataDir, "updates")
	reqPath := filepath.Join(dir, RequestName)
	b, rerr := readUntrusted(reqPath, 4096)
	os.Remove(reqPath)
	if rerr != nil {
		return fmt.Errorf("no update was requested: %w", rerr)
	}
	defer func() {
		if err != nil {
			writeResult(dir, err)
		}
	}()
	var req struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(b, &req) != nil || !reVersion.MatchString(req.Version) {
		return errors.New("invalid update request")
	}
	pub, err := PublicKey()
	if err != nil {
		return err
	}
	vdir := filepath.Join(dir, req.Version)
	mb, err := readUntrusted(filepath.Join(vdir, "update.json"), MaxManifestSize)
	if err != nil {
		return err
	}
	m, err := Parse(mb, pub)
	if err != nil {
		return err
	}
	if m.Version != req.Version {
		return errors.New("the update file is for another version")
	}
	if !Newer(m.Version, brand.Version) {
		return fmt.Errorf("version %s is not newer than the installed version %s", m.Version, brand.Version)
	}
	f, ok := m.Files[Platform()]
	if !ok {
		return fmt.Errorf("the update has no installer for %s", Platform())
	}
	data, err := readUntrusted(filepath.Join(vdir, f.FileName()), f.Size)
	if err != nil {
		return err
	}
	if err := verifyReader(bytes.NewReader(data), f); err != nil {
		return err
	}
	bin, err := programFromArchive(data)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return err
	}
	tmp := filepath.Join(filepath.Dir(exe), ".einvoice.new")
	os.Remove(tmp)
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0o755)
	if err != nil {
		return err
	}
	_, err = out.Write(bin)
	if err == nil {
		err = out.Sync()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, exe)
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	logf("installed version %s (was %s); restarting the service", m.Version, brand.Version)
	if out, err := exec.Command("systemctl", "restart", "--no-block", "einvoice.service").CombinedOutput(); err != nil {
		return fmt.Errorf("version %s was installed, but the service could not be restarted (%v %s); restart it with: sudo systemctl restart einvoice",
			m.Version, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// readUntrusted reads a regular file (not a symbolic link) of at most max bytes.
func readUntrusted(path string, max int64) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", filepath.Base(path))
	}
	b, err := io.ReadAll(io.LimitReader(f, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > max {
		return nil, fmt.Errorf("%s is too large", filepath.Base(path))
	}
	return b, nil
}

// programFromArchive returns the einvoice program from a release archive
// (einvoice-<version>-linux-amd64.tar.gz).
func programFromArchive(data []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, errors.New("the update archive does not contain the einvoice program")
		}
		if err != nil {
			return nil, err
		}
		name := strings.TrimPrefix(h.Name, "./")
		if h.Typeflag != tar.TypeReg || filepath.Base(name) != "einvoice" || strings.Count(name, "/") > 1 {
			continue
		}
		if h.Size <= 0 || h.Size > MaxFileSize {
			return nil, errors.New("the einvoice program in the update archive has an invalid size")
		}
		return io.ReadAll(io.LimitReader(tr, h.Size))
	}
}

// writeResult reports a failed installation to the service, which runs as
// another user: the file is created afresh (never through a link) and given
// to the owner of the updates folder.
func writeResult(dir string, failure error) {
	p := filepath.Join(dir, ResultName)
	os.Remove(p)
	b, _ := json.Marshal(map[string]any{"ok": false, "message": failure.Error(), "at": time.Now().UTC().Format(time.RFC3339)})
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0o640)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(b)
	if st, err := os.Stat(dir); err == nil {
		if sys, ok := st.Sys().(*syscall.Stat_t); ok {
			_ = f.Chown(int(sys.Uid), int(sys.Gid))
		}
	}
}

// InstallerRunning reports whether an installer is running (Windows only).
func InstallerRunning() bool { return false }
