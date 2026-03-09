package semver

import "fmt"

// Resolve resolves a user version reference against available exact versions.
// Rules:
//   - "1" resolves to latest 1.x.x
//   - "1.25" resolves to latest 1.25.x
//   - "1.25.4" resolves to exact 1.25.4
func Resolve(request string, available []string) (string, error) {
	ref, err := ParseReference(request)
	if err != nil {
		return "", err
	}

	parsed := make([]Version, 0, len(available))
	for _, raw := range available {
		v, parseErr := ParseVersion(raw)
		if parseErr != nil {
			continue
		}
		parsed = append(parsed, v)
	}

	if len(parsed) == 0 {
		return "", fmt.Errorf("no versions available")
	}

	var best Version
	found := false
	for _, v := range parsed {
		if !matches(ref, v) {
			continue
		}
		if !found || Compare(v, best) > 0 {
			best = v
			found = true
		}
	}

	if !found {
		switch ref.Parts {
		case 1:
			return "", fmt.Errorf("no version found for %d.x.x", ref.Major)
		case 2:
			return "", fmt.Errorf("no version found for %d.%d.x", ref.Major, ref.Minor)
		default:
			return "", fmt.Errorf("version %d.%d.%d not found", ref.Major, ref.Minor, ref.Patch)
		}
	}

	return best.String(), nil
}

func matches(ref Reference, v Version) bool {
	if v.Major != ref.Major {
		return false
	}
	if ref.Parts >= 2 && v.Minor != ref.Minor {
		return false
	}
	if ref.Parts == 3 && v.Patch != ref.Patch {
		return false
	}
	return true
}
