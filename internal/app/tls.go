// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"einvoicing/internal/brand"
	"einvoicing/internal/config"
)

// Certificate files in <data>/tls. The server certificate is issued by a
// local certificate authority created on first start. Installing ca.pem as a
// trusted root on office PCs and phones removes browser warnings and lets
// phones install the web app.
const (
	caCertName = "ca.pem"
	caKeyName  = "ca-key.pem"
	certName   = "cert.pem"
	keyName    = "key.pem"
	// leafDays stays below Apple's 825-day limit for TLS server certificates.
	leafDays = 800
	// renewBefore re-issues the server certificate this long before expiry.
	renewBefore = 30 * 24 * time.Hour
)

// CACertPath returns the local CA certificate path (it may not exist when an
// external certificate is configured or for installations made before the CA).
func CACertPath(dataDir string) string { return config.CACertPath(dataDir) }

// EnsureCertificates makes sure a usable server certificate exists, issuing
// it from the local CA. extraHosts are additional DNS names or IP addresses
// the server is reached by (config tls.hosts). The server certificate is
// re-issued when it nears expiry or when a current address is not covered.
func EnsureCertificates(dataDir string, extraHosts []string) (certFile, keyFile string, err error) {
	dir := filepath.Join(dataDir, "tls")
	certFile, keyFile = filepath.Join(dir, certName), filepath.Join(dir, keyName)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", "", err
	}
	caFile, caKeyFile := filepath.Join(dir, caCertName), filepath.Join(dir, caKeyName)
	if !exists(caFile) && exists(certFile) && exists(keyFile) {
		// Installation from before the local CA: keep its certificate.
		return certFile, keyFile, nil
	}
	ca, caKey, err := loadOrCreateCA(caFile, caKeyFile)
	if err != nil {
		return "", "", err
	}
	dns, ips := serverNames(extraHosts)
	if leafValid(certFile, keyFile, ca, dns, ips) {
		return certFile, keyFile, nil
	}
	if err := issueLeaf(certFile, keyFile, ca, caKey, dns, ips); err != nil {
		return "", "", err
	}
	return certFile, keyFile, nil
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func serial() *big.Int {
	n, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 120))
	return n
}

func loadOrCreateCA(certFile, keyFile string) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	if exists(certFile) && exists(keyFile) {
		pair, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, nil, err
		}
		cert, err := x509.ParseCertificate(pair.Certificate[0])
		if err != nil {
			return nil, nil, err
		}
		key, ok := pair.PrivateKey.(*ecdsa.PrivateKey)
		if !ok {
			return nil, nil, errors.New("unsupported CA key type")
		}
		return cert, key, nil
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	host, _ := os.Hostname()
	tpl := &x509.Certificate{
		SerialNumber:          serial(),
		Subject:               pkix.Name{CommonName: brand.ProductName + " Local CA (" + host + ")", Organization: []string{brand.Developer}},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	if err := writePair(certFile, keyFile, der, key); err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	return cert, key, err
}

// serverNames lists localhost, the computer name, every interface address and
// the configured extra hosts.
func serverNames(extra []string) (dns []string, ips []net.IP) {
	dns = []string{"localhost"}
	ips = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
	if host, _ := os.Hostname(); host != "" {
		dns = append(dns, host)
	}
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && !ipn.IP.IsLoopback() && !ipn.IP.IsLinkLocalUnicast() {
				ips = append(ips, ipn.IP)
			}
		}
	}
	for _, h := range extra {
		if ip := net.ParseIP(h); ip != nil {
			ips = append(ips, ip)
		} else if h != "" {
			dns = append(dns, h)
		}
	}
	return dns, ips
}

// leafValid reports whether the existing server certificate is signed by ca,
// is not close to expiry and covers every current name and address.
func leafValid(certFile, keyFile string, ca *x509.Certificate, dns []string, ips []net.IP) bool {
	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return false
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil || leaf.CheckSignatureFrom(ca) != nil || time.Until(leaf.NotAfter) < renewBefore {
		return false
	}
	for _, d := range dns {
		if leaf.VerifyHostname(d) != nil {
			return false
		}
	}
	for _, ip := range ips {
		if leaf.VerifyHostname(ip.String()) != nil {
			return false
		}
	}
	return true
}

func issueLeaf(certFile, keyFile string, ca *x509.Certificate, caKey *ecdsa.PrivateKey, dns []string, ips []net.IP) error {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	host, _ := os.Hostname()
	tpl := &x509.Certificate{
		SerialNumber: serial(),
		Subject:      pkix.Name{CommonName: host, Organization: []string{brand.ProductName}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().AddDate(0, 0, leafDays),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     dns,
		IPAddresses:  ips,
	}
	der, err := x509.CreateCertificate(rand.Reader, tpl, ca, &key.PublicKey, caKey)
	if err != nil {
		return err
	}
	return writePair(certFile, keyFile, der, key)
}

func writePair(certFile, keyFile string, der []byte, key *ecdsa.PrivateKey) error {
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}), 0o600); err != nil {
		return err
	}
	return os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o644)
}
