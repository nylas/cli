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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPClient_ListRoomResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Pin the contract: room resources are a grant-scoped GET on /resources.
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v3/grants/grant-123/resources", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-1",
			"data": []map[string]any{
				{
					"object":       "room_resource",
					"email":        "boardroom@example.com",
					"name":         "Boardroom",
					"capacity":     20,
					"building":     "HQ",
					"floor_name":   "5",
					"floor_number": 5,
				},
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	resources, err := client.ListRoomResources(context.Background(), "grant-123")

	require.NoError(t, err)
	require.Len(t, resources, 1)
	// The email is what callers reuse as a calendar ID, so it must survive intact.
	assert.Equal(t, "boardroom@example.com", resources[0].Email)
	assert.Equal(t, "Boardroom", resources[0].Name)
	assert.Equal(t, 20, resources[0].Capacity)
	assert.Equal(t, "HQ", resources[0].Building)
	assert.Equal(t, "5", resources[0].FloorName)
	assert.Equal(t, 5, resources[0].FloorNumber)
}

func TestHTTPClient_ListRoomResources_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	resources, err := client.ListRoomResources(context.Background(), "grant-123")

	require.NoError(t, err)
	assert.Empty(t, resources)
}

func TestHTTPClient_ListRoomResources_RequiresGrant(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")

	_, err := client.ListRoomResources(context.Background(), "")
	assert.Error(t, err)
}
