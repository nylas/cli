//go:build !integration

package oauthlogin

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/filelock"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/oauth"
	"github.com/nylas/cli/internal/adapters/oauthas"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockBrowser struct {
	openedURL string
	err       error
}

func (m *mockBrowser) Open(url string) error {
	m.openedURL = url
	return m.err
}

type fixture struct {
	service *Service
	client  *oauthas.MockClient
	browser *mockBrowser
	secrets *keyring.MockSecretStore
	server  *oauth.MockServer
	lock    *filelock.Lock
	clock   time.Time
}

// advance moves the shared clock the service and the fake server both read,
// so a token minted before the jump is genuinely stale afterwards.
func (f *fixture) advance(d time.Duration) {
	f.clock = f.clock.Add(d)
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	f := &fixture{
		client:  &oauthas.MockClient{},
		browser: &mockBrowser{},
		secrets: keyring.NewMockSecretStore(),
		server:  oauth.NewMockServer("auth-code-1"),
		clock:   time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC),
	}
	f.server.RedirectURI = "http://127.0.0.1:8080/callback"

	now := func() time.Time { return f.clock }
	f.client.Now = now
	f.lock = filelock.New(filepath.Join(t.TempDir(), "oauth-session.lock"))
	f.service = NewService(domain.DefaultOAuthClientID, f.client, f.server, f.browser, f.secrets, f.lock)
	f.service.now = now

	return f
}

func TestLogin_StoresTokensAndReportsSession(t *testing.T) {
	f := newFixture(t)

	result, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, oauthas.MockIssuer, result.Issuer)
	assert.Equal(t, domain.DefaultOAuthClientID, result.ClientID)
	assert.True(t, result.HasRefresh)

	stored := f.secrets.GetAll()
	assert.Equal(t, "mock-access-token", stored[ports.KeyOAuthAccessToken])
	assert.Equal(t, "mock-refresh-token", stored[ports.KeyOAuthRefreshToken])
	assert.Equal(t, "mock-id-token", stored[ports.KeyOAuthIDToken])
	assert.Equal(t, oauthas.MockIssuer, stored[ports.KeyOAuthIssuer])
}

func TestLogin_UsesTheStaticPublicClient(t *testing.T) {
	// The server registers one public client for the CLI; every leg of the
	// flow must name it, and no secret may be sent because there is none.
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	require.Len(t, f.client.AuthorizationCalls, 1)
	require.Len(t, f.client.ExchangeCalls, 1)
	assert.Equal(t, domain.DefaultOAuthClientID, f.client.AuthorizationCalls[0].ClientID)
	assert.Equal(t, domain.DefaultOAuthClientID, f.client.ExchangeCalls[0].ClientID)
	assert.Empty(t, f.client.ExchangeCalls[0].ClientSecret)
}

func TestLogin_UsesConfiguredClientIDOverride(t *testing.T) {
	f := newFixture(t)
	f.service = NewService("dev-client-1", f.client, f.server, f.browser, f.secrets, f.lock)
	f.service.now = func() time.Time { return f.clock }

	result, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, "dev-client-1", result.ClientID)
	assert.Equal(t, "dev-client-1", f.client.ExchangeCalls[0].ClientID)
}

func TestLogin_RejectsInvalidClientIDBeforeOpeningBrowser(t *testing.T) {
	f := newFixture(t)
	f.service = NewService("bad client&id", f.client, f.server, f.browser, f.secrets, f.lock)

	_, err := f.service.Login(context.Background(), nil)

	require.ErrorIs(t, err, domain.ErrOAuthInvalidClientID)
	assert.Empty(t, f.browser.openedURL)
}

