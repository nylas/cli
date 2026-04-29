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
	AllPolicies    []domain.Policy
	AgentPolicies  []domain.Policy
	PolicyRefsByID map[string][]policyAgentAccountRef
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
	policies, err = upsertPoliciesForAgentAccounts(ctx, client, policies, accounts)
	if err != nil {
		return nil, err
	}

	refsByPolicyID := buildPolicyAccountRefs(accounts)
	agentPolicies := filterPoliciesWithAgentAccounts(policies, refsByPolicyID)

	return &agentPolicyScope{
		AllPolicies:    policies,
		AgentPolicies:  agentPolicies,
		PolicyRefsByID: refsByPolicyID,
	}, nil
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
	merged := append([]domain.Policy(nil), policies...)
	seenPolicyIDs := make(map[string]struct{}, len(merged))
	for _, policy := range merged {
		seenPolicyIDs[policy.ID] = struct{}{}
	}

	for _, account := range accounts {
		policyID := strings.TrimSpace(account.Settings.PolicyID)
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

func resolveAgentPolicyFromScope(ctx context.Context, client ports.NylasClient, scope *agentPolicyScope, policyID string) (*domain.Policy, []policyAgentAccountRef, error) {
	policyID = strings.TrimSpace(policyID)
	if policyID != "" {
		policy := findPolicyByID(scope.AgentPolicies, policyID)
		if policy == nil {
			return nil, nil, common.NewUserError(
				"policy is not attached to a nylas agent account",
				"Use 'nylas agent policy list --all' to inspect provider=nylas policies",
			)
		}

		return policy, scope.PolicyRefsByID[policyID], nil
	}

	account, err := resolveDefaultAgentAccount(ctx, client)
	if err != nil {
		return nil, nil, err
	}

	defaultPolicyID := strings.TrimSpace(account.Settings.PolicyID)
	if defaultPolicyID == "" {
		return nil, nil, common.NewUserError(
			"default agent account does not have a policy",
			"Pass --policy-id or attach a policy to the active provider=nylas account first",
		)
	}

	policy := findPolicyByID(scope.AgentPolicies, defaultPolicyID)
	if policy == nil {
		return nil, nil, common.NewUserError(
			"default agent account policy is not attached to a nylas agent account",
			"Use 'nylas agent policy list --all' to inspect provider=nylas policies",
		)
	}

	return policy, []policyAgentAccountRef{{
		GrantID: account.ID,
		Email:   account.Email,
	}}, nil
}
