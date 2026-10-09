// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package service

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"einvoicing/internal/fbr"
)

// PublicIPServices answer with the caller's public IP address in plain text.
var PublicIPServices = []string{"https://api.ipify.org", "https://checkip.amazonaws.com"}

// PublicIP returns the address this server's internet traffic comes from:
// the address PRAL (or the licensed integrator) must whitelist before FBR
// accepts production calls. It asks a public echo service, so it only runs
// when a user asks for it.
func (s *Service) PublicIP(ctx context.Context) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	hc := fbr.NewHTTPClient(8 * time.Second)
	for _, u := range PublicIPServices {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			continue
		}
		resp, err := hc.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
		resp.Body.Close()
		if ip := net.ParseIP(strings.TrimSpace(string(body))); resp.StatusCode == http.StatusOK && ip != nil {
			return ip.String(), strings.TrimPrefix(u, "https://"), nil
		}
	}
	return "", "", errors.New("could not reach a public IP service from this server; ask your internet provider for the static public IP of this connection")
}
