package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/trevorphillipscoding/nvy/internal/env"
	"github.com/trevorphillipscoding/nvy/internal/shim"
	"github.com/trevorphillipscoding/nvy/plugins"
)

var localCmd = &cobra.Command{
	Use:   "local <tool> <version>",
	Short: "Pin a runtime version for the current directory",
	Long: `Write a .<tool>-version file in the current directory to pin a runtime version.

When you run a managed binary (e.g. "go", "node") from this directory or any
subdirectory, nvy will use the pinned version instead of the global default.

The version file is named .<tool>-version (e.g. .go-version, .node-version).
nvy walks up the directory tree to find it, so subdirectories inherit the pin.

Examples:
  nvy local go 1.22.1        # pin Go in this directory
  nvy local go@1.21.0        # same with @ syntax
  nvy local node 20.11.1

The runtime must already be installed. Run "nvy install <tool> <version>" first.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runLocal,
}

func runLocal(_ *cobra.Command, args []string) error {
	tool, version, err := parseToolVersion(args)
	if err != nil {
		return err
	}

	p, err := plugins.Get(tool)
	if err != nil {
		return err
	}
	tool = p.Name()

	// Verify the requested version is actually installed before pinning it.
	installDir := env.RuntimeDir(tool, version)
	if _, statErr := os.Stat(installDir); statErr != nil {
		return fmt.Errorf("%s %s is not installed — run: nvy install %s %s", tool, version, tool, version)
	}

	filename := "." + tool + "-version"
	if err := os.WriteFile(filename, []byte(version+"\n"), 0644); err != nil {
		return fmt.Errorf("writing %s: %w", filename, err)
	}

	cwd, _ := os.Getwd()
	fmt.Printf("pinned %s %s in %s\n", tool, version, cwd)
	fmt.Printf("  written to %s\n", filename)

	// Show what the global version is, so the user knows what they're overriding.
	if globalV, _ := shim.ResolveVersion(tool); globalV != "" && globalV != version {
		fmt.Printf("  (overrides global: %s)\n", globalV)
	}
	return nil
}
