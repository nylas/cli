//go:build !integration

package oauthas

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// discoveryDocument mirrors what dashboard-account serves, including the fact
// that every endpoint is an absolute URL derived from OAUTH_ISSUER rather
// than a path relative to where discovery was fetched.
func discoveryDocument(issuer string) map[string]any {
	return map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/oauth/authorize",
		"token_endpoint":                        issuer + "/oauth/token",
		"userinfo_endpoint":                     issuer + "/oauth/userinfo",
		"revocation_endpoint":                   issuer + "/oauth/revoke",
		"registration_endpoint":                 issuer + "/oauth/register",
		"jwks_uri":                              issuer + "/.well-known/jwks.json",
		"scopes_supported":                      []string{"openid", "email", "offline_access"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"response_types_supported":              []string{"code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic", "none"},
	}
}

// newTestServer starts a stub authorization server. handler receives every
// request other than discovery.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(discoveryDocument(server.URL))
	})
	if handler != nil {
		mux.HandleFunc("/", handler)
	}

	return server
}

func TestClient_Metadata(t *testing.T) {
	server := newTestServer(t, nil)
	client := NewClient(server.URL)

	metadata, err := client.Metadata(context.Background())

	require.NoError(t, err)
	assert.Equal(t, server.URL, metadata.Issuer)
	assert.Equal(t, server.URL+"/oauth/token", metadata.TokenEndpoint)
	assert.Equal(t, []string{"S256"}, metadata.CodeChallengeMethodsSupported)
}

func TestClient_Metadata_IsFetchedOnce(t *testing.T) {
	calls := 0
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_ = json.NewEncoder(w).Encode(discoveryDocument(server.URL))
	})

	client := NewClient(server.URL)
	_, err := client.Metadata(context.Background())
	require.NoError(t, err)
	_, err = client.Metadata(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, calls, "metadata should be cached for the client's lifetime")
}

func TestClient_Metadata_RejectsUnrelatedJSON(t *testing.T) {
	// Pointing the CLI at the wrong port is the most common local-setup
	// mistake; it must fail naming the missing fields, not much later with
	// an empty-URL request.
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	_, err := NewClient(server.URL).Metadata(context.Background())

	require.ErrorIs(t, err, domain.ErrOAuthMetadata)
}

func TestClient_AuthorizationURL(t *testing.T) {
	server := newTestServer(t, nil)
	client := NewClient(server.URL)

	raw, err := client.AuthorizationURL(context.Background(), domain.OAuthAuthorizationParams{
		ClientID:      "client-123",
		RedirectURI:   "http://localhost:9007/callback",
		Scopes:        domain.DefaultOAuthScopes(),
		State:         "state-abc",
		CodeChallenge: "challenge-xyz",
		Nonce:         "nonce-1",
	})
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	query := parsed.Query()

	assert.Equal(t, server.URL+"/oauth/authorize", parsed.Scheme+"://"+parsed.Host+parsed.Path)
	assert.Equal(t, "code", query.Get("response_type"))
	assert.Equal(t, "client-123", query.Get("client_id"))
	assert.Equal(t, "http://localhost:9007/callback", query.Get("redirect_uri"))
	assert.Equal(t, "openid email offline_access", query.Get("scope"))
	assert.Equal(t, "state-abc", query.Get("state"))
	assert.Equal(t, "challenge-xyz", query.Get("code_challenge"))
	assert.Equal(t, "S256", query.Get("code_challenge_method"))
	assert.Equal(t, "nonce-1", query.Get("nonce"))
}

func TestClient_AuthorizationURL_OmitsEmptyNonce(t *testing.T) {
	server := newTestServer(t, nil)

	raw, err := NewClient(server.URL).AuthorizationURL(context.Background(), domain.OAuthAuthorizationParams{
		ClientID: "client-123",
	})
	require.NoError(t, err)

	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.False(t, parsed.Query().Has("nonce"))
	assert.False(t, parsed.Query().Has("resource"), "no resource indicator unless one was asked for")
}

