package domain

import (
	"testing"
)

// TestAIConfig_IsConfigured tests the IsConfigured method.
func TestAIConfig_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		config *AIConfig
		want   bool
	}{
		{
			name:   "nil config returns false",
			config: nil,
			want:   false,
		},
		{
			name:   "empty config returns false",
			config: &AIConfig{},
			want:   false,
		},
		{
			name: "ollama only returns true",
			config: &AIConfig{
				Ollama: &OllamaConfig{Host: "http://localhost:11434", Model: "mistral"},
			},
			want: true,
		},
		{
			name: "claude only returns true",
			config: &AIConfig{
				Claude: &ClaudeConfig{Model: "claude-3"},
			},
			want: true,
		},
		{
			name: "openai only returns true",
			config: &AIConfig{
				OpenAI: &OpenAIConfig{Model: "gpt-4"},
			},
			want: true,
		},
		{
			name: "groq only returns true",
			config: &AIConfig{
				Groq: &GroqConfig{Model: "mixtral-8x7b"},
			},
			want: true,
		},
		{
			name: "openrouter only returns true",
			config: &AIConfig{
				OpenRouter: &OpenRouterConfig{Model: "anthropic/claude-3"},
			},
			want: true,
		},
		{
			name: "multiple providers returns true",
			config: &AIConfig{
				Ollama: &OllamaConfig{Host: "http://localhost:11434", Model: "mistral"},
				Claude: &ClaudeConfig{Model: "claude-3"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsConfigured()
			if got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAIConfig_ValidateForProvider tests the ValidateForProvider method.
func TestAIConfig_ValidateForProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   *AIConfig
		provider string
		wantErr  bool
	}{
		{
			name:     "nil config returns error",
			config:   nil,
			provider: "ollama",
			wantErr:  true,
		},
		{
			name: "valid ollama config",
			config: &AIConfig{
				Ollama: &OllamaConfig{Host: "http://localhost:11434", Model: "mistral"},
			},
			provider: "ollama",
			wantErr:  false,
		},
		{
			name: "missing ollama config",
			config: &AIConfig{
				DefaultProvider: "ollama",
			},
			provider: "ollama",
			wantErr:  true,
		},
		{
			name: "missing ollama host",
			config: &AIConfig{
				Ollama: &OllamaConfig{Model: "mistral"},
			},
			provider: "ollama",
			wantErr:  true,
		},
		{
			name: "missing ollama model",
			config: &AIConfig{
				Ollama: &OllamaConfig{Host: "http://localhost:11434"},
			},
			provider: "ollama",
			wantErr:  true,
		},
		{
			name: "valid claude config",
			config: &AIConfig{
				Claude: &ClaudeConfig{Model: "claude-3-5-sonnet"},
			},
			provider: "claude",
			wantErr:  false,
		},
		{
			name: "missing claude config",
			config: &AIConfig{
				DefaultProvider: "claude",
			},
			provider: "claude",
			wantErr:  true,
		},
		{
			name: "missing claude model",
			config: &AIConfig{
				Claude: &ClaudeConfig{APIKey: "sk-ant-xxx"},
			},
			provider: "claude",
			wantErr:  true,
		},
		{
			name: "valid openai config",
			config: &AIConfig{
				OpenAI: &OpenAIConfig{Model: "gpt-4"},
			},
			provider: "openai",
			wantErr:  false,
		},
		{
			name: "missing openai config",
			config: &AIConfig{
				DefaultProvider: "openai",
			},
			provider: "openai",
			wantErr:  true,
		},
		{
			name: "valid groq config",
			config: &AIConfig{
				Groq: &GroqConfig{Model: "mixtral-8x7b"},
			},
			provider: "groq",
			wantErr:  false,
		},
		{
			name: "missing groq config",
			config: &AIConfig{
				DefaultProvider: "groq",
			},
			provider: "groq",
			wantErr:  true,
		},
		{
			name: "valid openrouter config",
			config: &AIConfig{
				OpenRouter: &OpenRouterConfig{Model: "anthropic/claude-3"},
			},
			provider: "openrouter",
			wantErr:  false,
		},
		{
			name: "missing openrouter config",
			config: &AIConfig{
				DefaultProvider: "openrouter",
			},
			provider: "openrouter",
			wantErr:  true,
		},
		{
			name: "unknown provider",
			config: &AIConfig{
				Ollama: &OllamaConfig{Host: "http://localhost:11434", Model: "mistral"},
			},
			provider: "unknown",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateForProvider(tt.provider)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateForProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestDefaultAIConfig tests the DefaultAIConfig function.
func TestDefaultAIConfig(t *testing.T) {
	config := DefaultAIConfig()

	if config == nil {
		t.Fatal("DefaultAIConfig() returned nil")
		return
	}

	if config.DefaultProvider != "ollama" {
		t.Errorf("DefaultProvider = %q, want %q", config.DefaultProvider, "ollama")
	}

	if config.Ollama == nil {
		t.Fatal("Ollama config is nil")
	}

	if config.Ollama.Host != "http://localhost:11434" {
		t.Errorf("Ollama.Host = %q, want %q", config.Ollama.Host, "http://localhost:11434")
	}

	if config.Ollama.Model != "mistral:latest" {
		t.Errorf("Ollama.Model = %q, want %q", config.Ollama.Model, "mistral:latest")
	}

	// Verify default config is valid
	if !config.IsConfigured() {
		t.Error("DefaultAIConfig should return a configured config")
	}

	if err := config.ValidateForProvider("ollama"); err != nil {
		t.Errorf("DefaultAIConfig should be valid for ollama: %v", err)
	}
}
