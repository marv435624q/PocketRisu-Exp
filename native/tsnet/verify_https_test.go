package main

import (
	"context"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
)

func TestHTTPSVerificationChecksTrustAndDoesNotRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
			t.Error("verification request changed")
		}
		w.Header().Set("Location", "https://external.invalid/")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()
	server.Config.ErrorLog = nil
	u, _ := url.Parse(server.URL)
	pool := x509.NewCertPool()
	pool.AddCert(server.Certificate())
	dial := (&net.Dialer{}).DialContext
	trusted := verifyHTTPS(context.Background(), u.Host, dial, pool)
	if !trusted.Success || !trusted.TLSVerified || trusted.HTTPStatus != 302 || trusted.ErrorCode != "" {
		t.Fatalf("trusted local TLS result: %+v", trusted)
	}
	untrusted := verifyHTTPS(context.Background(), u.Host, dial, nil)
	if untrusted.Success || untrusted.TLSVerified || untrusted.ErrorCode != "certificate" {
		t.Fatal("untrusted certificate accepted")
	}
}

func TestHTTPSVerificationHTTPFailureAndGuard(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(502) }))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	pool := x509.NewCertPool()
	pool.AddCert(server.Certificate())
	result := verifyHTTPS(context.Background(), u.Host, (&net.Dialer{}).DialContext, pool)
	if result.Success || !result.TLSVerified || result.HTTPStatus != 502 || result.ErrorCode != "http-response" {
		t.Fatal("backend failure hidden")
	}
	state := filepath.Join(fixtureDir(t), "tailscale-state")
	if _, err := parseRuntimeOptions([]string{"--verify-https", "--state-dir", state}, io.Discard); err == nil {
		t.Fatal("unguarded online verification accepted")
	}
	if opts, err := parseRuntimeOptions([]string{"--verify-https", "--require-registered", "--state-dir", state}, io.Discard); err != nil || !opts.verifyHTTPS {
		t.Fatal("guarded server verification rejected")
	}
}