func TestClient_SendsResourceIndicatorOnEveryLeg(t *testing.T) {
	// RFC 8707: the resource goes on the authorization request AND on each
	// token request, or the token is issued without the MCP audience and the
	// MCP server refuses it.
	var forms []url.Values
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		forms = append(forms, r.PostForm)
		_, _ = w.Write([]byte(`{"access_token":"at","token_type":"Bearer","expires_in":900,"refresh_token":"rt"}`))
	})
	client := NewClient(server.URL)
	ctx := context.Background()

	raw, err := client.AuthorizationURL(ctx, domain.OAuthAuthorizationParams{
		ClientID: "client-123",
		Resource: domain.MCPResourceEU,
	})
	require.NoError(t, err)
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, domain.MCPResourceEU, parsed.Query().Get("resource"))

	_, err = client.ExchangeCode(ctx, domain.OAuthCodeExchange{ClientID: "client-123", Code: "c", Resource: domain.MCPResourceEU})
	require.NoError(t, err)
	_, err = client.Refresh(ctx, "client-123", "rt-1", domain.MCPResourceEU)
	require.NoError(t, err)
	_, err = client.Refresh(ctx, "client-123", "rt-2", "")
	require.NoError(t, err)

	require.Len(t, forms, 3)
	assert.Equal(t, domain.MCPResourceEU, forms[0].Get("resource"), "code exchange")
	assert.Equal(t, domain.MCPResourceEU, forms[1].Get("resource"), "refresh")
	assert.False(t, forms[2].Has("resource"), "a session with no resource sends none")
}

func TestClient_ExchangeCode(t *testing.T) {
	var got url.Values
	var contentType string

	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/oauth/token", r.URL.Path)
		require.NoError(t, r.ParseForm())
		got = r.PostForm
		contentType = r.Header.Get("Content-Type")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "at-1",
			"token_type": "Bearer",
			"expires_in": 3600,
			"refresh_token": "rt-1",
			"id_token": "idt-1",
			"scope": "openid email offline_access"
		}`))
	})

	client := NewClient(server.URL)
	client.now = func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }

	tokens, err := client.ExchangeCode(context.Background(), domain.OAuthCodeExchange{
		ClientID:     "client-123",
		Code:         "auth-code",
		RedirectURI:  "http://localhost:9007/callback",
		CodeVerifier: "verifier-abc",
	})
	require.NoError(t, err)

	assert.Equal(t, "application/x-www-form-urlencoded", contentType)
	assert.Equal(t, "authorization_code", got.Get("grant_type"))
	assert.Equal(t, "auth-code", got.Get("code"))
	assert.Equal(t, "verifier-abc", got.Get("code_verifier"))
	assert.Equal(t, "client-123", got.Get("client_id"))
	assert.False(t, got.Has("client_secret"), "a public client must not send a secret")

	assert.Equal(t, "at-1", tokens.AccessToken)
	assert.Equal(t, "rt-1", tokens.RefreshToken)
	assert.Equal(t, "idt-1", tokens.IDToken)
	assert.Equal(t, time.Date(2026, 9, 21, 13, 0, 0, 0, time.UTC), tokens.ExpiresAt)
}

func TestClient_ExchangeCode_SendsSecretForConfidentialClient(t *testing.T) {
	var got url.Values
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		got = r.PostForm
		_, _ = w.Write([]byte(`{"access_token":"at-1","token_type":"Bearer","expires_in":3600}`))
	})

	_, err := NewClient(server.URL).ExchangeCode(context.Background(), domain.OAuthCodeExchange{
		ClientID:     "client-123",
		ClientSecret: "shh",
		Code:         "auth-code",
	})
	require.NoError(t, err)

	assert.Equal(t, "shh", got.Get("client_secret"))
}

func TestClient_ExchangeCode_SurfacesOAuthError(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"code already redeemed"}`))
	})

	_, err := NewClient(server.URL).ExchangeCode(context.Background(), domain.OAuthCodeExchange{ClientID: "c"})

	var oauthErr *domain.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, "invalid_grant", oauthErr.Code)
	assert.Equal(t, "code already redeemed", oauthErr.Description)
	assert.Equal(t, http.StatusBadRequest, oauthErr.StatusCode)
}

