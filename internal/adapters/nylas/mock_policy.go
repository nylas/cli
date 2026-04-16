package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListPolicies(ctx context.Context) ([]domain.Policy, error) {
	return []domain.Policy{
		{
			ID:             "policy-1",
			Name:           "Default Policy",
			ApplicationID:  "app-123",
			OrganizationID: "org-123",
			Rules:          []string{"rule-1"},
		},
	}, nil
}

func (m *MockClient) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	return &domain.Policy{
		ID:             policyID,
		Name:           "Default Policy",
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
		Rules:          []string{"rule-1"},
	}, nil
}

func (m *MockClient) CreatePolicy(ctx context.Context, payload map[string]any) (*domain.Policy, error) {
	name, _ := payload["name"].(string)
	return &domain.Policy{
		ID:             "policy-new",
		Name:           name,
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
		Rules:          []string{"rule-1"},
	}, nil
}

func (m *MockClient) UpdatePolicy(ctx context.Context, policyID string, payload map[string]any) (*domain.Policy, error) {
	name, _ := payload["name"].(string)
	return &domain.Policy{
		ID:             policyID,
		Name:           name,
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
		Rules:          []string{"rule-1"},
	}, nil
}

func (m *MockClient) DeletePolicy(ctx context.Context, policyID string) error {
	return nil
}
