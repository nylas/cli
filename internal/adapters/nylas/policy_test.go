//go:build !integration
// +build !integration

package nylas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListPolicies(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/policies", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		requests++
		var response map[string]any
		if requests == 1 {
			_, ok := r.URL.Query()["page_token"]
			assert.False(t, ok)
			response = map[string]any{
				"data": []map[string]any{
					{
						"id":              "policy-001",
						"name":            "Default Policy",
						"application_id":  "app-123",
						"organization_id": "org-123",
						"created_at":      time.Now().Unix(),
						"updated_at":      time.Now().Unix(),
					},
				},
				"next_cursor": "cursor-2",
			}
		} else {
			assert.Equal(t, "cursor-2", r.URL.Query().Get("page_token"))
			response = map[string]any{
				"data": []map[string]any{
					{
						"id":              "policy-002",
						"name":            "Strict Policy",
						"application_id":  "app-123",
						"organization_id": "org-123",
						"created_at":      time.Now().Unix(),
						"updated_at":      time.Now().Unix(),
					},
				},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	policies, err := client.ListPolicies(context.Background())
	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, 2, requests)
	assert.Equal(t, "policy-001", policies[0].ID)
	assert.Equal(t, "policy-002", policies[1].ID)
}

func TestGetPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/policies/policy/001", r.URL.Path)
		assert.Equal(t, "/v3/policies/policy%2F001", r.RequestURI)
		assert.Equal(t, http.MethodGet, r.Method)

		response := map[string]any{
			"data": map[string]any{
				"id":              "policy/001",
				"name":            "Default Policy",
				"application_id":  "app-123",
				"organization_id": "org-123",
				"created_at":      time.Now().Unix(),
				"updated_at":      time.Now().Unix(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	policy, err := client.GetPolicy(context.Background(), "policy/001")
	require.NoError(t, err)
	assert.Equal(t, "policy/001", policy.ID)
	assert.Equal(t, "Default Policy", policy.Name)
}

func TestGetPolicyNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type":    "api_error",
			"message": "not found",
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	_, err := client.GetPolicy(context.Background(), "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrPolicyNotFound)
}

func TestCreatePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/policies", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "New Policy", payload["name"])

		response := map[string]any{
			"data": map[string]any{
				"id":              "policy-new",
				"name":            "New Policy",
				"application_id":  "app-123",
				"organization_id": "org-123",
				"created_at":      time.Now().Unix(),
				"updated_at":      time.Now().Unix(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	policy, err := client.CreatePolicy(context.Background(), map[string]any{"name": "New Policy"})
	require.NoError(t, err)
	assert.Equal(t, "policy-new", policy.ID)
	assert.Equal(t, "New Policy", policy.Name)
}

func TestUpdatePolicyAssignsRequestedIDWhenResponseOmitsIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/policies/policy/001", r.URL.Path)
		assert.Equal(t, "/v3/policies/policy%2F001", r.RequestURI)
		assert.Equal(t, http.MethodPut, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "Updated Policy", payload["name"])

		response := map[string]any{
			"data": map[string]any{
				"name":            "Updated Policy",
				"application_id":  "app-123",
				"organization_id": "org-123",
				"updated_at":      time.Now().Unix(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	policy, err := client.UpdatePolicy(context.Background(), "policy/001", map[string]any{"name": "Updated Policy"})
	require.NoError(t, err)
	assert.Equal(t, "policy/001", policy.ID)
	assert.Equal(t, "Updated Policy", policy.Name)
}

func TestDeletePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/policies/policy/001", r.URL.Path)
		assert.Equal(t, "/v3/policies/policy%2F001", r.RequestURI)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	require.NoError(t, client.DeletePolicy(context.Background(), "policy/001"))
}
