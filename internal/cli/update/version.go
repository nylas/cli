// Package update provides CLI self-update functionality.
package update

import (
	"strings"

	"golang.org/x/mod/semver"
)

// normalizeVersion ensures version has "v" prefix for semver comparison.
func normalizeVersion(version string) string {
	v := strings.TrimSpace(version)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// parseVersion removes "v" prefix from version string.
func parseVersion(tagName string) string {
	return strings.TrimPrefix(strings.TrimSpace(tagName), "v")
}

// isUpdateAvailable returns true if latest version is newer than current.
func isUpdateAvailable(current, latest string) bool {
	// Dev version always allows update
	if current == "dev" || current == "" {
		return true
	}

	c := normalizeVersion(current)
	l := normalizeVersion(latest)

	// If either version is invalid, allow update
	if !semver.IsValid(c) || !semver.IsValid(l) {
		return true
	}

	return semver.Compare(c, l) < 0
}

// compareVersions returns:
//
//	-1 if current < latest (update available)
//	 0 if current == latest
//	 1 if current > latest (dev/newer version)
func compareVersions(current, latest string) int {
	if current == "dev" || current == "" {
		return -1
	}

	c := normalizeVersion(current)
	l := normalizeVersion(latest)

	if !semver.IsValid(c) || !semver.IsValid(l) {
		return -1
	}

	return semver.Compare(c, l)
}
