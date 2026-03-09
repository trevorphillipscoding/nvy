// Package plugins defines the Plugin interface that every language runtime must implement.
// Adding a new language is as simple as:
//  1. Create plugins/<lang>/<lang>.go implementing Plugin
//  2. Add a blank import in plugins/all/all.go
package plugins

// DownloadSpec describes how to fetch and verify a runtime archive.
// Plugins return this from Resolve(); the install command does the rest.
type DownloadSpec struct {
	// URL is the direct HTTPS download URL for the archive.
	URL string

	// SHA256 is the expected hex-encoded SHA-256 hash of the downloaded file.
	// Set this when the hash is known at resolve time (e.g. hardcoded release hashes).
	SHA256 string

	// ChecksumURL is a URL to fetch the checksum from when SHA256 is not pre-known.
	// If ChecksumFilename is empty, the response body is treated as a raw hex SHA-256.
	// If ChecksumFilename is set, the response is parsed as a SHASUMS256-style file.
	ChecksumURL string

	// ChecksumFilename is the entry name to look up inside a SHASUMS256-style file.
	// Example: "node-v20.11.1-linux-x64.tar.gz"
	// Leave empty when ChecksumURL points directly to a single hex hash.
	ChecksumFilename string

	// StripComponents strips this many leading path components during extraction,
	// equivalent to tar(1)'s --strip-components flag.
	//
	// Both Go and Node tarballs have a single top-level directory:
	//   go1.22.1.linux-amd64.tar.gz      → go/bin/go         (strip 1)
	//   node-v20.11.1-linux-x64.tar.gz   → node-v.../bin/node (strip 1)
	StripComponents int
}

// Plugin is the interface every language runtime installer must satisfy.
// Keep implementations small — a typical plugin is ~60 lines.
type Plugin interface {
	// Name returns the canonical plugin identifier (e.g. "go", "node").
	Name() string

	// Aliases returns alternative names that route to this plugin (e.g. "golang", "nodejs").
	// Aliases are case-sensitive and must not conflict with other plugin names.
	Aliases() []string

	// LatestVersion returns the latest available full version string matching
	// prefix (e.g. "22" → "22.13.1", "3.13" → "3.13.2"). Called by the install
	// command when the user supplies a partial version. Plugins may use goos/goarch
	// to filter platform-specific releases.
	LatestVersion(prefix, goos, goarch string) (string, error)

	// Resolve returns a DownloadSpec for the given full version on the given platform.
	// version is always a complete version string (e.g. "22.13.1", not "22").
	// goos/goarch mirror runtime.GOOS / runtime.GOARCH values ("linux"/"darwin", "amd64"/"arm64").
	Resolve(version, goos, goarch string) (*DownloadSpec, error)
}
