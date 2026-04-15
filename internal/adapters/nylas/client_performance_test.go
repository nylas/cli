package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test mock client implements interface

func TestContextTimeouts(t *testing.T) {
	t.Run("enforces_context_timeout", func(t *testing.T) {
		// Server that delays response beyond the context timeout
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(10 * time.Second): // Longer than our test timeout
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				return
			}
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		// Use context with 3-second timeout to verify timeout enforcement
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		start := time.Now()
		_, err := client.GetFolders(ctx, "grant-123")
		elapsed := time.Since(start)

		// Should timeout in ~3 seconds, not wait for full 10 seconds
		assert.Error(t, err)
		assert.True(t, elapsed < 5*time.Second, "Should timeout near 3 seconds, got %v", elapsed)
		assert.True(t, elapsed > 2*time.Second, "Should wait at least 2 seconds, got %v", elapsed)
	})

	t.Run("respects_existing_context_timeout", func(t *testing.T) {
		// Server that delays briefly
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(3 * time.Second):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				return
			}
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		// Use context with short timeout (2 seconds)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		start := time.Now()
		_, err := client.GetFolders(ctx, "grant-123")
		elapsed := time.Since(start)

		// Should timeout in ~2 seconds, not wait for default 30 seconds
		assert.Error(t, err)
		assert.True(t, elapsed < 3*time.Second, "Should timeout near 2 seconds")
		assert.True(t, elapsed > 1*time.Second, "Should wait at least 1 second")
	})

	t.Run("successful_request_within_timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":            "folder-1",
						"grant_id":      "grant-123",
						"name":          "Inbox",
						"system_folder": "inbox",
					},
				},
			})
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		// Use context with reasonable timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		folders, err := client.GetFolders(ctx, "grant-123")
		require.NoError(t, err)
		assert.Len(t, folders, 1)
		assert.Equal(t, "Inbox", folders[0].Name)
	})
}
