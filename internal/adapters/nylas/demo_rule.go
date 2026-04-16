package nylas

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) ListRules(ctx context.Context) ([]domain.Rule, error) {
	enabled := true
	return []domain.Rule{
		{
			ID:             "rule-demo-1",
			Name:           "Demo Rule",
			Enabled:        &enabled,
			Trigger:        "inbound",
			ApplicationID:  "app-demo",
			OrganizationID: "org-demo",
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

func (d *DemoClient) GetRule(ctx context.Context, ruleID string) (*domain.Rule, error) {
	enabled := true
	return &domain.Rule{
		ID:             ruleID,
		Name:           "Demo Rule",
		Enabled:        &enabled,
		Trigger:        "inbound",
		ApplicationID:  "app-demo",
		OrganizationID: "org-demo",
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

func (d *DemoClient) CreateRule(ctx context.Context, payload map[string]any) (*domain.Rule, error) {
	name, _ := payload["name"].(string)
	trigger, _ := payload["trigger"].(string)
	if trigger == "" {
		trigger = "inbound"
	}
	enabled := true
	return &domain.Rule{
		ID:             "rule-demo-new",
		Name:           name,
		Enabled:        &enabled,
		Trigger:        trigger,
		ApplicationID:  "app-demo",
		OrganizationID: "org-demo",
	}, nil
}

func (d *DemoClient) UpdateRule(ctx context.Context, ruleID string, payload map[string]any) (*domain.Rule, error) {
	name, _ := payload["name"].(string)
	trigger, _ := payload["trigger"].(string)
	if trigger == "" {
		trigger = "inbound"
	}
	enabled := true
	return &domain.Rule{
		ID:             ruleID,
		Name:           name,
		Enabled:        &enabled,
		Trigger:        trigger,
		ApplicationID:  "app-demo",
		OrganizationID: "org-demo",
	}, nil
}

func (d *DemoClient) DeleteRule(ctx context.Context, ruleID string) error {
	return nil
}
