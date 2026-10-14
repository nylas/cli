//go:build !integration

package oauth

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashboardapp "github.com/nylas/cli/internal/app/dashboard"
	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/dashboard"
	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
)

// secrets are the token values in loggedInSession; structured output must
// never contain any of them.
var sessionSecrets = []string{"at-1", "rt-1", "idt-1"}

func assertNoTokens(t *testing.T, out string) {
	t.Helper()
	for _, secret := range sessionSecrets {
		assert.NotContains(t, out, secret, "structured output must not leak a token")
	}
}

func TestStatusCmd_JSONIsAStructuredReportWithoutTokens(t *testing.T) {
	session := loggedInSession()
	session.ServerURL = "https://dashboard.example.test"
	session.Resource = "https://mcp.us.nylas.com"
	withService(t, &fakeService{session: session})

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd(), "--json")
	require.NoError(t, err)

	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &report), "stdout must be pure JSON: %q", stdout)
	assert.Equal(t, true, report["logged_in"])
	assert.Equal(t, false, report["expired"])
	assert.Equal(t, "https://dashboard.example.test", report["server"])
	assert.Equal(t, "https://auth.example.test", report["issuer"])
	assert.Equal(t, "client-1", report["client_id"])
	assert.Equal(t, "https://mcp.us.nylas.com", report["resource"])
	assert.Equal(t, "openid email offline_access", report["scopes"])
	assert.Equal(t, true, report["has_refresh_token"])
	require.Contains(t, report, "expires_at")
	_, err = time.Parse(time.RFC3339, report["expires_at"].(string))
	assert.NoError(t, err, "expires_at is a timestamp")
	assertNoTokens(t, stdout)
}

func TestStatusCmd_JSONCarriesDecodedClaimsAsUnverified(t *testing.T) {
	session := loggedInSession()
	session.Tokens.AccessToken = jwtWithClaims(t, map[string]any{
		"aud":    "https://mcp.us.nylas.com",
		"scope":  "email.read",
		"grants": []map[string]string{{"id": "grant-1", "application_id": "app-1"}},
	})
	withService(t, &fakeService{session: session})

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd(), "--json")
	require.NoError(t, err)

	var report statusReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	require.NotNil(t, report.Claims)
	assert.False(t, report.Claims.Verified, "claims are decoded locally, never verified")
	assert.Equal(t, []string{"https://mcp.us.nylas.com"}, report.Claims.Audience)
	assert.Equal(t, []claimGrantEntry{{ID: "grant-1", ApplicationID: "app-1"}}, report.Claims.Grants)
	assert.NotContains(t, stdout, session.Tokens.AccessToken)
}

func TestStatusCmd_JSONVerifyIncludesUserInfo(t *testing.T) {
	svc := &fakeService{
		session:  loggedInSession(),
		userInfo: &domain.OAuthUserInfo{Subject: "user-1", Email: "dev@example.test", EmailVerified: true},
	}
	withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd(), "--json", "--verify")
	require.NoError(t, err)

	var report statusReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	require.NotNil(t, report.UserInfo)
	assert.Equal(t, "user-1", report.UserInfo.Subject)
}

func TestStatusCmd_JSONNotLoggedInIsAnError(t *testing.T) {
	withService(t, &fakeService{statusErr: domain.ErrOAuthNotLoggedIn})

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd(), "--json")

	require.ErrorIs(t, err, domain.ErrOAuthNotLoggedIn)
	assert.NotContains(t, stdout, `"logged_in"`, "no report is written when there is no session")
}

func TestOAuthCmds_QuietSuppressesOutput(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T) (string, string, error)
	}{
		{"status", func(t *testing.T) (string, string, error) {
			withService(t, &fakeService{session: loggedInSession()})
			return testutil.ExecuteSubCommand(newStatusCmd(), "--quiet")
		}},
		{"login", func(t *testing.T) (string, string, error) {
			withService(t, &fakeService{loginResult: &oauthlogin.LoginResult{Issuer: "i", ClientID: "c"}})
			return testutil.ExecuteSubCommand(newLoginCmd(), "--quiet")
		}},
		{"logout", func(t *testing.T) (string, string, error) {
			withService(t, &fakeService{})
			return testutil.ExecuteSubCommand(newLogoutCmd(), "--quiet")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := tt.run(t)
			require.NoError(t, err)
			assert.Empty(t, strings.TrimSpace(stdout))
			assert.Empty(t, strings.TrimSpace(stderr))
		})
	}
}

func TestLoginCmd_JSONReportsSessionAndDashboard(t *testing.T) {
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{
		Issuer:        "https://auth.example.test",
		ClientID:      "client-1",
		Scope:         "openid email",
		Resource:      "https://mcp.us.nylas.com",
		ExpiresAt:     time.Now().Add(time.Hour),
		DroppedScopes: []string{"notetaker.read"},
	}}
	withService(t, svc)

	stdout, stderr, err := testutil.ExecuteSubCommand(newLoginCmd(), "--json")
	require.NoError(t, err)

	var report loginReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &report), "stdout must be pure JSON: %q", stdout)
	assert.True(t, report.LoggedIn)
	assert.Equal(t, "https://mcp.us.nylas.com", report.Resource)
	assert.Equal(t, []string{"notetaker.read"}, report.DroppedScopes)
	assert.False(t, report.HasRefresh)
	assert.True(t, report.Dashboard.SignedIn)
	assert.Equal(t, "org_1", report.Dashboard.OrgPublicID)
	assert.Contains(t, stderr, "Opening your browser", "the prompt moves to stderr, off the JSON stream")
}

func TestLoginCmd_JSONDashboardOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		exchangeErr error
		want        dashboardOutcome
	}{
		{"kept dashboard login", dashboardapp.ErrDashboardLoginSessionKept, dashboardOutcome{KeptDashboardLogin: true}},
		{"exchange failed", errors.New("no exchange"), dashboardOutcome{Error: "no exchange"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dash := withService(t, &fakeService{loginResult: &oauthlogin.LoginResult{Issuer: "i", ClientID: "c"}})
			dash.exchangeErr = tt.exchangeErr

			stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd(), "--json")
			require.NoError(t, err, "the OAuth login itself succeeded")

			var report loginReport
			require.NoError(t, json.Unmarshal([]byte(stdout), &report))
			assert.Equal(t, tt.want, report.Dashboard)
		})
	}
}

func TestLogoutCmd_JSONReport(t *testing.T) {
	svc := &fakeService{}
	dash := withService(t, svc)
	dash.clearErr = errors.New("keyring locked")

	stdout, _, err := testutil.ExecuteSubCommand(newLogoutCmd(), "--json")
	require.NoError(t, err)

	var report logoutReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &report), "stdout must be pure JSON: %q", stdout)
	assert.True(t, report.LoggedOut)
	assert.Equal(t, "keyring locked", report.DashboardError)
	assert.True(t, svc.logoutCalled)
}

func TestWireDashboard_SetsTheDashboardHooks(t *testing.T) {
	originalSource, originalRelogin := dashboard.OAuthTokenSource, dashboard.OAuthRelogin
	t.Cleanup(func() {
		dashboard.OAuthTokenSource, dashboard.OAuthRelogin = originalSource, originalRelogin
	})

	assert.Nil(t, dashboard.OAuthTokenSource, "importing the oauth package must not wire the dashboard")
	assert.Nil(t, dashboard.OAuthRelogin, "importing the oauth package must not wire the dashboard")

	WireDashboard()

	assert.NotNil(t, dashboard.OAuthTokenSource)
	assert.NotNil(t, dashboard.OAuthRelogin)
}
