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

func TestCreateCustomGrant_ICloud(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connect/custom", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "icloud", payload["provider"])

		settings, ok := payload["settings"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "user@icloud.com", settings["username"])
		assert.Equal(t, "abcd-efgh-ijkl-mnop", settings["password"])

		response := map[string]any{
			"data": map[string]any{
				"id":           "icloud-grant-001",
				"email":        "user@icloud.com",
				"provider":     "icloud",
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

	grant, err := client.CreateCustomGrant(context.Background(), "icloud", map[string]any{
		"username": "user@icloud.com",
		"password": "abcd-efgh-ijkl-mnop",
	})
	require.NoError(t, err)
	assert.Equal(t, "icloud-grant-001", grant.ID)
	assert.Equal(t, "user@icloud.com", grant.Email)
	assert.Equal(t, "icloud", string(grant.Provider))
}

func TestCreateCustomGrant_Yahoo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v3/connect/custom", r.URL.Path)

		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "imap", payload["provider"])

		settings, ok := payload["settings"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "user@yahoo.com", settings["imap_username"])
		assert.Equal(t, "imap.mail.yahoo.com", settings["imap_host"])
		assert.Equal(t, float64(993), settings["imap_port"])
		assert.Equal(t, "yahoo", settings["type"])

		response := map[string]any{
			"data": map[string]any{
				"id":           "yahoo-grant-001",
				"email":        "user@yahoo.com",
				"provider":     "imap",
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

	grant, err := client.CreateCustomGrant(context.Background(), "imap", map[string]any{
		"imap_username": "user@yahoo.com",
		"imap_password": "app-password-123",
		"imap_host":     "imap.mail.yahoo.com",
		"imap_port":     993,
		"type":          "yahoo",
	})
	require.NoError(t, err)
	assert.Equal(t, "yahoo-grant-001", grant.ID)
	assert.Equal(t, "user@yahoo.com", grant.Email)
}

func TestCreateCustomGrant_IMAP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "imap", payload["provider"])

		settings, ok := payload["settings"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "user@company.com", settings["imap_username"])
		assert.Equal(t, "mail.company.com", settings["imap_host"])
		assert.Equal(t, float64(993), settings["imap_port"])
		assert.Equal(t, "smtp.company.com", settings["smtp_host"])
		assert.Equal(t, float64(465), settings["smtp_port"])

		response := map[string]any{
			"data": map[string]any{
				"id":           "imap-grant-001",
				"email":        "user@company.com",
				"provider":     "imap",
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

	grant, err := client.CreateCustomGrant(context.Background(), "imap", map[string]any{
		"imap_username": "user@company.com",
		"imap_password": "secret",
		"imap_host":     "mail.company.com",
		"imap_port":     993,
		"smtp_host":     "smtp.company.com",
		"smtp_port":     465,
	})
	require.NoError(t, err)
	assert.Equal(t, "imap-grant-001", grant.ID)
	assert.Equal(t, "user@company.com", grant.Email)
	assert.Equal(t, "imap", string(grant.Provider))
}

func TestCreateCustomGrant_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "Invalid IMAP credentials",
			},
		})
	}))
	defer server.Close()

	client := NewHTTPClient()
	client.baseURL = server.URL
	client.SetCredentials("", "", "test-api-key")

	_, err := client.CreateCustomGrant(context.Background(), "imap", map[string]any{
		"imap_username": "bad@example.com",
		"imap_password": "wrong",
		"imap_host":     "mail.example.com",
		"imap_port":     993,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid IMAP credentials")
}
