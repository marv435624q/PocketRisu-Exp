package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type runtimeOptions struct {
	stateDir, hostname, clientTarget string
	logoutOnly, checkArgs            bool
	requireRegistered                bool
	verifyHTTPS                      bool
}

var errArgumentParsing = errors.New("invalid command-line arguments")

func parseRuntimeOptions(args []string, output io.Writer) (runtimeOptions, error) {
	var opts runtimeOptions
	fs := flag.NewFlagSet("pocketrisu-tsnet", flag.ContinueOnError)
	var parserOutput bytes.Buffer
	fs.SetOutput(&parserOutput)
	fs.Usage = func() {
		_, _ = fmt.Fprintln(fs.Output(), "Standalone: --diagnose-argv | --build-info")
		fs.PrintDefaults()
	}
	fs.StringVar(&opts.stateDir, "state-dir", "", "private, persistent Tailscale state directory")
	fs.StringVar(&opts.hostname, "hostname", "pocketrisu", "tailnet device hostname")
	fs.BoolVar(&opts.logoutOnly, "logout-only", false, "log out and explicitly reset this device identity")
	fs.StringVar(&opts.clientTarget, "client-target", "", "client's configured tailnet host:port; no server publishing")
	fs.BoolVar(&opts.checkArgs, "check-args", false, "validate arguments without starting Tailscale")
	fs.BoolVar(&opts.requireRegistered, "require-registered", false, "reuse an existing registration; refuse interactive login")
	fs.BoolVar(&opts.verifyHTTPS, "verify-https", false, "verify this server's HTTPS over tsnet and stop; requires an existing registration")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = io.Copy(output, &parserOutput)
			return opts, flag.ErrHelp
		}
		// flag's bool/value errors can include the supplied value. Preserve
		// the exit class without copying paths, keys or other argument data.
		_, _ = fmt.Fprintln(output, "flag parsing error; use --help for supported options")
		return opts, errArgumentParsing
	}
	if fs.NArg() != 0 {
		_, _ = fmt.Fprintln(output, "unexpected positional argument; use --help for supported options")
		return opts, errArgumentParsing
	}
	if opts.verifyHTTPS && (!opts.requireRegistered || opts.logoutOnly || opts.clientTarget != "") {
		_, _ = fmt.Fprintln(output, "verify-https requires require-registered and server mode")
		return opts, errArgumentParsing
	}
	if !filepath.IsAbs(opts.stateDir) || filepath.Base(filepath.Clean(opts.stateDir)) != "tailscale-state" {
		return opts, errors.New("state-dir must be an absolute path ending in tailscale-state")
	}
	return opts, nil
}

type argvDiagnostic struct {
	Type                  string `json:"type"`
	RawArgc               int    `json:"rawArgc"`
	NormalizedArgc        int    `json:"normalizedArgc"`
	Android               bool   `json:"android"`
	MarkerPresent         bool   `json:"markerPresent"`
	MarkerAbsolute        bool   `json:"markerAbsolute"`
	FirstArgAbsolute      bool   `json:"firstArgAbsolute"`
	FirstArgMatchesMarker bool   `json:"firstArgMatchesMarker"`
	RemovedExecutableArg  bool   `json:"removedExecutableArg"`
}

func normalizeStartupArgs(raw []string, android bool, termuxExecutable string) ([]string, argvDiagnostic) {
	normalized := append([]string(nil), raw...)
	diag := argvDiagnostic{Type: "argv-diagnostic", RawArgc: len(raw), Android: android,
		MarkerPresent: termuxExecutable != "", MarkerAbsolute: filepath.IsAbs(termuxExecutable)}
	if len(raw) > 1 {
		diag.FirstArgAbsolute = filepath.IsAbs(raw[1])
		if diag.MarkerPresent && diag.FirstArgAbsolute {
			if marker, err := filepath.Abs(termuxExecutable); err == nil {
				markerInfo, markerErr := os.Stat(filepath.Clean(marker))
				argInfo, argErr := os.Stat(filepath.Clean(raw[1]))
				diag.FirstArgMatchesMarker = markerErr == nil && argErr == nil &&
					markerInfo.Mode().IsRegular() && argInfo.Mode().IsRegular() && os.SameFile(markerInfo, argInfo)
			}
		}
	}
	if android && diag.FirstArgMatchesMarker {
		normalized = append(normalized[:1], normalized[2:]...)
		diag.RemovedExecutableArg = true
	}
	diag.NormalizedArgc = len(normalized)
	return normalized, diag
}

func diagnosticRequested(raw []string) bool {
	return len(raw) == 2 && raw[1] == "--diagnose-argv" ||
		len(raw) == 3 && filepath.IsAbs(raw[1]) && raw[2] == "--diagnose-argv"
}
