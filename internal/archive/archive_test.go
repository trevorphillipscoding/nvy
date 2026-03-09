package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// buildTarGz creates an in-memory tar.gz archive for testing.
func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0755,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestExtractTarGz_BasicExtraction(t *testing.T) {
	data := buildTarGz(t, map[string]string{
		"prefix/bin/tool": "#!/bin/sh\necho hello",
		"prefix/lib/foo":  "library",
	})

	src := filepath.Join(t.TempDir(), "archive.tar.gz")
	if err := os.WriteFile(src, data, 0600); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if err := ExtractTarGz(src, dest, 1); err != nil {
		t.Fatalf("ExtractTarGz: %v", err)
	}

	// After strip-1, "prefix/bin/tool" becomes "bin/tool"
	content, err := os.ReadFile(filepath.Join(dest, "bin", "tool"))
	if err != nil {
		t.Fatalf("expected bin/tool to be extracted: %v", err)
	}
	if string(content) != "#!/bin/sh\necho hello" {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestExtractTarGz_ZipSlipRejected(t *testing.T) {
	// Build an archive with a path-traversal entry.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "prefix/../../../etc/evil",
		Mode:     0644,
		Size:     5,
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("evil")); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()

	src := filepath.Join(t.TempDir(), "evil.tar.gz")
	if err := os.WriteFile(src, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	err := ExtractTarGz(src, dest, 1)
	if err == nil {
		t.Fatal("expected path-traversal to be rejected, got nil error")
	}
}

func TestStripLeadingComponents(t *testing.T) {
	cases := []struct {
		name   string
		strip  int
		expect string
	}{
		{"go/bin/go", 1, "bin/go"},
		{"go/bin/go", 0, "go/bin/go"},
		{"go/bin/go", 2, "go"},
		{"go/bin/go", 3, ""},
		{"node-v20.0.0-linux-x64/bin/node", 1, "bin/node"},
	}
	for _, c := range cases {
		got := stripLeadingComponents(c.name, c.strip)
		if got != c.expect {
			t.Errorf("stripLeadingComponents(%q, %d) = %q; want %q", c.name, c.strip, got, c.expect)
		}
	}
}
