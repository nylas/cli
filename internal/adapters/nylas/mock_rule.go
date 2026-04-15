package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) ListRules(ctx context.Context) ([]domain.Rule, error) {
	enabled := true
	return []domain.Rule{
		{
			ID:             "rule-1",
			Name:           "Default Rule",
			Enabled:        &enabled,
			Trigger:        "inbound",
			ApplicationID:  "app-123",
			OrganizationID: "org-123",
			Match: &domain.RuleMatch{
				Operator: "all",
				Conditions: []domain.RuleCondition{{
					Field:    "from.domain",
					Operator: "is",
					Value:    "example.com",
				}},
			},
			Actions: []domain.RuleAction{{
				Type: "mark_as_spam",
			}},
		},
	}, nil
}

func (m *MockClient) GetRule(ctx context.Context, ruleID string) (*domain.Rule, error) {
	enabled := true
	return &domain.Rule{
		ID:             ruleID,
		Name:           "Default Rule",
		Enabled:        &enabled,
		Trigger:        "inbound",
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
		Match: &domain.RuleMatch{
			Operator: "all",
			Conditions: []domain.RuleCondition{{
				Field:    "from.domain",
				Operator: "is",
				Value:    "example.com",
			}},
		},
		Actions: []domain.RuleAction{{
			Type: "mark_as_spam",
		}},
	}, nil
}

func (m *MockClient) CreateRule(ctx context.Context, payload map[string]any) (*domain.Rule, error) {
	name, _ := payload["name"].(string)
	enabled := true
	return &domain.Rule{
		ID:             "rule-new",
		Name:           name,
		Enabled:        &enabled,
		Trigger:        "inbound",
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
	}, nil
}

func (m *MockClient) UpdateRule(ctx context.Context, ruleID string, payload map[string]any) (*domain.Rule, error) {
	name, _ := payload["name"].(string)
	enabled := true
	return &domain.Rule{
		ID:             ruleID,
		Name:           name,
		Enabled:        &enabled,
		Trigger:        "inbound",
		ApplicationID:  "app-123",
		OrganizationID: "org-123",
	}, nil
}

func (m *MockClient) DeleteRule(ctx context.Context, ruleID string) error {
	return nil
}
