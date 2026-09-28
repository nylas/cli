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

const (
	localServer = "account=http://localhost:3001 us=http://localhost:4000/graphql eu=http://localhost:4000/graphql"
	prodServer  = "account=https://dashboard-account.eu.nylas.com us=https://dashboard-api-gateway.us.nylas.com/graphql eu=https://dashboard-api-gateway.eu.nylas.com/graphql"
)

func dashboardSessionFor(server string) *memSecretStore {
	secrets := newMemSecretStore()
	secrets.data[ports.KeyDashboardUserToken] = "user-token"
	secrets.data[ports.KeyDashboardOrgToken] = "org-token"
	if server != "" {
		secrets.data[ports.KeyDashboardSessionServer] = server
	}
	return secrets
}

func TestAppService_RefusesASessionIssuedForOtherServers(t *testing.T) {
	gateway := &dashboardadapter.MockGatewayClient{
		CreateAPIKeyFn: func(context.Context, string, string, string, int, string, string) (*domain.GatewayCreatedAPIKey, error) {
			t.Fatal("a local session's token was sent to the production gateway")
			return nil, nil
		},
	}

	_, err := NewAppService(gateway, dashboardSessionFor(localServer)).
		WithServer(prodServer).
		CreateAPIKey(context.Background(), "app_1", "us", "My key", 0)

	require.ErrorIs(t, err, domain.ErrDashboardServerMismatch)
	assert.Contains(t, err.Error(), "http://localhost:3001")
}

func TestAppService_UsesASessionIssuedForTheseServers(t *testing.T) {
	var usedToken string
	gateway := &dashboardadapter.MockGatewayClient{
		CreateAPIKeyFn: func(_ context.Context, _, _, _ string, _ int, userToken, _ string) (*domain.GatewayCreatedAPIKey, error) {
			usedToken = userToken
			return &domain.GatewayCreatedAPIKey{}, nil
		},
	}

	_, err := NewAppService(gateway, dashboardSessionFor(prodServer)).
		WithServer(prodServer).
		CreateAPIKey(context.Background(), "app_1", "us", "My key", 0)

	require.NoError(t, err)
	assert.Equal(t, "user-token", usedToken)
}

func TestDomainService_RefusesASessionIssuedForOtherServers(t *testing.T) {
	account := &dashboardadapter.MockAccountClient{}

	_, err := NewDomainService(account, dashboardSessionFor(localServer)).
		WithServer(prodServer).
		ListDomains(context.Background(), 10, "")

	require.ErrorIs(t, err, domain.ErrDashboardServerMismatch)
}

func TestAuthService_AdoptsASessionStoredBeforeServersWereRecorded(t *testing.T) {
	secrets := dashboardSessionFor("")
	account := &dashboardadapter.MockAccountClient{
		GetCurrentSessionFn: func(context.Context, string, string) (*domain.DashboardSessionResponse, error) {
			return &domain.DashboardSessionResponse{}, nil
		},
	}

	_, err := NewAuthService(account, secrets).WithServer(prodServer).GetCurrentSession(context.Background())

	require.NoError(t, err)
	assert.Equal(t, prodServer, secrets.data[ports.KeyDashboardSessionServer])
}

func TestAuthService_RefreshRefusesASessionIssuedForOtherServers(t *testing.T) {
	account := &dashboardadapter.MockAccountClient{
		RefreshFn: func(context.Context, string, string) (*domain.DashboardRefreshResponse, error) {
			t.Fatal("a local refresh token was sent to the production account service")
			return nil, nil
		},
	}

	err := NewAuthService(account, dashboardSessionFor(localServer)).WithServer(prodServer).Refresh(context.Background())

	require.ErrorIs(t, err, domain.ErrDashboardServerMismatch)
}

func TestAuthService_LogoutClearsButDoesNotRevokeASessionForOtherServers(t *testing.T) {
	account := &dashboardadapter.MockAccountClient{
		LogoutFn: func(context.Context, string, string) error {
			t.Fatal("a local session's tokens were sent to the production account service")
			return nil
		},
	}
	secrets := dashboardSessionFor(localServer)

	err := NewAuthService(account, secrets).WithServer(prodServer).Logout(context.Background())

	require.ErrorIs(t, err, domain.ErrDashboardServerMismatch)
	assert.NotContains(t, secrets.data, ports.KeyDashboardUserToken)
	assert.NotContains(t, secrets.data, ports.KeyDashboardSessionServer)
}

func TestAuthService_LoginRecordsTheServer(t *testing.T) {
	secrets := dashboardSessionFor(localServer)
	account := &dashboardadapter.MockAccountClient{
		LoginFn: func(context.Context, string, string, string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error) {
			return &domain.DashboardAuthResponse{UserToken: "password-user", User: domain.DashboardUser{PublicID: "usr_1"}}, nil, nil
		},
	}

	_, _, err := NewAuthService(account, secrets).WithServer(prodServer).Login(context.Background(), "a@b.c", "pw", "")

	require.NoError(t, err)
	assert.Equal(t, prodServer, secrets.data[ports.KeyDashboardSessionServer])
}

