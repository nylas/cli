package keyring

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// TestCrossPlatformEncryptedFileStore tests the encrypted file store across platforms.
func TestCrossPlatformEncryptedFileStore(t *testing.T) {
	tmpDir := t.TempDir()

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

// TestDeriveKey tests key derivation across platforms.
func TestDeriveKey(t *testing.T) {
	t.Run("key_is_32_bytes", func(t *testing.T) {
		key, err := deriveKey()
		if err != nil {
			t.Fatalf("deriveKey failed: %v", err)
		}

		if len(key) != 32 {
			t.Errorf("Key length is %d, want 32", len(key))
		}
	})

	t.Run("key_is_deterministic", func(t *testing.T) {
		key1, err := deriveKey()
		if err != nil {
			t.Fatalf("deriveKey failed: %v", err)
		}

		key2, err := deriveKey()
		if err != nil {
			t.Fatalf("deriveKey failed: %v", err)
		}

		if string(key1) != string(key2) {
			t.Error("Key derivation is not deterministic")
		}
	})
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
