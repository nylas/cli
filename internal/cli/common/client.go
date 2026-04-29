package common

import (
	"context"
	"errors"
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
		secretStore, err := openSecretStore()
		if err != nil {
			return nil, err
		}

		apiKey, err = getStoredSecret(secretStore, ports.KeyAPIKey)
		if err != nil {
			return nil, err
		}
		if clientID == "" {
			clientID, err = getStoredSecret(secretStore, ports.KeyClientID)
			if err != nil {
				return nil, err
			}
		}
		if clientSecret == "" {
			clientSecret, err = getStoredSecret(secretStore, ports.KeyClientSecret)
			if err != nil {
				return nil, err
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
		secretStore, err := openSecretStore()
		if err != nil {
			return "", err
		}
		apiKey, err = getStoredSecret(secretStore, ports.KeyAPIKey)
		if err != nil {
			return "", err
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

// GetGrantID returns the grant ID from arguments, environment variable, local grant cache, or config file.
// It checks in this order:
// 1. Command line argument (if provided) - supports email lookup if arg contains "@"
// 2. Environment variable (NYLAS_GRANT_ID)
// 3. Stored default grant (from local grant cache)
func GetGrantID(args []string) (string, error) {
	// If provided as argument
	if len(args) > 0 && args[0] != "" {
		identifier := args[0]

		// Direct grant IDs should not depend on local secret-store health.
		if !containsAt(identifier) {
			return identifier, nil
		}
	}

	// Check environment variable
	if grantID := os.Getenv("NYLAS_GRANT_ID"); grantID != "" {
		return grantID, nil
	}

	grantStore, err := NewDefaultGrantStore()
	if err != nil {
		return "", err
	}

	// Email arguments require a local grant lookup.
	if len(args) > 0 && args[0] != "" {
		identifier := args[0]
		grant, err := grantStore.GetGrantByEmail(identifier)
		if err != nil {
			if errors.Is(err, domain.ErrGrantNotFound) {
				return "", fmt.Errorf("no grant found for email: %s", identifier)
			}
			return "", err
		}
		return grant.ID, nil
	}

	// Try to get default grant from the local grant cache first.
	grantID, err := grantStore.GetDefaultGrant()
	switch {
	case err == nil:
		return grantID, nil
	case !errors.Is(err, domain.ErrNoDefaultGrant):
		return "", err
	}

	return "", NewUserErrorWithSuggestions(
		"No grant ID provided. Run 'nylas auth list' to find a grant, then 'nylas auth switch <grant-id-or-email>' to set the default.",
		"List available grants with: nylas auth list",
		"Set a default grant with: nylas auth switch <grant-id-or-email>",
		"Use environment variable: export NYLAS_GRANT_ID=<grant-id>",
		"Or specify as argument: nylas [command] <grant-id>",
	)
}

// containsAt checks if a string contains "@" (for email detection).
func containsAt(s string) bool {
	return strings.ContainsRune(s, '@')
}

func openSecretStore() (ports.SecretStore, error) {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return nil, wrapSecretStoreError(err)
	}
	return secretStore, nil
}

func getStoredSecret(secretStore ports.SecretStore, key string) (string, error) {
	value, err := secretStore.Get(key)
	switch {
	case err == nil:
		return value, nil
	case errors.Is(err, domain.ErrSecretNotFound):
		return "", nil
	default:
		return "", wrapSecretStoreError(err)
	}
}

func wrapSecretStoreError(err error) error {
	if err == nil || errors.Is(err, domain.ErrSecretStoreFailed) {
		return err
	}
	return fmt.Errorf("%w: %v", domain.ErrSecretStoreFailed, err)
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
