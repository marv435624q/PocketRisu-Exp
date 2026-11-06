package main

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
)

var errRegistrationRequired = errors.New("existing Tailscale registration required; interactive login disabled")

type httpsPreparationError struct {
	Stage string
	Err   error
}

func (e *httpsPreparationError) Error() string { return e.Err.Error() }
func (e *httpsPreparationError) Unwrap() error { return e.Err }

func httpsPreparationStage(err error) string {
	var prep *httpsPreparationError
	if errors.As(err, &prep) {
		return prep.Stage
	}
	return ""
}

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

// A certificate must succeed before a Tailnet HTTPS listener is created.
// Certificate issuance uses the caller's context directly. ACME DNS-01 can
// legitimately take longer than 30 seconds, so this layer must not impose a
// shorter deadline than its caller.
func obtainHTTPSListener(ctx context.Context, domain string,
	certPair func(context.Context, string) ([]byte, []byte, error),
	listenTLS func(string, string) (net.Listener, error)) (net.Listener, error) {
	_, _, err := certPair(ctx, domain)
	if err != nil {
		return nil, &httpsPreparationError{Stage: "certificate", Err: err}
	}
	if err := ctx.Err(); err != nil {
		return nil, &httpsPreparationError{Stage: "post-certificate", Err: err}
	}
	ln, err := listenTLS("tcp", ":443")
	if err != nil {
		return nil, &httpsPreparationError{Stage: "listener", Err: err}
	}
	return ln, nil
}
