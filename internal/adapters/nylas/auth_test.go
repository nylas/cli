//go:build !integration

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

// newTestClient creates a new HTTPClient configured for testing.
func newTestClient(apiKey, clientID, clientSecret string) *HTTPClient {
	client := NewHTTPClient()
	client.SetCredentials(clientID, clientSecret, apiKey)
	return client
}

func TestHTTPClient_BuildAuthURL(t *testing.T) {
	client := newTestClient("test-api-key", "test-client-id", "test-client-secret")

	tests := []struct {
		name        string
		provider    domain.Provider
		redirectURI string
		wantInURL   []string
	}{
		{
			name:        "google provider",
			provider:    domain.ProviderGoogle,
			redirectURI: "http://localhost:8080/callback",
			wantInURL: []string{
				"client_id=test-client-id",
				"redirect_uri=http",
				"response_type=code",
				"provider=google",
				"access_type=offline",
			},
		},
		{
			name:        "microsoft provider",
			provider:    domain.ProviderMicrosoft,
			redirectURI: "https://example.com/auth/callback",
			wantInURL: []string{
				"client_id=test-client-id",
				"provider=microsoft",
			},
		},
		{
			name:        "imap provider",
			provider:    domain.ProviderIMAP,
			redirectURI: "http://localhost:3000/oauth",
			wantInURL: []string{
				"provider=imap",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := client.BuildAuthURL(tt.provider, tt.redirectURI)

			for _, want := range tt.wantInURL {
				assert.Contains(t, url, want)
			}
			assert.Contains(t, url, "/v3/connect/auth?")
		})
	}
}

func TestHTTPClient_ExchangeCode(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse interface{}
		serverStatus   int
		wantErr        bool
		wantGrant      *domain.Grant
	}{
		{
			name: "successful exchange",
			serverResponse: map[string]interface{}{
				"grant_id":      "grant-123",
				"access_token":  "access-token-abc",
				"refresh_token": "refresh-token-xyz",
				"email":         "user@example.com",
				"provider":      "google",
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
			wantGrant: &domain.Grant{
				ID:           "grant-123",
				Email:        "user@example.com",
				Provider:     domain.ProviderGoogle,
				AccessToken:  "access-token-abc",
				RefreshToken: "refresh-token-xyz",
				GrantStatus:  "valid",
			},
		},
		{
			name: "invalid code",
			serverResponse: map[string]interface{}{
				"error":       "invalid_grant",
				"description": "The authorization code has expired",
			},
			serverStatus: http.StatusBadRequest,
			wantErr:      true,
			wantGrant:    nil,
		},
		{
			name:           "server error",
			serverResponse: map[string]interface{}{"error": "internal_error"},
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
			wantGrant:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/v3/connect/token", r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := newTestClient("test-api-key", "test-client-id", "test-client-secret")
			client.SetBaseURL(server.URL)

			grant, err := client.ExchangeCode(context.Background(), "test-code", "http://localhost/callback")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, grant)
			} else {
				require.NoError(t, err)
				require.NotNil(t, grant)
				assert.Equal(t, tt.wantGrant.ID, grant.ID)
				assert.Equal(t, tt.wantGrant.Email, grant.Email)
				assert.Equal(t, tt.wantGrant.Provider, grant.Provider)
				assert.Equal(t, tt.wantGrant.AccessToken, grant.AccessToken)
				assert.Equal(t, tt.wantGrant.RefreshToken, grant.RefreshToken)
			}
		})
	}
}

func TestHTTPClient_ExchangeCode_UsesAPIKeyAsSecret(t *testing.T) {
	var receivedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"grant_id": "grant-123",
			"email":    "test@example.com",
			"provider": "google",
		})
	}))
	defer server.Close()

	// Test with no client_secret - should use API key
	client := newTestClient("my-api-key", "my-client-id", "")
	client.SetBaseURL(server.URL)

	_, err := client.ExchangeCode(context.Background(), "test-code", "http://localhost/callback")
	require.NoError(t, err)

	assert.Equal(t, "my-api-key", receivedBody["client_secret"])
	assert.Equal(t, "my-client-id", receivedBody["client_id"])
	assert.Equal(t, "test-code", receivedBody["code"])
}

