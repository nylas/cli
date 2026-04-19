package agent

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectPolicyScopedRules_SkipsDanglingReferences(t *testing.T) {
	enabled := true
	accounts := []policyAgentAccountRef{{
		GrantID: "grant-1",
		Email:   "agent@example.com",
	}}
	policy := &domain.Policy{
		ID:    "policy-1",
		Name:  "Primary Policy",
		Rules: []string{"missing-rule", " rule-2 ", "", "rule-1"},
	}
	allRules := []domain.Rule{
		{ID: "rule-1", Name: "First Rule", Enabled: &enabled},
		{ID: "rule-2", Name: "Second Rule", Enabled: &enabled},
	}

	rules, refs := collectPolicyScopedRules(policy, accounts, allRules)

	require.Len(t, rules, 2)
	assert.Equal(t, "rule-2", rules[0].ID)
	assert.Equal(t, "rule-1", rules[1].ID)
	assert.NotContains(t, refs, "missing-rule")
	assert.Equal(t, []rulePolicyRef{{
		PolicyID:   "policy-1",
		PolicyName: "Primary Policy",
		Accounts:   accounts,
	}}, refs["rule-1"])
}

func TestCollectPolicyScopedRules_ReturnsEmptyWhenPolicyOnlyHasDanglingReferences(t *testing.T) {
	policy := &domain.Policy{
		ID:    "policy-1",
		Name:  "Primary Policy",
		Rules: []string{"missing-rule"},
	}

	rules, refs := collectPolicyScopedRules(policy, nil, []domain.Rule{{ID: "rule-1"}})

	assert.Empty(t, rules)
	assert.Empty(t, refs)
}
