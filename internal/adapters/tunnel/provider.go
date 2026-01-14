// Package tunnel provides tunnel implementations for exposing local servers.
package tunnel

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/nylas/cli/internal/ports"
)

// Provider implements ports.TunnelProvider for creating tunnel instances.
type Provider struct{}

// NewProvider creates a new tunnel provider.
func NewProvider() *Provider {
	return &Provider{}
}

// IsAvailable checks if the tunnel provider is installed and available.
func (p *Provider) IsAvailable(provider string) bool {
	switch strings.ToLower(provider) {
	case "cloudflared", "cloudflare", "cf":
		_, err := exec.LookPath("cloudflared")
		return err == nil
	default:
		return false
	}
}

// CreateTunnel creates a new tunnel instance for the given provider.
func (p *Provider) CreateTunnel(provider, localURL string) (ports.Tunnel, error) {
	switch strings.ToLower(provider) {
	case "cloudflared", "cloudflare", "cf":
		return NewCloudflaredTunnel(localURL), nil
	default:
		return nil, fmt.Errorf("unsupported tunnel provider: %s. Supported: %s",
			provider, strings.Join(p.SupportedProviders(), ", "))
	}
}

// SupportedProviders returns a list of supported tunnel providers.
func (p *Provider) SupportedProviders() []string {
	return []string{"cloudflared"}
}

// Ensure Provider implements ports.TunnelProvider
var _ ports.TunnelProvider = (*Provider)(nil)
