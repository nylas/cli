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
func (d *DemoClient) BuildAuthURL(provider domain.Provider, redirectURI, state, codeChallenge string) string {
	return "https://demo.nylas.com/auth"
}

// ExchangeCode returns a mock grant.
func (d *DemoClient) ExchangeCode(ctx context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error) {
	return &domain.Grant{
		ID:          "demo-grant-id",
		Email:       "demo@example.com",
		Provider:    domain.ProviderGoogle,
		GrantStatus: "valid",
	}, nil
}

// CreateCustomGrant returns a mock grant for demo mode.
func (d *DemoClient) CreateCustomGrant(_ context.Context, provider string, _ map[string]any) (*domain.Grant, error) {
	return &domain.Grant{
		ID:          "demo-custom-grant-id",
		Email:       "demo@example.com",
		Provider:    domain.Provider(provider),
		GrantStatus: "valid",
	}, nil
}
