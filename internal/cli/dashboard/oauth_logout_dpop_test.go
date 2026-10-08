package dashboard

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nylas/cli/internal/adapters/filelock"
	"github.com/nylas/cli/internal/ports"
)

// originFailingStore fails reads of the session origin, like a keychain that
// locks between two reads.
type originFailingStore struct{ *memSecretStore }

func (s originFailingStore) Get(key string) (string, error) {
	if key == ports.KeyDashboardSessionOrigin {
		return "", errors.New("keychain locked")
	}
	return s.memSecretStore.Get(key)
}

func TestClearOAuthSessionThen_WithoutAnOAuthSessionCreatesNoDPoPKey(t *testing.T) {
	// `nylas oauth logout` has nothing to revoke here. Building the DPoP
	// service would create and store a key as a side effect of logging out.
	secrets := newMemSecretStore()
	secrets.data[ports.KeyDashboardUserToken] = "dashboard-login-token"
	lock := filelock.New(filepath.Join(t.TempDir(), "dashboard-session.lock"))
	thenRan := false

	clearErr, thenErr := clearOAuthSessionThen(context.Background(), secrets, lock, func() error {
		thenRan = true
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if unlock, err := lock.Lock(ctx); err == nil {
			_ = unlock()
			return errors.New("the dashboard lock was not held")
		}
		return nil
	})

	require.NoError(t, clearErr)
	require.NoError(t, thenErr)
	assert.True(t, thenRan)
	assert.NotContains(t, secrets.data, ports.KeyDashboardDPoPKey)
	assert.Equal(t, "dashboard-login-token", secrets.data[ports.KeyDashboardUserToken],
		"a session from `nylas dashboard login` is not the OAuth logout's to end")
}

func TestClearOAuthSessionThen_ReportsAnUnreadableOriginAndStillLogsOut(t *testing.T) {
	secrets := originFailingStore{newMemSecretStore()}
	lock := filelock.New(filepath.Join(t.TempDir(), "dashboard-session.lock"))
	thenRan := false

	clearErr, thenErr := clearOAuthSessionThen(context.Background(), secrets, lock, func() error {
		thenRan = true
		return nil
	})

	require.Error(t, clearErr, "logout must not report success while it cannot tell whether a session remains")
	require.NoError(t, thenErr)
	assert.True(t, thenRan)
	assert.NotContains(t, secrets.data, ports.KeyDashboardDPoPKey)
}
