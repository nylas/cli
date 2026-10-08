//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/oauthas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests drive the real dashboard-account authorization server. Point
// NYLAS_OAUTH_AS_URL at it (for a Tilt stack, http://localhost:3001) with its
// /dev routes enabled; without the variable they skip.
const oauthASEnv = "NYLAS_OAUTH_AS_URL"

// oauthTestRedirectURI carries a port while the static client is registered
// with http://127.0.0.1/callback and no port, so every exchange here also
// exercises the RFC 8252 rule that only the port of a loopback redirect URI
// is free. It is the spelling `nylas oauth login` advertises.
const oauthTestRedirectURI = "http://127.0.0.1:9007/callback"

// oauthTestClientID is the static public client, or NYLAS_OAUTH_CLIENT_ID
// when the server under test registers the CLI under another id.
func oauthTestClientID(t *testing.T) string {
	t.Helper()
	if id := os.Getenv("NYLAS_OAUTH_CLIENT_ID"); id != "" {
		require.NoError(t, domain.ValidateOAuthClientID(id))
		return id
	}
	return domain.DefaultOAuthClientID
}

func oauthUpstream(t *testing.T) string {
	t.Helper()
	upstream := strings.TrimRight(os.Getenv(oauthASEnv), "/")
	if upstream == "" {
		t.Skipf("set %s to run the OAuth authorization server integration tests", oauthASEnv)
	}
	return upstream
}

// newNormalizedASProxy fronts the authorization server so its discovery
// document advertises endpoints the test can actually reach.
//
// dashboard-account builds every endpoint from OAUTH_ISSUER, which in a local
// stack is often a tunnel hostname that is stale or unreachable. The client
// under test is spec-correct and follows whatever the document says, so
// without this it would chase a dead host. Only the origin is rewritten;
// every request is forwarded to the real server untouched.
func newNormalizedASProxy(t *testing.T, upstream string) *httptest.Server {
	t.Helper()

	// The proxy must not follow redirects itself: /oauth/authorize answers an
	// anonymous caller with a 302 to login, and swallowing it here would hide
	// whether the server accepted the authorization request at all.
	forwarder := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	var proxy *httptest.Server
	proxy = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		outbound, err := http.NewRequestWithContext(r.Context(), r.Method, upstream+r.URL.RequestURI(), bytes.NewReader(body))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		for name, values := range r.Header {
			for _, value := range values {
				outbound.Header.Add(name, value)
			}
		}
		outbound.Header.Del("Accept-Encoding")

		resp, err := forwarder.Do(outbound)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/.well-known/") {
			payload = rewriteIssuerOrigin(t, payload, proxy.URL)
		}

		for name, values := range resp.Header {
			if strings.EqualFold(name, "Content-Length") {
				continue
			}
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(payload)
	}))
	t.Cleanup(proxy.Close)

	return proxy
}

// rewriteIssuerOrigin replaces the advertised issuer origin with the proxy's.
func rewriteIssuerOrigin(t *testing.T, document []byte, proxyURL string) []byte {
	t.Helper()

	var parsed map[string]any
	if err := json.Unmarshal(document, &parsed); err != nil {
		return document
	}
	issuer, _ := parsed["issuer"].(string)
	if issuer == "" || issuer == proxyURL {
		return document
	}
	return []byte(strings.ReplaceAll(string(document), issuer, proxyURL))
}

type oauthTestSubject struct {
	userPublicID string
	orgPublicID  string
	email        string
}

func postDevJSON(t *testing.T, baseURL, path string, body, result any) {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	resp, err := http.Post(baseURL+path, "application/json", bytes.NewReader(payload))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Less(t, resp.StatusCode, 300,
		"%s failed (%d) — are the /dev routes enabled on the authorization server? %s", path, resp.StatusCode, raw)

	if result == nil {
		return
	}
	// Dev routes answer in the house {"data": ...} envelope.
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(raw, &envelope))
	require.NoError(t, json.Unmarshal(envelope.Data, result))
}

