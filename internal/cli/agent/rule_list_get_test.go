package agent

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type workspaceRuleTestClient struct {
	workspaces map[string]*domain.Workspace
	rules      map[string]*domain.Rule
	updates    map[string][]string
}

func (c *workspaceRuleTestClient) GetWorkspace(ctx context.Context, workspaceID string) (*domain.Workspace, error) {
	workspace := c.workspaces[workspaceID]
	if workspace == nil {
		return nil, domain.ErrWorkspaceNotFound
	}
	copy := *workspace
	copy.RulesIDs = append([]string(nil), workspace.RulesIDs...)
	return &copy, nil
}

func (c *workspaceRuleTestClient) UpdateWorkspace(ctx context.Context, workspaceID string, req *domain.UpdateWorkspaceRequest) (*domain.Workspace, error) {
	workspace := c.workspaces[workspaceID]
	if workspace == nil {
		return nil, domain.ErrWorkspaceNotFound
	}
	if req.RulesIDs != nil {
		workspace.RulesIDs = append([]string(nil), (*req.RulesIDs)...)
		if c.updates != nil {
			c.updates[workspaceID] = append([]string(nil), (*req.RulesIDs)...)
		}
	}
	return workspace, nil
}

func (c *workspaceRuleTestClient) GetRule(ctx context.Context, ruleID string) (*domain.Rule, error) {
	rule := c.rules[ruleID]
	if rule == nil {
		return nil, domain.ErrRuleNotFound
	}
	return rule, nil
}

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

func TestBuildRuleRefsByIDWithRuleIDsFallsBackToPolicyRulesWhenWorkspaceRulesAbsent(t *testing.T) {
	accounts := []policyAgentAccountRef{{
		GrantID: "grant-1",
		Email:   "agent@example.com",
	}}
	policies := []domain.Policy{{
		ID:    "policy-1",
		Name:  "Primary Policy",
		Rules: []string{"legacy-rule"},
	}}

	refs := buildRuleRefsByIDWithRuleIDs(policies, map[string][]policyAgentAccountRef{"policy-1": accounts}, map[string][]string{})

	require.Contains(t, refs, "legacy-rule")
	assert.Equal(t, accounts, refs["legacy-rule"][0].Accounts)
}

func TestBuildRuleRefsByIDWithRuleIDsUsesEmptyWorkspaceRulesWhenPresent(t *testing.T) {
	policies := []domain.Policy{{
		ID:    "policy-1",
		Name:  "Primary Policy",
		Rules: []string{"legacy-rule"},
	}}
	refsByPolicyID := map[string][]policyAgentAccountRef{
		"policy-1": {{GrantID: "grant-1", WorkspaceID: "workspace-1"}},
	}

	refs := buildRuleRefsByIDWithRuleIDs(policies, refsByPolicyID, map[string][]string{"policy-1": {}})

	assert.Empty(t, refs)
}

func TestWorkspacesLeftEmptyByRuleRemovalBlocksLastLiveRule(t *testing.T) {
	client := &workspaceRuleTestClient{
		workspaces: map[string]*domain.Workspace{
			"workspace-1": {ID: "workspace-1", Name: "Agent Workspace", RulesIDs: []string{"rule-1", "missing-rule"}},
		},
		rules: map[string]*domain.Rule{
			"rule-1": {ID: "rule-1"},
		},
	}
	refs := []rulePolicyRef{{
		PolicyID: "policy-1",
		Accounts: []policyAgentAccountRef{{
			GrantID:     "grant-1",
			WorkspaceID: "workspace-1",
		}},
	}}

	blocking, err := workspacesLeftEmptyByRuleRemoval(context.Background(), client, refs, "rule-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"Agent Workspace (workspace-1)"}, blocking)
}

func TestDetachRuleFromAgentWorkspacesRemovesAndRollsBackWorkspaceRule(t *testing.T) {
	client := &workspaceRuleTestClient{
		workspaces: map[string]*domain.Workspace{
			"workspace-1": {ID: "workspace-1", RulesIDs: []string{"rule-1", "rule-2"}},
		},
		updates: make(map[string][]string),
	}
	refs := []rulePolicyRef{{
		PolicyID: "policy-1",
		Accounts: []policyAgentAccountRef{{
			GrantID:     "grant-1",
			WorkspaceID: "workspace-1",
		}},
	}}

	rollback, err := detachRuleFromAgentWorkspaces(context.Background(), client, refs, "rule-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"rule-2"}, client.workspaces["workspace-1"].RulesIDs)
	assert.Equal(t, []string{"rule-2"}, client.updates["workspace-1"])

	require.NoError(t, rollback(context.Background()))
	assert.Equal(t, []string{"rule-1", "rule-2"}, client.workspaces["workspace-1"].RulesIDs)
}
