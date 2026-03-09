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
	"github.com/trevorphillipscoding/nvy/internal/verutil"
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

	// Resolve partial versions (e.g. "22" → "22.13.1", "3.13" → "3.13.2") via
	// the plugin's LatestVersion before calling Resolve with the full version.
	ver := rawVer
	if verutil.IsPartial(rawVer) {
		ver, err = p.LatestVersion(rawVer, env.OS(), env.Arch())
		if err != nil {
			return fmt.Errorf("resolving latest %s %s: %w", tool, rawVer, err)
		}
	}
	ver = verutil.Normalize(ver)

	spec, err := p.Resolve(ver, env.OS(), env.Arch())
	if err != nil {
		return fmt.Errorf("resolving %s %s: %w", tool, ver, err)
	}

	installDir := env.RuntimeDir(tool, ver)
	if _, statErr := os.Stat(installDir); statErr == nil {
		fmt.Printf("already installed: %s %s\n", tool, ver)
		fmt.Printf("  location: %s\n", installDir)
		fmt.Printf("  to activate: nvy global %s %s\n", tool, ver)
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
	defer func() { _ = os.RemoveAll(tmpDir) }()

	archivePath := filepath.Join(tmpDir, "archive.tar.gz")

	// ── Step 1: Download ────────────────────────────────────────────────────
	fmt.Printf("downloading %s %s\n", tool, ver)
	fmt.Printf("  from %s\n", spec.URL)
	if err := fetch.Download(spec.URL, archivePath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// ── Step 2: Verify checksum ─────────────────────────────────────────────
	sha256, err := fetch.ResolveChecksum(spec.SHA256, spec.ChecksumURL, spec.ChecksumFilename)
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

	fmt.Printf("installed %s %s\n", tool, ver)
	fmt.Printf("  run: nvy global %s %s\n", tool, ver)
	return nil
}
