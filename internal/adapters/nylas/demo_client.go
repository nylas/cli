package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// DemoClient is a client that returns realistic demo data for screenshots and demos.
// It implements the ports.NylasClient interface without requiring any credentials.
type DemoClient struct{}

// NewDemoClient creates a new DemoClient for demo mode.
func NewDemoClient() *DemoClient {
	return &DemoClient{}
}

// SetRegion is a no-op for demo client.
func (d *DemoClient) SetRegion(region string) {}

// SetCredentials is a no-op for demo client.
func (d *DemoClient) SetCredentials(clientID, clientSecret, apiKey string) {}

// BuildAuthURL returns a mock auth URL.
func (d *DemoClient) BuildAuthURL(provider domain.Provider, redirectURI string) string {
	return "https://demo.nylas.com/auth"
}

// ExchangeCode returns a mock grant.
func (d *DemoClient) ExchangeCode(ctx context.Context, code, redirectURI string) (*domain.Grant, error) {
	return &domain.Grant{
		ID:          "demo-grant-id",
		Email:       "demo@example.com",
		Provider:    domain.ProviderGoogle,
		GrantStatus: "valid",
	}, nil
}
