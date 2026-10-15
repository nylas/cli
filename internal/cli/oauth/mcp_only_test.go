//go:build !integration

package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// credentialsFake is a fakeService holding MCP-only credentials too.
type credentialsFake struct {
	*fakeService
	records   []oauthlogin.MCPCredentialRecord
	bound     map[string]*fakeService
	loggedOut []string
}

func (f *credentialsFake) ListMCPCredentials() ([]oauthlogin.MCPCredentialRecord, error) {
	return f.records, nil
}

func (f *credentialsFake) ForMCPCredential(id string) (loginService, error) {
	if svc, ok := f.bound[id]; ok {
		return svc, nil
	}
	return nil, oauthlogin.ErrMCPCredentialNotFound
}

func (f *credentialsFake) LogoutMCPCredential(_ context.Context, id string) error {
	f.loggedOut = append(f.loggedOut, id)
	return nil
}

func withCredentials(t *testing.T, svc *credentialsFake) *fakeDashboard {
	t.Helper()
	dash := withService(t, svc.fakeService)
	createLoginServiceFn = func() (loginService, error) { return svc, nil }
	return dash
}

func principalSession(t *testing.T) *oauthlogin.Session {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"sub": "user-1", "org": "org-1", "aud": domain.MCPResourceUS, "scope": "email.read offline_access",
		"exp": time.Now().Add(time.Hour).Unix(), "principal": map[string]string{"id": "V1StGXR8_Z5jdHi6B-myT", "region": "us"},
	})
	require.NoError(t, err)
	return &oauthlogin.Session{
		Issuer: "https://auth.example.test", ClientID: "client-1", Resource: domain.MCPResourceUS,
		Tokens: domain.OAuthTokens{
			AccessToken: "e30." + base64.RawURLEncoding.EncodeToString(body) + ".sig", RefreshToken: "rt-secret",
			Scope: "email.read offline_access", ExpiresAt: time.Now().Add(time.Hour),
		},
	}
}

func TestLoginOptionsFor_MCPOnly(t *testing.T) {
	opts, err := loginOptionsFor("", nil, "eu", true)
	require.NoError(t, err)
	assert.True(t, opts.MCPOnly)
	assert.Equal(t, domain.MCPResourceEU, opts.Resource)
	assert.NotContains(t, opts.Scopes, domain.OAuthScopeDashboardSession)

	opts, err = loginOptionsFor("mcp", nil, "", true)
	require.NoError(t, err)
	assert.True(t, opts.MCPOnly)

	_, err = loginOptionsFor("", []string{"openid"}, "", true)
	require.Error(t, err)
	_, err = loginOptionsFor("dashboard", nil, "", true)
	require.Error(t, err)

	opts, err = loginOptionsFor("mcp", nil, "", false)
	require.NoError(t, err)
	assert.False(t, opts.MCPOnly)
	assert.Contains(t, opts.Scopes, domain.OAuthScopeDashboardSession, "--for mcp still signs the dashboard in")
}

func TestLoginCmd_MCPOnlyNeverExchangesForADashboardSession(t *testing.T) {
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{
		Issuer: "i", ClientID: "c", Resource: domain.MCPResourceUS, HasRefresh: true,
		Credential: &oauthlogin.MCPCredentialRecord{
			ID: "0123456789ab", Audience: domain.MCPResourceUS, Org: "org-1",
			Principal: &domain.OAuthTokenPrincipal{ID: "V1StGXR8_Z5jdHi6B-myT", Region: "us"},
		},
	}}
	dash := withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd(), "--mcp-only")
	require.NoError(t, err)

	require.NotNil(t, svc.loginOpts)
	assert.True(t, svc.loginOpts.MCPOnly)
	assert.NotContains(t, svc.loginOpts.Scopes, domain.OAuthScopeDashboardSession)
	assert.Nil(t, dash.exchangedWith, "a restricted credential is never exchanged")
	assert.Contains(t, stdout, "MCP only")
	assert.Contains(t, stdout, "0123456789ab")
	assert.Contains(t, stdout, "V1StGXR8_Z5jdHi6B-myT (us)")
	assert.Contains(t, stdout, "nylas oauth login` for dashboard access")
	assert.Contains(t, stdout, "--credential 0123456789ab")
}

func TestLoginCmd_MCPOnlyWorksWithDevice(t *testing.T) {
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{Issuer: "i", ClientID: "c", HasRefresh: true,
		Credential: &oauthlogin.MCPCredentialRecord{ID: "0123456789ab"}}}
	withService(t, svc)

	_, _, err := testutil.ExecuteSubCommand(newLoginCmd(), "--mcp-only", "--device")
	require.NoError(t, err)
	assert.True(t, svc.loginOpts.MCPOnly)
	assert.True(t, svc.loginOpts.Device)
}

