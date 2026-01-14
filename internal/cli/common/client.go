package common

import (
	"fmt"
	"os"
	"sync"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

var (
	cachedClient ports.NylasClient
	clientOnce   sync.Once
	clientErr    error
)

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
func GetNylasClient() (ports.NylasClient, error) {
	// Load configuration
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err != nil {
		cfg = &domain.Config{Region: "us"}
	}

	// First, check environment variables (highest priority)
	apiKey := os.Getenv("NYLAS_API_KEY")
	clientID := os.Getenv("NYLAS_CLIENT_ID")
	clientSecret := os.Getenv("NYLAS_CLIENT_SECRET")

	// If API key not in env, try keyring/file store
	if apiKey == "" {
		secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
		if err == nil {
			apiKey, _ = secretStore.Get(ports.KeyAPIKey)
			if clientID == "" {
				clientID, _ = secretStore.Get(ports.KeyClientID)
			}
			if clientSecret == "" {
				clientSecret, _ = secretStore.Get(ports.KeyClientSecret)
			}
		}
	}

	// Validate that we have at least the API key
	if apiKey == "" {
		return nil, fmt.Errorf("API key not configured. Set NYLAS_API_KEY environment variable or run 'nylas auth config'")
	}

	// Create and configure the HTTP client
	c := nylas.NewHTTPClient()
	c.SetRegion(cfg.Region)
	c.SetCredentials(clientID, clientSecret, apiKey)

	return c, nil
}

// GetCachedNylasClient returns a singleton Nylas client.
// This is useful for CLI commands that need to make multiple API calls
// in a single command invocation, avoiding the overhead of creating
// a new client each time.
//
// Note: The client is cached for the lifetime of the process.
// For long-running processes or tests, use GetNylasClient() instead.
func GetCachedNylasClient() (ports.NylasClient, error) {
	clientOnce.Do(func() {
		cachedClient, clientErr = GetNylasClient()
	})
	return cachedClient, clientErr
}

// ResetCachedClient clears the cached client (useful for testing).
func ResetCachedClient() {
	cachedClient = nil
	clientErr = nil
	clientOnce = sync.Once{}
}

// GetAPIKey returns the API key from environment variable or keyring.
// It checks in this order:
// 1. Environment variable (NYLAS_API_KEY) - highest priority
// 2. System keyring (if available)
// 3. Encrypted file store (if keyring unavailable)
func GetAPIKey() (string, error) {
	// First check environment variable (highest priority)
	apiKey := os.Getenv("NYLAS_API_KEY")

	// If not in env, try keyring/file store
	if apiKey == "" {
		secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
		if err == nil {
			apiKey, _ = secretStore.Get(ports.KeyAPIKey)
		}
	}

	if apiKey == "" {
		return "", fmt.Errorf("API key not configured. Set NYLAS_API_KEY environment variable or run 'nylas auth config'")
	}

	return apiKey, nil
}

// GetGrantID returns the grant ID from arguments, environment variable, or keyring.
// It checks in this order:
// 1. Command line argument (if provided) - supports email lookup if arg contains "@"
// 2. Environment variable (NYLAS_GRANT_ID)
// 3. Stored default grant (from keyring/file)
func GetGrantID(args []string) (string, error) {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		// Fall back to env var only if keyring unavailable
		if grantID := os.Getenv("NYLAS_GRANT_ID"); grantID != "" {
			return grantID, nil
		}
		return "", fmt.Errorf("couldn't access secret store and NYLAS_GRANT_ID not set: %w", err)
	}
	grantStore := keyring.NewGrantStore(secretStore)

	// If provided as argument
	if len(args) > 0 && args[0] != "" {
		identifier := args[0]

		// If it looks like an email, try to find by email
		if containsAt(identifier) {
			grant, err := grantStore.GetGrantByEmail(identifier)
			if err != nil {
				return "", fmt.Errorf("no grant found for email: %s", identifier)
			}
			return grant.ID, nil
		}

		// Otherwise treat as grant ID
		return identifier, nil
	}

	// Check environment variable
	if grantID := os.Getenv("NYLAS_GRANT_ID"); grantID != "" {
		return grantID, nil
	}

	// Try to get default grant
	grantID, err := grantStore.GetDefaultGrant()
	if err != nil {
		return "", fmt.Errorf("no grant ID provided. Specify grant ID as argument, set NYLAS_GRANT_ID, or use 'nylas auth list' to see available grants")
	}

	return grantID, nil
}

// containsAt checks if a string contains "@" (for email detection).
func containsAt(s string) bool {
	for _, c := range s {
		if c == '@' {
			return true
		}
	}
	return false
}
