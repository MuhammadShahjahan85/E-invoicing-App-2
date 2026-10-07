// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import (
	"net/http"
	"testing"
)

// Registering all routes must not panic (pattern conflicts are detected at registration).
func TestRoutesRegister(t *testing.T) {
	s := &Server{}
	s.routes(http.NewServeMux())
}