func TestClient_ExchangeCode_FallsBackWhenErrorIsNotRFCShaped(t *testing.T) {
	// The house error envelope wraps "error" as an object, which no OAuth
	// client can read; the status code must still reach the user.
	server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL","message":"boom"}}`))
	})

	_, err := NewClient(server.URL).ExchangeCode(context.Background(), domain.OAuthCodeExchange{ClientID: "c"})

	var oauthErr *domain.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, http.StatusInternalServerError, oauthErr.StatusCode)
	assert.Contains(t, oauthErr.Description, "boom")
}

func TestClient_ExchangeCode_RejectsResponseWithoutAccessToken(t *testing.T) {
	server := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token_type":"Bearer"}`))
	})

	_, err := NewClient(server.URL).ExchangeCode(context.Background(), domain.OAuthCodeExchange{ClientID: "c"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "access_token")
}

func TestClient_Refresh_ReturnsRotatedToken(t *testing.T) {
	var got url.Values
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		got = r.PostForm
		// The server rotates on every use and burns the family on replay,
		// so the caller must persist this new value.
		_, _ = w.Write([]byte(`{"access_token":"at-2","token_type":"Bearer","expires_in":3600,"refresh_token":"rt-2"}`))
	})

	tokens, err := NewClient(server.URL).Refresh(context.Background(), "client-123", "rt-1", "")
	require.NoError(t, err)

	assert.Equal(t, "refresh_token", got.Get("grant_type"))
	assert.Equal(t, "rt-1", got.Get("refresh_token"))
	assert.Equal(t, "client-123", got.Get("client_id"))
	assert.Equal(t, "rt-2", tokens.RefreshToken)
}

func TestClient_Revoke(t *testing.T) {
	var got url.Values
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/oauth/revoke", r.URL.Path)
		require.NoError(t, r.ParseForm())
		got = r.PostForm
		w.WriteHeader(http.StatusOK)
	})

	err := NewClient(server.URL).Revoke(context.Background(), "client-123", "rt-1")
	require.NoError(t, err)

	assert.Equal(t, "rt-1", got.Get("token"))
	assert.Equal(t, "client-123", got.Get("client_id"))
}

func TestClient_UserInfo(t *testing.T) {
	var authorization string
	server := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/oauth/userinfo", r.URL.Path)
		authorization = r.Header.Get("Authorization")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":"user-1","email":"dev@example.test","email_verified":true,"org":"org-1"}`))
	})

	info, err := NewClient(server.URL).UserInfo(context.Background(), "at-1")
	require.NoError(t, err)

	assert.Equal(t, "Bearer at-1", authorization)
	assert.Equal(t, "user-1", info.Subject)
	assert.Equal(t, "dev@example.test", info.Email)
	assert.True(t, info.EmailVerified)
	assert.Equal(t, "org-1", info.Org)
}

func TestClient_FollowsDiscoveredEndpointHost(t *testing.T) {
	// dashboard-account builds every endpoint from OAUTH_ISSUER, which in a
	// tunnelled dev setup is a different host from where discovery was
	// fetched. The client must use the advertised URL, not its own base.
	tokenCalls := 0
	issuerMux := http.NewServeMux()
	issuer := httptest.NewServer(issuerMux)
	t.Cleanup(issuer.Close)
	issuerMux.HandleFunc("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		tokenCalls++
		_, _ = w.Write([]byte(`{"access_token":"at-1","token_type":"Bearer","expires_in":3600}`))
	})

	discoveryMux := http.NewServeMux()
	discovery := httptest.NewServer(discoveryMux)
	t.Cleanup(discovery.Close)
	discoveryMux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(discoveryDocument(issuer.URL))
	})

	_, err := NewClient(discovery.URL).ExchangeCode(context.Background(), domain.OAuthCodeExchange{ClientID: "c"})
	require.NoError(t, err)

	assert.Equal(t, 1, tokenCalls)
}
