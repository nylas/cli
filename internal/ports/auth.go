package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// AuthClient defines the interface for authentication and grant operations.
type AuthClient interface {
	// BuildAuthURL builds an OAuth authorization URL for a provider.
	BuildAuthURL(provider domain.Provider, redirectURI, state, codeChallenge string) string

	// ExchangeCode exchanges an authorization code for a grant.
	ExchangeCode(ctx context.Context, code, redirectURI, codeVerifier string) (*domain.Grant, error)

	// ListGrants returns all grants for the authenticated application.
	ListGrants(ctx context.Context) ([]domain.Grant, error)

	// GetGrant retrieves a specific grant by ID.
	GetGrant(ctx context.Context, grantID string) (*domain.Grant, error)

	// RevokeGrant revokes a specific grant.
	RevokeGrant(ctx context.Context, grantID string) error
}
