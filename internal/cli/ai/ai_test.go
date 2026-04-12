//go:build !integration

package ai

import (
	"testing"

	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// executeCommand executes a command and captures its output.
func executeCommand(root *cobra.Command, args ...string) (string, string, error) {
	return testutil.ExecuteCommand(root, args...)
}

func TestNewAICmd(t *testing.T) {
	cmd := NewAICmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "ai", cmd.Use)
	})

	t.Run("has_short_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Short)
		assert.Contains(t, cmd.Short, "AI")
	})

	t.Run("has_long_description", func(t *testing.T) {
		assert.NotEmpty(t, cmd.Long)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands, "AI command should have subcommands")
	})

	t.Run("has_required_subcommands", func(t *testing.T) {
		expectedCmds := []string{"config", "clear-data", "usage", "set-budget", "show-budget"}

		cmdMap := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			cmdMap[sub.Name()] = true
		}

		for _, expected := range expectedCmds {
			assert.True(t, cmdMap[expected], "Missing expected subcommand: %s", expected)
		}
	})
}

func TestAICommandHelp(t *testing.T) {
	cmd := NewAICmd()
	stdout, _, err := executeCommand(cmd, "--help")

	assert.NoError(t, err)

	// Check that help contains expected content
	expectedStrings := []string{
		"ai",
		"config",
		"Ollama",
		"Claude",
		"OpenAI",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, stdout, expected, "Help output should contain %q", expected)
	}
}

func TestConfigCommand(t *testing.T) {
	cmd := newConfigCmd()

	t.Run("command_name", func(t *testing.T) {
		assert.Equal(t, "config", cmd.Use)
	})

	t.Run("has_subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.NotEmpty(t, subcommands)

		cmdNames := make([]string, 0, len(subcommands))
		for _, sub := range subcommands {
			cmdNames = append(cmdNames, sub.Name())
		}

		assert.Contains(t, cmdNames, "show")
		assert.Contains(t, cmdNames, "list")
		assert.Contains(t, cmdNames, "get")
		assert.Contains(t, cmdNames, "set")
	})
}

func TestMaskAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "long API key",
			input:    "sk-proj-abcdefghijklmnopqrstuvwxyz",
			expected: "sk-proj-***...***wxyz",
		},
		{
			name:     "exact 13 characters",
			input:    "1234567890123",
			expected: "12345678***...***0123",
		},
		{
			name:     "short key 12 chars",
			input:    "123456789012",
			expected: "***",
		},
		{
			name:     "very short key",
			input:    "short",
			expected: "***",
		},
		{
			name:     "empty key",
			input:    "",
			expected: "***",
		},
		{
			name:     "real Claude API key format",
			input:    "sk-ant-api03-abcdefghijklmnopqrstuvwxyz123456789",
			expected: "sk-ant-a***...***6789",
		},
		{
			name:     "real OpenAI API key format",
			input:    "sk-proj-ABCDEFGHIJKLMNOPQRSTUVWXYZ12345678901234567890",
			expected: "sk-proj-***...***7890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskAPIKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetConfigValue(t *testing.T) {
	t.Parallel()

	// Create a full AI config for testing
	aiConfig := &domain.AIConfig{
		DefaultProvider: "ollama",
		Fallback: &domain.AIFallbackConfig{
			Enabled:   true,
			Providers: []string{"claude", "openai"},
		},
		Ollama: &domain.OllamaConfig{
			Host:  "http://localhost:11434",
			Model: "llama3.1:8b",
		},
		Claude: &domain.ClaudeConfig{
			APIKey: "test-claude-key",
			Model:  "claude-3-5-sonnet",
		},
		OpenAI: &domain.OpenAIConfig{
			APIKey: "test-openai-key",
			Model:  "gpt-4",
		},
		Groq: &domain.GroqConfig{
			APIKey: "test-groq-key",
			Model:  "llama-3.1-70b-versatile",
		},
		OpenRouter: &domain.OpenRouterConfig{
			APIKey: "test-openrouter-key",
			Model:  "anthropic/claude-3.5-sonnet",
		},
		Privacy: &domain.PrivacyConfig{
			AllowCloudAI:     true,
			DataRetention:    30,
			LocalStorageOnly: false,
		},
		Features: &domain.FeaturesConfig{
			NaturalLanguageScheduling: true,
			PredictiveScheduling:      true,
			FocusTimeProtection:       false,
			ConflictResolution:        true,
			EmailContextAnalysis:      false,
		},
	}

	tests := []struct {
		name      string
		key       string
		expected  string
		expectErr bool
	}{
		// Default provider
		{"default_provider", "default_provider", "ollama", false},

		// Fallback
		{"fallback.enabled", "fallback.enabled", "true", false},
		{"fallback.providers", "fallback.providers", "claude,openai", false},

		// Ollama
		{"ollama.host", "ollama.host", "http://localhost:11434", false},
		{"ollama.model", "ollama.model", "llama3.1:8b", false},

		// Claude
		{"claude.api_key", "claude.api_key", "test-claude-key", false},
		{"claude.model", "claude.model", "claude-3-5-sonnet", false},

		// OpenAI
		{"openai.api_key", "openai.api_key", "test-openai-key", false},
		{"openai.model", "openai.model", "gpt-4", false},

		// Groq
		{"groq.api_key", "groq.api_key", "test-groq-key", false},
		{"groq.model", "groq.model", "llama-3.1-70b-versatile", false},

		// OpenRouter
		{"openrouter.api_key", "openrouter.api_key", "test-openrouter-key", false},
		{"openrouter.model", "openrouter.model", "anthropic/claude-3.5-sonnet", false},

		// Privacy
		{"privacy.allow_cloud_ai", "privacy.allow_cloud_ai", "true", false},
		{"privacy.data_retention", "privacy.data_retention", "30", false},
		{"privacy.local_storage_only", "privacy.local_storage_only", "false", false},

		// Features
		{"features.natural_language_scheduling", "features.natural_language_scheduling", "true", false},
		{"features.predictive_scheduling", "features.predictive_scheduling", "true", false},
		{"features.focus_time_protection", "features.focus_time_protection", "false", false},
		{"features.conflict_resolution", "features.conflict_resolution", "true", false},
		{"features.email_context_analysis", "features.email_context_analysis", "false", false},

		// Error cases
		{"unknown key", "unknown", "", true},
		{"invalid subkey", "ollama", "", true},
		{"unknown ollama subkey", "ollama.unknown", "", true},
		{"unknown claude subkey", "claude.unknown", "", true},
		{"unknown openai subkey", "openai.unknown", "", true},
		{"unknown groq subkey", "groq.unknown", "", true},
		{"unknown openrouter subkey", "openrouter.unknown", "", true},
		{"unknown fallback subkey", "fallback.unknown", "", true},
		{"unknown privacy subkey", "privacy.unknown", "", true},
		{"unknown features subkey", "features.unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getConfigValue(aiConfig, tt.key)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetConfigValueNilConfigs(t *testing.T) {
	t.Parallel()

	// Create AI config with nil sub-configs
	aiConfig := &domain.AIConfig{
		DefaultProvider: "ollama",
	}

	tests := []struct {
		name string
		key  string
	}{
		{"fallback not configured", "fallback.enabled"},
		{"ollama not configured", "ollama.host"},
		{"claude not configured", "claude.model"},
		{"openai not configured", "openai.model"},
		{"groq not configured", "groq.model"},
		{"openrouter not configured", "openrouter.model"},
		{"privacy not configured", "privacy.allow_cloud_ai"},
		{"features not configured", "features.natural_language_scheduling"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getConfigValue(aiConfig, tt.key)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not configured")
		})
	}
}

func TestSetConfigValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		key       string
		value     string
		expectErr bool
		validate  func(t *testing.T, ai *domain.AIConfig)
	}{
		{
			name:      "set default_provider",
			key:       "default_provider",
			value:     "claude",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.Equal(t, "claude", ai.DefaultProvider)
			},
		},
		{
			name:      "set invalid provider",
			key:       "default_provider",
			value:     "invalid",
			expectErr: true,
			validate:  nil,
		},
		{
			name:      "set fallback.enabled",
			key:       "fallback.enabled",
			value:     "true",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.True(t, ai.Fallback.Enabled)
			},
		},
		{
			name:      "set fallback.providers",
			key:       "fallback.providers",
			value:     "claude,groq",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.Equal(t, []string{"claude", "groq"}, ai.Fallback.Providers)
			},
		},
		{
			name:      "set ollama.host",
			key:       "ollama.host",
			value:     "http://custom:11434",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.Equal(t, "http://custom:11434", ai.Ollama.Host)
			},
		},
		{
			name:      "set ollama.model",
			key:       "ollama.model",
			value:     "llama3.1:70b",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.Equal(t, "llama3.1:70b", ai.Ollama.Model)
			},
		},
		{
			name:      "set claude.api_key",
			key:       "claude.api_key",
			value:     "sk-ant-api03-test",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.Equal(t, "sk-ant-api03-test", ai.Claude.APIKey)
			},
		},
		{
			name:      "set privacy.allow_cloud_ai",
			key:       "privacy.allow_cloud_ai",
			value:     "true",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.True(t, ai.Privacy.AllowCloudAI)
			},
		},
		{
			name:      "set privacy.data_retention",
			key:       "privacy.data_retention",
			value:     "90",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.Equal(t, 90, ai.Privacy.DataRetention)
			},
		},
		{
			name:      "set privacy.data_retention invalid",
			key:       "privacy.data_retention",
			value:     "not-a-number",
			expectErr: true,
			validate:  nil,
		},
		{
			name:      "set features.natural_language_scheduling",
			key:       "features.natural_language_scheduling",
			value:     "true",
			expectErr: false,
			validate: func(t *testing.T, ai *domain.AIConfig) {
				assert.True(t, ai.Features.NaturalLanguageScheduling)
			},
		},
		{
			name:      "unknown key",
			key:       "unknown",
			value:     "value",
			expectErr: true,
			validate:  nil,
		},
		{
			name:      "invalid key format",
			key:       "ollama",
			value:     "value",
			expectErr: true,
			validate:  nil,
		},
		{
			name:      "unknown ollama subkey",
			key:       "ollama.unknown",
			value:     "value",
			expectErr: true,
			validate:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh config for each test
			aiConfig := &domain.AIConfig{}

			err := setConfigValue(aiConfig, tt.key, tt.value)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, aiConfig)
				}
			}
		})
	}
}

func TestSetConfigValueAllProviders(t *testing.T) {
	t.Parallel()

	// Test setting values for all providers
	providers := []string{"ollama", "claude", "openai", "groq", "openrouter"}

	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			aiConfig := &domain.AIConfig{}
			err := setConfigValue(aiConfig, "default_provider", provider)
			assert.NoError(t, err)
			assert.Equal(t, provider, aiConfig.DefaultProvider)
		})
	}
}
