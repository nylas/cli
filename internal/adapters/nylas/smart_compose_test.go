//go:build !integration
// +build !integration

package nylas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_SmartCompose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/messages/smart-compose", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Verify request body
		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, "Draft a thank you email", req["prompt"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"suggestion": "Thank you for your time in yesterday's meeting. I appreciate the insights you shared.",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response) // Test helper, encode error not actionable
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.SmartComposeRequest{
		Prompt: "Draft a thank you email",
	}

	suggestion, err := client.SmartCompose(ctx, "grant-123", req)

	require.NoError(t, err)
	assert.NotNil(t, suggestion)
	assert.Contains(t, suggestion.Suggestion, "Thank you")
}

func TestHTTPClient_SmartComposeReply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/messages/msg-456/smart-compose", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		// Verify request body
		var req map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&req)
		assert.NoError(t, err)
		assert.Equal(t, "Reply accepting the invitation", req["prompt"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"suggestion": "Thank you for the invitation. I'd be happy to attend the event.",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response) // Test helper, encode error not actionable
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.SmartComposeRequest{
		Prompt: "Reply accepting the invitation",
	}

	suggestion, err := client.SmartComposeReply(ctx, "grant-123", "msg-456", req)

	require.NoError(t, err)
	assert.NotNil(t, suggestion)
	assert.Contains(t, suggestion.Suggestion, "invitation")
}

func TestHTTPClient_SmartCompose_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Prompt exceeds maximum length",
			},
		}
		_ = json.NewEncoder(w).Encode(response) // Test helper, encode error not actionable
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.SmartComposeRequest{
		Prompt: "Very long prompt...",
	}

	suggestion, err := client.SmartCompose(ctx, "grant-123", req)

	assert.Error(t, err)
	assert.Nil(t, suggestion)
}
