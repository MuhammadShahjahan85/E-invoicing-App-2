// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build !windows

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func singleInstance() bool { return true }

// startService is not available here: on Linux the server runs under
// systemd (see packaging/linux).
func startService() error {
	return errors.New("the server is not running; start it with: sudo systemctl start einvoice")
}

// openWindow opens the app in the default browser.
func openWindow(url string) error {
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	}
	return exec.Command(name, url).Start()
}

func fail(msg string) { fmt.Fprintln(os.Stderr, msg) }
