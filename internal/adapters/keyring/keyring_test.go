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

func setFileStorePassphrase(t *testing.T) {
	t.Helper()

	orig := os.Getenv("NYLAS_FILE_STORE_PASSPHRASE")
	require.NoError(t, os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase"))
	t.Cleanup(func() {
		if orig != "" {
			_ = os.Setenv("NYLAS_FILE_STORE_PASSPHRASE", orig)
		} else {
			_ = os.Unsetenv("NYLAS_FILE_STORE_PASSPHRASE")
		}
	})
}

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
	setFileStorePassphrase(t)

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
	setFileStorePassphrase(t)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")

	store, err := keyring.NewSecretStore(tmpDir)
	require.NoError(t, err)
	require.NotNil(t, store)

	assert.True(t, store.IsAvailable())

	name := store.Name()
	assert.Equal(t, "encrypted file", name)

	t.Logf("Platform: %s, Secret store: %s", runtime.GOOS, name)
}

func TestSystemKeyring(t *testing.T) {
	if os.Getenv("NYLAS_RUN_SYSTEM_KEYRING_TESTS") != "true" {
		t.Skip("set NYLAS_RUN_SYSTEM_KEYRING_TESTS=true to run live system keyring tests")
	}

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
	setFileStorePassphrase(t)

	store, err := keyring.NewEncryptedFileStore(tmpDir)
	require.NoError(t, err, "EncryptedFileStore should be creatable on %s", runtime.GOOS)

	err = store.Set("cross-platform-test", "value-12345")
	require.NoError(t, err)

	value, err := store.Get("cross-platform-test")
	require.NoError(t, err)
	assert.Equal(t, "value-12345", value)

	t.Logf("Cross-platform key derivation works on: %s", runtime.GOOS)
}
