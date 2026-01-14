package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListApplications(ctx context.Context) ([]domain.Application, error) {
	return []domain.Application{
		{ID: "app-1", ApplicationID: "app-id-1", Region: "us"},
	}, nil
}

func (m *MockClient) GetApplication(ctx context.Context, appID string) (*domain.Application, error) {
	return &domain.Application{
		ID:            appID,
		ApplicationID: appID,
		Region:        "us",
	}, nil
}

func (m *MockClient) CreateApplication(ctx context.Context, req *domain.CreateApplicationRequest) (*domain.Application, error) {
	return &domain.Application{
		ID:            "new-app",
		ApplicationID: "new-app-id",
		Region:        req.Region,
	}, nil
}

func (m *MockClient) UpdateApplication(ctx context.Context, appID string, req *domain.UpdateApplicationRequest) (*domain.Application, error) {
	return &domain.Application{
		ID:            appID,
		ApplicationID: appID,
		Region:        "us",
	}, nil
}

func (m *MockClient) DeleteApplication(ctx context.Context, appID string) error {
	return nil
}

func (m *MockClient) ListConnectors(ctx context.Context) ([]domain.Connector, error) {
	return []domain.Connector{
		{ID: "conn-1", Name: "Google Connector", Provider: "google"},
		{ID: "conn-2", Name: "Microsoft Connector", Provider: "microsoft"},
	}, nil
}

func (m *MockClient) GetConnector(ctx context.Context, connectorID string) (*domain.Connector, error) {
	return &domain.Connector{
		ID:       connectorID,
		Name:     "Google Connector",
		Provider: "google",
	}, nil
}

func (m *MockClient) CreateConnector(ctx context.Context, req *domain.CreateConnectorRequest) (*domain.Connector, error) {
	return &domain.Connector{
		ID:       "new-conn",
		Name:     req.Name,
		Provider: req.Provider,
	}, nil
}

func (m *MockClient) UpdateConnector(ctx context.Context, connectorID string, req *domain.UpdateConnectorRequest) (*domain.Connector, error) {
	name := "Updated Connector"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.Connector{
		ID:       connectorID,
		Name:     name,
		Provider: "google",
	}, nil
}

func (m *MockClient) DeleteConnector(ctx context.Context, connectorID string) error {
	return nil
}

func (m *MockClient) ListCredentials(ctx context.Context, connectorID string) ([]domain.ConnectorCredential, error) {
	return []domain.ConnectorCredential{
		{ID: "cred-1", Name: "OAuth Credential", CredentialType: "oauth"},
	}, nil
}

func (m *MockClient) GetCredential(ctx context.Context, credentialID string) (*domain.ConnectorCredential, error) {
	return &domain.ConnectorCredential{
		ID:             credentialID,
		Name:           "OAuth Credential",
		CredentialType: "oauth",
	}, nil
}

func (m *MockClient) CreateCredential(ctx context.Context, connectorID string, req *domain.CreateCredentialRequest) (*domain.ConnectorCredential, error) {
	return &domain.ConnectorCredential{
		ID:             "new-cred",
		Name:           req.Name,
		CredentialType: req.CredentialType,
	}, nil
}

func (m *MockClient) UpdateCredential(ctx context.Context, credentialID string, req *domain.UpdateCredentialRequest) (*domain.ConnectorCredential, error) {
	name := "Updated Credential"
	if req.Name != nil {
		name = *req.Name
	}
	return &domain.ConnectorCredential{
		ID:             credentialID,
		Name:           name,
		CredentialType: "oauth",
	}, nil
}

func (m *MockClient) DeleteCredential(ctx context.Context, credentialID string) error {
	return nil
}
