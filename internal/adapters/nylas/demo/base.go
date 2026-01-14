package demo

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// Client is a demo client that returns realistic demo data for screenshots and demos.
// It implements the ports.NylasClient interface without requiring any credentials.
type Client struct{}

// New creates a new demo client.
func New() *Client {
	return &Client{}
}

// SetRegion is a no-op for demo client.
func (d *Client) SetRegion(region string) {}

// SetCredentials is a no-op for demo client.
func (d *Client) SetCredentials(clientID, clientSecret, apiKey string) {}

// BuildAuthURL returns a mock auth URL.
func (d *Client) BuildAuthURL(provider domain.Provider, redirectURI string) string {
	return "https://demo.nylas.com/auth"
}

// ExchangeCode returns a mock grant.
func (d *Client) ExchangeCode(ctx context.Context, code, redirectURI string) (*domain.Grant, error) {
	return &domain.Grant{
		ID:          "demo-grant-id",
		Email:       "demo@example.com",
		Provider:    domain.ProviderGoogle,
		GrantStatus: "valid",
	}, nil
}
