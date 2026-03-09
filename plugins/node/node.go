// Package node implements the nvy plugin for Node.js.
// It downloads official Node.js tarballs from nodejs.org and verifies them using
// the SHASUMS256.txt file published alongside each release.
package node

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/trevorphillipscoding/nvy/internal/verutil"
	"github.com/trevorphillipscoding/nvy/plugins"
)

const distBase = "https://nodejs.org/dist"

// releasesAPI is a var so tests can override it with an httptest server.
var releasesAPI = "https://nodejs.org/dist/index.json"

func init() {
	plugins.Register(New())
}

type nodePlugin struct{}

// New returns the Node.js plugin. Called by init(); exposed for testing.
func New() plugins.Plugin { return &nodePlugin{} }

func (n *nodePlugin) Name() string { return "node" }

func (n *nodePlugin) Aliases() []string { return []string{"nodejs", "node.js"} }

// LatestVersion returns the latest Node.js release whose version starts with
// prefix (e.g. "22" or "22.11"). goos/goarch are unused — Node releases are
// platform-agnostic in the version numbering scheme.
func (n *nodePlugin) LatestVersion(prefix, _, _ string) (string, error) {
	return findLatestNodeVersion(prefix)
}

// Resolve builds the download spec for a Node.js release tarball.
//
// Official naming convention:
//
//	node-v<version>-<os>-<arch>.tar.gz
//	SHASUMS256.txt  ← multi-entry file; we look up our filename inside it
//
// Example: node-v20.11.1-linux-x64.tar.gz
func (n *nodePlugin) Resolve(ver, goos, goarch string) (*plugins.DownloadSpec, error) {
	os, err := normalizeOS(goos)
	if err != nil {
		return nil, err
	}
	arch, err := normalizeArch(goarch)
	if err != nil {
		return nil, err
	}

	ver = verutil.Normalize(ver)
	// Node uses "v" prefix in both the URL path and the archive filename.
	filename := fmt.Sprintf("node-v%s-%s-%s.tar.gz", ver, os, arch)
	url := fmt.Sprintf("%s/v%s/%s", distBase, ver, filename)
	checksumURL := fmt.Sprintf("%s/v%s/SHASUMS256.txt", distBase, ver)

	return &plugins.DownloadSpec{
		URL:              url,
		ChecksumURL:      checksumURL,
		ChecksumFilename: filename, // SHASUMS256 mode: look up this filename in the file
		StripComponents:  1,        // archive has a top-level "node-v<version>-<os>-<arch>/" directory
	}, nil
}

// findLatestNodeVersion returns the latest Node.js release whose version
// starts with prefix (e.g. "22" or "22.11"). The dist/index.json is
// ordered newest-first, so the first match is the latest.
func findLatestNodeVersion(prefix string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(releasesAPI)
	if err != nil {
		return "", fmt.Errorf("node plugin: fetching releases: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("node plugin: reading releases response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("node plugin: releases API returned %s", resp.Status)
	}

	var releases []struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("node plugin: parsing releases JSON: %w", err)
	}

	// Versions are "v22.13.1"; add a dot after prefix to avoid "v2" matching "v20.x".
	wantPrefix := "v" + prefix + "."
	for _, r := range releases {
		if strings.HasPrefix(r.Version, wantPrefix) {
			return strings.TrimPrefix(r.Version, "v"), nil
		}
	}
	return "", fmt.Errorf("node plugin: no release found for Node.js %s.*", prefix)
}

func normalizeOS(goos string) (string, error) {
	switch goos {
	case "linux":
		return "linux", nil
	case "darwin":
		return "darwin", nil
	default:
		return "", fmt.Errorf("node plugin: unsupported OS %q", goos)
	}
}

func normalizeArch(goarch string) (string, error) {
	switch goarch {
	case "amd64":
		// Node.js uses "x64" instead of Go's "amd64"
		return "x64", nil
	case "arm64":
		return "arm64", nil
	default:
		return "", fmt.Errorf("node plugin: unsupported architecture %q (supported: amd64, arm64)", goarch)
	}
}
