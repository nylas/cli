//go:build !integration
// +build !integration

package nylas_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_UpdateMessage(t *testing.T) {
	tests := []struct {
		name       string
		request    *domain.UpdateMessageRequest
		wantFields map[string]any
	}{
		{
			name: "marks as read",
			request: func() *domain.UpdateMessageRequest {
				unread := false
				return &domain.UpdateMessageRequest{Unread: &unread}
			}(),
			wantFields: map[string]any{"unread": false},
		},
		{
			name: "marks as starred",
			request: func() *domain.UpdateMessageRequest {
				starred := true
				return &domain.UpdateMessageRequest{Starred: &starred}
			}(),
			wantFields: map[string]any{"starred": true},
		},
		{
			name: "moves to folders",
			request: func() *domain.UpdateMessageRequest {
				folders := []string{"Archive", "Important"}
				return &domain.UpdateMessageRequest{Folders: folders}
			}(),
			wantFields: map[string]any{"folders": []string{"Archive", "Important"}},
		},
		{
			// Regression: Gmail archive (drop INBOX) sends folders:[]. The
			// adapter must forward an explicit empty array — silently
			// dropping it leaves the message in the inbox while the UI
			// reports success.
			name: "archives to empty folders (gmail)",
			request: func() *domain.UpdateMessageRequest {
				empty := []string{}
				return &domain.UpdateMessageRequest{Folders: empty}
			}(),
			wantFields: map[string]any{"folders": []any{}},
		},
		{
			name: "updates multiple fields",
			request: func() *domain.UpdateMessageRequest {
				unread := true
				starred := true
				folders := []string{"INBOX"}
				return &domain.UpdateMessageRequest{
					Unread:  &unread,
					Starred: &starred,
					Folders: folders,
				}
			}(),
			wantFields: map[string]any{
				"unread":  true,
				"starred": true,
				"folders": []string{"INBOX"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "PUT", r.Method)
				assert.Equal(t, "/v3/grants/grant-123/messages/msg-456", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)

				for key := range tt.wantFields {
					assert.Contains(t, body, key, "Missing field: %s", key)
				}

				response := map[string]any{
					"data": map[string]any{
						"id":       "msg-456",
						"grant_id": "grant-123",
						"subject":  "Updated",
						"date":     1704067200,
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
			message, err := client.UpdateMessage(ctx, "grant-123", "msg-456", tt.request)

			require.NoError(t, err)
			assert.Equal(t, "msg-456", message.ID)
		})
	}
}

// TestHTTPClient_UpdateMessage_ForwardsEmptyFolders pins the Gmail-archive
// contract: when the caller passes []string{} (drop all labels), the PUT
// body MUST contain "folders":[] verbatim. The previous len()>0 guard
// silently elided the array, so archive succeeded in the UI but never
// happened upstream — a particularly nasty class of bug because the
// browser optimistically removes the row before the server lies. Today
// the adapter uses `!= nil` so a non-nil empty slice forwards correctly
// while nil is skipped (leave-alone).
func TestHTTPClient_UpdateMessage_ForwardsEmptyFolders(t *testing.T) {
	t.Parallel()

	var rawBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rawBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":       "msg-456",
				"grant_id": "grant-123",
				"subject":  "Archived",
				"date":     1704067200,
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	_, err := client.UpdateMessage(context.Background(), "grant-123", "msg-456", &domain.UpdateMessageRequest{
		Folders: []string{},
	})
	require.NoError(t, err)

	// Decode and assert the exact key shape — a nil Folders is skipped
	// by the adapter so the key would be absent; only a non-nil empty
	// slice produces "folders":[].
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(rawBody), &parsed))
	folders, present := parsed["folders"]
	require.True(t, present, "folders key must be present so Gmail archive reaches the API; raw=%s", rawBody)
	assert.Equal(t, []any{}, folders, "folders must serialize as an explicit empty array; got %#v", folders)
}

