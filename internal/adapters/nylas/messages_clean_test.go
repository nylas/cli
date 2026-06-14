//go:build !integration
// +build !integration

package nylas_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_CleanMessages(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/v3/grants/grant-123/messages/clean", r.URL.Path)
		_ = json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":           "msg-1",
					"grant_id":     "grant-123",
					"object":       "message",
					"subject":      "Re: Lunch",
					"conversation": "Sounds good, see you at noon.",
					"body":         "<div>Sounds good, see you at noon.</div><div>On Monday X wrote...</div>",
				},
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	keepLinks := false
	req := &domain.CleanMessagesRequest{
		MessageIDs:  []string{"msg-1"},
		IgnoreLinks: &keepLinks,
	}
	cleaned, err := client.CleanMessages(context.Background(), "grant-123", req)

	require.NoError(t, err)
	require.Len(t, cleaned, 1)
	// The whole point of clean is the parsed conversation text — assert it,
	// not just that a request happened.
	assert.Equal(t, "Sounds good, see you at noon.", cleaned[0].Conversation)
	assert.Equal(t, "msg-1", cleaned[0].ID)

	// The request must carry the exact message IDs and only the options the
	// caller set — a regression that sent the wrong IDs or leaked defaults must fail.
	assert.Equal(t, []any{"msg-1"}, body["message_id"])
	assert.Equal(t, false, body["ignore_links"])
	assert.NotContains(t, body, "images_as_markdown", "unset options must be omitted so API defaults apply")
}

func TestHTTPClient_CleanMessages_Validation(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	// No base URL set: validation must fail before any network call.

	t.Run("rejects empty message list", func(t *testing.T) {
		_, err := client.CleanMessages(context.Background(), "grant-123", &domain.CleanMessagesRequest{})
		assert.Error(t, err)
	})

	t.Run("rejects more than the API maximum", func(t *testing.T) {
		ids := make([]string, domain.CleanMessagesMaxIDs+1)
		for i := range ids {
			ids[i] = "m"
		}
		_, err := client.CleanMessages(context.Background(), "grant-123", &domain.CleanMessagesRequest{MessageIDs: ids})
		require.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "at most"), "error should explain the 20-ID limit, got: %v", err)
	})

	t.Run("requires a grant", func(t *testing.T) {
		_, err := client.CleanMessages(context.Background(), "", &domain.CleanMessagesRequest{MessageIDs: []string{"m1"}})
		assert.Error(t, err)
	})
}
