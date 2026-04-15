package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListPolicies(ctx context.Context) ([]domain.Policy, error) {
	return []domain.Policy{
		{
			ID:             "policy-demo-1",
			Name:           "Demo Policy",
			ApplicationID:  "app-demo",
			OrganizationID: "org-demo",
		},
	}, nil
}

func (d *DemoClient) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	return &domain.Policy{
		ID:             policyID,
		Name:           "Demo Policy",
		ApplicationID:  "app-demo",
		OrganizationID: "org-demo",
	}, nil
}

func (d *DemoClient) CreatePolicy(ctx context.Context, payload map[string]any) (*domain.Policy, error) {
	name, _ := payload["name"].(string)
	return &domain.Policy{
		ID:             "policy-demo-new",
		Name:           name,
		ApplicationID:  "app-demo",
		OrganizationID: "org-demo",
	}, nil
}

func (d *DemoClient) UpdatePolicy(ctx context.Context, policyID string, payload map[string]any) (*domain.Policy, error) {
	name, _ := payload["name"].(string)
	return &domain.Policy{
		ID:             policyID,
		Name:           name,
		ApplicationID:  "app-demo",
		OrganizationID: "org-demo",
	}, nil
}

func (d *DemoClient) DeletePolicy(ctx context.Context, policyID string) error {
	return nil
}
