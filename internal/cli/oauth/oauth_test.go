//go:build !integration

package oauth

import (
	"context"
	"errors"
	"testing"
	"time"

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
	loginScopes []string

	session   *oauthlogin.Session
	statusErr error

	accessToken string
	accessErr   error

	userInfo *domain.OAuthUserInfo
	userErr  error

	logoutCalled bool
	logoutErr    error
}

func (f *fakeService) Login(_ context.Context, scopes []string) (*oauthlogin.LoginResult, error) {
	f.loginScopes = scopes
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

// withService swaps the command factory for the duration of one test.
func withService(t *testing.T, svc *fakeService) {
	t.Helper()
	original := createLoginServiceFn
	createLoginServiceFn = func() (loginService, error) { return svc, nil }
	t.Cleanup(func() { createLoginServiceFn = original })
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

	assert.Equal(t, []string{"openid", "email"}, svc.loginScopes)
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
