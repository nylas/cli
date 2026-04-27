package keyring

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/nylas/cli/internal/domain"
	"golang.org/x/crypto/argon2"
)

const (
	fileStorePassphraseEnv = "NYLAS_FILE_STORE_PASSPHRASE"
	fileStoreSaltSize      = 16
)

// fileStoreKeyType describes which key(s) can decrypt the on-disk .secrets.enc file.
type fileStoreKeyType int

const (
	fileStoreKeyNone           fileStoreKeyType = iota // file does not exist or neither key decrypts it
	fileStoreKeyLegacyOnly                             // decryptable only with the legacy machine-derived key
	fileStoreKeyPassphraseOnly                         // decryptable only with the passphrase-derived key
	fileStoreKeyBoth                                   // decryptable with either key
)

// EncryptedFileStore implements SecretStore using an encrypted file.
// This is a fallback for environments where the system keyring is unavailable.
// Uses AES-256-GCM encryption with an Argon2id key derived from a user-supplied
// passphrase set via NYLAS_FILE_STORE_PASSPHRASE.
//
// REQUIREMENT: NYLAS_FILE_STORE_PASSPHRASE must be set before using this store
// on a fresh install. Existing installations that used the legacy machine-derived
// key will be migrated automatically the first time NYLAS_FILE_STORE_PASSPHRASE
// is supplied. If the environment variable is unset and no legacy file exists,
// construction fails immediately.
//
// To avoid the file store entirely, leave NYLAS_DISABLE_KEYRING unset and let
// the system keyring be used, or run `nylas auth config` to reconfigure.
type EncryptedFileStore struct {
	path         string
	keyPath      string
	saltPath     string
	passphrase   []byte
	migrationKey []byte
	legacyKey    []byte
	mu           sync.RWMutex
}

// NewEncryptedFileStore creates a new EncryptedFileStore rooted in configDir.
//
// Construction always succeeds — a fresh install (no passphrase, no legacy
// file) yields a store whose reads return ErrSecretNotFound and whose writes
// fail with a clear "set NYLAS_FILE_STORE_PASSPHRASE" error. This lets
// callers probe an empty store without crashing.
//
// To actually persist secrets, set NYLAS_FILE_STORE_PASSPHRASE to a strong
// value, or run `nylas auth config` to switch to the system keyring.
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
		// Enforce a minimum length so a 4-character passphrase isn't held
		// up as adequate defense. This is a deliberately gentle floor (12
		// characters) — long enough to make offline brute-force materially
		// expensive when combined with Argon2id, short enough that real
		// users can comply.
		if len(value) < minPassphraseLen {
			return nil, fmt.Errorf(
				"%s must be at least %d characters (got %d) — pick a longer passphrase",
				fileStorePassphraseEnv, minPassphraseLen, len(value),
			)
		}
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
//
// Holds the exclusive lock — not RLock — because loadSecrets→decrypt may
// run migrateToPassphrase on legacy data, which writes BOTH .secrets.salt
// and .secrets.enc. Two concurrent first-readers under RLock can interleave
// those writes and leave a salt/ciphertext pair that no longer decrypts.
// CLI workloads aren't read-heavy, so serializing reads is the right
// trade for guaranteed migration correctness.
func (f *EncryptedFileStore) Get(key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

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

// detectKeyType returns which key(s) can currently decrypt the on-disk file.
// It reads the file once and probes each key in order.  If the file does not
// exist, fileStoreKeyNone is returned with no error.
func (f *EncryptedFileStore) detectKeyType() (fileStoreKeyType, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fileStoreKeyNone, nil
		}
		return fileStoreKeyNone, err
	}

	hasPassphrase := false
	if key, err := f.passphraseKey(false); err == nil {
		if _, err := decryptWithKey(key, data); err == nil {
			hasPassphrase = true
		}
		zeroBytes(key)
	}

	hasLegacy := f.canDecryptWithLegacyKeys(data)

	switch {
	case hasPassphrase && hasLegacy:
		return fileStoreKeyBoth, nil
	case hasPassphrase:
		return fileStoreKeyPassphraseOnly, nil
	case hasLegacy:
		return fileStoreKeyLegacyOnly, nil
	default:
		return fileStoreKeyNone, nil
	}
}

