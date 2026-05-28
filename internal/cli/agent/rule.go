package agent

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

type rulePolicyRef struct {
	PolicyID   string
	PolicyName string
	Accounts   []policyAgentAccountRef
}

type resolvedRuleScope struct {
	Rule               *domain.Rule
	SelectedRefs       []rulePolicyRef
	AllAgentRefs       []rulePolicyRef
	SharedOutsideAgent bool
}

func newRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage agent rules",
		Long: `Manage rules attached to agent account workspaces.

Rules are backed by the /v3/rules API. The agent namespace scopes them through
provider=nylas account workspaces. This surface manages both inbound and
outbound rules attached to those workspaces.

Examples:
  nylas agent rule list
  nylas agent rule list --all
  nylas agent rule read <rule-id>
  nylas agent rule create --data-file rule.json
  nylas agent rule create --name "Archive outbound mail" --trigger outbound --condition recipient.domain,is,example.com --action archive
  nylas agent rule update <rule-id> --name "Updated Rule"
  nylas agent rule delete <rule-id> --yes`,
	}

	cmd.AddCommand(newRuleListCmd())
	cmd.AddCommand(newRuleGetCmd())
	cmd.AddCommand(newRuleReadCmd())
	cmd.AddCommand(newRuleCreateCmd())
	cmd.AddCommand(newRuleUpdateCmd())
	cmd.AddCommand(newRuleDeleteCmd())

	return cmd
}

func resolveDefaultAgentAccount(ctx context.Context, client ports.NylasClient) (*domain.AgentAccount, error) {
	grantID, err := common.GetGrantID(nil)
	if err != nil {
		return nil, common.WrapGetError("default grant", err)
	}

	account, err := client.GetAgentAccount(ctx, grantID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidGrant) {
			return nil, common.NewUserError(
				"default grant is not a nylas agent account",
				"Use 'nylas auth switch <grant-id>' to select a provider=nylas account",
			)
		}
		return nil, common.WrapGetError("default agent account", err)
	}

	return account, nil
}

func resolveAgentPolicy(ctx context.Context, client ports.NylasClient, policyID string) (*domain.Policy, []policyAgentAccountRef, error) {
	policyID = strings.TrimSpace(policyID)
	if policyID != "" {
		scope, err := loadAgentPolicyScope(ctx, client)
		if err != nil {
			return nil, nil, err
		}

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
	workspaceID := strings.TrimSpace(account.WorkspaceID)
	if workspaceID != "" {
		workspace, err := client.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, nil, common.WrapGetError("workspace", err)
		}
		if workspace == nil {
			return nil, nil, common.NewUserError("workspace not found", "The API returned an empty workspace response")
		}
		defaultPolicyID = strings.TrimSpace(workspace.PolicyID)
	}
	if defaultPolicyID == "" {
		return nil, nil, common.NewUserError(
			"default agent account does not have a policy",
			"Pass --policy-id or attach a policy to the active provider=nylas workspace first",
		)
	}

	policy, err := client.GetPolicy(ctx, defaultPolicyID)
	if err != nil {
		return nil, nil, common.WrapGetError("policy", err)
	}

	return policy, []policyAgentAccountRef{{
		GrantID:     account.ID,
		Email:       account.Email,
		WorkspaceID: workspaceID,
	}}, nil
}

func findPolicyByID(policies []domain.Policy, policyID string) *domain.Policy {
	for i := range policies {
		if policies[i].ID == policyID {
			return &policies[i]
		}
	}
	return nil
}

func buildRuleRefsByIDWithRuleIDs(policies []domain.Policy, refsByPolicyID map[string][]policyAgentAccountRef, ruleIDsByPolicy map[string][]string) map[string][]rulePolicyRef {
	refsByRuleID := make(map[string][]rulePolicyRef)
	for _, policy := range policies {
		accounts := refsByPolicyID[policy.ID]
		if len(accounts) == 0 {
			continue
		}

		ruleIDs := policy.Rules
		if workspaceRuleIDs, ok := ruleIDsByPolicy[policy.ID]; ok {
			ruleIDs = workspaceRuleIDs
		}
		seen := make(map[string]struct{}, len(ruleIDs))
		for _, ruleID := range ruleIDs {
			ruleID = strings.TrimSpace(ruleID)
			if ruleID == "" {
				continue
			}
			if _, ok := seen[ruleID]; ok {
				continue
			}
			seen[ruleID] = struct{}{}

			accountRefs := make([]policyAgentAccountRef, len(accounts))
			copy(accountRefs, accounts)

			refsByRuleID[ruleID] = append(refsByRuleID[ruleID], rulePolicyRef{
				PolicyID:   policy.ID,
				PolicyName: policy.Name,
				Accounts:   accountRefs,
			})
		}
	}

	for ruleID, refs := range refsByRuleID {
		slices.SortFunc(refs, func(a, b rulePolicyRef) int {
			if c := cmp.Compare(strings.ToLower(a.PolicyName), strings.ToLower(b.PolicyName)); c != 0 {
				return c
			}
			return cmp.Compare(a.PolicyID, b.PolicyID)
		})
		refsByRuleID[ruleID] = refs
	}

	return refsByRuleID
}

