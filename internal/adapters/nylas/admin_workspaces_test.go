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

// The manual-assign endpoint requires the exact field names assign_grants /
// remove_grants — the public docs describe them loosely as assign/remove, and
// the API rejects those with "At least one grant must be assigned or removed".
func TestHTTPClient_AssignWorkspaceGrants(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/workspaces/ws-1/manual-assign", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, []any{"grant-1"}, body["assign_grants"])
		_, hasRemove := body["remove_grants"]
		assert.False(t, hasRemove, "remove_grants must be omitted when empty")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"workspace_id":    "ws-1",
				"grants_assigned": []string{"grant-1"},
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	result, err := client.AssignWorkspaceGrants(context.Background(), "ws-1", &domain.WorkspaceAssignRequest{
		AssignGrants: []string{"grant-1"},
	})

	require.NoError(t, err)
	assert.Equal(t, "ws-1", result.WorkspaceID)
	assert.Equal(t, []string{"grant-1"}, result.GrantsAssigned)
	assert.Empty(t, result.GrantsRemoved)
}

func TestHTTPClient_AssignWorkspaceGrants_Remove(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, []any{"grant-2"}, body["remove_grants"])

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"workspace_id":   "ws-1",
				"grants_removed": []string{"grant-2"},
			},
		})
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	client.SetBaseURL(server.URL)

	result, err := client.AssignWorkspaceGrants(context.Background(), "ws-1", &domain.WorkspaceAssignRequest{
		RemoveGrants: []string{"grant-2"},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"grant-2"}, result.GrantsRemoved)
}

func TestHTTPClient_AssignWorkspaceGrants_Validation(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("client-id", "secret", "api-key")
	ctx := context.Background()

	tests := []struct {
		name        string
		workspaceID string
		req         *domain.WorkspaceAssignRequest
	}{
		{"missing workspace ID", "", &domain.WorkspaceAssignRequest{AssignGrants: []string{"g"}}},
		{"nil request", "ws-1", nil},
		{"no grants in either list", "ws-1", &domain.WorkspaceAssignRequest{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.AssignWorkspaceGrants(ctx, tt.workspaceID, tt.req)
			assert.Error(t, err)
		})
	}
}
