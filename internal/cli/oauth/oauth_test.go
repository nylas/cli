//go:build !integration

package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeService struct {
	loginResult *oauthlogin.LoginResult
	loginErr    error
	loginOpts   *oauthlogin.LoginOptions

	session   *oauthlogin.Session
	statusErr error

	accessToken string
	accessErr   error

	userInfo *domain.OAuthUserInfo
	userErr  error

	logoutCalled bool
	logoutErr    error
}

func (f *fakeService) Login(_ context.Context, opts oauthlogin.LoginOptions) (*oauthlogin.LoginResult, error) {
	f.loginOpts = &opts
	return f.loginResult, f.loginErr
}

func (f *fakeService) Status() (*oauthlogin.Session, error) { return f.session, f.statusErr }

func (f *fakeService) AccessToken(context.Context) (string, error) {
	return f.accessToken, f.accessErr
}

func (f *fakeService) UserInfo(context.Context) (*domain.OAuthUserInfo, error) {
	return f.userInfo, f.userErr
}

func (f *fakeService) Logout(context.Context) error {
	f.logoutCalled = true
	return f.logoutErr
}

// fakeDashboard records the dashboard session calls login and logout make.
type fakeDashboard struct {
	exchangeErr   error
	exchangedWith dashboardapp.OAuthAccessTokens
	cleared       bool
	clearErr      error
}

// withService swaps the command factory for the duration of one test, and
// the dashboard session hooks with it so no test reaches the real keyring.
func withService(t *testing.T, svc *fakeService) *fakeDashboard {
	t.Helper()
	dash := &fakeDashboard{}

	originalLogin := createLoginServiceFn
	originalExchange := exchangeDashboardSessionFn
	originalClear := clearDashboardSessionFn
	createLoginServiceFn = func() (loginService, error) { return svc, nil }
	exchangeDashboardSessionFn = func(_ context.Context, tokens dashboardapp.OAuthAccessTokens) (*domain.DashboardOAuthExchangeResponse, error) {
		dash.exchangedWith = tokens
		if dash.exchangeErr != nil {
			return nil, dash.exchangeErr
		}
		return &domain.DashboardOAuthExchangeResponse{OrgPublicID: "org_1"}, nil
	}
	clearDashboardSessionFn = func(context.Context) error {
		dash.cleared = true
		return dash.clearErr
	}
	t.Cleanup(func() {
		createLoginServiceFn = originalLogin
		exchangeDashboardSessionFn = originalExchange
		clearDashboardSessionFn = originalClear
	})
	return dash
}

func loggedInSession() *oauthlogin.Session {
	return &oauthlogin.Session{
		Issuer:   "https://auth.example.test",
		ClientID: "client-1",
		Tokens: domain.OAuthTokens{
			AccessToken:  "at-1",
			RefreshToken: "rt-1",
			IDToken:      "idt-1",
			Scope:        "openid email offline_access",
			ExpiresAt:    time.Now().Add(time.Hour),
		},
	}
}

func TestOAuthCmd_HasExpectedSubcommands(t *testing.T) {
	cmd := NewOAuthCmd()

	names := []string{}
	for _, sub := range cmd.Commands() {
		names = append(names, sub.Name())
	}

	assert.ElementsMatch(t, []string{"login", "status", "token", "logout"}, names)
}

func TestLoginCmd_ReportsSession(t *testing.T) {
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{
		Issuer:     "https://auth.example.test",
		ClientID:   "client-1",
		Scope:      "openid email offline_access",
		ExpiresAt:  time.Now().Add(time.Hour),
		HasRefresh: true,
	}}
	withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd())
	require.NoError(t, err)

	assert.Contains(t, stdout, "Logged in")
	assert.Contains(t, stdout, "https://auth.example.test")
	assert.Contains(t, stdout, "client-1")
}

func TestLoginCmd_WarnsWhenNoRefreshTokenIssued(t *testing.T) {
	// Without a refresh token the session dies in an hour, and the user
	// should know why before they hit it.
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{
		Issuer:     "https://auth.example.test",
		ClientID:   "client-1",
		Scope:      "openid email",
		HasRefresh: false,
	}}
	withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd())
	require.NoError(t, err)

	assert.Contains(t, stdout, "offline_access")
}

func TestLoginCmd_PassesScopeFlag(t *testing.T) {
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{Issuer: "i", ClientID: "c"}}
	withService(t, svc)

	_, _, err := testutil.ExecuteSubCommand(newLoginCmd(), "--scope", "openid,email")
	require.NoError(t, err)

	require.NotNil(t, svc.loginOpts)
	assert.Equal(t, []string{"openid", "email"}, svc.loginOpts.Scopes)
	assert.Empty(t, svc.loginOpts.Resource, "a plain login names no resource server")
}

