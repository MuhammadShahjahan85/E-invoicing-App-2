// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build !windows

package main

import "errors"

func firewallAllow(port string) error {
	return errors.New("only on Windows: open TCP port " + port + " with the system firewall (e.g. ufw allow " + port + "/tcp)")
}

func firewallRemove() error { return errors.New("only on Windows") }
