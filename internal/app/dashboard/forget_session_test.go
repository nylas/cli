package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
	"github.com/nylas/cli/internal/ports"
)

func TestForgetSession_RemovesTheSessionAndTheKey(t *testing.T) {
	// A stored DPoP key that is not a key blocks every dashboard command;
	// forgetting it with its session is how the user gets back to a login.
	secrets := oauthSessionSecrets(time.Now().Add(time.Hour))
	secrets.data[ports.KeyDashboardDPoPKey] = "not a key"

	require.NoError(t, ForgetSession(context.Background(), secrets, newTestLock(t)))

	assert.NotContains(t, secrets.data, ports.KeyDashboardUserToken)
	assert.NotContains(t, secrets.data, ports.KeyDashboardSessionOrigin)
	assert.NotContains(t, secrets.data, ports.KeyDashboardDPoPKey)
}

func TestSessionRenewer_ForgetIfOAuthThenClearsWithoutTheServer(t *testing.T) {
	// No account client: forgetting must not try a revocation it cannot sign.
	secrets := oauthSessionSecrets(time.Now().Add(time.Hour))
	thenRan := false

	clearErr, thenErr := NewSessionRenewer(nil, secrets, nil, newTestLock(t)).ForgetIfOAuthThen(context.Background(), func() error {
		thenRan = true
		return nil
	})

	require.NoError(t, clearErr)
	require.NoError(t, thenErr)
	assert.True(t, thenRan)
	assert.NotContains(t, secrets.data, ports.KeyDashboardUserToken,
		"the dashboard session from OAuth must not outlive the OAuth logout")
}

func TestSessionRenewer_ForgetIfOAuthThenKeepsADashboardLoginSession(t *testing.T) {
	secrets := newMemSecretStore()
	secrets.data[ports.KeyDashboardUserToken] = "dashboard-login-token"

	clearErr, thenErr := NewSessionRenewer(nil, secrets, nil, newTestLock(t)).ForgetIfOAuthThen(context.Background(), func() error { return nil })

	require.NoError(t, clearErr)
	require.NoError(t, thenErr)
	assert.Equal(t, "dashboard-login-token", secrets.data[ports.KeyDashboardUserToken])
}

func TestAuthService_LogoutReportsASessionWhoseServerCannotBeChecked(t *testing.T) {
	// A legacy session (no server recorded) whose origin cannot be read may
	// have come from another server, so it is cleared but not sent anywhere,
	// and the logout does not claim a revocation that never happened.
	account := &dashboardadapter.MockAccountClient{
		LogoutFn: func(context.Context, string, string) error {
			t.Fatal("tokens were sent to a server that could not be checked")
			return nil
		},
	}
	secrets := &failingSecretStore{memSecretStore: newMemSecretStore(), failGetKey: ports.KeyDashboardSessionOrigin}
	secrets.data[ports.KeyDashboardUserToken] = "user-token"

	err := NewAuthService(account, secrets).WithServer(prodServer).Logout(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cleared but not revoked")
	assert.NotContains(t, secrets.data, ports.KeyDashboardUserToken)
}
