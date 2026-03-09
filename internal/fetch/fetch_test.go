package fetch

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTestServer starts a TLS test server, replaces the package httpClient for
// the duration of fn, and restores it afterwards.
func withTestServer(t *testing.T, handler http.Handler, fn func(url string)) {
	t.Helper()
	ts := httptest.NewTLSServer(handler)
	t.Cleanup(ts.Close)

	orig := httpClient
	httpClient = ts.Client()
	t.Cleanup(func() { httpClient = orig })

	fn(ts.URL)
}

func TestVerifySHA256_Match(t *testing.T) {
	content := []byte("hello, world")
	tmp := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(tmp, content, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	if err := VerifySHA256(tmp, hex.EncodeToString(sum[:])); err != nil {
		t.Errorf("VerifySHA256: %v", err)
	}
}

func TestVerifySHA256_Mismatch(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(tmp, []byte("actual content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(tmp, strings.Repeat("a", 64)); err == nil {
		t.Error("expected mismatch error, got nil")
	}
}

func TestVerifySHA256_InvalidHashLength(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(tmp, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := VerifySHA256(tmp, "tooshort"); err == nil {
		t.Error("expected invalid hash length error, got nil")
	}
}

func TestDownload_RejectsHTTP(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "out")
	if err := Download("http://example.com/file", tmp); err == nil {
		t.Error("expected error for HTTP URL, got nil")
	}
}

func TestBytes_RejectsHTTP(t *testing.T) {
	if _, err := Bytes("http://example.com/checksums"); err == nil {
		t.Error("expected error for HTTP URL, got nil")
	}
}

func TestDownload_Success(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("file content"))
	}), func(url string) {
		tmp := filepath.Join(t.TempDir(), "out")
		if err := Download(url, tmp); err != nil {
			t.Fatalf("Download: %v", err)
		}
		data, _ := os.ReadFile(tmp)
		if string(data) != "file content" {
			t.Errorf("got %q; want %q", data, "file content")
		}
	})
}

func TestDownload_NotFound(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}), func(url string) {
		tmp := filepath.Join(t.TempDir(), "out")
		if err := Download(url, tmp); err == nil {
			t.Error("expected error for 404, got nil")
		}
	})
}

func TestBytes_Success(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("checksum data"))
	}), func(url string) {
		data, err := Bytes(url)
		if err != nil {
			t.Fatalf("Bytes: %v", err)
		}
		if string(data) != "checksum data" {
			t.Errorf("got %q; want %q", data, "checksum data")
		}
	})
}

func TestBytes_NotFound(t *testing.T) {
	withTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}), func(url string) {
		if _, err := Bytes(url); err == nil {
			t.Error("expected error for 404, got nil")
		}
	})
}

func TestProgressWriter(t *testing.T) {
	// With known total.
	pw := &progressWriter{total: 1024 * 1024}
	n, err := pw.Write(make([]byte, 512*1024))
	if err != nil {
		t.Errorf("Write: %v", err)
	}
	if n != 512*1024 {
		t.Errorf("Write returned %d; want %d", n, 512*1024)
	}
	pw.finish()

	// Without known total (Content-Length: -1).
	pw2 := &progressWriter{total: -1}
	if _, err := pw2.Write(make([]byte, 100)); err != nil {
		t.Errorf("Write (no total): %v", err)
	}
	pw2.finish()
}
