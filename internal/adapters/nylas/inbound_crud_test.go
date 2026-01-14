//go:build !integration
// +build !integration

package nylas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// LIST INBOUND INBOXES TESTS
// =============================================================================

func TestListInboundInboxes(t *testing.T) {
	t.Run("successful_list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v3/grants", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "inbox", r.URL.Query().Get("provider"))

			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":           "inbox-001",
						"email":        "support@app.nylas.email",
						"provider":     "inbox",
						"grant_status": "valid",
						"created_at":   time.Now().Add(-24 * time.Hour).Unix(),
						"updated_at":   time.Now().Unix(),
					},
					{
						"id":           "inbox-002",
						"email":        "sales@app.nylas.email",
						"provider":     "inbox",
						"grant_status": "valid",
						"created_at":   time.Now().Add(-48 * time.Hour).Unix(),
						"updated_at":   time.Now().Unix(),
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		inboxes, err := client.ListInboundInboxes(context.Background())

		require.NoError(t, err)
		assert.Len(t, inboxes, 2)
		assert.Equal(t, "inbox-001", inboxes[0].ID)
		assert.Equal(t, "support@app.nylas.email", inboxes[0].Email)
		assert.Equal(t, "valid", inboxes[0].GrantStatus)
		assert.Equal(t, "inbox-002", inboxes[1].ID)
	})

	t.Run("filters_by_inbox_provider", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return mix of inbox and non-inbox providers
			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":           "inbox-001",
						"email":        "support@app.nylas.email",
						"provider":     "inbox",
						"grant_status": "valid",
						"created_at":   time.Now().Unix(),
					},
					{
						"id":           "inbox-002",
						"email":        "user@gmail.com",
						"provider":     "google", // Should be filtered out
						"grant_status": "valid",
						"created_at":   time.Now().Unix(),
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		inboxes, err := client.ListInboundInboxes(context.Background())

		require.NoError(t, err)
		assert.Len(t, inboxes, 1)
		assert.Equal(t, "support@app.nylas.email", inboxes[0].Email)
	})

	t.Run("handles_empty_response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"data": []map[string]interface{}{},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		inboxes, err := client.ListInboundInboxes(context.Background())

		require.NoError(t, err)
		assert.Empty(t, inboxes)
	})

	t.Run("handles_api_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid API key",
				},
			})
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "invalid-key")

		_, err := client.ListInboundInboxes(context.Background())

		assert.Error(t, err)
	})
}

// =============================================================================
// GET INBOUND INBOX TESTS
// =============================================================================

func TestGetInboundInbox(t *testing.T) {
	t.Run("successful_get", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v3/grants/inbox-001", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":           "inbox-001",
					"email":        "support@app.nylas.email",
					"grant_status": "valid",
					"provider":     "inbox",
					"created_at":   time.Now().Add(-24 * time.Hour).Unix(),
					"updated_at":   time.Now().Unix(),
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		inbox, err := client.GetInboundInbox(context.Background(), "inbox-001")

		require.NoError(t, err)
		assert.Equal(t, "inbox-001", inbox.ID)
		assert.Equal(t, "support@app.nylas.email", inbox.Email)
		assert.Equal(t, "valid", inbox.GrantStatus)
	})

	t.Run("validates_inbox_provider", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":           "inbox-001",
					"email":        "user@gmail.com",
					"grant_status": "valid",
					"provider":     "google", // Not inbox provider
					"created_at":   time.Now().Unix(),
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		_, err := client.GetInboundInbox(context.Background(), "inbox-001")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not an inbound inbox")
	})

	t.Run("handles_not_found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Grant not found",
				},
			})
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		_, err := client.GetInboundInbox(context.Background(), "nonexistent")

		assert.Error(t, err)
	})
}

// =============================================================================
// CREATE INBOUND INBOX TESTS
// =============================================================================

func TestCreateInboundInbox(t *testing.T) {
	t.Run("successful_create", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v3/grants", r.URL.Path)
			assert.Equal(t, http.MethodPost, r.Method)

			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "inbox", body["provider"])
			settings := body["settings"].(map[string]interface{})
			assert.Equal(t, "support", settings["email"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":           "new-inbox-001",
					"email":        "support@app.nylas.email",
					"grant_status": "valid",
					"provider":     "virtual",
					"created_at":   time.Now().Unix(),
					"updated_at":   time.Now().Unix(),
				},
			}

			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		inbox, err := client.CreateInboundInbox(context.Background(), "support")

		require.NoError(t, err)
		assert.Equal(t, "new-inbox-001", inbox.ID)
		assert.Equal(t, "support@app.nylas.email", inbox.Email)
		assert.Equal(t, "valid", inbox.GrantStatus)
	})

	t.Run("handles_conflict_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Email already exists",
				},
			})
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		_, err := client.CreateInboundInbox(context.Background(), "existing")

		assert.Error(t, err)
	})

	t.Run("handles_validation_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Invalid email prefix",
				},
			})
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		_, err := client.CreateInboundInbox(context.Background(), "invalid@prefix")

		assert.Error(t, err)
	})
}

// =============================================================================
// DELETE INBOUND INBOX TESTS
// =============================================================================

func TestDeleteInboundInbox(t *testing.T) {
	t.Run("successful_delete", func(t *testing.T) {
		callCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// First call: GetInboundInbox to verify it's an inbox provider
				assert.Equal(t, "/v3/grants/inbox-001", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)

				response := map[string]interface{}{
					"data": map[string]interface{}{
						"id":           "inbox-001",
						"email":        "support@app.nylas.email",
						"grant_status": "valid",
						"provider":     "inbox",
						"created_at":   time.Now().Unix(),
					},
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(response) // Test helper, encode error not actionable
			} else {
				// Second call: RevokeGrant (DELETE)
				assert.Equal(t, "/v3/grants/inbox-001", r.URL.Path)
				assert.Equal(t, http.MethodDelete, r.Method)

				w.WriteHeader(http.StatusNoContent)
			}
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		err := client.DeleteInboundInbox(context.Background(), "inbox-001")

		assert.NoError(t, err)
		assert.Equal(t, 2, callCount, "Expected 2 API calls (get + delete)")
	})

	t.Run("handles_not_found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Grant not found",
				},
			})
		}))
		defer server.Close()

		client := NewHTTPClient()
		client.baseURL = server.URL
		client.SetCredentials("", "", "test-api-key")

		err := client.DeleteInboundInbox(context.Background(), "nonexistent")

		assert.Error(t, err)
	})
}
