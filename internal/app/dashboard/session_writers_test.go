package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionRenewer_RenewRejectedRefusesAReplacementForAnotherServer(t *testing.T) {
	// Another process stored a fresh session for server B. A process
	// configured for A, whose request was just refused, must not pick it up
	// and send B's tokens to A.
	secrets := oauthSessionSecrets(time.Now().Add(time.Hour))
	secrets.data[ports.KeyDashboardUserToken] = "b-user"
	secrets.data[ports.KeyDashboardOrgToken] = "b-org"
	secrets.data[ports.KeyDashboardSessionServer] = "https://b"
	var exchangedWith string

	user, org, err := NewSessionRenewer(exchangingAccount("exchanged", &exchangedWith), secrets, &fakeOAuthTokens{token: "at"}, newTestLock(t)).
		WithServer("https://a").
		renewRejected(context.Background(), "a-user")

	require.ErrorIs(t, err, domain.ErrDashboardServerMismatch)
	assert.Empty(t, user)
	assert.Empty(t, org)
	assert.Empty(t, exchangedWith, "nothing is exchanged over another server's session")
}

func TestSessionRenewer_RenewRejectedReusesAReplacementForThisServer(t *testing.T) {
	secrets := oauthSessionSecrets(time.Now().Add(time.Hour))
	secrets.data[ports.KeyDashboardUserToken] = "replacement"
	secrets.data[ports.KeyDashboardOrgToken] = "replacement-org"
	secrets.data[ports.KeyDashboardSessionServer] = "https://a"
	var exchangedWith string

	user, org, err := NewSessionRenewer(exchangingAccount("exchanged", &exchangedWith), secrets, &fakeOAuthTokens{token: "at"}, newTestLock(t)).
		WithServer("https://a").
		renewRejected(context.Background(), "rejected")

	require.NoError(t, err)
	assert.Equal(t, "replacement", user)
	assert.Equal(t, "replacement-org", org)
	assert.Empty(t, exchangedWith)
}

func TestSessionRenewer_FailedLoginExchangeDropsThePreviousOAuthSession(t *testing.T) {
	// `oauth login` (or `orgs switch`) has already replaced the OAuth
	// session. Keeping the dashboard session exchanged from the old one would
	// leave every dashboard command acting for the old organization.
	secrets := oauthSessionSecrets(time.Now().Add(time.Hour))
	account := &dashboardadapter.MockAccountClient{
		ExchangeOAuthTokenFn: func(context.Context, string) (*domain.DashboardOAuthExchangeResponse, error) {
			return nil, errors.New("exchange timed out")
		},
		LogoutFn: func(context.Context, string, string) error { return nil },
	}

	_, err := NewSessionRenewer(account, secrets, &fakeOAuthTokens{token: "at"}, newTestLock(t)).Login(context.Background())

	require.Error(t, err)
	assert.NotContains(t, secrets.data, ports.KeyDashboardUserToken)
	assert.NotContains(t, secrets.data, ports.KeyDashboardOrgToken)
}

func TestSessionRenewer_ExchangeWithoutASessionStoresNothing(t *testing.T) {
	secrets := newMemSecretStore()
	account := &dashboardadapter.MockAccountClient{
		ExchangeOAuthTokenFn: func(context.Context, string) (*domain.DashboardOAuthExchangeResponse, error) {
			return &domain.DashboardOAuthExchangeResponse{ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}

	_, err := NewSessionRenewer(account, secrets, &fakeOAuthTokens{token: "at"}, newTestLock(t)).Login(context.Background())

	require.Error(t, err)
	assert.NotContains(t, secrets.data, ports.KeyDashboardSessionOrigin)
}

type unlockFailingLock struct{}

func (unlockFailingLock) Lock(context.Context) (func() error, error) {
	return func() error { return errors.New("unlock failed") }, nil
}

func TestSessionRenewer_ClearIfOAuthThenRunsThenOnceWhenUnlockFails(t *testing.T) {
	// A second, unlocked run of then could log out a session a concurrent
	// login stored after the first.
	secrets := oauthSessionSecrets(time.Now().Add(time.Hour))
	account := &dashboardadapter.MockAccountClient{LogoutFn: func(context.Context, string, string) error { return nil }}
	calls := 0

	clearErr, thenErr := NewSessionRenewer(account, secrets, nil, unlockFailingLock{}).
		ClearIfOAuthThen(context.Background(), func() error { calls++; return nil })

	assert.Equal(t, 1, calls)
	require.Error(t, clearErr, "the unlock failure is still reported")
	require.NoError(t, thenErr)
}

func TestAuthService_SessionWritersTakeTheDashboardSessionLock(t *testing.T) {
	// A password login, a logout or an app selection running while a
	// renewal holds the lock would interleave their keys with its write.
	account := &dashboardadapter.MockAccountClient{
		LoginFn: func(context.Context, string, string, string) (*domain.DashboardAuthResponse, *domain.DashboardMFARequired, error) {
			return &domain.DashboardAuthResponse{UserToken: "password-user"}, nil, nil
		},
		LogoutFn: func(context.Context, string, string) error { return nil },
	}

	for name, write := range map[string]func(context.Context, *AuthService) error{
		"login": func(ctx context.Context, s *AuthService) error {
			_, _, err := s.Login(ctx, "a@example.com", "pw", "")
			return err
		},
		"logout": func(ctx context.Context, s *AuthService) error { return s.Logout(ctx) },
		"apps use": func(ctx context.Context, s *AuthService) error {
			return s.SetActiveApp(ctx, "app_2", "eu")
		},
	} {
		t.Run(name, func(t *testing.T) {
			secrets := newMemSecretStore()
			secrets.data[ports.KeyDashboardUserToken] = "held-user"
			secrets.data[ports.KeyDashboardAppID] = "app_1"
			lock := newTestLock(t)
			unlock, err := lock.Lock(context.Background())
			require.NoError(t, err)
			defer func() { _ = unlock() }()

			svc := NewAuthService(account, secrets).WithSessionRenewer(NewSessionRenewer(account, secrets, nil, lock))
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			require.Error(t, write(ctx, svc))
			assert.Equal(t, "held-user", secrets.data[ports.KeyDashboardUserToken])
			assert.Equal(t, "app_1", secrets.data[ports.KeyDashboardAppID])
		})
	}
}

func TestAuthService_RefreshReusesTokensAnotherProcessStored(t *testing.T) {
	// Refreshing the pair another process already rotated would be refused;
	// the stored replacement is what this process should use.
	secrets := newMemSecretStore()
	secrets.data[ports.KeyDashboardUserToken] = "rotated-user"
	secrets.data[ports.KeyDashboardOrgToken] = "rotated-org"
	account := &dashboardadapter.MockAccountClient{} // a refresh would panic

	user, org, err := NewAuthService(account, secrets).
		WithSessionRenewer(NewSessionRenewer(account, secrets, nil, newTestLock(t))).
		refreshTokens(context.Background(), "spent-user", "spent-org")

	require.NoError(t, err)
	assert.Equal(t, "rotated-user", user)
	assert.Equal(t, "rotated-org", org)
}
