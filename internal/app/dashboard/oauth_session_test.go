package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
)

type fakeOAuthTokens struct {
	token string
	calls int
}

func (f *fakeOAuthTokens) AccessToken(context.Context) (string, error) {
	f.calls++
	return f.token, nil
}

// exchangingAccount answers an exchange with the next session tokens and
// records the access token it was given. Every other method is unset, so an
// unexpected call — a refresh the server would refuse — panics the test.
func exchangingAccount(userToken string, exchangedWith *string) *dashboardadapter.MockAccountClient {
	return &dashboardadapter.MockAccountClient{
		ExchangeOAuthTokenFn: func(_ context.Context, accessToken string) (*domain.DashboardOAuthExchangeResponse, error) {
			*exchangedWith = accessToken
			return &domain.DashboardOAuthExchangeResponse{
				DashboardAuthResponse: domain.DashboardAuthResponse{
					UserToken: userToken,
					OrgToken:  "org-" + userToken,
					User:      domain.DashboardUser{PublicID: "usr_1"},
				},
				OrgPublicID: "org_1",
				ExpiresAt:   time.Now().Add(15 * time.Minute),
			}, nil
		},
	}
}

func oauthSessionSecrets(expiresAt time.Time) *memSecretStore {
	secrets := newMemSecretStore()
	secrets.data[ports.KeyDashboardUserToken] = "old-user"
	secrets.data[ports.KeyDashboardOrgToken] = "old-org"
	secrets.data[ports.KeyDashboardAppID] = "app_1"
	secrets.data[ports.KeyDashboardSessionOrigin] = sessionOriginOAuth
	secrets.data[ports.KeyDashboardSessionExpiresAt] = expiresAt.UTC().Format(time.RFC3339)
	return secrets
}

func TestSessionRenewer_LoginStoresTheSessionAndResetsTheAppSelection(t *testing.T) {
	var exchangedWith string
	secrets := newMemSecretStore()
	secrets.data[ports.KeyDashboardAppID] = "app_from_a_previous_login"
	tokens := &fakeOAuthTokens{token: "at-1"}

	resp, err := NewSessionRenewer(exchangingAccount("new-user", &exchangedWith), secrets, tokens).Login(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "at-1", exchangedWith)
	assert.Equal(t, "org_1", resp.OrgPublicID)
	assert.Equal(t, "new-user", secrets.data[ports.KeyDashboardUserToken])
	assert.Equal(t, "org-new-user", secrets.data[ports.KeyDashboardOrgToken])
	assert.Equal(t, "org_1", secrets.data[ports.KeyDashboardOrgPublicID])
	assert.Equal(t, sessionOriginOAuth, secrets.data[ports.KeyDashboardSessionOrigin])
	assert.NotEmpty(t, secrets.data[ports.KeyDashboardSessionExpiresAt])
	assert.NotContains(t, secrets.data, ports.KeyDashboardAppID)
}

func TestSessionRenewer_EnsureFreshLeavesADashboardLoginSessionAlone(t *testing.T) {
	secrets := newMemSecretStore()
	secrets.data[ports.KeyDashboardUserToken] = "dashboard-login-token"
	tokens := &fakeOAuthTokens{token: "at-1"}

	err := NewSessionRenewer(&dashboardadapter.MockAccountClient{}, secrets, tokens).EnsureFresh(context.Background())
	require.NoError(t, err)

	assert.Zero(t, tokens.calls)
	assert.Equal(t, "dashboard-login-token", secrets.data[ports.KeyDashboardUserToken])
}

func TestSessionRenewer_EnsureFreshKeepsASessionThatIsNotAboutToExpire(t *testing.T) {
	secrets := oauthSessionSecrets(time.Now().Add(10 * time.Minute))
	tokens := &fakeOAuthTokens{token: "at-1"}

	err := NewSessionRenewer(&dashboardadapter.MockAccountClient{}, secrets, tokens).EnsureFresh(context.Background())
	require.NoError(t, err)

	assert.Zero(t, tokens.calls)
	assert.Equal(t, "old-user", secrets.data[ports.KeyDashboardUserToken])
}

