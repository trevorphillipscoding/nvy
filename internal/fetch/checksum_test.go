package fetch

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseHashFile_Found(t *testing.T) {
	data := []byte(`# SHA-256 checksums
abc123def456  node-v20.11.1-linux-x64.tar.gz
789abcdef012  node-v20.11.1-darwin-arm64.tar.gz
`)
	hash, err := ParseHashFile(data, "node-v20.11.1-linux-x64.tar.gz")
	if err != nil {
		t.Fatalf("ParseHashFile: %v", err)
	}
	if hash != "abc123def456" {
		t.Errorf("hash = %q; want abc123def456", hash)
	}
}

func TestParseHashFile_NotFound(t *testing.T) {
	data := []byte("abc123  file-a.tar.gz\ndef456  file-b.tar.gz\n")
	_, err := ParseHashFile(data, "file-c.tar.gz")
	if err == nil {
		t.Error("expected error for missing filename, got nil")
	}
}

func TestParseHashFile_SkipsBlankAndComments(t *testing.T) {
	data := []byte("\n# comment\n\nabc123  target.tar.gz\n")
	hash, err := ParseHashFile(data, "target.tar.gz")
	if err != nil {
		t.Fatalf("ParseHashFile: %v", err)
	}
	if hash != "abc123" {
		t.Errorf("hash = %q; want abc123", hash)
	}
}

func TestResolveChecksum_PreKnownHash(t *testing.T) {
	hash, err := ResolveChecksum("deadbeef", "", "")
	if err != nil {
		t.Fatalf("ResolveChecksum: %v", err)
	}
	if hash != "deadbeef" {
		t.Errorf("hash = %q; want deadbeef", hash)
	}
}

func TestResolveChecksum_NoHashOrURL(t *testing.T) {
	_, err := ResolveChecksum("", "", "")
	if err == nil {
		t.Error("expected error when neither hash nor URL provided, got nil")
	}
}

func TestResolveChecksum_RawHexURL(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("  abc123def456  \n"))
	}), func(url string) {
		hash, err := ResolveChecksum("", url, "")
		if err != nil {
			t.Fatalf("ResolveChecksum: %v", err)
		}
		if hash != "abc123def456" {
			t.Errorf("hash = %q; want abc123def456", hash)
		}
	})
}

func TestResolveChecksum_SHASUMSFile(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("abc123  file-a.tar.gz\ndef456  file-b.tar.gz\n"))
	}), func(url string) {
		hash, err := ResolveChecksum("", url, "file-b.tar.gz")
		if err != nil {
			t.Fatalf("ResolveChecksum: %v", err)
		}
		if hash != "def456" {
			t.Errorf("hash = %q; want def456", hash)
		}
	})
}

func TestResolveChecksum_FetchError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	orig := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = orig }()

	_, err := ResolveChecksum("", srv.URL, "")
	if err == nil {
		t.Error("expected error for 404, got nil")
	}
}
