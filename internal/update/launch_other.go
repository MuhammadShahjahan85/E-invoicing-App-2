// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build !windows && !linux

package update

import "errors"

// Capability reports whether this installation can install updates itself
// and, if not, how to install them.
func Capability(string) (bool, string) {
	return false, "Updates cannot be installed automatically on this operating system: download the update below and install it by hand."
}

// Launch is not supported on this operating system.
func Launch(Staged) error {
	return errors.New("updates cannot be installed automatically on this operating system")
}

// InstallerRunning reports whether an installer is running (Windows only).
func InstallerRunning() bool { return false }
