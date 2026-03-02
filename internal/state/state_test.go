package state

import (
	"os"
	"testing"

	"github.com/trevorphillipscoding/nvy/internal/env"
)

func TestSetAndGetGlobal(t *testing.T) {
	// Redirect ~/.nvy to a temp dir so tests don't touch real state.
	tmp := t.TempDir()
	t.Setenv("NVY_DIR", tmp)

	if err := SetGlobal("go", "1.22.1"); err != nil {
		t.Fatalf("SetGlobal: %v", err)
	}
	if err := SetGlobal("node", "20.11.1"); err != nil {
		t.Fatalf("SetGlobal: %v", err)
	}

	v, ok := GetGlobal("go")
	if !ok || v != "1.22.1" {
		t.Errorf("GetGlobal(go) = %q, %v; want 1.22.1, true", v, ok)
	}

	v, ok = GetGlobal("node")
	if !ok || v != "20.11.1" {
		t.Errorf("GetGlobal(node) = %q, %v; want 20.11.1, true", v, ok)
	}

	_, ok = GetGlobal("python")
	if ok {
		t.Error("GetGlobal for unset tool should return ok=false")
	}
}

func TestAllGlobals(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	SetGlobal("go", "1.22.1")   //nolint:errcheck
	SetGlobal("node", "20.0.0") //nolint:errcheck

	all, err := AllGlobals()
	if err != nil {
		t.Fatalf("AllGlobals: %v", err)
	}
	if all["go"] != "1.22.1" {
		t.Errorf("expected go=1.22.1, got %s", all["go"])
	}
	if all["node"] != "20.0.0" {
		t.Errorf("expected node=20.0.0, got %s", all["node"])
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	// Should not error when global.json doesn't exist yet.
	all, err := AllGlobals()
	if err != nil {
		t.Fatalf("AllGlobals on missing file: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected empty state, got %v", all)
	}
}

func TestAtomicWrite(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	SetGlobal("go", "1.22.1") //nolint:errcheck

	// Verify the file was written.
	if _, err := os.Stat(env.GlobalStatePath()); err != nil {
		t.Fatalf("global.json not created: %v", err)
	}

	// Overwrite should work cleanly.
	if err := SetGlobal("go", "1.21.0"); err != nil {
		t.Fatalf("SetGlobal overwrite: %v", err)
	}
	v, _ := GetGlobal("go")
	if v != "1.21.0" {
		t.Errorf("expected 1.21.0 after overwrite, got %s", v)
	}
}
