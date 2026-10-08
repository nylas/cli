package keyring

import (
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gokeyring "github.com/zalando/go-keyring"
)

// useSizeLimitedKeychain swaps in go-keyring's in-memory provider. It has no
// size limit of its own, so the tests check each stored item against the
// smallest real one (Windows Credential Manager, 2560 bytes) directly.
func useSizeLimitedKeychain(t *testing.T) {
	t.Helper()
	gokeyring.MockInit()
	t.Cleanup(gokeyring.MockInit)
}

func TestSystemKeyring_StoresAValueLargerThanOneKeychainItem(t *testing.T) {
	useSizeLimitedKeychain(t)
	kr := NewSystemKeyring()
	value := strings.Repeat("eyJhbGciOiJFZERTQSJ9.", 300) // ~6.6 KB, like a large JWT

	require.NoError(t, kr.Set("oauth_access_token", value))

	got, err := kr.Get("oauth_access_token")
	require.NoError(t, err)
	assert.Equal(t, value, got)

	header, err := gokeyring.Get(serviceName, "oauth_access_token")
	require.NoError(t, err)
	gen, n, ok := parseChunkHeader(header)
	require.True(t, ok, "the key holds a header, not the value")
	for i := range n {
		part, err := gokeyring.Get(serviceName, chunkKey("oauth_access_token", gen, i))
		require.NoError(t, err)
		assert.LessOrEqual(t, len(part), 2560, "every item fits the smallest keychain limit")
	}
}

func TestSystemKeyring_ReplacingALargeValueRemovesItsChunks(t *testing.T) {
	useSizeLimitedKeychain(t)
	kr := NewSystemKeyring()
	require.NoError(t, kr.Set("k", strings.Repeat("a", 5000)))
	header, _ := gokeyring.Get(serviceName, "k")
	gen, _, _ := parseChunkHeader(header)

	require.NoError(t, kr.Set("k", "short"))

	got, err := kr.Get("k")
	require.NoError(t, err)
	assert.Equal(t, "short", got)
	_, err = gokeyring.Get(serviceName, chunkKey("k", gen, 0))
	assert.ErrorIs(t, err, gokeyring.ErrNotFound)
}

func TestSystemKeyring_ReplacingALargeValueWithAnotherUsesNewChunks(t *testing.T) {
	useSizeLimitedKeychain(t)
	kr := NewSystemKeyring()
	require.NoError(t, kr.Set("k", strings.Repeat("a", 5000)))
	first, _ := gokeyring.Get(serviceName, "k")

	second := strings.Repeat("b", 4100)
	require.NoError(t, kr.Set("k", second))

	got, err := kr.Get("k")
	require.NoError(t, err)
	assert.Equal(t, second, got)
	oldGen, _, _ := parseChunkHeader(first)
	_, err = gokeyring.Get(serviceName, chunkKey("k", oldGen, 0))
	assert.ErrorIs(t, err, gokeyring.ErrNotFound)
}

func TestSystemKeyring_DeleteRemovesTheChunks(t *testing.T) {
	useSizeLimitedKeychain(t)
	kr := NewSystemKeyring()
	require.NoError(t, kr.Set("k", strings.Repeat("a", 5000)))
	header, _ := gokeyring.Get(serviceName, "k")
	gen, n, _ := parseChunkHeader(header)

	require.NoError(t, kr.Delete("k"))

	_, err := kr.Get("k")
	assert.ErrorIs(t, err, domain.ErrSecretNotFound)
	for i := range n {
		_, err := gokeyring.Get(serviceName, chunkKey("k", gen, i))
		assert.ErrorIs(t, err, gokeyring.ErrNotFound)
	}
}

func TestSystemKeyring_AMissingChunkIsAnErrorNotATruncatedValue(t *testing.T) {
	useSizeLimitedKeychain(t)
	kr := NewSystemKeyring()
	require.NoError(t, kr.Set("k", strings.Repeat("a", 5000)))
	header, _ := gokeyring.Get(serviceName, "k")
	gen, _, _ := parseChunkHeader(header)
	require.NoError(t, gokeyring.Delete(serviceName, chunkKey("k", gen, 1)))

	_, err := kr.Get("k")

	require.Error(t, err)
	assert.NotErrorIs(t, err, domain.ErrSecretNotFound)
}

func TestSystemKeyring_AShortValueThatLooksLikeAHeaderRoundTrips(t *testing.T) {
	useSizeLimitedKeychain(t)
	kr := NewSystemKeyring()
	value := keychainChunkPrefix + "0123456789abcdef:1"

	require.NoError(t, kr.Set("k", value))

	got, err := kr.Get("k")
	require.NoError(t, err)
	assert.Equal(t, value, got)
}
