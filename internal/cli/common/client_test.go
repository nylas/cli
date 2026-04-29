//go:build !integration

package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainsAt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"email address", "user@example.com", true},
		{"with plus sign", "user+tag@example.com", true},
		{"multiple @", "a@b@c", true},
		{"@ at start", "@example.com", true},
		{"@ at end", "user@", true},
		{"just @", "@", true},
		{"no @", "username", false},
		{"empty string", "", false},
		{"spaces only", "   ", false},
		{"number string", "12345", false},
		{"special chars no @", "user.name+tag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNylasClient_WithEnvVar(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origClientID := os.Getenv("NYLAS_CLIENT_ID")
	origClientSecret := os.Getenv("NYLAS_CLIENT_SECRET")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_CLIENT_ID", origClientID)
		setEnvOrUnset("NYLAS_CLIENT_SECRET", origClientSecret)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	}()

	// Set test env vars
	_ = os.Setenv("NYLAS_API_KEY", "test-api-key-12345")
	_ = os.Setenv("NYLAS_CLIENT_ID", "test-client-id")
	_ = os.Setenv("NYLAS_CLIENT_SECRET", "test-client-secret")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	client, err := GetNylasClient()

	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestGetNylasClient_NoAPIKey(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("HOME", origHome)
	}()

	// Use temp directory to isolate from real config/credentials
	tempDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir)
	_ = os.Setenv("HOME", tempDir)

	// Clear API key and disable keyring
	_ = os.Unsetenv("NYLAS_API_KEY")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	client, err := GetNylasClient()

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API key not configured")
}

func TestGetAPIKey_WithEnvVar(t *testing.T) {
	// Save original env var
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	}()

	// Set test env var
	testKey := "test-api-key-67890"
	_ = os.Setenv("NYLAS_API_KEY", testKey)
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	apiKey, err := GetAPIKey()

	require.NoError(t, err)
	assert.Equal(t, testKey, apiKey)
}

func TestGetAPIKey_NoAPIKey(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("HOME", origHome)
	}()

	// Use temp directory to isolate from real config/credentials
	tempDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir)
	_ = os.Setenv("HOME", tempDir)

	// Clear API key and disable keyring
	_ = os.Unsetenv("NYLAS_API_KEY")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	apiKey, err := GetAPIKey()

	assert.Error(t, err)
	assert.Empty(t, apiKey)
	assert.Contains(t, err.Error(), "API key not configured")
}

func TestGetGrantID_WithArgument(t *testing.T) {
	configDir := seedLockedFileStore(t, func(store *keyring.EncryptedFileStore) {
		require.NoError(t, store.Set("placeholder", "value"))
	})
	require.DirExists(t, configDir)
	t.Setenv("NYLAS_GRANT_ID", "")

	grantID, err := GetGrantID([]string{"grant-id-12345"})

	require.NoError(t, err)
	assert.Equal(t, "grant-id-12345", grantID)
}

func TestGetGrantID_WithEnvVar(t *testing.T) {
	testGrantID := "env-grant-id-67890"
	seedLockedFileStore(t, func(store *keyring.EncryptedFileStore) {
		require.NoError(t, store.Set("placeholder", "value"))
	})
	t.Setenv("NYLAS_GRANT_ID", testGrantID)

	grantID, err := GetGrantID([]string{})

	require.NoError(t, err)
	assert.Equal(t, testGrantID, grantID)
}

func TestGetGrantID_EmptyArgs(t *testing.T) {
	// Save original env vars
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origXDGCacheHome := os.Getenv("XDG_CACHE_HOME")
	origHome := os.Getenv("HOME")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("XDG_CACHE_HOME", origXDGCacheHome)
		setEnvOrUnset("HOME", origHome)
	}()

	// Use temp directory to isolate from real config file
	tempDir := t.TempDir()
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir)
	_ = os.Setenv("XDG_CACHE_HOME", filepath.Join(tempDir, "cache"))
	_ = os.Setenv("HOME", tempDir)
	_ = os.Unsetenv("NYLAS_GRANT_ID")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	// Test with empty args and no env var
	grantID, err := GetGrantID([]string{})

	// Should fail - no grant ID available
	assert.Error(t, err)
	assert.Empty(t, grantID)
}

func TestGetGrantID_EmptyStringArg(t *testing.T) {
	testGrantID := "env-grant-fallback"
	seedLockedFileStore(t, func(store *keyring.EncryptedFileStore) {
		require.NoError(t, store.Set("placeholder", "value"))
	})
	t.Setenv("NYLAS_GRANT_ID", testGrantID)

	grantID, err := GetGrantID([]string{""})

	require.NoError(t, err)
	assert.Equal(t, testGrantID, grantID)
}

func TestGetGrantID_PrefersStoredDefaultOverStaleConfig(t *testing.T) {
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origFileStorePassphrase := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	origHome := os.Getenv("HOME")

	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("NYLAS_FILE_STORE_PASSPHRASE", origFileStorePassphrase)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfigHome)
		setEnvOrUnset("HOME", origHome)
	}()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	_ = os.Setenv("XDG_CONFIG_HOME", configHome)
	_ = os.Setenv("HOME", tempDir)
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase")
	_ = os.Unsetenv("NYLAS_GRANT_ID")

	configStore := config.NewFileStore(filepath.Join(configHome, "nylas", "config.yaml"))
	require.NoError(t, configStore.Save(&domain.Config{
		Region:       "us",
		DefaultGrant: "stale-config-grant",
		Grants:       []domain.GrantInfo{{ID: "stale-config-grant", Email: "stale@example.com"}},
	}))

	grantStore, err := NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SaveGrant(domain.GrantInfo{ID: "stored-default", Email: "active@example.com"}))
	require.NoError(t, grantStore.SetDefaultGrant("stored-default"))

	grantID, err := GetGrantID(nil)
	require.NoError(t, err)
	assert.Equal(t, "stored-default", grantID)
}

