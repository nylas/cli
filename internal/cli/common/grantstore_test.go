//go:build !integration

package common

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultGrantStoreMigratesLegacyDefaultGrant(t *testing.T) {
	tempDir := t.TempDir()
	cacheHome := filepath.Join(tempDir, "cache")
	configHome := filepath.Join(tempDir, "config")
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase")

	configStore := config.NewDefaultFileStore()
	require.NoError(t, configStore.Save(&domain.Config{
		Region:       "us",
		DefaultGrant: "config-default",
	}))

	legacyStore, err := keyring.NewEncryptedFileStore(config.DefaultConfigDir())
	require.NoError(t, err)
	legacyGrants := []domain.GrantInfo{
		{ID: "grant-1", Email: "one@example.com", Provider: domain.ProviderGoogle},
		{ID: "grant-2", Email: "two@example.com", Provider: domain.ProviderMicrosoft},
	}
	data, err := json.Marshal(legacyGrants)
	require.NoError(t, err)
	require.NoError(t, legacyStore.Set("grants", string(data)))
	require.NoError(t, legacyStore.Set("default_grant", "grant-2"))

	grantStore, err := NewDefaultGrantStore()
	require.NoError(t, err)

	defaultGrant, err := grantStore.GetDefaultGrant()
	require.NoError(t, err)
	assert.Equal(t, "grant-2", defaultGrant)

	grants, err := grantStore.ListGrants()
	require.NoError(t, err)
	assert.Len(t, grants, 2)

	_, err = legacyStore.Get("grants")
	assert.True(t, errors.Is(err, domain.ErrSecretNotFound), "legacy grants key should be deleted")
	_, err = legacyStore.Get("default_grant")
	assert.True(t, errors.Is(err, domain.ErrSecretNotFound), "legacy default key should be deleted")
}

func TestNewDefaultGrantStoreDoesNotReimportConfigDefaultWhenCacheExists(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tempDir, "cache"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, "config"))
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")

	configStore := config.NewDefaultFileStore()
	require.NoError(t, configStore.Save(&domain.Config{
		Region:       "us",
		DefaultGrant: "stale-config-default",
	}))

	grantStore, err := NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SaveGrant(domain.GrantInfo{
		ID:    "grant-1",
		Email: "one@example.com",
	}))
	require.NoError(t, grantStore.SetDefaultGrant(""))

	reopened, err := NewDefaultGrantStore()
	require.NoError(t, err)

	defaultGrant, err := reopened.GetDefaultGrant()
	assert.Empty(t, defaultGrant)
	assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
}