func filterRulesWithAgentPolicies(rules []domain.Rule, refsByRuleID map[string][]rulePolicyRef) []domain.Rule {
	filtered := make([]domain.Rule, 0, len(rules))
	for _, rule := range rules {
		if len(refsByRuleID[rule.ID]) == 0 {
			continue
		}
		filtered = append(filtered, rule)
	}
	return filtered
}

func resolveScopedRule(ctx context.Context, client ports.NylasClient, ruleID, policyID string, all bool) (*resolvedRuleScope, error) {
	scope, err := loadAgentPolicyScope(ctx, client)
	if err != nil {
		return nil, err
	}

	refsByRuleID := buildRuleRefsByIDWithRuleIDs(scope.AgentPolicies, scope.PolicyRefsByID, scope.RuleIDsByPolicy)
	allRefs := refsByRuleID[ruleID]
	if len(allRefs) == 0 {
		return nil, common.NewUserError(
			"rule is not attached to a nylas agent policy",
			"Use 'nylas agent rule list --all' to inspect provider=nylas rules",
		)
	}

	selectedRefs := allRefs
	if !all {
		targetPolicy, _, err := resolveAgentPolicyFromScope(ctx, client, scope, policyID)
		if err != nil {
			return nil, err
		}

		selectedRefs = filterRuleRefsByPolicyID(allRefs, targetPolicy.ID)
		if len(selectedRefs) == 0 {
			return nil, common.NewUserError(
				"rule is not attached to the selected policy",
				"Use 'nylas agent rule list --all' to inspect all agent-scoped rules",
			)
		}
	}

	rule, err := client.GetRule(ctx, ruleID)
	if err != nil {
		return nil, common.WrapGetError("rule", err)
	}

	return &resolvedRuleScope{
		Rule:               rule,
		SelectedRefs:       selectedRefs,
		AllAgentRefs:       allRefs,
		SharedOutsideAgent: ruleReferencedOutsideAgentScope(scope.AllPolicies, scope.AgentPolicies, ruleID),
	}, nil
}

func filterRuleRefsByPolicyID(refs []rulePolicyRef, policyID string) []rulePolicyRef {
	filtered := make([]rulePolicyRef, 0, len(refs))
	for _, ref := range refs {
		if ref.PolicyID == policyID {
			filtered = append(filtered, ref)
		}
	}
	return filtered
}

func ruleReferencedOutsideAgentScope(allPolicies, agentPolicies []domain.Policy, ruleID string) bool {
	agentPolicyIDs := make(map[string]struct{}, len(agentPolicies))
	for _, policy := range agentPolicies {
		agentPolicyIDs[policy.ID] = struct{}{}
	}

	for _, policy := range allPolicies {
		if !policyContainsRule(policy, ruleID) {
			continue
		}
		if _, ok := agentPolicyIDs[policy.ID]; !ok {
			return true
		}
	}

	return false
}

func policyContainsRule(policy domain.Policy, ruleID string) bool {
	for _, candidate := range policy.Rules {
		if strings.TrimSpace(candidate) == ruleID {
			return true
		}
	}
	return false
}

func appendUniqueString(items []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return append([]string(nil), items...)
	}

	updated := append([]string(nil), items...)
	if !slices.Contains(updated, value) {
		updated = append(updated, value)
	}
	return updated
}

func removeString(items []string, value string) []string {
	value = strings.TrimSpace(value)
	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) == value {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func attachRuleToAgentWorkspaces(ctx context.Context, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
	UpdateWorkspace(context.Context, string, *domain.UpdateWorkspaceRequest) (*domain.Workspace, error)
}, accounts []policyAgentAccountRef, ruleID string) error {
	seenWorkspaceIDs := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		workspaceID := strings.TrimSpace(account.WorkspaceID)
		if workspaceID == "" {
			continue
		}
		if _, seen := seenWorkspaceIDs[workspaceID]; seen {
			continue
		}
		seenWorkspaceIDs[workspaceID] = struct{}{}

		workspace, err := client.GetWorkspace(ctx, workspaceID)
		if err != nil {
			return err
		}
		if workspace == nil {
			return common.NewUserError("workspace not found", "The API returned an empty workspace response")
		}
		updatedRules := appendUniqueString(workspace.RulesIDs, ruleID)
		if slices.Equal(updatedRules, workspace.RulesIDs) {
			continue
		}
		if _, err := client.UpdateWorkspace(ctx, workspaceID, &domain.UpdateWorkspaceRequest{RulesIDs: &updatedRules}); err != nil {
			return err
		}
	}
	if len(seenWorkspaceIDs) == 0 {
		return common.NewUserError(
			"agent account has no workspace",
			"The selected provider=nylas account is missing a workspace to attach the rule to; reconnect the account and try again",
		)
	}
	return nil
}

