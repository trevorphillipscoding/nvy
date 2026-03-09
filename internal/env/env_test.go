package env_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/trevorphillipscoding/nvy/internal/env"
)

func TestNvyDir_Default(t *testing.T) {
	t.Setenv("NVY_DIR", "")
	got := env.NvyDir()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".nvy")
	if got != want {
		t.Errorf("NvyDir() = %q; want %q", got, want)
	}
}

func TestNvyDir_Override(t *testing.T) {
	t.Setenv("NVY_DIR", "/custom/nvy")
	if got := env.NvyDir(); got != "/custom/nvy" {
		t.Errorf("NvyDir() = %q; want /custom/nvy", got)
	}
}

func TestPathFunctions(t *testing.T) {
	t.Setenv("NVY_DIR", "/base")
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"RuntimesDir", env.RuntimesDir(), "/base/runtimes"},
		{"RuntimeDir", env.RuntimeDir("go", "1.22.1"), "/base/runtimes/go/1.22.1"},
		{"RuntimeBinDir", env.RuntimeBinDir("go", "1.22.1"), "/base/runtimes/go/1.22.1/bin"},
		{"ShimsDir", env.ShimsDir(), "/base/shims"},
		{"StateDir", env.StateDir(), "/base/state"},
		{"GlobalStatePath", env.GlobalStatePath(), "/base/state/global.json"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q; want %q", c.name, c.got, c.want)
		}
	}
}

func TestOS(t *testing.T) {
	if got := env.OS(); got != runtime.GOOS {
		t.Errorf("OS() = %q; want %q", got, runtime.GOOS)
	}
}

func TestArch(t *testing.T) {
	if got := env.Arch(); got != runtime.GOARCH {
		t.Errorf("Arch() = %q; want %q", got, runtime.GOARCH)
	}
}

func TestMkTempDir(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	d1, err := env.MkTempDir()
	if err != nil {
		t.Fatalf("MkTempDir: %v", err)
	}
	defer func() { _ = os.RemoveAll(d1) }()

	if _, err := os.Stat(d1); err != nil {
		t.Errorf("temp dir not created: %v", err)
	}

	d2, err := env.MkTempDir()
	if err != nil {
		t.Fatalf("MkTempDir second call: %v", err)
	}
	defer func() { _ = os.RemoveAll(d2) }()

	if d1 == d2 {
		t.Error("MkTempDir returned same directory twice")
	}
}

func TestAtomicInstall(t *testing.T) {
	base := t.TempDir()
	t.Setenv("NVY_DIR", base)

	src := filepath.Join(base, "src")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "file"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(base, "dst")
	if err := env.AtomicInstall(src, dst); err != nil {
		t.Fatalf("AtomicInstall: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dst, "file"))
	if err != nil {
		t.Fatalf("reading installed file: %v", err)
	}
	if string(data) != "content" {
		t.Errorf("content = %q; want content", data)
	}
}

func TestAtomicInstall_Replace(t *testing.T) {
	base := t.TempDir()
	t.Setenv("NVY_DIR", base)

	dst := filepath.Join(base, "dst")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "old"), []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(base, "src")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "new"), []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := env.AtomicInstall(src, dst); err != nil {
		t.Fatalf("AtomicInstall replace: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "new")); err != nil {
		t.Error("new file should exist after replace")
	}
	if _, err := os.Stat(filepath.Join(dst, "old")); err == nil {
		t.Error("old file should not exist after replace")
	}
}
