// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

// Package app wires configuration, storage, services and the web server.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/config"
	"einvoicing/internal/db"
	"einvoicing/internal/httpapi"
	"einvoicing/internal/license"
	"einvoicing/internal/security"
	"einvoicing/internal/service"
	"einvoicing/internal/store"
	"einvoicing/internal/webui"
)

// App is a running instance.
type App struct {
	Cfg     config.Config
	Log     *slog.Logger
	Svc     *service.Service
	License *license.Manager
	closeDB func() error
	logFile io.Closer
}

// Open initialises everything except the HTTP listener.
func Open(dataDir string, logToStdout bool) (*App, error) {
	cfg, err := config.Load(dataDir)
	if err != nil {
		return nil, err
	}
	logDir := filepath.Join(cfg.DataDir, "logs")
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return nil, err
	}
	lf, err := os.OpenFile(filepath.Join(logDir, "einvoice.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	var w io.Writer = lf
	if logToStdout {
		w = io.MultiWriter(lf, os.Stdout)
	}
	level := slog.LevelInfo
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	log := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: level}))

	d, err := db.Open(filepath.Join(cfg.DataDir, "einvoice.db"))
	if err != nil {
		lf.Close()
		return nil, err
	}
	vault, err := security.OpenVault(cfg.DataDir)
	if err != nil {
		d.Close()
		lf.Close()
		return nil, err
	}
	st := store.New(d)
	lic, err := license.NewManager(st)
	if err != nil {
		d.Close()
		lf.Close()
		return nil, err
	}
	svc := service.New(st, vault, service.Options{
		Endpoints:     cfg.FBR.Endpoints,
		HTTPTimeout:   time.Duration(cfg.FBR.TimeoutSeconds) * time.Second,
		DataDir:       cfg.DataDir,
		BackupDir:     cfg.BackupDir,
		CNICThreshold: cfg.FBR.CNICThreshold,
		License:       lic,
		Logger:        log,
	})
	if err := svc.EnsureSeedData(context.Background()); err != nil {
		log.Warn("seed data", "err", err)
	}
	return &App{Cfg: cfg, Log: log, Svc: svc, License: lic, closeDB: d.Close, logFile: lf}, nil
}

// Close releases resources.
func (a *App) Close() {
	if a.closeDB != nil {
		_ = a.closeDB()
	}
	if a.logFile != nil {
		_ = a.logFile.Close()
	}
}

// Serve runs the web server and background worker until ctx is cancelled.
func (a *App) Serve(ctx context.Context) error {
	ui, err := webui.Dist()
	if err != nil {
		a.Log.Warn("web UI not embedded", "err", err)
	}
	api := httpapi.New(a.Svc, a.License, ui, a.Log, a.Cfg.TLS.Enabled)
	srv := &http.Server{
		Addr:              a.Cfg.Listen,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 15 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}
	wctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go a.Svc.RunWorker(wctx, service.WorkerOptions{
		Interval:        time.Duration(a.Cfg.Worker.IntervalSeconds) * time.Second,
		BackupHour:      a.Cfg.Worker.BackupHour,
		BackupRetention: a.Cfg.Worker.BackupRetention,
	})

	errc := make(chan error, 1)
	scheme := "http"
	if a.Cfg.TLS.Enabled {
		scheme = "https"
	}
	go func() {
		if a.Cfg.TLS.Enabled {
			cert, key := a.Cfg.TLS.CertFile, a.Cfg.TLS.KeyFile
			if cert == "" || key == "" {
				var err error
				if cert, key, err = EnsureCertificates(a.Cfg.DataDir, a.Cfg.TLS.Hosts); err != nil {
					errc <- fmt.Errorf("TLS certificate: %w", err)
					return
				}
			}
			errc <- srv.ListenAndServeTLS(cert, key)
			return
		}
		errc <- srv.ListenAndServe()
	}()
	a.Log.Info("started", "product", brand.ProductName, "version", brand.Version, "listen", a.Cfg.Listen, "tls", a.Cfg.TLS.Enabled, "data", a.Cfg.DataDir)
	fmt.Printf("%s %s listening on %s://%s (data: %s)\n", brand.ProductName, brand.Version, scheme, displayAddr(a.Cfg.Listen), a.Cfg.DataDir)

	select {
	case <-ctx.Done():
		sctx, c2 := context.WithTimeout(context.Background(), 20*time.Second)
		defer c2()
		_ = srv.Shutdown(sctx)
		a.Log.Info("stopped")
		return nil
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func displayAddr(listen string) string {
	if strings.HasPrefix(listen, "0.0.0.0:") || strings.HasPrefix(listen, ":") {
		return "localhost:" + listen[strings.LastIndex(listen, ":")+1:]
	}
	return listen
}
