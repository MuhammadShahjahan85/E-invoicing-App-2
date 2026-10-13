// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build !linux

package update

import "errors"

// Apply is the Linux update helper; elsewhere the installer does its work.
func Apply(string, func(format string, args ...any)) error {
	return errors.New("einvoice update apply is only used on Linux servers")
}
