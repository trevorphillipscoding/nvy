package python

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/trevorphillipscoding/nvy/internal/verutil"
)

func TestParseVersion(t *testing.T) {
	cases := []struct {
		input     string
		wantVer   string
		wantTag   string
		wantError bool
	}{
		{"3.12.5", "3.12.5", "", false},
		{"3.12.5+20240814", "3.12.5", "20240814", false},
		{"3.12", "3.12", "", false},
		{"3", "3", "", false},
		{"", "", "", true},
		{"   ", "", "", true},
	}
	for _, c := range cases {
		ver, tag, err := parseVersion(c.input)
		if c.wantError {
			if err == nil {
				t.Errorf("parseVersion(%q): expected error, got nil", c.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseVersion(%q): unexpected error: %v", c.input, err)
			continue
		}
		if ver != c.wantVer {
			t.Errorf("parseVersion(%q) ver = %q; want %q", c.input, ver, c.wantVer)
		}
		if tag != c.wantTag {
			t.Errorf("parseVersion(%q) tag = %q; want %q", c.input, tag, c.wantTag)
		}
	}
}

func TestParseVersionTuple(t *testing.T) {
	cases := []struct {
		input string
		want  [3]int
	}{
		{"3.12.5", [3]int{3, 12, 5}},
		{"3.12.0", [3]int{3, 12, 0}},
		{"1.0.0", [3]int{1, 0, 0}},
		{"10.20.30", [3]int{10, 20, 30}},
		{"3.12", [3]int{3, 12, 0}},
		{"3", [3]int{3, 0, 0}},
	}
	for _, c := range cases {
		got := verutil.ParseTuple(c.input)
		if got != c.want {
			t.Errorf("parseVersionTuple(%q) = %v; want %v", c.input, got, c.want)
		}
	}
}

func TestCmpVersionTuple(t *testing.T) {
	cases := []struct {
		a, b [3]int
		want int
	}{
		{[3]int{3, 12, 5}, [3]int{3, 12, 5}, 0},
		{[3]int{3, 12, 5}, [3]int{3, 12, 4}, 1},
		{[3]int{3, 12, 4}, [3]int{3, 12, 5}, -1},
		{[3]int{3, 13, 0}, [3]int{3, 12, 9}, 1},
		{[3]int{4, 0, 0}, [3]int{3, 99, 99}, 1},
		{[3]int{1, 0, 0}, [3]int{2, 0, 0}, -1},
		{[3]int{0, 0, 0}, [3]int{0, 0, 0}, 0},
	}
	for _, c := range cases {
		got := verutil.CmpTuple(c.a, c.b)
		if got != c.want {
			t.Errorf("cmpVersionTuple(%v, %v) = %d; want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestNormalizeTriple(t *testing.T) {
	cases := []struct {
		goos    string
		goarch  string
		want    string
		wantErr bool
	}{
		{"linux", "amd64", "x86_64-unknown-linux-gnu", false},
		{"linux", "arm64", "aarch64-unknown-linux-gnu", false},
		{"darwin", "amd64", "x86_64-apple-darwin", false},
		{"darwin", "arm64", "aarch64-apple-darwin", false},
		{"windows", "amd64", "", true},
		{"linux", "mips", "", true},
		{"plan9", "arm64", "", true},
	}
	for _, c := range cases {
		got, err := normalizeTriple(c.goos, c.goarch)
		if c.wantErr {
			if err == nil {
				t.Errorf("normalizeTriple(%q, %q): expected error, got nil", c.goos, c.goarch)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeTriple(%q, %q): unexpected error: %v", c.goos, c.goarch, err)
			continue
		}
		if got != c.want {
			t.Errorf("normalizeTriple(%q, %q) = %q; want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func makeFindLatestServer(t *testing.T, triple string) *httptest.Server {
	t.Helper()
	assets := []struct {
		Name string `json:"name"`
	}{
		{Name: fmt.Sprintf("cpython-3.13.1+20240814-%s-install_only.tar.gz", triple)},
		{Name: fmt.Sprintf("cpython-3.12.8+20240814-%s-install_only.tar.gz", triple)},
		{Name: fmt.Sprintf("cpython-3.12.5+20240101-%s-install_only.tar.gz", triple)},
		{Name: "cpython-3.13.1+20240814-aarch64-apple-darwin-install_only.tar.gz"}, // different triple
		{Name: "not-a-valid-asset-name.tar.gz"},
	}
	releases := []struct {
		Assets interface{} `json:"assets"`
	}{{Assets: assets}}
	body, _ := json.Marshal(releases)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
}

func TestFindLatest(t *testing.T) {
	triple := "x86_64-unknown-linux-gnu"
	srv := makeFindLatestServer(t, triple)
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	pyVersion, tag, err := findLatest("3", triple)
	if err != nil {
		t.Fatalf("findLatest: %v", err)
	}
	// Should pick the highest version: 3.13.1 > 3.12.8 > 3.12.5
	if pyVersion != "3.13.1" {
		t.Errorf("pyVersion = %q; want 3.13.1", pyVersion)
	}
	if tag != "20240814" {
		t.Errorf("tag = %q; want 20240814", tag)
	}
}

func TestFindLatest_Minor(t *testing.T) {
	triple := "x86_64-unknown-linux-gnu"
	srv := makeFindLatestServer(t, triple)
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	pyVersion, tag, err := findLatest("3.12", triple)
	if err != nil {
		t.Fatalf("findLatest 3.12: %v", err)
	}
	if pyVersion != "3.12.8" {
		t.Errorf("pyVersion = %q; want 3.12.8", pyVersion)
	}
	if tag != "20240814" {
		t.Errorf("tag = %q; want 20240814", tag)
	}
}

func TestFindLatest_NotFound(t *testing.T) {
	triple := "x86_64-unknown-linux-gnu"
	srv := makeFindLatestServer(t, triple)
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, _, err := findLatest("4", triple)
	if err == nil {
		t.Error("expected error for version 4.*, got nil")
	}
}

func TestFindLatest_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, _, err := findLatest("3", "x86_64-unknown-linux-gnu")
	if err == nil {
		t.Error("expected error for server 500, got nil")
	}
}

func TestFindLatest_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	_, _, err := findLatest("3", "x86_64-unknown-linux-gnu")
	if err == nil {
		t.Error("expected error for bad JSON, got nil")
	}
}

func TestFindReleaseTag_AtomServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	orig := releasesAtom
	releasesAtom = srv.URL
	defer func() { releasesAtom = orig }()

	_, err := findReleaseTag("3.12.5", "x86_64-unknown-linux-gnu")
	if err == nil {
		t.Error("expected error for atom server 500, got nil")
	}
}

func TestFindReleaseTag_NoTags(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<feed><entry>no tags here</entry></feed>"))
	}))
	defer srv.Close()

	orig := releasesAtom
	releasesAtom = srv.URL
	defer func() { releasesAtom = orig }()

	_, err := findReleaseTag("3.12.5", "x86_64-unknown-linux-gnu")
	if err == nil {
		t.Error("expected error when no tags found, got nil")
	}
}

func TestLatestVersion(t *testing.T) {
	triple := "x86_64-unknown-linux-gnu"
	assets := []struct {
		Name string `json:"name"`
	}{
		{Name: fmt.Sprintf("cpython-3.12.8+20240814-%s-install_only.tar.gz", triple)},
	}
	releases := []struct {
		Assets interface{} `json:"assets"`
	}{{Assets: assets}}
	body, _ := json.Marshal(releases)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	orig := releasesAPI
	releasesAPI = srv.URL
	defer func() { releasesAPI = orig }()

	p := New()
	ver, err := p.LatestVersion("3.12", "linux", "amd64")
	if err != nil {
		t.Fatalf("LatestVersion: %v", err)
	}
	if ver != "3.12.8" {
		t.Errorf("LatestVersion = %q; want 3.12.8", ver)
	}
}
