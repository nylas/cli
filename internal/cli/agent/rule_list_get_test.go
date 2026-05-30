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

func TestDetachRuleFromAgentWorkspacesRemovesAndRollsBackWorkspaceRule(t *testing.T) {
	client := &workspaceRuleTestClient{
		workspaces: map[string]*domain.Workspace{
			"workspace-1": {ID: "workspace-1", RulesIDs: []string{"rule-1", "rule-2"}},
		},
		updates: make(map[string][]string),
	}
	accounts := []policyAgentAccountRef{{
		GrantID:     "grant-1",
		WorkspaceID: "workspace-1",
	}}

	rollback, err := detachRuleFromAgentWorkspaces(context.Background(), client, accounts, "rule-1")

	require.NoError(t, err)
	assert.Equal(t, []string{"rule-2"}, client.workspaces["workspace-1"].RulesIDs)
	assert.Equal(t, []string{"rule-2"}, client.updates["workspace-1"])

	require.NoError(t, rollback(context.Background()))
	assert.Equal(t, []string{"rule-1", "rule-2"}, client.workspaces["workspace-1"].RulesIDs)
}
