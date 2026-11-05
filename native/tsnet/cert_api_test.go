package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tailscale.com/ipn/ipnlocal"
	"tailscale.com/ipn/localapi"
)

// Exercise the actual registered cert route without contacting ACME or a
// tailnet: a caller lacking certificate permission must reach the cert handler
// and receive its 403, rather than the missing-route 404 seen on Android R2.
// Android inclusion itself is separately verified using target-specific go list.
func TestLocalCertificateRouteIsRegisteredAndChecksPermission(t *testing.T) {
	handler := localapi.NewHandler(localapi.HandlerConfig{
		Backend: &ipnlocal.LocalBackend{},
		Logf: func(string, ...any) {},
	})
	request := httptest.NewRequest(http.MethodGet,
		"http://local-tailscaled.sock/localapi/v0/cert/pocketrisu.example.ts.net?type=pair", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "cert access denied") {
		t.Fatalf("certificate route not registered or permission check changed: status=%d body=%q",
			response.Code, response.Body.String())
	}
}
