// Package keyring provides secure credential storage using the OS keychain.
package keyring

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

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

// Keychains cap the size of one item: Windows Credential Manager at 2560
// bytes, and the macOS `security` command at 4096 for the whole command line.
// A token can outgrow that, so a value longer than keychainChunkSize is split
// across several items and the key holds a header naming them.
const (
	keychainChunkSize   = 2000
	keychainChunkPrefix = "nylas:chunked:v1:"
	keychainMaxChunks   = 64
)

// Set stores a secret value for the given key.
//
// A long value's chunks are written under a fresh generation id before the
// header that points at them, so a reader never pairs a header with chunks
// from another write. The previous generation is removed afterwards.
func (k *SystemKeyring) Set(key, value string) error {
	previous, _ := keyring.Get(serviceName, key)

	if len(value) <= keychainChunkSize && !strings.HasPrefix(value, keychainChunkPrefix) {
		if err := keyring.Set(serviceName, key, value); err != nil {
			return err
		}
		deleteChunks(key, previous)
		return nil
	}

	parts := splitChunks(value)
	if len(parts) > keychainMaxChunks {
		return fmt.Errorf("secret %s is too large for the system keyring (%d bytes)", key, len(value))
	}
	gen, err := newChunkGeneration()
	if err != nil {
		return err
	}
	for i, part := range parts {
		if err := keyring.Set(serviceName, chunkKey(key, gen, i), part); err != nil {
			deleteChunks(key, chunkHeader(gen, i))
			return err
		}
	}
	if err := keyring.Set(serviceName, key, chunkHeader(gen, len(parts))); err != nil {
		deleteChunks(key, chunkHeader(gen, len(parts)))
		return err
	}
	deleteChunks(key, previous)
	return nil
}

// Get retrieves a secret value for the given key.
//
// A chunked value is a header and its chunks, read separately. Another
// process can replace the value in between, removing the chunks this read is
// about to fetch, so a missing chunk re-reads the header and tries again
// while it keeps changing.
func (k *SystemKeyring) Get(key string) (string, error) {
	var lastErr error
	for attempt := 0; attempt < chunkReadAttempts; attempt++ {
		value, err := keyringGet(serviceName, key)
		if errors.Is(err, keyring.ErrNotFound) {
			return "", domain.ErrSecretNotFound
		}
		if err != nil {
			return "", err
		}
		gen, n, ok := parseChunkHeader(value)
		if !ok {
			return value, nil
		}
		joined, err := readChunks(key, gen, n)
		if err == nil {
			return joined, nil
		}
		lastErr = err
		if current, _ := keyringGet(serviceName, key); current == value {
			break // The header did not move: the chunk is really gone.
		}
	}
	// The value was removed by hand, or kept changing under this read.
	return "", fmt.Errorf("secret %s is incomplete in the system keyring: %w", key, lastErr)
}

// keyringGet is keyring.Get, replaceable so a test can rewrite a value in the
// middle of a chunked read.
var keyringGet = keyring.Get

// chunkReadAttempts bounds Get's retries while another process rewrites a
// chunked value.
const chunkReadAttempts = 3

func readChunks(key, gen string, n int) (string, error) {
	var b strings.Builder
	for i := range n {
		part, err := keyringGet(serviceName, chunkKey(key, gen, i))
		if err != nil {
			return "", err
		}
		b.WriteString(part)
	}
	return b.String(), nil
}

// Delete removes a secret for the given key.
func (k *SystemKeyring) Delete(key string) error {
	previous, _ := keyring.Get(serviceName, key)
	err := keyring.Delete(serviceName, key)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return err
	}
	deleteChunks(key, previous)
	return nil
}

func splitChunks(value string) []string {
	var parts []string
	for len(value) > keychainChunkSize {
		parts = append(parts, value[:keychainChunkSize])
		value = value[keychainChunkSize:]
	}
	return append(parts, value)
}

func newChunkGeneration() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate a keyring chunk id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func chunkKey(key, gen string, i int) string {
	return fmt.Sprintf("%s.chunk.%s.%d", key, gen, i)
}

func chunkHeader(gen string, n int) string {
	return fmt.Sprintf("%s%s:%d", keychainChunkPrefix, gen, n)
}

func parseChunkHeader(value string) (gen string, n int, ok bool) {
	rest, found := strings.CutPrefix(value, keychainChunkPrefix)
	if !found {
		return "", 0, false
	}
	gen, count, found := strings.Cut(rest, ":")
	if !found || len(gen) != 16 {
		return "", 0, false
	}
	if _, err := hex.DecodeString(gen); err != nil {
		return "", 0, false
	}
	n, err := strconv.Atoi(count)
	if err != nil || n < 1 || n > keychainMaxChunks {
		return "", 0, false
	}
	return gen, n, true
}

// deleteChunks removes the chunks a header points at, best effort: a leftover
// chunk is unreachable once no header names it.
func deleteChunks(key, header string) {
	gen, n, ok := parseChunkHeader(header)
	if !ok {
		return
	}
	for i := range n {
		_ = keyring.Delete(serviceName, chunkKey(key, gen, i))
	}
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
