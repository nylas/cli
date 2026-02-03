package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// NylasClient defines the composite interface for interacting with the Nylas API.
// It embeds domain-specific sub-interfaces following the Interface Segregation Principle.
type NylasClient interface {
	// Domain-specific interfaces
	AuthClient
	MessageClient
	CalendarClient
	ContactClient
	WebhookClient
	NotetakerClient
	InboundClient
	SchedulerClient
	AdminClient
	TransactionalClient

	// Configuration methods
	SetRegion(region string)
	SetCredentials(clientID, clientSecret, apiKey string)
}

// ============================================================================
// OTHER INTERFACES
// ============================================================================

// OAuthServer defines the interface for the OAuth callback server.
type OAuthServer interface {
	// Start starts the server on the configured port.
	Start() error

	// Stop stops the server.
	Stop() error

	// WaitForCallback waits for the OAuth callback and returns the auth code.
	WaitForCallback(ctx context.Context) (string, error)

	// GetRedirectURI returns the redirect URI for OAuth.
	GetRedirectURI() string
}

// Browser defines the interface for opening URLs in the browser.
type Browser interface {
	// Open opens a URL in the default browser.
	Open(url string) error
}

// GrantStore defines the interface for storing grant information.
type GrantStore interface {
	// SaveGrant saves grant info to storage.
	SaveGrant(info domain.GrantInfo) error

	// GetGrant retrieves grant info by ID.
	GetGrant(grantID string) (*domain.GrantInfo, error)

	// GetGrantByEmail retrieves grant info by email.
	GetGrantByEmail(email string) (*domain.GrantInfo, error)

	// ListGrants returns all stored grants.
	ListGrants() ([]domain.GrantInfo, error)

	// DeleteGrant removes a grant from storage.
	DeleteGrant(grantID string) error

	// SetDefaultGrant sets the default grant ID.
	SetDefaultGrant(grantID string) error

	// GetDefaultGrant returns the default grant ID.
	GetDefaultGrant() (string, error)

	// ClearGrants removes all grants from storage.
	ClearGrants() error
}