func TestLogin_FailsClosedOnUnregisteredRedirectURI(t *testing.T) {
	// Only http://127.0.0.1/callback and http://localhost/callback are
	// registered. Anything else would be refused by the server after the
	// user has signed in, so it must be refused before the browser opens.
	for _, uri := range []string{
		"http://127.0.0.1:8080/other",
		"https://127.0.0.1:8080/callback",
		"http://192.168.1.5:8080/callback",
		"http://127.0.0.1/callback",
	} {
		t.Run(uri, func(t *testing.T) {
			f := newFixture(t)
			f.server.RedirectURI = uri

			_, err := f.service.Login(context.Background(), nil)

			require.ErrorIs(t, err, domain.ErrOAuthRedirectURI)
			assert.Empty(t, f.browser.openedURL)
			assert.Empty(t, f.client.AuthorizationCalls)
		})
	}
}

func TestLogin_AdvertisesTheLoopbackIPRedirect(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, "http://127.0.0.1:8080/callback", f.client.AuthorizationCalls[0].RedirectURI)
}

func TestLogin_SendsRFCCompliantPKCEChallenge(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	require.Len(t, f.client.AuthorizationCalls, 1)
	require.Len(t, f.client.ExchangeCalls, 1)

	challenge := f.client.AuthorizationCalls[0].CodeChallenge
	verifier := f.client.ExchangeCalls[0].CodeVerifier

	// The verifier sent to the token endpoint must be the pre-image of the
	// challenge sent to the authorization endpoint, or the exchange fails.
	sum := sha256.Sum256([]byte(verifier))
	assert.Equal(t, base64.RawURLEncoding.EncodeToString(sum[:]), challenge)
	assert.Regexp(t, `^[A-Za-z0-9\-_]{43}$`, challenge)
}

func TestLogin_BindsCallbackStateToAuthorizationRequest(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	sentState := f.client.AuthorizationCalls[0].State
	assert.NotEmpty(t, sentState)
	assert.Equal(t, sentState, f.server.ExpectedState,
		"the callback server must reject any state but the one we sent")
}

func TestLogin_ExchangesAgainstTheSameRedirectURI(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	// The server compares the redirect_uri at the token endpoint against the
	// one in the authorization request; a mismatch is invalid_grant.
	assert.Equal(t, f.client.AuthorizationCalls[0].RedirectURI, f.client.ExchangeCalls[0].RedirectURI)
	assert.Equal(t, f.server.GetRedirectURI(), f.client.ExchangeCalls[0].RedirectURI)
}

func TestLogin_OpensBrowserAndStopsServer(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	assert.NotEmpty(t, f.browser.openedURL)
	assert.True(t, f.server.StartCalled)
	assert.True(t, f.server.StopCalled)
}

func TestLogin_RequestsOfflineAccessByDefault(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	assert.Contains(t, f.client.AuthorizationCalls[0].Scopes, domain.OAuthScopeOfflineAccess)
}

func TestLogin_IgnoresAndClearsLegacyRegisteredClientID(t *testing.T) {
	// Builds that used dynamic registration left a client id in the keyring.
	// It must not be used — it names a client the server no longer has — and
	// a fresh login should not leave it behind.
	f := newFixture(t)
	require.NoError(t, f.secrets.Set(legacyKeyOAuthClientID, "stale-dcr-client"))

	result, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, domain.DefaultOAuthClientID, result.ClientID)
	assert.NotContains(t, f.secrets.GetAll(), legacyKeyOAuthClientID)
}

func TestLogin_FailsWhenCallbackFails(t *testing.T) {
	f := newFixture(t)
	f.server.AuthCode = ""

	_, err := f.service.Login(context.Background(), nil)

	require.ErrorIs(t, err, domain.ErrAuthFailed)
	assert.Empty(t, f.secrets.GetAll()[ports.KeyOAuthAccessToken])
}

func TestAccessToken_ReturnsStoredTokenWhileValid(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	token, err := f.service.AccessToken(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "mock-access-token", token)
	assert.Empty(t, f.client.RefreshCalls)
}

func TestAccessToken_RefreshesWhenExpired(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	f.advance(2 * time.Hour)

	token, err := f.service.AccessToken(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "mock-access-token-2", token)
	assert.Equal(t, []string{"mock-refresh-token"}, f.client.RefreshCalls)
}

