package keyring

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestEncryptedFileStore_ConcurrentFirstReadMigration covers the race
// flagged in code review: when several callers hit Get on a legacy-only
// .secrets.enc, the first read triggers migration which writes BOTH
// .secrets.salt and .secrets.enc. Under the old RLock-based Get, two
// readers could interleave those writes and leave a salt that didn't
// match the on-disk ciphertext — silently bricking the store.
//
// The test launches N concurrent Get calls, then re-opens the store
// from disk and verifies the migrated salt+ciphertext are self-
// consistent and round-trip the original plaintext.
func TestEncryptedFileStore_ConcurrentFirstReadMigration(t *testing.T) {
	tmpDir := t.TempDir()
	passphrase := setFileStorePassphrase(t)

	// Seed a legacy-encrypted secrets file. Use the machine-derived
	// legacy key — the same path real installations would have come
	// from before passphrase support landed.
	legacyKey, err := deriveLegacyKey()
	if err != nil {
		t.Fatalf("deriveLegacyKey failed: %v", err)
	}
	const legacyValue = "legacy-value"
	legacyJSON := []byte(`{"api_key":"` + legacyValue + `"}`)
	legacyCiphertext, err := encryptWithKey(legacyKey, legacyJSON)
	if err != nil {
		t.Fatalf("encryptWithKey failed: %v", err)
	}
	secretsPath := filepath.Join(tmpDir, ".secrets.enc")
	if err := os.WriteFile(secretsPath, legacyCiphertext, 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	store, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewEncryptedFileStore failed: %v", err)
	}

	// Fan out — enough goroutines to make any race observable, gated
	// on a single barrier so they all hit Get in roughly the same
	// instant.
	const concurrency = 32
	var (
		wg      sync.WaitGroup
		barrier = make(chan struct{})
		mu      sync.Mutex
		results = make([]string, 0, concurrency)
		errs    = make([]error, 0)
	)
	wg.Add(concurrency)
	for range concurrency {
		go func() {
			defer wg.Done()
			<-barrier
			v, gerr := store.Get("api_key")
			mu.Lock()
			defer mu.Unlock()
			if gerr != nil {
				errs = append(errs, gerr)
				return
			}
			results = append(results, v)
		}()
	}
	close(barrier)
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("concurrent Get returned %d errors; first: %v", len(errs), errs[0])
	}
	if len(results) != concurrency {
		t.Fatalf("got %d results, want %d", len(results), concurrency)
	}
	for i, v := range results {
		if v != legacyValue {
			t.Fatalf("result[%d] = %q, want %q", i, v, legacyValue)
		}
	}

	// On-disk consistency check: salt and ciphertext must match.
	saltRaw, err := os.ReadFile(filepath.Join(tmpDir, ".secrets.salt"))
	if err != nil {
		t.Fatalf("read .secrets.salt: %v", err)
	}
	salt, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(saltRaw)))
	if err != nil {
		t.Fatalf("decode salt: %v", err)
	}
	ciphertext, err := os.ReadFile(secretsPath)
	if err != nil {
		t.Fatalf("read .secrets.enc: %v", err)
	}
	plaintext, err := decryptWithKey(derivePassphraseKey([]byte(passphrase), salt), ciphertext)
	if err != nil {
		t.Fatalf("salt/ciphertext mismatch — store would be unrecoverable: %v", err)
	}
	if !strings.Contains(string(plaintext), legacyValue) {
		t.Fatalf("decrypted plaintext %q missing %q", string(plaintext), legacyValue)
	}

	// And the migrated ciphertext must NOT decrypt with the legacy key
	// any more — proves the migration actually flipped the encryption.
	if _, err := decryptWithKey(legacyKey, ciphertext); err == nil {
		t.Fatal("migrated ciphertext still decrypts with the legacy key")
	}

	// Fresh store from the same dir should also Get the value, proving
	// the persisted state is openable from a cold start (not just from
	// the in-process store that performed the migration).
	fresh, err := NewEncryptedFileStore(tmpDir)
	if err != nil {
		t.Fatalf("re-open NewEncryptedFileStore: %v", err)
	}
	v, err := fresh.Get("api_key")
	if err != nil {
		t.Fatalf("fresh.Get after migration: %v", err)
	}
	if v != legacyValue {
		t.Fatalf("fresh.Get = %q, want %q", v, legacyValue)
	}
}
