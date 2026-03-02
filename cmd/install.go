package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/trevorphillipscoding/nvy/internal/archive"
	"github.com/trevorphillipscoding/nvy/internal/env"
	"github.com/trevorphillipscoding/nvy/internal/fetch"
	"github.com/trevorphillipscoding/nvy/internal/version"
	"github.com/trevorphillipscoding/nvy/plugins"
)

var installCmd = &cobra.Command{
	Use:   "install <tool> <version>",
	Short: "Download and install a runtime version",
	Long: `Download, verify, and install a runtime version.

Examples:
  nvy install go 1.22.1
  nvy install go@1.22.1
  nvy install node 20.11.1
  nvy install node@20.11.1

The runtime is installed to ~/.nvy/runtimes/<tool>/<version>/.
Run "nvy global <tool> <version>" afterwards to activate it.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runInstall,
}

func runInstall(_ *cobra.Command, args []string) error {
	tool, rawVer, err := parseToolVersion(args)
	if err != nil {
		return err
	}

	p, err := plugins.Get(tool)
	if err != nil {
		return err
	}
	tool = p.Name() // normalise alias → canonical name (e.g. "golang" → "go")

	spec, err := p.Resolve(rawVer, env.OS(), env.Arch())
	if err != nil {
		return fmt.Errorf("resolving %s %s: %w", tool, rawVer, err)
	}

	// Use the plugin's resolved version if it resolved a partial input (e.g. "3.12" → "3.12.8"),
	// otherwise fall back to standard normalization (e.g. "1.26" → "1.26.0").
	resolvedVer := spec.ResolvedVersion
	if resolvedVer == "" {
		resolvedVer = version.Normalize(rawVer)
	}

	installDir := env.RuntimeDir(tool, resolvedVer)
	if _, statErr := os.Stat(installDir); statErr == nil {
		fmt.Printf("already installed: %s %s\n", tool, resolvedVer)
		fmt.Printf("  location: %s\n", installDir)
		fmt.Printf("  to activate: nvy global %s %s\n", tool, resolvedVer)
		return nil
	}

	// Enforce HTTPS — belt-and-suspenders check on top of the fetch package's own check.
	if !strings.HasPrefix(spec.URL, "https://") {
		return fmt.Errorf("plugin returned a non-HTTPS URL (%s) — refusing to proceed", spec.URL)
	}

	// All work happens inside a temp dir under ~/.nvy/tmp/ so we stay on the
	// same filesystem as the final destination, enabling atomic os.Rename.
	tmpDir, err := env.MkTempDir()
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir) // clean up on any exit path

	archivePath := filepath.Join(tmpDir, "archive.tar.gz")

	// ── Step 1: Download ────────────────────────────────────────────────────
	fmt.Printf("downloading %s %s\n", tool, resolvedVer)
	fmt.Printf("  from %s\n", spec.URL)
	if err := fetch.Download(spec.URL, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// ── Step 2: Verify checksum ─────────────────────────────────────────────
	sha256, err := resolveChecksum(spec)
	if err != nil {
		return fmt.Errorf("fetching checksum: %w", err)
	}
	fmt.Printf("verifying checksum\n")
	if err := fetch.VerifySHA256(archivePath, sha256); err != nil {
		return err
	}
	fmt.Printf("  checksum OK\n")

	// ── Step 3: Extract ─────────────────────────────────────────────────────
	fmt.Printf("extracting\n")
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := archive.ExtractTarGz(archivePath, extractDir, spec.StripComponents); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// ── Step 4: Atomic install ──────────────────────────────────────────────
	if err := env.AtomicInstall(extractDir, installDir); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	fmt.Printf("installed %s %s\n", tool, resolvedVer)
	fmt.Printf("  run: nvy global %s %s\n", tool, resolvedVer)
	return nil
}

// resolveChecksum fetches or returns the expected SHA-256 hex string for a download.
//
// Resolution priority:
//  1. spec.SHA256 — pre-known hash (fastest, no network call)
//  2. spec.ChecksumURL — fetch hash from remote; if ChecksumFilename is set,
//     parse as a SHASUMS256-style file; otherwise treat the body as a raw hex hash.
func resolveChecksum(spec *plugins.DownloadSpec) (string, error) {
	if spec.SHA256 != "" {
		return spec.SHA256, nil
	}
	if spec.ChecksumURL == "" {
		return "", fmt.Errorf("plugin provided neither SHA256 nor ChecksumURL")
	}
	data, err := fetch.FetchBytes(spec.ChecksumURL)
	if err != nil {
		return "", fmt.Errorf("fetching checksum from %s: %w", spec.ChecksumURL, err)
	}
	if spec.ChecksumFilename != "" {
		return parseHashFile(data, spec.ChecksumFilename)
	}
	// Plain format: the entire response body is the hex SHA-256.
	return strings.TrimSpace(string(data)), nil
}

// parseHashFile parses a SHASUMS256-style file and returns the hash for filename.
//
// Format (each line):
//
//	<hex-sha256>  <filename>
func parseHashFile(data []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %q in checksum file", filename)
}

// parseToolVersion accepts either:
//
//	["go", "1.22.1"]     — two separate arguments
//	["go@1.22.1"]        — single argument with @ separator
//
// The version is returned trimmed of whitespace and trailing dots
// (e.g. "1.26." → "1.26"). Each plugin normalizes or resolves
// versions in its own Resolve() implementation.
func parseToolVersion(args []string) (tool, ver string, err error) {
	clean := func(v string) string {
		return strings.TrimRight(strings.TrimSpace(v), ".")
	}
	if len(args) == 2 {
		return strings.TrimSpace(args[0]), clean(args[1]), nil
	}
	// Single arg must use the tool@version form.
	parts := strings.SplitN(args[0], "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("specify a version: nvy install <tool> <version>  or  nvy install <tool>@<version>")
	}
	return strings.TrimSpace(parts[0]), clean(parts[1]), nil
}
