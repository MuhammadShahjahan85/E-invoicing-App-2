// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build windows

// Package winsvc runs the application as a Windows service.
package winsvc

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"einvoicing/internal/brand"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Supported reports whether Windows service management is available.
const Supported = true

// IsService reports whether the process was started by the service manager.
func IsService() bool {
	ok, err := svc.IsWindowsService()
	return err == nil && ok
}

type handler struct {
	run func(ctx context.Context) error
}

func (h *handler) Execute(_ []string, req <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.run(ctx) }()
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for {
		select {
		case c := <-req:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				status <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case <-done:
				case <-time.After(30 * time.Second):
				}
				return false, 0
			}
		case err := <-done:
			cancel()
			if err != nil {
				return false, 1
			}
			return false, 0
		}
	}
}

// Run executes the application under the service manager.
func Run(run func(ctx context.Context) error) error {
	return svc.Run(brand.WindowsServiceName, &handler{run: run})
}

// Install registers the service with automatic start. When the service
// already exists (an upgrade), its registration is updated instead, e.g.
// for a new program folder.
func Install(dataDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager (run as Administrator): %w", err)
	}
	defer m.Disconnect()
	const description = "FBR Digital Invoicing integration and sales tax invoicing server"
	s, err := m.OpenService(brand.WindowsServiceName)
	if err == nil {
		defer s.Close()
		if err := update(s.Handle, commandLine(exe, "serve", "--data", dataDir), description); err != nil {
			return err
		}
	} else {
		s, err = m.CreateService(brand.WindowsServiceName, exe, mgr.Config{
			DisplayName: brand.WindowsServiceDisplay,
			Description: description,
			StartType:   mgr.StartAutomatic,
		}, "serve", "--data", dataDir)
		if err != nil {
			return err
		}
		defer s.Close()
	}
	_ = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400)
	if err := allowUsersToStart(s.Handle); err != nil {
		// The service works without it; only an administrator can then start it.
		fmt.Fprintln(os.Stderr, "warning: signed-in users cannot start the service:", err)
	}
	return nil
}

// update points an existing service at this program, with automatic start.
// Other settings (account, delayed start, dependencies) are left as they are.
func update(h windows.Handle, cmdLine, description string) error {
	p := func(s string) *uint16 { u, _ := windows.UTF16PtrFromString(s); return u }
	if err := windows.ChangeServiceConfig(h, windows.SERVICE_NO_CHANGE, windows.SERVICE_AUTO_START, windows.SERVICE_NO_CHANGE,
		p(cmdLine), nil, nil, nil, nil, nil, p(brand.WindowsServiceDisplay)); err != nil {
		return err
	}
	d := windows.SERVICE_DESCRIPTION{Description: p(description)}
	return windows.ChangeServiceConfig2(h, windows.SERVICE_CONFIG_DESCRIPTION, (*byte)(unsafe.Pointer(&d)))
}

// commandLine quotes a command line the way CreateService does.
func commandLine(exe string, args ...string) string {
	s := syscall.EscapeArg(exe)
	for _, a := range args {
		s += " " + syscall.EscapeArg(a)
	}
	return s
}

// serviceDACL is the Windows default for services, except that signed-in
// (interactive) users may also start the service (RP), so that the desktop
// shortcut can start it without administrator rights. Stopping, changing
// and deleting it still need an administrator.
const serviceDACL = "D:(A;;CCLCSWRPWPDTLOCRRC;;;SY)(A;;CCDCLCSWRPWPDTLOCRSDRCWDWO;;;BA)(A;;CCLCSWRPLOCRRC;;;IU)(A;;CCLCSWLOCRRC;;;SU)"

func allowUsersToStart(h windows.Handle) error {
	sd, err := windows.SecurityDescriptorFromString(serviceDACL)
	if err != nil {
		return err
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetSecurityInfo(h, windows.SE_SERVICE, windows.DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
}

// Uninstall removes the service.
func Uninstall() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(brand.WindowsServiceName)
	if err != nil {
		return fmt.Errorf("service not installed")
	}
	defer s.Close()
	_, _ = s.Control(svc.Stop)
	return s.Delete()
}

// Start starts the service.
func Start() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(brand.WindowsServiceName)
	if err != nil {
		return err
	}
	defer s.Close()
	return s.Start()
}

// Stop stops the service.
func Stop() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(brand.WindowsServiceName)
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = s.Control(svc.Stop)
	return err
}
