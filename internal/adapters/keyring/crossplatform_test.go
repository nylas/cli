package keyring

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func setFileStorePassphrase(t *testing.T) string {
	t.Helper()

	orig := os.Getenv(fileStorePassphraseEnv)
	passphrase := "test-file-store-passphrase"
	if err := os.Setenv(fileStorePassphraseEnv, passphrase); err != nil {
		t.Fatalf("failed to set %s: %v", fileStorePassphraseEnv, err)
	}
	t.Cleanup(func() {
		if orig != "" {
			_ = os.Setenv(fileStorePassphraseEnv, orig)
		} else {
			_ = os.Unsetenv(fileStorePassphraseEnv)
		}
	})

	return passphrase
}

// TestCrossPlatformEncryptedFileStore tests the encrypted file store across platforms.
func TestCrossPlatformEncryptedFileStore(t *testing.T) {
	tmpDir := t.TempDir()
	setFileStorePassphrase(t)

	store, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create encrypted file store: %v", err)
	}

	t.Run("set_and_get_secret", func(t *testing.T) {
		key := "test_api_key"
		value := "nyk_v0_TestKeyValue123"

		if err := store.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		got, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if got != value {
			t.Errorf("Got %q, want %q", got, value)
		}
	})

	t.Run("get_nonexistent_returns_error", func(t *testing.T) {
		_, err := store.Get("nonexistent_key")
		if err != domain.ErrSecretNotFound {
			t.Errorf("Expected ErrSecretNotFound, got %v", err)
		}
	})

	t.Run("delete_secret", func(t *testing.T) {
		key := "delete_test_key"
		value := "delete_test_value"

		if err := store.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		if err := store.Delete(key); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err := store.Get(key)
		if err != domain.ErrSecretNotFound {
			t.Errorf("Expected ErrSecretNotFound after delete, got %v", err)
		}
	})

	t.Run("delete_nonexistent_is_ok", func(t *testing.T) {
		if err := store.Delete("nonexistent_key"); err != nil {
			t.Errorf("Delete nonexistent key should not error: %v", err)
		}
	})

	t.Run("is_available", func(t *testing.T) {
		if !store.IsAvailable() {
			t.Error("Encrypted file store should always be available")
		}
	})

	t.Run("name_returns_encrypted_file", func(t *testing.T) {
		name := store.Name()
		if name != "encrypted file" {
			t.Errorf("Expected 'encrypted file', got %q", name)
		}
	})

	t.Run("file_permissions_are_secure", func(t *testing.T) {
		key := "permission_test"
		value := "permission_value"

		if err := store.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		filePath := filepath.Join(tmpDir, ".secrets.enc")
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("Failed to stat secrets file: %v", err)
		}

		// On Unix, check file permissions are 0600
		if runtime.GOOS != "windows" {
			mode := info.Mode().Perm()
			if mode != 0600 {
				t.Errorf("File permissions are %o, want 0600", mode)
			}

			saltInfo, err := os.Stat(filepath.Join(tmpDir, ".secrets.salt"))
			if err != nil {
				t.Fatalf("Failed to stat salt file: %v", err)
			}
			if saltMode := saltInfo.Mode().Perm(); saltMode != 0600 {
				t.Errorf("Salt file permissions are %o, want 0600", saltMode)
			}
		}
	})

	t.Run("handles_special_characters_in_values", func(t *testing.T) {
		key := "special_chars"
		value := "test!@#$%^&*(){}[]|\\:\";<>?,./'`~"

		if err := store.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		got, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if got != value {
			t.Errorf("Got %q, want %q", got, value)
		}
	})

	t.Run("handles_unicode_values", func(t *testing.T) {
		key := "unicode_test"
		value := "测试值 🔐 تست مقدار"

		if err := store.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		got, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if got != value {
			t.Errorf("Got %q, want %q", got, value)
		}
	})

	t.Run("handles_empty_value", func(t *testing.T) {
		key := "empty_value_test"
		value := ""

		if err := store.Set(key, value); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		got, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if got != value {
			t.Errorf("Got %q, want empty string", got)
		}
	})

	t.Run("handles_large_value", func(t *testing.T) {
		key := "large_value_test"
		// Create a large value (10KB - reasonable for secrets)
		value := make([]byte, 10*1024)
		for i := range value {
			value[i] = byte('a' + (i % 26)) // Use printable chars
		}
		valueStr := string(value)

		if err := store.Set(key, valueStr); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		got, err := store.Get(key)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}

		if len(got) != len(valueStr) {
			t.Errorf("Large value length mismatch: got %d, want %d", len(got), len(valueStr))
		}
	})
}

