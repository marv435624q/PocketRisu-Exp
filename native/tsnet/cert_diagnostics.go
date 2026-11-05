package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const certStageMarker = "pocketrisu-cert-stage: "

var allowedCertStages = map[string]bool{
	"account": true,
	"order": true,
	"dns": true,
	"dns-lookup": true,
	"set-dns": true,
	"noise-key": true,
	"noise-client": true,
	"fresh-noise-client": true,
	"set-dns-post": true,
	"set-dns-response": true,
	"noise-dial-80": true,
	"noise-dial-443": true,
	"noise-upgrade-80": true,
	"noise-upgrade-443": true,
	"noise-handshake-80": true,
	"noise-handshake-443": true,
	"noise-h2-request": true,
	"acme-accept": true,
	"issuance-wait": true,
	"certificate": true,
}

var unprefixedControlCertStages = map[string]bool{
	"noise-key": true,
	"noise-client": true,
	"fresh-noise-client": true,
	"set-dns-post": true,
	"set-dns-response": true,
	"noise-dial-80": true,
	"noise-dial-443": true,
	"noise-upgrade-80": true,
	"noise-upgrade-443": true,
	"noise-handshake-80": true,
	"noise-handshake-443": true,
	"noise-h2-request": true,
}

var allowedCertStageStates = map[string]bool{
	"start": true,
	"done": true,
	"error": true,
}

type httpsProgress struct {
	Type      string `json:"type"`
	Stage     string `json:"stage"`
	State     string `json:"state"`
	ElapsedMS int64  `json:"elapsedMs"`
}

type certProgressTracker struct {
	mu           sync.Mutex
	starts       map[string]time.Time
	targetPrefix string
	now          func() time.Time
	emit         func(httpsProgress)
}

func newCertProgressTracker(emit func(httpsProgress)) *certProgressTracker {
	return &certProgressTracker{starts: map[string]time.Time{}, now: time.Now, emit: emit}
}

func (t *certProgressTracker) Begin(domain string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	clear(t.starts)
	t.targetPrefix = "cert(" + strconv.Quote(strings.TrimSuffix(domain, ".")) + "): "
}

func (t *certProgressTracker) End() {
	t.mu.Lock()
	defer t.mu.Unlock()
	clear(t.starts)
	t.targetPrefix = ""
}

func parseCertStageLog(format string) (stage, state string, ok bool) {
	i := strings.Index(format, certStageMarker)
	if i < 0 {
		return "", "", false
	}
	fields := strings.Fields(format[i+len(certStageMarker):])
	if len(fields) != 2 || !allowedCertStages[fields[0]] || !allowedCertStageStates[fields[1]] {
		return "", "", false
	}
	return fields[0], fields[1], true
}

// Logf intentionally never formats or emits args. Tailscale may prefix its
// format with a certificate domain; only the exact allowlisted marker suffix
// becomes structured output.
func (t *certProgressTracker) Logf(format string, _ ...any) {
	stage, state, ok := parseCertStageLog(format)
	marker := strings.Index(format, certStageMarker)
	if !ok || marker < 0 || t.emit == nil {
		return
	}
	now := t.now()
	var elapsed int64
	t.mu.Lock()
	if t.targetPrefix == "" {
		t.mu.Unlock()
		return
	}
	if !unprefixedControlCertStages[stage] && !strings.Contains(format[:marker], t.targetPrefix) {
		t.mu.Unlock()
		return
	}
	if state == "start" {
		t.starts[stage] = now
	} else if started, found := t.starts[stage]; found {
		elapsed = now.Sub(started).Milliseconds()
		delete(t.starts, stage)
	}
	t.mu.Unlock()
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed > 300000 {
		elapsed = 300000
	}
	t.emit(httpsProgress{Type: "https-progress", Stage: stage, State: state, ElapsedMS: elapsed})
}

func networkErrorClass(err error) string {
	if err == nil {
		return ""
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns"
	}
	var unknownAuthority x509.UnknownAuthorityError
	var hostnameError x509.HostnameError
	var invalidCertificate x509.CertificateInvalidError
	var recordHeader tls.RecordHeaderError
	if errors.As(err, &unknownAuthority) || errors.As(err, &hostnameError) ||
		errors.As(err, &invalidCertificate) || errors.As(err, &recordHeader) ||
		strings.Contains(strings.ToLower(err.Error()), "tls:") {
		return "tls"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return "tcp"
	}
	return "other"
}
