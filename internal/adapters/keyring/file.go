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
	"golang.org/x/crypto/argon2"
)

const (
	fileStorePassphraseEnv = "NYLAS_FILE_STORE_PASSPHRASE"
	fileStoreSaltSize      = 16
)

// EncryptedFileStore implements SecretStore using an encrypted file.
// This is a fallback for environments where the system keyring is unavailable.
// Uses AES-256-GCM encryption with a key derived from user-supplied secret material.
type EncryptedFileStore struct {
	path         string
	keyPath      string
	saltPath     string
	passphrase   []byte
	migrationKey []byte
	legacyKey    []byte
	mu           sync.RWMutex
}

// NewEncryptedFileStore creates a new EncryptedFileStore.
// The secrets are stored in an encrypted file within the config directory.
func NewEncryptedFileStore(configDir string) (*EncryptedFileStore, error) {
	path := filepath.Join(configDir, ".secrets.enc")
	keyPath := filepath.Join(configDir, ".secrets.key")
	saltPath := filepath.Join(configDir, ".secrets.salt")

	legacyKey, err := deriveLegacyKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive legacy encryption key: %w", err)
	}

	migrationKey, err := readCompatibilityMasterKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load legacy file-store key: %w", err)
	}

	var passphrase []byte
	if value := os.Getenv(fileStorePassphraseEnv); value != "" {
		passphrase = []byte(value)
	}

	return &EncryptedFileStore{
		path:         path,
		keyPath:      keyPath,
		saltPath:     saltPath,
		passphrase:   passphrase,
		migrationKey: migrationKey,
		legacyKey:    legacyKey,
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
	if err := os.WriteFile(f.path, ciphertext, 0600); err != nil {
		return err
	}

	// Remove the plaintext migration key once the store has been rewritten.
	if f.keyPath != "" {
		_ = os.Remove(f.keyPath)
	}

	return nil
}

// encrypt encrypts plaintext using AES-256-GCM.
func (f *EncryptedFileStore) encrypt(plaintext []byte) ([]byte, error) {
	key, err := f.passphraseKey(true)
	if err != nil {
		return nil, err
	}
	return encryptWithKey(key, plaintext)
}

// decrypt decrypts ciphertext using AES-256-GCM.
func (f *EncryptedFileStore) decrypt(data []byte) ([]byte, error) {
	if key, err := f.passphraseKey(false); err == nil {
		plaintext, err := decryptWithKey(key, data)
		if err == nil {
			return plaintext, nil
		}
	} else if !os.IsNotExist(err) && len(f.passphrase) > 0 {
		return nil, err
	}

	if len(f.migrationKey) > 0 {
		plaintext, err := decryptWithKey(f.migrationKey, data)
		if err == nil {
			return plaintext, nil
		}
	}

	if len(f.legacyKey) > 0 {
		plaintext, err := decryptWithKey(f.legacyKey, data)
		if err == nil {
			return plaintext, nil
		}
	}

	if len(f.passphrase) == 0 {
		return nil, fmt.Errorf("%s must be set to unlock the encrypted file store", fileStorePassphraseEnv)
	}

	return nil, fmt.Errorf("failed to decrypt encrypted file store with the configured passphrase")
}

func encryptWithKey(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
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

func decryptWithKey(key, data []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
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

func readCompatibilityMasterKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid master key length: %d", len(key))
	}
	return key, nil
}

func (f *EncryptedFileStore) passphraseKey(createSalt bool) ([]byte, error) {
	if len(f.passphrase) == 0 {
		return nil, fmt.Errorf("%s must be set to use the encrypted file secret store", fileStorePassphraseEnv)
	}

	salt, err := f.loadSalt(createSalt)
	if err != nil {
		return nil, err
	}

	return derivePassphraseKey(f.passphrase, salt), nil
}

func (f *EncryptedFileStore) loadSalt(create bool) ([]byte, error) {
	data, err := os.ReadFile(f.saltPath)
	if err == nil {
		salt, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
		if err != nil {
			return nil, err
		}
		if len(salt) != fileStoreSaltSize {
			return nil, fmt.Errorf("invalid file-store salt length: %d", len(salt))
		}
		return salt, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	if !create {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(f.saltPath), 0700); err != nil {
		return nil, err
	}

	salt := make([]byte, fileStoreSaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	encoded := base64.StdEncoding.EncodeToString(salt)
	if err := os.WriteFile(f.saltPath, []byte(encoded), 0600); err != nil {
		return nil, err
	}

	return salt, nil
}

func derivePassphraseKey(passphrase, salt []byte) []byte {
	// Argon2id keeps the fallback store bound to user-supplied secret material
	// instead of host metadata while staying fast enough for CLI use.
	return argon2.IDKey(passphrase, salt, 1, 64*1024, 4, 32)
}

// deriveLegacyKey derives the pre-v2 machine-specific fallback key so older
// encrypted files can still be read and rewritten with a passphrase-derived key.
func deriveLegacyKey() ([]byte, error) {
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
