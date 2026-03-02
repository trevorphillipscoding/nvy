// Package python implements the nvy plugin for CPython.
// It downloads pre-built CPython tarballs from the python-build-standalone
// project (https://github.com/indygreg/python-build-standalone) and verifies
// them using the SHA256SUMS file published alongside each release.
package python

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/trevorphillipscoding/nvy/plugins"
)

const (
	downloadBase = "https://github.com/indygreg/python-build-standalone/releases/download"
	releasesAtom = "https://github.com/indygreg/python-build-standalone/releases.atom"
)

// tagPattern matches 8-digit date tags used by python-build-standalone releases.
var tagPattern = regexp.MustCompile(`/releases/tag/(\d{8})`)

func init() {
	plugins.Register(New())
}

type pythonPlugin struct{}

// New returns the Python plugin. Called by init(); exposed for testing.
func New() plugins.Plugin { return &pythonPlugin{} }

func (p *pythonPlugin) Name() string { return "python" }

func (p *pythonPlugin) Aliases() []string { return []string{"python3", "py"} }

// Resolve builds the download spec for a CPython release from python-build-standalone.
//
// Version formats accepted:
//
//	3.12.5             — discovers the latest release tag via the GitHub API
//	3.12.5+20240814    — uses the given build tag directly (no network call)
//
// Official naming convention:
//
//	cpython-<version>+<tag>-<triple>-install_only.tar.gz
//	SHA256SUMS  ← multi-entry file; we look up our filename inside it
//
// Example: cpython-3.12.5+20240814-x86_64-unknown-linux-gnu-install_only.tar.gz
func (p *pythonPlugin) Resolve(version, goos, goarch string) (*plugins.DownloadSpec, error) {
	triple, err := normalizeTriple(goos, goarch)
	if err != nil {
		return nil, err
	}

	pyVersion, tag, err := parseVersion(version)
	if err != nil {
		return nil, err
	}

	if tag == "" {
		tag, err = findReleaseTag(pyVersion, triple)
		if err != nil {
			return nil, err
		}
	}

	filename := fmt.Sprintf("cpython-%s+%s-%s-install_only.tar.gz", pyVersion, tag, triple)
	url := fmt.Sprintf("%s/%s/%s", downloadBase, tag, filename)
	checksumURL := fmt.Sprintf("%s/%s/SHA256SUMS", downloadBase, tag)

	return &plugins.DownloadSpec{
		URL:              url,
		ChecksumURL:      checksumURL,
		ChecksumFilename: filename, // SHASUMS256 mode: look up this filename in SHA256SUMS
		StripComponents:  1,        // archive top-level is "python/"
	}, nil
}

// parseVersion splits an optional +tag suffix from the version string.
//
//	"3.12.5"          → ("3.12.5", "", nil)
//	"3.12.5+20240814" → ("3.12.5", "20240814", nil)
func parseVersion(version string) (pyVersion, tag string, err error) {
	parts := strings.SplitN(version, "+", 2)
	pyVersion = strings.TrimSpace(parts[0])
	if pyVersion == "" {
		return "", "", fmt.Errorf("python plugin: empty version string")
	}
	if len(parts) == 2 {
		tag = strings.TrimSpace(parts[1])
	}
	return pyVersion, tag, nil
}

// normalizeTriple maps GOOS/GOARCH to the target triple used in python-build-standalone filenames.
func normalizeTriple(goos, goarch string) (string, error) {
	switch goos + "/" + goarch {
	case "linux/amd64":
		return "x86_64-unknown-linux-gnu", nil
	case "linux/arm64":
		return "aarch64-unknown-linux-gnu", nil
	case "darwin/amd64":
		return "x86_64-apple-darwin", nil
	case "darwin/arm64":
		return "aarch64-apple-darwin", nil
	default:
		return "", fmt.Errorf("python plugin: unsupported platform %s/%s (supported: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)", goos, goarch)
	}
}

// findReleaseTag fetches the project's Atom release feed (no auth required, not
// subject to the GitHub API rate limit) and probes each recent release tag with
// a HEAD request until it finds one that contains the requested CPython build.
func findReleaseTag(pyVersion, triple string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get(releasesAtom)
	if err != nil {
		return "", fmt.Errorf("python plugin: fetching release feed: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("python plugin: reading release feed: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("python plugin: release feed returned %s", resp.Status)
	}

	// Collect unique tags in the order they appear in the feed (newest first).
	seen := make(map[string]bool)
	var tags []string
	for _, m := range tagPattern.FindAllSubmatch(body, -1) {
		tag := string(m[1])
		if !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	if len(tags) == 0 {
		return "", fmt.Errorf("python plugin: no release tags found in release feed")
	}

	// Probe each tag with a cheap HEAD request.
	for _, tag := range tags {
		filename := fmt.Sprintf("cpython-%s+%s-%s-install_only.tar.gz", pyVersion, tag, triple)
		url := fmt.Sprintf("%s/%s/%s", downloadBase, tag, filename)
		r, err := client.Head(url)
		if err == nil {
			r.Body.Close()
			if r.StatusCode == http.StatusOK {
				return tag, nil
			}
		}
	}

	return "", fmt.Errorf("python plugin: no release found for Python %s on %s in the latest %d releases; specify a build tag to install older versions (e.g. %s+20240814)", pyVersion, triple, len(tags), pyVersion)
}
