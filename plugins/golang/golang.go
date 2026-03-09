// Package golang implements the nvy plugin for the Go programming language runtime.
// It downloads official Go tarballs from dl.google.com and verifies them using
// the per-file SHA-256 checksums published alongside each release.
package golang

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/trevorphillipscoding/nvy/internal/semver"
	"github.com/trevorphillipscoding/nvy/plugins"
)

const downloadBase = "https://dl.google.com/go"

// releasesAPI is a var so tests can override it with an httptest server.
// include=all returns every stable release, not just the two most-recent minor branches.
var releasesAPI = "https://go.dev/dl/?mode=json&include=all"

func init() {
	plugins.Register(New())
}

type goPlugin struct{}

// New returns the Go plugin. Called by init(); exposed for testing.
func New() plugins.Plugin { return &goPlugin{} }

func (g *goPlugin) Name() string { return "go" }

func (g *goPlugin) Aliases() []string { return []string{"golang"} }

// AvailableVersions returns all stable Go versions (including archived releases) as exact semantic versions.
func (g *goPlugin) AvailableVersions(_, _ string) ([]string, error) {
	return fetchStableGoVersions()
}

// Resolve builds the download spec for a Go release tarball.
//
// Official naming convention:
//
//	go<version>.<os>-<arch>.tar.gz
//	go<version>.<os>-<arch>.tar.gz.sha256  ← single-line hex SHA-256
//
// Example: go1.22.1.linux-amd64.tar.gz
func (g *goPlugin) Resolve(ver, goos, goarch string) (*plugins.DownloadSpec, error) {
	os, err := normalizeOS(goos)
	if err != nil {
		return nil, err
	}
	arch, err := normalizeArch(goarch)
	if err != nil {
		return nil, err
	}

	v, err := semver.ParseVersion(ver)
	if err != nil {
		return nil, fmt.Errorf("go plugin: %w", err)
	}
	ver = v.String()
	filename := fmt.Sprintf("go%s.%s-%s.tar.gz", ver, os, arch)
	url := fmt.Sprintf("%s/%s", downloadBase, filename)

	return &plugins.DownloadSpec{
		URL:              url,
		ChecksumURL:      url + ".sha256",
		ChecksumFilename: "", // plain mode: response body is the raw hex hash
		StripComponents:  1,  // archive has a top-level "go/" directory to strip
	}, nil
}

func fetchStableGoVersions() ([]string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(releasesAPI)
	if err != nil {
		return nil, fmt.Errorf("go plugin: fetching releases: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("go plugin: reading releases response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("go plugin: releases API returned %s", resp.Status)
	}

	var releases []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("go plugin: parsing releases JSON: %w", err)
	}

	versions := make([]string, 0, len(releases))
	for _, r := range releases {
		if !r.Stable {
			continue
		}
		v := strings.TrimPrefix(r.Version, "go")
		if _, parseErr := semver.ParseVersion(v); parseErr == nil {
			versions = append(versions, v)
		}
	}
	if len(versions) == 0 {
		return nil, fmt.Errorf("go plugin: no stable semantic versions found")
	}
	return versions, nil
}

func normalizeOS(goos string) (string, error) {
	switch goos {
	case "linux":
		return "linux", nil
	case "darwin":
		return "darwin", nil
	default:
		return "", fmt.Errorf("go plugin: unsupported OS %q (supported: linux, darwin)", goos)
	}
}

func normalizeArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		return "amd64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("go plugin: unsupported architecture %q (supported: amd64, arm64)", goarch)
	}
}
