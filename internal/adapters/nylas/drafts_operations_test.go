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

func TestHTTPClient_GetDrafts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/drafts", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":       "draft-1",
					"grant_id": "grant-123",
					"subject":  "Draft 1",
					"body":     "Body 1",
					"to":       []map[string]string{{"email": "user1@example.com"}},
				},
				{
					"id":       "draft-2",
					"grant_id": "grant-123",
					"subject":  "Draft 2",
					"body":     "Body 2",
					"to":       []map[string]string{{"email": "user2@example.com"}},
				},
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
	drafts, err := client.GetDrafts(ctx, "grant-123", 10)

	require.NoError(t, err)
	assert.Len(t, drafts, 2)
	assert.Equal(t, "draft-1", drafts[0].ID)
	assert.Equal(t, "Draft 1", drafts[0].Subject)
	assert.Equal(t, "draft-2", drafts[1].ID)
}

func TestHTTPClient_GetDraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/drafts/draft-abc", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":       "draft-abc",
				"grant_id": "grant-123",
				"subject":  "Important Draft",
				"body":     "Draft content here",
				"from":     []map[string]string{{"email": "sender@example.com", "name": "Sender"}},
				"to":       []map[string]string{{"email": "recipient@example.com", "name": "Recipient"}},
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
	draft, err := client.GetDraft(ctx, "grant-123", "draft-abc")

	require.NoError(t, err)
	assert.Equal(t, "draft-abc", draft.ID)
	assert.Equal(t, "Important Draft", draft.Subject)
	assert.Equal(t, "Draft content here", draft.Body)
	assert.Len(t, draft.To, 1)
	assert.Equal(t, "recipient@example.com", draft.To[0].Email)
}

func TestHTTPClient_GetDraft_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "Draft not found"},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	draft, err := client.GetDraft(ctx, "grant-123", "nonexistent")

	require.Error(t, err)
	assert.Nil(t, draft)
	assert.ErrorIs(t, err, domain.ErrDraftNotFound)
}

func TestHTTPClient_DeleteDraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/drafts/draft-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteDraft(ctx, "grant-123", "draft-delete")

	require.NoError(t, err)
}

func TestHTTPClient_SendDraft(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/drafts/draft-send", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		response := map[string]any{
			"data": map[string]any{
				"id":       "msg-sent-123",
				"grant_id": "grant-123",
				"subject":  "Sent Draft",
				"body":     "This was sent",
				"from":     []map[string]string{{"email": "sender@example.com"}},
				"to":       []map[string]string{{"email": "recipient@example.com"}},
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
	message, err := client.SendDraft(ctx, "grant-123", "draft-send", nil)

	require.NoError(t, err)
	assert.Equal(t, "msg-sent-123", message.ID)
	assert.Equal(t, "Sent Draft", message.Subject)
	assert.Equal(t, "This was sent", message.Body)
}

func TestHTTPClient_SendDraft_WithSignatureID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/drafts/draft-send", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]string
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "sig-123", body["signature_id"])

		response := map[string]any{
			"data": map[string]any{
				"id":       "msg-sent-456",
				"grant_id": "grant-123",
				"subject":  "Signed Draft",
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
	message, err := client.SendDraft(ctx, "grant-123", "draft-send", &domain.SendDraftRequest{SignatureID: "sig-123"})

	require.NoError(t, err)
	assert.Equal(t, "msg-sent-456", message.ID)
	assert.Equal(t, "Signed Draft", message.Subject)
}
