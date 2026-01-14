//go:build !integration
// +build !integration

package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_GetMessages(t *testing.T) {
	tests := []struct {
		name           string
		grantID        string
		limit          int
		serverResponse map[string]any
		statusCode     int
		wantErr        bool
		wantCount      int
	}{
		{
			name:    "returns messages successfully",
			grantID: "grant-123",
			limit:   10,
			serverResponse: map[string]any{
				"data": []map[string]any{
					{
						"id":        "msg-1",
						"grant_id":  "grant-123",
						"thread_id": "thread-1",
						"subject":   "Test Subject",
						"from":      []map[string]string{{"name": "Alice", "email": "alice@example.com"}},
						"to":        []map[string]string{{"name": "Bob", "email": "bob@example.com"}},
						"body":      "Test body content",
						"snippet":   "Test body...",
						"date":      1704067200,
						"unread":    true,
						"starred":   false,
						"folders":   []string{"INBOX"},
					},
					{
						"id":        "msg-2",
						"grant_id":  "grant-123",
						"thread_id": "thread-2",
						"subject":   "Another Subject",
						"from":      []map[string]string{{"name": "Charlie", "email": "charlie@example.com"}},
						"to":        []map[string]string{{"name": "Bob", "email": "bob@example.com"}},
						"body":      "Another body",
						"date":      1704153600,
						"unread":    false,
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantCount:  2,
		},
		{
			name:    "returns empty list when no messages",
			grantID: "grant-456",
			limit:   10,
			serverResponse: map[string]any{
				"data": []any{},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.URL.Path, "/v3/grants/"+tt.grantID+"/messages")
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			messages, err := client.GetMessages(ctx, tt.grantID, tt.limit)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, messages, tt.wantCount)
		})
	}
}

func TestHTTPClient_GetMessagesWithParams(t *testing.T) {
	tests := []struct {
		name         string
		params       *domain.MessageQueryParams
		wantQuery    map[string]string
		notWantQuery []string
	}{
		{
			name: "includes all filter params",
			params: &domain.MessageQueryParams{
				Limit:     25,
				Subject:   "important",
				From:      "sender@example.com",
				To:        "recipient@example.com",
				ThreadID:  "thread-123",
				PageToken: "next-page",
			},
			wantQuery: map[string]string{
				"limit":      "25",
				"subject":    "important",
				"from":       "sender@example.com",
				"to":         "recipient@example.com",
				"thread_id":  "thread-123",
				"page_token": "next-page",
			},
		},
		{
			name: "includes boolean filters",
			params: func() *domain.MessageQueryParams {
				unread := true
				starred := false
				hasAttachment := true
				return &domain.MessageQueryParams{
					Limit:         10,
					Unread:        &unread,
					Starred:       &starred,
					HasAttachment: &hasAttachment,
				}
			}(),
			wantQuery: map[string]string{
				"unread":         "true",
				"starred":        "false",
				"has_attachment": "true",
			},
		},
		{
			name: "includes date range params",
			params: &domain.MessageQueryParams{
				Limit:          10,
				ReceivedBefore: 1704153600,
				ReceivedAfter:  1704067200,
			},
			wantQuery: map[string]string{
				"received_before": "1704153600",
				"received_after":  "1704067200",
			},
		},
		{
			name: "includes search query",
			params: &domain.MessageQueryParams{
				Limit:       10,
				SearchQuery: "meeting notes",
			},
			wantQuery: map[string]string{
				"q": "meeting notes",
			},
		},
		{
			name: "includes folder filter",
			params: &domain.MessageQueryParams{
				Limit: 10,
				In:    []string{"INBOX", "SENT"},
			},
			wantQuery: map[string]string{
				"in": "INBOX",
			},
		},
		{
			name:   "uses default limit for nil params",
			params: nil,
			wantQuery: map[string]string{
				"limit": "10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, expectedValue := range tt.wantQuery {
					actualValue := r.URL.Query().Get(key)
					if actualValue == "" {
						values := r.URL.Query()[key]
						if len(values) > 0 {
							actualValue = values[0]
						}
					}
					assert.Equal(t, expectedValue, actualValue, "Query param %s mismatch", key)
				}

				for _, key := range tt.notWantQuery {
					assert.Empty(t, r.URL.Query().Get(key), "Query param %s should not be present", key)
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []any{},
				})
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			_, _ = client.GetMessagesWithParams(ctx, "grant-123", tt.params)
		})
	}
}

