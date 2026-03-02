package node

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestFindLatestNodeVersion(t *testing.T) {
	releases := []struct {
		Version string `json:"version"`
	}{
		{Version: "v22.13.1"},
		{Version: "v22.13.0"},
		{Version: "v20.18.2"},
		{Version: "v20.18.1"},
	}
	body, _ := json.Marshal(releases)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	cases := []struct {
		prefix  string
		want    string
		wantErr bool
	}{
		{"22", "22.13.1", false},
		{"20", "20.18.2", false},
		{"18", "", true},
	}
	for _, c := range cases {
		got, err := findLatestNodeVersion(c.prefix)
		if c.wantErr {
			if err == nil {
				t.Errorf("findLatestNodeVersion(%q): expected error, got %q", c.prefix, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("findLatestNodeVersion(%q): unexpected error: %v", c.prefix, err)
			continue
		}
		if got != c.want {
			t.Errorf("findLatestNodeVersion(%q) = %q; want %q", c.prefix, got, c.want)
		}
	}
}

func TestFindLatestNodeVersion_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, err := findLatestNodeVersion("22")
	if err == nil {
		t.Error("expected error for server 500, got nil")
	}
}

func TestFindLatestNodeVersion_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, err := findLatestNodeVersion("22")
	if err == nil {
		t.Error("expected error for bad JSON, got nil")
	}
}

func TestResolve_PartialVersion(t *testing.T) {
	releases := []struct {
		Version string `json:"version"`
	}{
		{Version: "v22.13.1"},
	}
	body, _ := json.Marshal(releases)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	p := New()
	spec, err := p.Resolve("22", "linux", "amd64")
	if err != nil {
		t.Fatalf("Resolve partial version: %v", err)
	}
	if spec.ResolvedVersion != "22.13.1" {
		t.Errorf("ResolvedVersion = %q; want 22.13.1", spec.ResolvedVersion)
	}
	if !strings.Contains(spec.URL, "22.13.1") {
		t.Errorf("URL should contain resolved version, got %q", spec.URL)
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
