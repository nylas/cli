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

// Scheduler Configuration Tests

func TestHTTPClient_ListSchedulerConfigurations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/configurations", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "config-1",
					"name": "30 Minute Meeting",
					"slug": "30min",
				},
				{
					"id":   "config-2",
					"name": "1 Hour Meeting",
					"slug": "1hour",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	configs, err := client.ListSchedulerConfigurations(ctx)

	require.NoError(t, err)
	assert.Len(t, configs, 2)
	assert.Equal(t, "config-1", configs[0].ID)
	assert.Equal(t, "30 Minute Meeting", configs[0].Name)
	assert.Equal(t, "30min", configs[0].Slug)
}

func TestHTTPClient_GetSchedulerConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/configurations/config-123", r.URL.Path)
		assert.Equal(t, "GET", r.Method)

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "config-123",
				"name": "Interview Meeting",
				"slug": "interview",
				"participants": []map[string]interface{}{
					{
						"email":        "interviewer@example.com",
						"name":         "Interviewer",
						"is_organizer": true,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	config, err := client.GetSchedulerConfiguration(ctx, "config-123")

	require.NoError(t, err)
	assert.Equal(t, "config-123", config.ID)
	assert.Equal(t, "Interview Meeting", config.Name)
	assert.Len(t, config.Participants, 1)
	assert.Equal(t, "interviewer@example.com", config.Participants[0].Email)
}

func TestHTTPClient_CreateSchedulerConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/configurations", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New Meeting Type", body["name"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "config-new",
				"name": "New Meeting Type",
				"slug": "new-meeting",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.CreateSchedulerConfigurationRequest{
		Name: "New Meeting Type",
	}
	config, err := client.CreateSchedulerConfiguration(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "config-new", config.ID)
	assert.Equal(t, "New Meeting Type", config.Name)
}

func TestHTTPClient_UpdateSchedulerConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/configurations/config-456", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "Updated Meeting", body["name"])

		response := map[string]interface{}{
			"data": map[string]interface{}{
				"id":   "config-456",
				"name": "Updated Meeting",
				"slug": "updated",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode( // Test helper, encode error not actionable
			response)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	req := &domain.UpdateSchedulerConfigurationRequest{
		Name: strPtr("Updated Meeting"),
	}
	config, err := client.UpdateSchedulerConfiguration(ctx, "config-456", req)

	require.NoError(t, err)
	assert.Equal(t, "config-456", config.ID)
	assert.Equal(t, "Updated Meeting", config.Name)
}

func TestHTTPClient_DeleteSchedulerConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/scheduling/configurations/config-delete", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	ctx := context.Background()
	err := client.DeleteSchedulerConfiguration(ctx, "config-delete")

	require.NoError(t, err)
}

// Scheduler Session Tests
