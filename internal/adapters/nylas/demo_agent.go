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
			Settings: domain.AgentAccountSettings{
				PolicyID: "policy-demo-1",
			},
		},
	}, nil
}

func (d *DemoClient) GetAgentAccount(ctx context.Context, grantID string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          grantID,
		Provider:    domain.ProviderNylas,
		Email:       "demo-agent@example.com",
		GrantStatus: "valid",
		Settings: domain.AgentAccountSettings{
			PolicyID: "policy-demo-1",
		},
	}, nil
}

func (d *DemoClient) CreateAgentAccount(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          "agent-demo-new",
		Provider:    domain.ProviderNylas,
		Email:       email,
		GrantStatus: "valid",
		Settings: domain.AgentAccountSettings{
			PolicyID: policyID,
		},
	}, nil
}

func (d *DemoClient) DeleteAgentAccount(ctx context.Context, grantID string) error {
	return nil
}