// seedOAuthSubject creates a fresh user and organization to authorize as.
func seedOAuthSubject(t *testing.T, upstream string) oauthTestSubject {
	t.Helper()

	email := fmt.Sprintf("cli-oauth-it-%d@nylas.com", time.Now().UnixNano())
	var created struct {
		User struct {
			PublicID string `json:"publicId"`
			Email    string `json:"email"`
		} `json:"user"`
		Organization struct {
			PublicID string `json:"publicId"`
		} `json:"organization"`
	}
	postDevJSON(t, upstream, "/dev/create-seed-user", map[string]any{
		"email":              email,
		"firstName":          "CLI",
		"lastName":           "Integration",
		"emailVerified":      true,
		"createOrganization": true,
		"organizationName":   "CLI Integration Org",
		"organizationRegion": "us",
	}, &created)

	require.NotEmpty(t, created.User.PublicID)
	require.NotEmpty(t, created.Organization.PublicID)

	return oauthTestSubject{
		userPublicID: created.User.PublicID,
		orgPublicID:  created.Organization.PublicID,
		email:        created.User.Email,
	}
}

// mintAuthorizationCode bypasses the browser consent screen. The consent
// screen needs a UAS-connected mailbox, which a local stack does not have;
// the dev route exists precisely so the token exchange stays testable.
func mintAuthorizationCode(t *testing.T, upstream, clientID string, subject oauthTestSubject, challenge string) string {
	t.Helper()

	// A refresh token is only issued against an active consent grant.
	postDevJSON(t, upstream, "/dev/oauth/consent-grant", map[string]any{
		"clientId":     clientID,
		"userPublicId": subject.userPublicID,
		"orgPublicId":  subject.orgPublicID,
		"scopes":       domain.DefaultOAuthScopes(),
	}, nil)

	var minted struct {
		Code string `json:"code"`
	}
	postDevJSON(t, upstream, "/dev/oauth/authorization-code", map[string]any{
		"clientId":      clientID,
		"userPublicId":  subject.userPublicID,
		"orgPublicId":   subject.orgPublicID,
		"redirectUri":   oauthTestRedirectURI,
		"scopes":        domain.DefaultOAuthScopes(),
		"codeChallenge": challenge,
	}, &minted)

	require.NotEmpty(t, minted.Code)
	return minted.Code
}

func TestOAuthAS_DiscoveryMatchesWhatTheClientNeeds(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)

	metadata, err := client.Metadata(context.Background())
	require.NoError(t, err)

	assert.Contains(t, metadata.CodeChallengeMethodsSupported, "S256")
	assert.Contains(t, metadata.TokenEndpointAuthMethods, "none",
		"the CLI is a public client and cannot authenticate otherwise")
	assert.Contains(t, metadata.ScopesSupported, domain.OAuthScopeOfflineAccess,
		"without offline_access the server issues no refresh token")
	assert.NotEmpty(t, metadata.RevocationEndpoint)
	assert.NotEmpty(t, metadata.UserInfoEndpoint)
}

func TestOAuthAS_FullAuthorizationCodeExchange(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	ctx := context.Background()

	subject := seedOAuthSubject(t, upstream)
	clientID := oauthTestClientID(t)

	pkce, err := domain.NewPKCE()
	require.NoError(t, err)
	code := mintAuthorizationCode(t, upstream, clientID, subject, pkce.Challenge)

	tokens, err := client.ExchangeCode(ctx, domain.OAuthCodeExchange{
		ClientID:     clientID,
		Code:         code,
		RedirectURI:  oauthTestRedirectURI,
		CodeVerifier: pkce.Verifier,
	})
	require.NoError(t, err)

	assert.NotEmpty(t, tokens.AccessToken)
	assert.Equal(t, "Bearer", tokens.TokenType)
	assert.NotEmpty(t, tokens.RefreshToken, "offline_access was consented")
	assert.NotEmpty(t, tokens.IDToken, "openid was consented")
	assert.False(t, tokens.ExpiresAt.IsZero(), "ExpiresAt must be derived from expires_in")
	assert.False(t, tokens.IsExpired(time.Now()))

	// The CLI decodes (never verifies) these for `nylas oauth status`.
	claims, err := domain.DecodeOAuthAccessToken(tokens.AccessToken)
	require.NoError(t, err, "the server should issue a JWT access token")
	assert.Equal(t, subject.userPublicID, claims.Subject)
	assert.Equal(t, clientID, claims.ClientID)
	assert.False(t, claims.ExpiresAt.IsZero())

	info, err := client.UserInfo(ctx, tokens.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, subject.userPublicID, info.Subject)
	assert.Equal(t, subject.email, info.Email)
	assert.Equal(t, subject.orgPublicID, info.Org)
}

