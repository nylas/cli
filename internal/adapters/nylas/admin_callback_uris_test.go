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

// Callback URI Tests

func TestHTTPClient_ListCallbackURIs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications/callback-uris", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":       "cb-1",
					"url":      "http://localhost:9007/callback",
					"platform": "web",
				},
				{
					"id":       "cb-2",
					"url":      "https://myapp.com/oauth/callback",
					"platform": "web",
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
	uris, err := client.ListCallbackURIs(ctx)

	require.NoError(t, err)
	assert.Len(t, uris, 2)
	assert.Equal(t, "cb-1", uris[0].ID)
	assert.Equal(t, "http://localhost:9007/callback", uris[0].URL)
	assert.Equal(t, "web", uris[0].Platform)
	assert.Equal(t, "cb-2", uris[1].ID)
}

func TestHTTPClient_ListCallbackURIs_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"data": []map[string]any{},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	uris, err := client.ListCallbackURIs(ctx)

	require.NoError(t, err)
	assert.Empty(t, uris)
}

func TestHTTPClient_GetCallbackURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications/callback-uris/cb-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":       "cb-123",
				"url":      "http://localhost:9007/callback",
				"platform": "web",
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
	uri, err := client.GetCallbackURI(ctx, "cb-123")

	require.NoError(t, err)
	assert.Equal(t, "cb-123", uri.ID)
	assert.Equal(t, "http://localhost:9007/callback", uri.URL)
	assert.Equal(t, "web", uri.Platform)
}

func TestHTTPClient_GetCallbackURI_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	uri, err := client.GetCallbackURI(ctx, "")

	require.Error(t, err)
	assert.Nil(t, uri)
	assert.Contains(t, err.Error(), "callback URI ID")
}

func TestHTTPClient_CreateCallbackURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications/callback-uris", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "http://localhost:9007/callback", body["url"])
		assert.Equal(t, "web", body["platform"])

		response := map[string]any{
			"data": map[string]any{
				"id":       "cb-new",
				"url":      "http://localhost:9007/callback",
				"platform": "web",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.CreateCallbackURIRequest{
		URL:      "http://localhost:9007/callback",
		Platform: "web",
	}
	uri, err := client.CreateCallbackURI(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "cb-new", uri.ID)
	assert.Equal(t, "http://localhost:9007/callback", uri.URL)
	assert.Equal(t, "web", uri.Platform)
}

func TestHTTPClient_UpdateCallbackURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications/callback-uris/cb-456", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "https://myapp.com/new-callback", body["url"])

		response := map[string]any{
			"data": map[string]any{
				"id":       "cb-456",
				"url":      "https://myapp.com/new-callback",
				"platform": "web",
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
	newURL := "https://myapp.com/new-callback"
	req := &domain.UpdateCallbackURIRequest{
		URL: &newURL,
	}
	uri, err := client.UpdateCallbackURI(ctx, "cb-456", req)

	require.NoError(t, err)
	assert.Equal(t, "cb-456", uri.ID)
	assert.Equal(t, "https://myapp.com/new-callback", uri.URL)
}

func TestHTTPClient_UpdateCallbackURI_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	testURL := "http://test.com"
	req := &domain.UpdateCallbackURIRequest{URL: &testURL}
	uri, err := client.UpdateCallbackURI(ctx, "", req)

	require.Error(t, err)
	assert.Nil(t, uri)
	assert.Contains(t, err.Error(), "callback URI ID")
}

func TestHTTPClient_DeleteCallbackURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications/callback-uris/cb-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteCallbackURI(ctx, "cb-delete")

	require.NoError(t, err)
}

func TestHTTPClient_DeleteCallbackURI_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	err := client.DeleteCallbackURI(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "callback URI ID")
}

func TestHTTPClient_ListCallbackURIs_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "api.server_error",
				"message": "internal server error",
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	uris, err := client.ListCallbackURIs(ctx)

	require.Error(t, err)
	assert.Nil(t, uris)
}

func TestHTTPClient_GetCallbackURI_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "api.not_found_error",
				"message": "RedirectURI not found",
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	uri, err := client.GetCallbackURI(ctx, "nonexistent")

	require.Error(t, err)
	assert.Nil(t, uri)
}
