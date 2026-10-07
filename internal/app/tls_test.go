// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package app

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func TestLocalCAIssuesTrustedServerCertificate(t *testing.T) {
	dir := t.TempDir()
	cert, key, err := EnsureCertificates(dir, []string{"einvoice.office.local", "10.8.0.5"})
	if err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(CACertPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		t.Fatal("ca.pem is not a certificate")
	}
	pair, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(pair.Certificate[0])
	for _, name := range []string{"localhost", "127.0.0.1", "einvoice.office.local", "10.8.0.5"} {
		if _, err := leaf.Verify(x509.VerifyOptions{Roots: pool, DNSName: name}); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	if days := leaf.NotAfter.Sub(leaf.NotBefore).Hours() / 24; days > 825 {
		t.Errorf("server certificate valid %.0f days; Apple devices require at most 825", days)
	}
	block, _ := pem.Decode(caPEM)
	if ca, err := x509.ParseCertificate(block.Bytes); err != nil || !ca.IsCA {
		t.Errorf("ca.pem must be a CA certificate (err=%v)", err)
	}

	// A second start keeps the same certificate; a new host name re-issues it from the same CA.
	before, _ := os.ReadFile(cert)
	if _, _, err := EnsureCertificates(dir, []string{"einvoice.office.local", "10.8.0.5"}); err != nil {
		t.Fatal(err)
	}
	same, _ := os.ReadFile(cert)
	if string(before) != string(same) {
		t.Error("certificate re-issued without reason")
	}
	if _, _, err := EnsureCertificates(dir, []string{"new.office.local"}); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(cert)
	caAfter, _ := os.ReadFile(CACertPath(dir))
	if string(after) == string(before) || string(caAfter) != string(caPEM) {
		t.Error("a new host must re-issue the server certificate but keep the CA")
	}
}
