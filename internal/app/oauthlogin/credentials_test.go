//go:build !integration

package oauthlogin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	principalA = "V1StGXR8_Z5jdHi6B-myT"
	principalB = "Uakgb_J5m9g-0JDMbcJqL"
)

// tokenClaims are the claims an access token in these tests carries.
type tokenClaims struct {
	aud       string
	scope     string
	org       string
	principal *domain.OAuthTokenPrincipal
}

func principalToken(t *testing.T, c tokenClaims) string {
	t.Helper()
	claims := map[string]any{
		"sub":       "user-1",
		"org":       c.org,
		"aud":       c.aud,
		"scope":     c.scope,
		"client_id": domain.DefaultOAuthClientID,
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
	}
	if c.principal != nil {
		claims["principal"] = c.principal
	}
	body, err := json.Marshal(claims)
	require.NoError(t, err)
	return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(body) + ".sig"
}

func usPrincipal(id string) *domain.OAuthTokenPrincipal {
	return &domain.OAuthTokenPrincipal{ID: id, Region: "us"}
}

const mcpOnlyScope = "email.read offline_access"

// issue makes the fake server answer the next code exchange with a token
// carrying c, and a refresh token named refresh.
func (f *fixture) issue(t *testing.T, c tokenClaims, refresh string) {
	t.Helper()
	token := principalToken(t, c)
	f.client.ExchangeCodeFunc = func(context.Context, domain.OAuthCodeExchange) (*domain.OAuthTokens, error) {
		return &domain.OAuthTokens{AccessToken: token, RefreshToken: refresh, Scope: c.scope, ExpiresAt: f.clock.Add(15 * time.Minute)}, nil
	}
}

func mcpOnlyLogin() LoginOptions {
	return LoginOptions{Scopes: domain.MCPOnlyOAuthScopes(), Resource: domain.MCPResourceUS, MCPOnly: true}
}

func TestMCPOnlyOAuthScopes_NeverRequestDashboardSession(t *testing.T) {
	assert.NotContains(t, domain.MCPOnlyOAuthScopes(), domain.OAuthScopeDashboardSession)
	assert.Contains(t, domain.MCPOAuthScopes(), domain.OAuthScopeDashboardSession, "the dashboard-capable preset is unchanged")
	assert.Contains(t, domain.MCPOnlyOAuthScopes(), domain.OAuthScopeOfflineAccess)
}

func TestLogin_MCPOnlyIsStoredBesideTheDefaultSession(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	defaultToken := f.secrets.GetAll()[ports.KeyOAuthAccessToken]

	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-a")
	result, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	assert.NotContains(t, f.client.AuthorizationCalls[1].Scopes, domain.OAuthScopeDashboardSession)
	require.NotNil(t, result.Credential)
	assert.Equal(t, principalA, result.Credential.Principal.ID)
	assert.Equal(t, domain.MCPResourceUS, result.Credential.Audience)

	stored := f.secrets.GetAll()
	assert.Equal(t, defaultToken, stored[ports.KeyOAuthAccessToken], "an MCP-only login never replaces the default session")
	assert.Equal(t, "mock-refresh-token", stored[ports.KeyOAuthRefreshToken])
	assert.Equal(t, "refresh-a", stored["oauth_mcp_"+result.Credential.ID+"_refresh_token"])
	assert.Empty(t, f.client.RevokeCalls, "nothing was replaced, so nothing is revoked")
	assert.NotContains(t, stored[ports.KeyOAuthMCPCredentials], "refresh-a", "the index carries no token")
}

func TestLogin_MCPOnlyRejectsARequestThatWouldGrantDashboardAccess(t *testing.T) {
	f := newFixture(t)
	opts := mcpOnlyLogin()
	opts.Scopes = domain.MCPOAuthScopes()
	_, err := f.service.Login(context.Background(), opts)
	require.ErrorContains(t, err, "cannot request dashboard.session")
	assert.Empty(t, f.client.AuthorizationCalls)

	opts = mcpOnlyLogin()
	opts.Resource = ""
	_, err = f.service.Login(context.Background(), opts)
	require.Error(t, err)
}