func TestOAuthAS_RejectsReplayedAuthorizationCode(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	ctx := context.Background()

	subject := seedOAuthSubject(t, upstream)
	clientID := oauthTestClientID(t)
	pkce, err := domain.NewPKCE()
	require.NoError(t, err)
	code := mintAuthorizationCode(t, upstream, clientID, subject, pkce.Challenge)

	exchange := domain.OAuthCodeExchange{
		ClientID:     clientID,
		Code:         code,
		RedirectURI:  oauthTestRedirectURI,
		CodeVerifier: pkce.Verifier,
	}
	_, err = client.ExchangeCode(ctx, exchange)
	require.NoError(t, err)

	_, err = client.ExchangeCode(ctx, exchange)

	var oauthErr *domain.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, "invalid_grant", oauthErr.Code)
}

func TestOAuthAS_RejectsMismatchedCodeVerifier(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	ctx := context.Background()

	subject := seedOAuthSubject(t, upstream)
	clientID := oauthTestClientID(t)
	pkce, err := domain.NewPKCE()
	require.NoError(t, err)
	other, err := domain.NewPKCE()
	require.NoError(t, err)
	code := mintAuthorizationCode(t, upstream, clientID, subject, pkce.Challenge)

	_, err = client.ExchangeCode(ctx, domain.OAuthCodeExchange{
		ClientID:     clientID,
		Code:         code,
		RedirectURI:  oauthTestRedirectURI,
		CodeVerifier: other.Verifier,
	})

	var oauthErr *domain.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, "invalid_grant", oauthErr.Code)
}

func TestOAuthAS_RefreshRotatesAndDetectsReuse(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	ctx := context.Background()

	subject := seedOAuthSubject(t, upstream)
	clientID := oauthTestClientID(t)
	pkce, err := domain.NewPKCE()
	require.NoError(t, err)
	code := mintAuthorizationCode(t, upstream, clientID, subject, pkce.Challenge)

	tokens, err := client.ExchangeCode(ctx, domain.OAuthCodeExchange{
		ClientID:     clientID,
		Code:         code,
		RedirectURI:  oauthTestRedirectURI,
		CodeVerifier: pkce.Verifier,
	})
	require.NoError(t, err)
	require.NotEmpty(t, tokens.RefreshToken)

	refreshed, err := client.Refresh(ctx, clientID, tokens.RefreshToken, "")
	require.NoError(t, err)

	assert.NotEmpty(t, refreshed.AccessToken)
	assert.NotEmpty(t, refreshed.RefreshToken)
	assert.NotEqual(t, tokens.RefreshToken, refreshed.RefreshToken,
		"the server rotates the refresh token on every use")

	// Replaying the consumed token is what burns the whole family, which is
	// why oauthlogin never carries an old refresh token forward.
	_, err = client.Refresh(ctx, clientID, tokens.RefreshToken, "")
	var oauthErr *domain.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, "invalid_grant", oauthErr.Code)
}

func TestOAuthAS_RevokeEndsTheSession(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	ctx := context.Background()

	subject := seedOAuthSubject(t, upstream)
	clientID := oauthTestClientID(t)
	pkce, err := domain.NewPKCE()
	require.NoError(t, err)
	code := mintAuthorizationCode(t, upstream, clientID, subject, pkce.Challenge)

	tokens, err := client.ExchangeCode(ctx, domain.OAuthCodeExchange{
		ClientID:     clientID,
		Code:         code,
		RedirectURI:  oauthTestRedirectURI,
		CodeVerifier: pkce.Verifier,
	})
	require.NoError(t, err)

	require.NoError(t, client.Revoke(ctx, clientID, tokens.RefreshToken))

	_, err = client.Refresh(ctx, clientID, tokens.RefreshToken, "")
	require.Error(t, err, "a revoked refresh token must not mint new tokens")
}