// TestHTTPClient_UpdateMessage_EmptyFolders_PropagatesUpstreamError pins
// that a 4xx from Nylas on an archive PUT (e.g. Gmail rate limit, label
// permission denied) is surfaced as a real error rather than silently
// reported as success. The optimistic UI in Air relies on this — without
// error propagation the row stays visually archived while the server
// rejects the change, and the next refresh re-introduces the email.
func TestHTTPClient_UpdateMessage_EmptyFolders_PropagatesUpstreamError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pin that we even saw the request (no early exit on empty
		// folders) — a regression that elided the body would never
		// reach this assertion.
		raw, _ := io.ReadAll(r.Body)
		assert.Contains(t, string(raw), `"folders":[]`,
			"adapter must still send the empty-folders body even when the upstream errors; raw=%s", string(raw))
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"label permission denied"}`))
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	_, err := client.UpdateMessage(context.Background(), "grant-123", "msg-456", &domain.UpdateMessageRequest{
		Folders: []string{},
	})
	require.Error(t, err, "4xx upstream must surface as a real error, not nil")
}

// TestHTTPClient_UpdateMessage_EmptyFolders_PropagatesServerError pins
// the 5xx path: a transient Nylas outage during archive must not be
// silently swallowed. Air's offline queue keys off this error to retry
// the action when connectivity returns.
func TestHTTPClient_UpdateMessage_EmptyFolders_PropagatesServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream timeout"}`))
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	_, err := client.UpdateMessage(context.Background(), "grant-123", "msg-456", &domain.UpdateMessageRequest{
		Folders: []string{},
	})
	require.Error(t, err, "5xx upstream must surface as a real error so the offline queue can retry")
}

func TestHTTPClient_UpdateMessage_RetriesReplayBody(t *testing.T) {
	t.Parallel()

	var requestBodies []string
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requestBodies = append(requestBodies, string(body))

		w.Header().Set("Content-Type", "application/json")
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"message": "Rate limit exceeded"},
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":       "msg-456",
				"grant_id": "grant-123",
				"subject":  "Updated",
				"date":     1704067200,
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)
	client.SetMaxRetries(1)

	unread := false
	message, err := client.UpdateMessage(context.Background(), "grant-123", "msg-456", &domain.UpdateMessageRequest{
		Unread: &unread,
	})

	require.NoError(t, err)
	require.NotNil(t, message)
	require.Len(t, requestBodies, 2)
	assert.NotEmpty(t, requestBodies[0])
	assert.Equal(t, requestBodies[0], requestBodies[1], "retried request should replay the original JSON body")
}

