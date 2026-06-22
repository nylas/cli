// Package providers implements a provider registry pattern for extensible multi-provider support.
package providers

import (
	"sync"

	"github.com/nylas/cli/internal/ports"
)

// ProviderFactory is a function that creates a new provider client
type ProviderFactory func(config ProviderConfig) (ports.NylasClient, error)

// ProviderConfig contains configuration for initializing a provider
type ProviderConfig struct {
	APIKey       string
	ClientID     string
	ClientSecret string
	BaseURL      string
	Region       string
}

// Registry maintains a registry of provider factories
type Registry struct {
	mu        sync.RWMutex
	factories map[string]ProviderFactory
}

var defaultRegistry = &Registry{
	factories: make(map[string]ProviderFactory),
}

// Register registers a provider factory with the default registry
// Providers should call this in their init() function
func Register(name string, factory ProviderFactory) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.factories[name] = factory
}
