// Package verutil provides shared version-string utilities.
package verutil

import (
	"strconv"
	"strings"
)

// Normalize expands short versions to full major.minor.patch form:
//
//	"20"            → "20.0.0"
//	"1.26"          → "1.26.0"
//	"3.12.5"        → "3.12.5"
//	"3.12+20240814" → "3.12.0+20240814" (+tag preserved)
func Normalize(v string) string {
	base, tag, hasTag := strings.Cut(v, "+")
	switch strings.Count(base, ".") {
	case 0:
		base += ".0.0"
	case 1:
		base += ".0"
	}
	if hasTag {
		return base + "+" + tag
	}
	return base
}

// ParseTuple parses "major.minor.patch" into an integer triple.
// Missing components default to zero.
func ParseTuple(v string) [3]int {
	var result [3]int
	parts := strings.SplitN(v, ".", 3)
	for i, p := range parts {
		if i >= 3 {
			break
		}
		result[i], _ = strconv.Atoi(p)
	}
	return result
}

// CmpTuple compares two version triples and returns -1, 0, or 1 for a < b, a == b, a > b.
func CmpTuple(a, b [3]int) int {
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// IsPartial reports whether v is a partial version (fewer than two dots before
// any +tag suffix). e.g. "22", "3.13", "1.26" are partial; "3.13.2" is not.
func IsPartial(v string) bool {
	base, _, _ := strings.Cut(v, "+")
	return strings.Count(base, ".") < 2
}
