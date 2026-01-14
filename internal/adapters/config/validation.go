package config

import (
	"fmt"
	"os"
	"strings"
)

// RequiredEnvVar represents a required environment variable.
type RequiredEnvVar struct {
	Name        string
	Description string
	Optional    bool
}

// ValidateRequiredEnvVars validates that all required environment variables are set.
// Returns a slice of missing variables.
func ValidateRequiredEnvVars(vars []RequiredEnvVar) []string {
	var missing []string

	for _, v := range vars {
		if v.Optional {
			continue
		}

		value := os.Getenv(v.Name)
		if value == "" {
			missing = append(missing, v.Name)
		}
	}

	return missing
}

// FormatMissingEnvVars formats missing environment variables into a user-friendly error message.
func FormatMissingEnvVars(missing []string, vars []RequiredEnvVar) string {
	if len(missing) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Missing required environment variables:\n\n")

	// Create a map of descriptions
	descriptions := make(map[string]string)
	for _, v := range vars {
		descriptions[v.Name] = v.Description
	}

	for _, name := range missing {
		desc := descriptions[name]
		if desc == "" {
			desc = "No description available"
		}
		sb.WriteString(fmt.Sprintf("  %s - %s\n", name, desc))
	}

	return sb.String()
}

// ValidateAPICredentials validates Nylas API credentials from environment.
func ValidateAPICredentials() error {
	requiredVars := []RequiredEnvVar{
		{
			Name:        "NYLAS_API_KEY",
			Description: "Your Nylas API key",
			Optional:    false,
		},
	}

	missing := ValidateRequiredEnvVars(requiredVars)
	if len(missing) > 0 {
		return fmt.Errorf("%s\nSet via: export NYLAS_API_KEY=your_key_here\nOr run: nylas auth config",
			FormatMissingEnvVars(missing, requiredVars))
	}

	return nil
}