// TestDeriveLegacyKey tests the legacy key derivation kept for migration.
func TestDeriveLegacyKey(t *testing.T) {
	t.Run("key_is_32_bytes", func(t *testing.T) {
		key, err := deriveLegacyKey()
		if err != nil {
			t.Fatalf("deriveLegacyKey failed: %v", err)
		}

		if len(key) != 32 {
			t.Errorf("Key length is %d, want 32", len(key))
		}
	})

	t.Run("key_is_deterministic", func(t *testing.T) {
		key1, err := deriveLegacyKey()
		if err != nil {
			t.Fatalf("deriveLegacyKey failed: %v", err)
		}

		key2, err := deriveLegacyKey()
		if err != nil {
			t.Fatalf("deriveLegacyKey failed: %v", err)
		}

		if string(key1) != string(key2) {
			t.Error("Key derivation is not deterministic")
		}
	})
}

func TestEncryptedFileStore_RequiresPassphraseForWrites(t *testing.T) {
	orig := os.Getenv(fileStorePassphraseEnv)
	if orig != "" {
		_ = os.Unsetenv(fileStorePassphraseEnv)
		t.Cleanup(func() { _ = os.Setenv(fileStorePassphraseEnv, orig) })
	}

	store, err := NewEncryptedFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewEncryptedFileStore failed: %v", err)
	}

	err = store.Set("api_key", "value")
	if err == nil {
		t.Fatal("Set succeeded without passphrase")
	}
	if !strings.Contains(err.Error(), fileStorePassphraseEnv) {
		t.Fatalf("Set error %q does not mention %s", err.Error(), fileStorePassphraseEnv)
	}
}

func TestEncryptedFileStore_MigratesLegacyCiphertext(t *testing.T) {
	tmpDir := t.TempDir()
	passphrase := setFileStorePassphrase(t)
	legacyKey, err := deriveLegacyKey()
	if err != nil {
		t.Fatalf("deriveLegacyKey failed: %v", err)
	}

	legacyCiphertext, err := encryptWithKey(legacyKey, []byte(`{"api_key":"legacy-value"}`))
	if err != nil {
		t.Fatalf("encryptWithKey failed: %v", err)
	}

	secretsPath := filepath.Join(tmpDir, ".secrets.enc")
	if err := os.WriteFile(secretsPath, legacyCiphertext, 0600); err != nil {
		t.Fatalf("failed to write legacy secrets file: %v", err)
	}

	store, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptedFileStore failed: %v", err)
	}

	value, err := store.Get("api_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "legacy-value" {
		t.Fatalf("Get returned %q, want %q", value, "legacy-value")
	}

	if err := store.Set("new_key", "new-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to read rewritten secrets file: %v", err)
	}

	if _, err := decryptWithKey(legacyKey, data); err == nil {
		t.Fatal("rewritten secrets file should no longer use the legacy key")
	}

	salt, err := os.ReadFile(filepath.Join(tmpDir, ".secrets.salt"))
	if err != nil {
		t.Fatalf("failed to read salt file: %v", err)
	}
	decodedSalt, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(salt)))
	if err != nil {
		t.Fatalf("failed to decode salt: %v", err)
	}
	plaintext, err := decryptWithKey(derivePassphraseKey([]byte(passphrase), decodedSalt), data)
	if err != nil {
		t.Fatalf("failed to decrypt rewritten secrets file with passphrase-derived key: %v", err)
	}
	if string(plaintext) == "" {
		t.Fatal("rewritten secrets plaintext should not be empty")
	}
}

