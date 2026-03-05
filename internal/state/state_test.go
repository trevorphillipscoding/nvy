package state

import (
	"os"
	"slices"
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

func TestDeleteGlobal(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	SetGlobal("go", "1.22.1")   //nolint:errcheck
	SetGlobal("node", "20.0.0") //nolint:errcheck

	if err := DeleteGlobal("go"); err != nil {
		t.Fatalf("DeleteGlobal: %v", err)
	}

	_, ok := GetGlobal("go")
	if ok {
		t.Error("go should be absent after DeleteGlobal")
	}

	// node must still be present.
	v, ok := GetGlobal("node")
	if !ok || v != "20.0.0" {
		t.Errorf("node should still be set, got %q %v", v, ok)
	}
}

func TestDeleteGlobal_Noop(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	// Deleting a tool that was never set should not error.
	if err := DeleteGlobal("python"); err != nil {
		t.Errorf("DeleteGlobal on unset tool: %v", err)
	}
}

func TestRegisterAndLookupShims(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	if err := RegisterShims("go", []string{"go", "gofmt"}); err != nil {
		t.Fatalf("RegisterShims: %v", err)
	}

	tool, ok := LookupShim("go")
	if !ok || tool != "go" {
		t.Errorf("LookupShim(go) = %q, %v; want go, true", tool, ok)
	}

	tool, ok = LookupShim("gofmt")
	if !ok || tool != "go" {
		t.Errorf("LookupShim(gofmt) = %q, %v; want go, true", tool, ok)
	}

	_, ok = LookupShim("node")
	if ok {
		t.Error("LookupShim(node) should return false for unregistered binary")
	}
}

func TestUnregisterShims(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	RegisterShims("go", []string{"go", "gofmt"})   //nolint:errcheck
	RegisterShims("node", []string{"node", "npm"}) //nolint:errcheck

	removed, err := UnregisterShims("go")
	if err != nil {
		t.Fatalf("UnregisterShims: %v", err)
	}
	if len(removed) != 2 {
		t.Errorf("expected 2 removed, got %d: %v", len(removed), removed)
	}
	if !slices.Contains(removed, "go") || !slices.Contains(removed, "gofmt") {
		t.Errorf("expected [go gofmt] in removed list, got %v", removed)
	}

	// go shims should be gone.
	if _, ok := LookupShim("go"); ok {
		t.Error("go shim should be removed")
	}
	if _, ok := LookupShim("gofmt"); ok {
		t.Error("gofmt shim should be removed")
	}

	// node shims should still be present.
	if tool, ok := LookupShim("node"); !ok || tool != "node" {
		t.Errorf("node shim should still be present, got %q %v", tool, ok)
	}
}

func TestUnregisterShims_Noop(t *testing.T) {
	t.Setenv("NVY_DIR", t.TempDir())

	// Unregistering a tool with no shims should return nil, nil.
	removed, err := UnregisterShims("python")
	if err != nil {
		t.Errorf("UnregisterShims on unregistered tool: %v", err)
	}
	if len(removed) != 0 {
		t.Errorf("expected empty removed list, got %v", removed)
	}
}
