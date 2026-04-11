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

func TestHTTPClient_GetSignatures(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/signatures", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":         "sig-1",
					"name":       "Work",
					"body":       "<p>Best regards</p>",
					"created_at": 1704067200,
					"updated_at": 1704068200,
				},
				{
					"id":         "sig-2",
					"name":       "Mobile",
					"body":       "<p>Sent from my phone</p>",
					"created_at": 1704067200,
					"updated_at": 1704069200,
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

	signatures, err := client.GetSignatures(context.Background(), "grant-123")

	require.NoError(t, err)
	require.Len(t, signatures, 2)
	assert.Equal(t, "sig-1", signatures[0].ID)
	assert.Equal(t, "Work", signatures[0].Name)
	assert.Equal(t, "sig-2", signatures[1].ID)
}

func TestHTTPClient_GetSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/signatures/sig-123", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":         "sig-123",
				"name":       "Work",
				"body":       "<p>Best regards</p>",
				"created_at": 1704067200,
				"updated_at": 1704068200,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	signature, err := client.GetSignature(context.Background(), "grant-123", "sig-123")

	require.NoError(t, err)
	require.NotNil(t, signature)
	assert.Equal(t, "sig-123", signature.ID)
	assert.Equal(t, "Work", signature.Name)
}

func TestHTTPClient_GetSignature_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "Signature not found"},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	signature, err := client.GetSignature(context.Background(), "grant-123", "missing")

	require.Error(t, err)
	assert.Nil(t, signature)
	assert.ErrorIs(t, err, domain.ErrSignatureNotFound)
}

func TestHTTPClient_CreateSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/signatures", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var body map[string]string
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "Work", body["name"])
		assert.Equal(t, "<p>Best regards</p>", body["body"])

		response := map[string]any{
			"data": map[string]any{
				"id":         "sig-new",
				"name":       body["name"],
				"body":       body["body"],
				"created_at": 1704067200,
				"updated_at": 1704067200,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	signature, err := client.CreateSignature(context.Background(), "grant-123", &domain.CreateSignatureRequest{
		Name: "Work",
		Body: "<p>Best regards</p>",
	})

	require.NoError(t, err)
	require.NotNil(t, signature)
	assert.Equal(t, "sig-new", signature.ID)
	assert.Equal(t, "Work", signature.Name)
}

func TestHTTPClient_CreateSignature_DoesNotRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		assert.Equal(t, "/v3/grants/grant-123/signatures", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "temporary signature write failure"},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	signature, err := client.CreateSignature(context.Background(), "grant-123", &domain.CreateSignatureRequest{
		Name: "Work",
		Body: "<p>Best regards</p>",
	})

	require.Error(t, err)
	assert.Nil(t, signature)
	assert.Equal(t, 1, attempts)
}

func TestHTTPClient_UpdateSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/signatures/sig-123", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		var body map[string]string
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "Updated", body["name"])
		assert.Equal(t, "<p>Updated body</p>", body["body"])

		response := map[string]any{
			"data": map[string]any{
				"id":         "sig-123",
				"name":       body["name"],
				"body":       body["body"],
				"created_at": 1704067200,
				"updated_at": 1704069200,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	signature, err := client.UpdateSignature(context.Background(), "grant-123", "sig-123", &domain.UpdateSignatureRequest{
		Name: stringPtr("Updated"),
		Body: stringPtr("<p>Updated body</p>"),
	})

	require.NoError(t, err)
	require.NotNil(t, signature)
	assert.Equal(t, "Updated", signature.Name)
	assert.Equal(t, "<p>Updated body</p>", signature.Body)
}

func TestHTTPClient_UpdateSignature_DoesNotRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		assert.Equal(t, "/v3/grants/grant-123/signatures/sig-123", r.URL.Path)
		assert.Equal(t, http.MethodPut, r.Method)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "temporary signature update failure"},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	signature, err := client.UpdateSignature(context.Background(), "grant-123", "sig-123", &domain.UpdateSignatureRequest{
		Name: stringPtr("Updated"),
		Body: stringPtr("<p>Updated body</p>"),
	})

	require.Error(t, err)
	assert.Nil(t, signature)
	assert.Equal(t, 1, attempts)
}

func TestHTTPClient_DeleteSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-123/signatures/sig-123", r.URL.Path)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	err := client.DeleteSignature(context.Background(), "grant-123", "sig-123")

	require.NoError(t, err)
}

func TestMockClient_Signatures(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	signatures, err := mock.GetSignatures(ctx, "grant-123")
	require.NoError(t, err)
	require.Len(t, signatures, 1)
	assert.True(t, mock.GetSignaturesCalled)

	signature, err := mock.GetSignature(ctx, "grant-123", "sig-123")
	require.NoError(t, err)
	require.NotNil(t, signature)
	assert.Equal(t, "sig-123", signature.ID)
	assert.True(t, mock.GetSignatureCalled)
	assert.Equal(t, "sig-123", mock.LastSignatureID)

	created, err := mock.CreateSignature(ctx, "grant-123", &domain.CreateSignatureRequest{
		Name: "New",
		Body: "<p>Body</p>",
	})
	require.NoError(t, err)
	assert.Equal(t, "New", created.Name)
	assert.True(t, mock.CreateSignatureCalled)

	updated, err := mock.UpdateSignature(ctx, "grant-123", "sig-123", &domain.UpdateSignatureRequest{
		Name: stringPtr("Updated"),
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
	assert.True(t, mock.UpdateSignatureCalled)

	err = mock.DeleteSignature(ctx, "grant-123", "sig-123")
	require.NoError(t, err)
	assert.True(t, mock.DeleteSignatureCalled)
}

func TestDemoClient_Signatures(t *testing.T) {
	ctx := context.Background()
	client := nylas.NewDemoClient()

	signatures, err := client.GetSignatures(ctx, "demo-grant")
	require.NoError(t, err)
	require.Len(t, signatures, 2)

	signature, err := client.GetSignature(ctx, "demo-grant", "sig-demo-work")
	require.NoError(t, err)
	require.NotNil(t, signature)
	assert.Equal(t, "sig-demo-work", signature.ID)

	missing, err := client.GetSignature(ctx, "demo-grant", "missing")
	require.Error(t, err)
	assert.Nil(t, missing)
	assert.ErrorIs(t, err, domain.ErrSignatureNotFound)
}

func stringPtr(value string) *string {
	return &value
}