func TestLogin_TwoPrincipalsDoNotOverwriteEachOther(t *testing.T) {
	f := newFixture(t)
	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-a")
	a, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalB)}, "refresh-b")
	b, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1"}, "refresh-none")
	none, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	ids := map[string]bool{a.Credential.ID: true, b.Credential.ID: true, none.Credential.ID: true}
	assert.Len(t, ids, 3, "a different principal, or none, is a different authorization")

	records, err := f.service.ListMCPCredentials()
	require.NoError(t, err)
	assert.Len(t, records, 3)
	for id, refresh := range map[string]string{a.Credential.ID: "refresh-a", b.Credential.ID: "refresh-b", none.Credential.ID: "refresh-none"} {
		bound, err := f.service.ForMCPCredential(id)
		require.NoError(t, err)
		session, err := bound.Status()
		require.NoError(t, err)
		assert.Equal(t, refresh, session.Tokens.RefreshToken)
	}
	assert.Empty(t, f.client.RevokeCalls)
}

func TestLogin_SamePrincipalReplacesOnlyItsOwnCredential(t *testing.T) {
	f := newFixture(t)
	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-a1")
	first, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)
	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalB)}, "refresh-b")
	_, err = f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-a2")
	again, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	assert.Equal(t, first.Credential.ID, again.Credential.ID)
	assert.Equal(t, []string{"refresh-a1"}, f.client.RevokeCalls, "the replaced family is revoked, and only that one")
	records, err := f.service.ListMCPCredentials()
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestLogin_MCPOnlyRefusesATokenItCannotFileSafely(t *testing.T) {
	for name, claims := range map[string]tokenClaims{
		"dashboard.session granted": {aud: domain.MCPResourceUS, scope: "email.read dashboard.session", org: "org-1"},
		"principal in another region": {aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1",
			principal: &domain.OAuthTokenPrincipal{ID: principalA, Region: "eu"}},
		"malformed principal id": {aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1",
			principal: &domain.OAuthTokenPrincipal{ID: "../../etc", Region: "us"}},
		"another audience": {aud: domain.MCPResourceEU, scope: mcpOnlyScope, org: "org-1"},
		"no organization":  {aud: domain.MCPResourceUS, scope: mcpOnlyScope},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.issue(t, claims, "refresh-refused")
			_, err := f.service.Login(context.Background(), mcpOnlyLogin())
			require.Error(t, err)
			assert.Empty(t, f.secrets.GetAll(), "nothing is stored")
			assert.Equal(t, []string{"refresh-refused"}, f.client.RevokeCalls, "a token the CLI will not keep is revoked")
		})
	}
}

func TestLogin_DashboardLoginNeverStoresAPrincipalToken(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	before := f.secrets.GetAll()

	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-p")
	_, err = f.service.Login(context.Background(), mcpLogin())
	require.ErrorIs(t, err, ErrPrincipalNeedsMCPOnly)
	assert.Equal(t, before, f.secrets.GetAll(), "the dashboard-capable session is untouched")
	assert.Equal(t, []string{"refresh-p"}, f.client.RevokeCalls)
}

// devicePrincipalClient answers a device poll with a principal token.
type devicePrincipalClient struct {
	*deviceTestClient
	tokens *domain.OAuthTokens
}

func (c *devicePrincipalClient) PollDeviceTokens(context.Context, string, string, string) (*domain.OAuthTokens, error) {
	c.polls++
	return c.tokens, nil
}

