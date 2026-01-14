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

func TestHTTPClient_GetThreads(t *testing.T) {
	tests := []struct {
		name           string
		params         *domain.ThreadQueryParams
		serverResponse map[string]interface{}
		statusCode     int
		wantCount      int
		wantErr        bool
	}{
		{
			name:   "returns threads successfully",
			params: nil,
			serverResponse: map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":                           "thread-1",
						"grant_id":                     "grant-123",
						"subject":                      "Project Discussion",
						"snippet":                      "Let's discuss the project...",
						"has_attachments":              true,
						"has_drafts":                   false,
						"starred":                      false,
						"unread":                       true,
						"earliest_message_date":        1704067200,
						"latest_message_received_date": 1704153600,
						"latest_message_sent_date":     1704153600,
						"participants": []map[string]string{
							{"name": "Alice", "email": "alice@example.com"},
							{"name": "Bob", "email": "bob@example.com"},
						},
						"message_ids": []string{"msg-1", "msg-2", "msg-3"},
						"draft_ids":   []string{},
						"folders":     []string{"INBOX"},
					},
					{
						"id":                           "thread-2",
						"grant_id":                     "grant-123",
						"subject":                      "Another Thread",
						"unread":                       false,
						"starred":                      true,
						"earliest_message_date":        1704000000,
						"latest_message_received_date": 1704100000,
						"latest_message_sent_date":     1704100000,
						"participants": []map[string]string{
							{"name": "Charlie", "email": "charlie@example.com"},
						},
						"message_ids": []string{"msg-4"},
					},
				},
			},
			statusCode: http.StatusOK,
			wantCount:  2,
			wantErr:    false,
		},
		{
			name:   "returns empty list",
			params: &domain.ThreadQueryParams{Limit: 10},
			serverResponse: map[string]interface{}{
				"data": []interface{}{},
			},
			statusCode: http.StatusOK,
			wantCount:  0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Contains(t, r.URL.Path, "/v3/grants/grant-123/threads")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			threads, err := client.GetThreads(ctx, "grant-123", tt.params)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, threads, tt.wantCount)

			if tt.wantCount > 0 {
				assert.Equal(t, "thread-1", threads[0].ID)
				assert.Equal(t, "Project Discussion", threads[0].Subject)
				assert.True(t, threads[0].HasAttachments)
				assert.True(t, threads[0].Unread)
				assert.Len(t, threads[0].Participants, 2)
				assert.Len(t, threads[0].MessageIDs, 3)
			}
		})
	}
}

func TestHTTPClient_GetThreads_QueryParams(t *testing.T) {
	tests := []struct {
		name      string
		params    *domain.ThreadQueryParams
		wantQuery map[string]string
	}{
		{
			name: "includes limit and offset",
			params: &domain.ThreadQueryParams{
				Limit:  25,
				Offset: 50,
			},
			wantQuery: map[string]string{
				"limit":  "25",
				"offset": "50",
			},
		},
		{
			name: "includes subject filter",
			params: &domain.ThreadQueryParams{
				Limit:   10,
				Subject: "important",
			},
			wantQuery: map[string]string{
				"subject": "important",
			},
		},
		{
			name: "includes from filter",
			params: &domain.ThreadQueryParams{
				Limit: 10,
				From:  "sender@example.com",
			},
			wantQuery: map[string]string{
				"from": "sender@example.com",
			},
		},
		{
			name: "includes to filter",
			params: &domain.ThreadQueryParams{
				Limit: 10,
				To:    "recipient@example.com",
			},
			wantQuery: map[string]string{
				"to": "recipient@example.com",
			},
		},
		{
			name: "includes unread filter",
			params: func() *domain.ThreadQueryParams {
				unread := true
				return &domain.ThreadQueryParams{
					Limit:  10,
					Unread: &unread,
				}
			}(),
			wantQuery: map[string]string{
				"unread": "true",
			},
		},
		{
			name: "includes starred filter",
			params: func() *domain.ThreadQueryParams {
				starred := true
				return &domain.ThreadQueryParams{
					Limit:   10,
					Starred: &starred,
				}
			}(),
			wantQuery: map[string]string{
				"starred": "true",
			},
		},
		{
			name: "includes search query",
			params: &domain.ThreadQueryParams{
				Limit:       10,
				SearchQuery: "meeting notes",
			},
			wantQuery: map[string]string{
				"q": "meeting notes",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, expectedValue := range tt.wantQuery {
					assert.Equal(t, expectedValue, r.URL.Query().Get(key), "Query param %s mismatch", key)
				}

				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"data": []interface{}{},
				})
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			_, _ = client.GetThreads(ctx, "grant-123", tt.params)
		})
	}
}

func TestHTTPClient_GetThread(t *testing.T) {
	tests := []struct {
		name           string
		threadID       string
		serverResponse map[string]interface{}
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:     "returns thread successfully",
			threadID: "thread-123",
			serverResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"id":                           "thread-123",
					"grant_id":                     "grant-123",
					"subject":                      "Important Discussion",
					"snippet":                      "This is the beginning...",
					"has_attachments":              true,
					"has_drafts":                   true,
					"starred":                      true,
					"unread":                       false,
					"earliest_message_date":        1704000000,
					"latest_message_received_date": 1704100000,
					"latest_message_sent_date":     1704050000,
					"participants": []map[string]string{
						{"name": "Alice", "email": "alice@example.com"},
					},
					"message_ids": []string{"msg-1", "msg-2"},
					"draft_ids":   []string{"draft-1"},
					"folders":     []string{"INBOX", "Important"},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:     "returns error for not found",
			threadID: "nonexistent",
			serverResponse: map[string]interface{}{
				"error": map[string]string{"message": "thread not found"},
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
				expectedPath := "/v3/grants/grant-123/threads/" + tt.threadID
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
			thread, err := client.GetThread(ctx, "grant-123", tt.threadID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.threadID, thread.ID)
			assert.Equal(t, "Important Discussion", thread.Subject)
			assert.True(t, thread.HasAttachments)
			assert.True(t, thread.HasDrafts)
			assert.True(t, thread.Starred)
			assert.False(t, thread.Unread)
		})
	}
}

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

				var body map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&body)

				for _, field := range tt.wantFields {
					assert.Contains(t, body, field, "Missing field: %s", field)
				}

				response := map[string]interface{}{
					"data": map[string]interface{}{
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
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
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
		response := map[string]interface{}{
			"data": map[string]interface{}{
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
		response    map[string]interface{}
		errContains string
	}{
		{
			name:       "handles 401 unauthorized",
			statusCode: http.StatusUnauthorized,
			response: map[string]interface{}{
				"error": map[string]string{"message": "Invalid API key"},
			},
			errContains: "Invalid API key",
		},
		{
			name:       "handles 403 forbidden",
			statusCode: http.StatusForbidden,
			response: map[string]interface{}{
				"error": map[string]string{"message": "Access denied"},
			},
			errContains: "Access denied",
		},
		{
			name:       "handles 500 server error",
			statusCode: http.StatusInternalServerError,
			response: map[string]interface{}{
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
