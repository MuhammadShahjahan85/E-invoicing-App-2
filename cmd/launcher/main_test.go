// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package main

import (
	"testing"

	"einvoicing/internal/config"
)

func TestAddress(t *testing.T) {
	cases := []struct {
		listen   string
		tls      bool
		certFile string
		hosts    []string
		url      string
		hostPort string
	}{
		{"0.0.0.0:8443", true, "", nil, "https://localhost:8443/", "localhost:8443"},
		{"127.0.0.1:9443", true, "", nil, "https://localhost:9443/", "localhost:9443"},
		{"[::]:8443", true, "", nil, "https://localhost:8443/", "localhost:8443"},
		{"192.168.1.10:8443", true, "", nil, "https://192.168.1.10:8443/", "192.168.1.10:8443"},
		{"127.0.0.1:8080", false, "", nil, "http://localhost:8080/", "localhost:8080"},
		// A company certificate is issued for the company's own host name.
		{"0.0.0.0:443", true, `C:\certs\einvoice.pem`, []string{"einvoice.office.local"}, "https://einvoice.office.local:443/", "einvoice.office.local:443"},
		// The local CA covers localhost, so extra names do not change the address.
		{"0.0.0.0:8443", true, "", []string{"einvoice.office.local"}, "https://localhost:8443/", "localhost:8443"},
		{"bad", true, "", nil, "https://localhost:8443/", "localhost:8443"},
	}
	for _, c := range cases {
		cfg := config.Default()
		cfg.Listen, cfg.TLS.Enabled, cfg.TLS.CertFile, cfg.TLS.Hosts = c.listen, c.tls, c.certFile, c.hosts
		url, hp := address(cfg)
		if url != c.url || hp != c.hostPort {
			t.Errorf("%s: got %s %s, want %s %s", c.listen, url, hp, c.url, c.hostPort)
		}
	}
}
