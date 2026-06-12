package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
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

func TestOllamaClient_StreamChat_HTTPError(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"unauthorized", http.StatusUnauthorized},
		{"rate limited", http.StatusTooManyRequests},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"error":"request failed"}`))
			}))
			defer server.Close()

			client := NewOllamaClient(&domain.OllamaConfig{
				Host:  server.URL,
				Model: "mistral:latest",
			})

			var chunks []string
			err := client.StreamChat(context.Background(), &domain.ChatRequest{
				Messages: []domain.ChatMessage{{Role: "user", Content: "Hello"}},
			}, func(chunk string) error {
				chunks = append(chunks, chunk)
				return nil
			})

			if err == nil {
				t.Fatalf("StreamChat() error = nil, want error for HTTP %d", tt.status)
			}
			if !strings.Contains(err.Error(), strconv.Itoa(tt.status)) {
				t.Errorf("error %q does not mention status %d", err.Error(), tt.status)
			}
			if len(chunks) != 0 {
				t.Errorf("expected no chunks on HTTP error, got %v", chunks)
			}
		})
	}
}
