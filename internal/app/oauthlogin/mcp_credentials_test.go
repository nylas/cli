//go:build !integration

package oauthlogin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mcpToken builds an unsigned access token carrying the contract claims.
func mcpToken(t *testing.T, aud string, grants ...domain.OAuthTokenGrant) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"sub":       "user-1",
		"aud":       aud,
		"scope":     "email.read offline_access",
		"client_id": domain.DefaultOAuthClientID,
		"exp":       time.Now().Add(15 * time.Minute).Unix(),
		"grants":    grants,
	})
	require.NoError(t, err)
	return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(body) + ".sig"
}

func mcpLogin() LoginOptions {
	return LoginOptions{Scopes: domain.MCPOAuthScopes(), Resource: domain.MCPResourceUS, DropUnsupportedScopes: true}
}

func TestLogin_SendsAndStoresTheResourceIndicator(t *testing.T) {
	f := newFixture(t)

	result, err := f.service.Login(context.Background(), mcpLogin())
	require.NoError(t, err)

	assert.Equal(t, domain.MCPResourceUS, result.Resource)
	assert.Equal(t, domain.MCPResourceUS, f.client.AuthorizationCalls[0].Resource)
	assert.Equal(t, domain.MCPResourceUS, f.client.ExchangeCalls[0].Resource)
	assert.Equal(t, domain.MCPResourceUS, f.secrets.GetAll()[ports.KeyOAuthResource])
}

func TestAccessToken_RefreshKeepsTheResourceIndicator(t *testing.T) {
	// A refresh without the resource would come back with no MCP audience,
	// and every request after the first fifteen minutes would be refused.
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), mcpLogin())
	require.NoError(t, err)
	f.advance(2 * time.Hour)

	_, err = f.service.AccessToken(context.Background())
	require.NoError(t, err)

	assert.Equal(t, []string{domain.MCPResourceUS}, f.client.RefreshResources)
	assert.Equal(t, domain.MCPResourceUS, f.secrets.GetAll()[ports.KeyOAuthResource])
}

func TestLogin_PlainLoginClearsAStaleResource(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), mcpLogin())
	require.NoError(t, err)

	_, err = f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)

	assert.NotContains(t, f.secrets.GetAll(), ports.KeyOAuthResource)
}

func TestLogin_RejectsAnUnknownResource(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), LoginOptions{Resource: "https://evil.example"})

	require.ErrorIs(t, err, domain.ErrMCPResource)
	assert.Empty(t, f.browser.openedURL)
}

func TestLogin_DropsScopesTheServerDoesNotOffer(t *testing.T) {
	f := newFixture(t)
	f.client.MetadataFunc = func(context.Context) (*domain.OAuthServerMetadata, error) {
		return &domain.OAuthServerMetadata{
			Issuer:                "https://auth.example.test",
			AuthorizationEndpoint: "https://auth.example.test/oauth/authorize",
			TokenEndpoint:         "https://auth.example.test/oauth/token",
			ScopesSupported:       []string{"openid", "email.read", "calendar.read", "grants.read", "offline_access"},
		}, nil
	}

	result, err := f.service.Login(context.Background(), mcpLogin())
	require.NoError(t, err)

	assert.Equal(t, []string{"email.read", "calendar.read", "grants.read", "offline_access"},
		f.client.AuthorizationCalls[0].Scopes)
	assert.ElementsMatch(t, []string{"email.send", "calendar.write", "contacts.read", "notetaker.read"}, result.DroppedScopes)
}

func TestLogin_KeepsScopesWhenFilteringIsOff(t *testing.T) {
	f := newFixture(t)
	f.client.MetadataFunc = func(context.Context) (*domain.OAuthServerMetadata, error) {
		return &domain.OAuthServerMetadata{
			Issuer: "i", AuthorizationEndpoint: "a", TokenEndpoint: "t",
			ScopesSupported: []string{"openid"},
		}, nil
	}

	_, err := f.service.Login(context.Background(), LoginOptions{Scopes: []string{"openid", "email.read"}})
	require.NoError(t, err)

	assert.Equal(t, []string{"openid", "email.read"}, f.client.AuthorizationCalls[0].Scopes,
		"an explicit --scope list is sent as given")
}

