//go:build integration

package integration

import "strings"

// isUnavailableErr reports whether an API error means the feature/endpoint is
// simply not enabled for the test account (so the test should skip rather than
// fail). Mirrors the inline checks used by the existing notetaker tests.
func isUnavailableErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range []string{"not found", "forbidden", "not available", "not supported", "no access", "invalid_request"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
