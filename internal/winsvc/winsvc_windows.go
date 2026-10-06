//go:build windows

// Package winsvc runs the application as a Windows service.
package winsvc

import (
	"context"
	"fmt"
	"os"
	"time"

	"einvoicing/internal/brand"

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

// Install registers the service with automatic start.
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
	if s, err := m.OpenService(brand.WindowsServiceName); err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", brand.WindowsServiceName)
	}
	s, err := m.CreateService(brand.WindowsServiceName, exe, mgr.Config{
		DisplayName: brand.WindowsServiceDisplay,
		Description: "FBR Digital Invoicing integration and sales tax invoicing server",
		StartType:   mgr.StartAutomatic,
	}, "serve", "--data", dataDir)
	if err != nil {
		return err
	}
	defer s.Close()
	_ = s.SetRecoveryActions([]mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 10 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 60 * time.Second},
	}, 86400)
	return nil
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
