package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/trevorphillipscoding/nvy/internal/env"
	"github.com/trevorphillipscoding/nvy/internal/state"
	"github.com/trevorphillipscoding/nvy/internal/verutil"
	"github.com/trevorphillipscoding/nvy/plugins"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <tool> <version>",
	Short: "Remove an installed runtime version",
	Long: `Remove an installed runtime version from ~/.nvy/runtimes/.

If the version being removed is the currently active global version,
its shims are also removed and the global selection is cleared.

Examples:
  nvy uninstall go 1.22.1
  nvy uninstall go@1.22.1
  nvy uninstall node 20.11.1`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runUninstall,
}

func runUninstall(_ *cobra.Command, args []string) error {
	tool, ver, err := parseToolVersion(args)
	if err != nil {
		return err
	}

	p, err := plugins.Get(tool)
	if err != nil {
		return err
	}
	tool = p.Name() // normalise alias

	if verutil.IsPartial(ver) {
		if best := env.FindBestInstalled(tool, ver); best != "" {
			ver = best
		}
	}

	installDir := env.RuntimeDir(tool, ver)
	if _, err := os.Stat(installDir); err != nil {
		return fmt.Errorf("%s %s is not installed", tool, ver)
	}

	// If this version is the active global, clean up shims and state.
	if activeVer, isActive := state.GetGlobal(tool); isActive && activeVer == ver {
		binaries, err := state.UnregisterShims(tool)
		if err != nil {
			return fmt.Errorf("removing shim registrations: %w", err)
		}
		shimsDir := env.ShimsDir()
		for _, bin := range binaries {
			_ = os.Remove(filepath.Join(shimsDir, bin))
		}
		if err := state.DeleteGlobal(tool); err != nil {
			return fmt.Errorf("clearing global version: %w", err)
		}
		fmt.Printf("removed shims for %s\n", tool)
	}

	if err := os.RemoveAll(installDir); err != nil {
		return fmt.Errorf("removing %s: %w", installDir, err)
	}
	fmt.Printf("uninstalled %s %s\n", tool, ver)
	return nil
}
