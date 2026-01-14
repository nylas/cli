package common

import (
	"github.com/nylas/cli/internal/adapters/client"
	"github.com/nylas/cli/internal/ports"
)

// defaultFactory is the global client factory instance.
// Using a factory allows for proper dependency injection and testability.
var defaultFactory ports.ClientFactory = client.NewFactory()

// GetClientFactory returns the global client factory.
// This is the preferred way to access client creation functionality,
// following the hexagonal architecture pattern.
func GetClientFactory() ports.ClientFactory {
	return defaultFactory
}

// SetClientFactory sets a custom client factory (useful for testing).
func SetClientFactory(factory ports.ClientFactory) {
	defaultFactory = factory
}

// GetNylasClient creates a Nylas API client with credentials from environment variables or keyring.
// It checks credentials in this order:
// 1. Environment variables (NYLAS_API_KEY, NYLAS_CLIENT_ID, NYLAS_CLIENT_SECRET) - highest priority
// 2. System keyring (if available and env vars not set)
// 3. Encrypted file store (if keyring unavailable)
//
// This allows the CLI to work in multiple environments:
// - CI/CD pipelines (environment variables)
// - Docker containers (environment variables)
// - Integration tests (environment variables with NYLAS_DISABLE_KEYRING=true)
// - Local development (keyring)
//
// For better testability, consider using GetClientFactory().CreateClient() instead.
func GetNylasClient() (ports.NylasClient, error) {
	return defaultFactory.CreateClient()
}

// GetCachedNylasClient returns a singleton Nylas client.
// This is useful for CLI commands that need to make multiple API calls
// in a single command invocation, avoiding the overhead of creating
// a new client each time.
//
// Note: The client is cached for the lifetime of the process.
// For long-running processes or tests, use GetNylasClient() instead.
//
// For better testability, consider using GetClientFactory().GetCachedClient() instead.
func GetCachedNylasClient() (ports.NylasClient, error) {
	return defaultFactory.GetCachedClient()
}

// ResetCachedClient clears the cached client (useful for testing).
//
// For better testability, consider using GetClientFactory().ResetCache() instead.
func ResetCachedClient() {
	defaultFactory.ResetCache()
}

// GetAPIKey returns the API key from environment variable or keyring.
// It checks in this order:
// 1. Environment variable (NYLAS_API_KEY) - highest priority
// 2. System keyring (if available)
// 3. Encrypted file store (if keyring unavailable)
//
// For better testability, consider using GetClientFactory().GetAPIKey() instead.
func GetAPIKey() (string, error) {
	return defaultFactory.GetAPIKey()
}

// GetGrantID returns the grant ID from arguments, environment variable, or keyring.
// It checks in this order:
// 1. Command line argument (if provided) - supports email lookup if arg contains "@"
// 2. Environment variable (NYLAS_GRANT_ID)
// 3. Stored default grant (from keyring/file)
//
// For better testability, consider using GetClientFactory().GetGrantID() instead.
func GetGrantID(args []string) (string, error) {
	return defaultFactory.GetGrantID(args)
}
