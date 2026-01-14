//go:build !integration
// +build !integration

package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_UpdateThread(t *testing.T) {
	tests := []struct {
		name       string
		request    *domain.UpdateMessageRequest
		wantFields []string
	}{
		{
			name: "marks thread as read",
			request: func() *domain.UpdateMessageRequest {
				unread := false
				return &domain.UpdateMessageRequest{Unread: &unread}
			}(),
			wantFields: []string{"unread"},
		},
		{
			name: "marks thread as starred",
			request: func() *domain.UpdateMessageRequest {
				starred := true
				return &domain.UpdateMessageRequest{Starred: &starred}
			}(),
			wantFields: []string{"starred"},
		},
		{
			name: "moves thread to folders",
			request: &domain.UpdateMessageRequest{
				Folders: []string{"Archive", "Work"},
			},
			wantFields: []string{"folders"},
		},
		{
			name: "updates multiple fields",
			request: func() *domain.UpdateMessageRequest {
				unread := true
				starred := true
				return &domain.UpdateMessageRequest{
					Unread:  &unread,
					Starred: &starred,
					Folders: []string{"INBOX"},
				}
			}(),
			wantFields: []string{"unread", "starred", "folders"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/threads/thread-456", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)

				for _, field := range tt.wantFields {
					assert.Contains(t, body, field, "Missing field: %s", field)
				}

				response := map[string]any{
					"data": map[string]any{
						"id":                           "thread-456",
						"grant_id":                     "grant-123",
						"subject":                      "Updated Thread",
						"earliest_message_date":        1704000000,
						"latest_message_received_date": 1704100000,
						"latest_message_sent_date":     1704050000,
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
			thread, err := client.UpdateThread(ctx, "grant-123", "thread-456", tt.request)

			require.NoError(t, err)
			assert.Equal(t, "thread-456", thread.ID)
		})
	}
}

func TestHTTPClient_DeleteThread(t *testing.T) {
	tests := []struct {
		name       string
		threadID   string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "deletes with 200",
			threadID:   "thread-123",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "deletes with 204",
			threadID:   "thread-456",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "returns error for not found",
			threadID:   "nonexistent",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				expectedPath := "/v3/grants/grant-123/threads/" + tt.threadID
				assert.Equal(t, expectedPath, r.URL.Path)

				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					_ = json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]string{"message": "not found"},
					})
				}
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			err := client.DeleteThread(ctx, "grant-123", tt.threadID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPClient_GetThread_FullConversion(t *testing.T) {
	now := time.Now().Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"data": map[string]any{
				"id":                           "thread-full",
				"grant_id":                     "grant-full",
				"subject":                      "Complete Thread",
				"snippet":                      "This is the complete thread...",
				"has_attachments":              true,
				"has_drafts":                   true,
				"starred":                      true,
				"unread":                       false,
				"earliest_message_date":        now - 86400,
				"latest_message_received_date": now - 3600,
				"latest_message_sent_date":     now,
				"participants": []map[string]string{
					{"name": "Alice Smith", "email": "alice@example.com"},
					{"name": "Bob Jones", "email": "bob@example.com"},
					{"name": "Carol White", "email": "carol@example.com"},
				},
				"message_ids": []string{"msg-1", "msg-2", "msg-3", "msg-4"},
				"draft_ids":   []string{"draft-1", "draft-2"},
				"folders":     []string{"INBOX", "Important", "Work"},
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
	thread, err := client.GetThread(ctx, "grant-full", "thread-full")

	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, "thread-full", thread.ID)
	assert.Equal(t, "grant-full", thread.GrantID)
	assert.Equal(t, "Complete Thread", thread.Subject)
	assert.Equal(t, "This is the complete thread...", thread.Snippet)
	assert.True(t, thread.HasAttachments)
	assert.True(t, thread.HasDrafts)
	assert.True(t, thread.Starred)
	assert.False(t, thread.Unread)

	// Verify timestamps
	assert.Equal(t, time.Unix(now-86400, 0), thread.EarliestMessageDate)
	assert.Equal(t, time.Unix(now-3600, 0), thread.LatestMessageRecvDate)
	assert.Equal(t, time.Unix(now, 0), thread.LatestMessageSentDate)

	// Verify participants
	require.Len(t, thread.Participants, 3)
	assert.Equal(t, "Alice Smith", thread.Participants[0].Name)
	assert.Equal(t, "alice@example.com", thread.Participants[0].Email)
	assert.Equal(t, "Bob Jones", thread.Participants[1].Name)

	// Verify IDs
	require.Len(t, thread.MessageIDs, 4)
	assert.Equal(t, "msg-1", thread.MessageIDs[0])
	require.Len(t, thread.DraftIDs, 2)
	assert.Equal(t, "draft-1", thread.DraftIDs[0])

	// Verify folders
	require.Len(t, thread.FolderIDs, 3)
	assert.Contains(t, thread.FolderIDs, "INBOX")
	assert.Contains(t, thread.FolderIDs, "Important")
}

func TestHTTPClient_GetThreads_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		response    map[string]any
		errContains string
	}{
		{
			name:       "handles 401 unauthorized",
			statusCode: http.StatusUnauthorized,
			response: map[string]any{
				"error": map[string]string{"message": "Invalid API key"},
			},
			errContains: "Invalid API key",
		},
		{
			name:       "handles 403 forbidden",
			statusCode: http.StatusForbidden,
			response: map[string]any{
				"error": map[string]string{"message": "Access denied"},
			},
			errContains: "Access denied",
		},
		{
			name:       "handles 500 server error",
			statusCode: http.StatusInternalServerError,
			response: map[string]any{
				"error": map[string]string{"message": "Internal server error"},
			},
			errContains: "Internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.response)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			_, err := client.GetThreads(ctx, "grant-123", nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}
