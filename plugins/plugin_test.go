package plugins_test

import (
	"strings"
	"testing"

	"github.com/trevorphillipscoding/nvy/plugins"
	_ "github.com/trevorphillipscoding/nvy/plugins/all" // register all built-in plugins
	"github.com/trevorphillipscoding/nvy/plugins/golang"
	"github.com/trevorphillipscoding/nvy/plugins/node"
	"github.com/trevorphillipscoding/nvy/plugins/python"
)

func TestGoPlugin_Resolve(t *testing.T) {
	p := golang.New()

	if p.Name() != "go" {
		t.Errorf("Name() = %q; want go", p.Name())
	}

	spec, err := p.Resolve("1.22.1", "linux", "amd64")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(spec.URL, "https://") {
		t.Errorf("URL must be HTTPS, got %q", spec.URL)
	}
	if !strings.Contains(spec.URL, "go1.22.1") {
		t.Errorf("URL should contain version, got %q", spec.URL)
	}
	if !strings.Contains(spec.URL, "linux-amd64") {
		t.Errorf("URL should contain platform, got %q", spec.URL)
	}
	if spec.StripComponents != 1 {
		t.Errorf("StripComponents = %d; want 1", spec.StripComponents)
	}
	if spec.ChecksumURL == "" {
		t.Error("ChecksumURL must not be empty")
	}
}

func TestGoPlugin_UnsupportedPlatform(t *testing.T) {
	p := golang.New()
	_, err := p.Resolve("1.22.1", "plan9", "mips")
	if err == nil {
		t.Error("expected error for unsupported platform, got nil")
	}
}

func TestGoPlugin_DarwinARM64(t *testing.T) {
	p := golang.New()
	spec, err := p.Resolve("1.22.1", "darwin", "arm64")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.Contains(spec.URL, "darwin-arm64") {
		t.Errorf("URL should contain darwin-arm64, got %q", spec.URL)
	}
}

func TestNodePlugin_Resolve(t *testing.T) {
	p := node.New()

	if p.Name() != "node" {
		t.Errorf("Name() = %q; want node", p.Name())
	}

	spec, err := p.Resolve("20.11.1", "linux", "amd64")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(spec.URL, "https://") {
		t.Errorf("URL must be HTTPS, got %q", spec.URL)
	}
	// Node uses "x64" for amd64
	if !strings.Contains(spec.URL, "x64") {
		t.Errorf("URL should contain x64 for amd64, got %q", spec.URL)
	}
	if !strings.Contains(spec.URL, "v20.11.1") {
		t.Errorf("URL should contain version with v prefix, got %q", spec.URL)
	}
	if spec.ChecksumFilename == "" {
		t.Error("ChecksumFilename must be set for SHASUMS256.txt parsing")
	}
	if spec.StripComponents != 1 {
		t.Errorf("StripComponents = %d; want 1", spec.StripComponents)
	}
}

func TestNodePlugin_ArchMapping(t *testing.T) {
	p := node.New()
	cases := []struct {
		arch    string
		wantSub string
	}{
		{"amd64", "x64"},
		{"arm64", "arm64"},
	}
	for _, c := range cases {
		spec, err := p.Resolve("20.0.0", "linux", c.arch)
		if err != nil {
			t.Errorf("Resolve linux/%s: %v", c.arch, err)
			continue
		}
		if !strings.Contains(spec.URL, c.wantSub) {
			t.Errorf("URL for %s should contain %q, got %q", c.arch, c.wantSub, spec.URL)
		}
	}
}

func TestPythonPlugin_Resolve(t *testing.T) {
	p := python.New()

	if p.Name() != "python" {
		t.Errorf("Name() = %q; want python", p.Name())
	}

	// Use the explicit +tag format to avoid a network call.
	spec, err := p.Resolve("3.12.5+20240814", "linux", "amd64")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.HasPrefix(spec.URL, "https://") {
		t.Errorf("URL must be HTTPS, got %q", spec.URL)
	}
	if !strings.Contains(spec.URL, "3.12.5") {
		t.Errorf("URL should contain version, got %q", spec.URL)
	}
	if !strings.Contains(spec.URL, "x86_64-unknown-linux-gnu") {
		t.Errorf("URL should contain linux/amd64 triple, got %q", spec.URL)
	}
	if !strings.Contains(spec.URL, "20240814") {
		t.Errorf("URL should contain the build tag, got %q", spec.URL)
	}
	if spec.ChecksumFilename == "" {
		t.Error("ChecksumFilename must be set for SHA256SUMS parsing")
	}
	if spec.StripComponents != 1 {
		t.Errorf("StripComponents = %d; want 1", spec.StripComponents)
	}
}