func TestDeviceLogin_MCPOnlyIsStoredAsItsOwnCredential(t *testing.T) {
	f, c, opts, _ := deviceFixture(t)
	scope := mcpOnlyScope
	client := &devicePrincipalClient{deviceTestClient: c, tokens: &domain.OAuthTokens{
		AccessToken:  principalToken(t, tokenClaims{aud: domain.MCPResourceEU, scope: scope, org: "org-1", principal: &domain.OAuthTokenPrincipal{ID: principalA, Region: "eu"}}),
		RefreshToken: "device-refresh", Scope: scope, ExpiresAt: f.clock.Add(15 * time.Minute),
	}}
	f.service.client = client
	opts.Scopes, opts.Resource, opts.MCPOnly = domain.MCPOnlyOAuthScopes(), domain.MCPResourceEU, true

	result, err := f.service.Login(context.Background(), *opts)
	require.NoError(t, err)

	assert.NotContains(t, c.scopes, domain.OAuthScopeDashboardSession)
	assert.Equal(t, domain.MCPResourceEU, c.resource)
	require.NotNil(t, result.Credential)
	assert.Equal(t, "eu", result.Credential.Principal.Region)
	assert.NotContains(t, f.secrets.GetAll(), ports.KeyOAuthAccessToken, "the default session is not written")
}

func TestSelectMCPCredential(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.SelectMCPCredential("")
	require.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)

	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-a")
	a, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	selected, err := f.service.SelectMCPCredential("")
	require.NoError(t, err)
	assert.Equal(t, a.Credential.ID, selected.CredentialID(), "the only MCP-only credential is used when there is no default session")

	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalB)}, "refresh-b")
	_, err = f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)
	_, err = f.service.SelectMCPCredential("")
	require.ErrorIs(t, err, ErrMCPCredentialAmbiguous)

	selected, err = f.service.SelectMCPCredential(a.Credential.ID)
	require.NoError(t, err)
	assert.Equal(t, a.Credential.ID, selected.CredentialID())
	_, err = f.service.SelectMCPCredential("000000000000")
	require.ErrorIs(t, err, ErrMCPCredentialNotFound)

	f.client.ExchangeCodeFunc = nil
	_, err = f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	selected, err = f.service.SelectMCPCredential("")
	require.NoError(t, err)
	assert.Empty(t, selected.CredentialID(), "a default session keeps being the default")
}

func TestAccessToken_RefreshesOnlyTheSelectedCredential(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-a")
	a, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	f.advance(time.Hour)
	bound, err := f.service.ForMCPCredential(a.Credential.ID)
	require.NoError(t, err)
	_, err = bound.AccessToken(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []string{"refresh-a"}, f.client.RefreshCalls)
	assert.Equal(t, []string{domain.MCPResourceUS}, f.client.RefreshResources)
	stored := f.secrets.GetAll()
	assert.Equal(t, "mock-refresh-token-2", stored["oauth_mcp_"+a.Credential.ID+"_refresh_token"])
	assert.Equal(t, "mock-refresh-token", stored[ports.KeyOAuthRefreshToken], "the default session is not refreshed")
}

func TestRefresh_RetriesTemporarilyUnavailableWithTheSameRefreshToken(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	waits := []time.Duration{}
	f.service.waitRetry = func(_ context.Context, d time.Duration) error { waits = append(waits, d); return nil }
	answers := []error{
		&domain.OAuthError{Code: "temporarily_unavailable", StatusCode: 503, RetryAfter: 3 * time.Second},
		&domain.OAuthError{Code: "temporarily_unavailable", StatusCode: 503},
	}
	f.client.RefreshFunc = func(context.Context, string, string, string) (*domain.OAuthTokens, error) {
		if len(answers) > 0 {
			err := answers[0]
			answers = answers[1:]
			return nil, err
		}
		return &domain.OAuthTokens{AccessToken: "fresh", RefreshToken: "rotated", ExpiresAt: f.clock.Add(time.Hour)}, nil
	}

	f.advance(2 * time.Hour)
	token, err := f.service.AccessToken(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "fresh", token)
	assert.Equal(t, []string{"mock-refresh-token", "mock-refresh-token", "mock-refresh-token"}, f.client.RefreshCalls)
	assert.Equal(t, []time.Duration{3 * time.Second, 2 * time.Second}, waits, "Retry-After first, then backoff")
	assert.Equal(t, "rotated", f.secrets.GetAll()[ports.KeyOAuthRefreshToken])
}

