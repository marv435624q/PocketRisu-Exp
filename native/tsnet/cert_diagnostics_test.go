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
	tracker.Logf(`control: pocketrisu-cert-stage: noise-key start`, "private-arg")
	now = base.Add(1750 * time.Millisecond)
	tracker.Logf(`control: pocketrisu-cert-stage: noise-key done`, "private-arg")
	tracker.Logf(`control: pocketrisu-cert-stage: fresh-noise-client start`, "private-arg")
	now = base.Add(1875 * time.Millisecond)
	tracker.Logf(`control: pocketrisu-cert-stage: fresh-noise-client done`, "private-arg")
	tracker.Logf(`control: pocketrisu-cert-stage: whoami-wrote-request done`, "private-arg")
	tracker.Logf(`control: pocketrisu-cert-stage: setdns-first-byte done`, "private-arg")
	tracker.Logf(`control: pocketrisu-cert-stage: noise-upgrade-443 start`, "private-arg")
	now = base.Add(2000 * time.Millisecond)
	tracker.Logf(`control: pocketrisu-cert-stage: noise-upgrade-443 done`, "private-arg")
	tracker.Logf(`control: pocketrisu-cert-stage: order start`, "private-arg")
	tracker.End()
	tracker.Logf(`control: pocketrisu-cert-stage: noise-client start`, "private-arg")
	tracker.Logf(`acme: cert("target.example"): pocketrisu-cert-stage: certificate start`, "private-arg")
	if len(got) != 10 || got[0].Stage != "order" || got[0].ElapsedMS != 0 ||
		got[1].State != "done" || got[1].ElapsedMS != 1250 ||
		got[2].Stage != "noise-key" || got[2].State != "start" ||
		got[3].Stage != "noise-key" || got[3].ElapsedMS != 500 ||
		got[4].Stage != "fresh-noise-client" || got[4].State != "start" ||
		got[5].Stage != "fresh-noise-client" || got[5].ElapsedMS != 125 ||
		got[6].Stage != "whoami-wrote-request" || got[6].State != "done" ||
		got[7].Stage != "setdns-first-byte" || got[7].State != "done" ||
		got[8].Stage != "noise-upgrade-443" || got[8].State != "start" ||
		got[9].Stage != "noise-upgrade-443" || got[9].ElapsedMS != 125 {
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
