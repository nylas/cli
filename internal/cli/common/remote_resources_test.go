//go:build !integration

package common

import (
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGrantIdentifier_WithEmail(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "nylas")
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(configDir))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase")
	t.Setenv("NYLAS_API_KEY", "")
	t.Setenv("NYLAS_GRANT_ID", "")

	grantStore, err := NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SaveGrant(domain.GrantInfo{
		ID:    "grant-123",
		Email: "user@example.com",
	}))

	grantID, err := ResolveGrantIdentifier("user@example.com")

	require.NoError(t, err)
	assert.Equal(t, "grant-123", grantID)
}

func TestResolveGrantIdentifier_WithEmailIgnoresEnvGrantFallback(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "nylas")
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(configDir))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase")
	t.Setenv("NYLAS_API_KEY", "")
	t.Setenv("NYLAS_GRANT_ID", "env-default-grant")

	grantStore, err := NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SaveGrant(domain.GrantInfo{
		ID:    "grant-email",
		Email: "lookup@example.com",
	}))

	grantID, err := ResolveGrantIdentifier("lookup@example.com")

	require.NoError(t, err)
	assert.Equal(t, "grant-email", grantID)
}

func TestResolveScopeGrantID_GrantScopeUsesGrantLookup(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "nylas")
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(configDir))
	t.Setenv("HOME", t.TempDir())
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "test-file-store-passphrase")
	t.Setenv("NYLAS_API_KEY", "")
	t.Setenv("NYLAS_GRANT_ID", "")

	grantStore, err := NewDefaultGrantStore()
	require.NoError(t, err)
	require.NoError(t, grantStore.SaveGrant(domain.GrantInfo{
		ID:    "grant-456",
		Email: "grant@example.com",
	}))

	grantID, err := ResolveScopeGrantID(domain.ScopeGrant, "grant@example.com")

	require.NoError(t, err)
	assert.Equal(t, "grant-456", grantID)
}

func TestResolveScopeGrantID_AuditGrantHook(t *testing.T) {
	originalHook := AuditGrantHook
	t.Cleanup(func() {
		AuditGrantHook = originalHook
	})

	var hookedGrantID string
	AuditGrantHook = func(grantID string) {
		hookedGrantID = grantID
	}

	grantID, err := ResolveScopeGrantID(domain.ScopeGrant, "grant-789")

	require.NoError(t, err)
	assert.Equal(t, "grant-789", grantID)
	assert.Equal(t, "grant-789", hookedGrantID)
}

func TestResolveScopeGrantID_AppScopeSkipsAuditGrantHook(t *testing.T) {
	originalHook := AuditGrantHook
	t.Cleanup(func() {
		AuditGrantHook = originalHook
	})

	called := false
	AuditGrantHook = func(string) {
		called = true
	}

	grantID, err := ResolveScopeGrantID(domain.ScopeApplication, "")

	require.NoError(t, err)
	assert.Empty(t, grantID)
	assert.False(t, called)
}

func TestResolveScopeGrantID_AppScopeRejectsGrantID(t *testing.T) {
	grantID, err := ResolveScopeGrantID(domain.ScopeApplication, "grant-789")

	require.Error(t, err)
	assert.Empty(t, grantID)
	assert.Contains(t, err.Error(), "`--grant-id` requires `--scope grant`")
}
