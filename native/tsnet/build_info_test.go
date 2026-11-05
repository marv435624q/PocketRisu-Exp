package main

import (
	"strings"
	"testing"
)

func TestValidateBuildInformation(t *testing.T) {
	id := strings.Repeat("a", 64)
	valid := buildInformation{Type: "build-info", Candidate: "tsnet-v1.104.5-cert5", BaseVersion: "1.104.5", ShortVersion: "1.104.5", LongVersion: "1.104.5-pocketrisu-exp-cert5-s" + id[:12], SourceID: id, Custom: true, GoVersion: "go1.27.3", GOOS: "android", GOARCH: "arm64"}
	cases := []struct {
		name      string
		edit      func(*buildInformation)
		wantError bool
	}{
		{"valid", func(*buildInformation) {}, false},
		{"base", func(v *buildInformation) { v.BaseVersion = "1.104.4" }, true},
		{"short", func(v *buildInformation) { v.ShortVersion = "1.104.5-ERR-BuildInfo" }, true},
		{"long", func(v *buildInformation) { v.LongVersion = "1.104.5-ERR-BuildInfo" }, true},
		{"wrong identity prefix", func(v *buildInformation) { v.LongVersion = "1.104.5-pocketrisu-exp-cert5-sbbbbbbbbbbbb" }, true},
		{"short identity", func(v *buildInformation) { v.SourceID = "abc" }, true},
		{"nonhex identity", func(v *buildInformation) { v.SourceID = strings.Repeat("g", 64) }, true},
		{"custom", func(v *buildInformation) { v.Custom = false }, true},
		{"runtime", func(v *buildInformation) { v.GOOS = "" }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info := valid
			tc.edit(&info)
			if got := validateBuildInformation(info); (got != nil) != tc.wantError {
				t.Fatalf("validateBuildInformation() error = %v, wantError %v", got, tc.wantError)
			}
		})
	}
}