func TestGetGrantID_FallsBackToConfigWhenLegacyStoreLocked(t *testing.T) {
	configDir := seedLockedFileStore(t, func(store *keyring.EncryptedFileStore) {
		require.NoError(t, store.Set("grants", `[{"id":"stored-default","email":"active@example.com","provider":"google"}]`))
		require.NoError(t, store.Set("default_grant", "stored-default"))
	})

	configStore := config.NewFileStore(filepath.Join(configDir, "config.yaml"))
	require.NoError(t, configStore.Save(&domain.Config{
		Region:       "us",
		DefaultGrant: "stale-config-grant",
		Grants:       []domain.GrantInfo{{ID: "stale-config-grant", Email: "stale@example.com"}},
	}))

	grantID, err := GetGrantID(nil)

	require.NoError(t, err)
	assert.Equal(t, "stale-config-grant", grantID)
}

func TestGetGrantID_DoesNotUseStaleConfigDefaultWhenCacheExists(t *testing.T) {
	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tempDir, "cache"))
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_API_KEY", "")
	t.Setenv("NYLAS_GRANT_ID", "")

	configStore := config.NewFileStore(filepath.Join(configHome, "nylas", "config.yaml"))
	require.NoError(t, configStore.Save(&domain.Config{
		Region:       "us",
		DefaultGrant: "stale-config-grant",
	}))

	grantStore, err := NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SaveGrant(domain.GrantInfo{
		ID:    "grant-1",
		Email: "one@example.com",
	}))
	require.NoError(t, grantStore.SetDefaultGrant(""))

	grantID, err := GetGrantID(nil)
	require.Error(t, err)
	assert.Empty(t, grantID)
	assert.Contains(t, err.Error(), "No grant ID provided")
	assert.Contains(t, err.Error(), "nylas auth list")
	assert.Contains(t, err.Error(), "nylas auth switch")
}

// setEnvOrUnset sets an environment variable if value is non-empty, otherwise unsets it
func setEnvOrUnset(key, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	} else {
		_ = os.Unsetenv(key)
	}
}

func seedLockedFileStore(t *testing.T, seed func(store *keyring.EncryptedFileStore)) string {
	t.Helper()

	tempDir := t.TempDir()
	configHome := filepath.Join(tempDir, "xdg")
	configDir := filepath.Join(configHome, "nylas")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase")
	t.Setenv("NYLAS_API_KEY", "")
	t.Setenv("NYLAS_GRANT_ID", "")

	store, err := keyring.NewEncryptedFileStore(configDir)
	require.NoError(t, err)
	if seed != nil {
		seed(store)
	}

	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "")
	return configDir
}

func TestGetNylasClient_EnvVarPriority(t *testing.T) {
	// This test verifies that environment variables take priority over keyring
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origClientID := os.Getenv("NYLAS_CLIENT_ID")
	origClientSecret := os.Getenv("NYLAS_CLIENT_SECRET")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_CLIENT_ID", origClientID)
		setEnvOrUnset("NYLAS_CLIENT_SECRET", origClientSecret)
	}()

	// Set env vars - these should be used regardless of keyring state
	_ = os.Setenv("NYLAS_API_KEY", "env-api-key")
	_ = os.Setenv("NYLAS_CLIENT_ID", "env-client-id")
	_ = os.Setenv("NYLAS_CLIENT_SECRET", "env-client-secret")

	client, err := GetNylasClient()

	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestGetAPIKey_EnvVarPriority(t *testing.T) {
	// Save original env var
	origAPIKey := os.Getenv("NYLAS_API_KEY")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
	}()

	// Set env var
	_ = os.Setenv("NYLAS_API_KEY", "priority-test-key")

	apiKey, err := GetAPIKey()

	require.NoError(t, err)
	assert.Equal(t, "priority-test-key", apiKey)
}

func TestGetAPIKey_ReportsLockedFileStore(t *testing.T) {
	seedLockedFileStore(t, func(store *keyring.EncryptedFileStore) {
		require.NoError(t, store.Set(ports.KeyAPIKey, "stored-api-key"))
	})

	apiKey, err := GetAPIKey()

	require.Error(t, err)
	assert.Empty(t, apiKey)
	assert.ErrorIs(t, err, domain.ErrSecretStoreFailed)
	assert.Contains(t, err.Error(), "NYLAS_FILE_STORE_PASSPHRASE")
}

func TestGetNylasClient_ReportsLockedFileStore(t *testing.T) {
	seedLockedFileStore(t, func(store *keyring.EncryptedFileStore) {
		require.NoError(t, store.Set(ports.KeyAPIKey, "stored-api-key"))
		require.NoError(t, store.Set(ports.KeyClientID, "stored-client-id"))
	})

	client, err := GetNylasClient()

	require.Error(t, err)
	assert.Nil(t, client)
	assert.ErrorIs(t, err, domain.ErrSecretStoreFailed)
	assert.Contains(t, err.Error(), "NYLAS_FILE_STORE_PASSPHRASE")
}

func TestContainsAt_UnicodeSupport(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"unicode email", "用户@example.com", true},
		{"emoji without @", "🎉test", false},
		{"emoji with @", "🎉@test", true},
		{"cyrillic without @", "тест", false},
		{"cyrillic with @", "тест@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
