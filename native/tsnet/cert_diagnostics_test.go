package main

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestParseCertStageLogIsNarrow(t *testing.T) {
	stage, state, ok := parseCertStageLog(`cert("private.example"): pocketrisu-cert-stage: dns start`)
	if !ok || stage != "dns" || state != "start" {
		t.Fatal("safe marker was not parsed")
	}
	for _, value := range []string{
		"pocketrisu-cert-stage: private start",
		"pocketrisu-cert-stage: dns private",
		"pocketrisu-cert-stage: dns start private",
		"ordinary log",
	} {
		if _, _, ok := parseCertStageLog(value); ok {
			t.Fatalf("unexpected marker accepted: %q", value)
		}
	}
}

func TestCertProgressTrackerEmitsOnlyStructuredAllowlist(t *testing.T) {
	base := time.Unix(100, 0)
	now := base
	var got []httpsProgress
	tracker := newCertProgressTracker(func(p httpsProgress) { got = append(got, p) })
	tracker.now = func() time.Time { return now }
	tracker.Begin("target.example")
	tracker.Logf(`cert("other.example"): pocketrisu-cert-stage: order start`, "private-arg")
	tracker.Logf(`acme: cert("target.example"): pocketrisu-cert-stage: order start`, "private-arg")
	now = base.Add(1250 * time.Millisecond)
	tracker.Logf(`acme: cert("target.example"): pocketrisu-cert-stage: order done`, "private-arg")
	tracker.End()
	tracker.Logf(`acme: cert("target.example"): pocketrisu-cert-stage: certificate start`, "private-arg")
	if len(got) != 2 || got[0].Stage != "order" || got[0].ElapsedMS != 0 ||
		got[1].State != "done" || got[1].ElapsedMS != 1250 {
		t.Fatalf("unexpected progress: %#v", got)
	}
}

func TestNetworkErrorClass(t *testing.T) {
	if got := networkErrorClass(context.DeadlineExceeded); got != "timeout" {
		t.Fatalf("deadline=%q", got)
	}
	if got := networkErrorClass(&net.DNSError{Err: "synthetic", Name: "example.invalid"}); got != "dns" {
		t.Fatalf("dns=%q", got)
	}
	if got := networkErrorClass(&net.OpError{Op: "dial", Net: "tcp", Err: errors.New("synthetic")}); got != "tcp" {
		t.Fatalf("tcp=%q", got)
	}
	if got := networkErrorClass(errors.New("tls: synthetic handshake failure")); got != "tls" {
		t.Fatalf("tls=%q", got)
	}
}
