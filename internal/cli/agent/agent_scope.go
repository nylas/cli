package agent

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type agentPolicyScope struct {
	AllPolicies     []domain.Policy
	AgentPolicies   []domain.Policy
	PolicyRefsByID  map[string][]policyAgentAccountRef
	WorkspacesByID  map[string]*domain.Workspace
	RuleIDsByPolicy map[string][]string
}

func loadAgentPolicyScope(ctx context.Context, client ports.NylasClient) (*agentPolicyScope, error) {
	policies, err := client.ListPolicies(ctx)
	if err != nil {
		return nil, common.WrapListError("policies", err)
	}

	accounts, err := listAgentAccountsForPolicyScope(ctx, client)
	if err != nil {
		return nil, err
	}
	workspacesByID, err := loadAgentWorkspaces(ctx, client, accounts)
	if err != nil {
		return nil, err
	}
	policies, err = upsertPoliciesForAgentAccountsWithWorkspaces(ctx, client, policies, accounts, workspacesByID)
	if err != nil {
		return nil, err
	}

	refsByPolicyID := buildPolicyAccountRefsWithWorkspaces(accounts, workspacesByID)
	agentPolicies := filterPoliciesWithAgentAccounts(policies, refsByPolicyID)

	return &agentPolicyScope{
		AllPolicies:     policies,
		AgentPolicies:   agentPolicies,
		PolicyRefsByID:  refsByPolicyID,
		WorkspacesByID:  workspacesByID,
		RuleIDsByPolicy: buildWorkspaceRuleIDsByPolicy(accounts, workspacesByID),
	}, nil
}

func loadAgentWorkspaces(ctx context.Context, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
}, accounts []domain.AgentAccount) (map[string]*domain.Workspace, error) {
	workspacesByID := make(map[string]*domain.Workspace)
	for _, account := range accounts {
		workspaceID := strings.TrimSpace(account.WorkspaceID)
		if workspaceID == "" {
			continue
		}
		if _, seen := workspacesByID[workspaceID]; seen {
			continue
		}
		workspace, err := client.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, common.WrapGetError("workspace", err)
		}
		if workspace == nil {
			return nil, common.NewUserError("workspace not found", "The API returned an empty workspace response")
		}
		workspacesByID[workspaceID] = workspace
	}
	return workspacesByID, nil
}

func buildWorkspaceRuleIDsByPolicy(accounts []domain.AgentAccount, workspacesByID map[string]*domain.Workspace) map[string][]string {
	ruleIDsByPolicy := make(map[string][]string)
	for _, account := range accounts {
		workspace := workspacesByID[strings.TrimSpace(account.WorkspaceID)]
		if workspace == nil {
			continue
		}
		policyID := strings.TrimSpace(workspace.PolicyID)
		if policyID == "" {
			continue
		}
		if _, ok := ruleIDsByPolicy[policyID]; !ok {
			ruleIDsByPolicy[policyID] = []string{}
		}
		for _, ruleID := range workspace.RulesIDs {
			ruleIDsByPolicy[policyID] = appendUniqueString(ruleIDsByPolicy[policyID], ruleID)
		}
	}
	return ruleIDsByPolicy
}

func listAgentAccountsForPolicyScope(ctx context.Context, client ports.NylasClient) ([]domain.AgentAccount, error) {
	accounts, err := client.ListAgentAccounts(ctx)
	if err != nil {
		return nil, common.WrapListError("agent accounts", err)
	}

	defaultAccount := getConfiguredDefaultAgentAccount(ctx, client)
	if defaultAccount == nil {
		return accounts, nil
	}

	return upsertAgentAccount(accounts, *defaultAccount), nil
}

func getConfiguredDefaultAgentAccount(ctx context.Context, client interface {
	GetAgentAccount(context.Context, string) (*domain.AgentAccount, error)
}) *domain.AgentAccount {
	grantID := strings.TrimSpace(os.Getenv("NYLAS_GRANT_ID"))
	if grantID == "" {
		var err error
		grantID, err = common.GetGrantID(nil)
		if err != nil {
			return nil
		}
		grantID = strings.TrimSpace(grantID)
	}
	if grantID == "" {
		return nil
	}

	account, err := client.GetAgentAccount(ctx, grantID)
	if err != nil {
		return nil
	}
	return account
}

func upsertAgentAccount(accounts []domain.AgentAccount, account domain.AgentAccount) []domain.AgentAccount {
	if strings.TrimSpace(account.ID) == "" || account.Provider != domain.ProviderNylas {
		return accounts
	}

	merged := append([]domain.AgentAccount(nil), accounts...)
	for i := range merged {
		if merged[i].ID == account.ID {
			merged[i] = account
			return merged
		}
	}

	return append(merged, account)
}

func upsertPoliciesForAgentAccounts(ctx context.Context, client interface {
	GetPolicy(context.Context, string) (*domain.Policy, error)
}, policies []domain.Policy, accounts []domain.AgentAccount) ([]domain.Policy, error) {
	return upsertPoliciesForAgentAccountsWithWorkspaces(ctx, client, policies, accounts, nil)
}

func upsertPoliciesForAgentAccountsWithWorkspaces(ctx context.Context, client interface {
	GetPolicy(context.Context, string) (*domain.Policy, error)
}, policies []domain.Policy, accounts []domain.AgentAccount, workspacesByID map[string]*domain.Workspace) ([]domain.Policy, error) {
	merged := append([]domain.Policy(nil), policies...)
	seenPolicyIDs := make(map[string]struct{}, len(merged))
	for _, policy := range merged {
		seenPolicyIDs[policy.ID] = struct{}{}
	}

	for _, account := range accounts {
		policyID := strings.TrimSpace(account.Settings.PolicyID)
		if workspace := workspacesByID[strings.TrimSpace(account.WorkspaceID)]; workspace != nil {
			policyID = strings.TrimSpace(workspace.PolicyID)
		}
		if policyID == "" {
			continue
		}
		if _, seen := seenPolicyIDs[policyID]; seen {
			continue
		}

		policy, err := client.GetPolicy(ctx, policyID)
		if err != nil {
			if errors.Is(err, domain.ErrPolicyNotFound) {
				continue
			}
			return nil, common.WrapGetError("policy", err)
		}
		merged = append(merged, *policy)
		seenPolicyIDs[policy.ID] = struct{}{}
	}

	return merged, nil
}
