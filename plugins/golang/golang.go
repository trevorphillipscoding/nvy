// Package golang implements the nvy plugin for the Go programming language runtime.
// It downloads official Go tarballs from dl.google.com and verifies them using
// the per-file SHA-256 checksums published alongside each release.
package golang

import (
	"fmt"

	"github.com/trevorphillipscoding/nvy/plugins"
)

const (
	downloadBase = "https://dl.google.com/go"
)

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
func (g *goPlugin) Resolve(version, goos, goarch string) (*plugins.DownloadSpec, error) {
	os, err := normalizeOS(goos)
	if err != nil {
		return nil, err
	}
	arch, err := normalizeArch(goarch)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("go%s.%s-%s.tar.gz", version, os, arch)
	url := fmt.Sprintf("%s/%s", downloadBase, filename)

	return &plugins.DownloadSpec{
		URL: url,
		// The .sha256 file contains a single hex-encoded SHA-256 hash, no filename prefix.
		ChecksumURL:      url + ".sha256",
		ChecksumFilename: "", // plain mode: response body is the raw hex hash
		StripComponents:  1,  // archive has a top-level "go/" directory to strip
	}, nil
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
