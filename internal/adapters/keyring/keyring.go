// Package keyring provides secure credential storage using the OS keychain.
package keyring

import (
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
	if err == keyring.ErrNotFound {
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
		// No credentials in file store either, use keyring for fresh setup
		return kr, nil
	}

	// Migrate credentials from file store to keyring
	if apiKey != "" {
		_ = kr.Set(ports.KeyAPIKey, apiKey)
	}
	if clientID, err := fileStore.Get(ports.KeyClientID); err == nil && clientID != "" {
		_ = kr.Set(ports.KeyClientID, clientID)
	}
	if clientSecret, err := fileStore.Get(ports.KeyClientSecret); err == nil && clientSecret != "" {
		_ = kr.Set(ports.KeyClientSecret, clientSecret)
	}

	// Migrate grants data
	if grants, err := fileStore.Get("grants"); err == nil && grants != "" {
		_ = kr.Set("grants", grants)
	}
	if defaultGrant, err := fileStore.Get("default_grant"); err == nil && defaultGrant != "" {
		_ = kr.Set("default_grant", defaultGrant)
	}

	return kr, nil
}
