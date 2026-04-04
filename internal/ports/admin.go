package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// AdminClient defines the interface for administrative operations.
type AdminClient interface {
	// ================================
	// APPLICATION OPERATIONS
	// ================================

	// ListApplications retrieves all applications.
	ListApplications(ctx context.Context) ([]domain.Application, error)

	// GetApplication retrieves a specific application.
	GetApplication(ctx context.Context, appID string) (*domain.Application, error)

	// CreateApplication creates a new application.
	CreateApplication(ctx context.Context, req *domain.CreateApplicationRequest) (*domain.Application, error)

	// UpdateApplication updates an existing application.
	UpdateApplication(ctx context.Context, appID string, req *domain.UpdateApplicationRequest) (*domain.Application, error)

	// DeleteApplication deletes an application.
	DeleteApplication(ctx context.Context, appID string) error

	// ListCallbackURIs retrieves all callback URIs for the application.
	ListCallbackURIs(ctx context.Context) ([]domain.CallbackURI, error)

	// GetCallbackURI retrieves a specific callback URI.
	GetCallbackURI(ctx context.Context, uriID string) (*domain.CallbackURI, error)

	// CreateCallbackURI creates a new callback URI for the application.
	CreateCallbackURI(ctx context.Context, req *domain.CreateCallbackURIRequest) (*domain.CallbackURI, error)

	// UpdateCallbackURI updates an existing callback URI.
	UpdateCallbackURI(ctx context.Context, uriID string, req *domain.UpdateCallbackURIRequest) (*domain.CallbackURI, error)

	// DeleteCallbackURI deletes a callback URI.
	DeleteCallbackURI(ctx context.Context, uriID string) error

	// ================================
	// CONNECTOR OPERATIONS
	// ================================

	// ListConnectors retrieves all connectors.
	ListConnectors(ctx context.Context) ([]domain.Connector, error)

	// GetConnector retrieves a specific connector.
	GetConnector(ctx context.Context, connectorID string) (*domain.Connector, error)

	// CreateConnector creates a new connector.
	CreateConnector(ctx context.Context, req *domain.CreateConnectorRequest) (*domain.Connector, error)

	// UpdateConnector updates an existing connector.
	UpdateConnector(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error)

	// DeleteConnector deletes a connector.
	DeleteConnector(ctx context.Context, connectorID string) error

	// ================================
	// CREDENTIAL OPERATIONS
	// ================================

	// ListCredentials retrieves all credentials for a connector.
	ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error)

	// GetCredential retrieves a specific credential.
	GetCredential(ctx context.Context, credentialID string) (*domain.ConnectorCredential, error)

	// CreateCredential creates a new credential.
	CreateCredential(ctx context.Context, connectorID string, req *domain.CreateCredentialRequest) (*domain.ConnectorCredential, error)

	// UpdateCredential updates an existing credential.
	UpdateCredential(ctx context.Context, credentialID string, req *domain.UpdateCredentialRequest) (*domain.ConnectorCredential, error)

	// DeleteCredential deletes a credential.
	DeleteCredential(ctx context.Context, credentialID string) error

	// ================================
	// ADMIN GRANT OPERATIONS
	// ================================

	// ListAllGrants retrieves all grants with query parameters.
	ListAllGrants(ctx context.Context, params *domain.GrantsQueryParams) ([]domain.Grant, error)

	// GetGrantStats retrieves grant statistics.
	GetGrantStats(ctx context.Context) (*domain.GrantStats, error)
}
