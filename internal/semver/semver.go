// Package semver provides nvy's single global version semantics.
//
// Supported references:
//   - major            (e.g. "1")
//   - major.minor      (e.g. "1.25")
//   - major.minor.patch (e.g. "1.25.4")
//
// Build metadata, prerelease tags, and language-specific variants are not
// supported by design to keep version behavior consistent across all runtimes.
package semver

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Reference is a requested version with 1 to 3 numeric components.
type Reference struct {
	Major int
	Minor int
	Patch int
	Parts int
}

// Version is a strict three-part semantic version.
type Version struct {
	Major int
	Minor int
	Patch int
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// ParseReference parses version references in nvy's global format.
func ParseReference(input string) (Reference, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return Reference{}, fmt.Errorf("version is required")
	}
	parts := strings.Split(trimmed, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return Reference{}, fmt.Errorf("invalid version %q: expected major, major.minor, or major.minor.patch", input)
	}

	vals := [3]int{}
	for i, p := range parts {
		if p == "" {
			return Reference{}, fmt.Errorf("invalid version %q: empty component", input)
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return Reference{}, fmt.Errorf("invalid version %q: non-numeric component %q", input, p)
		}
		vals[i] = n
	}

	return Reference{Major: vals[0], Minor: vals[1], Patch: vals[2], Parts: len(parts)}, nil
}

// ParseVersion parses an exact semantic version (major.minor.patch).
func ParseVersion(input string) (Version, error) {
	ref, err := ParseReference(input)
	if err != nil {
		return Version{}, err
	}
	if ref.Parts != 3 {
		return Version{}, fmt.Errorf("invalid exact version %q: expected major.minor.patch", input)
	}
	return Version{Major: ref.Major, Minor: ref.Minor, Patch: ref.Patch}, nil
}

// Compare returns -1 if a < b, 0 if a == b, and 1 if a > b.
func Compare(a, b Version) int {
	if a.Major != b.Major {
		if a.Major < b.Major {
			return -1
		}
		return 1
	}
	if a.Minor != b.Minor {
		if a.Minor < b.Minor {
			return -1
		}
		return 1
	}
	if a.Patch != b.Patch {
		if a.Patch < b.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// SortStringsDesc sorts exact semantic versions in descending order.
// Invalid versions are kept but always sorted after valid versions.
func SortStringsDesc(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		vi, ei := ParseVersion(versions[i])
		vj, ej := ParseVersion(versions[j])
		switch {
		case ei == nil && ej == nil:
			return Compare(vi, vj) > 0
		case ei == nil:
			return true
		case ej == nil:
			return false
		default:
			return versions[i] > versions[j]
		}
	})
}
