package main

import (
	"os"
	"path/filepath"

	"github.com/trevorphillipscoding/nvy/cmd"
	"github.com/trevorphillipscoding/nvy/internal/shim"

	// Register all built-in plugins before any command runs.
	_ "github.com/trevorphillipscoding/nvy/plugins/all"
)

func main() {
	// Shim detection: ~/.nvy/shims/go and ~/.nvy/shims/node are symlinks to this binary.
	// When invoked via such a symlink, os.Args[0] base is the tool name, not "nvy".
	// In that case, act as a transparent version-resolving shim rather than the CLI.
	if name := filepath.Base(os.Args[0]); name != "nvy" {
		shim.Run(name, os.Args[1:])
		return // never reached on success (syscall.Exec replaces the process)
	}

	cmd.Execute()
}
