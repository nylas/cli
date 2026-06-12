package agentgraph

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func graphFixture() ([]domain.AgentAccount, []domain.Workspace, []domain.Policy, []domain.Rule, []domain.AgentList) {
	enabled := true
	accounts := []domain.AgentAccount{
		{ID: "grant-1", Email: "support@app.nylas.email", GrantStatus: "valid", WorkspaceID: "ws-1"},
	}
	workspaces := []domain.Workspace{
		{ID: "ws-1", Name: "Support workspace", AutoGroup: true, Default: true, PolicyID: "policy-1", RulesIDs: []string{"rule-1"}},
	}
	policies := []domain.Policy{
		{ID: "policy-1", Name: "Default Policy"},
	}
	rules := []domain.Rule{
		{
			ID:      "rule-1",
			Name:    "Block listed domains",
			Trigger: "inbound",
			Enabled: &enabled,
			Match: &domain.RuleMatch{
				Operator: "all",
				Conditions: []domain.RuleCondition{
					{Field: "from.domain", Operator: "in_list", Value: []any{"list-1"}},
				},
			},
		},
	}
	lists := []domain.AgentList{
		{ID: "list-1", Name: "Blocked domains", Type: "domain", ItemsCount: 3},
	}
	return accounts, workspaces, policies, rules, lists
}

func TestBuild_LinksResources(t *testing.T) {
	accounts, workspaces, policies, rules, lists := graphFixture()

	overview := Build(accounts, workspaces, policies, rules, lists)

	require.Len(t, overview.Accounts, 1)
	acct := overview.Accounts[0]
	assert.Equal(t, "support@app.nylas.email", acct.Email)
	assert.Equal(t, "ws-1", acct.WorkspaceID)
	assert.Equal(t, "Support workspace", acct.WorkspaceName)
	assert.True(t, acct.AutoGroup)
	assert.True(t, acct.Default)
	assert.False(t, acct.WorkspaceMissing)

	require.NotNil(t, acct.Policy)
	assert.Equal(t, "Default Policy", acct.Policy.Name)
	assert.False(t, acct.Policy.Missing)

	require.Len(t, acct.Rules, 1)
	rule := acct.Rules[0]
	assert.Equal(t, "Block listed domains", rule.Name)
	assert.False(t, rule.Missing)
	require.Len(t, rule.Lists, 1)
	assert.Equal(t, "Blocked domains", rule.Lists[0].Name)
	assert.Equal(t, 3, rule.Lists[0].ItemsCount)
	assert.False(t, rule.Lists[0].Missing)

	assert.Empty(t, overview.OrphanRules)
	assert.Empty(t, overview.OrphanPolicies)
	assert.Empty(t, overview.UnusedLists)
}

func TestBuild_FlagsDanglingReferences(t *testing.T) {
	accounts, workspaces, _, _, _ := graphFixture()
	// Policy, rule, and list referenced by the workspace/rule no longer exist.
	workspaces[0].RulesIDs = []string{"rule-gone"}
	workspaces[0].PolicyID = "policy-gone"

	overview := Build(accounts, workspaces, nil, nil, nil)

	require.Len(t, overview.Accounts, 1)
	acct := overview.Accounts[0]

	require.NotNil(t, acct.Policy)
	assert.True(t, acct.Policy.Missing, "deleted policy must be flagged, not hidden")
	assert.Equal(t, "policy-gone", acct.Policy.ID)

	require.Len(t, acct.Rules, 1)
	assert.True(t, acct.Rules[0].Missing, "dangling rule_ids entry must be flagged")
	assert.Equal(t, "rule-gone", acct.Rules[0].ID)
}

func TestBuild_FlagsMissingListReference(t *testing.T) {
	accounts, workspaces, policies, rules, _ := graphFixture()

	overview := Build(accounts, workspaces, policies, rules, nil)

	require.Len(t, overview.Accounts, 1)
	require.Len(t, overview.Accounts[0].Rules, 1)
	ruleLists := overview.Accounts[0].Rules[0].Lists
	require.Len(t, ruleLists, 1)
	assert.True(t, ruleLists[0].Missing, "in_list condition pointing at a deleted list must be flagged")
	assert.Equal(t, "list-1", ruleLists[0].ID)
}

func TestBuild_ResolvesAccountSettingsPolicy(t *testing.T) {
	// A policy can be attached at the account level (Settings.PolicyID) with
	// the workspace carrying no policy_id; it must show on the tree and must
	// not be reported as orphaned.
	accounts, workspaces, policies, rules, lists := graphFixture()
	workspaces[0].PolicyID = ""
	accounts[0].Settings.PolicyID = "policy-1"

	overview := Build(accounts, workspaces, policies, rules, lists)

	require.Len(t, overview.Accounts, 1)
	require.NotNil(t, overview.Accounts[0].Policy, "settings-attached policy must appear on the tree")
	assert.Equal(t, "policy-1", overview.Accounts[0].Policy.ID)
	assert.False(t, overview.Accounts[0].Policy.Missing)
	assert.Empty(t, overview.OrphanPolicies, "settings-attached policy must not be reported orphaned")
}

