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

// Application Tests

func TestHTTPClient_ListApplications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":              "app-1",
					"application_id":  "app-id-1",
					"organization_id": "org-1",
					"region":          "us",
				},
				{
					"id":              "app-2",
					"application_id":  "app-id-2",
					"organization_id": "org-2",
					"region":          "eu",
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
	apps, err := client.ListApplications(ctx)

	require.NoError(t, err)
	assert.Len(t, apps, 2)
	assert.Equal(t, "app-1", apps[0].ID)
	assert.Equal(t, "us", apps[0].Region)
}

func TestHTTPClient_GetApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications/app-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":              "app-123",
				"application_id":  "app-id-123",
				"organization_id": "org-456",
				"region":          "us",
				"environment":     "production",
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
	app, err := client.GetApplication(ctx, "app-123")

	require.NoError(t, err)
	assert.Equal(t, "app-123", app.ID)
	assert.Equal(t, "app-id-123", app.ApplicationID)
	assert.Equal(t, "production", app.Environment)
}

func TestHTTPClient_CreateApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Application", body["name"])
		assert.Equal(t, "us", body["region"])

		response := map[string]any{
			"data": map[string]any{
				"id":     "app-new",
				"name":   "New Application",
				"region": "us",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(response) // Test helper, encode error not actionable
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.CreateApplicationRequest{
		Name:   "New Application",
		Region: "us",
	}
	app, err := client.CreateApplication(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "app-new", app.ID)
	assert.Equal(t, "us", app.Region)
}

func TestHTTPClient_UpdateApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications/app-456", r.URL.Path)
		assert.Equal(t, "PATCH", r.Method)

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Application", body["name"])

		response := map[string]any{
			"data": map[string]any{
				"id":   "app-456",
				"name": "Updated Application",
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
	name := "Updated Application"
	req := &domain.UpdateApplicationRequest{
		Name: &name,
	}
	app, err := client.UpdateApplication(ctx, "app-456", req)

	require.NoError(t, err)
	assert.Equal(t, "app-456", app.ID)
}

func TestHTTPClient_DeleteApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/applications/app-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteApplication(ctx, "app-delete")

	require.NoError(t, err)
}

// Connector Tests
