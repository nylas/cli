package dashboard

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dashboardadapter "github.com/nylas/cli/internal/adapters/dashboard"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// originUnreadable is an OAuth session whose origin marker cannot be read,
// like a keychain that locks between two reads.
func originUnreadable() *failingSecretStore {
	return &failingSecretStore{
		memSecretStore: oauthSessionSecrets(time.Now().Add(-time.Minute)),
		failGetKey:     ports.KeyDashboardSessionOrigin,
	}
}

func TestIsOAuthSession_ReportsAFailedReadInsteadOfNo(t *testing.T) {
	fromOAuth, err := IsOAuthSession(originUnreadable())

	require.ErrorIs(t, err, domain.ErrSecretStoreFailed)
	assert.False(t, fromOAuth)
}

func TestAuthService_RefreshDoesNotRefreshWhenTheOriginCannotBeRead(t *testing.T) {
	// Read as "not from OAuth", the session would go to a refresh the server
	// refuses for OAuth sessions, hiding the keyring failure behind it.
	refreshed := false
	account := &dashboardadapter.MockAccountClient{
		RefreshFn: func(context.Context, string, string) (*domain.DashboardRefreshResponse, error) {
			refreshed = true
			return &domain.DashboardRefreshResponse{UserToken: "refreshed"}, nil
		},
	}

	_, _, err := NewAuthService(account, originUnreadable()).refreshTokens(context.Background(), "old-user", "old-org")

	require.ErrorIs(t, err, domain.ErrSecretStoreFailed)
	assert.False(t, refreshed)
}

func TestSessionRenewer_EnsureFreshReportsAnUnreadableOrigin(t *testing.T) {
	// An expired OAuth session must not be sent as-is because its origin
	// could not be read.
	err := NewSessionRenewer(&dashboardadapter.MockAccountClient{}, originUnreadable(), &fakeOAuthTokens{token: "at"}, newTestLock(t)).
		EnsureFresh(context.Background())

	require.ErrorIs(t, err, domain.ErrSecretStoreFailed)
}

func TestSessionRenewer_LoginDoesNotExchangeWhenTheOriginCannotBeRead(t *testing.T) {
	// Without the origin it is unknown whether a stored session is the
	// user's own dashboard login, so it must not be overwritten.
	secrets := &failingSecretStore{memSecretStore: newMemSecretStore(), failGetKey: ports.KeyDashboardSessionOrigin}
	exchanged := false
	account := &dashboardadapter.MockAccountClient{
		ExchangeOAuthTokenFn: func(context.Context, string) (*domain.DashboardOAuthExchangeResponse, error) {
			exchanged = true
			return &domain.DashboardOAuthExchangeResponse{
				DashboardAuthResponse: domain.DashboardAuthResponse{UserToken: "new-user"},
				ExpiresAt:             time.Now().Add(time.Hour),
			}, nil
		},
	}

	_, err := NewSessionRenewer(account, secrets, &fakeOAuthTokens{token: "at"}, newTestLock(t)).Login(context.Background())

	require.ErrorIs(t, err, domain.ErrSecretStoreFailed)
	assert.NotErrorIs(t, err, ErrDashboardLoginSessionKept)
	assert.False(t, exchanged)
}
