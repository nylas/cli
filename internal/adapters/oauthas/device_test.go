//go:build !integration

package oauthas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceClient_DiscoveryAndForms(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc(discoveryPath, func(w http.ResponseWriter, _ *http.Request) {
		doc := discoveryDocument(server.URL)
		doc["device_authorization_endpoint"] = server.URL + "/device-start"
		_ = json.NewEncoder(w).Encode(doc)
	})
	mux.HandleFunc("/device-start", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, domain.DefaultOAuthClientID, r.Form.Get("client_id"))
		assert.Equal(t, "openid dashboard.session", r.Form.Get("scope"))
		assert.Equal(t, "https://mcp.us.nylas.com", r.Form.Get("resource"))
		_ = json.NewEncoder(w).Encode(domain.OAuthDeviceAuthorization{DeviceCode: strings.Repeat("a", 32), UserCode: "BCDF-GHJK", VerificationURI: server.URL + "/verify", ExpiresIn: 600, Interval: 5})
	})
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, domain.OAuthDeviceGrant, r.Form.Get("grant_type"))
		assert.Equal(t, strings.Repeat("a", 32), r.Form.Get("device_code"))
		assert.Equal(t, "https://mcp.us.nylas.com", r.Form.Get("resource"))
		_ = json.NewEncoder(w).Encode(domain.OAuthTokens{AccessToken: "token", ExpiresIn: 900})
	})
	c := NewClient(server.URL)
	auth, err := c.StartDeviceAuthorization(context.Background(), domain.DefaultOAuthClientID, []string{"openid", "dashboard.session"}, "https://mcp.us.nylas.com")
	require.NoError(t, err)
	tokens, err := c.PollDeviceTokens(context.Background(), domain.DefaultOAuthClientID, auth.DeviceCode, "https://mcp.us.nylas.com")
	require.NoError(t, err)
	assert.Equal(t, "token", tokens.AccessToken)
}

func TestDeviceClient_ErrorsNeverEchoResponseSecrets(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"authorization_pending","error_description":"secret-device-code"}`))
	})
	_, err := NewClient(server.URL).PollDeviceTokens(context.Background(), domain.DefaultOAuthClientID, strings.Repeat("a", 32), "")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "secret-device-code")
	var oauthErr *domain.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, "authorization_pending", oauthErr.Code)
}

func TestMetadata_RejectsCrossHostDeviceEndpoint(t *testing.T) {
	metadata := domain.OAuthServerMetadata{Issuer: "https://auth.example.test", AuthorizationEndpoint: "https://auth.example.test/authorize", TokenEndpoint: "https://auth.example.test/token", DeviceAuthorizationEndpoint: "https://other.example.test/device"}
	require.ErrorIs(t, metadata.ValidateFor(metadata.Issuer), domain.ErrOAuthMetadata)
}
