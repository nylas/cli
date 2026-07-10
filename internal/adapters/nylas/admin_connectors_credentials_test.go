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

func TestHTTPClient_ListConnectors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":       "conn-1",
					"name":     "Google Connector",
					"provider": "google",
				},
				{
					"id":       "conn-2",
					"name":     "Microsoft Connector",
					"provider": "microsoft",
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
	connectors, err := client.ListConnectors(ctx)

	require.NoError(t, err)
	assert.Len(t, connectors, 2)
	assert.Equal(t, "conn-1", connectors[0].ID)
	assert.Equal(t, "google", connectors[0].Provider)
}

func TestHTTPClient_GetConnector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":       "conn-123",
				"name":     "IMAP Connector",
				"provider": "imap",
				"settings": map[string]any{
					"imap_host": "imap.example.com",
					"imap_port": 993,
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
	connector, err := client.GetConnector(ctx, "conn-123")

	require.NoError(t, err)
	assert.Equal(t, "conn-123", connector.ID)
	assert.Equal(t, "imap", connector.Provider)
	assert.NotNil(t, connector.Settings)
}

func TestHTTPClient_GetConnector_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	conn, err := client.GetConnector(ctx, "")

	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "connector ID")
}

func TestHTTPClient_CreateConnector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Connector", body["name"])
		assert.Equal(t, "google", body["provider"])

		response := map[string]any{
			"data": map[string]any{
				"id":       "conn-new",
				"name":     "New Connector",
				"provider": "google",
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
	req := &domain.CreateConnectorRequest{
		Name:     "New Connector",
		Provider: "google",
	}
	connector, err := client.CreateConnector(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "conn-new", connector.ID)
	assert.Equal(t, "google", connector.Provider)
}

func TestHTTPClient_UpdateConnector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-789", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Connector", body["name"])

		response := map[string]any{
			"data": map[string]any{
				"id":   "conn-789",
				"name": "Updated Connector",
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
	name := "Updated Connector"
	req := &domain.UpdateConnectorRequest{
		Name: &name,
	}
	connector, err := client.UpdateConnector(ctx, "conn-789", req)

	require.NoError(t, err)
	assert.Equal(t, "conn-789", connector.ID)
}

func TestHTTPClient_UpdateConnector_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	name := "Test"
	req := &domain.UpdateConnectorRequest{Name: &name}

	conn, err := client.UpdateConnector(ctx, "", req)

	require.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "connector ID")
}

func TestHTTPClient_DeleteConnector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteConnector(ctx, "conn-delete")

	require.NoError(t, err)
}

func TestHTTPClient_DeleteConnector_EmptyID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	err := client.DeleteConnector(ctx, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "connector ID")
}

// Connector Credential Tests

func TestHTTPClient_ListCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123/creds", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{"id": "cred-1", "name": "First Credential"},
				{"id": "cred-2", "name": "Second Credential"},
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
	credentials, err := client.ListCredentials(ctx, "conn-123")

	require.NoError(t, err)
	assert.Len(t, credentials, 2)
	assert.Equal(t, "cred-1", credentials[0].ID)
	assert.Equal(t, "First Credential", credentials[0].Name)
}

func TestHTTPClient_ListCredentials_Paginates(t *testing.T) {
	// A full first page (len == limit) must trigger a follow-up page; a short
	// second page ends it. Guards against silently truncating at the default 10.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123/creds", r.URL.Path)
		limit := r.URL.Query().Get("limit")
		require.NotEmpty(t, limit, "list must send a limit")
		w.Header().Set("Content-Type", "application/json")
		// Page 1 omits offset (defaults to 0); page 2 sends offset=200.
		if off := r.URL.Query().Get("offset"); off == "" || off == "0" {
			full := make([]map[string]any, 200)
			for i := range full {
				full[i] = map[string]any{"id": "cred", "name": "c"}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": full, "limit": 200})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":  []map[string]any{{"id": "cred-last", "name": "last"}},
			"limit": 200,
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	credentials, err := client.ListCredentials(context.Background(), "conn-123")

	require.NoError(t, err)
	assert.Len(t, credentials, 201, "all pages must be aggregated")
	assert.Equal(t, "cred-last", credentials[200].ID)
}

func TestHTTPClient_ListCredentials_EmptyReturnsNonNilSlice(t *testing.T) {
	// An empty result must be a non-nil slice so JSON output is `[]`, not `null`.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	creds, err := client.ListCredentials(context.Background(), "conn-123")

	require.NoError(t, err)
	assert.NotNil(t, creds, "empty result must be a non-nil slice (marshals to [] not null)")
	assert.Empty(t, creds)
}

func TestHTTPClient_ListCredentials_EmptyConnectorID(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	ctx := context.Background()
	creds, err := client.ListCredentials(ctx, "")

	require.Error(t, err)
	assert.Nil(t, creds)
	assert.Contains(t, err.Error(), "connector ID")
}

func TestHTTPClient_GetCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123/creds/cred-456", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":   "cred-456",
				"name": "Test Credential",
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
	credential, err := client.GetCredential(ctx, "conn-123", "cred-456")

	require.NoError(t, err)
	assert.Equal(t, "cred-456", credential.ID)
	assert.Equal(t, "Test Credential", credential.Name)
}

func TestHTTPClient_CreateCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123/creds", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Credential", body["name"])
		// credential_type is a request-only field (not echoed in the response).
		assert.Equal(t, "connector", body["credential_type"])

		response := map[string]any{
			"data": map[string]any{
				"id":   "cred-new",
				"name": "New Credential",
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
	req := &domain.CreateCredentialRequest{
		Name:           "New Credential",
		CredentialType: "connector",
	}
	credential, err := client.CreateCredential(ctx, "conn-123", req)

	require.NoError(t, err)
	assert.Equal(t, "cred-new", credential.ID)
	assert.Equal(t, "New Credential", credential.Name)
}

func TestHTTPClient_UpdateCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123/creds/cred-789", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Credential", body["name"])

		response := map[string]any{
			"data": map[string]any{
				"id":   "cred-789",
				"name": "Updated Credential",
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
	name := "Updated Credential"
	req := &domain.UpdateCredentialRequest{
		Name: &name,
	}
	credential, err := client.UpdateCredential(ctx, "conn-123", "cred-789", req)

	require.NoError(t, err)
	assert.Equal(t, "cred-789", credential.ID)
}

func TestHTTPClient_DeleteCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123/creds/cred-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteCredential(ctx, "conn-123", "cred-delete")

	require.NoError(t, err)
}

// Grant Administration Tests
