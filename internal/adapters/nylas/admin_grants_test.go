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

func TestHTTPClient_ListAllGrants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":           "grant-1",
					"provider":     "google",
					"email":        "user1@example.com",
					"grant_status": "valid",
				},
				{
					"id":           "grant-2",
					"provider":     "microsoft",
					"email":        "user2@example.com",
					"grant_status": "valid",
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
	grants, err := client.ListAllGrants(ctx, nil)

	require.NoError(t, err)
	assert.Len(t, grants, 2)
	assert.Equal(t, "grant-1", grants[0].ID)
	assert.Equal(t, "google", string(grants[0].Provider))
	assert.Equal(t, "valid", grants[0].GrantStatus)
}

func TestHTTPClient_ListAllGrants_WithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		// Check query parameters
		query := r.URL.Query()
		assert.Equal(t, "10", query.Get("limit"))
		assert.Equal(t, "conn-123", query.Get("connector_id"))

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":           "grant-1",
					"provider":     "google",
					"email":        "user@example.com",
					"grant_status": "valid",
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
	params := &domain.GrantsQueryParams{
		Limit:       10,
		ConnectorID: "conn-123",
	}
	grants, err := client.ListAllGrants(ctx, params)

	require.NoError(t, err)
	assert.Len(t, grants, 1)
	assert.Equal(t, "google", string(grants[0].Provider))
}

func TestHTTPClient_GetGrantStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":           "grant-1",
					"provider":     "google",
					"grant_status": "valid",
				},
				{
					"id":           "grant-2",
					"provider":     "microsoft",
					"grant_status": "valid",
				},
				{
					"id":           "grant-3",
					"provider":     "google",
					"grant_status": "invalid",
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
	stats, err := client.GetGrantStats(ctx)

	require.NoError(t, err)
	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 2, stats.ByProvider["google"])
	assert.Equal(t, 1, stats.ByProvider["microsoft"])
	assert.Equal(t, 2, stats.Valid)
	assert.Equal(t, 1, stats.Invalid)
}

// Mock Client Tests

func TestMockClient_AdminOperations(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	// Application tests
	apps, err := mock.ListApplications(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, apps)

	app, err := mock.GetApplication(ctx, "app-123")
	require.NoError(t, err)
	assert.Equal(t, "app-123", app.ID)

	createAppReq := &domain.CreateApplicationRequest{Name: "Test App"}
	createdApp, err := mock.CreateApplication(ctx, createAppReq)
	require.NoError(t, err)
	assert.NotEmpty(t, createdApp.ID)

	appName := "Updated App"
	updateAppReq := &domain.UpdateApplicationRequest{Name: &appName}
	updatedApp, err := mock.UpdateApplication(ctx, "app-456", updateAppReq)
	require.NoError(t, err)
	assert.Equal(t, "app-456", updatedApp.ID)

	err = mock.DeleteApplication(ctx, "app-789")
	require.NoError(t, err)

	// Connector tests
	connectors, err := mock.ListConnectors(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, connectors)

	connector, err := mock.GetConnector(ctx, "conn-123")
	require.NoError(t, err)
	assert.Equal(t, "conn-123", connector.ID)

	createConnReq := &domain.CreateConnectorRequest{Name: "Test Connector", Provider: "google"}
	createdConn, err := mock.CreateConnector(ctx, createConnReq)
	require.NoError(t, err)
	assert.NotEmpty(t, createdConn.ID)

	connName := "Updated Connector"
	updateConnReq := &domain.UpdateConnectorRequest{Name: &connName}
	updatedConn, err := mock.UpdateConnector(ctx, "conn-456", updateConnReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Connector", updatedConn.Name)

	err = mock.DeleteConnector(ctx, "conn-789")
	require.NoError(t, err)

	// Credential tests
	credentials, err := mock.ListCredentials(ctx, "conn-123")
	require.NoError(t, err)
	assert.NotEmpty(t, credentials)

	credential, err := mock.GetCredential(ctx, "cred-456")
	require.NoError(t, err)
	assert.Equal(t, "cred-456", credential.ID)

	createCredReq := &domain.CreateCredentialRequest{Name: "Test Cred", CredentialType: "oauth"}
	createdCred, err := mock.CreateCredential(ctx, "conn-123", createCredReq)
	require.NoError(t, err)
	assert.NotEmpty(t, createdCred.ID)

	credName := "Updated Cred"
	updateCredReq := &domain.UpdateCredentialRequest{Name: &credName}
	updatedCred, err := mock.UpdateCredential(ctx, "cred-789", updateCredReq)
	require.NoError(t, err)
	assert.Equal(t, "Updated Cred", updatedCred.Name)

	err = mock.DeleteCredential(ctx, "cred-delete")
	require.NoError(t, err)

	// Grant tests
	grants, err := mock.ListAllGrants(ctx, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, grants)

	stats, err := mock.GetGrantStats(ctx)
	require.NoError(t, err)
	assert.Greater(t, stats.Total, 0)
}

// Demo Client Tests

func TestDemoClient_AdminOperations(t *testing.T) {
	ctx := context.Background()
	demo := nylas.NewDemoClient()

	// Application tests
	apps, err := demo.ListApplications(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, apps)

	app, err := demo.GetApplication(ctx, "demo-app")
	require.NoError(t, err)
	assert.NotEmpty(t, app.ID)

	createAppReq := &domain.CreateApplicationRequest{Name: "Demo App"}
	createdApp, err := demo.CreateApplication(ctx, createAppReq)
	require.NoError(t, err)
	assert.NotEmpty(t, createdApp.ID)

	// Connector tests
	connectors, err := demo.ListConnectors(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, connectors)

	connector, err := demo.GetConnector(ctx, "demo-conn")
	require.NoError(t, err)
	assert.NotEmpty(t, connector.ID)

	// Credential tests
	credentials, err := demo.ListCredentials(ctx, "demo-conn")
	require.NoError(t, err)
	assert.NotEmpty(t, credentials)

	// Grant tests
	grants, err := demo.ListAllGrants(ctx, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, grants)

	stats, err := demo.GetGrantStats(ctx)
	require.NoError(t, err)
	assert.Greater(t, stats.Total, 0)
	assert.NotEmpty(t, stats.ByProvider)
}

// Error Handling Tests
