package keyring

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/nylas/cli/internal/domain"
)

// EncryptedFileStore implements SecretStore using an encrypted file.
// This is a fallback for environments where the system keyring is unavailable.
// Uses AES-256-GCM encryption with a machine-specific key.
type EncryptedFileStore struct {
	path string
	key  []byte
	mu   sync.RWMutex
}

// NewEncryptedFileStore creates a new EncryptedFileStore.
// The secrets are stored in an encrypted file within the config directory.
func NewEncryptedFileStore(configDir string) (*EncryptedFileStore, error) {
	path := filepath.Join(configDir, ".secrets.enc")
	key, err := deriveKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}
	return &EncryptedFileStore{
		path: path,
		key:  key,
	}, nil
}

// Set stores a secret value for the given key.
func (f *EncryptedFileStore) Set(key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	secrets, err := f.loadSecrets()
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("%w: %v", domain.ErrSecretStoreFailed, err)
	}
	if secrets == nil {
		secrets = make(map[string]string)
	}

	secrets[key] = value
	if err := f.saveSecrets(secrets); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrSecretStoreFailed, err)
	}
	return nil
}

// Get retrieves a secret value for the given key.
func (f *EncryptedFileStore) Get(key string) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	secrets, err := f.loadSecrets()
	if err != nil {
		if os.IsNotExist(err) {
			return "", domain.ErrSecretNotFound
		}
		return "", fmt.Errorf("%w: %v", domain.ErrSecretStoreFailed, err)
	}

	value, ok := secrets[key]
	if !ok {
		return "", domain.ErrSecretNotFound
	}
	return value, nil
}

// Delete removes a secret for the given key.
func (f *EncryptedFileStore) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	secrets, err := f.loadSecrets()
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Already doesn't exist
		}
		return fmt.Errorf("%w: %v", domain.ErrSecretStoreFailed, err)
	}

	delete(secrets, key)
	if err := f.saveSecrets(secrets); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrSecretStoreFailed, err)
	}
	return nil
}

// IsAvailable always returns true for file-based storage.
func (f *EncryptedFileStore) IsAvailable() bool {
	return true
}

// Name returns the name of the secret store backend.
func (f *EncryptedFileStore) Name() string {
	return "encrypted file"
}

// loadSecrets loads and decrypts the secrets file.
func (f *EncryptedFileStore) loadSecrets() (map[string]string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return nil, err
	}

	plaintext, err := f.decrypt(data)
	if err != nil {
		return nil, err
	}

	var secrets map[string]string
	if err := json.Unmarshal(plaintext, &secrets); err != nil {
		return nil, err
	}
	return secrets, nil
}

// saveSecrets encrypts and saves the secrets file.
func (f *EncryptedFileStore) saveSecrets(secrets map[string]string) error {
	plaintext, err := json.Marshal(secrets)
	if err != nil {
		return err
	}

	ciphertext, err := f.encrypt(plaintext)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Write with restrictive permissions
	return os.WriteFile(f.path, ciphertext, 0600)
}

// encrypt encrypts plaintext using AES-256-GCM.
func (f *EncryptedFileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// decrypt decrypts ciphertext using AES-256-GCM.
func (f *EncryptedFileStore) decrypt(data []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// deriveKey derives a 32-byte encryption key from machine-specific identifiers.
// This makes the encrypted file non-portable but provides reasonable security.
func deriveKey() ([]byte, error) {
	// Collect machine-specific identifiers
	var identifiers []byte

	// Add hostname
	hostname, _ := os.Hostname()
	identifiers = append(identifiers, []byte(hostname)...)

	// Add user info
	identifiers = append(identifiers, []byte(os.Getenv("USER"))...)
	identifiers = append(identifiers, []byte(os.Getenv("USERNAME"))...) // Windows

	// Add home directory
	home, _ := os.UserHomeDir()
	identifiers = append(identifiers, []byte(home)...)

	// Add OS-specific machine ID if available
	machineID := getMachineID()
	identifiers = append(identifiers, []byte(machineID)...)

	// Add a static salt to prevent rainbow table attacks
	salt := []byte("nylas-cli-v1-secret-store")
	identifiers = append(identifiers, salt...)

	// Derive key using SHA-256
	hash := sha256.Sum256(identifiers)
	return hash[:], nil
}

// getMachineID attempts to read a machine-specific ID.
func getMachineID() string {
	switch runtime.GOOS {
	case "linux":
		// Try /etc/machine-id (systemd)
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			return string(data)
		}
		// Try /var/lib/dbus/machine-id
		if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			return string(data)
		}
	case "darwin":
		// Try to get hardware UUID from system profiler
		if data, err := os.ReadFile("/var/db/SystemKey"); err == nil {
			return string(data)
		}
		// Fallback: use boot time + serial from IOKit
		if data, err := os.ReadFile("/Library/Preferences/SystemConfiguration/com.apple.smb.server.plist"); err == nil {
			return string(data)
		}
	case "windows":
		// Try to read MachineGuid from registry path
		programData := os.Getenv("PROGRAMDATA")
		if programData != "" {
			// Construct and clean the path to prevent traversal
			guidPath := filepath.Join(programData, "Microsoft", "Crypto", "RSA", "MachineKeys", ".GUID")
			cleanPath := filepath.Clean(guidPath)

			// Validate the path starts with the expected base (security check)
			if strings.HasPrefix(cleanPath, filepath.Clean(programData)) {
				if data, err := os.ReadFile(cleanPath); err == nil {
					return string(data)
				}
			}
		}
		// Fallback: use system drive serial
		systemRoot := os.Getenv("SYSTEMROOT")
		if systemRoot != "" {
			return systemRoot
		}
	}
	return ""
}
