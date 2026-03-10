package golang

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/trevorphillipscoding/nvy/internal/semver"
)

func TestNormalizeOS(t *testing.T) {
	cases := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"linux", "linux", false},
		{"darwin", "darwin", false},
		{"windows", "", true},
		{"freebsd", "", true},
	}
	for _, c := range cases {
		got, err := normalizeOS(c.input)
		if c.wantErr {
			if err == nil {
				t.Errorf("normalizeOS(%q): expected error, got nil", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeOS(%q): unexpected error: %v", c.input, err)
			continue
		}
		if got != c.want {
			t.Errorf("normalizeOS(%q) = %q; want %q", c.input, got, c.want)
		}
	}
}

func TestNormalizeArch(t *testing.T) {
	cases := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"amd64", "amd64", false},
		{"arm64", "arm64", false},
		{"386", "", true},
		{"mips", "", true},
	}
	for _, c := range cases {
		got, err := normalizeArch(c.input)
		if c.wantErr {
			if err == nil {
				t.Errorf("normalizeArch(%q): expected error, got nil", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeArch(%q): unexpected error: %v", c.input, err)
			continue
		}
		if got != c.want {
			t.Errorf("normalizeArch(%q) = %q; want %q", c.input, got, c.want)
		}
	}
}

func TestFetchStableGoVersions(t *testing.T) {
	releases := []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}{
		{Version: "go1.24.1", Stable: true},
		{Version: "go1.24.0", Stable: true},
		{Version: "go1.23.5", Stable: true},
		{Version: "go1.22.12", Stable: true},
		{Version: "go1.22.0", Stable: true},   // archived version
		{Version: "go1.21.13", Stable: true},  // archived version
		{Version: "go1.22rc1", Stable: false}, // pre-release, must be excluded
	}
	body, _ := json.Marshal(releases)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	versions, err := fetchStableGoVersions()
	if err != nil {
		t.Fatalf("fetchStableGoVersions: %v", err)
	}

	// Latest version within 1.24 branch
	resolved, err := semver.Resolve("1.24", versions)
	if err != nil {
		t.Fatalf("Resolve 1.24: %v", err)
	}
	if resolved != "1.24.1" {
		t.Errorf("Resolve(1.24) = %q; want 1.24.1", resolved)
	}

	// Archived version must also be resolvable by exact version
	resolved, err = semver.Resolve("1.22.0", versions)
	if err != nil {
		t.Fatalf("Resolve 1.22.0: %v", err)
	}
	if resolved != "1.22.0" {
		t.Errorf("Resolve(1.22.0) = %q; want 1.22.0", resolved)
	}

	// Archived minor branch
	resolved, err = semver.Resolve("1.21", versions)
	if err != nil {
		t.Fatalf("Resolve 1.21: %v", err)
	}
	if resolved != "1.21.13" {
		t.Errorf("Resolve(1.21) = %q; want 1.21.13", resolved)
	}
}

func TestFetchStableGoVersions_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, err := fetchStableGoVersions()
	if err == nil {
		t.Error("expected error for server 500, got nil")
	}
}

func TestFetchStableGoVersions_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, err := fetchStableGoVersions()
	if err == nil {
		t.Error("expected error for bad JSON, got nil")
	}
}

func TestAvailableVersions(t *testing.T) {
	releases := []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}{
		{Version: "go1.22.3", Stable: true},
	}
	body, _ := json.Marshal(releases)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	p := New()
	versions, err := p.AvailableVersions("linux", "amd64")
	if err != nil {
		t.Fatalf("AvailableVersions: %v", err)
	}
	ver, err := semver.Resolve("1.22", versions)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if ver != "1.22.3" {
		t.Errorf("resolved version = %q; want 1.22.3", ver)
	}
}

func TestResolve_FullVersion(t *testing.T) {
	p := New()
	spec, err := p.Resolve("1.22.3", "linux", "amd64")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.Contains(spec.URL, "1.22.3") {
		t.Errorf("URL should contain version, got %q", spec.URL)
	}
}

func TestResolve_UnsupportedArch(t *testing.T) {
	p := New()
	_, err := p.Resolve("1.22.1", "linux", "mips")
	if err == nil {
		t.Error("expected error for unsupported arch, got nil")
	}
}
