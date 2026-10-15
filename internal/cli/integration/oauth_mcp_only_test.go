//go:build integration

package integration

import (
	"encoding/base64"
	"encoding/json"
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

// An MCP-only device sign-in against a fake AS: it never requests
// dashboard.session, never calls the dashboard exchange, and is stored as
// its own credential that status lists and token --credential reads.
func TestCLI_OAuthMCPOnly_FakeServerLogin(t *testing.T) {
	require.NotEmpty(t, testBinary, "build the CLI before this integration test")
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	var exchanges atomic.Int32
	payload, err := json.Marshal(map[string]any{
		"sub": "user-1", "org": "org-1", "aud": server.URL, "scope": "email.read offline_access",
		"exp": time.Now().Add(15 * time.Minute).Unix(), "principal": map[string]string{"id": "V1StGXR8_Z5jdHi6B-myT", "region": "us"},
	})
	require.NoError(t, err)
	token := "e30." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(domain.OAuthServerMetadata{Issuer: server.URL, AuthorizationEndpoint: server.URL + "/authorize", TokenEndpoint: server.URL + "/token", DeviceAuthorizationEndpoint: server.URL + "/device"})
	})
	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		assert.NotContains(t, strings.Fields(r.Form.Get("scope")), domain.OAuthScopeDashboardSession)
		assert.Equal(t, server.URL, r.Form.Get("resource"))
		_ = json.NewEncoder(w).Encode(domain.OAuthDeviceAuthorization{DeviceCode: "memory-only-device-code", UserCode: "BCDF-GHJK", VerificationURI: server.URL + "/verify", ExpiresIn: 600, Interval: 1})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(domain.OAuthTokens{AccessToken: token, RefreshToken: "refresh-secret", TokenType: "Bearer", Scope: "email.read offline_access", ExpiresIn: 900})
	})
	mux.HandleFunc("/auth/cli/oauth/exchange", func(w http.ResponseWriter, _ *http.Request) {
		exchanges.Add(1)
		w.WriteHeader(http.StatusForbidden)
	})
	env := map[string]string{"NYLAS_DASHBOARD_ACCOUNT_URL": server.URL, "NYLAS_MCP_US_URL": server.URL, "NYLAS_DISABLE_KEYRING": "true",
		"XDG_CONFIG_HOME": t.TempDir(), "SSH_CONNECTION": "", "SSH_CLIENT": "", "SSH_TTY": "", "NYLAS_OAUTH_CLIENT_ID": ""}

	out, stderr, err := runCLIWithOverrides(30*time.Second, env, "oauth", "login", "--mcp-only", "--device", "--region", "us")
	require.NoError(t, err, stderr)
	assert.Contains(t, out, "MCP only")
	assert.Contains(t, out, "V1StGXR8_Z5jdHi6B-myT (us)")
	assert.Zero(t, exchanges.Load(), "an MCP-only login is never exchanged for a dashboard session")
	assert.NotContains(t, out+stderr, token)
	assert.NotContains(t, out+stderr, "refresh-secret")

	status, stderr, err := runCLIWithOverrides(10*time.Second, env, "oauth", "status")
	require.NoError(t, err, stderr)
	assert.Contains(t, status, "No default session")
	assert.Contains(t, status, "limited to an IAM principal")
	assert.NotContains(t, status, token)
	id := ""
	for _, line := range strings.Split(status, "\n") {
		if rest, ok := strings.CutPrefix(line, "MCP credential "); ok {
			id = strings.TrimSpace(rest)
		}
	}
	require.NotEmpty(t, id)

	printed, stderr, err := runCLIWithOverrides(10*time.Second, env, "oauth", "token", "--credential", id)
	require.NoError(t, err, stderr)
	assert.Equal(t, token, strings.TrimSpace(printed))
}
