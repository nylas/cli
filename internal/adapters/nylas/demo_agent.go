package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListAgentAccounts(ctx context.Context) ([]domain.AgentAccount, error) {
	return []domain.AgentAccount{
		{
			ID:          "agent-demo-1",
			Provider:    domain.ProviderNylas,
			Email:       "demo-agent@example.com",
			GrantStatus: "valid",
			WorkspaceID: "workspace-demo-1",
		},
	}, nil
}

func (d *DemoClient) GetAgentAccount(ctx context.Context, grantID string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          grantID,
		Provider:    domain.ProviderNylas,
		Email:       "demo-agent@example.com",
		GrantStatus: "valid",
		WorkspaceID: "workspace-demo-1",
	}, nil
}

func (d *DemoClient) CreateAgentAccount(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          "agent-demo-new",
		Provider:    domain.ProviderNylas,
		Email:       email,
		Name:        name,
		GrantStatus: "valid",
		WorkspaceID: "workspace-demo-new",
	}, nil
}

func (d *DemoClient) UpdateAgentAccount(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          grantID,
		Provider:    domain.ProviderNylas,
		Email:       email,
		Name:        name,
		GrantStatus: "valid",
		WorkspaceID: "workspace-demo-1",
	}, nil
}

func (d *DemoClient) DeleteAgentAccount(ctx context.Context, grantID string) error {
	return nil
}
