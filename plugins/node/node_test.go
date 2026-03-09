package node

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
		{"plan9", "", true},
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
		{"amd64", "x64", false},
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

func TestFetchNodeVersions(t *testing.T) {
	releases := []struct {
		Version string `json:"version"`
	}{
		{Version: "v22.13.1"},
		{Version: "v22.13.0"},
		{Version: "v20.18.2"},
		{Version: "v20.18.1"},
	}
	body, _ := json.Marshal(releases)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	versions, err := fetchNodeVersions()
	if err != nil {
		t.Fatalf("fetchNodeVersions: %v", err)
	}

	resolved, err := semver.Resolve("22", versions)
	if err != nil {
		t.Fatalf("Resolve 22: %v", err)
	}
	if resolved != "22.13.1" {
		t.Errorf("Resolve(22) = %q; want 22.13.1", resolved)
	}
}

func TestFetchNodeVersions_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, err := fetchNodeVersions()
	if err == nil {
		t.Error("expected error for server 500, got nil")
	}
}

func TestFetchNodeVersions_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, err := fetchNodeVersions()
	if err == nil {
		t.Error("expected error for bad JSON, got nil")
	}
}

func TestAvailableVersions(t *testing.T) {
	releases := []struct {
		Version string `json:"version"`
	}{
		{Version: "v22.13.1"},
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
	ver, err := semver.Resolve("22", versions)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if ver != "22.13.1" {
		t.Errorf("resolved version = %q; want 22.13.1", ver)
	}
}

func TestResolve_FullVersion(t *testing.T) {
	p := New()
	spec, err := p.Resolve("22.13.1", "linux", "amd64")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !strings.Contains(spec.URL, "22.13.1") {
		t.Errorf("URL should contain version, got %q", spec.URL)
	}
}

func TestResolve_UnsupportedArch(t *testing.T) {
	p := New()
	_, err := p.Resolve("20.11.1", "linux", "mips")
	if err == nil {
		t.Error("expected error for unsupported arch, got nil")
	}
}

func TestResolve_UnsupportedOS(t *testing.T) {
	p := New()
	_, err := p.Resolve("20.11.1", "windows", "amd64")
	if err == nil {
		t.Error("expected error for unsupported OS, got nil")
	}
}
