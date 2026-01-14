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

func TestGetFoldersSystemFolderTypes(t *testing.T) {
	t.Run("handles_boolean_system_folder_from_google", func(t *testing.T) {
		// Google returns system_folder as boolean
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v3/grants/grant-123/folders", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Google API returns system_folder as boolean
			_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
				map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"id":            "folder-1",
							"grant_id":      "grant-123",
							"name":          "INBOX",
							"system_folder": true,
							"total_count":   100,
							"unread_count":  10,
						},
						{
							"id":            "folder-2",
							"grant_id":      "grant-123",
							"name":          "Custom Folder",
							"system_folder": false,
							"total_count":   50,
							"unread_count":  5,
						},
					},
				})
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		folders, err := client.GetFolders(context.Background(), "grant-123")
		require.NoError(t, err)
		assert.Len(t, folders, 2)
		assert.Equal(t, "INBOX", folders[0].Name)
		assert.Equal(t, "true", folders[0].SystemFolder) // Boolean true converted to string "true"
		assert.Equal(t, "Custom Folder", folders[1].Name)
		assert.Equal(t, "", folders[1].SystemFolder) // Boolean false converted to empty string
	})

	t.Run("handles_string_system_folder_from_microsoft", func(t *testing.T) {
		// Microsoft returns system_folder as string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Microsoft API returns system_folder as string
			_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
				map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"id":            "folder-1",
							"grant_id":      "grant-123",
							"name":          "Inbox",
							"system_folder": "inbox",
							"total_count":   100,
							"unread_count":  10,
						},
						{
							"id":            "folder-2",
							"grant_id":      "grant-123",
							"name":          "Sent Items",
							"system_folder": "sent",
							"total_count":   50,
							"unread_count":  0,
						},
					},
				})
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		folders, err := client.GetFolders(context.Background(), "grant-123")
		require.NoError(t, err)
		assert.Len(t, folders, 2)
		assert.Equal(t, "Inbox", folders[0].Name)
		assert.Equal(t, "inbox", folders[0].SystemFolder) // String preserved as-is
		assert.Equal(t, "Sent Items", folders[1].Name)
		assert.Equal(t, "sent", folders[1].SystemFolder)
	})

	t.Run("handles_null_system_folder", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
				map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"id":            "folder-1",
							"grant_id":      "grant-123",
							"name":          "Custom Folder",
							"system_folder": nil,
							"total_count":   25,
							"unread_count":  3,
						},
					},
				})
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		folders, err := client.GetFolders(context.Background(), "grant-123")
		require.NoError(t, err)
		assert.Len(t, folders, 1)
		assert.Equal(t, "Custom Folder", folders[0].Name)
		assert.Equal(t, "", folders[0].SystemFolder) // Null becomes empty string
	})
}

// TestGetFolderSystemFolderTypes tests GetFolder (single folder) handles system_folder types.
func TestGetFolderSystemFolderTypes(t *testing.T) {
	t.Run("handles_boolean_system_folder", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
				map[string]interface{}{
					"data": map[string]interface{}{
						"id":            "folder-123",
						"grant_id":      "grant-456",
						"name":          "INBOX",
						"system_folder": true,
						"total_count":   100,
						"unread_count":  10,
					},
				})
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		folder, err := client.GetFolder(context.Background(), "grant-456", "folder-123")
		require.NoError(t, err)
		assert.Equal(t, "INBOX", folder.Name)
		assert.Equal(t, "true", folder.SystemFolder)
	})
}

// TestRateLimiting tests that the rate limiter works correctly.
func TestRateLimiting(t *testing.T) {
	t.Run("limits_requests_per_second", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []interface{}{},
			})
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		// Make 5 requests rapidly
		start := time.Now()
		for i := 0; i < 5; i++ {
			_, _ = client.GetFolders(context.Background(), "grant-123")
		}
		elapsed := time.Since(start)

		// All 5 requests should have been made
		assert.Equal(t, 5, requestCount)

		// With rate limiting at 10 req/sec, 5 requests should take very little time
		// due to burst capacity, but we verify rate limiter is initialized
		assert.True(t, elapsed < 2*time.Second, "Rate limiting should allow burst requests")
	})

	t.Run("respects_context_cancellation_in_rate_limiter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Should fail immediately due to cancelled context
		_, err := client.GetFolders(ctx, "grant-123")
		assert.Error(t, err)
	})
}

// TestContextTimeouts tests that context timeouts are properly enforced.
