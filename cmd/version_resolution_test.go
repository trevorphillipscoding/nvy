package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/trevorphillipscoding/nvy/plugins"
)

type testPlugin struct {
	versions []string
}

func (p *testPlugin) Name() string      { return "test" }
func (p *testPlugin) Aliases() []string { return nil }
func (p *testPlugin) AvailableVersions(_, _ string) ([]string, error) {
	return p.versions, nil
}
func (p *testPlugin) Resolve(version, _, _ string) (*plugins.DownloadSpec, error) {
	return &plugins.DownloadSpec{URL: "https://example.invalid/" + version}, nil
}

func TestResolveInstallVersion_UsesSharedResolver(t *testing.T) {
	p := &testPlugin{versions: []string{"1.24.4", "1.25.0", "1.25.3"}}

	got, err := resolveInstallVersion(p, "1")
	if err != nil {
		t.Fatalf("resolveInstallVersion: %v", err)
	}
	if got != "1.25.3" {
		t.Fatalf("got %q, want 1.25.3", got)
	}

	got, err = resolveInstallVersion(p, "1.25")
	if err != nil {
		t.Fatalf("resolveInstallVersion: %v", err)
	}
	if got != "1.25.3" {
		t.Fatalf("got %q, want 1.25.3", got)
	}

	got, err = resolveInstallVersion(p, "1.24.4")
	if err != nil {
		t.Fatalf("resolveInstallVersion: %v", err)
	}
	if got != "1.24.4" {
		t.Fatalf("got %q, want 1.24.4", got)
	}
}

func TestResolveInstalledVersion_UsesSharedResolver(t *testing.T) {
	base := t.TempDir()
	t.Setenv("NVY_DIR", base)

	for _, v := range []string{"1.24.4", "1.25.0", "1.25.3"} {
		if err := os.MkdirAll(filepath.Join(base, "runtimes", "go", v), 0755); err != nil {
			t.Fatal(err)
		}
	}

	got, err := resolveInstalledVersion("go", "1")
	if err != nil {
		t.Fatalf("resolveInstalledVersion: %v", err)
	}
	if got != "1.25.3" {
		t.Fatalf("got %q, want 1.25.3", got)
	}

	if _, err := resolveInstalledVersion("go", "1.24.5"); err == nil {
		t.Fatal("expected error for missing exact version")
	}
}
