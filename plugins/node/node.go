// Package node implements the nvy plugin for Node.js.
// It downloads official Node.js tarballs from nodejs.org and verifies them using
// the SHASUMS256.txt file published alongside each release.
package node

import (
	"fmt"

	"github.com/trevorphillipscoding/nvy/plugins"
)

const (
	distBase = "https://nodejs.org/dist"
)

func init() {
	plugins.Register(New())
}

type nodePlugin struct{}

// New returns the Node.js plugin. Called by init(); exposed for testing.
func New() plugins.Plugin { return &nodePlugin{} }

func (n *nodePlugin) Name() string { return "node" }

func (n *nodePlugin) Aliases() []string { return []string{"nodejs", "node.js"} }

// Resolve builds the download spec for a Node.js release tarball.
//
// Official naming convention:
//
//	node-v<version>-<os>-<arch>.tar.gz
//	SHASUMS256.txt  ← multi-entry file; we look up our filename inside it
//
// Example: node-v20.11.1-linux-x64.tar.gz
func (n *nodePlugin) Resolve(version, goos, goarch string) (*plugins.DownloadSpec, error) {
	os, err := normalizeOS(goos)
	if err != nil {
		return nil, err
	}
	arch, err := normalizeArch(goarch)
	if err != nil {
		return nil, err
	}

	// Node uses "v" prefix in both the URL path and the archive filename.
	filename := fmt.Sprintf("node-v%s-%s-%s.tar.gz", version, os, arch)
	url := fmt.Sprintf("%s/v%s/%s", distBase, version, filename)
	checksumURL := fmt.Sprintf("%s/v%s/SHASUMS256.txt", distBase, version)

	return &plugins.DownloadSpec{
		URL:              url,
		ChecksumURL:      checksumURL,
		ChecksumFilename: filename, // SHASUMS256 mode: look up this filename in the file
		StripComponents:  1,        // archive has a top-level "node-v<version>-<os>-<arch>/" directory
	}, nil
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
