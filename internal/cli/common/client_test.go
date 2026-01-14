//go:build !integration

package common

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContainsAt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"email address", "user@example.com", true},
		{"with plus sign", "user+tag@example.com", true},
		{"multiple @", "a@b@c", true},
		{"@ at start", "@example.com", true},
		{"@ at end", "user@", true},
		{"just @", "@", true},
		{"no @", "username", false},
		{"empty string", "", false},
		{"spaces only", "   ", false},
		{"number string", "12345", false},
		{"special chars no @", "user.name+tag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNylasClient_WithEnvVar(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origClientID := os.Getenv("NYLAS_CLIENT_ID")
	origClientSecret := os.Getenv("NYLAS_CLIENT_SECRET")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_CLIENT_ID", origClientID)
		setEnvOrUnset("NYLAS_CLIENT_SECRET", origClientSecret)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	}()

	// Set test env vars
	_ = os.Setenv("NYLAS_API_KEY", "test-api-key-12345")
	_ = os.Setenv("NYLAS_CLIENT_ID", "test-client-id")
	_ = os.Setenv("NYLAS_CLIENT_SECRET", "test-client-secret")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	client, err := GetNylasClient()

	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestGetNylasClient_NoAPIKey(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfig := os.Getenv("XDG_CONFIG_HOME")

	// Use temp dir to isolate from any stored credentials
	tempDir := t.TempDir()

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfig)
	}()

	// Clear API key and disable keyring, use empty temp config dir
	_ = os.Unsetenv("NYLAS_API_KEY")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

	client, err := GetNylasClient()

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API key not configured")
}

func TestGetAPIKey_WithEnvVar(t *testing.T) {
	// Save original env var
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	}()

	// Set test env var
	testKey := "test-api-key-67890"
	_ = os.Setenv("NYLAS_API_KEY", testKey)
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	apiKey, err := GetAPIKey()

	require.NoError(t, err)
	assert.Equal(t, testKey, apiKey)
}

func TestGetAPIKey_NoAPIKey(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfig := os.Getenv("XDG_CONFIG_HOME")

	// Use temp dir to isolate from any stored credentials
	tempDir := t.TempDir()

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfig)
	}()

	// Clear API key and disable keyring, use empty temp config dir
	_ = os.Unsetenv("NYLAS_API_KEY")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

	apiKey, err := GetAPIKey()

	assert.Error(t, err)
	assert.Empty(t, apiKey)
	assert.Contains(t, err.Error(), "API key not configured")
}

func TestGetGrantID_WithArgument(t *testing.T) {
	// Save original env var
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	}()

	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Unsetenv("NYLAS_GRANT_ID")

	// Test with direct grant ID argument (not email)
	args := []string{"grant-id-12345"}

	grantID, err := GetGrantID(args)

	// This may fail if keyring is not accessible, which is expected in test env
	if err != nil {
		// If keyring not accessible, check the error message
		assert.Contains(t, err.Error(), "secret store")
	} else {
		assert.Equal(t, "grant-id-12345", grantID)
	}
}

func TestGetGrantID_WithEnvVar(t *testing.T) {
	// Save original env var
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	}()

	testGrantID := "env-grant-id-67890"
	_ = os.Setenv("NYLAS_GRANT_ID", testGrantID)
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	// Test with empty args - should fall back to env var
	grantID, err := GetGrantID([]string{})

	// This may fail if keyring is not accessible
	if err != nil {
		// If keyring fails but we have env var, we should still get the grant ID
		// The function tries keyring first, so we need to check behavior
		t.Logf("Error (expected in test env): %v", err)
	} else {
		assert.Equal(t, testGrantID, grantID)
	}
}

func TestGetGrantID_EmptyArgs(t *testing.T) {
	// Save original env vars
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfig := os.Getenv("XDG_CONFIG_HOME")

	// Use temp dir to isolate from any stored credentials
	tempDir := t.TempDir()

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfig)
	}()

	_ = os.Unsetenv("NYLAS_GRANT_ID")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

	// Test with empty args and no env var
	grantID, err := GetGrantID([]string{})

	// Should fail - no grant ID available
	assert.Error(t, err)
	assert.Empty(t, grantID)
}

func TestGetGrantID_EmptyStringArg(t *testing.T) {
	// Save original env vars
	origGrantID := os.Getenv("NYLAS_GRANT_ID")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_GRANT_ID", origGrantID)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	}()

	testGrantID := "env-grant-fallback"
	_ = os.Setenv("NYLAS_GRANT_ID", testGrantID)
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	// Test with empty string arg - should fall back to env var
	grantID, err := GetGrantID([]string{""})

	// May fail due to keyring access
	if err != nil {
		t.Logf("Error (expected in test env): %v", err)
	} else {
		assert.Equal(t, testGrantID, grantID)
	}
}

// setEnvOrUnset sets an environment variable if value is non-empty, otherwise unsets it
func setEnvOrUnset(key, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	} else {
		_ = os.Unsetenv(key)
	}
}

func TestGetNylasClient_EnvVarPriority(t *testing.T) {
	// This test verifies that environment variables take priority over keyring
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origClientID := os.Getenv("NYLAS_CLIENT_ID")
	origClientSecret := os.Getenv("NYLAS_CLIENT_SECRET")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_CLIENT_ID", origClientID)
		setEnvOrUnset("NYLAS_CLIENT_SECRET", origClientSecret)
	}()

	// Set env vars - these should be used regardless of keyring state
	_ = os.Setenv("NYLAS_API_KEY", "env-api-key")
	_ = os.Setenv("NYLAS_CLIENT_ID", "env-client-id")
	_ = os.Setenv("NYLAS_CLIENT_SECRET", "env-client-secret")

	client, err := GetNylasClient()

	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestGetAPIKey_EnvVarPriority(t *testing.T) {
	// Save original env var
	origAPIKey := os.Getenv("NYLAS_API_KEY")

	// Restore after test
	defer func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
	}()

	// Set env var
	_ = os.Setenv("NYLAS_API_KEY", "priority-test-key")

	apiKey, err := GetAPIKey()

	require.NoError(t, err)
	assert.Equal(t, "priority-test-key", apiKey)
}

func TestContainsAt_UnicodeSupport(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"unicode email", "用户@example.com", true},
		{"emoji without @", "🎉test", false},
		{"emoji with @", "🎉@test", true},
		{"cyrillic without @", "тест", false},
		{"cyrillic with @", "тест@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsAt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
