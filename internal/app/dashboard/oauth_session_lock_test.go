package dashboard

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type concurrentTokens struct{}

func (concurrentTokens) AccessTokenValidFor(context.Context, time.Duration) (string, error) {
	return "at", nil
}

func TestSessionRenewer_ConcurrentRenewalsExchangeOnceAndStoreOneSession(t *testing.T) {
	// Two processes renewing at once must not interleave their writes: the
	// stored user token, org token and expiry have to come from one exchange.
	secrets := keyring.NewMockSecretStore()
	require.NoError(t, secrets.Set(ports.KeyDashboardUserToken, "old-user"))
	require.NoError(t, secrets.Set(ports.KeyDashboardSessionOrigin, sessionOriginOAuth))
	require.NoError(t, secrets.Set(ports.KeyDashboardSessionExpiresAt, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)))

	var exchanges atomic.Int32
	account := &dashboardadapter.MockAccountClient{
		ExchangeOAuthTokenFn: func(context.Context, string) (*domain.DashboardOAuthExchangeResponse, error) {
			n := exchanges.Add(1)
			time.Sleep(20 * time.Millisecond) // widen the window a racing writer would use
			return &domain.DashboardOAuthExchangeResponse{
				DashboardAuthResponse: domain.DashboardAuthResponse{
					UserToken: fmt.Sprintf("user-%d", n),
					OrgToken:  fmt.Sprintf("org-%d", n),
				},
				ExpiresAt: time.Now().Add(15 * time.Minute),
			}, nil
		},
	}
	lock := newTestLock(t)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// A renewer per goroutine, sharing only the store and the lock
			// file, as two processes would.
			errs[i] = NewSessionRenewer(account, secrets, concurrentTokens{}, lock).EnsureFresh(context.Background())
		}()
	}
	wg.Wait()

	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	assert.Equal(t, int32(1), exchanges.Load(), "the waiter must reuse the session the first renewal stored")
	stored := secrets.GetAll()
	assert.Equal(t, "user-1", stored[ports.KeyDashboardUserToken])
	assert.Equal(t, "org-1", stored[ports.KeyDashboardOrgToken])
}

func TestSessionRenewer_AsksForATokenThatOutlivesTheRenewWindow(t *testing.T) {
	// The exchanged session ends when the access token does. A token with
	// less than the renew window left would come back with the same expiry,
	// and every command in that window would exchange again for nothing.
	var exchangedWith string
	tokens := &fakeOAuthTokens{token: "at"}
	secrets := oauthSessionSecrets(time.Now().Add(30 * time.Second))

	require.NoError(t, NewSessionRenewer(exchangingAccount("renewed", &exchangedWith), secrets, tokens, newTestLock(t)).EnsureFresh(context.Background()))

	assert.GreaterOrEqual(t, tokens.minValid, oauthSessionRenewBefore)
}

func TestSessionRenewer_LoginKeepsADashboardLoginSession(t *testing.T) {
	// `nylas oauth login --for mcp` must not silently replace a password or
	// SSO session: its org and app selection are the user's, and it would be
	// left live on the server.
	secrets := newMemSecretStore()
	secrets.data[ports.KeyDashboardUserToken] = "password-user"
	secrets.data[ports.KeyDashboardAppID] = "app_1"
	account := &dashboardadapter.MockAccountClient{} // an exchange would panic

	_, err := NewSessionRenewer(account, secrets, &fakeOAuthTokens{token: "at"}, newTestLock(t)).Login(context.Background())

	require.ErrorIs(t, err, ErrDashboardLoginSessionKept)
	assert.Equal(t, "password-user", secrets.data[ports.KeyDashboardUserToken])
	assert.Equal(t, "app_1", secrets.data[ports.KeyDashboardAppID])
}

func TestSessionRenewer_FailsClosedWithoutALock(t *testing.T) {
	secrets := oauthSessionSecrets(time.Now().Add(-time.Minute))

	err := NewSessionRenewer(&dashboardadapter.MockAccountClient{}, secrets, &fakeOAuthTokens{token: "at"}, nil).EnsureFresh(context.Background())

	require.Error(t, err)
	assert.Equal(t, "old-user", secrets.data[ports.KeyDashboardUserToken])
}

func TestSessionRenewer_ClearIfOAuthThenRunsTheOAuthLogoutUnderTheLock(t *testing.T) {
	// Between the dashboard clear and the OAuth logout, a concurrent
	// `nylas oauth login` must not be able to store a new dashboard session.
	lock := newTestLock(t)
	secrets := oauthSessionSecrets(time.Now().Add(time.Hour))
	account := &dashboardadapter.MockAccountClient{
		LogoutFn: func(context.Context, string, string) error { return nil },
	}
	order := []string{}

	clearErr, thenErr := NewSessionRenewer(account, secrets, nil, lock).ClearIfOAuthThen(context.Background(), func() error {
		_, stillStored := secrets.data[ports.KeyDashboardUserToken]
		order = append(order, fmt.Sprintf("then(dashboard stored=%v)", stillStored))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if unlock, err := lock.Lock(ctx); err == nil {
			_ = unlock()
			return fmt.Errorf("the dashboard lock was not held")
		}
		return nil
	})

	require.NoError(t, clearErr)
	require.NoError(t, thenErr)
	assert.Equal(t, []string{"then(dashboard stored=false)"}, order)
}