func TestLoginCmd_SurfacesFailure(t *testing.T) {
	withService(t, &fakeService{loginErr: domain.ErrAuthTimeout})

	_, _, err := testutil.ExecuteSubCommand(newLoginCmd())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrAuthTimeout)
}

func TestStatusCmd_ShowsStoredSession(t *testing.T) {
	withService(t, &fakeService{session: loggedInSession()})

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd())
	require.NoError(t, err)

	assert.Contains(t, stdout, "Logged in")
	assert.Contains(t, stdout, "https://auth.example.test")
	assert.Contains(t, stdout, "Refresh:    present")
	assert.NotContains(t, stdout, "at-1", "the access token itself must not be printed")
}

func TestStatusCmd_FlagsExpiredToken(t *testing.T) {
	session := loggedInSession()
	session.Tokens.ExpiresAt = time.Now().Add(-time.Hour)
	withService(t, &fakeService{session: session})

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd())
	require.NoError(t, err)

	assert.Contains(t, stdout, "expired")
}

func TestStatusCmd_NotLoggedIn(t *testing.T) {
	withService(t, &fakeService{statusErr: domain.ErrOAuthNotLoggedIn})

	_, _, err := testutil.ExecuteSubCommand(newStatusCmd())

	require.Error(t, err)
	var cliErr *common.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Contains(t, cliErr.Message, "nylas oauth login")
}

func TestStatusCmd_VerifyCallsUserInfo(t *testing.T) {
	svc := &fakeService{
		session:  loggedInSession(),
		userInfo: &domain.OAuthUserInfo{Subject: "user-1", Email: "dev@example.test", EmailVerified: true},
	}
	withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd(), "--verify")
	require.NoError(t, err)

	assert.Contains(t, stdout, "user-1")
	assert.Contains(t, stdout, "dev@example.test")
}

func TestStatusCmd_WithoutVerifyDoesNotCallUserInfo(t *testing.T) {
	svc := &fakeService{session: loggedInSession(), userErr: errors.New("should not be called")}
	withService(t, svc)

	_, _, err := testutil.ExecuteSubCommand(newStatusCmd())

	require.NoError(t, err)
}

func TestTokenCmd_PrintsBareToken(t *testing.T) {
	// The documented use is command substitution into a curl header, so the
	// output has to be the token and nothing else.
	withService(t, &fakeService{accessToken: "at-42"})

	stdout, _, err := testutil.ExecuteSubCommand(newTokenCmd())
	require.NoError(t, err)

	assert.Equal(t, "at-42\n", stdout)
}

func TestTokenCmd_NotLoggedIn(t *testing.T) {
	withService(t, &fakeService{accessErr: domain.ErrOAuthNotLoggedIn})

	_, _, err := testutil.ExecuteSubCommand(newTokenCmd())

	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)
}

func TestLogoutCmd(t *testing.T) {
	svc := &fakeService{}
	withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newLogoutCmd())
	require.NoError(t, err)

	assert.True(t, svc.logoutCalled)
	assert.Contains(t, stdout, "Logged out")
}

func TestLogoutCmd_SurfacesRevocationFailure(t *testing.T) {
	withService(t, &fakeService{logoutErr: errors.New("server unreachable")})

	_, _, err := testutil.ExecuteSubCommand(newLogoutCmd())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "server unreachable")
}

// jwtWithClaims builds an unsigned JWT; status only decodes, never verifies.
func jwtWithClaims(t *testing.T, claims map[string]any) string {
	t.Helper()
	body, err := json.Marshal(claims)
	require.NoError(t, err)
	return "eyJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(body) + ".sig"
}

func TestStatusCmd_ShowsDecodedTokenClaims(t *testing.T) {
	session := loggedInSession()
	session.Tokens.AccessToken = jwtWithClaims(t, map[string]any{
		"aud":    "https://mcp.us.nylas.com",
		"scope":  "email.read offline_access",
		"exp":    time.Now().Add(15 * time.Minute).Unix(),
		"grants": []map[string]string{{"id": "grant-1", "application_id": "app-1"}},
	})
	withService(t, &fakeService{session: session})

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd())
	require.NoError(t, err)

	assert.Contains(t, stdout, "NOT verified", "the output must not imply the claims were checked")
	assert.Contains(t, stdout, "Audience: https://mcp.us.nylas.com")
	assert.Contains(t, stdout, "Scopes:   email.read offline_access")
	assert.Contains(t, stdout, "grant-1 (application app-1)")
	assert.NotContains(t, stdout, session.Tokens.AccessToken, "the token itself must never be printed")
}

