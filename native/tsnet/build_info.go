package main

import (
	"errors"
	"regexp"
	"runtime"
	"strings"

	tailscale "tailscale.com"
	"tailscale.com/version"
)

// Set by the candidate build helper from the exact build inputs.
var buildSourceID string

type buildInformation struct {
	Type         string `json:"type"`
	Candidate    string `json:"candidate"`
	BaseVersion  string `json:"baseVersion"`
	ShortVersion string `json:"shortVersion"`
	LongVersion  string `json:"longVersion"`
	SourceID     string `json:"sourceId"`
	Custom       bool   `json:"custom"`
	GoVersion    string `json:"goVersion"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
}

var sourceIDPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

func validateBuildInformation(info buildInformation) error {
	if info.Type != "build-info" || info.Candidate != "tsnet-v1.104.5-cert7" || info.BaseVersion != "1.104.5" || info.ShortVersion != info.BaseVersion {
		return errors.New("candidate base version is inconsistent")
	}
	if !info.Custom || !sourceIDPattern.MatchString(info.SourceID) {
		return errors.New("candidate source identity is missing or invalid")
	}
	if strings.Contains(info.LongVersion, "ERR-BuildInfo") || info.LongVersion != "1.104.5-pocketrisu-exp-cert7-s"+info.SourceID[:12] {
		return errors.New("candidate long version is missing or invalid")
	}
	if info.GoVersion == "" || info.GOOS == "" || info.GOARCH == "" {
		return errors.New("runtime build information is missing")
	}
	return nil
}

func currentBuildInformation() (buildInformation, error) {
	info := buildInformation{
		Type: "build-info", Candidate: "tsnet-v1.104.5-cert7", BaseVersion: strings.TrimSpace(tailscale.VersionDotTxt),
		ShortVersion: version.Short(), LongVersion: version.Long(), SourceID: buildSourceID,
		Custom: true, GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
	}
	return info, validateBuildInformation(info)
}
