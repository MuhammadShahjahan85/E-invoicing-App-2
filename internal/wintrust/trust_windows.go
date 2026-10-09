// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

//go:build windows

// Package wintrust adds the local certificate authority to the Windows
// trusted root store, so that browsers on this computer open the HTTPS
// address without a warning.
package wintrust

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Supported reports whether the trust store can be changed on this system.
const Supported = true

const encoding = windows.X509_ASN_ENCODING | windows.PKCS_7_ASN_ENCODING

func rootStore() (windows.Handle, error) {
	name, _ := windows.UTF16PtrFromString("ROOT")
	store, err := windows.CertOpenStore(windows.CERT_STORE_PROV_SYSTEM, 0, 0, windows.CERT_SYSTEM_STORE_LOCAL_MACHINE, uintptr(unsafe.Pointer(name)))
	if err != nil {
		return 0, fmt.Errorf("open the computer's trusted root store (run as Administrator): %w", err)
	}
	return store, nil
}

// Trust adds the DER certificate to the computer's trusted root store.
func Trust(der []byte) error {
	if len(der) == 0 {
		return errors.New("empty certificate")
	}
	store, err := rootStore()
	if err != nil {
		return err
	}
	defer windows.CertCloseStore(store, 0)
	ctx, err := windows.CertCreateCertificateContext(encoding, &der[0], uint32(len(der)))
	if err != nil {
		return err
	}
	defer windows.CertFreeCertificateContext(ctx)
	return windows.CertAddCertificateContextToStore(store, ctx, windows.CERT_STORE_ADD_REPLACE_EXISTING, nil)
}

// Untrust removes the DER certificate from the computer's trusted root
// store; it is not an error when it is not there.
func Untrust(der []byte) error {
	if len(der) == 0 {
		return nil
	}
	store, err := rootStore()
	if err != nil {
		return err
	}
	defer windows.CertCloseStore(store, 0)
	ctx, err := windows.CertCreateCertificateContext(encoding, &der[0], uint32(len(der)))
	if err != nil {
		return err
	}
	defer windows.CertFreeCertificateContext(ctx)
	found, err := windows.CertFindCertificateInStore(store, encoding, 0, windows.CERT_FIND_EXISTING, unsafe.Pointer(ctx), nil)
	if err != nil || found == nil {
		return nil
	}
	// CertDeleteCertificateFromStore frees found.
	return windows.CertDeleteCertificateFromStore(found)
}
