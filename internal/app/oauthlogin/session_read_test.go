//go:build !integration

package oauthlogin

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// afterFirstRead runs hook once, right after the first read of key returns.
type afterFirstRead struct {
	ports.SecretStore
	key  string
	hook func()
	done bool
}

func (s *afterFirstRead) Get(key string) (string, error) {
	value, err := s.SecretStore.Get(key)
	if key == s.key && !s.done {
		s.done = true
		s.hook()
	}
	return value, err
}

func TestAccessToken_ReadStraddlingALoginNeverPairsOneServersTokenWithAnother(t *testing.T) {
	// A process configured for server B reads the session while a login
	// against B replaces A's. It must never send A's token to B, which is
	// what reading A's token and then B's server name would do.
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	aToken := f.secrets.GetAll()[ports.KeyOAuthAccessToken]
	require.NotEmpty(t, aToken)

	const serverB = "https://b.example"
	f.client.ServerURLValue = serverB
	f.service.secrets = &afterFirstRead{SecretStore: f.secrets, key: ports.KeyOAuthAccessToken, hook: func() {
		require.NoError(t, f.secrets.Set(ports.KeyOAuthServerURL, serverB))
		require.NoError(t, f.secrets.Set(ports.KeyOAuthAccessToken, "b-token"))
		require.NoError(t, f.secrets.Set(ports.KeyOAuthExpiresAt, f.clock.Add(time.Hour).Format(time.RFC3339)))
	}}

	token, err := f.service.AccessToken(context.Background())

	if err == nil {
		assert.Equal(t, "b-token", token)
	}
	assert.NotEqual(t, aToken, token)
}

func TestMCPCredentials_ServesAFreshTokenFromMemory(t *testing.T) {
	// Every session read is several keyring reads, each an Argon2 key
	// derivation on the file store; a token with time left is served from
	// memory instead.
	f := newFixture(t)
	f.client.ExchangeCodeFunc = func(context.Context, domain.OAuthCodeExchange) (*domain.OAuthTokens, error) {
		return &domain.OAuthTokens{
			AccessToken: mcpToken(t, domain.MCPResourceUS), TokenType: "Bearer",
			ExpiresAt: f.clock.Add(15 * time.Minute), RefreshToken: "rt",
		}, nil
	}
	_, err := f.service.Login(context.Background(), mcpLogin())
	require.NoError(t, err)
	creds := NewMCPCredentials(f.service)
	first, err := creds.Credential(context.Background())
	require.NoError(t, err)

	require.NoError(t, ClearSession(f.secrets))
	cached, err := creds.Credential(context.Background())
	require.NoError(t, err)
	assert.Same(t, first, cached)

	// Past the token's own expiry the session is read again.
	f.advance(365 * 24 * time.Hour)
	_, err = creds.Credential(context.Background())
	require.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)
}

func TestLogin_RevokesTheRefreshTokenItReplaces(t *testing.T) {
	// Every repeated login (and every `orgs switch`) would otherwise leave
	// a live refresh-token family behind on the server.
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	replaced := f.secrets.GetAll()[ports.KeyOAuthRefreshToken]
	require.NotEmpty(t, replaced)

	_, err = f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)

	assert.Equal(t, []string{replaced}, f.client.RevokeCalls)
}

func TestLogin_DoesNotRevokeAnotherServersSession(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)
	f.client.ServerURLValue = "https://other.example"

	_, err = f.service.Login(context.Background(), LoginOptions{})
	require.NoError(t, err)

	assert.Empty(t, f.client.RevokeCalls)
}
