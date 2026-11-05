package main

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"
)

var errRegistrationRequired = errors.New("existing Tailscale registration required; interactive login disabled")

func clearInheritedTailscaleAuth(requireExisting bool) {
	// Preserve ordinary APK startup's existing environment contract.
	names := []string{"TS_AUTHKEY", "TS_CLIENT_SECRET", "TS_CONTROL_URL"}
	if requireExisting {
		names = append(names, "TS_AUTH_KEY", "TS_CLIENT_ID", "TS_ID_TOKEN", "TS_AUDIENCE", "TSNET_FORCE_LOGIN")
	}
	for _, name := range names {
		_ = os.Unsetenv(name)
	}
}

// Inspect metadata only. Missing or empty state must fail before starting tsnet.
// The backend remains responsible for interpreting existing state contents.
func requireExistingState(dir string) error {
	st, err := os.Stat(filepath.Join(dir, "tailscaled.state"))
	if err != nil || !st.Mode().IsRegular() || st.Size() == 0 {
		return errRegistrationRequired
	}
	return nil
}

func allowInteractiveLogin(requireRegistered bool) error {
	if requireRegistered {
		return errRegistrationRequired
	}
	return nil
}

// A certificate must succeed before a Tailnet HTTPS listener is created. The
// existing timeout is retained; server Noise key readiness is fixed in the
// control client, not by waiting longer or ignoring certificate errors here.
func obtainHTTPSListener(ctx context.Context, domain string,
	certPair func(context.Context, string) ([]byte, []byte, error),
	listenTLS func(string, string) (net.Listener, error)) (net.Listener, error) {
	certCtx, finish := context.WithTimeout(ctx, 30*time.Second)
	_, _, err := certPair(certCtx, domain)
	finish()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return listenTLS("tcp", ":443")
}
