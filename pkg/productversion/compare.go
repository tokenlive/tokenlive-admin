package productversion

import (
	"regexp"
	"strings"

	"golang.org/x/mod/semver"
)

var fullVersion = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+].*)?$`)

// StableVersion returns a canonical stable version with a leading v.
// It requires all three numeric components and ignores valid build metadata.
func StableVersion(raw string) (string, bool) {
	if !fullVersion.MatchString(raw) {
		return "", false
	}
	version := raw
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if !semver.IsValid(version) || semver.Prerelease(version) != "" {
		return "", false
	}
	return semver.Canonical(version), true
}

// Compare reports available, current, ahead, uncomparable, or no_candidate.
// Only explicitly marked releases with stable versions can be compared.
func Compare(current Build, target string) string {
	cv, ok := StableVersion(current.Version)
	if current.Kind != "release" || !ok {
		return "uncomparable"
	}
	tv, ok := StableVersion(target)
	if !ok {
		return "no_candidate"
	}
	switch semver.Compare(cv, tv) {
	case -1:
		return "available"
	case 0:
		return "current"
	default:
		return "ahead"
	}
}
