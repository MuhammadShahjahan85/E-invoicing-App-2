// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"

	"einvoicing/internal/brand"
)

// singleInstance keeps a second click on the shortcut from opening a
// second window while the first launcher is still waiting for the server.
func singleInstance() bool {
	name, _ := windows.UTF16PtrFromString(`Local\` + brand.WindowsServiceName + "Launcher")
	_, err := windows.CreateMutex(nil, false, name)
	return !errors.Is(err, windows.ERROR_ALREADY_EXISTS)
}

// startService asks Windows to start the service. The installer allows
// signed-in users to start it, so this needs no administrator rights.
func startService() error {
	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return err
	}
	defer windows.CloseServiceHandle(scm)
	name, _ := windows.UTF16PtrFromString(brand.WindowsServiceName)
	s, err := windows.OpenService(scm, name, windows.SERVICE_START|windows.SERVICE_QUERY_STATUS)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return errNotInstalled
		}
		return err
	}
	defer windows.CloseServiceHandle(s)
	if err := windows.StartService(s, 0, nil); err != nil && !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		return err
	}
	return nil
}

// openWindow shows the app in Edge or Chrome app mode, or else in the
// default browser.
func openWindow(url string) error {
	if b := appBrowser(); b != "" {
		cmd := exec.Command(b, "--app="+url)
		if err := cmd.Start(); err == nil {
			_ = cmd.Process.Release()
			return nil
		}
	}
	verb, _ := windows.UTF16PtrFromString("open")
	target, _ := windows.UTF16PtrFromString(url)
	return windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL)
}

// appBrowser finds Microsoft Edge (part of Windows 10 and 11) or Google
// Chrome.
func appBrowser() string {
	for _, exe := range []string{"msedge.exe", "chrome.exe"} {
		for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
			if p := appPath(root, exe); p != "" {
				return p
			}
		}
	}
	for _, p := range []string{
		filepath.Join(os.Getenv("ProgramFiles(x86)"), `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`),
		filepath.Join(os.Getenv("ProgramFiles"), `Google\Chrome\Application\chrome.exe`),
		filepath.Join(os.Getenv("LocalAppData"), `Google\Chrome\Application\chrome.exe`),
	} {
		if isFile(p) {
			return p
		}
	}
	return ""
}

// appPath reads a program's location from the App Paths registry key.
func appPath(root registry.Key, exe string) string {
	k, err := registry.OpenKey(root, `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\`+exe, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	p, typ, err := k.GetStringValue("")
	if err != nil {
		return ""
	}
	if typ == registry.EXPAND_SZ {
		if x, err := registry.ExpandString(p); err == nil {
			p = x
		}
	}
	p = strings.Trim(p, `" `)
	if !isFile(p) {
		return ""
	}
	return p
}

func isFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// fail shows an error message (the launcher has no console).
func fail(msg string) {
	title, _ := windows.UTF16PtrFromString(brand.ProductName)
	text, _ := windows.UTF16PtrFromString(msg)
	_, _ = windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONWARNING|windows.MB_SETFOREGROUND)
}