func TestOAuthAS_AuthorizationURLIsAcceptedByTheServer(t *testing.T) {
	// The browser step cannot run headless, but the server still validates
	// the request shape before it redirects to login — a malformed
	// code_challenge or scope is rejected here rather than at that redirect.
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	ctx := context.Background()

	clientID := oauthTestClientID(t)
	pkce, err := domain.NewPKCE()
	require.NoError(t, err)
	state, err := domain.NewOAuthState()
	require.NoError(t, err)

	authURL, err := client.AuthorizationURL(ctx, domain.OAuthAuthorizationParams{
		ClientID:      clientID,
		RedirectURI:   oauthTestRedirectURI,
		Scopes:        domain.DefaultOAuthScopes(),
		State:         state,
		CodeChallenge: pkce.Challenge,
	})
	require.NoError(t, err)

	httpClient := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := httpClient.Get(authURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	// An anonymous caller is sent to login; a rejected request would instead
	// come back 400, or redirect to the callback carrying ?error=.
	require.Equal(t, http.StatusFound, resp.StatusCode, "expected a redirect to login")
	location, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	assert.Empty(t, location.Query().Get("error"), "the server rejected the authorization request")
}

// exchangeFreshSession seeds a subject and returns its first token set.
func exchangeFreshSession(t *testing.T, client *oauthas.Client, upstream string) *domain.OAuthTokens {
	t.Helper()
	clientID := oauthTestClientID(t)
	subject := seedOAuthSubject(t, upstream)
	pkce, err := domain.NewPKCE()
	require.NoError(t, err)
	code := mintAuthorizationCode(t, upstream, clientID, subject, pkce.Challenge)

	tokens, err := client.ExchangeCode(context.Background(), domain.OAuthCodeExchange{
		ClientID:     clientID,
		Code:         code,
		RedirectURI:  oauthTestRedirectURI,
		CodeVerifier: pkce.Verifier,
	})
	require.NoError(t, err)
	require.NotEmpty(t, tokens.RefreshToken)
	return tokens
}

func TestOAuthAS_AuthorizationURLWithMCPResourceIsAccepted(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	pkce, err := domain.NewPKCE()
	require.NoError(t, err)
	state, err := domain.NewOAuthState()
	require.NoError(t, err)

	authURL, err := client.AuthorizationURL(context.Background(), domain.OAuthAuthorizationParams{
		ClientID:      oauthTestClientID(t),
		RedirectURI:   oauthTestRedirectURI,
		Scopes:        domain.DefaultOAuthScopes(),
		State:         state,
		CodeChallenge: pkce.Challenge,
		Resource:      domain.MCPResourceUS,
	})
	require.NoError(t, err)

	httpClient := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := httpClient.Get(authURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusFound, resp.StatusCode)
	location, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	assert.Empty(t, location.Query().Get("error"), "the MCP resource indicator was rejected")
}

func TestOAuthAS_RefreshForMCPResourceIssuesMCPAudience(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	tokens := exchangeFreshSession(t, client, upstream)

	refreshed, err := client.Refresh(context.Background(), oauthTestClientID(t), tokens.RefreshToken, domain.MCPResourceUS)
	require.NoError(t, err)

	claims, err := domain.DecodeOAuthAccessToken(refreshed.AccessToken)
	require.NoError(t, err)
	endpoint, err := domain.MCPEndpointFromAudience(claims.Audience)
	require.NoError(t, err, "a token requested for the MCP server must name it as its audience")
	assert.Equal(t, domain.MCPResourceUS, endpoint)
}

func TestOAuthAS_RejectsUnknownResource(t *testing.T) {
	upstream := oauthUpstream(t)
	client := oauthas.NewClient(newNormalizedASProxy(t, upstream).URL)
	tokens := exchangeFreshSession(t, client, upstream)

	_, err := client.Refresh(context.Background(), oauthTestClientID(t), tokens.RefreshToken, "https://not-a-nylas-resource.example")

	var oauthErr *domain.OAuthError
	require.ErrorAs(t, err, &oauthErr)
	assert.Equal(t, "invalid_target", oauthErr.Code)
}