func TestStatusCmd_OpaqueTokenIsReportedNotFatal(t *testing.T) {
	withService(t, &fakeService{session: loggedInSession()})

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd())
	require.NoError(t, err)

	assert.Contains(t, stdout, "unavailable")
	assert.NotContains(t, stdout, "at-1")
}

func TestStatusCmd_HelpSaysClaimsAreNotVerified(t *testing.T) {
	assert.Contains(t, newStatusCmd().Long, "NOT VERIFIED")
}

func TestResolveClientID(t *testing.T) {
	t.Run("defaults to the static client", func(t *testing.T) {
		t.Setenv(clientIDEnv, "")
		id, err := resolveClientID()
		require.NoError(t, err)
		assert.Equal(t, domain.DefaultOAuthClientID, id)
	})
	t.Run("honours a valid override", func(t *testing.T) {
		t.Setenv(clientIDEnv, "dev-client-1")
		id, err := resolveClientID()
		require.NoError(t, err)
		assert.Equal(t, "dev-client-1", id)
	})
	t.Run("rejects an invalid override instead of falling back", func(t *testing.T) {
		t.Setenv(clientIDEnv, "bad id&x=1")
		_, err := resolveClientID()
		require.ErrorIs(t, err, domain.ErrOAuthInvalidClientID)
	})
}

func TestLoginOptions_MCPPreset(t *testing.T) {
	for region, resource := range map[string]string{"": domain.MCPResourceUS, "us": domain.MCPResourceUS, "eu": domain.MCPResourceEU} {
		opts, err := loginOptions("mcp", nil, region)
		require.NoError(t, err, region)
		assert.Equal(t, resource, opts.Resource, "the resource follows the configured region %q", region)
		assert.Equal(t, domain.MCPOAuthScopes(), opts.Scopes)
		assert.True(t, opts.DropUnsupportedScopes)
	}
}

func TestLoginOptions_Rejections(t *testing.T) {
	_, err := loginOptions("mcp", []string{"email.read"}, "us")
	require.Error(t, err, "--for picks the scopes; a hand-picked list would be half an MCP login")

	_, err = loginOptions("calendar", nil, "us")
	require.Error(t, err)

	_, err = loginOptions("mcp", nil, "ap")
	require.ErrorIs(t, err, domain.ErrMCPResource, "an unknown region must not default to US")
}

func TestLoginCmd_ForMCPPassesThePreset(t *testing.T) {
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{
		Issuer: "i", ClientID: "c", Resource: domain.MCPResourceUS,
		DroppedScopes: []string{"notetaker.read"}, HasRefresh: true,
	}}
	withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd(), "--for", "mcp")
	require.NoError(t, err)

	require.NotNil(t, svc.loginOpts)
	assert.Contains(t, []string{domain.MCPResourceUS, domain.MCPResourceEU}, svc.loginOpts.Resource)
	assert.Contains(t, stdout, "Resource:")
	assert.Contains(t, stdout, "notetaker.read", "scopes the server did not offer are reported")
}

func TestLoginCmd_SignsInTheDashboardCommands(t *testing.T) {
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{Issuer: "i", ClientID: "c", HasRefresh: true}}
	dash := withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd())
	require.NoError(t, err)

	assert.Same(t, svc, dash.exchangedWith, "the dashboard session is exchanged from this login's tokens")
	assert.Contains(t, stdout, "signed in to organization org_1")
}

func TestLoginCmd_DashboardExchangeFailureIsOnlyAWarning(t *testing.T) {
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{Issuer: "i", ClientID: "c", HasRefresh: true}}
	dash := withService(t, svc)
	dash.exchangeErr = errors.New("server does not support the exchange")

	stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd())
	require.NoError(t, err, "the OAuth login itself succeeded")

	assert.Contains(t, stdout, "Logged in")
	assert.Contains(t, stdout, "Dashboard commands are not signed in")
}

func TestLogoutCmd_EndsTheDashboardSessionItCreated(t *testing.T) {
	svc := &fakeService{}
	dash := withService(t, svc)

	_, _, err := testutil.ExecuteSubCommand(newLogoutCmd())
	require.NoError(t, err)

	assert.True(t, dash.cleared)
	assert.True(t, svc.logoutCalled)
}

func TestLogoutCmd_StillRevokesWhenTheDashboardCannotBeEnded(t *testing.T) {
	svc := &fakeService{}
	dash := withService(t, svc)
	dash.clearErr = errors.New("keyring locked")

	stdout, _, err := testutil.ExecuteSubCommand(newLogoutCmd())
	require.NoError(t, err)

	assert.True(t, svc.logoutCalled)
	assert.Contains(t, stdout, "Could not end the dashboard session")
}
