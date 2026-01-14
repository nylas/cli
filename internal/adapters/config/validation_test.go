package config

import (
	"os"
	"strings"
	"testing"
)

func TestValidateRequiredEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		vars     []RequiredEnvVar
		envVars  map[string]string
		expected []string
	}{
		{
			name: "all required vars present",
			vars: []RequiredEnvVar{
				{Name: "TEST_VAR1", Description: "Test var 1", Optional: false},
				{Name: "TEST_VAR2", Description: "Test var 2", Optional: false},
			},
			envVars: map[string]string{
				"TEST_VAR1": "value1",
				"TEST_VAR2": "value2",
			},
			expected: []string{},
		},
		{
			name: "one required var missing",
			vars: []RequiredEnvVar{
				{Name: "TEST_VAR1", Description: "Test var 1", Optional: false},
				{Name: "TEST_VAR2", Description: "Test var 2", Optional: false},
			},
			envVars: map[string]string{
				"TEST_VAR1": "value1",
			},
			expected: []string{"TEST_VAR2"},
		},
		{
			name: "all required vars missing",
			vars: []RequiredEnvVar{
				{Name: "TEST_VAR1", Description: "Test var 1", Optional: false},
				{Name: "TEST_VAR2", Description: "Test var 2", Optional: false},
			},
			envVars:  map[string]string{},
			expected: []string{"TEST_VAR1", "TEST_VAR2"},
		},
		{
			name: "optional vars ignored",
			vars: []RequiredEnvVar{
				{Name: "TEST_VAR1", Description: "Test var 1", Optional: false},
				{Name: "TEST_VAR2", Description: "Test var 2", Optional: true},
			},
			envVars: map[string]string{
				"TEST_VAR1": "value1",
			},
			expected: []string{},
		},
		{
			name:     "no vars to validate",
			vars:     []RequiredEnvVar{},
			envVars:  map[string]string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
			}
			defer func() {
				// Clean up
				for k := range tt.envVars {
					_ = os.Unsetenv(k)
				}
				for _, v := range tt.vars {
					_ = os.Unsetenv(v.Name)
				}
			}()

			missing := ValidateRequiredEnvVars(tt.vars)

			if len(missing) != len(tt.expected) {
				t.Errorf("ValidateRequiredEnvVars() returned %d missing vars, want %d", len(missing), len(tt.expected))
			}

			for i, m := range missing {
				if i >= len(tt.expected) {
					break
				}
				if m != tt.expected[i] {
					t.Errorf("ValidateRequiredEnvVars() missing[%d] = %q, want %q", i, m, tt.expected[i])
				}
			}
		})
	}
}

func TestFormatMissingEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		missing  []string
		vars     []RequiredEnvVar
		contains []string
	}{
		{
			name:     "no missing vars",
			missing:  []string{},
			vars:     []RequiredEnvVar{},
			contains: []string{},
		},
		{
			name:    "one missing var",
			missing: []string{"TEST_VAR"},
			vars: []RequiredEnvVar{
				{Name: "TEST_VAR", Description: "A test variable"},
			},
			contains: []string{"Missing required environment variables", "TEST_VAR", "A test variable"},
		},
		{
			name:    "multiple missing vars",
			missing: []string{"VAR1", "VAR2"},
			vars: []RequiredEnvVar{
				{Name: "VAR1", Description: "First var"},
				{Name: "VAR2", Description: "Second var"},
			},
			contains: []string{"VAR1", "VAR2", "First var", "Second var"},
		},
		{
			name:    "missing var with no description",
			missing: []string{"NO_DESC"},
			vars: []RequiredEnvVar{
				{Name: "NO_DESC", Description: ""},
			},
			contains: []string{"NO_DESC", "No description available"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatMissingEnvVars(tt.missing, tt.vars)

			if len(tt.missing) == 0 && result != "" {
				t.Errorf("FormatMissingEnvVars() with no missing vars should return empty string, got %q", result)
			}

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("FormatMissingEnvVars() result should contain %q\nGot: %s", expected, result)
				}
			}
		})
	}
}

func TestValidateAPICredentials(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantError bool
	}{
		{
			name:      "API key present",
			apiKey:    "test-api-key",
			wantError: false,
		},
		{
			name:      "API key missing",
			apiKey:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.apiKey != "" {
				_ = os.Setenv("NYLAS_API_KEY", tt.apiKey)
			} else {
				_ = os.Unsetenv("NYLAS_API_KEY")
			}
			defer func() {
				_ = os.Unsetenv("NYLAS_API_KEY")
			}()

			err := ValidateAPICredentials()

			if tt.wantError && err == nil {
				t.Error("ValidateAPICredentials() expected error, got nil")
			}

			if !tt.wantError && err != nil {
				t.Errorf("ValidateAPICredentials() unexpected error: %v", err)
			}

			if tt.wantError && err != nil {
				// Check that error message contains helpful info
				errMsg := err.Error()
				if !strings.Contains(errMsg, "NYLAS_API_KEY") {
					t.Errorf("Error message should mention NYLAS_API_KEY, got: %s", errMsg)
				}
			}
		})
	}
}
