package common

import "strings"

// Truncate shortens a string to maxLen characters, adding "..." if truncated.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ExtractDomain extracts the domain portion from an email address.
// For example, "info@qasim.nylas.email" returns "qasim.nylas.email".
// Returns empty string if the email format is invalid.
func ExtractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}
