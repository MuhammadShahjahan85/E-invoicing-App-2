// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

// Package config loads the installation configuration (config.json in the
// data directory), creating a default file on first run.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"einvoicing/internal/brand"
	"einvoicing/internal/fbr"
)

// Config is the on-disk configuration.
type Config struct {
	// Listen is the address the web server binds to. "0.0.0.0:8443" serves
	// the whole LAN; "127.0.0.1:8443" restricts access to this computer.
	Listen string `json:"listen"`
	// DataDir holds the database, master key, certificates, logs and backups.
	DataDir string `json:"-"`
	// BackupDir overrides <DataDir>/backups (e.g. a mapped network drive).
	BackupDir string `json:"backupDir"`
	TLS       TLS    `json:"tls"`
	FBR       FBR    `json:"fbr"`
	Worker    Worker `json:"worker"`
	LogLevel  string `json:"logLevel"`
}

// TLS configures HTTPS.
type TLS struct {
	Enabled bool `json:"enabled"`
	// CertFile/KeyFile: leave empty to use a certificate issued by the
	// installation's local certificate authority (<data>/tls/ca.pem).
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
	// Hosts lists extra DNS names or IP addresses the server is reached by
	// (e.g. "einvoice.office.local" or a VPN address) for the local certificate.
	Hosts []string `json:"hosts"`
}

// FBR configures the DI API.
type FBR struct {
	Endpoints      fbr.Endpoints `json:"endpoints"`
	TimeoutSeconds int           `json:"timeoutSeconds"`
	// CNICThreshold warns when an unregistered buyer without CNIC receives
	// an invoice above this value (empty disables).
	CNICThreshold string `json:"cnicThreshold"`
}

// Worker configures background processing.
type Worker struct {
	IntervalSeconds int `json:"intervalSeconds"`
	BackupHour      int `json:"backupHour"` // local hour 0-23, -1 disables automatic backups
	BackupRetention int `json:"backupRetention"`
}

// Default returns the default configuration.
func Default() Config {
	return Config{
		Listen:   "0.0.0.0:8443",
		TLS:      TLS{Enabled: true},
		FBR:      FBR{Endpoints: fbr.DefaultEndpoints(), TimeoutSeconds: 30, CNICThreshold: "100000"},
		Worker:   Worker{IntervalSeconds: 20, BackupHour: 23, BackupRetention: 30},
		LogLevel: "info",
	}
}

// DefaultDataDir picks a sensible data directory for the platform.
func DefaultDataDir() string {
	if d := os.Getenv("EINV_DATA_DIR"); d != "" {
		return d
	}
	if runtime.GOOS == "windows" {
		if pd := os.Getenv("ProgramData"); pd != "" {
			return filepath.Join(pd, brand.WindowsServiceName)
		}
	}
	exe, err := os.Executable()
	if err == nil {
		return filepath.Join(filepath.Dir(exe), "data")
	}
	return "data"
}

// Load reads <dataDir>/config.json, writing defaults when it is missing.
func Load(dataDir string) (Config, error) {
	cfg := Default()
	if err := os.MkdirAll(dataDir, 0o750); err != nil {
		return cfg, err
	}
	abs, err := filepath.Abs(dataDir)
	if err == nil {
		dataDir = abs
	}
	cfg.DataDir = dataDir
	path := filepath.Join(dataDir, "config.json")
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		out, _ := json.MarshalIndent(cfg, "", "  ")
		if err := os.WriteFile(path, out, 0o640); err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, errors.New("config.json is invalid: " + err.Error())
	}
	cfg.DataDir = dataDir
	cfg.FBR.Endpoints = cfg.FBR.Endpoints.Merge(fbr.DefaultEndpoints())
	if v := os.Getenv("EINV_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if strings.TrimSpace(cfg.Listen) == "" {
		cfg.Listen = Default().Listen
	}
	if cfg.FBR.TimeoutSeconds <= 0 {
		cfg.FBR.TimeoutSeconds = 30
	}
	return cfg, nil
}

// CACertPath is the local certificate authority created for HTTPS
// (<data>/tls/ca.pem). Office PCs and phones install it to trust the server.
func CACertPath(dataDir string) string { return filepath.Join(dataDir, "tls", "ca.pem") }
