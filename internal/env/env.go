// Package env resolves all filesystem paths used by nvy and provides
// platform-detection helpers. Everything lives under ~/.nvy/:
//
//	~/.nvy/
//	├── runtimes/<tool>/<version>/   installed runtime trees
//	├── shims/                       symlinks to the nvy binary, named after each tool binary
//	├── state/
//	│   ├── global.json              active global versions
//	│   └── owners.json              binary → tool mapping
//	└── tmp/                         staging area for in-progress installs
package env

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// NvyDir returns the root nvy directory (~/.nvy).
func NvyDir() string {
	if override := os.Getenv("NVY_DIR"); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic("nvy: cannot determine home directory: " + err.Error())
	}
	return filepath.Join(home, ".nvy")
}

// RuntimesDir returns ~/.nvy/runtimes.
func RuntimesDir() string { return filepath.Join(NvyDir(), "runtimes") }

// RuntimeDir returns ~/.nvy/runtimes/<tool>/<version>.
func RuntimeDir(tool, version string) string {
	return filepath.Join(RuntimesDir(), tool, version)
}

// RuntimeBinDir returns ~/.nvy/runtimes/<tool>/<version>/bin.
func RuntimeBinDir(tool, version string) string {
	return filepath.Join(RuntimeDir(tool, version), "bin")
}

// ShimsDir returns ~/.nvy/shims — the directory users add to their PATH.
// Each file here is a symlink to the nvy binary named after a managed tool binary
// (e.g. "go", "node", "npm"). When invoked, nvy detects the name and resolves
// the correct runtime version before exec'ing the real binary.
func ShimsDir() string { return filepath.Join(NvyDir(), "shims") }

// StateDir returns ~/.nvy/state.
func StateDir() string { return filepath.Join(NvyDir(), "state") }

// GlobalStatePath returns ~/.nvy/state/global.json.
func GlobalStatePath() string { return filepath.Join(StateDir(), "global.json") }

// OS returns the current operating system name in Go's convention (linux, darwin).
func OS() string { return runtime.GOOS }

// Arch returns the current CPU architecture in Go's convention (amd64, arm64).
func Arch() string { return runtime.GOARCH }

// MkTempDir creates a temporary staging directory inside ~/.nvy/tmp/.
// The caller is responsible for calling os.RemoveAll on the returned path.
func MkTempDir() (string, error) {
	tmpRoot := filepath.Join(NvyDir(), "tmp")
	if err := os.MkdirAll(tmpRoot, 0700); err != nil {
		return "", fmt.Errorf("creating nvy tmp dir: %w", err)
	}
	// Use a random suffix so concurrent installs don't collide.
	suffix, err := randomHex(8)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(tmpRoot, suffix)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating staging dir: %w", err)
	}
	return dir, nil
}

// AtomicInstall moves src directory to dst, replacing any existing installation.
//
// Strategy:
//  1. Move existing dst → dst.old.<random> (if present)
//  2. Move src → dst
//  3. Remove the old backup
//
// This ensures dst is always either the old or the new version; never a partial state.
// Both paths must be on the same filesystem (both live under ~/.nvy/).
func AtomicInstall(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("creating parent dir: %w", err)
	}

	// If a previous version exists, stash it aside before replacing.
	var oldBackup string
	if _, err := os.Lstat(dst); err == nil {
		suffix, err := randomHex(8)
		if err != nil {
			return err
		}
		oldBackup = dst + ".old." + suffix
		if err := os.Rename(dst, oldBackup); err != nil {
			return fmt.Errorf("stashing old install: %w", err)
		}
	}

	// Move the freshly extracted tree into place.
	if err := os.Rename(src, dst); err != nil {
		// Try to restore the backup on failure.
		if oldBackup != "" {
			_ = os.Rename(oldBackup, dst)
		}
		return fmt.Errorf("moving install into place: %w", err)
	}

	// Clean up the old backup asynchronously — not critical.
	if oldBackup != "" {
		go func() { _ = os.RemoveAll(oldBackup) }()
	}
	return nil
}

// InstalledVersions returns installed runtime directory names for a tool.
// These are expected to be exact semantic versions (major.minor.patch).
func InstalledVersions(tool string) ([]string, error) {
	dir := filepath.Join(RuntimesDir(), tool)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(entries))

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		versions = append(versions, e.Name())
	}
	return versions, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating random suffix: %w", err)
	}
	return hex.EncodeToString(b), nil
}
