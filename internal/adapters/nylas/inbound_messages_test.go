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

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GET INBOUND MESSAGES TESTS
// =============================================================================

func TestGetInboundMessages(t *testing.T) {
	t.Run("successful_get_messages", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v3/grants/inbox-001/messages", r.URL.Path)
			assert.Equal(t, http.MethodGet, r.Method)

			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":        "msg-001",
						"grant_id":  "inbox-001",
						"subject":   "Test Subject 1",
						"from":      []map[string]string{{"name": "John", "email": "john@example.com"}},
						"to":        []map[string]string{{"name": "Support", "email": "support@app.nylas.email"}},
						"date":      time.Now().Add(-1 * time.Hour).Unix(),
						"unread":    true,
						"starred":   false,
						"snippet":   "This is a test message...",
						"body":      "This is a test message body.",
						"thread_id": "thread-001",
					},
					{
						"id":        "msg-002",
						"grant_id":  "inbox-001",
						"subject":   "Test Subject 2",
						"from":      []map[string]string{{"name": "Jane", "email": "jane@example.com"}},
						"to":        []map[string]string{{"name": "Support", "email": "support@app.nylas.email"}},
						"date":      time.Now().Add(-2 * time.Hour).Unix(),
						"unread":    false,
						"starred":   true,
						"snippet":   "Another test message...",
						"body":      "Another test message body.",
						"thread_id": "thread-002",
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

		messages, err := client.GetInboundMessages(context.Background(), "inbox-001", nil)

		require.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, "msg-001", messages[0].ID)
		assert.Equal(t, "Test Subject 1", messages[0].Subject)
		assert.True(t, messages[0].Unread)
		assert.Equal(t, "msg-002", messages[1].ID)
		assert.True(t, messages[1].Starred)
	})

	t.Run("with_limit_param", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "5", r.URL.Query().Get("limit"))

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

		params := &domain.MessageQueryParams{Limit: 5}
		_, err := client.GetInboundMessages(context.Background(), "inbox-001", params)

		assert.NoError(t, err)
	})

	t.Run("with_unread_param", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "true", r.URL.Query().Get("unread"))

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

		unread := true
		params := &domain.MessageQueryParams{Unread: &unread}
		_, err := client.GetInboundMessages(context.Background(), "inbox-001", params)

		assert.NoError(t, err)
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

		messages, err := client.GetInboundMessages(context.Background(), "inbox-001", nil)

		require.NoError(t, err)
		assert.Empty(t, messages)
	})

	t.Run("handles_api_error", func(t *testing.T) {
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

		_, err := client.GetInboundMessages(context.Background(), "nonexistent", nil)

		assert.Error(t, err)
	})
}

// =============================================================================
// MOCK CLIENT TESTS
// =============================================================================

func TestMockClient_InboundMethods(t *testing.T) {
	mock := NewMockClient()
	ctx := context.Background()

	t.Run("ListInboundInboxes", func(t *testing.T) {
		inboxes, err := mock.ListInboundInboxes(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, inboxes)
	})

	t.Run("GetInboundInbox", func(t *testing.T) {
		inbox, err := mock.GetInboundInbox(ctx, "test-id")
		assert.NoError(t, err)
		assert.NotNil(t, inbox)
	})

	t.Run("CreateInboundInbox", func(t *testing.T) {
		inbox, err := mock.CreateInboundInbox(ctx, "test")
		assert.NoError(t, err)
		assert.NotNil(t, inbox)
		assert.Contains(t, inbox.Email, "test")
	})

	t.Run("DeleteInboundInbox", func(t *testing.T) {
		err := mock.DeleteInboundInbox(ctx, "test-id")
		assert.NoError(t, err)
	})

	t.Run("GetInboundMessages", func(t *testing.T) {
		messages, err := mock.GetInboundMessages(ctx, "test-id", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, messages)
	})
}

// =============================================================================
// DEMO CLIENT TESTS
// =============================================================================

func TestDemoClient_InboundMethods(t *testing.T) {
	demo := NewDemoClient()
	ctx := context.Background()

	t.Run("ListInboundInboxes", func(t *testing.T) {
		inboxes, err := demo.ListInboundInboxes(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, inboxes)
		// Should have realistic demo data
		assert.Contains(t, inboxes[0].Email, "nylas.email")
	})

	t.Run("GetInboundInbox", func(t *testing.T) {
		inbox, err := demo.GetInboundInbox(ctx, "inbox-demo-001")
		assert.NoError(t, err)
		assert.NotNil(t, inbox)
	})

	t.Run("CreateInboundInbox", func(t *testing.T) {
		inbox, err := demo.CreateInboundInbox(ctx, "test")
		assert.NoError(t, err)
		assert.NotNil(t, inbox)
		assert.Contains(t, inbox.Email, "nylas.email")
	})

	t.Run("DeleteInboundInbox", func(t *testing.T) {
		err := demo.DeleteInboundInbox(ctx, "inbox-demo-001")
		assert.NoError(t, err)
	})

	t.Run("GetInboundMessages", func(t *testing.T) {
		messages, err := demo.GetInboundMessages(ctx, "inbox-demo-001", nil)
		assert.NoError(t, err)
		assert.NotEmpty(t, messages)
		// Should have realistic demo data with various message types
		assert.NotEmpty(t, messages[0].Subject)
		assert.NotEmpty(t, messages[0].From)
	})
}