func TestEncryptedFileStore_MigratesLegacyMasterKeyCiphertext(t *testing.T) {
	tmpDir := t.TempDir()
	passphrase := setFileStorePassphrase(t)

	migrationKey := make([]byte, 32)
	if _, err := rand.Read(migrationKey); err != nil {
		t.Fatalf("rand.Read failed: %v", err)
	}

	ciphertext, err := encryptWithKey(migrationKey, []byte(`{"api_key":"migrated-value"}`))
	if err != nil {
		t.Fatalf("encryptWithKey failed: %v", err)
	}

	secretsPath := filepath.Join(tmpDir, ".secrets.enc")
	if err := os.WriteFile(secretsPath, ciphertext, 0600); err != nil {
		t.Fatalf("failed to write secrets file: %v", err)
	}

	keyPath := filepath.Join(tmpDir, ".secrets.key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(migrationKey)), 0600); err != nil {
		t.Fatalf("failed to write migration key: %v", err)
	}

	store, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptedFileStore failed: %v", err)
	}

	value, err := store.Get("api_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if value != "migrated-value" {
		t.Fatalf("Get returned %q, want %q", value, "migrated-value")
	}

	if err := store.Set("new_key", "new-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Fatalf("migration key file should be removed after rewrite, stat err = %v", err)
	}

	salt, err := os.ReadFile(filepath.Join(tmpDir, ".secrets.salt"))
	if err != nil {
		t.Fatalf("failed to read salt file: %v", err)
	}
	decodedSalt, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(salt)))
	if err != nil {
		t.Fatalf("failed to decode salt: %v", err)
	}

	data, err := os.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("failed to read rewritten secrets file: %v", err)
	}
	if _, err := decryptWithKey(migrationKey, data); err == nil {
		t.Fatal("rewritten secrets file should no longer use the plaintext migration key")
	}
	if _, err := decryptWithKey(derivePassphraseKey([]byte(passphrase), decodedSalt), data); err != nil {
		t.Fatalf("failed to decrypt rewritten secrets file with passphrase-derived key: %v", err)
	}
}

func TestEncryptedFileStore_ReopensWithSamePassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	setFileStorePassphrase(t)

	store, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptedFileStore failed: %v", err)
	}
	if err := store.Set("api_key", "reopen-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	reopened, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("reopen NewEncryptedFileStore failed: %v", err)
	}
	value, err := reopened.Get("api_key")
	if err != nil {
		t.Fatalf("Get after reopen failed: %v", err)
	}
	if value != "reopen-value" {
		t.Fatalf("Get after reopen returned %q, want %q", value, "reopen-value")
	}
}

func TestEncryptedFileStore_RequiresPassphraseForReads(t *testing.T) {
	tmpDir := t.TempDir()
	orig := os.Getenv(fileStorePassphraseEnv)
	passphrase := setFileStorePassphrase(t)

	store, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptedFileStore failed: %v", err)
	}
	if err := store.Set("api_key", "read-protected-value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	_ = os.Unsetenv(fileStorePassphraseEnv)
	t.Cleanup(func() {
		if orig != "" {
			_ = os.Setenv(fileStorePassphraseEnv, orig)
		} else {
			_ = os.Unsetenv(fileStorePassphraseEnv)
		}
	})

	reopened, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("reopen NewEncryptedFileStore failed: %v", err)
	}

	_, err = reopened.Get("api_key")
	if err == nil {
		t.Fatal("Get succeeded without passphrase")
	}
	if !errors.Is(err, domain.ErrSecretStoreFailed) {
		t.Fatalf("Get error = %v, want ErrSecretStoreFailed", err)
	}
	if !strings.Contains(err.Error(), fileStorePassphraseEnv) {
		t.Fatalf("Get error %q does not mention %s", err.Error(), fileStorePassphraseEnv)
	}

	_ = os.Setenv(fileStorePassphraseEnv, passphrase)
}

// TestGetMachineID tests machine ID retrieval across platforms.
func TestGetMachineID(t *testing.T) {
	t.Logf("Running on %s/%s", runtime.GOOS, runtime.GOARCH)

	machineID := getMachineID()
	t.Logf("Machine ID: %q (length: %d)", machineID, len(machineID))

	// Machine ID might be empty on some systems, that's OK
	// The key derivation has fallbacks
}

// TestNewSecretStore tests secret store creation with fallback.
func TestNewSecretStore(t *testing.T) {
	tmpDir := t.TempDir()
	setFileStorePassphrase(t)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")

	store, err := NewSecretStore(tmpDir)
	if err != nil {
		t.Fatalf("NewSecretStore failed: %v", err)
	}

	t.Logf("Platform: %s, Secret store type: %s", runtime.GOOS, store.Name())

	// Test basic operations
	key := "store_test"
	value := "store_value"

	if err := store.Set(key, value); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got != value {
		t.Errorf("Got %q, want %q", got, value)
	}

	// Cleanup
	_ = store.Delete(key)
}

// TestConcurrentAccess tests concurrent access to the encrypted file store.
func TestConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	setFileStorePassphrase(t)

	store, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create encrypted file store: %v", err)
	}

	done := make(chan bool)
	errChan := make(chan error, 100)

	// Run concurrent writes
	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := "concurrent_key"
			value := "concurrent_value"

			for j := 0; j < 10; j++ {
				if err := store.Set(key, value); err != nil {
					errChan <- err
				}
				if _, err := store.Get(key); err != nil {
					errChan <- err
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	close(errChan)
	for err := range errChan {
		t.Errorf("Concurrent access error: %v", err)
	}
}
