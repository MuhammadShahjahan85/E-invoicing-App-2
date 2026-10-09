// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Command launcher opens Veridian E-invoicing Pakistan in a window of its
// own. The desktop and Start menu shortcuts run it (VeridianEInvoicing.exe):
// it starts the Windows service when it is not running, waits until the
// server answers and shows the app in Microsoft Edge or Google Chrome "app"
// mode, without tabs, an address bar or a command window.
//
//	VeridianEInvoicing.exe          open the dashboard
//	VeridianEInvoicing.exe help     open a page, e.g. the user guide
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/config"
)

// errNotInstalled means the background service is missing.
var errNotInstalled = errors.New("the " + brand.ProductName + " service is not installed")

func main() {
	if !singleInstance() {
		return // another launcher is already opening the window
	}
	dataDir := config.DefaultDataDir()
	cfg, err := config.Read(dataDir)
	if errors.Is(err, fs.ErrPermission) {
		cfg, err = config.Default(), nil // the defaults are right for most installations
	}
	if err != nil {
		fail(fmt.Sprintf("The settings file %s could not be read:\n\n%v\n\nAsk your administrator to correct it.", filepath.Join(dataDir, "config.json"), err))
		return
	}
	url, addr := address(cfg)
	if len(os.Args) > 1 {
		url += strings.TrimLeft(os.Args[1], "/")
	}
	if !answers(addr, 2*time.Second) {
		if err := startService(); err != nil {
			if errors.Is(err, errNotInstalled) {
				fail("The " + brand.ProductName + " service is not installed on this computer.\n\nRun the setup program again to repair the installation.")
			} else {
				fail("Windows could not start the " + brand.ProductName + " service:\n\n" + err.Error() + "\n\nStart it from Services (services.msc) or ask your administrator.")
			}
			return
		}
		if !waitFor(addr, 90*time.Second) {
			fail("The " + brand.ProductName + " service was started but is not answering at " + url + ".\n\nThe log files in " + filepath.Join(dataDir, "logs") + " show why. Contact support: " + brand.SupportContact)
			return
		}
	}
	if err := openWindow(url); err != nil {
		fail("The app could not be opened in a browser:\n\n" + err.Error() + "\n\nOpen " + url + " in Microsoft Edge or Google Chrome.")
	}
}

// address returns the URL to open and the host:port to check, from the
// server's listen address: a server listening on all addresses is opened
// as https://localhost:<port>/.
func address(cfg config.Config) (url, hostPort string) {
	host, port, err := net.SplitHostPort(cfg.Listen)
	if err != nil {
		host, port = "", "8443"
	}
	switch host {
	case "", "0.0.0.0", "::", "127.0.0.1", "::1", "localhost":
		host = "localhost"
	}
	// A certificate of the company's own is issued for its host name.
	if cfg.TLS.Enabled && cfg.TLS.CertFile != "" && len(cfg.TLS.Hosts) > 0 && host == "localhost" {
		host = cfg.TLS.Hosts[0]
	}
	scheme := "https"
	if !cfg.TLS.Enabled {
		scheme = "http"
	}
	hostPort = net.JoinHostPort(host, port)
	return scheme + "://" + hostPort + "/", hostPort
}

// answers reports whether the server accepts connections.
func answers(hostPort string, timeout time.Duration) bool {
	c, err := net.DialTimeout("tcp", hostPort, timeout)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// waitFor waits until the server accepts connections (the first start
// creates the database and certificates).
func waitFor(hostPort string, limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		if answers(hostPort, time.Second) {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}
