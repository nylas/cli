package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestNewBaseClient(t *testing.T) {
	t.Run("creates client with default timeout", func(t *testing.T) {
		client := NewBaseClient("test-key", "test-model", "https://api.example.com", 0)

		if client.apiKey != "test-key" {
			t.Errorf("expected apiKey 'test-key', got %q", client.apiKey)
		}
		if client.model != "test-model" {
			t.Errorf("expected model 'test-model', got %q", client.model)
		}
		if client.baseURL != "https://api.example.com" {
			t.Errorf("expected baseURL 'https://api.example.com', got %q", client.baseURL)
		}
		if client.client.Timeout != 120*time.Second {
			t.Errorf("expected timeout 120s, got %v", client.client.Timeout)
		}
	})

	t.Run("creates client with custom timeout", func(t *testing.T) {
		client := NewBaseClient("key", "model", "url", 30*time.Second)

		if client.client.Timeout != 30*time.Second {
			t.Errorf("expected timeout 30s, got %v", client.client.Timeout)
		}
	})
}

func TestIsConfigured(t *testing.T) {
	t.Run("returns true when API key is set", func(t *testing.T) {
		client := NewBaseClient("test-key", "model", "url", 0)

		if !client.IsConfigured() {
			t.Error("expected IsConfigured to return true")
		}
	})

	t.Run("returns false when API key is empty", func(t *testing.T) {
		client := NewBaseClient("", "model", "url", 0)

		if client.IsConfigured() {
			t.Error("expected IsConfigured to return false")
		}
	})
}

func TestGetModel(t *testing.T) {
	tests := []struct {
		name         string
		defaultModel string
		requestModel string
		expected     string
	}{
		{
			name:         "returns request model when provided",
			defaultModel: "default-model",
			requestModel: "custom-model",
			expected:     "custom-model",
		},
		{
			name:         "returns default model when request model is empty",
			defaultModel: "default-model",
			requestModel: "",
			expected:     "default-model",
		},
		{
			name:         "returns empty when both are empty",
			defaultModel: "",
			requestModel: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewBaseClient("key", tt.defaultModel, "url", 0)
			result := client.GetModel(tt.requestModel)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDoJSONRequest(t *testing.T) {
	t.Run("successful request with body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			}
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Errorf("expected Authorization header, got %s", r.Header.Get("Authorization"))
			}

			// Read and verify body
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if body["test"] != "data" {
				t.Errorf("expected body['test']='data', got %q", body["test"])
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result": "success"}`))
		}))
		defer server.Close()

		client := NewBaseClient("test-key", "model", server.URL, 0)
		requestBody := map[string]string{"test": "data"}
		headers := map[string]string{"Authorization": "Bearer test-key"}

		resp, err := client.DoJSONRequest(context.Background(), http.MethodPost, "/test", requestBody, headers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("successful request without body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewBaseClient("key", "model", server.URL, 0)

		resp, err := client.DoJSONRequest(context.Background(), http.MethodGet, "/test", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewBaseClient("key", "model", server.URL, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := client.DoJSONRequest(ctx, http.MethodGet, "/test", nil, nil)
		if err == nil {
			t.Error("expected error for cancelled context")
		}
	})

	t.Run("handles invalid body marshaling", func(t *testing.T) {
		client := NewBaseClient("key", "model", "http://example.com", 0)

		// Channels cannot be marshaled to JSON
		invalidBody := make(chan int)

		_, err := client.DoJSONRequest(context.Background(), http.MethodPost, "/test", invalidBody, nil)
		if err == nil {
			t.Error("expected error for invalid body")
		}
		if err != nil && !contains(err.Error(), "failed to marshal request") {
			t.Errorf("expected marshal error, got: %v", err)
		}
	})
}

func TestReadJSONResponse(t *testing.T) {
	t.Run("successful decode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"name": "test", "value": 123}`))
		}))
		defer server.Close()

		client := NewBaseClient("key", "model", server.URL, 0)
		resp, err := client.DoJSONRequest(context.Background(), http.MethodGet, "/test", nil, nil)
		if err != nil {
			t.Fatalf("unexpected request error: %v", err)
		}

		var result map[string]any
		err = client.ReadJSONResponse(resp, &result)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}

		if result["name"] != "test" {
			t.Errorf("expected name='test', got %v", result["name"])
		}
		if result["value"].(float64) != 123 {
			t.Errorf("expected value=123, got %v", result["value"])
		}
	})

	t.Run("handles HTTP error status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error": "bad request"}`))
		}))
		defer server.Close()

		client := NewBaseClient("key", "model", server.URL, 0)
		resp, err := client.DoJSONRequest(context.Background(), http.MethodGet, "/test", nil, nil)
		if err != nil {
			t.Fatalf("unexpected request error: %v", err)
		}

		var result map[string]any
		err = client.ReadJSONResponse(resp, &result)
		if err == nil {
			t.Error("expected error for HTTP 400")
		}
		if err != nil && !contains(err.Error(), "API error (status 400)") {
			t.Errorf("expected API error message, got: %v", err)
		}
	})

	t.Run("truncates oversized error body", func(t *testing.T) {
		// Error bodies are capped at maxErrorBodyBytes (10KB), matching the
		// Nylas HTTP client, so a hostile/huge error response can't balloon memory.
		hugeBody := strings.Repeat("x", 64*1024)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(hugeBody))
		}))
		defer server.Close()

		client := NewBaseClient("key", "model", server.URL, 0)
		resp, err := client.DoJSONRequest(context.Background(), http.MethodGet, "/test", nil, nil)
		if err != nil {
			t.Fatalf("unexpected request error: %v", err)
		}

		var result map[string]any
		err = client.ReadJSONResponse(resp, &result)
		if err == nil {
			t.Fatal("expected error for HTTP 500")
		}
		if !contains(err.Error(), "API error (status 500)") {
			t.Errorf("expected API error message, got: %v", err)
		}
		if got := strings.Count(err.Error(), "x"); got != maxErrorBodyBytes {
			t.Errorf("error body length = %d, want truncated to %d", got, maxErrorBodyBytes)
		}
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{invalid json`))
		}))
		defer server.Close()

		client := NewBaseClient("key", "model", server.URL, 0)
		resp, err := client.DoJSONRequest(context.Background(), http.MethodGet, "/test", nil, nil)
		if err != nil {
			t.Fatalf("unexpected request error: %v", err)
		}

		var result map[string]any
		err = client.ReadJSONResponse(resp, &result)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
		if err != nil && !contains(err.Error(), "failed to decode response") {
			t.Errorf("expected decode error, got: %v", err)
		}
	})
}

