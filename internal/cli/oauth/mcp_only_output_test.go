//go:build !integration

package oauth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nylas/cli/internal/app/oauthlogin"
	"github.com/nylas/cli/internal/cli/testutil"
	"github.com/nylas/cli/internal/domain"
)

func TestLoginCmd_MCPOnlyJSONNeverExchangesForADashboardSession(t *testing.T) {
	// The structured form must keep the text form's promise: a credential
	// that may be narrowed by a principal never becomes a dashboard session.
	svc := &fakeService{loginResult: &oauthlogin.LoginResult{
		Issuer: "i", ClientID: "c", Resource: domain.MCPResourceUS, HasRefresh: true,
		Credential: &oauthlogin.MCPCredentialRecord{
			ID: "0123456789ab", Audience: domain.MCPResourceUS, Org: "org-1",
			Principal: &domain.OAuthTokenPrincipal{ID: "V1StGXR8_Z5jdHi6B-myT", Region: "us"},
		},
	}}
	dash := withService(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newLoginCmd(), "--mcp-only", "--json")
	require.NoError(t, err)

	assert.Nil(t, dash.exchangedWith, "a restricted credential is never exchanged")
	var report loginReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &report), "stdout must be pure JSON")
	assert.False(t, report.Dashboard.SignedIn)
	require.NotNil(t, report.MCPCredential)
	assert.Equal(t, "0123456789ab", report.MCPCredential.ID)
	require.NotNil(t, report.MCPCredential.Principal)
	assert.Equal(t, "V1StGXR8_Z5jdHi6B-myT", report.MCPCredential.Principal.ID)
}

func TestStatusCmd_JSONListsMCPOnlyCredentialsWithoutTokens(t *testing.T) {
	session := principalSession(t)
	svc := &credentialsFake{
		fakeService: &fakeService{statusErr: domain.ErrOAuthNotLoggedIn},
		records:     []oauthlogin.MCPCredentialRecord{{ID: "0123456789ab"}, {ID: "ba9876543210"}},
		bound:       map[string]*fakeService{"0123456789ab": {session: session}},
	}
	withCredentials(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd(), "--json")
	require.NoError(t, err, "MCP-only credentials are a session to report even without a default one")

	assert.NotContains(t, stdout, session.Tokens.AccessToken)
	assert.NotContains(t, stdout, "rt-secret")
	var report statusReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	assert.False(t, report.LoggedIn, "there is no default session")
	require.Len(t, report.MCPCredentials, 2)

	usable := report.MCPCredentials[0]
	require.NotNil(t, usable.Session)
	assert.Equal(t, "MCP only, limited to an IAM principal (no dashboard access)", usable.Session.Access)
	require.NotNil(t, usable.Session.Claims)
	require.NotNil(t, usable.Session.Claims.Principal)
	assert.Equal(t, "V1StGXR8_Z5jdHi6B-myT", usable.Session.Claims.Principal.ID)
	assert.Equal(t, "org-1", usable.Session.Claims.Org)

	broken := report.MCPCredentials[1]
	assert.Nil(t, broken.Session)
	assert.NotEmpty(t, broken.Error, "an unreadable credential is reported, not dropped")
}

func TestStatusCmd_JSONCredentialShowsOnlyThatCredential(t *testing.T) {
	svc := &credentialsFake{
		fakeService: &fakeService{session: loggedInSession()},
		records:     []oauthlogin.MCPCredentialRecord{{ID: "0123456789ab"}},
		bound:       map[string]*fakeService{"0123456789ab": {session: principalSession(t)}},
	}
	withCredentials(t, svc)

	stdout, _, err := testutil.ExecuteSubCommand(newStatusCmd(), "--json", "--credential", "0123456789ab")
	require.NoError(t, err)

	var report statusReport
	require.NoError(t, json.Unmarshal([]byte(stdout), &report))
	assert.True(t, report.LoggedIn)
	assert.Equal(t, domain.MCPResourceUS, report.Resource)
	assert.Empty(t, report.MCPCredentials)

	_, _, err = testutil.ExecuteSubCommand(newStatusCmd(), "--json", "--credential", "000000000000")
	require.ErrorIs(t, err, oauthlogin.ErrMCPCredentialNotFound)
}
