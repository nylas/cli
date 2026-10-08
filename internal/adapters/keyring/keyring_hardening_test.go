package keyring

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gokeyring "github.com/zalando/go-keyring"
)

func TestSystemKeyring_ReadRetriesWhenTheValueIsReplacedMidRead(t *testing.T) {
	// Another process replaced the value after this read fetched its header,
	// deleting the chunks the header names. The read must follow the new
	// header, not report the secret as broken.
	useSizeLimitedKeychain(t)
	kr := NewSystemKeyring()
	require.NoError(t, kr.Set("k", strings.Repeat("a", 5000)))
	replacement := strings.Repeat("b", 5000)

	replaced := false
	keyringGet = func(service, key string) (string, error) {
		value, err := gokeyring.Get(service, key)
		if !replaced && strings.Contains(key, ".chunk.") {
			replaced = true
			require.NoError(t, kr.Set("k", replacement))
			return gokeyring.Get(service, key) // the chunk is gone now
		}
		return value, err
	}
	t.Cleanup(func() { keyringGet = gokeyring.Get })

	got, err := kr.Get("k")

	require.NoError(t, err)
	assert.Equal(t, replacement, got)
}

func TestEncryptedFileStore_ReadsFromAReadOnlyConfigDirectory(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX directory permissions enforced")
	}
	dir := t.TempDir()
	setFileStorePassphrase(t)
	store, err := NewEncryptedFileStore(dir)
	require.NoError(t, err)
	require.NoError(t, store.Set("api_key", "secret"))
	require.NoError(t, os.Remove(filepath.Join(dir, ".secrets.lock")))
	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	got, err := store.Get("api_key")
	require.NoError(t, err)
	assert.Equal(t, "secret", got)

	_, err = store.Get("missing")
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	assert.Error(t, store.Set("api_key", "other"), "a write still needs the lock")
}

func TestEncryptedFileStore_NothingStoredNeedsNoLockFile(t *testing.T) {
	dir := t.TempDir()
	setFileStorePassphrase(t)
	store, err := NewEncryptedFileStore(dir)
	require.NoError(t, err)

	_, err = store.Get("api_key")

	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	assert.NoFileExists(t, filepath.Join(dir, ".secrets.lock"))
}
