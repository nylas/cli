//go:build !integration
// +build !integration

package nylas

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseError_ParsesTopLevelMessage(t *testing.T) {
	client := NewHTTPClient()

	resp := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body: io.NopCloser(strings.NewReader(`{
			"type": "invalid_request_error",
			"message": "extra fields not permitted: app_password"
		}`)),
	}

	err := client.parseError(resp)
	require.Error(t, err)

	var apiErr *domain.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	assert.Equal(t, "invalid_request_error", apiErr.Type)
	assert.Equal(t, "extra fields not permitted: app_password", apiErr.Message)
}

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
				"settings": map[string]any{
					"policy_id": "policy-123",
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

	account, err := client.CreateAgentAccount(context.Background(), "agent@example.com", "ValidAgentPass123ABC!", "policy-123")
	require.NoError(t, err)
	assert.Equal(t, "agent-new", account.ID)
	assert.Equal(t, "agent@example.com", account.Email)
	assert.Equal(t, "policy-123", account.Settings.PolicyID)
}

func TestUpdateAgentAccount(t *testing.T) {
	var getCalls int32
	var patchCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/agent-123", r.URL.Path)

		switch r.Method {
		case http.MethodGet:
			atomic.AddInt32(&getCalls, 1)
			response := map[string]any{
				"data": map[string]any{
					"id":           "agent-123",
					"email":        "agent@example.com",
					"provider":     "nylas",
					"grant_status": "valid",
					"created_at":   time.Now().Unix(),
					"settings": map[string]any{
						"policy_id": "policy-123",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case http.MethodPatch:
			atomic.AddInt32(&patchCalls, 1)

			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))

			settings, ok := payload["settings"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, "agent@example.com", settings["email"])
			assert.Equal(t, "ValidAgentPass123ABC!", settings["app_password"])
			assert.Equal(t, "policy-123", settings["policy_id"])

			response := map[string]any{
				"data": map[string]any{
					"id":           "agent-123",
					"email":        "agent@example.com",
					"provider":     "nylas",
					"grant_status": "valid",
					"created_at":   time.Now().Unix(),
					"settings": map[string]any{
						"policy_id": "policy-123",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	account, err := client.UpdateAgentAccount(context.Background(), "agent-123", "agent@example.com", "ValidAgentPass123ABC!")
	require.NoError(t, err)
	assert.Equal(t, "agent-123", account.ID)
	assert.Equal(t, "agent@example.com", account.Email)
	assert.Equal(t, "policy-123", account.Settings.PolicyID)
	assert.EqualValues(t, 1, atomic.LoadInt32(&getCalls))
	assert.EqualValues(t, 1, atomic.LoadInt32(&patchCalls))
}

func TestUpdateAgentAccount_PreservesEmptyPolicyID(t *testing.T) {
	var patchCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/agent-123", r.URL.Path)

		switch r.Method {
		case http.MethodGet:
			response := map[string]any{
				"data": map[string]any{
					"id":           "agent-123",
					"email":        "agent@example.com",
					"provider":     "nylas",
					"grant_status": "valid",
					"created_at":   time.Now().Unix(),
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		case http.MethodPatch:
			atomic.AddInt32(&patchCalls, 1)

			var payload map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))

			settings, ok := payload["settings"].(map[string]any)
			require.True(t, ok)
			_, hasPolicyID := settings["policy_id"]
			assert.False(t, hasPolicyID)

			response := map[string]any{
				"data": map[string]any{
					"id":           "agent-123",
					"email":        "agent@example.com",
					"provider":     "nylas",
					"grant_status": "valid",
					"created_at":   time.Now().Unix(),
				},
			}

			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	account, err := client.UpdateAgentAccount(context.Background(), "agent-123", "agent@example.com", "ValidAgentPass123ABC!")
	require.NoError(t, err)
	assert.Equal(t, "agent-123", account.ID)
	assert.EqualValues(t, 1, atomic.LoadInt32(&patchCalls))
}

func TestCreateAgentAccount_RejectsNonNylasResponse(t *testing.T) {
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

	_, err := client.CreateAgentAccount(context.Background(), "agent@example.com", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-nylas managed grant")
}

func TestUpdateAgentAccount_RejectsNonNylasResponse(t *testing.T) {
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

	_, err := client.UpdateAgentAccount(context.Background(), "grant-001", "agent@example.com", "ValidAgentPass123ABC!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grant is not a nylas agent account")
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

func TestCreateAgentAccount_DoesNotInventPolicyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	account, err := client.CreateAgentAccount(context.Background(), "agent@example.com", "", "policy-123")
	require.NoError(t, err)
	assert.Equal(t, "", account.Settings.PolicyID)
}

func TestUpdateAgentAccount_RejectsNonNylasGrantBeforePatch(t *testing.T) {
	var patchCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/grants/grant-001", r.URL.Path)

		switch r.Method {
		case http.MethodGet:
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
		case http.MethodPatch:
			atomic.AddInt32(&patchCalls, 1)
			t.Fatalf("unexpected PATCH for non-nylas grant")
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	_, err := client.UpdateAgentAccount(context.Background(), "grant-001", "agent@example.com", "ValidAgentPass123ABC!")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "grant is not a nylas agent account")
	assert.EqualValues(t, 0, atomic.LoadInt32(&patchCalls))
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