func TestRefreshAccessToken_RefreshesARejectedButUnexpiredToken(t *testing.T) {
	// A 401 on a token the clock says is still valid (revoked, or rotated
	// keys) must still refresh — expiry is not the only reason to.
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)

	token, err := f.service.RefreshAccessToken(context.Background(), "mock-access-token")
	require.NoError(t, err)

	assert.Equal(t, "mock-access-token-2", token)
	assert.Len(t, f.client.RefreshCalls, 1)
}

func TestRefreshAccessToken_UsesATokenAnotherProcessAlreadyReplaced(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)

	token, err := f.service.RefreshAccessToken(context.Background(), "some-older-token")
	require.NoError(t, err)

	assert.Equal(t, "mock-access-token", token, "the stored token is not the one that was refused")
	assert.Empty(t, f.client.RefreshCalls)
}

func TestMCPCredentials_RouteToTheTokenAudience(t *testing.T) {
	f := newFixture(t)
	grants := []domain.OAuthTokenGrant{{ID: "grant-1", ApplicationID: "app-1"}}
	f.client.ExchangeCodeFunc = func(context.Context, domain.OAuthCodeExchange) (*domain.OAuthTokens, error) {
		return &domain.OAuthTokens{
			AccessToken: mcpToken(t, domain.MCPResourceEU, grants...), TokenType: "Bearer",
			ExpiresAt: f.clock.Add(15 * time.Minute), RefreshToken: "rt",
		}, nil
	}
	_, err := f.service.Login(context.Background(), LoginOptions{Resource: domain.MCPResourceEU})
	require.NoError(t, err)

	cred, err := NewMCPCredentials(f.service).Credential(context.Background())
	require.NoError(t, err)

	assert.Equal(t, domain.MCPResourceEU, cred.Endpoint, "the audience decides the host, not the region")
	assert.True(t, cred.GrantScoped)
	assert.Equal(t, grants, cred.Grants)
	assert.True(t, cred.AllowsGrantHint("grant-1"))
	assert.False(t, cred.AllowsGrantHint("grant-2"))
}

func TestMCPCredentials_RefuseATokenForAnotherServer(t *testing.T) {
	f := newFixture(t)
	f.client.ExchangeCodeFunc = func(context.Context, domain.OAuthCodeExchange) (*domain.OAuthTokens, error) {
		return &domain.OAuthTokens{
			AccessToken: mcpToken(t, "https://attacker.example"), TokenType: "Bearer",
			ExpiresAt: f.clock.Add(15 * time.Minute),
		}, nil
	}
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)

	_, err = NewMCPCredentials(f.service).Credential(context.Background())

	require.ErrorIs(t, err, domain.ErrMCPResource)
}

func TestMCPCredentials_RenewRefreshesTheRejectedToken(t *testing.T) {
	f := newFixture(t)
	first := mcpToken(t, domain.MCPResourceUS)
	f.client.ExchangeCodeFunc = func(context.Context, domain.OAuthCodeExchange) (*domain.OAuthTokens, error) {
		return &domain.OAuthTokens{AccessToken: first, TokenType: "Bearer", ExpiresAt: f.clock.Add(15 * time.Minute), RefreshToken: "rt-1"}, nil
	}
	second := mcpToken(t, domain.MCPResourceUS, domain.OAuthTokenGrant{ID: "g"})
	f.client.RefreshFunc = func(context.Context, string, string, string) (*domain.OAuthTokens, error) {
		return &domain.OAuthTokens{AccessToken: second, TokenType: "Bearer", ExpiresAt: f.clock.Add(15 * time.Minute), RefreshToken: "rt-2"}, nil
	}
	_, err := f.service.Login(context.Background(), mcpLogin())
	require.NoError(t, err)

	creds := NewMCPCredentials(f.service)
	cred, err := creds.Credential(context.Background())
	require.NoError(t, err)
	renewed, err := creds.Renew(context.Background(), cred)
	require.NoError(t, err)

	assert.Equal(t, second, renewed.Token)
	assert.Equal(t, []string{"rt-1"}, f.client.RefreshCalls)
}

func TestMCPCredentials_NotLoggedIn(t *testing.T) {
	f := newFixture(t)

	_, err := NewMCPCredentials(f.service).Credential(context.Background())

	require.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)
}
