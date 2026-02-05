package common

import (
	"context"
	"fmt"
	"os"
	"strings"
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

	// AuditGrantHook is called when a grant ID is resolved (set by cli package).
	AuditGrantHook func(grantID string)
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
		return nil, NewUserErrorWithSuggestions(
			"API key not configured",
			"Configure with: nylas auth config",
			"Or use environment variable: export NYLAS_API_KEY=<your-key>",
			"Get your API key from: https://dashboard.nylas.com",
		)
	}

	// Create and configure the HTTP client
	c := nylas.NewHTTPClient()

	// Apply API configuration
	if cfg.API != nil && cfg.API.BaseURL != "" {
		c.SetBaseURL(cfg.API.BaseURL)
	} else {
		c.SetRegion(cfg.Region)
	}

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
		return "", NewUserErrorWithSuggestions(
			"API key not configured",
			"Configure with: nylas auth config",
			"Or use environment variable: export NYLAS_API_KEY=<your-key>",
			"Get your API key from: https://dashboard.nylas.com",
		)
	}

	return apiKey, nil
}

// GetGrantID returns the grant ID from arguments, environment variable, config file, or keyring.
// It checks in this order:
// 1. Command line argument (if provided) - supports email lookup if arg contains "@"
// 2. Environment variable (NYLAS_GRANT_ID)
// 3. Config file (default_grant)
// 4. Stored default grant (from keyring/file)
func GetGrantID(args []string) (string, error) {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		// Fall back to env var and config only if keyring unavailable
		if grantID := os.Getenv("NYLAS_GRANT_ID"); grantID != "" {
			return grantID, nil
		}

		// Try config file
		configStore := config.NewDefaultFileStore()
		cfg, err := configStore.Load()
		if err == nil && cfg.DefaultGrant != "" {
			return cfg.DefaultGrant, nil
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

	// Check config file
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err == nil && cfg.DefaultGrant != "" {
		return cfg.DefaultGrant, nil
	}

	// Try to get default grant from keyring
	grantID, err := grantStore.GetDefaultGrant()
	if err != nil {
		return "", NewUserErrorWithSuggestions(
			"No grant ID provided",
			"Check available grants with: nylas auth list",
			"Set default grant with: nylas config set default_grant <grant-id>",
			"Use environment variable: export NYLAS_GRANT_ID=<grant-id>",
			"Or specify as argument: nylas [command] <grant-id>",
		)
	}

	return grantID, nil
}

// containsAt checks if a string contains "@" (for email detection).
func containsAt(s string) bool {
	return strings.ContainsRune(s, '@')
}

// WithClient is a generic helper that handles client setup, context creation, and grant ID resolution.
// This reduces boilerplate in commands by handling all the common setup in one place.
//
// Usage:
//
//	return common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) error {
//	    calendars, err := client.GetCalendars(ctx, grantID)
//	    if err != nil {
//	        return err
//	    }
//	    // ... process calendars ...
//	    return nil
//	})
func WithClient[T any](args []string, fn func(ctx context.Context, client ports.NylasClient, grantID string) (T, error)) (T, error) {
	var zero T

	// Get Nylas client
	client, err := GetNylasClient()
	if err != nil {
		return zero, err
	}

	// Get grant ID
	grantID, err := GetGrantID(args)
	if err != nil {
		return zero, err
	}

	// Notify audit system of grant usage
	if AuditGrantHook != nil {
		AuditGrantHook(grantID)
	}

	// Create context with timeout
	ctx, cancel := CreateContext()
	defer cancel()

	// Execute function
	return fn(ctx, client, grantID)
}

// WithClientNoGrant is a generic helper for commands that don't need a grant ID.
// This is useful for admin commands or commands that operate without a specific account.
//
// Usage:
//
//	return common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) error {
//	    // ... use client ...
//	    return nil
//	})
func WithClientNoGrant[T any](fn func(ctx context.Context, client ports.NylasClient) (T, error)) (T, error) {
	var zero T

	// Get Nylas client
	client, err := GetNylasClient()
	if err != nil {
		return zero, err
	}

	// Create context with timeout
	ctx, cancel := CreateContext()
	defer cancel()

	// Execute function
	return fn(ctx, client)
}
