package keyring_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockSecretStore(t *testing.T) {
	store := keyring.NewMockSecretStore()

	t.Run("set and get", func(t *testing.T) {
		err := store.Set("key1", "value1")
		require.NoError(t, err)

		value, err := store.Get("key1")
		require.NoError(t, err)
		assert.Equal(t, "value1", value)
	})

	t.Run("get nonexistent key", func(t *testing.T) {
		_, err := store.Get("nonexistent")
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("delete key", func(t *testing.T) {
		err := store.Set("key2", "value2")
		require.NoError(t, err)

		err = store.Delete("key2")
		require.NoError(t, err)

		_, err = store.Get("key2")
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("delete nonexistent key is ok", func(t *testing.T) {
		err := store.Delete("nonexistent")
		assert.NoError(t, err)
	})

	t.Run("is available", func(t *testing.T) {
		assert.True(t, store.IsAvailable())

		store.SetAvailable(false)
		assert.False(t, store.IsAvailable())

		store.SetAvailable(true)
		assert.True(t, store.IsAvailable())
	})

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "mock", store.Name())
	})

	t.Run("reset", func(t *testing.T) {
		err := store.Set("key", "value")
		require.NoError(t, err)

		store.Reset()

		_, err = store.Get("key")
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("get all", func(t *testing.T) {
		store.Reset()
		_ = store.Set("a", "1")
		_ = store.Set("b", "2")

		all := store.GetAll()
		assert.Equal(t, "1", all["a"])
		assert.Equal(t, "2", all["b"])
	})

	t.Run("custom set func", func(t *testing.T) {
		store.SetFunc = func(key, value string) error {
			return domain.ErrSecretStoreFailed
		}

		err := store.Set("key", "value")
		assert.ErrorIs(t, err, domain.ErrSecretStoreFailed)

		store.SetFunc = nil
	})
}

func TestEncryptedFileStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := keyring.NewEncryptedFileStore(tmpDir)
	require.NoError(t, err)

	t.Run("set and get", func(t *testing.T) {
		err := store.Set("test-key", "test-value")
		require.NoError(t, err)

		value, err := store.Get("test-key")
		require.NoError(t, err)
		assert.Equal(t, "test-value", value)
	})

	t.Run("get nonexistent key", func(t *testing.T) {
		_, err := store.Get("nonexistent-key")
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("delete key", func(t *testing.T) {
		err := store.Set("delete-me", "value")
		require.NoError(t, err)

		err = store.Delete("delete-me")
		require.NoError(t, err)

		_, err = store.Get("delete-me")
		assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	})

	t.Run("delete nonexistent key is ok", func(t *testing.T) {
		err := store.Delete("never-existed")
		assert.NoError(t, err)
	})

	t.Run("is available", func(t *testing.T) {
		assert.True(t, store.IsAvailable())
	})

	t.Run("name", func(t *testing.T) {
		assert.Equal(t, "encrypted file", store.Name())
	})

	t.Run("stores client_id correctly", func(t *testing.T) {
		err := store.Set(ports.KeyClientID, "my-client-id-123")
		require.NoError(t, err)

		clientID, err := store.Get(ports.KeyClientID)
		require.NoError(t, err)
		assert.Equal(t, "my-client-id-123", clientID)
	})

	t.Run("stores api_key correctly", func(t *testing.T) {
		err := store.Set(ports.KeyAPIKey, "my-api-key-456")
		require.NoError(t, err)

		apiKey, err := store.Get(ports.KeyAPIKey)
		require.NoError(t, err)
		assert.Equal(t, "my-api-key-456", apiKey)
	})

	t.Run("file is created with correct permissions", func(t *testing.T) {
		secretsPath := filepath.Join(tmpDir, ".secrets.enc")
		info, err := os.Stat(secretsPath)
		require.NoError(t, err)

		if runtime.GOOS != "windows" {
			perm := info.Mode().Perm()
			assert.Equal(t, os.FileMode(0600), perm, "secrets file should have 0600 permissions")
		}
	})
}

func TestNewSecretStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := keyring.NewSecretStore(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, store)

	assert.True(t, store.IsAvailable())

	name := store.Name()
	assert.True(t, name == "system keyring" || name == "encrypted file",
		"store name should be 'system keyring' or 'encrypted file', got: %s", name)

	t.Logf("Platform: %s, Secret store: %s", runtime.GOOS, name)
}

func TestSystemKeyring(t *testing.T) {
	kr := keyring.NewSystemKeyring()
	require.NotNil(t, kr)

	assert.Equal(t, "system keyring", kr.Name())

	isAvailable := kr.IsAvailable()
	t.Logf("Platform: %s, System keyring available: %v", runtime.GOOS, isAvailable)

	if isAvailable {
		t.Run("set and get with system keyring", func(t *testing.T) {
			testKey := "__nylas_test_key__"
			testValue := "test-value-12345"

			err := kr.Set(testKey, testValue)
			require.NoError(t, err)

			value, err := kr.Get(testKey)
			require.NoError(t, err)
			assert.Equal(t, testValue, value)

			err = kr.Delete(testKey)
			require.NoError(t, err)

			_, err = kr.Get(testKey)
			assert.ErrorIs(t, err, domain.ErrSecretNotFound)
		})
	}
}

func TestCrossPlatformKeyDerivation(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := keyring.NewEncryptedFileStore(tmpDir)
	require.NoError(t, err, "EncryptedFileStore should be creatable on %s", runtime.GOOS)

	err = store.Set("cross-platform-test", "value-12345")
	require.NoError(t, err)

	value, err := store.Get("cross-platform-test")
	require.NoError(t, err)
	assert.Equal(t, "value-12345", value)

	t.Logf("Cross-platform key derivation works on: %s", runtime.GOOS)
}

func TestGrantStore(t *testing.T) {
	secrets := keyring.NewMockSecretStore()
	store := keyring.NewGrantStore(secrets)

	t.Run("save and get grant", func(t *testing.T) {
		info := domain.GrantInfo{
			ID:       "test-grant-1",
			Email:    "test@example.com",
			Provider: domain.ProviderGoogle,
		}

		err := store.SaveGrant(info)
		require.NoError(t, err)

		retrieved, err := store.GetGrant("test-grant-1")
		require.NoError(t, err)
		assert.Equal(t, info.ID, retrieved.ID)
		assert.Equal(t, info.Email, retrieved.Email)
	})

	t.Run("get grant by email", func(t *testing.T) {
		retrieved, err := store.GetGrantByEmail("test@example.com")
		require.NoError(t, err)
		assert.Equal(t, "test-grant-1", retrieved.ID)
	})

	t.Run("list grants", func(t *testing.T) {
		grants, err := store.ListGrants()
		require.NoError(t, err)
		assert.Len(t, grants, 1)
	})

	t.Run("set and get default grant", func(t *testing.T) {
		err := store.SetDefaultGrant("test-grant-1")
		require.NoError(t, err)

		defaultID, err := store.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "test-grant-1", defaultID)
	})

	t.Run("delete grant", func(t *testing.T) {
		err := store.DeleteGrant("test-grant-1")
		require.NoError(t, err)

		_, err = store.GetGrant("test-grant-1")
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)
	})
}

func TestGrantStore_MultipleGrants(t *testing.T) {
	secrets := keyring.NewMockSecretStore()
	store := keyring.NewGrantStore(secrets)

	// Create test grants
	grant1 := domain.GrantInfo{
		ID:       "grant-1",
		Email:    "user1@gmail.com",
		Provider: domain.ProviderGoogle,
	}
	grant2 := domain.GrantInfo{
		ID:       "grant-2",
		Email:    "user2@outlook.com",
		Provider: domain.ProviderMicrosoft,
	}
	grant3 := domain.GrantInfo{
		ID:       "grant-3",
		Email:    "user3@gmail.com",
		Provider: domain.ProviderGoogle,
	}

	t.Run("save multiple grants", func(t *testing.T) {
		require.NoError(t, store.SaveGrant(grant1))
		require.NoError(t, store.SaveGrant(grant2))
		require.NoError(t, store.SaveGrant(grant3))

		grants, err := store.ListGrants()
		require.NoError(t, err)
		assert.Len(t, grants, 3)
	})

	t.Run("get grant by email from multiple", func(t *testing.T) {
		retrieved, err := store.GetGrantByEmail("user2@outlook.com")
		require.NoError(t, err)
		assert.Equal(t, "grant-2", retrieved.ID)
		assert.Equal(t, domain.ProviderMicrosoft, retrieved.Provider)
	})

	t.Run("set and switch default grant", func(t *testing.T) {
		// Set grant1 as default
		require.NoError(t, store.SetDefaultGrant("grant-1"))
		defaultID, err := store.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)

		// Switch to grant2
		require.NoError(t, store.SetDefaultGrant("grant-2"))
		defaultID, err = store.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-2", defaultID)

		// Switch to grant3
		require.NoError(t, store.SetDefaultGrant("grant-3"))
		defaultID, err = store.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-3", defaultID)
	})

	t.Run("update existing grant preserves order", func(t *testing.T) {
		updatedGrant2 := domain.GrantInfo{
			ID:       "grant-2",
			Email:    "user2-updated@outlook.com",
			Provider: domain.ProviderMicrosoft,
		}
		require.NoError(t, store.SaveGrant(updatedGrant2))

		retrieved, err := store.GetGrant("grant-2")
		require.NoError(t, err)
		assert.Equal(t, "user2-updated@outlook.com", retrieved.Email)

		// Should still have 3 grants
		grants, err := store.ListGrants()
		require.NoError(t, err)
		assert.Len(t, grants, 3)
	})

	t.Run("delete grant preserves others", func(t *testing.T) {
		require.NoError(t, store.DeleteGrant("grant-2"))

		grants, err := store.ListGrants()
		require.NoError(t, err)
		assert.Len(t, grants, 2)

		// Remaining grants should be grant1 and grant3
		_, err = store.GetGrant("grant-1")
		require.NoError(t, err)
		_, err = store.GetGrant("grant-3")
		require.NoError(t, err)
		_, err = store.GetGrant("grant-2")
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)
	})

	t.Run("clear grants removes all", func(t *testing.T) {
		require.NoError(t, store.ClearGrants())

		grants, err := store.ListGrants()
		require.NoError(t, err)
		assert.Len(t, grants, 0)

		_, err = store.GetDefaultGrant()
		assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
	})

	t.Run("no default grant error", func(t *testing.T) {
		_, err := store.GetDefaultGrant()
		assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
	})
}

