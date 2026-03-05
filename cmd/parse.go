package cmd

import (
	"fmt"
	"strings"
)

// parseToolVersion accepts either:
//
//	["go", "1.22.1"]     — two separate arguments
//	["go@1.22.1"]        — single argument with @ separator
//
// The version is returned trimmed of whitespace and trailing dots
// (e.g. "1.26." → "1.26"). Each plugin normalizes or resolves
// versions in its own Resolve() implementation.
func parseToolVersion(args []string) (tool, ver string, err error) {
	clean := func(v string) string {
		return strings.TrimRight(strings.TrimSpace(v), ".")
	}
	if len(args) == 2 {
		return strings.TrimSpace(args[0]), clean(args[1]), nil
	}
	// Single arg must use the tool@version form.
	parts := strings.SplitN(args[0], "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("specify a version: nvy install <tool> <version>  or  nvy install <tool>@<version>")
	}
	return strings.TrimSpace(parts[0]), clean(parts[1]), nil
}