func TestHTTPClient_DeleteMessage(t *testing.T) {
	tests := []struct {
		name       string
		grantID    string
		messageID  string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "deletes successfully with 200",
			grantID:    "grant-123",
			messageID:  "msg-456",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "deletes successfully with 204",
			grantID:    "grant-123",
			messageID:  "msg-789",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "returns error for not found",
			grantID:    "grant-123",
			messageID:  "nonexistent",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				expectedPath := "/v3/grants/" + tt.grantID + "/messages/" + tt.messageID
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
			err := client.DeleteMessage(ctx, tt.grantID, tt.messageID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPClient_GetMessages_ErrorHandling(t *testing.T) {
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
			name:       "handles 429 rate limited",
			statusCode: http.StatusTooManyRequests,
			response: map[string]any{
				"error": map[string]string{"message": "Rate limit exceeded"},
			},
			errContains: "Rate limit exceeded",
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
			_, err := client.GetMessages(ctx, "grant-123", 10)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestHTTPClient_GetMessage_FullConversion(t *testing.T) {
	timestamp := time.Now().Unix()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"data": map[string]any{
				"id":        "msg-full",
				"grant_id":  "grant-full",
				"thread_id": "thread-full",
				"subject":   "Complete Message",
				"from": []map[string]string{
					{"name": "Alice Smith", "email": "alice@example.com"},
				},
				"to": []map[string]string{
					{"name": "Bob Jones", "email": "bob@example.com"},
					{"name": "Carol White", "email": "carol@example.com"},
				},
				"cc": []map[string]string{
					{"name": "Dave Brown", "email": "dave@example.com"},
				},
				"bcc": []map[string]string{
					{"name": "Eve Black", "email": "eve@example.com"},
				},
				"reply_to": []map[string]string{
					{"name": "Reply Handler", "email": "reply@example.com"},
				},
				"body":    "<html><body><p>Full body content</p></body></html>",
				"snippet": "Full body content",
				"date":    timestamp,
				"unread":  true,
				"starred": false,
				"folders": []string{"INBOX", "Important"},
				"attachments": []map[string]any{
					{
						"id":           "attach-1",
						"filename":     "document.pdf",
						"content_type": "application/pdf",
						"size":         50000,
						"content_id":   "",
						"is_inline":    false,
					},
					{
						"id":           "attach-2",
						"filename":     "image.png",
						"content_type": "image/png",
						"size":         25000,
						"content_id":   "cid:123",
						"is_inline":    true,
					},
				},
				"metadata":   map[string]string{"key1": "value1", "key2": "value2"},
				"created_at": timestamp,
				"object":     "message",
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
	msg, err := client.GetMessage(ctx, "grant-full", "msg-full")

	require.NoError(t, err)

	// Verify all fields
	assert.Equal(t, "msg-full", msg.ID)
	assert.Equal(t, "grant-full", msg.GrantID)
	assert.Equal(t, "thread-full", msg.ThreadID)
	assert.Equal(t, "Complete Message", msg.Subject)

	// From
	require.Len(t, msg.From, 1)
	assert.Equal(t, "Alice Smith", msg.From[0].Name)
	assert.Equal(t, "alice@example.com", msg.From[0].Email)

	// To
	require.Len(t, msg.To, 2)
	assert.Equal(t, "Bob Jones", msg.To[0].Name)
	assert.Equal(t, "Carol White", msg.To[1].Name)

	// CC
	require.Len(t, msg.Cc, 1)
	assert.Equal(t, "Dave Brown", msg.Cc[0].Name)

	// BCC
	require.Len(t, msg.Bcc, 1)
	assert.Equal(t, "Eve Black", msg.Bcc[0].Name)

	// Reply-To
	require.Len(t, msg.ReplyTo, 1)
	assert.Equal(t, "Reply Handler", msg.ReplyTo[0].Name)

	// Body and snippet
	assert.Contains(t, msg.Body, "Full body content")
	assert.Equal(t, "Full body content", msg.Snippet)

	// Flags
	assert.True(t, msg.Unread)
	assert.False(t, msg.Starred)

	// Folders
	assert.Contains(t, msg.Folders, "INBOX")
	assert.Contains(t, msg.Folders, "Important")

	// Attachments
	require.Len(t, msg.Attachments, 2)
	assert.Equal(t, "document.pdf", msg.Attachments[0].Filename)
	assert.Equal(t, "application/pdf", msg.Attachments[0].ContentType)
	assert.False(t, msg.Attachments[0].IsInline)
	assert.Equal(t, "image.png", msg.Attachments[1].Filename)
	assert.True(t, msg.Attachments[1].IsInline)
	assert.Equal(t, "cid:123", msg.Attachments[1].ContentID)

	// Metadata
	assert.Equal(t, "value1", msg.Metadata["key1"])
	assert.Equal(t, "value2", msg.Metadata["key2"])

	// Object type
	assert.Equal(t, "message", msg.Object)

	// Timestamps
	assert.Equal(t, time.Unix(timestamp, 0), msg.Date)
	assert.Equal(t, time.Unix(timestamp, 0), msg.CreatedAt)
}
