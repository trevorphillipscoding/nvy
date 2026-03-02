package golang

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

func TestFindLatestGoVersion(t *testing.T) {
	releases := []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}{
		{Version: "go1.24.1", Stable: true},
		{Version: "go1.24.0", Stable: true},
		{Version: "go1.23.5", Stable: true},
		{Version: "go1.22.12", Stable: true},
		{Version: "go1.22rc1", Stable: false},
	}
	body, _ := json.Marshal(releases)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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
		{"1.24", "1.24.1", false},
		{"1.23", "1.23.5", false},
		{"1.22", "1.22.12", false},
		{"1.25", "", true},
	}
	for _, c := range cases {
		got, err := findLatestGoVersion(c.prefix)
		if c.wantErr {
			if err == nil {
				t.Errorf("findLatestGoVersion(%q): expected error, got %q", c.prefix, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("findLatestGoVersion(%q): unexpected error: %v", c.prefix, err)
			continue
		}
		if got != c.want {
			t.Errorf("findLatestGoVersion(%q) = %q; want %q", c.prefix, got, c.want)
		}
	}
}

func TestFindLatestGoVersion_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, err := findLatestGoVersion("1.22")
	if err == nil {
		t.Error("expected error for server 500, got nil")
	}
}

func TestFindLatestGoVersion_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, err := findLatestGoVersion("1.22")
	if err == nil {
		t.Error("expected error for bad JSON, got nil")
	}
}

func TestResolve_PartialVersion(t *testing.T) {
	releases := []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}{
		{Version: "go1.22.3", Stable: true},
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
	spec, err := p.Resolve("1.22", "linux", "amd64")
	if err != nil {
		t.Fatalf("Resolve partial version: %v", err)
	}
	if spec.ResolvedVersion != "1.22.3" {
		t.Errorf("ResolvedVersion = %q; want 1.22.3", spec.ResolvedVersion)
	}
	if !strings.Contains(spec.URL, "1.22.3") {
		t.Errorf("URL should contain resolved version, got %q", spec.URL)
	}
}

func TestResolve_UnsupportedArch(t *testing.T) {
	p := New()
	_, err := p.Resolve("1.22.1", "linux", "mips")
	if err == nil {
		t.Error("expected error for unsupported arch, got nil")
	}
}
