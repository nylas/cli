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
	t.Run("enforces_default_timeout", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping long-running timeout test in short mode")
		}

		// Server that delays response beyond default timeout
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(95 * time.Second) // Longer than default 90s timeout
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		// Use context without timeout - should apply default 90s timeout
		start := time.Now()
		_, err := client.GetFolders(context.Background(), "grant-123")
		elapsed := time.Since(start)

		// Should timeout in ~90 seconds, not wait for full 95 seconds
		assert.Error(t, err)
		assert.True(t, elapsed < 92*time.Second, "Should timeout near 90 seconds")
		assert.True(t, elapsed > 89*time.Second, "Should wait at least 89 seconds")
	})

	t.Run("respects_existing_context_timeout", func(t *testing.T) {
		// Server that delays briefly
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(3 * time.Second)
			w.WriteHeader(http.StatusOK)
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
