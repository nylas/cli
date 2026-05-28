package agent

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

type policyLookupClient struct {
	policies map[string]domain.Policy
}

func (c policyLookupClient) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	policy, ok := c.policies[policyID]
	if !ok {
		return nil, domain.ErrPolicyNotFound
	}
	return &policy, nil
}

type workspaceLookupClient struct {
	workspaces map[string]*domain.Workspace
	err        error
}

func (c workspaceLookupClient) GetWorkspace(ctx context.Context, workspaceID string) (*domain.Workspace, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.workspaces[workspaceID], nil
}


func TestUpsertAgentAccount(t *testing.T) {
	accounts := []domain.AgentAccount{
		{
			ID:       "grant-a",
			Email:    "old@example.com",
			Provider: domain.ProviderNylas,
			Settings: domain.AgentAccountSettings{PolicyID: "policy-old"},
		},
	}

	updated := upsertAgentAccount(accounts, domain.AgentAccount{
		ID:       "grant-a",
		Email:    "agent@example.com",
		Provider: domain.ProviderNylas,
		Settings: domain.AgentAccountSettings{PolicyID: "policy-new"},
	})
	if assert.Len(t, updated, 1) {
		assert.Equal(t, "agent@example.com", updated[0].Email)
		assert.Equal(t, "policy-new", updated[0].Settings.PolicyID)
	}
	assert.Equal(t, "old@example.com", accounts[0].Email)

	updated = upsertAgentAccount(updated, domain.AgentAccount{
		ID:       "grant-b",
		Email:    "beta@example.com",
		Provider: domain.ProviderNylas,
		Settings: domain.AgentAccountSettings{PolicyID: "policy-new"},
	})
	if assert.Len(t, updated, 2) {
		assert.Equal(t, "grant-b", updated[1].ID)
	}

	updated = upsertAgentAccount(updated, domain.AgentAccount{
		ID:       "grant-google",
		Email:    "mail@example.com",
		Provider: domain.ProviderGoogle,
	})
	assert.Len(t, updated, 2)
}

func TestUpsertPoliciesForAgentAccounts(t *testing.T) {
	policies := []domain.Policy{{ID: "policy-existing", Name: "Existing"}}
	accounts := []domain.AgentAccount{
		{
			ID:       "grant-existing",
			Provider: domain.ProviderNylas,
			Settings: domain.AgentAccountSettings{
				PolicyID: "policy-existing",
			},
		},
		{
			ID:       "grant-fresh",
			Provider: domain.ProviderNylas,
			Settings: domain.AgentAccountSettings{
				PolicyID: "policy-fresh",
			},
		},
		{
			ID:       "grant-dangling",
			Provider: domain.ProviderNylas,
			Settings: domain.AgentAccountSettings{
				PolicyID: "policy-missing",
			},
		},
	}
	client := policyLookupClient{
		policies: map[string]domain.Policy{
			"policy-fresh": {ID: "policy-fresh", Name: "Fresh"},
		},
	}

	updated, err := upsertPoliciesForAgentAccounts(context.Background(), client, policies, accounts)
	assert.NoError(t, err)
	if assert.Len(t, updated, 2) {
		assert.Equal(t, "policy-existing", updated[0].ID)
		assert.Equal(t, "policy-fresh", updated[1].ID)
	}
	assert.Len(t, policies, 1)
}

func TestLoadAgentWorkspacesFailsClosedOnLookupError(t *testing.T) {
	accounts := []domain.AgentAccount{{
		ID:          "grant-1",
		Provider:    domain.ProviderNylas,
		WorkspaceID: "workspace-1",
	}}

	workspaces, err := loadAgentWorkspaces(context.Background(), workspaceLookupClient{err: domain.ErrWorkspaceNotFound}, accounts)

	assert.Error(t, err)
	assert.Nil(t, workspaces)
}

func TestBuildWorkspaceRuleIDsByPolicyKeepsEmptyWorkspaceRules(t *testing.T) {
	accounts := []domain.AgentAccount{{
		ID:          "grant-1",
		Provider:    domain.ProviderNylas,
		WorkspaceID: "workspace-1",
	}}
	workspacesByID := map[string]*domain.Workspace{
		"workspace-1": {ID: "workspace-1", PolicyID: "policy-1", RulesIDs: nil},
	}

	ruleIDsByPolicy := buildWorkspaceRuleIDsByPolicy(accounts, workspacesByID)

	ruleIDs, ok := ruleIDsByPolicy["policy-1"]
	assert.True(t, ok)
	assert.Empty(t, ruleIDs)
}

func TestUpsertPoliciesForAgentAccountsUsesWorkspacePolicy(t *testing.T) {
	policies := []domain.Policy{{ID: "policy-existing", Name: "Existing"}}
	accounts := []domain.AgentAccount{{
		ID:          "grant-fresh",
		Provider:    domain.ProviderNylas,
		WorkspaceID: "workspace-1",
		Settings: domain.AgentAccountSettings{
			PolicyID: "legacy-policy",
		},
	}}
	workspacesByID := map[string]*domain.Workspace{
		"workspace-1": {ID: "workspace-1", PolicyID: "policy-fresh"},
	}
	client := policyLookupClient{
		policies: map[string]domain.Policy{
			"policy-fresh": {ID: "policy-fresh", Name: "Fresh"},
		},
	}

	updated, err := upsertPoliciesForAgentAccountsWithWorkspaces(context.Background(), client, policies, accounts, workspacesByID)

	assert.NoError(t, err)
	if assert.Len(t, updated, 2) {
		assert.Equal(t, "policy-existing", updated[0].ID)
		assert.Equal(t, "policy-fresh", updated[1].ID)
	}
}

