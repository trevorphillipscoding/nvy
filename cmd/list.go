package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/trevorphillipscoding/nvy/internal/env"
	"github.com/trevorphillipscoding/nvy/internal/shim"
	"github.com/trevorphillipscoding/nvy/internal/state"
	"github.com/trevorphillipscoding/nvy/plugins"
)

var listCmd = &cobra.Command{
	Use:   "list [tool]",
	Short: "List installed runtime versions",
	Long: `List all installed runtime versions, or just those for a specific tool.

  *  marks the active global version
  »  marks a local version pinned for the current directory (via nvy local)

Examples:
  nvy list
  nvy list go
  nvy list node`,
	Aliases: []string{"ls"},
	Args:    cobra.MaximumNArgs(1),
	RunE:    runList,
}

func runList(_ *cobra.Command, args []string) error {
	globals, err := state.AllGlobals()
	if err != nil {
		return fmt.Errorf("reading global state: %w", err)
	}

	cwd, _ := os.Getwd()

	if len(args) == 1 {
		p, err := plugins.Get(args[0])
		if err != nil {
			return err
		}
		return printTool(p.Name(), globals, cwd)
	}

	allPlugins := plugins.All()
	if len(allPlugins) == 0 {
		fmt.Println("no plugins registered")
		return nil
	}

	extra := extraToolsOnDisk(allPlugins)

	printed := false
	for _, p := range allPlugins {
		if printed {
			fmt.Println()
		}
		if err := printTool(p.Name(), globals, cwd); err != nil {
			return err
		}
		printed = true
	}
	for _, name := range extra {
		if printed {
			fmt.Println()
		}
		if err := printTool(name, globals, cwd); err != nil {
			return err
		}
		printed = true
	}

	if !printed {
		fmt.Println("nothing installed — run: nvy install <tool> <version>")
	}
	return nil
}

// printTool lists installed versions for one tool.
// *  = active global
// »  = local pin for the current directory (from .<tool>-version)
func printTool(tool string, globals map[string]string, cwd string) error {
	toolDir := filepath.Join(env.RuntimesDir(), tool)
	versions, err := listVersions(toolDir)
	if err != nil {
		fmt.Printf("%s  (none installed)\n", tool)
		return nil
	}
	if len(versions) == 0 {
		fmt.Printf("%s  (none installed)\n", tool)
		return nil
	}

	globalVersion := globals[tool]
	localVersion := shim.FindLocalVersion(tool, cwd)

	fmt.Printf("%s\n", tool)
	for _, v := range versions {
		var markers []string
		if v == globalVersion {
			markers = append(markers, "global")
		}
		if v == localVersion {
			markers = append(markers, "local")
		}

		prefix := "  "
		switch v {
		case localVersion:
			prefix = "» "
		case globalVersion:
			prefix = "* "
		}

		if len(markers) > 0 {
			fmt.Printf("  %s%s  (%s)\n", prefix, v, strings.Join(markers, ", "))
		} else {
			fmt.Printf("  %s%s\n", prefix, v)
		}
	}
	return nil
}

// listVersions returns installed versions for a tool, sorted newest first.
func listVersions(toolDir string) ([]string, error) {
	entries, err := os.ReadDir(toolDir)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			versions = append(versions, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	return versions, nil
}

// extraToolsOnDisk returns tool directories in ~/.nvy/runtimes/ with no registered plugin.
func extraToolsOnDisk(known []plugins.Plugin) []string {
	knownNames := map[string]bool{}
	for _, p := range known {
		knownNames[p.Name()] = true
	}
	entries, err := os.ReadDir(env.RuntimesDir())
	if err != nil {
		return nil
	}
	var extra []string
	for _, e := range entries {
		if e.IsDir() && !knownNames[e.Name()] {
			extra = append(extra, e.Name())
		}
	}
	sort.Strings(extra)
	return extra
}