func TestSessionRenewer_EnsureFreshReexchangesNearExpiryAndKeepsTheActiveApp(t *testing.T) {
	var exchangedWith string
	secrets := oauthSessionSecrets(time.Now().Add(30 * time.Second))
	tokens := &fakeOAuthTokens{token: "at-2"}

	err := NewSessionRenewer(exchangingAccount("renewed-user", &exchangedWith), secrets, tokens).EnsureFresh(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "at-2", exchangedWith)
	assert.Equal(t, "renewed-user", secrets.data[ports.KeyDashboardUserToken])
	assert.Equal(t, "app_1", secrets.data[ports.KeyDashboardAppID], "a renewal is not a new login")
}

func TestAppService_CreateAPIKeyRenewsAnExpiredOAuthSessionFirst(t *testing.T) {
	var exchangedWith, usedToken string
	secrets := oauthSessionSecrets(time.Now().Add(-time.Minute))
	renewer := NewSessionRenewer(exchangingAccount("renewed-user", &exchangedWith), secrets, &fakeOAuthTokens{token: "at-3"})
	gateway := &dashboardadapter.MockGatewayClient{
		CreateAPIKeyFn: func(_ context.Context, _, _, _ string, _ int, userToken, _ string) (*domain.GatewayCreatedAPIKey, error) {
			usedToken = userToken
			return &domain.GatewayCreatedAPIKey{}, nil
		},
	}

	_, err := NewAppService(gateway, secrets).WithSessionRenewer(renewer).CreateAPIKey(context.Background(), "app_1", "us", "My key", 0)
	require.NoError(t, err)

	assert.Equal(t, "at-3", exchangedWith)
	assert.Equal(t, "renewed-user", usedToken)
}

func TestAuthService_RefreshExchangesAnOAuthSessionInsteadOfRefreshingIt(t *testing.T) {
	var exchangedWith string
	secrets := oauthSessionSecrets(time.Now().Add(10 * time.Minute))
	account := exchangingAccount("exchanged-user", &exchangedWith)
	renewer := NewSessionRenewer(account, secrets, &fakeOAuthTokens{token: "at-4"})

	err := NewAuthService(account, secrets).WithSessionRenewer(renewer).Refresh(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "at-4", exchangedWith)
	assert.Equal(t, "exchanged-user", secrets.data[ports.KeyDashboardUserToken])
}

func TestAuthService_RefreshWithoutARenewerRefusesAnOAuthSession(t *testing.T) {
	secrets := oauthSessionSecrets(time.Now().Add(10 * time.Minute))

	err := NewAuthService(&dashboardadapter.MockAccountClient{}, secrets).Refresh(context.Background())

	assert.ErrorIs(t, err, errOAuthSessionNotRefreshable)
}

func TestAuthService_DashboardLoginClearsTheOAuthMarkers(t *testing.T) {
	secrets := oauthSessionSecrets(time.Now().Add(10 * time.Minute))
	account := &dashboardadapter.MockAccountClient{
		LoginFn: func(context.Context, string, string, string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error) {
			return &domain.DashboardAuthResponse{UserToken: "password-user", User: domain.DashboardUser{PublicID: "usr_1"}}, nil, nil
		},
	}

	_, _, err := NewAuthService(account, secrets).Login(context.Background(), "a@b.c", "pw", "")
	require.NoError(t, err)

	assert.NotContains(t, secrets.data, ports.KeyDashboardSessionOrigin)
	assert.NotContains(t, secrets.data, ports.KeyDashboardSessionExpiresAt)
}

func TestSessionRenewer_ClearIfOAuthEndsOnlyAnOAuthSession(t *testing.T) {
	loggedOut := false
	account := &dashboardadapter.MockAccountClient{
		LogoutFn: func(context.Context, string, string) error {
			loggedOut = true
			return nil
		},
	}

	dashboardLogin := newMemSecretStore()
	dashboardLogin.data[ports.KeyDashboardUserToken] = "dashboard-login-token"
	require.NoError(t, NewSessionRenewer(account, dashboardLogin, nil).ClearIfOAuth(context.Background()))
	assert.False(t, loggedOut)
	assert.Equal(t, "dashboard-login-token", dashboardLogin.data[ports.KeyDashboardUserToken])

	oauth := oauthSessionSecrets(time.Now().Add(10 * time.Minute))
	require.NoError(t, NewSessionRenewer(account, oauth, nil).ClearIfOAuth(context.Background()))
	assert.True(t, loggedOut)
	assert.NotContains(t, oauth.data, ports.KeyDashboardUserToken)
	assert.NotContains(t, oauth.data, ports.KeyDashboardSessionOrigin)
}
