// Package keyring provides secure credential storage using the OS keychain.
package keyring

import (
	"errors"
	"fmt"
	"os"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/zalando/go-keyring"
)

const serviceName = "nylas"

// SystemKeyring implements SecretStore using the system keychain.
type SystemKeyring struct{}

// NewSystemKeyring creates a new SystemKeyring instance.
func NewSystemKeyring() *SystemKeyring {
	return &SystemKeyring{}
}

// Set stores a secret value for the given key.
func (k *SystemKeyring) Set(key, value string) error {
	return keyring.Set(serviceName, key, value)
}

// Get retrieves a secret value for the given key.
func (k *SystemKeyring) Get(key string) (string, error) {
	value, err := keyring.Get(serviceName, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", domain.ErrSecretNotFound
	}
	return value, err
}

// Delete removes a secret for the given key.
func (k *SystemKeyring) Delete(key string) error {
	err := keyring.Delete(serviceName, key)
	if err == keyring.ErrNotFound {
		return nil // Already deleted
	}
	return err
}

// IsAvailable checks if the system keychain is available.
func (k *SystemKeyring) IsAvailable() bool {
	testKey := "__nylas_keyring_test__"
	err := keyring.Set(serviceName, testKey, "test")
	if err != nil {
		return false
	}
	_ = keyring.Delete(serviceName, testKey)
	return true
}

// Name returns the name of the secret store backend.
func (k *SystemKeyring) Name() string {
	return "system keyring"
}

// NewSecretStore creates a SecretStore, preferring system keyring with file fallback.
// If the system keyring is available but empty, and the encrypted file has credentials,
// it will migrate the credentials to the system keyring.
func NewSecretStore(configDir string) (ports.SecretStore, error) {
	// Check if keyring is disabled via environment variable (useful for testing)
	if os.Getenv("NYLAS_DISABLE_KEYRING") == "true" {
		return NewEncryptedFileStore(configDir)
	}

	kr := NewSystemKeyring()
	if !kr.IsAvailable() {
		return NewEncryptedFileStore(configDir)
	}

	// System keyring is available - check if it has credentials
	_, err := kr.Get(ports.KeyAPIKey)
	if err == nil {
		// Keyring has credentials, use it
		return kr, nil
	}

	// Keyring is available but empty - check if file store has credentials
	fileStore, err := NewEncryptedFileStore(configDir)
	if err != nil {
		// Can't create file store, just use keyring
		return kr, nil
	}

	// Check if file store has credentials
	apiKey, err := fileStore.Get(ports.KeyAPIKey)
	if err != nil {
		if errors.Is(err, domain.ErrSecretNotFound) {
			// No credentials in file store either, use keyring for fresh setup
			return kr, nil
		}
		return nil, err
	}

	// Migrate credentials from file store to keyring. Keep going on per-key
	// failures so a single broken entry doesn't block the rest of the move,
	// but surface the failures so the user knows something didn't migrate.
	var migrationErrs []error
	migrate := func(key, value string) {
		if value == "" {
			return
		}
		if err := kr.Set(key, value); err != nil {
			migrationErrs = append(migrationErrs, fmt.Errorf("migrate %s: %w", key, err))
		}
	}

	migrate(ports.KeyAPIKey, apiKey)
	if clientID, err := fileStore.Get(ports.KeyClientID); err == nil {
		migrate(ports.KeyClientID, clientID)
	}
	if clientSecret, err := fileStore.Get(ports.KeyClientSecret); err == nil {
		migrate(ports.KeyClientSecret, clientSecret)
	}

	if len(migrationErrs) > 0 {
		// Print to stderr but do not fail — the keyring is usable even with
		// partial migration; users may need to re-run `nylas auth config`.
		fmt.Fprintf(os.Stderr, "warning: %d secrets failed to migrate from file store to keyring; re-run `nylas auth config` to retry\n", len(migrationErrs))
		for _, e := range migrationErrs {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
	}

	return kr, nil
}