func TestSessionRenewer_LoginRecordsTheServer(t *testing.T) {
	var exchangedWith string
	secrets := dashboardSessionFor(localServer)

	_, err := NewSessionRenewer(exchangingAccount("new-user", &exchangedWith), secrets, &fakeOAuthTokens{token: "at-1"}, newTestLock(t)).
		WithServer(prodServer).
		Login(context.Background())

	require.NoError(t, err)
	assert.Equal(t, prodServer, secrets.data[ports.KeyDashboardSessionServer])
}

func TestSessionRenewer_EnsureFreshReplacesAnExpiredSessionForOtherServers(t *testing.T) {
	// The OAuth session is checked against its own server before its token
	// is handed out, so a renewal yields a session for the configured servers.
	var exchangedWith string
	secrets := oauthSessionSecrets(time.Now().Add(-time.Minute))
	secrets.data[ports.KeyDashboardSessionServer] = localServer
	renewer := NewSessionRenewer(exchangingAccount("renewed-user", &exchangedWith), secrets, &fakeOAuthTokens{token: "at-2"}, newTestLock(t)).
		WithServer(prodServer)

	require.NoError(t, renewer.EnsureFresh(context.Background()))
	assert.Equal(t, prodServer, secrets.data[ports.KeyDashboardSessionServer])
}

func TestDashboardSessionServer_IgnoresTrailingSlashesAndSpace(t *testing.T) {
	assert.Equal(t,
		domain.DashboardSessionServer("https://a.example", "https://us.example/graphql", "https://eu.example/graphql"),
		domain.DashboardSessionServer(" https://a.example/ ", "https://us.example/graphql/", "https://eu.example/graphql"),
	)
	assert.NotEqual(t,
		domain.DashboardSessionServer("https://a.example", "https://us.example/graphql", "https://eu.example/graphql"),
		domain.DashboardSessionServer("https://a.example", "http://localhost:4000/graphql", "https://eu.example/graphql"),
	)
}

func TestAuthService_RefusesAnUnrecordedOAuthSessionFromAnotherAccountServer(t *testing.T) {
	// A session exchanged from a local OAuth login before servers were
	// recorded must not be adopted by production on the first command.
	secrets := oauthSessionSecrets(time.Now().Add(10 * time.Minute))
	secrets.data[ports.KeyOAuthServerURL] = "http://localhost:3001/"
	account := &dashboardadapter.MockAccountClient{
		GetCurrentSessionFn: func(context.Context, string, string) (*domain.DashboardSessionResponse, error) {
			t.Fatal("a local session was sent to the production account service")
			return nil, nil
		},
	}

	_, err := NewAuthService(account, secrets).WithServer(prodServer).GetCurrentSession(context.Background())

	require.ErrorIs(t, err, domain.ErrDashboardServerMismatch)
	assert.NotContains(t, secrets.data, ports.KeyDashboardSessionServer)
}

func TestAuthService_AdoptsAnUnrecordedOAuthSessionFromTheSameAccountServer(t *testing.T) {
	secrets := oauthSessionSecrets(time.Now().Add(10 * time.Minute))
	secrets.data[ports.KeyOAuthServerURL] = "https://dashboard-account.eu.nylas.com"
	account := &dashboardadapter.MockAccountClient{
		GetCurrentSessionFn: func(context.Context, string, string) (*domain.DashboardSessionResponse, error) {
			return &domain.DashboardSessionResponse{}, nil
		},
	}

	_, err := NewAuthService(account, secrets).WithServer(prodServer).GetCurrentSession(context.Background())

	require.NoError(t, err)
	assert.Equal(t, prodServer, secrets.data[ports.KeyDashboardSessionServer])
}

func TestSessionRenewer_ClearIfOAuthDropsAnUnrecordedSessionFromAnotherServerWithoutSendingIt(t *testing.T) {
	// `nylas oauth logout` ends the dashboard session first, while the OAuth
	// session's server is still stored, so an old local session is recognised
	// as local and cleared here rather than being sent to the new server.
	secrets := oauthSessionSecrets(time.Now().Add(10 * time.Minute))
	secrets.data[ports.KeyOAuthServerURL] = "http://localhost:3001"
	account := &dashboardadapter.MockAccountClient{
		LogoutFn: func(context.Context, string, string) error {
			t.Fatal("a local session's tokens were sent to another server")
			return nil
		},
	}

	err := NewSessionRenewer(account, secrets, nil, newTestLock(t)).WithServer(prodServer).ClearIfOAuth(context.Background())

	require.ErrorIs(t, err, domain.ErrDashboardServerMismatch)
	assert.NotContains(t, secrets.data, ports.KeyDashboardUserToken)
	assert.NotContains(t, secrets.data, ports.KeyDashboardSessionOrigin)
}
