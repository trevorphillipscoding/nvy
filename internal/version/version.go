// Package version provides shared version-string utilities.
package version

import "strings"

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
