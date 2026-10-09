// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build !windows

// Package wintrust adds the local certificate authority to the Windows
// trusted root store. On other systems install <data>/tls/ca.pem with the
// system's own tools.
package wintrust

import "errors"

// Supported reports whether the trust store can be changed on this system.
const Supported = false

var errUnsupported = errors.New("only on Windows: install <data>/tls/ca.pem in the system trust store with its own tools")

// Trust is only available on Windows.
func Trust(der []byte) error { return errUnsupported }

// Untrust is only available on Windows.
func Untrust(der []byte) error { return errUnsupported }
