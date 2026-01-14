// Package ports defines interfaces for client and service factories.
package ports

// ClientFactory defines the interface for creating Nylas API clients.
// This allows CLI code to request clients through the port layer instead
// of directly instantiating adapters, following hexagonal architecture.
type ClientFactory interface {
	// CreateClient creates a new Nylas API client configured with credentials
	// from environment variables or the secret store.
	// Returns an error if no valid credentials are found.
	CreateClient() (NylasClient, error)

	// GetCachedClient returns a singleton client instance.
	// This avoids creating multiple clients for commands that make
	// several API calls. The client is cached for the process lifetime.
	GetCachedClient() (NylasClient, error)

	// ResetCache clears the cached client (useful for testing).
	ResetCache()

	// GetAPIKey returns the API key from environment or secret store.
	// Useful for operations that need the raw API key.
	GetAPIKey() (string, error)

	// GetGrantID returns the grant ID from arguments, environment, or store.
	// It supports email lookup if the argument contains "@".
	GetGrantID(args []string) (string, error)
}

// TunnelProvider defines the interface for creating tunnel instances.
// This abstracts tunnel creation from CLI commands, allowing for easier
// testing and provider switching.
type TunnelProvider interface {
	// IsAvailable checks if the tunnel provider is installed and available.
	IsAvailable(provider string) bool

	// CreateTunnel creates a new tunnel instance for the given provider.
	// The localURL is the local server URL to tunnel to.
	// Returns an error if the provider is not supported or unavailable.
	CreateTunnel(provider, localURL string) (Tunnel, error)

	// SupportedProviders returns a list of supported tunnel providers.
	SupportedProviders() []string
}
