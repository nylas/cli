package ai

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestNewOllamaClient(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		client := NewOllamaClient(nil)
		if client != nil {
			t.Error("expected nil client for nil config")
		}
	})

	t.Run("custom config", func(t *testing.T) {
		config := &domain.OllamaConfig{
			Host:  "http://custom:8080",
			Model: "llama2",
		}
		client := NewOllamaClient(config)

		if client == nil {
			t.Fatal("expected non-nil client")
			return
		}

		if client.baseURL != "http://custom:8080" {
			t.Errorf("baseURL = %q, want %q", client.baseURL, "http://custom:8080")
		}

		if client.model != "llama2" {
			t.Errorf("model = %q, want %q", client.model, "llama2")
		}

		if client.client == nil {
			t.Error("HTTP client is nil")
		}
	})
}

func TestOllamaClient_Name(t *testing.T) {
	client := NewOllamaClient(&domain.OllamaConfig{
		Host:  "http://localhost:11434",
		Model: "mistral:latest",
	})
	if name := client.Name(); name != "ollama" {
		t.Errorf("Name() = %q, want %q", name, "ollama")
	}
}

func TestOllamaClient_GetModel(t *testing.T) {
	client := NewOllamaClient(&domain.OllamaConfig{
		Host:  "http://localhost:11434",
		Model: "mistral:latest",
	})

	tests := []struct {
		name         string
		requestModel string
		want         string
	}{
		{
			name:         "use request model",
			requestModel: "llama2",
			want:         "llama2",
		},
		{
			name:         "use default model",
			requestModel: "",
			want:         "mistral:latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.GetModel(tt.requestModel)
			if got != tt.want {
				t.Errorf("GetModel() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Note: ConvertMessages and ConvertTools tests moved to base_client_test.go
// since these are now shared functions in base_client.go

func TestOllamaClient_IsAvailable(t *testing.T) {
	client := NewOllamaClient(&domain.OllamaConfig{
		Host:  "http://localhost:11434",
		Model: "mistral:latest",
	})

	ctx := context.Background()

	// This will attempt to connect to localhost:11434
	// In unit tests, this will likely fail unless Ollama is running
	// We're just testing that the method doesn't panic
	_ = client.IsAvailable(ctx)
}
