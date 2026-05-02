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

func TestConvertThread(t *testing.T) {
	now := time.Now().Unix()

	apiThread := threadResponse{
		ID:                    "thread-123",
		GrantID:               "grant-456",
		HasAttachments:        true,
		HasDrafts:             false,
		Starred:               true,
		Unread:                false,
		EarliestMessageDate:   now - 3600,
		LatestMessageRecvDate: now - 1800,
		LatestMessageSentDate: now,
		Participants: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			{Name: "Alice", Email: "alice@example.com"},
			{Name: "Bob", Email: "bob@example.com"},
		},
		MessageIDs: []string{"msg-1", "msg-2", "msg-3"},
		DraftIDs:   []string{},
		FolderIDs:  []string{"folder-1"},
		Snippet:    "This is a preview...",
		Subject:    "Important Discussion",
	}

	thread := convertThread(apiThread)

	assert.Equal(t, "thread-123", thread.ID)
	assert.Equal(t, "grant-456", thread.GrantID)
	assert.True(t, thread.HasAttachments)
	assert.False(t, thread.HasDrafts)
	assert.True(t, thread.Starred)
	assert.False(t, thread.Unread)
	assert.Equal(t, time.Unix(now-3600, 0), thread.EarliestMessageDate)
	assert.Equal(t, time.Unix(now-1800, 0), thread.LatestMessageRecvDate)
	assert.Equal(t, time.Unix(now, 0), thread.LatestMessageSentDate)

	// Test participants conversion using util.Map
	assert.Len(t, thread.Participants, 2)
	assert.Equal(t, "Alice", thread.Participants[0].Name)
	assert.Equal(t, "alice@example.com", thread.Participants[0].Email)
	assert.Equal(t, "Bob", thread.Participants[1].Name)
	assert.Equal(t, "bob@example.com", thread.Participants[1].Email)

	assert.Equal(t, []string{"msg-1", "msg-2", "msg-3"}, thread.MessageIDs)
	assert.Equal(t, []string{}, thread.DraftIDs)
	assert.Equal(t, []string{"folder-1"}, thread.FolderIDs)
	assert.Equal(t, "This is a preview...", thread.Snippet)
	assert.Equal(t, "Important Discussion", thread.Subject)
}

func TestConvertThreads(t *testing.T) {
	now := time.Now().Unix()

	apiThreads := []threadResponse{
		{
			ID:      "thread-1",
			GrantID: "grant-1",
			Subject: "Thread One",
			Participants: []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{
				{Name: "User1", Email: "user1@example.com"},
			},
			EarliestMessageDate:   now,
			LatestMessageRecvDate: now,
			LatestMessageSentDate: now,
		},
		{
			ID:      "thread-2",
			GrantID: "grant-2",
			Subject: "Thread Two",
			Participants: []struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}{
				{Name: "User2", Email: "user2@example.com"},
			},
			EarliestMessageDate:   now,
			LatestMessageRecvDate: now,
			LatestMessageSentDate: now,
		},
	}

	// Test convertThreads uses util.Map
	threads := convertThreads(apiThreads)

	assert.Len(t, threads, 2)
	assert.Equal(t, "thread-1", threads[0].ID)
	assert.Equal(t, "Thread One", threads[0].Subject)
	assert.Equal(t, "thread-2", threads[1].ID)
	assert.Equal(t, "Thread Two", threads[1].Subject)

	// Verify participants were converted correctly
	assert.Len(t, threads[0].Participants, 1)
	assert.Equal(t, "User1", threads[0].Participants[0].Name)
	assert.Len(t, threads[1].Participants, 1)
	assert.Equal(t, "User2", threads[1].Participants[0].Name)
}

func TestConvertThreads_Empty(t *testing.T) {
	// Test with empty slice
	threads := convertThreads([]threadResponse{})
	assert.NotNil(t, threads)
	assert.Len(t, threads, 0)
}

