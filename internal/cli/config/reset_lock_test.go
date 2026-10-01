package config

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/filelock"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearSessions_WaitsForARefreshInFlight(t *testing.T) {
	// A `nylas mcp serve` refresh holds the OAuth lock while it stores the
	// rotated tokens. A reset that cleared without the lock would report
	// success and then have the session written back behind it.
	dir := t.TempDir()
	dashboardLock := filelock.New(filepath.Join(dir, "dashboard-session.lock"))
	oauthLock := filelock.New(filepath.Join(dir, "oauth-session.lock"))
	secrets := keyring.NewMockSecretStore()
	require.NoError(t, secrets.Set(ports.KeyOAuthAccessToken, "at-1"))

	unlock, err := oauthLock.Lock(context.Background())
	require.NoError(t, err)
	refreshed := make(chan struct{})
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = secrets.Set(ports.KeyOAuthRefreshToken, "rt-rotated")
		_ = secrets.Set(ports.KeyOAuthAccessToken, "at-rotated")
		_ = unlock()
		close(refreshed)
	}()

	require.NoError(t, clearSessions(context.Background(), secrets, dashboardLock, oauthLock))
	<-refreshed

	stored := secrets.GetAll()
	assert.NotContains(t, stored, ports.KeyOAuthRefreshToken)
	assert.NotContains(t, stored, ports.KeyOAuthAccessToken)
}
