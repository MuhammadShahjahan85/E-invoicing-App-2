//go:build !windows

// Package winsvc runs the application as a Windows service. On other
// platforms use systemd (see packaging/linux).
package winsvc

import (
	"context"
	"errors"
)

// Supported reports whether Windows service management is available.
const Supported = false

var errNotWindows = errors.New("Windows services are only available on Windows; on Linux use the systemd unit in packaging/linux")

// IsService always returns false off Windows.
func IsService() bool { return false }

// Run is not supported off Windows.
func Run(func(ctx context.Context) error) error { return errNotWindows }

// Install is not supported off Windows.
func Install(string) error { return errNotWindows }

// Uninstall is not supported off Windows.
func Uninstall() error { return errNotWindows }

// Start is not supported off Windows.
func Start() error { return errNotWindows }

// Stop is not supported off Windows.
func Stop() error { return errNotWindows }