func TestHTTPClient_GetMessagesWithCursor(t *testing.T) {
	t.Run("returns pagination info", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]any{
				"data": []map[string]any{
					{"id": "msg-1", "subject": "First", "date": 1704067200},
					{"id": "msg-2", "subject": "Second", "date": 1704153600},
				},
				"next_cursor": "eyJsYXN0X2lkIjoibXNnLTIifQ==",
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		result, err := client.GetMessagesWithCursor(ctx, "grant-123", &domain.MessageQueryParams{Limit: 2})

		require.NoError(t, err)
		assert.Len(t, result.Data, 2)
		assert.Equal(t, "eyJsYXN0X2lkIjoibXNnLTIifQ==", result.Pagination.NextCursor)
		assert.True(t, result.Pagination.HasMore)
	})

	t.Run("handles last page without cursor", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]any{
				"data": []map[string]any{
					{"id": "msg-1", "subject": "Last", "date": 1704067200},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		result, err := client.GetMessagesWithCursor(ctx, "grant-123", nil)

		require.NoError(t, err)
		assert.Empty(t, result.Pagination.NextCursor)
		assert.False(t, result.Pagination.HasMore)
	})
}

func TestHTTPClient_GetMessage(t *testing.T) {
	tests := []struct {
		name           string
		grantID        string
		messageID      string
		serverResponse map[string]any
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:      "returns message successfully",
			grantID:   "grant-123",
			messageID: "msg-456",
			serverResponse: map[string]any{
				"data": map[string]any{
					"id":        "msg-456",
					"grant_id":  "grant-123",
					"thread_id": "thread-789",
					"subject":   "Test Email",
					"from":      []map[string]string{{"name": "Sender", "email": "sender@example.com"}},
					"to":        []map[string]string{{"name": "Receiver", "email": "receiver@example.com"}},
					"cc":        []map[string]string{{"name": "CC Person", "email": "cc@example.com"}},
					"bcc":       []map[string]string{{"name": "BCC Person", "email": "bcc@example.com"}},
					"reply_to":  []map[string]string{{"name": "Reply", "email": "reply@example.com"}},
					"body":      "<p>Email body content</p>",
					"snippet":   "Email body content",
					"date":      1704067200,
					"unread":    true,
					"starred":   true,
					"folders":   []string{"INBOX"},
					"attachments": []map[string]any{
						{
							"id":           "attach-1",
							"filename":     "report.pdf",
							"content_type": "application/pdf",
							"size":         12345,
							"is_inline":    false,
						},
					},
					"metadata":   map[string]string{"custom_key": "custom_value"},
					"created_at": 1704067200,
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:      "returns error for not found",
			grantID:   "grant-123",
			messageID: "nonexistent",
			serverResponse: map[string]any{
				"error": map[string]string{"message": "message not found"},
			},
			statusCode:  http.StatusNotFound,
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				expectedPath := "/v3/grants/" + tt.grantID + "/messages/" + tt.messageID
				assert.Equal(t, expectedPath, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			message, err := client.GetMessage(ctx, tt.grantID, tt.messageID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.messageID, message.ID)
			assert.Equal(t, tt.grantID, message.GrantID)
			assert.Equal(t, "Test Email", message.Subject)
			assert.Len(t, message.From, 1)
			assert.Equal(t, "Sender", message.From[0].Name)
			assert.Len(t, message.Attachments, 1)
			assert.True(t, message.Unread)
			assert.True(t, message.Starred)
		})
	}
}
