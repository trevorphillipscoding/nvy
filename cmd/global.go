package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/trevorphillipscoding/nvy/internal/env"
	"github.com/trevorphillipscoding/nvy/internal/state"
	"github.com/trevorphillipscoding/nvy/internal/verutil"
	"github.com/trevorphillipscoding/nvy/plugins"
)

var globalCmd = &cobra.Command{
	Use:   "global <tool> <version>",
	Short: "Set the global active version for a runtime",
	Long: `Activate a globally installed runtime version.

This creates shim symlinks in ~/.nvy/shims/ that point to the nvy binary.
When you run "go" or "node", nvy resolves the correct version and execs it.
The selection is saved to ~/.nvy/state/global.json.

Examples:
  nvy global go 1.22.1
  nvy global go@1.22.1
  nvy global node 20.11.1

The runtime must already be installed. Run "nvy install <tool> <version>" first.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runGlobal,
}

func runGlobal(_ *cobra.Command, args []string) error {
	tool, version, err := parseToolVersion(args)
	if err != nil {
		return err
	}

	p, err := plugins.Get(tool)
	if err != nil {
		return err
	}
	tool = p.Name() // normalise alias

	ver := version
	if verutil.IsPartial(version) {
		if best := env.FindBestInstalled(tool, version); best != "" {
			ver = best
		} else {
			return fmt.Errorf("%s %s.* is not installed — run: nvy install %s %s", tool, version, tool, version)
		}
	}

	runtimeBinDir := env.RuntimeBinDir(tool, ver)
	if _, statErr := os.Stat(runtimeBinDir); statErr != nil {
		return fmt.Errorf("%s %s is not installed — run: nvy install %s %s", tool, ver, tool, ver)
	}

	// The nvy binary itself acts as the shim. When invoked as "go" or "npm",
	// it reads os.Args[0], resolves the version, and syscall.Exec's the real binary.
	nvyExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolving nvy binary path: %w", err)
	}

	nvyBinDir := env.ShimsDir()
	if err := os.MkdirAll(nvyBinDir, 0755); err != nil {
		return fmt.Errorf("creating %s: %w", nvyBinDir, err)
	}

	created, err := createShims(runtimeBinDir, nvyBinDir, nvyExe, tool)
	if err != nil {
		return fmt.Errorf("creating shims: %w", err)
	}

	if err := state.SetGlobal(tool, ver); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Printf("now using %s %s\n", tool, ver)
	if len(created) > 0 {
		fmt.Printf("  binaries: %s\n", strings.Join(created, ", "))
	}

	// Warn when ~/.nvy/bin is not yet in PATH — common first-time gotcha.
	if !nvyBinInPath(nvyBinDir) {
		fmt.Printf("\n  PATH not configured. Add this to your shell profile:\n")
		fmt.Printf("    export PATH=\"%s:$PATH\"\n", nvyBinDir)
	}
	return nil
}

// createShims creates a symlink in nvyBinDir for each executable in runtimeBinDir.
// Each symlink points to nvyExe (the nvy binary), not directly to the runtime binary.
// This is the shim mechanism: when invoked as "go", nvy detects the name and
// resolves the correct version at runtime.
func createShims(runtimeBinDir, nvyBinDir, nvyExe, tool string) ([]string, error) {
	entries, err := os.ReadDir(runtimeBinDir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", runtimeBinDir, err)
	}

	var created []string

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		// Only shim executables (skip data files, headers, etc. in bin/).
		if info.Mode()&0111 == 0 {
			continue
		}

		dst := filepath.Join(nvyBinDir, e.Name())
		_ = os.Remove(dst) // replace any existing symlink or file
		if err := os.Symlink(nvyExe, dst); err != nil {
			return nil, fmt.Errorf("creating shim for %s: %w", e.Name(), err)
		}
		created = append(created, e.Name())
	}

	// Record binary → tool ownership so the shim knows which version file to check.
	if err := state.RegisterShims(tool, created); err != nil {
		return nil, fmt.Errorf("registering shim owners: %w", err)
	}
	return created, nil
}

// nvyBinInPath reports whether dir appears in the current PATH.
func nvyBinInPath(dir string) bool {
	for _, p := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		if p == dir {
			return true
		}
	}
	return false
}