// canDecryptWithLegacyKeys returns true when the ciphertext can be opened by
// either the migration master key or the legacy machine-derived key.
func (f *EncryptedFileStore) canDecryptWithLegacyKeys(data []byte) bool {
	if len(f.migrationKey) > 0 {
		if _, err := decryptWithKey(f.migrationKey, data); err == nil {
			return true
		}
	}
	if len(f.legacyKey) > 0 {
		if _, err := decryptWithKey(f.legacyKey, data); err == nil {
			return true
		}
	}
	return false
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

// saveSecrets encrypts and saves the secrets file atomically.
func (f *EncryptedFileStore) saveSecrets(secrets map[string]string) error {
	plaintext, err := json.Marshal(secrets)
	if err != nil {
		return err
	}

	ciphertext, err := f.encrypt(plaintext)
	if err != nil {
		return err
	}

	// Ensure directory exists.
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Atomic write: write to a sibling temp file, then rename.
	tmp, err := os.CreateTemp(dir, ".secrets.enc.tmp.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	// Clean up the temp file on any failure path.
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(ciphertext); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, f.path); err != nil {
		return err
	}
	committed = true

	// Remove the plaintext migration key once the store has been rewritten.
	if f.keyPath != "" {
		_ = os.Remove(f.keyPath)
	}

	return nil
}

// encrypt encrypts plaintext using AES-256-GCM.
//
// Passphrase rules enforced here:
//   - If passphrase is set: encrypt with the passphrase-derived key.
//   - If passphrase is unset AND a legacy file exists: refuse — the caller must
//     set NYLAS_FILE_STORE_PASSPHRASE so the migration path in decrypt can run first.
//   - If passphrase is unset AND no legacy file: refuse — this is a new install
//     that should never have been constructed (NewEncryptedFileStore checks this).
func (f *EncryptedFileStore) encrypt(plaintext []byte) ([]byte, error) {
	if len(f.passphrase) == 0 {
		// Distinguish between "legacy file exists" and "fresh install" for clearer errors.
		if _, statErr := os.Stat(f.path); statErr == nil {
			return nil, fmt.Errorf(
				"encrypted file store requires %s to migrate from the legacy machine-derived key. "+
					"Set it and re-run, or run `nylas auth config` to switch to the system keyring",
				fileStorePassphraseEnv,
			)
		}
		return nil, fmt.Errorf(
			"%s must be set to use the encrypted file secret store. "+
				"Set it and re-run, or run `nylas auth config` to switch to the system keyring",
			fileStorePassphraseEnv,
		)
	}

	key, err := f.passphraseKey(true)
	if err != nil {
		return nil, err
	}
	defer zeroBytes(key)
	return encryptWithKey(key, plaintext)
}

// decrypt decrypts ciphertext using AES-256-GCM.
//
// Decryption order:
//  1. Passphrase key (if passphrase is set) — normal path.
//  2. Legacy key (migration master key or machine-derived key):
//     - If passphrase is NOT set: return an error requiring the user to set it.
//     - If passphrase IS set: re-encrypt with the passphrase key (one-shot
//     migration), print a notice to stderr, and return the plaintext.
//  3. Neither key works: return an error.
func (f *EncryptedFileStore) decrypt(data []byte) ([]byte, error) {
	// 1. Try passphrase key first.
	if len(f.passphrase) > 0 {
		if key, err := f.passphraseKey(false); err == nil {
			plaintext, decErr := decryptWithKey(key, data)
			zeroBytes(key)
			if decErr == nil {
				return plaintext, nil
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		// Passphrase set but salt missing or passphrase wrong — fall through to legacy.
	}

	// 2. Try legacy keys.
	if plaintext, legacyKey, ok := f.tryLegacyDecrypt(data); ok {
		_ = legacyKey // used only for the migration path below
		if len(f.passphrase) == 0 {
			// Legacy decryption succeeded but no passphrase — block to force migration.
			return nil, fmt.Errorf(
				"encrypted file store requires %s to migrate from the legacy machine-derived key. "+
					"Set it and re-run, or run `nylas auth config` to switch to the system keyring",
				fileStorePassphraseEnv,
			)
		}

		// Passphrase is available — perform one-shot migration.
		if migrateErr := f.migrateToPassphrase(plaintext); migrateErr != nil {
			// Migration failed (e.g. disk write error). Return the plaintext so
			// the caller's operation still succeeds; the legacy file remains intact.
			fmt.Fprintf(os.Stderr, "warning: failed to migrate encrypted file store: %v\n", migrateErr)
		} else {
			fmt.Fprintf(os.Stderr,
				"notice: encrypted file store migrated to passphrase-derived key (%s)\n",
				fileStorePassphraseEnv)
		}
		return plaintext, nil
	}

	// 3. Nothing worked.
	if len(f.passphrase) == 0 {
		return nil, fmt.Errorf(
			"%s must be set to unlock the encrypted file store",
			fileStorePassphraseEnv,
		)
	}
	return nil, fmt.Errorf("failed to decrypt encrypted file store with the configured passphrase")
}

// tryLegacyDecrypt attempts decryption with the migration master key first,
// then the machine-derived legacy key.  Returns the plaintext, the key used,
// and whether decryption succeeded.
func (f *EncryptedFileStore) tryLegacyDecrypt(data []byte) (plaintext []byte, key []byte, ok bool) {
	if len(f.migrationKey) > 0 {
		if pt, err := decryptWithKey(f.migrationKey, data); err == nil {
			return pt, f.migrationKey, true
		}
	}
	if len(f.legacyKey) > 0 {
		if pt, err := decryptWithKey(f.legacyKey, data); err == nil {
			return pt, f.legacyKey, true
		}
	}
	return nil, nil, false
}

// migrateToPassphrase re-encrypts plaintext with the passphrase-derived key and
// atomically writes it to disk.  If this fails, the on-disk legacy file is left
// intact so the next invocation can retry.
func (f *EncryptedFileStore) migrateToPassphrase(plaintext []byte) error {
	ciphertext, err := f.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("re-encrypt for migration: %w", err)
	}

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir for migration: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".secrets.enc.tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file for migration: %w", err)
	}
	tmpPath := tmp.Name()

	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file for migration: %w", err)
	}
	if _, err := tmp.Write(ciphertext); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file for migration: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file for migration: %w", err)
	}
	if err := os.Rename(tmpPath, f.path); err != nil {
		return fmt.Errorf("rename temp file for migration: %w", err)
	}
	committed = true

	// Remove the plaintext migration master key now that re-encryption succeeded.
	if f.keyPath != "" {
		_ = os.Remove(f.keyPath)
	}

	return nil
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

// argon2id parameters. The OWASP 2024 guidance is t=2, m=19MiB, p=1 as
// the absolute minimum for password storage; modern hosts comfortably
// support t=3, m=64MiB, p=4 for a CLI use case where derive happens once
// per process. Increasing t from 1 (the previous setting) to 3 raises
// offline-cracking cost ~3x for any attacker who exfiltrates the salt and
// ciphertext.
const (
	argon2idTime    uint32 = 3
	argon2idMemory  uint32 = 64 * 1024 // 64 MiB
	argon2idThreads uint8  = 4
	argon2idKeyLen  uint32 = 32

	// minPassphraseLen is the minimum length we accept for
	// NYLAS_FILE_STORE_PASSPHRASE. Argon2id alone cannot save a 4-character
	// passphrase from offline brute-force.
	minPassphraseLen = 12
)

func derivePassphraseKey(passphrase, salt []byte) []byte {
	// Argon2id keeps the fallback store bound to user-supplied secret material
	// instead of host metadata while staying fast enough for CLI use.
	return argon2.IDKey(passphrase, salt, argon2idTime, argon2idMemory, argon2idThreads, argon2idKeyLen)
}
