package agent

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestPrintRuleDetails(t *testing.T) {
	priority := 10
	enabled := true
	rule := domain.Rule{
		ID:             "rule-123",
		Name:           "Block Example",
		Description:    "Blocks example.com",
		Priority:       &priority,
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
		CreatedAt: domain.UnixTime{Time: time.Date(2026, time.April, 13, 16, 49, 44, 0, time.UTC)},
		UpdatedAt: domain.UnixTime{Time: time.Date(2026, time.April, 13, 16, 49, 44, 0, time.UTC)},
	}

	output := captureStdout(t, func() {
		printRuleDetails(rule, []rulePolicyRef{{
			PolicyID:   "policy-123",
			PolicyName: "Default Policy",
			Accounts: []policyAgentAccountRef{{
				GrantID: "grant-123",
				Email:   "agent@example.com",
			}},
		}})
	})

	assert.Contains(t, output, "Rule:         Block Example")
	assert.Contains(t, output, "Policies:")
	assert.Contains(t, output, "Default Policy")
	assert.Contains(t, output, "agent@example.com")
	assert.Contains(t, output, "Match:")
	assert.Contains(t, output, "from.domain is example.com")
	assert.Contains(t, output, "Actions:")
	assert.Contains(t, output, "mark_as_spam")
}
