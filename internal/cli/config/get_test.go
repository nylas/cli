package config

import (
	"strings"
	"testing"
)

func TestSnakeToPascal(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single word lowercase",
			input: "region",
			want:  "Region",
		},
		{
			name:  "snake_case two words",
			input: "default_grant",
			want:  "DefaultGrant",
		},
		{
			name:  "snake_case three words",
			input: "api_base_url",
			want:  "APIBaseUrl",
		},
		{
			name:  "already camelCase",
			input: "timeout",
			want:  "Timeout",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single letter",
			input: "a",
			want:  "A",
		},
		{
			name:  "multiple underscores",
			input: "some_long_field_name",
			want:  "SomeLongFieldName",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := snakeToPascal(tt.input)
			if got != tt.want {
				t.Errorf("snakeToPascal(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetConfigValue(t *testing.T) {
	// Test struct matching typical config structure
	type APIConfig struct {
		Timeout string
		BaseUrl string
	}

	type OutputConfig struct {
		Format string
		Color  string
	}

	type TestConfig struct {
		Region       string
		DefaultGrant string
		API          *APIConfig
		Output       OutputConfig
	}

	tests := []struct {
		name    string
		cfg     any
		key     string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "get top-level field",
			cfg: &TestConfig{
				Region: "us",
			},
			key:  "region",
			want: "us",
		},
		{
			name: "get snake_case field",
			cfg: &TestConfig{
				DefaultGrant: "grant_123",
			},
			key:  "default_grant",
			want: "grant_123",
		},
		{
			name: "get nested field with pointer",
			cfg: &TestConfig{
				API: &APIConfig{
					Timeout: "90s",
				},
			},
			key:  "api.timeout",
			want: "90s",
		},
		{
			name: "get nested field without pointer",
			cfg: &TestConfig{
				Output: OutputConfig{
					Format: "json",
				},
			},
			key:  "output.format",
			want: "json",
		},
		{
			name: "get from nil pointer returns empty",
			cfg: &TestConfig{
				API: nil,
			},
			key:  "api.timeout",
			want: "",
		},
		{
			name: "unknown field returns error",
			cfg: &TestConfig{
				Region: "us",
			},
			key:     "unknown_field",
			wantErr: true,
			errMsg:  "unknown config key",
		},
		{
			name: "invalid nested access returns error",
			cfg: &TestConfig{
				Region: "us",
			},
			key:     "region.nested",
			wantErr: true,
			errMsg:  "cannot access field",
		},
		{
			name: "deeply nested field",
			cfg: &TestConfig{
				API: &APIConfig{
					BaseUrl: "https://api.us.nylas.com",
				},
			},
			key:  "api.base_url",
			want: "https://api.us.nylas.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getConfigValue(tt.cfg, tt.key)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getConfigValue() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("getConfigValue() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("getConfigValue() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("getConfigValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetConfigValue_WithNonStruct(t *testing.T) {
	// Test with non-struct value
	value := "plain string"
	_, err := getConfigValue(value, "field")

	if err == nil {
		t.Error("getConfigValue() with non-struct should return error, got nil")
	}
}