func TestRefresh_KeepsTheSessionWhenTheServerStaysUnavailable(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	f.service.waitRetry = func(context.Context, time.Duration) error { return nil }
	f.client.RefreshFunc = func(context.Context, string, string, string) (*domain.OAuthTokens, error) {
		return nil, &domain.OAuthError{Code: "temporarily_unavailable", StatusCode: 503}
	}

	f.advance(2 * time.Hour)
	_, err = f.service.AccessToken(context.Background())
	require.ErrorContains(t, err, "your sign-in is kept")
	assert.Len(t, f.client.RefreshCalls, refreshAttempts)
	assert.Equal(t, "mock-refresh-token", f.secrets.GetAll()[ports.KeyOAuthRefreshToken], "a 503 never signs the user out")
}

func TestRefresh_DoesNotRetryAnotherError(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	f.service.waitRetry = func(context.Context, time.Duration) error {
		t.Fatal("an invalid_grant is final")
		return nil
	}
	f.client.RefreshFunc = func(context.Context, string, string, string) (*domain.OAuthTokens, error) {
		return nil, &domain.OAuthError{Code: "invalid_grant", StatusCode: 400}
	}
	f.advance(2 * time.Hour)
	_, err = f.service.AccessToken(context.Background())
	require.Error(t, err)
	assert.Len(t, f.client.RefreshCalls, 1)
}

func TestLogoutMCPCredential_RemovesOnlyThatCredential(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-a")
	a, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)
	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalB)}, "refresh-b")
	b, err := f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	require.NoError(t, f.service.LogoutMCPCredential(context.Background(), a.Credential.ID))

	assert.Equal(t, []string{"refresh-a"}, f.client.RevokeCalls)
	for key := range f.secrets.GetAll() {
		assert.False(t, strings.Contains(key, a.Credential.ID), "key %s outlived the logout", key)
	}
	records, err := f.service.ListMCPCredentials()
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, b.Credential.ID, records[0].ID)
	assert.Equal(t, "mock-refresh-token", f.secrets.GetAll()[ports.KeyOAuthRefreshToken], "the default session stays")
	require.ErrorIs(t, f.service.LogoutMCPCredential(context.Background(), a.Credential.ID), ErrMCPCredentialNotFound)
}

func TestClearSessionLocked_ClearsEveryCredential(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	f.issue(t, tokenClaims{aud: domain.MCPResourceUS, scope: mcpOnlyScope, org: "org-1", principal: usPrincipal(principalA)}, "refresh-a")
	_, err = f.service.Login(context.Background(), mcpOnlyLogin())
	require.NoError(t, err)

	require.NoError(t, ClearSessionLocked(context.Background(), f.secrets, f.lock))
	assert.Empty(t, f.secrets.GetAll())
}

func TestReadIndex_DropsAnEntryThatIsNotAnID(t *testing.T) {
	f := newFixture(t)
	require.NoError(t, f.secrets.Set(ports.KeyOAuthMCPCredentials, `[{"id":"../api_key"},{"id":"0123456789ab"}]`))
	records, err := f.service.ListMCPCredentials()
	require.NoError(t, err)
	require.Len(t, records, 1)
	assert.Equal(t, "0123456789ab", records[0].ID)

	require.NoError(t, f.secrets.Set(ports.KeyOAuthMCPCredentials, `not json`))
	_, err = f.service.ListMCPCredentials()
	require.True(t, err != nil && !errors.Is(err, domain.ErrOAuthNotLoggedIn))
}