func TestPythonPlugin_TripleMapping(t *testing.T) {
	p := python.New()
	cases := []struct {
		goos    string
		goarch  string
		wantSub string
	}{
		{"linux", "amd64", "x86_64-unknown-linux-gnu"},
		{"linux", "arm64", "aarch64-unknown-linux-gnu"},
		{"darwin", "amd64", "x86_64-apple-darwin"},
		{"darwin", "arm64", "aarch64-apple-darwin"},
	}
	for _, c := range cases {
		spec, err := p.Resolve("3.12.5+20240814", c.goos, c.goarch)
		if err != nil {
			t.Errorf("Resolve %s/%s: %v", c.goos, c.goarch, err)
			continue
		}
		if !strings.Contains(spec.URL, c.wantSub) {
			t.Errorf("URL for %s/%s should contain %q, got %q", c.goos, c.goarch, c.wantSub, spec.URL)
		}
	}
}

func TestPythonPlugin_UnsupportedPlatform(t *testing.T) {
	p := python.New()
	_, err := p.Resolve("3.12.5+20240814", "windows", "amd64")
	if err == nil {
		t.Error("expected error for unsupported platform, got nil")
	}
}

func TestPythonPlugin_ExplicitTagNoResolvedVersion(t *testing.T) {
	p := python.New()
	// When a full version+tag is provided, ResolvedVersion should be empty
	// (no resolution was needed; the install dir uses the input version as-is).
	spec, err := p.Resolve("3.12.5+20240814", "linux", "amd64")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if spec.ResolvedVersion != "" {
		t.Errorf("ResolvedVersion should be empty for explicit version+tag, got %q", spec.ResolvedVersion)
	}
}

func TestRegistry_AliasResolution(t *testing.T) {
	cases := []struct {
		alias    string
		wantName string
	}{
		{"golang", "go"},
		{"nodejs", "node"},
		{"node.js", "node"},
		{"python3", "python"},
		{"py", "python"},
	}
	for _, c := range cases {
		p, err := plugins.Get(c.alias)
		if err != nil {
			t.Errorf("Get(%q): %v", c.alias, err)
			continue
		}
		if p.Name() != c.wantName {
			t.Errorf("Get(%q).Name() = %q; want %q", c.alias, p.Name(), c.wantName)
		}
	}
}

func TestRegistry_UnknownTool(t *testing.T) {
	_, err := plugins.Get("ruby")
	if err == nil {
		t.Error("expected error for unknown tool, got nil")
	}
}

func TestRegistry_All(t *testing.T) {
	all := plugins.All()
	if len(all) == 0 {
		t.Fatal("All() returned empty slice")
	}
	// Must be sorted by canonical name.
	for i := 1; i < len(all); i++ {
		if all[i].Name() < all[i-1].Name() {
			t.Errorf("All() not sorted: %q before %q", all[i-1].Name(), all[i].Name())
		}
	}
	// Must include the three built-in plugins.
	names := make(map[string]bool, len(all))
	for _, p := range all {
		names[p.Name()] = true
	}
	for _, want := range []string{"go", "node", "python"} {
		if !names[want] {
			t.Errorf("All() missing plugin %q", want)
		}
	}
}

func TestGoPlugin_UnsupportedArch(t *testing.T) {
	p := golang.New()
	_, err := p.Resolve("1.22.1", "linux", "mips")
	if err == nil {
		t.Error("expected error for unsupported arch, got nil")
	}
}

func TestNodePlugin_Darwin(t *testing.T) {
	p := node.New()
	spec, err := p.Resolve("20.11.1", "darwin", "arm64")
	if err != nil {
		t.Fatalf("Resolve darwin/arm64: %v", err)
	}
	if !strings.Contains(spec.URL, "darwin") {
		t.Errorf("URL should contain darwin, got %q", spec.URL)
	}
}

func TestNodePlugin_UnsupportedPlatform(t *testing.T) {
	p := node.New()
	_, err := p.Resolve("20.11.1", "windows", "amd64")
	if err == nil {
		t.Error("expected error for unsupported platform, got nil")
	}
}

func TestNodePlugin_UnsupportedArch(t *testing.T) {
	p := node.New()
	_, err := p.Resolve("20.11.1", "linux", "mips")
	if err == nil {
		t.Error("expected error for unsupported arch, got nil")
	}
}

func TestPythonPlugin_ParseVersionError(t *testing.T) {
	p := python.New()
	_, err := p.Resolve("", "linux", "amd64")
	if err == nil {
		t.Error("expected error for empty version, got nil")
	}
}