func hasWorkspaceRefs(refs []rulePolicyRef) bool {
	for _, ref := range refs {
		for _, account := range ref.Accounts {
			if strings.TrimSpace(account.WorkspaceID) != "" {
				return true
			}
		}
	}
	return false
}

func workspacesLeftEmptyByRuleRemoval(ctx context.Context, client interface {
	GetRule(context.Context, string) (*domain.Rule, error)
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
}, refs []rulePolicyRef, ruleID string) ([]string, error) {
	workspaces, err := loadReferencedWorkspaces(ctx, client, refs)
	if err != nil {
		return nil, err
	}

	blocking := make([]string, 0)
	for _, workspace := range workspaces {
		if !stringSliceContains(workspace.RulesIDs, ruleID) {
			continue
		}

		liveRemaining := false
		for _, candidate := range removeString(workspace.RulesIDs, ruleID) {
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}

			_, err := client.GetRule(ctx, candidate)
			switch {
			case err == nil:
				liveRemaining = true
			case errors.Is(err, domain.ErrRuleNotFound):
				continue
			default:
				return nil, err
			}
			if liveRemaining {
				break
			}
		}
		if !liveRemaining {
			blocking = append(blocking, formatWorkspaceRef(workspace))
		}
	}
	return blocking, nil
}

func detachRuleFromAgentWorkspaces(ctx context.Context, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
	UpdateWorkspace(context.Context, string, *domain.UpdateWorkspaceRequest) (*domain.Workspace, error)
}, refs []rulePolicyRef, ruleID string) (func(context.Context) error, error) {
	workspaces, err := loadReferencedWorkspaces(ctx, client, refs)
	if err != nil {
		return nil, err
	}

	originalRulesByWorkspaceID := make(map[string][]string)
	updatedWorkspaceIDs := make([]string, 0)

	for _, workspace := range workspaces {
		if !stringSliceContains(workspace.RulesIDs, ruleID) {
			continue
		}

		originalRulesByWorkspaceID[workspace.ID] = append([]string(nil), workspace.RulesIDs...)
		updatedRules := removeString(workspace.RulesIDs, ruleID)
		if _, err := client.UpdateWorkspace(ctx, workspace.ID, &domain.UpdateWorkspaceRequest{RulesIDs: &updatedRules}); err != nil {
			if rollbackErr := rollbackWorkspaceRuleUpdates(ctx, client, originalRulesByWorkspaceID, updatedWorkspaceIDs); rollbackErr != nil {
				return nil, fmt.Errorf("failed to detach rule from workspace %s: %w (rollback failed: %v)", workspace.ID, err, rollbackErr)
			}
			return nil, err
		}
		updatedWorkspaceIDs = append(updatedWorkspaceIDs, workspace.ID)
	}

	return func(ctx context.Context) error {
		return rollbackWorkspaceRuleUpdates(ctx, client, originalRulesByWorkspaceID, updatedWorkspaceIDs)
	}, nil
}

func loadReferencedWorkspaces(ctx context.Context, client interface {
	GetWorkspace(context.Context, string) (*domain.Workspace, error)
}, refs []rulePolicyRef) ([]domain.Workspace, error) {
	seenWorkspaceIDs := make(map[string]struct{})
	workspaces := make([]domain.Workspace, 0)
	for _, ref := range refs {
		for _, account := range ref.Accounts {
			workspaceID := strings.TrimSpace(account.WorkspaceID)
			if workspaceID == "" {
				continue
			}
			if _, seen := seenWorkspaceIDs[workspaceID]; seen {
				continue
			}
			seenWorkspaceIDs[workspaceID] = struct{}{}

			workspace, err := client.GetWorkspace(ctx, workspaceID)
			if err != nil {
				return nil, err
			}
			if workspace == nil {
				return nil, common.NewUserError("workspace not found", "The API returned an empty workspace response")
			}
			workspaces = append(workspaces, *workspace)
		}
	}
	return workspaces, nil
}

func rollbackWorkspaceRuleUpdates(ctx context.Context, client interface {
	UpdateWorkspace(context.Context, string, *domain.UpdateWorkspaceRequest) (*domain.Workspace, error)
}, originalRulesByWorkspaceID map[string][]string, updatedWorkspaceIDs []string) error {
	var failures []string
	for _, workspaceID := range updatedWorkspaceIDs {
		rules := append([]string(nil), originalRulesByWorkspaceID[workspaceID]...)
		if _, err := client.UpdateWorkspace(ctx, workspaceID, &domain.UpdateWorkspaceRequest{RulesIDs: &rules}); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", workspaceID, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to rollback workspace updates: %s", strings.Join(failures, "; "))
	}
	return nil
}

func stringSliceContains(items []string, value string) bool {
	value = strings.TrimSpace(value)
	for _, item := range items {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}

func formatWorkspaceRef(workspace domain.Workspace) string {
	name := strings.TrimSpace(workspace.Name)
	if name == "" {
		return workspace.ID
	}
	return fmt.Sprintf("%s (%s)", name, workspace.ID)
}
