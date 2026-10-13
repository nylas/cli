//go:build integration

package integration

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The real CLI talks to a fake AS and stores its session in an isolated file
// store. No provider credentials or production services are involved.
func TestCLI_OAuthDevice_FakeServerLogin(t *testing.T) {
	require.NotEmpty(t, testBinary, "build the CLI before this integration test")
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	deviceCode := "memory-only-device-code"
	var polls atomic.Int32
	payload, err := json.Marshal(map[string]any{"sub": "user", "org": "org", "scope": "openid email", "exp": time.Now().Add(15 * time.Minute).Unix(), "aud": server.URL})
	require.NoError(t, err)
	token := "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(domain.OAuthServerMetadata{Issuer: server.URL, AuthorizationEndpoint: server.URL + "/authorize", TokenEndpoint: server.URL + "/token", DeviceAuthorizationEndpoint: server.URL + "/device", ScopesSupported: []string{"openid", "email"}})
	})
	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, domain.DefaultOAuthClientID, r.Form.Get("client_id"))
		assert.Equal(t, "openid email", r.Form.Get("scope"))
		_ = json.NewEncoder(w).Encode(domain.OAuthDeviceAuthorization{DeviceCode: deviceCode, UserCode: "BCDF-GHJK", VerificationURI: server.URL + "/verify", VerificationURIComplete: server.URL + "/verify?user_code=BCDF-GHJK", ExpiresIn: 600, Interval: 1})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.Equal(t, domain.OAuthDeviceGrant, r.Form.Get("grant_type"))
		assert.Equal(t, deviceCode, r.Form.Get("device_code"))
		if polls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
			return
		}
		_ = json.NewEncoder(w).Encode(domain.OAuthTokens{AccessToken: token, TokenType: "Bearer", Scope: "openid email", ExpiresIn: 900})
	})
	env := map[string]string{"NYLAS_DASHBOARD_ACCOUNT_URL": server.URL, "NYLAS_DISABLE_KEYRING": "true", "XDG_CONFIG_HOME": t.TempDir(), "SSH_CONNECTION": "", "SSH_CLIENT": "", "SSH_TTY": "", "NYLAS_OAUTH_CLIENT_ID": ""}
	out, stderr, err := runCLIWithOverrides(30*time.Second, env, "oauth", "login", "--device", "--scope", "openid,email")
	require.NoError(t, err, stderr)
	assert.Contains(t, out, "BCDF-GHJK")
	assert.Contains(t, out, "Logged in")
	assert.NotContains(t, out+stderr, deviceCode)
	assert.NotContains(t, out+stderr, token)
	assert.EqualValues(t, 2, polls.Load())
	status, stderr, err := runCLIWithOverrides(10*time.Second, env, "oauth", "status")
	require.NoError(t, err, stderr)
	assert.Contains(t, status, server.URL)
	assert.NotContains(t, status+stderr, token)
}
