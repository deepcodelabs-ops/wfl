package validator

import (
	"fmt"

	semver "github.com/Masterminds/semver/v3"
)

// IsBreakingBump returns true if intentVersion has a higher major than implVersion.
// BREAKING CHANGE in wfl convention = a major semver bump (e.g. v1.x → v2.0.0).
func IsBreakingBump(intentVersion, implVersion string) (bool, error) {
	iv, err := semver.NewVersion(intentVersion)
	if err != nil {
		return false, fmt.Errorf("invalid intent version %q: %w", intentVersion, err)
	}
	av, err := semver.NewVersion(implVersion)
	if err != nil {
		return false, fmt.Errorf("invalid impl version %q: %w", implVersion, err)
	}
	return iv.Major() > av.Major(), nil
}

// BumpKind returns "patch", "minor", "major", or "unknown" given two semver strings.
func BumpKind(from, to string) string {
	fv, err := semver.NewVersion(from)
	if err != nil {
		return "unknown"
	}
	tv, err := semver.NewVersion(to)
	if err != nil {
		return "unknown"
	}
	if tv.Major() > fv.Major() {
		return "major"
	}
	if tv.Minor() > fv.Minor() {
		return "minor"
	}
	if tv.Patch() > fv.Patch() {
		return "patch"
	}
	return "unknown"
}