func TestDoJSONRequestAndDecode(t *testing.T) {
	t.Run("successful request and decode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "ok"}`))
		}))
		defer server.Close()

		client := NewBaseClient("key", "model", server.URL, 0)

		var result map[string]string
		err := client.DoJSONRequestAndDecode(context.Background(), http.MethodGet, "/test", nil, nil, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result["status"] != "ok" {
			t.Errorf("expected status='ok', got %q", result["status"])
		}
	})

	t.Run("handles request error", func(t *testing.T) {
		client := NewBaseClient("key", "model", "http://invalid-url-that-does-not-exist.local", 0)

		var result map[string]string
		err := client.DoJSONRequestAndDecode(context.Background(), http.MethodGet, "/test", nil, nil, &result)
		if err == nil {
			t.Error("expected error for invalid URL")
		}
	})

	t.Run("handles decode error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`error`))
		}))
		defer server.Close()

		client := NewBaseClient("key", "model", server.URL, 0)

		var result map[string]string
		err := client.DoJSONRequestAndDecode(context.Background(), http.MethodGet, "/test", nil, nil, &result)
		if err == nil {
			t.Error("expected error for HTTP 400")
		}
	})
}

func TestConvertMessagesToMaps(t *testing.T) {
	messages := []domain.ChatMessage{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	converted := ConvertMessagesToMaps(messages)

	if len(converted) != len(messages) {
		t.Errorf("converted messages count = %d, want %d", len(converted), len(messages))
	}

	for i, msg := range converted {
		if msg["role"] != messages[i].Role {
			t.Errorf("message[%d] role = %q, want %q", i, msg["role"], messages[i].Role)
		}
		if msg["content"] != messages[i].Content {
			t.Errorf("message[%d] content = %q, want %q", i, msg["content"], messages[i].Content)
		}
	}
}

func TestConvertToolsOpenAIFormat(t *testing.T) {
	tools := []domain.Tool{
		{
			Name:        "get_weather",
			Description: "Get current weather",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]any{
						"type":        "string",
						"description": "City name",
					},
				},
			},
		},
	}

	converted := ConvertToolsOpenAIFormat(tools)

	if len(converted) != len(tools) {
		t.Errorf("converted tools count = %d, want %d", len(converted), len(tools))
	}

	if converted[0]["type"] != "function" {
		t.Errorf("tool type = %v, want %q", converted[0]["type"], "function")
	}

	fn, ok := converted[0]["function"].(map[string]any)
	if !ok {
		t.Fatal("function field is not a map")
	}

	if fn["name"] != tools[0].Name {
		t.Errorf("function name = %v, want %q", fn["name"], tools[0].Name)
	}

	if fn["description"] != tools[0].Description {
		t.Errorf("function description = %v, want %q", fn["description"], tools[0].Description)
	}
}

func TestFallbackStreamChat(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		chatFunc := func(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
			return &domain.ChatResponse{Content: "Hello!"}, nil
		}

		var received string
		callback := func(chunk string) error {
			received = chunk
			return nil
		}

		err := FallbackStreamChat(context.Background(), &domain.ChatRequest{}, chatFunc, callback)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if received != "Hello!" {
			t.Errorf("expected 'Hello!', got %q", received)
		}
	})

	t.Run("chat error", func(t *testing.T) {
		chatFunc := func(ctx context.Context, req *domain.ChatRequest) (*domain.ChatResponse, error) {
			return nil, context.DeadlineExceeded
		}

		callback := func(chunk string) error {
			return nil
		}

		err := FallbackStreamChat(context.Background(), &domain.ChatRequest{}, chatFunc, callback)
		if err != context.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", err)
		}
	})
}

// Helper function
func TestAPIError(t *testing.T) {
	t.Run("includes body in message", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(`{"error":"bad request"}`)),
		}
		err := apiError(resp)
		if err == nil {
			t.Fatal("expected error")
		}
		if !contains(err.Error(), `API error (status 400): {"error":"bad request"}`) {
			t.Errorf("expected body in error message, got: %v", err)
		}
	})

	t.Run("falls back to status-only on empty body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("")),
		}
		err := apiError(resp)
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "API error (status 500)" {
			t.Errorf("expected status-only message, got: %v", err)
		}
	})

	t.Run("truncates oversized body", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(strings.Repeat("x", 64*1024))),
		}
		err := apiError(resp)
		if err == nil {
			t.Fatal("expected error")
		}
		if got := strings.Count(err.Error(), "x"); got != maxErrorBodyBytes {
			t.Errorf("error body length = %d, want truncated to %d", got, maxErrorBodyBytes)
		}
	})

	t.Run("falls back to status-only on body read error", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(errReader{}),
		}
		err := apiError(resp)
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "API error (status 502)" {
			t.Errorf("expected status-only message, got: %v", err)
		}
	})
}

// errReader always fails, simulating a connection dropped mid-body.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
