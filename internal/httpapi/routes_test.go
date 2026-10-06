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
