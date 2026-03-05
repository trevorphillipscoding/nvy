package fetch

import (
	"fmt"
	"strings"
)

// ResolveChecksum returns the expected SHA-256 hex hash for a download.
//
// Resolution priority:
//  1. sha256 — pre-known hash (fastest, no network call)
//  2. checksumURL — fetch hash from remote; if checksumFilename is set,
//     parse as a SHASUMS256-style file; otherwise treat the body as a raw hex hash.
func ResolveChecksum(sha256, checksumURL, checksumFilename string) (string, error) {
	if sha256 != "" {
		return sha256, nil
	}
	if checksumURL == "" {
		return "", fmt.Errorf("neither SHA256 hash nor checksum URL provided")
	}
	data, err := FetchBytes(checksumURL)
	if err != nil {
		return "", fmt.Errorf("fetching checksum from %s: %w", checksumURL, err)
	}
	if checksumFilename != "" {
		return ParseHashFile(data, checksumFilename)
	}
	// Plain format: the entire response body is the hex SHA-256.
	return strings.TrimSpace(string(data)), nil
}

// ParseHashFile parses a SHASUMS256-style file and returns the hash for filename.
//
// Expected format (each line):
//
//	<hex-sha256>  <filename>
//
// Lines starting with # and blank lines are skipped.
func ParseHashFile(data []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %q in checksum file", filename)
}
