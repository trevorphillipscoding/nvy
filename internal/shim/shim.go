// Package shim implements the version-resolution and exec logic used when the
// nvy binary is invoked via a symlink named after a managed tool (e.g. "go", "node").
//
// How it works:
//
//  1. "nvy global go 1.22.1" creates ~/.nvy/shims/go → /path/to/nvy  (symlink)
//  2. User runs "go build" → shell resolves to ~/.nvy/shims/go → nvy binary
//  3. main() sees os.Args[0] base == "go", calls shim.Run("go", args)
//  4. This package resolves which version to use (local → global) and
//     uses syscall.Exec to fully replace the process with the real binary —
//     no subprocess, no signal wrapping, zero overhead.
//
// Version resolution order:
//
//  1. .<tool>-version file, searched from cwd up to filesystem root
//  2. Global version from ~/.nvy/state/global.json
//  3. Error with a helpful message
package shim

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/trevorphillipscoding/nvy/internal/env"
	"github.com/trevorphillipscoding/nvy/internal/state"
	"github.com/trevorphillipscoding/nvy/internal/verutil"
)

// Run resolves the active version for binary's owning tool, then replaces the
// current process with the real binary via syscall.Exec. It never returns on
// success; on failure it prints to stderr and calls os.Exit(1).
func Run(binary string, args []string) {
	tool, ok := state.LookupShim(binary)
	if !ok {
		fmt.Fprintf(os.Stderr,
			"nvy: no shim registered for %q\n  run: nvy global <tool> <version>\n", binary)
		os.Exit(1)
	}

	version, err := ResolveVersion(tool)
	if err != nil {
		fmt.Fprintln(os.Stderr, "nvy:", err)
		os.Exit(1)
	}

	binPath := filepath.Join(env.RuntimeBinDir(tool, version), binary)
	if _, err := os.Stat(binPath); err != nil {
		fmt.Fprintf(os.Stderr,
			"nvy: %s not found for %s %s\n  run: nvy install %s %s\n",
			binary, tool, version, tool, version)
		os.Exit(1)
	}

	// syscall.Exec replaces this process entirely — the shim disappears.
	// The real binary inherits stdin/stdout/stderr, all fds, env, and signals directly.
	//
	// Pass binPath (not the bare binary name) as argv[0] so that interpreters
	// like Python use it to locate their standard library. When argv[0] is a
	// bare name (e.g. "python"), Python searches PATH and may find an unrelated
	// system installation first.
	if err := syscall.Exec(binPath, append([]string{binPath}, args...), os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "nvy: exec %s: %v\n", binPath, err)
		os.Exit(1)
	}
}

// ResolveVersion determines which version of tool to use.
// Exported so cmd/local.go can use it to display the currently active version.
func ResolveVersion(tool string) (string, error) {
	// 1. Local version file (.<tool>-version), walking up from cwd.
	if cwd, err := os.Getwd(); err == nil {
		if v := findLocalVersion(tool, cwd); v != "" {
			return resolveToInstalled(tool, v), nil
		}
	}

	// 2. Global version.
	if v, ok := state.GetGlobal(tool); ok {
		return resolveToInstalled(tool, v), nil
	}

	return "", fmt.Errorf(
		"no version configured for %s\n"+
			"  set global: nvy global %s <version>\n"+
			"  set local:  nvy local %s <version>",
		tool, tool, tool,
	)
}

// resolveToInstalled maps a possibly-partial version string (e.g. "3.13") to
// the best matching version already installed on disk (e.g. "3.13.2").
// For full versions it falls through to verutil.Normalize unchanged.
// If no installed match is found for a partial version, we still normalize so
// the subsequent stat check produces a clear "not installed" error.
func resolveToInstalled(tool, v string) string {
	base := strings.SplitN(v, "+", 2)[0]
	if strings.Count(base, ".") < 2 {
		if best := env.FindBestInstalled(tool, base); best != "" {
			return best
		}
	}
	return verutil.Normalize(v)
}

// FindLocalVersion walks up from dir looking for a .<tool>-version file.
// Returns "" if none is found. Exported for use in cmd/list.go.
func FindLocalVersion(tool, dir string) string {
	return findLocalVersion(tool, dir)
}

// findLocalVersion walks up the directory tree from dir, looking for .<tool>-version.
// Returns the trimmed version string, or "" if not found before reaching the root.
func findLocalVersion(tool, dir string) string {
	filename := "." + tool + "-version"
	for {
		data, err := os.ReadFile(filepath.Join(dir, filename))
		if err == nil {
			v := strings.TrimSpace(string(data))
			if v != "" {
				return v
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // filesystem root reached
		}
		dir = parent
	}
	return ""
}
