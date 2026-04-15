package nylas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRules(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/rules", r.URL.Path)
		_, ok := r.URL.Query()["page_token"]
		assert.False(t, ok)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"items": []map[string]any{
					{
						"id":      "rule-001",
						"name":    "Rule One",
						"enabled": true,
						"trigger": "inbound",
					},
					{
						"id":      "rule-002",
						"name":    "Rule Two",
						"enabled": false,
						"trigger": "inbound",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	rules, err := client.ListRules(context.Background())
	require.NoError(t, err)
	require.Len(t, rules, 2)
	assert.Equal(t, "rule-001", rules[0].ID)
	assert.Equal(t, "rule-002", rules[1].ID)
}

func TestGetRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/rules/rule/001", r.URL.Path)
		assert.Equal(t, "/v3/rules/rule%2F001", r.RequestURI)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":      "rule/001",
				"name":    "Rule One",
				"enabled": true,
				"trigger": "inbound",
			},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	rule, err := client.GetRule(context.Background(), "rule/001")
	require.NoError(t, err)
	assert.Equal(t, "rule/001", rule.ID)
	assert.Equal(t, "Rule One", rule.Name)
}

func TestCreateRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/rules", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "New Rule", payload["name"])

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id":      "rule-new",
				"name":    "New Rule",
				"enabled": true,
				"trigger": "inbound",
			},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	rule, err := client.CreateRule(context.Background(), map[string]any{"name": "New Rule"})
	require.NoError(t, err)
	assert.Equal(t, "rule-new", rule.ID)
	assert.Equal(t, "New Rule", rule.Name)
}

func TestUpdateRuleAssignsRequestedIDWhenResponseOmitsIt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/rules/rule/001", r.URL.Path)
		assert.Equal(t, "/v3/rules/rule%2F001", r.RequestURI)
		assert.Equal(t, http.MethodPut, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "Updated Rule", payload["name"])

		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"name":    "Updated Rule",
				"enabled": true,
				"trigger": "inbound",
			},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	rule, err := client.UpdateRule(context.Background(), "rule/001", map[string]any{"name": "Updated Rule"})
	require.NoError(t, err)
	assert.Equal(t, "rule/001", rule.ID)
	assert.Equal(t, "Updated Rule", rule.Name)
}

func TestDeleteRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/rules/rule/001", r.URL.Path)
		assert.Equal(t, "/v3/rules/rule%2F001", r.RequestURI)
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	require.NoError(t, client.DeleteRule(context.Background(), "rule/001"))
}

func TestGetRuleNotFound(t *testing.T) {
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

	_, err := client.GetRule(context.Background(), "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrRuleNotFound)
}
