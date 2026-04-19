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
			Settings: domain.AgentAccountSettings{
				PolicyID: "policy-1",
			},
		},
	}, nil
}

func (m *MockClient) GetAgentAccount(ctx context.Context, grantID string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          grantID,
		Provider:    domain.ProviderNylas,
		Email:       "agent@example.com",
		GrantStatus: "valid",
		Settings: domain.AgentAccountSettings{
			PolicyID: "policy-1",
		},
	}, nil
}

func (m *MockClient) CreateAgentAccount(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          "agent-new",
		Provider:    domain.ProviderNylas,
		Email:       email,
		GrantStatus: "valid",
		Settings: domain.AgentAccountSettings{
			PolicyID: policyID,
		},
	}, nil
}

func (m *MockClient) UpdateAgentAccount(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error) {
	return &domain.AgentAccount{
		ID:          grantID,
		Provider:    domain.ProviderNylas,
		Email:       email,
		GrantStatus: "valid",
		Settings:    domain.AgentAccountSettings{PolicyID: "policy-1"},
	}, nil
}

func (m *MockClient) DeleteAgentAccount(ctx context.Context, grantID string) error {
	return nil
}