func TestStatusCmd_ListsMCPOnlyCredentialsWithoutTokens(t *testing.T) {
	session := principalSession(t)
	svc := &credentialsFake{
		fakeService: &fakeService{statusErr: domain.ErrOAuthNotLoggedIn},
		records:     []oauthlogin.MCPCredentialRecord{{ID: "0123456789ab"}},
		bound:       map[string]*fakeService{"0123456789ab": {session: session}},
	}
	withCredentials(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd())
	require.NoError(t, err)

	assert.Contains(t, stdout, "No default session")
	assert.Contains(t, stdout, "MCP credential 0123456789ab")
	assert.Contains(t, stdout, "MCP only, limited to an IAM principal")
	assert.Contains(t, stdout, "Principal: V1StGXR8_Z5jdHi6B-myT (us)")
	assert.Contains(t, stdout, "Audience: "+domain.MCPResourceUS)
	assert.NotContains(t, stdout, session.Tokens.AccessToken)
	assert.NotContains(t, stdout, "rt-secret")
}

func TestStatusCmd_NothingStoredIsNotLoggedIn(t *testing.T) {
	svc := &credentialsFake{fakeService: &fakeService{statusErr: domain.ErrOAuthNotLoggedIn}}
	withCredentials(t, svc)

	_, _, err := testutil.ExecuteSubCommand(newStatusCmd())
	require.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)
}

func TestStatusCmd_DefaultSessionSaysItIsDashboardCapable(t *testing.T) {
	session := loggedInSession()
	session.Tokens.Scope = "openid offline_access dashboard.session"
	withService(t, &fakeService{session: session})

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd())
	require.NoError(t, err)
	assert.Contains(t, stdout, "dashboard commands (your full role)")
}

func TestTokenCmd_CredentialPrintsThatCredentialsToken(t *testing.T) {
	svc := &credentialsFake{
		fakeService: &fakeService{accessToken: "default-token"},
		bound:       map[string]*fakeService{"0123456789ab": {accessToken: "mcp-token"}},
	}
	withCredentials(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newTokenCmd(), "--credential", "0123456789ab")
	require.NoError(t, err)
	assert.Equal(t, "mcp-token\n", stdout)

	_, _, err = testutil.ExecuteSubCommand(newTokenCmd(), "--credential", "ffffffffffff")
	require.ErrorIs(t, err, oauthlogin.ErrMCPCredentialNotFound)
}

func TestLogoutCmd_CredentialLeavesTheDefaultSessionAndDashboardAlone(t *testing.T) {
	svc := &credentialsFake{fakeService: &fakeService{}}
	dash := withCredentials(t, svc)

	_, _, err := testutil.ExecuteSubCommand(newLogoutCmd(), "--credential", "0123456789ab")
	require.NoError(t, err)
	assert.Equal(t, []string{"0123456789ab"}, svc.loggedOut)
	assert.False(t, svc.logoutCalled)
	assert.False(t, dash.cleared)
}

func TestLogoutCmd_AllLogsOutOfEveryCredential(t *testing.T) {
	svc := &credentialsFake{
		fakeService: &fakeService{},
		records:     []oauthlogin.MCPCredentialRecord{{ID: "0123456789ab"}, {ID: "ba9876543210"}},
	}
	dash := withCredentials(t, svc)

	_, _, err := testutil.ExecuteSubCommand(newLogoutCmd(), "--all")
	require.NoError(t, err)
	assert.True(t, svc.logoutCalled)
	assert.True(t, dash.cleared)
	assert.Equal(t, []string{"0123456789ab", "ba9876543210"}, svc.loggedOut)

	_, _, err = testutil.ExecuteSubCommand(newLogoutCmd(), "--all", "--credential", "0123456789ab")
	require.Error(t, err)
}

func TestLogoutCmd_WithoutFlagsKeepsMCPOnlyCredentials(t *testing.T) {
	svc := &credentialsFake{fakeService: &fakeService{}, records: []oauthlogin.MCPCredentialRecord{{ID: "0123456789ab"}}}
	withCredentials(t, svc)

	_, _, err := testutil.ExecuteSubCommand(newLogoutCmd())
	require.NoError(t, err)
	assert.True(t, svc.logoutCalled)
	assert.Empty(t, svc.loggedOut)
}
