//go:build !integration

package client

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

func TestFactory_CreateClient_WithEnvVar(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origClientID := os.Getenv("NYLAS_CLIENT_ID")
	origClientSecret := os.Getenv("NYLAS_CLIENT_SECRET")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	t.Cleanup(func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_CLIENT_ID", origClientID)
		setEnvOrUnset("NYLAS_CLIENT_SECRET", origClientSecret)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	})

	// Set test env vars
	_ = os.Setenv("NYLAS_API_KEY", "test-api-key-12345")
	_ = os.Setenv("NYLAS_CLIENT_ID", "test-client-id")
	_ = os.Setenv("NYLAS_CLIENT_SECRET", "test-client-secret")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	factory := NewFactory()
	client, err := factory.CreateClient()

	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestFactory_CreateClient_NoAPIKey(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfig := os.Getenv("XDG_CONFIG_HOME")

	// Use temp dir to isolate from any stored credentials
	tempDir := t.TempDir()

	// Restore after test
	t.Cleanup(func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfig)
	})

	// Clear API key and disable keyring, use empty temp config dir
	_ = os.Unsetenv("NYLAS_API_KEY")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

	factory := NewFactoryWithConfigDir(tempDir)
	client, err := factory.CreateClient()

	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "API key not configured")
}

func TestFactory_GetAPIKey_WithEnvVar(t *testing.T) {
	// Save original env var
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")

	// Restore after test
	t.Cleanup(func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
	})

	// Set test env var
	testKey := "test-api-key-67890"
	_ = os.Setenv("NYLAS_API_KEY", testKey)
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")

	factory := NewFactory()
	apiKey, err := factory.GetAPIKey()

	require.NoError(t, err)
	assert.Equal(t, testKey, apiKey)
}

func TestFactory_GetAPIKey_NoAPIKey(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")
	origDisableKeyring := os.Getenv("NYLAS_DISABLE_KEYRING")
	origXDGConfig := os.Getenv("XDG_CONFIG_HOME")

	// Use temp dir to isolate from any stored credentials
	tempDir := t.TempDir()

	// Restore after test
	t.Cleanup(func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
		setEnvOrUnset("NYLAS_DISABLE_KEYRING", origDisableKeyring)
		setEnvOrUnset("XDG_CONFIG_HOME", origXDGConfig)
	})

	// Clear API key and disable keyring, use empty temp config dir
	_ = os.Unsetenv("NYLAS_API_KEY")
	_ = os.Setenv("NYLAS_DISABLE_KEYRING", "true")
	_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

	factory := NewFactoryWithConfigDir(tempDir)
	apiKey, err := factory.GetAPIKey()

	assert.Error(t, err)
	assert.Empty(t, apiKey)
	assert.Contains(t, err.Error(), "API key not configured")
}

func TestFactory_GetCachedClient(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")

	// Restore after test
	t.Cleanup(func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
	})

	_ = os.Setenv("NYLAS_API_KEY", "test-key")

	factory := NewFactory()

	// First call should create client
	client1, err1 := factory.GetCachedClient()
	require.NoError(t, err1)
	require.NotNil(t, client1)

	// Second call should return same client
	client2, err2 := factory.GetCachedClient()
	require.NoError(t, err2)
	assert.Equal(t, client1, client2) // Same instance
}

func TestFactory_ResetCache(t *testing.T) {
	// Save original env vars
	origAPIKey := os.Getenv("NYLAS_API_KEY")

	// Restore after test
	t.Cleanup(func() {
		setEnvOrUnset("NYLAS_API_KEY", origAPIKey)
	})

	_ = os.Setenv("NYLAS_API_KEY", "test-key")

	factory := NewFactory()

	// Get cached client
	client1, _ := factory.GetCachedClient()

	// Reset cache
	factory.ResetCache()

	// Next call should create new client
	client2, _ := factory.GetCachedClient()

	// Pointers should be different (new instance)
	assert.NotSame(t, client1, client2)
}

// setEnvOrUnset sets an environment variable if value is non-empty, otherwise unsets it
func setEnvOrUnset(key, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	} else {
		_ = os.Unsetenv(key)
	}
}