func TestConvertThread_EmptyParticipants(t *testing.T) {
	now := time.Now().Unix()

	apiThread := threadResponse{
		ID:      "thread-empty",
		GrantID: "grant-123",
		Subject: "No Participants",
		Participants: []struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{},
		EarliestMessageDate:   now,
		LatestMessageRecvDate: now,
		LatestMessageSentDate: now,
	}

	thread := convertThread(apiThread)

	assert.Equal(t, "thread-empty", thread.ID)
	assert.Equal(t, "No Participants", thread.Subject)
	assert.NotNil(t, thread.Participants)
	assert.Len(t, thread.Participants, 0)
}

// HTTP Client Tests

func TestHTTPClient_GetThreads(t *testing.T) {
	now := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/threads", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":                           "thread-1",
					"grant_id":                     "grant-123",
					"subject":                      "Meeting Notes",
					"snippet":                      "Let's discuss...",
					"starred":                      true,
					"unread":                       false,
					"has_attachments":              true,
					"has_drafts":                   false,
					"earliest_message_date":        now - 7200,
					"latest_message_received_date": now - 3600,
					"latest_message_sent_date":     now,
					"participants": []map[string]string{
						{"name": "Alice", "email": "alice@example.com"},
					},
					"message_ids": []string{"msg-1", "msg-2"},
					"draft_ids":   []string{},
					"folders":     []string{"folder-inbox"},
				},
			},
			"request_id": "req-123",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	threads, err := client.GetThreads(ctx, "grant-123", nil)

	require.NoError(t, err)
	assert.Len(t, threads, 1)
	assert.Equal(t, "thread-1", threads[0].ID)
	assert.Equal(t, "Meeting Notes", threads[0].Subject)
	assert.True(t, threads[0].Starred)
	assert.False(t, threads[0].Unread)
	assert.Len(t, threads[0].Participants, 1)
	assert.Equal(t, "Alice", threads[0].Participants[0].Name)
}

func TestHTTPClient_GetThreads_WithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-filter/threads", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Check query params
		assert.Equal(t, "20", r.URL.Query().Get("limit"))
		assert.Equal(t, "5", r.URL.Query().Get("offset"))
		assert.Equal(t, "Important", r.URL.Query().Get("subject"))
		assert.Equal(t, "alice@example.com", r.URL.Query().Get("from"))
		assert.Equal(t, "bob@example.com", r.URL.Query().Get("to"))
		assert.Equal(t, "true", r.URL.Query().Get("unread"))
		assert.Equal(t, "false", r.URL.Query().Get("starred"))
		assert.Equal(t, "project X", r.URL.Query().Get("q"))

		response := map[string]any{
			"data":       []map[string]any{},
			"request_id": "req-filter",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	unread := true
	starred := false
	params := &domain.ThreadQueryParams{
		Limit:       20,
		Offset:      5,
		Subject:     "Important",
		From:        "alice@example.com",
		To:          "bob@example.com",
		Unread:      &unread,
		Starred:     &starred,
		SearchQuery: "project X",
	}
	threads, err := client.GetThreads(ctx, "grant-filter", params)

	require.NoError(t, err)
	assert.Len(t, threads, 0)
}

func TestHTTPClient_GetThread(t *testing.T) {
	now := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-456/threads/thread-abc", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":                           "thread-abc",
				"grant_id":                     "grant-456",
				"subject":                      "Project Update",
				"snippet":                      "Here's the latest...",
				"starred":                      false,
				"unread":                       true,
				"has_attachments":              true,
				"has_drafts":                   true,
				"earliest_message_date":        now - 86400,
				"latest_message_received_date": now - 1800,
				"latest_message_sent_date":     now,
				"participants": []map[string]string{
					{"name": "Bob", "email": "bob@example.com"},
					{"name": "Charlie", "email": "charlie@example.com"},
				},
				"message_ids": []string{"msg-10", "msg-11", "msg-12"},
				"draft_ids":   []string{"draft-5"},
				"folders":     []string{"folder-work", "folder-important"},
			},
			"request_id": "req-456",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	thread, err := client.GetThread(ctx, "grant-456", "thread-abc")

	require.NoError(t, err)
	assert.Equal(t, "thread-abc", thread.ID)
	assert.Equal(t, "Project Update", thread.Subject)
	assert.False(t, thread.Starred)
	assert.True(t, thread.Unread)
	assert.True(t, thread.HasAttachments)
	assert.True(t, thread.HasDrafts)
	assert.Len(t, thread.Participants, 2)
	assert.Equal(t, "Bob", thread.Participants[0].Name)
	assert.Equal(t, "Charlie", thread.Participants[1].Name)
	assert.Len(t, thread.MessageIDs, 3)
	assert.Len(t, thread.DraftIDs, 1)
	assert.Len(t, thread.FolderIDs, 2)
}

