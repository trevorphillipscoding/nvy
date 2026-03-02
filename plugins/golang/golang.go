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

	"github.com/trevorphillipscoding/nvy/internal/version"
	"github.com/trevorphillipscoding/nvy/plugins"
)

const downloadBase = "https://dl.google.com/go"

// releasesAPI is a var so tests can override it with an httptest server.
var releasesAPI = "https://go.dev/dl/?mode=json"

func init() {
	plugins.Register(New())
}

type goPlugin struct{}

// New returns the Go plugin. Called by init(); exposed for testing.
func New() plugins.Plugin { return &goPlugin{} }

func (g *goPlugin) Name() string { return "go" }

func (g *goPlugin) Aliases() []string { return []string{"golang"} }

// Resolve builds the download spec for a Go release tarball.
//
// Official naming convention:
//
//	go<version>.<os>-<arch>.tar.gz
//	go<version>.<os>-<arch>.tar.gz.sha256  ← single-line hex SHA-256
//
// Example: go1.22.1.linux-amd64.tar.gz
//
// Partial versions (fewer than two dots, no +tag) resolve to the latest
// matching stable release:
//
//	"1"    → latest 1.x.y
//	"1.26" → latest 1.26.x
func (g *goPlugin) Resolve(ver, goos, goarch string) (*plugins.DownloadSpec, error) {
	os, err := normalizeOS(goos)
	if err != nil {
		return nil, err
	}
	arch, err := normalizeArch(goarch)
	if err != nil {
		return nil, err
	}

	var resolvedVersion string
	base := strings.SplitN(ver, "+", 2)[0]
	if strings.Count(base, ".") < 2 {
		latest, err := findLatestGoVersion(base)
		if err != nil {
			return nil, err
		}
		resolvedVersion = latest
		ver = latest
	} else {
		ver = version.Normalize(ver)
	}

	filename := fmt.Sprintf("go%s.%s-%s.tar.gz", ver, os, arch)
	url := fmt.Sprintf("%s/%s", downloadBase, filename)

	return &plugins.DownloadSpec{
		URL: url,
		// The .sha256 file contains a single hex-encoded SHA-256 hash, no filename prefix.
		ChecksumURL:      url + ".sha256",
		ChecksumFilename: "", // plain mode: response body is the raw hex hash
		StripComponents:  1,  // archive has a top-level "go/" directory to strip
		ResolvedVersion:  resolvedVersion,
	}, nil
}

// findLatestGoVersion returns the latest stable Go release whose version
// starts with prefix (e.g. "1" or "1.26"). The go.dev/dl API returns releases
// newest-first, so the first match is the latest.
func findLatestGoVersion(prefix string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(releasesAPI)
	if err != nil {
		return "", fmt.Errorf("go plugin: fetching releases: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("go plugin: reading releases response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("go plugin: releases API returned %s", resp.Status)
	}

	var releases []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("go plugin: parsing releases JSON: %w", err)
	}

	// Versions in the API are "go1.24.1"; add a dot after prefix to avoid
	// "go1.2" matching "go1.20.x".
	wantPrefix := "go" + prefix + "."
	for _, r := range releases {
		if r.Stable && strings.HasPrefix(r.Version, wantPrefix) {
			return strings.TrimPrefix(r.Version, "go"), nil
		}
	}
	return "", fmt.Errorf("go plugin: no stable release found for Go %s.*", prefix)
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