func TestBuild_SettingsPolicyWithoutWorkspace(t *testing.T) {
	accounts, _, policies, _, _ := graphFixture()
	accounts[0].WorkspaceID = ""
	accounts[0].Settings.PolicyID = "policy-1"

	overview := Build(accounts, nil, policies, nil, nil)

	require.Len(t, overview.Accounts, 1)
	require.NotNil(t, overview.Accounts[0].Policy, "settings policy must resolve even without a workspace")
	assert.Equal(t, "policy-1", overview.Accounts[0].Policy.ID)
	assert.Empty(t, overview.OrphanPolicies)
}

func TestBuild_FlagsMissingWorkspace(t *testing.T) {
	accounts, _, _, _, _ := graphFixture()

	overview := Build(accounts, nil, nil, nil, nil)

	require.Len(t, overview.Accounts, 1)
	assert.True(t, overview.Accounts[0].WorkspaceMissing, "account pointing at a deleted workspace must be flagged")
}

func TestBuild_CountsSharedWorkspaces(t *testing.T) {
	accounts, workspaces, policies, rules, lists := graphFixture()
	accounts = append(accounts, domain.AgentAccount{
		ID: "grant-2", Email: "second@app.nylas.email", GrantStatus: "valid", WorkspaceID: "ws-1",
	})

	overview := Build(accounts, workspaces, policies, rules, lists)

	require.Len(t, overview.Accounts, 2)
	// Both accounts share ws-1, so each sees one other account on its workspace.
	assert.Equal(t, 1, overview.Accounts[0].SharedWith, "auto-group sharing must be surfaced")
	assert.Equal(t, 1, overview.Accounts[1].SharedWith)
}

func TestBuild_ReportsOrphansAndUnused(t *testing.T) {
	accounts, workspaces, policies, rules, lists := graphFixture()
	policies = append(policies, domain.Policy{ID: "policy-orphan", Name: "Unattached Policy"})
	rules = append(rules, domain.Rule{ID: "rule-orphan", Name: "Unattached Rule"})
	lists = append(lists, domain.AgentList{ID: "list-unused", Name: "Unused list", Type: "tld"})

	overview := Build(accounts, workspaces, policies, rules, lists)

	require.Len(t, overview.OrphanPolicies, 1)
	assert.Equal(t, "policy-orphan", overview.OrphanPolicies[0].ID)
	require.Len(t, overview.OrphanRules, 1)
	assert.Equal(t, "rule-orphan", overview.OrphanRules[0].ID)
	require.Len(t, overview.UnusedLists, 1)
	assert.Equal(t, "list-unused", overview.UnusedLists[0].ID)
}

func TestBuild_NilEnabledMeansEnabled(t *testing.T) {
	// The API omits "enabled" for rules that are on; a nil pointer must render
	// as enabled, not panic and not show as disabled.
	accounts, workspaces, policies, rules, lists := graphFixture()
	rules[0].Enabled = nil

	overview := Build(accounts, workspaces, policies, rules, lists)

	require.Len(t, overview.Accounts, 1)
	require.Len(t, overview.Accounts[0].Rules, 1)
	assert.True(t, overview.Accounts[0].Rules[0].Enabled)
}

func TestBuild_IncludesWorkspaceViews(t *testing.T) {
	accounts, workspaces, policies, rules, lists := graphFixture()
	// A workspace with no member accounts must still appear on the board.
	workspaces = append(workspaces, domain.Workspace{ID: "ws-empty", Name: "Empty workspace"})

	overview := Build(accounts, workspaces, policies, rules, lists)

	require.Len(t, overview.Workspaces, 2)

	ws := overview.Workspaces[0]
	assert.Equal(t, "ws-1", ws.ID)
	assert.Equal(t, "Support workspace", ws.Name)
	assert.True(t, ws.Default)
	assert.True(t, ws.AutoGroup)
	require.NotNil(t, ws.Policy)
	assert.Equal(t, "policy-1", ws.Policy.ID)
	require.Len(t, ws.Rules, 1)
	assert.Equal(t, "rule-1", ws.Rules[0].ID)
	require.Len(t, ws.Accounts, 1)
	assert.Equal(t, "support@app.nylas.email", ws.Accounts[0].Email)

	empty := overview.Workspaces[1]
	assert.Equal(t, "ws-empty", empty.ID)
	assert.Nil(t, empty.Policy)
	assert.Empty(t, empty.Accounts)
}

func TestConditionListIDs(t *testing.T) {
	tests := []struct {
		name      string
		condition domain.RuleCondition
		want      []string
	}{
		{
			name:      "non in_list operator yields nothing",
			condition: domain.RuleCondition{Field: "from.domain", Operator: "is", Value: "example.com"},
			want:      nil,
		},
		{
			name:      "single string value",
			condition: domain.RuleCondition{Field: "from.domain", Operator: "in_list", Value: "list-1"},
			want:      []string{"list-1"},
		},
		{
			name:      "array of values from JSON decode",
			condition: domain.RuleCondition{Field: "from.domain", Operator: "in_list", Value: []any{"list-1", "list-2"}},
			want:      []string{"list-1", "list-2"},
		},
		{
			name:      "string slice value",
			condition: domain.RuleCondition{Field: "from.domain", Operator: "in_list", Value: []string{"list-1"}},
			want:      []string{"list-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, conditionListIDs(tt.condition))
		})
	}
}
