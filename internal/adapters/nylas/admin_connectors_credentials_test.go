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

		response := map[string]interface{}{
			"data": []map[string]interface{}{
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

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":       "conn-123",
				"name":     "IMAP Connector",
				"provider": "imap",
				"settings": map[string]interface{}{
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

func TestHTTPClient_CreateConnector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Connector", body["name"])
		assert.Equal(t, "google", body["provider"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
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

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Connector", body["name"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
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

// Connector Credential Tests

func TestHTTPClient_ListCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123/credentials", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":              "cred-1",
					"name":            "OAuth Credential",
					"credential_type": "oauth",
				},
				{
					"id":              "cred-2",
					"name":            "Service Account",
					"credential_type": "service_account",
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
	credentials, err := client.ListCredentials(ctx, "conn-123")

	require.NoError(t, err)
	assert.Len(t, credentials, 2)
	assert.Equal(t, "cred-1", credentials[0].ID)
	assert.Equal(t, "oauth", credentials[0].CredentialType)
}

func TestHTTPClient_GetCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/credentials/cred-456", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":              "cred-456",
				"name":            "Test Credential",
				"credential_type": "oauth",
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
	credential, err := client.GetCredential(ctx, "cred-456")

	require.NoError(t, err)
	assert.Equal(t, "cred-456", credential.ID)
	assert.Equal(t, "oauth", credential.CredentialType)
}

func TestHTTPClient_CreateCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connectors/conn-123/credentials", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Credential", body["name"])
		assert.Equal(t, "oauth", body["credential_type"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":              "cred-new",
				"name":            "New Credential",
				"credential_type": "oauth",
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
		CredentialType: "oauth",
	}
	credential, err := client.CreateCredential(ctx, "conn-123", req)

	require.NoError(t, err)
	assert.Equal(t, "cred-new", credential.ID)
	assert.Equal(t, "oauth", credential.CredentialType)
}

func TestHTTPClient_UpdateCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/credentials/cred-789", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Credential", body["name"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
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
	credential, err := client.UpdateCredential(ctx, "cred-789", req)

	require.NoError(t, err)
	assert.Equal(t, "cred-789", credential.ID)
}

func TestHTTPClient_DeleteCredential(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/credentials/cred-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteCredential(ctx, "cred-delete")

	require.NoError(t, err)
}

// Grant Administration Tests
