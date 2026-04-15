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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListAgentAccounts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "nylas", r.URL.Query().Get("provider"))

		response := map[string]any{
			"data": []map[string]any{
				{
					"id":            "agent-001",
					"email":         "agent@example.com",
					"provider":      "nylas",
					"grant_status":  "valid",
					"credential_id": "cred-123",
					"settings": map[string]any{
						"policy_id": "policy-123",
					},
					"created_at": time.Now().Unix(),
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	accounts, err := client.ListAgentAccounts(context.Background())
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	assert.Equal(t, "agent-001", accounts[0].ID)
	assert.Equal(t, "agent@example.com", accounts[0].Email)
	assert.Equal(t, "policy-123", accounts[0].Settings.PolicyID)
}

func TestListAgentAccounts_PaginatesAllResults(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "nylas", r.URL.Query().Get("provider"))

		requests++
		var response map[string]any
		if r.URL.Query().Get("page_token") == "" {
			response = map[string]any{
				"data": []map[string]any{
					{
						"id":           "agent-001",
						"email":        "first@example.com",
						"provider":     "nylas",
						"grant_status": "valid",
					},
				},
				"next_cursor": "cursor-2",
			}
		} else {
			assert.Equal(t, "cursor-2", r.URL.Query().Get("page_token"))
			response = map[string]any{
				"data": []map[string]any{
					{
						"id":           "agent-002",
						"email":        "second@example.com",
						"provider":     "nylas",
						"grant_status": "valid",
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

	accounts, err := client.ListAgentAccounts(context.Background())
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	assert.Equal(t, 2, requests)
	assert.Equal(t, "agent-001", accounts[0].ID)
	assert.Equal(t, "agent-002", accounts[1].ID)
}

func TestCreateAgentAccount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connect/custom", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "nylas", payload["provider"])

		settings, ok := payload["settings"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "agent@example.com", settings["email"])
		assert.Equal(t, "ValidAgentPass123ABC!", settings["app_password"])
		assert.Equal(t, "policy-123", settings["policy_id"])

		response := map[string]any{
			"data": map[string]any{
				"id":           "agent-new",
				"email":        "agent@example.com",
				"provider":     "nylas",
				"grant_status": "valid",
				"created_at":   time.Now().Unix(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	account, err := client.CreateAgentAccount(context.Background(), "agent@example.com", "ValidAgentPass123ABC!", "policy-123")
	require.NoError(t, err)
	assert.Equal(t, "agent-new", account.ID)
	assert.Equal(t, "agent@example.com", account.Email)
	assert.Equal(t, "policy-123", account.Settings.PolicyID)
}

func TestCreateAgentAccount_DirectResponseFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":           "agent-direct",
			"email":        "agent@example.com",
			"provider":     "nylas",
			"grant_status": "valid",
			"created_at":   time.Now().Unix(),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	account, err := client.CreateAgentAccount(context.Background(), "agent@example.com", "", "")
	require.NoError(t, err)
	assert.Equal(t, "agent-direct", account.ID)
	assert.Equal(t, "agent@example.com", account.Email)
}

func TestGetAgentAccountRejectsNonNylasProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"data": map[string]any{
				"id":           "grant-001",
				"email":        "user@gmail.com",
				"provider":     "google",
				"grant_status": "valid",
				"created_at":   time.Now().Unix(),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	_, err := client.GetAgentAccount(context.Background(), "grant-001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a nylas agent account")
}