func TestHTTPClient_GetThread_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"message": "Thread not found",
			},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	thread, err := client.GetThread(ctx, "grant-123", "nonexistent")

	require.Error(t, err)
	assert.Nil(t, thread)
	assert.ErrorIs(t, err, domain.ErrThreadNotFound)
}

func TestHTTPClient_UpdateThread(t *testing.T) {
	now := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-update/threads/thread-789", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, false, body["unread"])
		assert.Equal(t, true, body["starred"])

		response := map[string]any{
			"data": map[string]any{
				"id":                           "thread-789",
				"grant_id":                     "grant-update",
				"subject":                      "Marked Important",
				"starred":                      true,
				"unread":                       false,
				"has_attachments":              false,
				"has_drafts":                   false,
				"earliest_message_date":        now,
				"latest_message_received_date": now,
				"latest_message_sent_date":     now,
				"participants":                 []map[string]string{},
				"message_ids":                  []string{"msg-20"},
				"draft_ids":                    []string{},
				"folders":                      []string{"folder-starred"},
			},
			"request_id": "req-update",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	unread := false
	starred := true
	req := &domain.UpdateMessageRequest{
		Unread:  &unread,
		Starred: &starred,
	}

	thread, err := client.UpdateThread(ctx, "grant-update", "thread-789", req)

	require.NoError(t, err)
	assert.Equal(t, "thread-789", thread.ID)
	assert.True(t, thread.Starred)
	assert.False(t, thread.Unread)
}

func TestHTTPClient_UpdateThread_WithFolders(t *testing.T) {
	now := time.Now().Unix()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-folders/threads/thread-move", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		folders, ok := body["folders"].([]any)
		assert.True(t, ok)
		assert.Contains(t, folders, "folder-archive")
		assert.Contains(t, folders, "folder-done")

		response := map[string]any{
			"data": map[string]any{
				"id":                           "thread-move",
				"grant_id":                     "grant-folders",
				"subject":                      "Archived Thread",
				"starred":                      false,
				"unread":                       false,
				"has_attachments":              false,
				"has_drafts":                   false,
				"earliest_message_date":        now,
				"latest_message_received_date": now,
				"latest_message_sent_date":     now,
				"participants":                 []map[string]string{},
				"message_ids":                  []string{"msg-30"},
				"draft_ids":                    []string{},
				"folders":                      []string{"folder-archive", "folder-done"},
			},
			"request_id": "req-folders",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	folders := []string{"folder-archive", "folder-done"}
	req := &domain.UpdateMessageRequest{
		Folders: folders,
	}

	thread, err := client.UpdateThread(ctx, "grant-folders", "thread-move", req)

	require.NoError(t, err)
	assert.Equal(t, "thread-move", thread.ID)
	assert.Len(t, thread.FolderIDs, 2)
	assert.Contains(t, thread.FolderIDs, "folder-archive")
	assert.Contains(t, thread.FolderIDs, "folder-done")
}

func TestHTTPClient_DeleteThread(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-delete/threads/thread-del", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteThread(ctx, "grant-delete", "thread-del")

	require.NoError(t, err)
}

func TestHTTPClient_DeleteThread_EmptyThreadID(t *testing.T) {
	client := NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	err := client.DeleteThread(ctx, "grant-123", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "thread ID")
}
