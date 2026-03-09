package shim_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trevorphillipscoding/nvy/internal/env"
	"github.com/trevorphillipscoding/nvy/internal/shim"
	"github.com/trevorphillipscoding/nvy/internal/state"
)

func TestFindLocalVersion_Found(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "project", "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write version file in the parent dir; search starts from the subdirectory.
	if err := os.WriteFile(filepath.Join(dir, "project", ".go-version"), []byte("1.22.1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := shim.FindLocalVersion("go", subdir)
	if got != "1.22.1" {
		t.Errorf("FindLocalVersion = %q; want 1.22.1", got)
	}
}

func TestFindLocalVersion_NotFound(t *testing.T) {
	dir := t.TempDir()
	got := shim.FindLocalVersion("go", dir)
	if got != "" {
		t.Errorf("FindLocalVersion = %q; want empty string", got)
	}
}

func TestFindLocalVersion_BlankFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("  \n  "), 0644); err != nil {
		t.Fatal(err)
	}
	got := shim.FindLocalVersion("go", dir)
	if got != "" {
		t.Errorf("FindLocalVersion with blank file = %q; want empty string", got)
	}
}

func TestResolveVersion_NoVersion(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())
	// Chdir to an isolated temp dir so no .go-version file is found walking up.
	t.Chdir(t.TempDir())

	_, err := shim.ResolveVersion("go")
	if err == nil {
		t.Error("expected error when no version configured, got nil")
	}
}

func TestResolveVersion_GlobalVersion(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())
	// Chdir to an isolated temp dir so local version file doesn't interfere.
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(env.RuntimeDir("go", "1.22.1"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := state.SetGlobal("go", "1.22.1"); err != nil {
		t.Fatal(err)
	}

	got, err := shim.ResolveVersion("go")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "1.22.1" {
		t.Errorf("ResolveVersion = %q; want 1.22.1", got)
	}
}

func TestResolveVersion_LocalFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NVY_DIR", tmp)
	if err := os.MkdirAll(env.RuntimeDir("go", "1.21.0"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmp, ".go-version"), []byte("1.21.0"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(tmp)

	got, err := shim.ResolveVersion("go")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "1.21.0" {
		t.Errorf("ResolveVersion = %q; want 1.21.0", got)
	}
}

func TestResolveVersion_LocalOverridesGlobal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NVY_DIR", tmp)
	if err := os.MkdirAll(env.RuntimeDir("go", "1.22.1"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(env.RuntimeDir("go", "1.21.0"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set global to a different version.
	if err := state.SetGlobal("go", "1.22.1"); err != nil {
		t.Fatal(err)
	}

	// Write a local version file.
	if err := os.WriteFile(filepath.Join(tmp, ".go-version"), []byte("1.21.0"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Chdir(tmp)

	got, err := shim.ResolveVersion("go")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	// Local version should take precedence.
	if got != "1.21.0" {
		t.Errorf("ResolveVersion = %q; want 1.21.0 (local should override global)", got)
	}
}

func TestResolveVersion_PartialUsesSameResolver(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("NVY_DIR", tmp)
	t.Chdir(tmp)

	for _, v := range []string{"1.25.1", "1.25.3", "1.24.9"} {
		if err := os.MkdirAll(env.RuntimeDir("go", v), 0755); err != nil {
			t.Fatal(err)
		}
	}

	if err := state.SetGlobal("go", "1.25"); err != nil {
		t.Fatal(err)
	}

	got, err := shim.ResolveVersion("go")
	if err != nil {
		t.Fatalf("ResolveVersion: %v", err)
	}
	if got != "1.25.3" {
		t.Fatalf("ResolveVersion = %q; want 1.25.3", got)
	}
}