// TestGrantStore_DefaultGrantBehavior tests the behavior of default grants
// when grants are deleted. Note: DeleteGrant does NOT automatically clear
// the default_grant key - that's the caller's responsibility (auth service).
func TestGrantStore_DefaultGrantBehavior(t *testing.T) {
	secrets := keyring.NewMockSecretStore()
	store := keyring.NewGrantStore(secrets)

	grant1 := domain.GrantInfo{
		ID:       "grant-1",
		Email:    "user1@gmail.com",
		Provider: domain.ProviderGoogle,
	}

	t.Run("delete default grant leaves stale reference", func(t *testing.T) {
		// Save a grant and set it as default
		require.NoError(t, store.SaveGrant(grant1))
		require.NoError(t, store.SetDefaultGrant(grant1.ID))

		// Verify it's the default
		defaultID, err := store.GetDefaultGrant()
		require.NoError(t, err)
		assert.Equal(t, "grant-1", defaultID)

		// Delete the grant
		require.NoError(t, store.DeleteGrant(grant1.ID))

		// Verify grant is deleted
		_, err = store.GetGrant(grant1.ID)
		assert.ErrorIs(t, err, domain.ErrGrantNotFound)

		// IMPORTANT: The default_grant key still contains the old ID
		// This is by design - the caller (auth service) must handle this
		defaultID, err = store.GetDefaultGrant()
		require.NoError(t, err, "GetDefaultGrant should NOT return error - key still exists")
		assert.Equal(t, "grant-1", defaultID, "Default still points to deleted grant")
	})

	t.Run("clear grants clears default", func(t *testing.T) {
		// Re-add a grant
		require.NoError(t, store.SaveGrant(grant1))
		require.NoError(t, store.SetDefaultGrant(grant1.ID))

		// Clear all grants
		require.NoError(t, store.ClearGrants())

		// Now default should be cleared
		_, err := store.GetDefaultGrant()
		assert.ErrorIs(t, err, domain.ErrNoDefaultGrant)
	})
}
