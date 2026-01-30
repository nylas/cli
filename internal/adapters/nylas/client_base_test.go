package nylas_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/version"
	"github.com/stretchr/testify/assert"
)

// Test mock client implements interface

func TestMockClientImplementsInterface(t *testing.T) {
	var _ interface {
		SetRegion(region string)
		SetCredentials(clientID, clientSecret, apiKey string)
		BuildAuthURL(provider domain.Provider, redirectURI string) string
	} = nylas.NewMockClient()
}

func TestNewHTTPClient(t *testing.T) {
	client := nylas.NewHTTPClient()
	assert.NotNil(t, client)
}

func TestHTTPClient_SetRegion(t *testing.T) {
	client := nylas.NewHTTPClient()

	t.Run("sets US region by default", func(t *testing.T) {
		client.SetRegion("us")
		url := client.BuildAuthURL(domain.ProviderGoogle, "http://localhost")
		assert.Contains(t, url, "api.us.nylas.com")
	})

	t.Run("sets EU region", func(t *testing.T) {
		client.SetRegion("eu")
		url := client.BuildAuthURL(domain.ProviderGoogle, "http://localhost")
		assert.Contains(t, url, "api.eu.nylas.com")
	})
}

func TestHTTPClient_SetCredentials(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("my-client-id", "my-secret", "my-api-key")

	url := client.BuildAuthURL(domain.ProviderGoogle, "http://localhost")
	assert.Contains(t, url, "client_id=my-client-id")
}

func TestHTTPClient_BuildAuthURL(t *testing.T) {
	client := nylas.NewHTTPClient()
	client.SetCredentials("test-client-id", "", "")

	tests := []struct {
		name        string
		provider    domain.Provider
		redirectURI string
		wantContain []string
	}{
		{
			name:        "Google provider",
			provider:    domain.ProviderGoogle,
			redirectURI: "http://localhost:8080/callback",
			wantContain: []string{
				"provider=google",
				"redirect_uri=http",
				"client_id=test-client-id",
				"response_type=code",
			},
		},
		{
			name:        "Microsoft provider",
			provider:    domain.ProviderMicrosoft,
			redirectURI: "http://localhost:8080/callback",
			wantContain: []string{
				"provider=microsoft",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := client.BuildAuthURL(tt.provider, tt.redirectURI)
			for _, want := range tt.wantContain {
				assert.Contains(t, url, want)
			}
		})
	}
}

func TestHTTPClient_UserAgent(t *testing.T) {
	var receivedUserAgent string

	// Create test server that captures the User-Agent header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUserAgent = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"request_id": "test-123"}`))
	}))
	defer server.Close()

	client := nylas.NewHTTPClient()
	client.SetBaseURL(server.URL)
	client.SetCredentials("test-client", "test-secret", "test-api-key")

	// Make a request that will trigger the User-Agent header
	_, _ = client.ListGrants(context.Background())

	// Verify User-Agent was set correctly
	expectedUA := version.UserAgent()
	assert.Equal(t, expectedUA, receivedUserAgent, "User-Agent header should match version.UserAgent()")

	// Verify format: nylas-cli/VERSION (OS/ARCH)
	assert.True(t, strings.HasPrefix(receivedUserAgent, "nylas-cli/"), "User-Agent should start with 'nylas-cli/'")
	assert.Contains(t, receivedUserAgent, "(", "User-Agent should contain platform info")
	assert.Contains(t, receivedUserAgent, ")", "User-Agent should contain platform info")
}
