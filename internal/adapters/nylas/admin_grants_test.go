package nylas_test

import (
	"context"
	"encoding/json"
	"fmt"
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

		response := map[string]any{
			"data": []map[string]any{
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

		// limit is the API page size (max 200), not the caller's cap.
		query := r.URL.Query()
		assert.Equal(t, "200", query.Get("limit"))
		assert.Equal(t, "conn-123", query.Get("connector_id"))

		response := map[string]any{
			"data": []map[string]any{
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

func TestHTTPClient_ListAllGrants_FollowsPagination(t *testing.T) {
	// Regression: the Nylas v3 /v3/grants endpoint is offset-paginated
	// (limit, offset) and does NOT return next_cursor. ListAllGrants
	// previously made a single request and silently truncated to the API
	// default page size (10), so any tenant with >10 grants — including
	// the `nylas auth config` flow via SyncGrants → ListGrants — only
	// saw the first page.
	const apiPageSize = 200 // mirrors the unexported grantPageSize constant
	full := make([]map[string]any, 0, apiPageSize)
	for i := range apiPageSize {
		full = append(full, map[string]any{
			"id":           fmt.Sprintf("grant-page1-%d", i),
			"provider":     "google",
			"grant_status": "valid",
		})
	}

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		query := r.URL.Query()
		assert.Equal(t, "200", query.Get("limit"), "should request the API max page size")

		w.Header().Set("Content-Type", "application/json")
		switch query.Get("offset") {
		case "", "0":
			// First page is full — implementation must fetch another.
			_ = json.NewEncoder(w).Encode(map[string]any{"data": full})
		case "200":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "grant-tail-1", "provider": "microsoft", "grant_status": "valid"},
					{"id": "grant-tail-2", "provider": "microsoft", "grant_status": "valid"},
				},
			})
		default:
			t.Fatalf("unexpected offset %q", query.Get("offset"))
		}
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	grants, err := client.ListAllGrants(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "should have advanced offset and made a second request")
	assert.Len(t, grants, 202)
	assert.Equal(t, "grant-tail-2", grants[201].ID)
}

func TestHTTPClient_ListAllGrants_StopsOnShortPage(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "grant-1", "provider": "google", "grant_status": "valid"},
				{"id": "grant-2", "provider": "google", "grant_status": "valid"},
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	grants, err := client.ListAllGrants(context.Background(), nil)
	require.NoError(t, err)
	assert.Equal(t, 1, calls, "short first page must terminate pagination")
	assert.Len(t, grants, 2)
}

func TestHTTPClient_ListAllGrants_LimitCapsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "grant-1", "provider": "google", "grant_status": "valid"},
				{"id": "grant-2", "provider": "google", "grant_status": "valid"},
				{"id": "grant-3", "provider": "google", "grant_status": "valid"},
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	grants, err := client.ListAllGrants(context.Background(), &domain.GrantsQueryParams{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, grants, 2, "client-side limit should cap results")
}

func TestHTTPClient_GetGrantStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": []map[string]any{
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
