package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	base := os.Getenv("POCKETRISU_TSNET_TEST_DIR")
	if base == "" || !filepath.IsAbs(base) {
		t.Fatal("POCKETRISU_TSNET_TEST_DIR must be absolute")
	}
	if err := os.MkdirAll(base, 0700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(base, "args-case-")
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func fixtureFile(t *testing.T, name string) string {
	t.Helper()
	p := filepath.Join(fixtureDir(t), name)
	if err := os.WriteFile(p, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestNormalizeStartupArgs(t *testing.T) {
	exe := fixtureFile(t, "synthetic-binary")
	other := fixtureFile(t, "synthetic-binary")
	directory := fixtureDir(t)
	rel, err := filepath.Rel(mustGetwd(t), exe)
	if err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(fixtureDir(t), "synthetic-link")
	linkOK := os.Symlink(exe, link) == nil
	cases := []struct {
		name          string
		raw           []string
		android       bool
		marker        string
		remove, match bool
	}{
		{"absolute", []string{"different-argv0", exe, "--help"}, true, exe, true, true},
		{"relative marker", []string{"bin", exe, "--help"}, true, rel, true, true},
		{"no marker", []string{"bin", exe, "--help"}, true, "", false, false},
		{"non android", []string{"bin", exe, "--help"}, false, exe, false, true},
		{"different file same name", []string{"bin", other, "--help"}, true, exe, false, false},
		{"directory is not executable file", []string{"bin", directory, "--help"}, true, directory, false, false},
		{"relative first arg", []string{"bin", rel, "--help"}, true, exe, false, false},
		{"ordinary options", []string{"bin", "--state-dir", "/safe/tailscale-state"}, true, exe, false, false},
		{"one insertion only", []string{"bin", exe, exe, "--help"}, true, exe, true, true},
		{"no arguments", []string{"bin"}, true, exe, false, false},
	}
	if linkOK {
		cases = append(cases, struct {
			name          string
			raw           []string
			android       bool
			marker        string
			remove, match bool
		}{"symlink", []string{"bin", link, "--help"}, true, exe, true, true})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]string(nil), tc.raw...)
			got, diag := normalizeStartupArgs(tc.raw, tc.android, tc.marker)
			want := append([]string(nil), before...)
			if tc.remove {
				want = append(want[:1], want[2:]...)
			}
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(tc.raw, before) {
				t.Fatal("normalization or input preservation failed")
			}
			if diag.RemovedExecutableArg != tc.remove || diag.FirstArgMatchesMarker != tc.match || diag.RawArgc != len(before) || diag.NormalizedArgc != len(want) {
				t.Fatal("diagnostic mismatch")
			}
		})
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return wd
}

func TestParseOptionsAndDuplicateControl(t *testing.T) {
	exe := fixtureFile(t, "synthetic-executable")
	state := filepath.Join(fixtureDir(t), "tailscale-state")
	raw := []string{"pocketrisu", exe, "--state-dir", state, "--hostname", "example", "--client-target", "example.invalid:443", "--logout-only", "--check-args"}
	// Go's standard flag parser stops before the duplicate positional argument.
	fs := flag.NewFlagSet("control", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	originalState := fs.String("state-dir", "", "")
	if err := fs.Parse(raw[1:]); err != nil || *originalState != "" {
		t.Fatal("control did not reproduce flag stopping")
	}
	normalized, _ := normalizeStartupArgs(raw, true, exe)
	opts, err := parseRuntimeOptions(normalized[1:], io.Discard)
	if err != nil || opts.stateDir != state || opts.hostname != "example" || opts.clientTarget != "example.invalid:443" || !opts.logoutOnly || !opts.checkArgs {
		t.Fatal("normalized flags not recognized", err)
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("argument parsing must not create state")
	}
	if _, err := parseRuntimeOptions([]string{"--help"}, io.Discard); !errors.Is(err, flag.ErrHelp) {
		t.Fatal("help error")
	}
	if _, err := parseRuntimeOptions([]string{"--unknown"}, io.Discard); !errors.Is(err, errArgumentParsing) {
		t.Fatal("unknown flag")
	}
	if _, err := parseRuntimeOptions([]string{"--state-dir", "/not-a-valid-name"}, io.Discard); err == nil || err.Error() != "state-dir must be an absolute path ending in tailscale-state" {
		t.Fatal("state validation")
	}
	var output bytes.Buffer
	if _, err := parseRuntimeOptions([]string{"private-positional-value"}, &output); !errors.Is(err, errArgumentParsing) || strings.Contains(output.String(), "private-positional-value") {
		t.Fatal("positional value leaked or accepted")
	}
	opts, err = parseRuntimeOptions([]string{"--state-dir", state}, io.Discard)
	if err != nil || opts.hostname != "pocketrisu" || opts.logoutOnly || opts.checkArgs || opts.clientTarget != "" {
		t.Fatal("default options", err)
	}
}

func TestNormalAPKAndExecArgumentsStayIntact(t *testing.T) {
	for _, args := range [][]string{
		{"apk", "--state-dir", "/private/tailscale-state", "--hostname", "apk"},
		{"apk", "--exec-cloudflared", "/path/to/libcloudflared_exec.so"},
		{"apk", "--client-target", "host:443", "--logout-only"},
	} {
		got, diag := normalizeStartupArgs(args, true, "")
		if !reflect.DeepEqual(got, args) || diag.RemovedExecutableArg {
			t.Fatal("normal arguments changed")
		}
	}
	exe := fixtureFile(t, "synthetic-executable")
	got, _ := normalizeStartupArgs([]string{"apk", exe, "--exec-cloudflared", "/dev/null"}, true, exe)
	if !reflect.DeepEqual(got, []string{"apk", "--exec-cloudflared", "/dev/null"}) {
		t.Fatal("special dispatch arguments changed")
	}
}

func TestDiagnosticIsNarrowAndContainsNoValues(t *testing.T) {
	exe := fixtureFile(t, "secret-sentinel")
	cases := []struct {
		raw  []string
		want bool
	}{
		{[]string{"bin", "--diagnose-argv"}, true},
		{[]string{"bin", exe, "--diagnose-argv"}, true},
		{[]string{"bin", "relative", "--diagnose-argv"}, false},
		{[]string{"bin", "--hostname", "--diagnose-argv"}, false},
		{[]string{"bin", exe, "--diagnose-argv", "extra"}, false},
	}
	for _, tc := range cases {
		if diagnosticRequested(tc.raw) != tc.want {
			t.Fatal("diagnostic request shape")
		}
	}
	_, diag := normalizeStartupArgs([]string{"secret-sentinel", exe, "--diagnose-argv"}, true, exe)
	encoded, err := json.Marshal(diag)
	if err != nil || strings.Contains(string(encoded), exe) || strings.Contains(string(encoded), "secret-sentinel") {
		t.Fatal("diagnostic leaked values")
	}
}

func TestParserErrorsAndHelpDoNotEchoSuppliedValues(t *testing.T) {
	for _, args := range [][]string{
		{"--check-args", "--logout-only=private-value-sentinel"},
		{"--private-value-sentinel"},
		{"private-value-sentinel"},
		{"--hostname", "private-value-sentinel", "--help"},
	} {
		var output bytes.Buffer
		_, err := parseRuntimeOptions(args, &output)
		if err == nil || strings.Contains(output.String(), "private-value-sentinel") {
			t.Fatal("parser failure/help leaked supplied argument data")
		}
	}
}
