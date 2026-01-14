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

func TestHTTPClient_GetThreads(t *testing.T) {
	tests := []struct {
		name           string
		params         *domain.ThreadQueryParams
		serverResponse map[string]any
		statusCode     int
		wantCount      int
		wantErr        bool
	}{
		{
			name:   "returns threads successfully",
			params: nil,
			serverResponse: map[string]any{
				"data": []map[string]any{
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
			serverResponse: map[string]any{
				"data": []any{},
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
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []any{},
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
		serverResponse map[string]any
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:     "returns thread successfully",
			threadID: "thread-123",
			serverResponse: map[string]any{
				"data": map[string]any{
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
			serverResponse: map[string]any{
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