func TestHTTPClient_ListGrants(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse interface{}
		serverStatus   int
		wantErr        bool
		wantCount      int
	}{
		{
			name: "list multiple grants",
			serverResponse: map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"id":           "grant-1",
						"email":        "user1@example.com",
						"provider":     "google",
						"grant_status": "valid",
					},
					{
						"id":           "grant-2",
						"email":        "user2@outlook.com",
						"provider":     "microsoft",
						"grant_status": "valid",
					},
				},
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
			wantCount:    2,
		},
		{
			name: "empty grants list",
			serverResponse: map[string]interface{}{
				"data": []map[string]interface{}{},
			},
			serverStatus: http.StatusOK,
			wantErr:      false,
			wantCount:    0,
		},
		{
			name:           "unauthorized",
			serverResponse: map[string]interface{}{"error": "unauthorized"},
			serverStatus:   http.StatusUnauthorized,
			wantErr:        true,
			wantCount:      0,
		},
		{
			name:           "server error",
			serverResponse: map[string]interface{}{"error": "internal_error"},
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
			wantCount:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/v3/grants", r.URL.Path)
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := newTestClient("test-api-key", "test-client-id", "")
			client.SetBaseURL(server.URL)

			grants, err := client.ListGrants(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, grants, tt.wantCount)
			}
		})
	}
}

func TestHTTPClient_GetGrant(t *testing.T) {
	tests := []struct {
		name           string
		grantID        string
		serverResponse interface{}
		serverStatus   int
		wantErr        error
		wantGrant      *domain.Grant
	}{
		{
			name:    "get existing grant",
			grantID: "grant-123",
			serverResponse: map[string]interface{}{
				"data": map[string]interface{}{
					"id":           "grant-123",
					"email":        "user@example.com",
					"provider":     "google",
					"grant_status": "valid",
				},
			},
			serverStatus: http.StatusOK,
			wantErr:      nil,
			wantGrant: &domain.Grant{
				ID:          "grant-123",
				Email:       "user@example.com",
				Provider:    domain.ProviderGoogle,
				GrantStatus: "valid",
			},
		},
		{
			name:           "grant not found",
			grantID:        "nonexistent",
			serverResponse: map[string]interface{}{"error": "not_found"},
			serverStatus:   http.StatusNotFound,
			wantErr:        domain.ErrGrantNotFound,
			wantGrant:      nil,
		},
		{
			name:           "unauthorized",
			grantID:        "grant-123",
			serverResponse: map[string]interface{}{"error": "unauthorized"},
			serverStatus:   http.StatusUnauthorized,
			wantErr:        domain.ErrAPIError,
			wantGrant:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/v3/grants/"+tt.grantID, r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			client := newTestClient("test-api-key", "test-client-id", "")
			client.SetBaseURL(server.URL)

			grant, err := client.GetGrant(context.Background(), tt.grantID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, grant)
			} else {
				require.NoError(t, err)
				require.NotNil(t, grant)
				assert.Equal(t, tt.wantGrant.ID, grant.ID)
				assert.Equal(t, tt.wantGrant.Email, grant.Email)
			}
		})
	}
}

func TestHTTPClient_RevokeGrant(t *testing.T) {
	tests := []struct {
		name         string
		grantID      string
		serverStatus int
		wantErr      error
	}{
		{
			name:         "successful revoke",
			grantID:      "grant-123",
			serverStatus: http.StatusOK,
			wantErr:      nil,
		},
		{
			name:         "successful revoke with no content",
			grantID:      "grant-456",
			serverStatus: http.StatusNoContent,
			wantErr:      nil,
		},
		{
			name:         "grant not found",
			grantID:      "nonexistent",
			serverStatus: http.StatusNotFound,
			wantErr:      domain.ErrGrantNotFound,
		},
		{
			name:         "unauthorized",
			grantID:      "grant-789",
			serverStatus: http.StatusUnauthorized,
			wantErr:      domain.ErrAPIError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "DELETE", r.Method)
				assert.Equal(t, "/v3/grants/"+tt.grantID, r.URL.Path)

				w.WriteHeader(tt.serverStatus)
			}))
			defer server.Close()

			client := newTestClient("test-api-key", "test-client-id", "")
			client.SetBaseURL(server.URL)

			err := client.RevokeGrant(context.Background(), tt.grantID)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHTTPClient_ListGrants_NetworkError(t *testing.T) {
	client := newTestClient("test-api-key", "test-client-id", "")
	client.SetBaseURL("http://localhost:1") // Invalid port

	_, err := client.ListGrants(context.Background())

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNetworkError)
}

func TestHTTPClient_GetGrant_NetworkError(t *testing.T) {
	client := newTestClient("test-api-key", "test-client-id", "")
	client.SetBaseURL("http://localhost:1") // Invalid port

	_, err := client.GetGrant(context.Background(), "test-grant")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNetworkError)
}

func TestHTTPClient_RevokeGrant_NetworkError(t *testing.T) {
	client := newTestClient("test-api-key", "test-client-id", "")
	client.SetBaseURL("http://localhost:1") // Invalid port

	err := client.RevokeGrant(context.Background(), "test-grant")

	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNetworkError)
}

func TestHTTPClient_ExchangeCode_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		<-r.Context().Done()
	}))
	defer server.Close()

	client := newTestClient("test-api-key", "test-client-id", "")
	client.SetBaseURL(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.ExchangeCode(ctx, "test-code", "http://localhost/callback")

	assert.Error(t, err)
}
