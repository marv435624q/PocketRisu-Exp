package main

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPSCertificateFailureNeverCreatesListener(t *testing.T) {
	for _, want := range []error{context.DeadlineExceeded, errors.New("zero serverNoiseKey")} {
		t.Run(want.Error(), func(t *testing.T) {
			called := false
			ln, err := obtainHTTPSListener(context.Background(), "synthetic.invalid",
				func(context.Context, string) ([]byte, []byte, error) { return nil, nil, want },
				func(string, string) (net.Listener, error) { called = true; return nil, nil })
			if !errors.Is(err, want) || ln != nil || called {
				t.Fatalf("certificate error not propagated before listener: err=%v, listener=%v, called=%v", err, ln, called)
			}
			if got := httpsPreparationStage(err); got != "certificate" {
				t.Fatalf("certificate failure stage=%q", got)
			}
		})
	}
}

func TestHTTPSListenerFailureAndCancellation(t *testing.T) {
	want := errors.New("synthetic listener failure")
	cert := func(context.Context, string) ([]byte, []byte, error) { return nil, nil, nil }
	ln, err := obtainHTTPSListener(context.Background(), "synthetic.invalid", cert,
		func(network, address string) (net.Listener, error) {
			if network != "tcp" || address != ":443" {
				t.Error("Tailnet HTTPS endpoint changed")
			}
			return nil, want
		})
	if ln != nil || !errors.Is(err, want) {
		t.Fatal("listener error hidden")
	}
	if got := httpsPreparationStage(err); got != "listener" {
		t.Fatalf("listener failure stage=%q", got)
	}
	ctx, cancel := context.WithCancel(context.Background())
	called := false
	ln, err = obtainHTTPSListener(ctx, "synthetic.invalid",
		func(context.Context, string) ([]byte, []byte, error) { cancel(); return nil, nil, nil },
		func(string, string) (net.Listener, error) { called = true; return nil, nil })
	if ln != nil || !errors.Is(err, context.Canceled) || called {
		t.Fatal("listener started after stop")
	}
	if got := httpsPreparationStage(err); got != "post-certificate" {
		t.Fatalf("post-certificate cancellation stage=%q", got)
	}
}

func TestHTTPSCertificateThenListenerAndProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("risu-auth") != "synthetic-auth" || r.URL.Path != "/synthetic" {
			t.Error("proxy request changed")
		}
		_, _ = io.WriteString(w, "synthetic HTTPS response")
	}))
	defer upstream.Close()
	target, _ := url.Parse(upstream.URL)
	var proxy *httptest.Server
	certificateReady := false
	listenerCalled := false
	ln, err := obtainHTTPSListener(context.Background(), "synthetic.invalid",
		func(context.Context, string) ([]byte, []byte, error) { certificateReady = true; return nil, nil, nil },
		func(network, address string) (net.Listener, error) {
			if !certificateReady || network != "tcp" || address != ":443" {
				t.Fatal("HTTPS readiness order changed")
			}
			listenerCalled = true
			// A real TLS listener on a loopback ephemeral port stands in for
			// tsnet.ListenTLS(:443). It is not a Tailnet/ACME service test.
			proxy = httptest.NewUnstartedServer(newProxy(target))
			proxy.StartTLS()
			return proxy.Listener, nil
		})
	if err != nil || ln == nil || !listenerCalled {
		t.Fatal("ready listener not returned")
	}
	defer proxy.Close()
	req, _ := http.NewRequest("GET", proxy.URL+"/synthetic", nil)
	req.Header.Set("risu-auth", "synthetic-auth")
	res, err := proxy.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil || res.StatusCode != 200 || string(data) != "synthetic HTTPS response" || res.TLS == nil || res.TLS.Version < tls.VersionTLS12 {
		t.Fatal("local HTTPS proxy response failed")
	}
}

func TestRequireRegisteredMode(t *testing.T) {
	state := filepath.Join(fixtureDir(t), "tailscale-state")
	opts, err := parseRuntimeOptions([]string{"--require-registered", "--check-args", "--state-dir", state}, io.Discard)
	if err != nil || !opts.requireRegistered || !opts.checkArgs {
		t.Fatal("guarded parser options missing")
	}
	if !errors.Is(allowInteractiveLogin(opts.requireRegistered), errRegistrationRequired) || allowInteractiveLogin(false) != nil {
		t.Fatal("guarded verification must refuse interactive login while ordinary mode stays unchanged")
	}
}

func TestRequireRegisteredRejectsMissingAndEmptyState(t *testing.T) {
	dir := filepath.Join(fixtureDir(t), "tailscale-state")
	if !errors.Is(requireExistingState(dir), errRegistrationRequired) {
		t.Fatal("missing state must fail")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("guard must not create state")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(dir, "tailscaled.state")
	if err := os.WriteFile(state, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(requireExistingState(dir), errRegistrationRequired) {
		t.Fatal("empty state must fail")
	}
	data := []byte("synthetic state sentinel\n")
	if err := os.WriteFile(state, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := requireExistingState(dir); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(state)
	if err != nil || string(after) != string(data) {
		t.Fatal("guard changed state")
	}
}

func TestRegistrationGuardDoesNotChangeOrdinaryStartupEnvironment(t *testing.T) {
	original := []string{"TS_AUTHKEY", "TS_CLIENT_SECRET", "TS_CONTROL_URL"}
	extra := []string{"TS_AUTH_KEY", "TS_CLIENT_ID", "TS_ID_TOKEN", "TS_AUDIENCE", "TSNET_FORCE_LOGIN"}
	for _, name := range append(original, extra...) {
		t.Setenv(name, "synthetic")
	}
	clearInheritedTailscaleAuth(false)
	for _, name := range original {
		if _, ok := os.LookupEnv(name); ok {
			t.Fatal("ordinary startup auth clearing changed")
		}
	}
	for _, name := range extra {
		if os.Getenv(name) != "synthetic" {
			t.Fatal("guard changed ordinary startup")
		}
	}
	clearInheritedTailscaleAuth(true)
	for _, name := range extra {
		if _, ok := os.LookupEnv(name); ok {
			t.Fatal("guard inherited credentials")
		}
	}
}
