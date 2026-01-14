// Package client provides the ClientFactory adapter implementation.
package client

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

// Factory implements ports.ClientFactory for creating Nylas API clients.
type Factory struct {
	configDir string

	mu           sync.Mutex
	cachedClient ports.NylasClient
	clientOnce   sync.Once
	clientErr    error
}

// NewFactory creates a new client factory using the default config directory.
func NewFactory() *Factory {
	return &Factory{
		configDir: config.DefaultConfigDir(),
	}
}

// NewFactoryWithConfigDir creates a new client factory with a custom config directory.
func NewFactoryWithConfigDir(configDir string) *Factory {
	return &Factory{
		configDir: configDir,
	}
}

// CreateClient creates a new Nylas API client with credentials from
// environment variables or the secret store.
func (f *Factory) CreateClient() (ports.NylasClient, error) {
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
		secretStore, err := keyring.NewSecretStore(f.configDir)
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

// GetCachedClient returns a singleton Nylas client.
func (f *Factory) GetCachedClient() (ports.NylasClient, error) {
	f.clientOnce.Do(func() {
		f.cachedClient, f.clientErr = f.CreateClient()
	})
	return f.cachedClient, f.clientErr
}

// ResetCache clears the cached client (useful for testing).
func (f *Factory) ResetCache() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cachedClient = nil
	f.clientErr = nil
	f.clientOnce = sync.Once{}
}

// GetAPIKey returns the API key from environment variable or keyring.
func (f *Factory) GetAPIKey() (string, error) {
	// First check environment variable (highest priority)
	apiKey := os.Getenv("NYLAS_API_KEY")

	// If not in env, try keyring/file store
	if apiKey == "" {
		secretStore, err := keyring.NewSecretStore(f.configDir)
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
func (f *Factory) GetGrantID(args []string) (string, error) {
	secretStore, err := keyring.NewSecretStore(f.configDir)
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

// Ensure Factory implements ports.ClientFactory
var _ ports.ClientFactory = (*Factory)(nil)
