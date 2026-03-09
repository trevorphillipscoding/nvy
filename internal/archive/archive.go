// Package archive handles extraction of runtime archives.
//
// Security: every archive entry is validated against a path-traversal attack
// (the "Zip Slip" vulnerability) before any file is written to disk.
// An entry whose resolved path escapes the destination directory is rejected
// and the entire extraction is aborted.
package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// maxFileSize caps the size of any single extracted file at 2 GB.
// This prevents decompression bombs from exhausting disk space.
const maxFileSize = 2 * 1024 * 1024 * 1024

// ExtractTarGz extracts the tar.gz archive at src into dest,
// stripping stripComponents leading path components from every entry
// (equivalent to tar(1)'s --strip-components flag).
//
// dest is created if it does not exist.
// Returns an error if any archive entry would escape dest (Zip Slip protection).
func ExtractTarGz(src, dest string, stripComponents int) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("reading gzip stream: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)

	// Resolve dest to an absolute path once; used for path-traversal checks.
	destAbs, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolving destination path: %w", err)
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading archive entry: %w", err)
		}

		// Apply --strip-components: skip entries that don't have enough depth.
		entryPath := stripLeadingComponents(hdr.Name, stripComponents)
		if entryPath == "" {
			continue // entry was entirely consumed by stripping
		}

		target := filepath.Join(dest, filepath.FromSlash(entryPath))

		// ── Zip Slip guard ──────────────────────────────────────────────────────
		// filepath.Join cleans ".." sequences; Abs makes the result absolute so
		// we can safely check containment with a prefix match.
		targetAbs, err := filepath.Abs(target)
		if err != nil {
			return fmt.Errorf("resolving entry path %q: %w", hdr.Name, err)
		}
		if !strings.HasPrefix(targetAbs, destAbs+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry %q would escape destination — aborting", hdr.Name)
		}
		// ────────────────────────────────────────────────────────────────────────

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, hdr.FileInfo().Mode()|0111); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}

		case tar.TypeReg:
			if err := extractFile(tr, target, hdr.FileInfo().Mode()); err != nil {
				return fmt.Errorf("extracting %s: %w", hdr.Name, err)
			}

		case tar.TypeSymlink:
			// Validate symlink target is not absolute and doesn't escape dest.
			if filepath.IsAbs(hdr.Linkname) {
				return fmt.Errorf("archive contains absolute symlink %q → %q — aborting", hdr.Name, hdr.Linkname)
			}
			linkTarget := filepath.Join(filepath.Dir(target), hdr.Linkname)
			linkAbs, err := filepath.Abs(linkTarget)
			if err != nil {
				return fmt.Errorf("resolving symlink %q: %w", hdr.Name, err)
			}
			if !strings.HasPrefix(linkAbs, destAbs+string(os.PathSeparator)) {
				return fmt.Errorf("symlink %q → %q escapes destination — aborting", hdr.Name, hdr.Linkname)
			}
			// Ensure parent directory exists (some tarballs order symlinks before dirs).
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("creating parent directory for symlink %s: %w", target, err)
			}
			// Remove any existing file/link at target before creating the new link.
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return fmt.Errorf("creating symlink %s: %w", target, err)
			}

		case tar.TypeLink:
			// Hard links: resolve the link target within dest.
			linkSrc := filepath.Join(dest, filepath.FromSlash(
				stripLeadingComponents(hdr.Linkname, stripComponents),
			))
			_ = os.Remove(target)
			if err := os.Link(linkSrc, target); err != nil {
				return fmt.Errorf("creating hard link %s: %w", target, err)
			}

		default:
			// Skip devices, FIFOs, etc. — not needed for language runtimes.
		}
	}
	return nil
}

// extractFile writes an individual regular file from the tar stream.
func extractFile(r io.Reader, dest string, mode os.FileMode) error {
	// Ensure the parent directory exists (some tarballs omit directory entries).
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	limited := &io.LimitedReader{R: r, N: maxFileSize + 1}
	if _, err := io.Copy(f, limited); err != nil {
		return err
	}
	if limited.N == 0 {
		return fmt.Errorf("file exceeds maximum allowed size (%d bytes) — possible decompression bomb", maxFileSize)
	}
	return nil
}

// stripLeadingComponents removes n leading slash-separated path components.
// Returns "" if name has fewer than n+1 components (entry should be skipped).
func stripLeadingComponents(name string, n int) string {
	// Normalise separators and trim leading slashes.
	name = strings.TrimLeft(filepath.ToSlash(name), "/")
	for i := 0; i < n; i++ {
		idx := strings.IndexByte(name, '/')
		if idx == -1 {
			return "" // not enough components
		}
		name = name[idx+1:]
	}
	return name
}
