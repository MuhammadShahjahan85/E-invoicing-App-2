// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build windows

package update

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"

	"einvoicing/internal/brand"
)

// Capability reports whether this installation can install updates itself
// and, if not, how to install them.
func Capability(string) (bool, string) {
	if ok, err := svc.IsWindowsService(); err == nil && ok {
		return true, ""
	}
	return false, "The software is not running as its Windows service, so it cannot restart itself: download the installer below and run it."
}

// Launch starts the installer in silent update mode, detached from this
// process. The installer stops the service, replaces the program and starts
// the service again.
func Launch(st Staged) error {
	// Hold the installer open, denying writes and deletion, from the final
	// check until it has started.
	p, err := windows.UTF16PtrFromString(st.Path)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return err
	}
	f := os.NewFile(uintptr(h), st.Path)
	defer f.Close()
	if err := verifyReader(f, st.File); err != nil {
		return err
	}
	flags := uint32(windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS)
	cmd := exec.Command(st.Path, "/S", "/UPDATE")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: flags | windows.CREATE_BREAKAWAY_FROM_JOB}
	if err := cmd.Start(); err != nil {
		// The service's job may not allow leaving it: start inside it.
		cmd = exec.Command(st.Path, "/S", "/UPDATE")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: flags}
		if err := cmd.Start(); err != nil {
			return err
		}
	}
	return cmd.Process.Release()
}

// InstallerRunning reports whether the installer is running: it holds a
// named mutex (see .onInit in packaging/windows/installer.nsi).
func InstallerRunning() bool {
	name, err := windows.UTF16PtrFromString(brand.WindowsServiceName + "Setup")
	if err != nil {
		return false
	}
	h, err := windows.OpenMutex(windows.SYNCHRONIZE, false, name)
	if err != nil {
		return false
	}
	windows.CloseHandle(h)
	return true
}
