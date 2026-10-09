// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"einvoicing/internal/brand"
)

// firewallRuleNames are the rule of this product and the one created by
// builds released as "Veridian E-invoicing PK".
var firewallRuleNames = []string{brand.ProductName, "Veridian E-invoicing PK"}

// firewallAllow opens the port to computers on the local network only.
func firewallAllow(port string) error {
	_ = firewallRemove()
	return netsh(`add rule name="` + brand.ProductName + `" dir=in action=allow protocol=TCP localport=` + port +
		` remoteip=localsubnet description="Office computers on the local network use ` + brand.ProductName + `"`)
}

func firewallRemove() error {
	for _, n := range firewallRuleNames {
		_ = netsh(`delete rule name="` + n + `"`) // fails when the rule does not exist
	}
	return nil
}

// netsh runs "netsh advfirewall firewall <args>" with the command line as
// written: rule names contain spaces and netsh expects name="...".
func netsh(args string) error {
	exe := filepath.Join(os.Getenv("SystemRoot"), "System32", "netsh.exe")
	cmd := exec.Command(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: syscall.EscapeArg(exe) + " advfirewall firewall " + args, HideWindow: true}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
