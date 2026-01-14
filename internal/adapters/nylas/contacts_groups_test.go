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

func TestHTTPClient_GetContactGroups(t *testing.T) {
	t.Run("returns contact groups", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/v3/grants/grant-123/contacts/groups", r.URL.Path)

			response := map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":       "group-1",
						"grant_id": "grant-123",
						"name":     "Work",
						"path":     "/Work",
						"object":   "contact_group",
					},
					{
						"id":       "group-2",
						"grant_id": "grant-123",
						"name":     "Family",
						"path":     "/Family",
						"object":   "contact_group",
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
		groups, err := client.GetContactGroups(ctx, "grant-123")

		require.NoError(t, err)
		assert.Len(t, groups, 2)
		assert.Equal(t, "group-1", groups[0].ID)
		assert.Equal(t, "Work", groups[0].Name)
		assert.Equal(t, "/Work", groups[0].Path)
	})
}

func TestHTTPClient_GetContactGroup(t *testing.T) {
	tests := []struct {
		name           string
		groupID        string
		serverResponse map[string]interface{}
		statusCode     int
		wantErr        bool
		errContains    string
	}{
		{
			name:    "returns group",
			groupID: "group-123",
			serverResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"id":       "group-123",
					"grant_id": "grant-123",
					"name":     "Friends",
					"path":     "/Friends",
					"object":   "contact_group",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:    "returns error for not found",
			groupID: "nonexistent",
			serverResponse: map[string]interface{}{
				"error": map[string]string{"message": "contact group not found"},
			},
			statusCode:  http.StatusNotFound,
			wantErr:     true,
			errContains: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				expectedPath := "/v3/grants/grant-123/contacts/groups/" + tt.groupID
				assert.Equal(t, expectedPath, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			group, err := client.GetContactGroup(ctx, "grant-123", tt.groupID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.groupID, group.ID)
		})
	}
}

func TestHTTPClient_CreateContactGroup(t *testing.T) {
	t.Run("creates group", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/v3/grants/grant-123/contacts/groups", r.URL.Path)

			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "Colleagues", body["name"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":       "new-group-123",
					"grant_id": "grant-123",
					"name":     "Colleagues",
					"path":     "/Colleagues",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		client := nylas.NewHTTPClient()
		client.SetCredentials("client-id", "secret", "api-key")
		client.SetBaseURL(server.URL)

		ctx := context.Background()
		req := &domain.CreateContactGroupRequest{Name: "Colleagues"}
		group, err := client.CreateContactGroup(ctx, "grant-123", req)

		require.NoError(t, err)
		assert.Equal(t, "Colleagues", group.Name)
	})
}

func TestHTTPClient_UpdateContactGroup(t *testing.T) {
	t.Run("updates group", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "PUT", r.Method)
			assert.Equal(t, "/v3/grants/grant-123/contacts/groups/group-456", r.URL.Path)

			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "Updated Name", body["name"])

			response := map[string]interface{}{
				"data": map[string]interface{}{
					"id":       "group-456",
					"grant_id": "grant-123",
					"name":     "Updated Name",
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
		updatedName := "Updated Name"
		req := &domain.UpdateContactGroupRequest{Name: &updatedName}
		group, err := client.UpdateContactGroup(ctx, "grant-123", "group-456", req)

		require.NoError(t, err)
		assert.Equal(t, "Updated Name", group.Name)
	})
}

func TestHTTPClient_DeleteContactGroup(t *testing.T) {
	tests := []struct {
		name       string
		groupID    string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "deletes with 200",
			groupID:    "group-123",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "deletes with 204",
			groupID:    "group-456",
			statusCode: http.StatusNoContent,
			wantErr:    false,
		},
		{
			name:       "returns error for not found",
			groupID:    "nonexistent",
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				expectedPath := "/v3/grants/grant-123/contacts/groups/" + tt.groupID
				assert.Equal(t, expectedPath, r.URL.Path)

				w.WriteHeader(tt.statusCode)
				if tt.statusCode >= 400 {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]string{"message": "not found"},
					})
				}
			}))
			defer server.Close()

			client := nylas.NewHTTPClient()
			client.SetCredentials("client-id", "secret", "api-key")
			client.SetBaseURL(server.URL)

			ctx := context.Background()
			err := client.DeleteContactGroup(ctx, "grant-123", tt.groupID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