func TestAccessToken_PersistsRotatedRefreshToken(t *testing.T) {
	// The server rotates on every use and burns the family if an old token
	// reappears, so the new one must land in the store before it is needed.
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)
	f.advance(2 * time.Hour)

	_, err = f.service.AccessToken(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "mock-refresh-token-2", f.secrets.GetAll()[ports.KeyOAuthRefreshToken])
}

func TestAccessToken_DoesNotResurrectOldRefreshTokenWhenServerOmitsOne(t *testing.T) {
	// Carrying the previous refresh token forward would replay a token the
	// server has already consumed, which revokes the entire family.
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)
	f.advance(2 * time.Hour)
	f.client.RefreshFunc = func(context.Context, string, string) (*domain.OAuthTokens, error) {
		return &domain.OAuthTokens{AccessToken: "at-only", TokenType: "Bearer", ExpiresIn: 3600}, nil
	}

	_, err = f.service.AccessToken(context.Background())
	require.NoError(t, err)

	assert.Empty(t, f.secrets.GetAll()[ports.KeyOAuthRefreshToken])
}

func TestAccessToken_WithoutSession(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.AccessToken(context.Background())

	require.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)
}

func TestAccessToken_ExpiredWithNoRefreshToken(t *testing.T) {
	f := newFixture(t)
	f.client.ExchangeCodeFunc = func(context.Context, domain.OAuthCodeExchange) (*domain.OAuthTokens, error) {
		return &domain.OAuthTokens{AccessToken: "at-1", TokenType: "Bearer", ExpiresIn: 3600}, nil
	}
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)
	f.advance(2 * time.Hour)

	_, err = f.service.AccessToken(context.Background())

	require.ErrorIs(t, err, domain.ErrOAuthNoRefreshToken)
}

func TestStatus_ReportsStoredSession(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	session, err := f.service.Status()
	require.NoError(t, err)

	assert.Equal(t, oauthas.MockIssuer, session.Issuer)
	assert.Equal(t, domain.DefaultOAuthClientID, session.ClientID)
	assert.Equal(t, "openid email offline_access", session.Tokens.Scope)
	assert.False(t, session.Tokens.IsExpired(f.clock))
}

func TestLogout_RevokesRefreshTokenAndClearsEverything(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	require.NoError(t, f.service.Logout(context.Background()))

	// Revoking the refresh token takes the whole family with it; revoking
	// only the access token would leave the grant alive.
	assert.Equal(t, []string{"mock-refresh-token"}, f.client.RevokeCalls)
	for _, key := range sessionKeys {
		assert.NotContains(t, f.secrets.GetAll(), key)
	}
}

func TestLogout_ClearsLocalStateWhenRevocationFails(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)
	f.client.RevokeFunc = func(context.Context, string, string) error {
		return errors.New("server unreachable")
	}

	err = f.service.Logout(context.Background())

	require.Error(t, err, "the revocation failure is still reported")
	assert.NotContains(t, f.secrets.GetAll(), ports.KeyOAuthAccessToken,
		"local tokens must be gone regardless")
}

func TestLogout_WithoutSessionSucceeds(t *testing.T) {
	f := newFixture(t)

	require.NoError(t, f.service.Logout(context.Background()))
	assert.Empty(t, f.client.RevokeCalls)
}

func TestUserInfo_UsesCurrentAccessToken(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.Login(context.Background(), nil)
	require.NoError(t, err)

	var seen string
	f.client.UserInfoFunc = func(_ context.Context, accessToken string) (*domain.OAuthUserInfo, error) {
		seen = accessToken
		return &domain.OAuthUserInfo{Subject: "user-1", Email: "dev@example.test"}, nil
	}

	info, err := f.service.UserInfo(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "mock-access-token", seen)
	assert.Equal(t, "dev@example.test", info.Email)
}
