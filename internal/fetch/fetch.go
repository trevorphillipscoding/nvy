// Package fetch handles secure file downloads and checksum verification.
//
// Security decisions:
//   - Only HTTPS URLs are accepted; plain HTTP is rejected outright.
//   - Redirects are followed but the final URL must also be HTTPS.
//   - SHA-256 is verified before the caller extracts anything.
//   - Downloads are streamed to disk (no full in-memory buffering of large archives).
package fetch

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// httpClient is a shared client with sane timeouts and TLS hardening.
// It does NOT follow redirects to HTTP — the custom CheckRedirect enforces HTTPS.
var httpClient = &http.Client{
	Timeout: 30 * time.Minute, // large binaries may be slow on constrained networks
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		// Reasonable connection and response timeouts.
		ResponseHeaderTimeout: 60 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("refusing HTTP redirect to %s — HTTPS required", req.URL)
		}
		if len(via) >= 5 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	},
}

// Download fetches url and writes the response body to destPath.
// It streams the download and displays a simple progress indicator.
// Only HTTPS URLs are accepted.
func Download(url, destPath string) error {
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("refusing to download from non-HTTPS URL: %s", url)
	}

	resp, err := httpClient.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("creating %s: %w", destPath, err)
	}
	defer f.Close()

	pw := &progressWriter{total: resp.ContentLength}
	if _, err := io.Copy(f, io.TeeReader(resp.Body, pw)); err != nil {
		return fmt.Errorf("downloading %s: %w", url, err)
	}
	pw.finish()
	return nil
}

// FetchBytes performs a GET request and returns the response body as bytes.
// Only HTTPS URLs are accepted.
func FetchBytes(url string) ([]byte, error) {
	if !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("refusing non-HTTPS URL: %s", url)
	}

	resp, err := httpClient.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %s", url, resp.Status)
	}

	// Limit to 1 MB to protect against unexpectedly large checksum files.
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

// VerifySHA256 computes the SHA-256 of the file at path and compares it to expected.
// expected must be a lowercase hex-encoded string.
func VerifySHA256(path, expected string) error {
	expected = strings.TrimSpace(strings.ToLower(expected))
	if len(expected) != 64 {
		return fmt.Errorf("invalid SHA-256 hash length: got %d chars, want 64", len(expected))
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s for verification: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("hashing %s: %w", path, err)
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		return fmt.Errorf("SHA-256 mismatch:\n  expected: %s\n  got:      %s\n\nThe downloaded file may be corrupt or tampered with. Aborting.", expected, got)
	}
	return nil
}

// progressWriter tracks bytes written and prints a compact progress line to stdout.
type progressWriter struct {
	total   int64 // -1 if unknown (no Content-Length header)
	written int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)

	mb := float64(pw.written) / (1024 * 1024)
	if pw.total > 0 {
		pct := float64(pw.written) / float64(pw.total) * 100
		fmt.Printf("\r  %.1f / %.1f MB (%.0f%%)", mb, float64(pw.total)/(1024*1024), pct)
	} else {
		fmt.Printf("\r  %.1f MB", mb)
	}
	return n, nil
}

func (pw *progressWriter) finish() {
	// Move to a new line after the progress output.
	fmt.Println()
}
