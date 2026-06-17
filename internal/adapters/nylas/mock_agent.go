package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListAgentAccounts(ctx context.Context) ([]domain.AgentAccount, error) {
	return []domain.AgentAccount{
		{
			ID:          "agent-1",
			Provider:    domain.ProviderNylas,
			Email:       "agent@example.com",
			GrantStatus: "valid",
			WorkspaceID: "workspace-1",
		},
	}, nil
}

func (m *MockClient) GetAgentAccount(ctx context.Context, grantID string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          grantID,
		Provider:    domain.ProviderNylas,
		Email:       "agent@example.com",
		GrantStatus: "valid",
		WorkspaceID: "workspace-1",
	}, nil
}

func (m *MockClient) CreateAgentAccount(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          "agent-new",
		Provider:    domain.ProviderNylas,
		Email:       email,
		Name:        name,
		GrantStatus: "valid",
		WorkspaceID: "workspace-new",
	}, nil
}

func (m *MockClient) UpdateAgentAccount(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          grantID,
		Provider:    domain.ProviderNylas,
		Email:       email,
		Name:        name,
		GrantStatus: "valid",
		WorkspaceID: "workspace-1",
	}, nil
}

func (m *MockClient) DeleteAgentAccount(ctx context.Context, grantID string) error {
	return nil
}
