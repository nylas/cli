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

type agentPolicyScope struct {
	AllPolicies    []domain.Policy
	AgentPolicies  []domain.Policy
	PolicyRefsByID map[string][]policyAgentAccountRef
}

type resolvedRuleScope struct {
	Rule               *domain.Rule
	SelectedRefs       []rulePolicyRef
	AllAgentRefs       []rulePolicyRef
	AllAgentPolicies   []domain.Policy
	SharedOutsideAgent bool
}

func newRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage agent rules",
		Long: `Manage rules used by policies attached to agent accounts.

Rules are backed by the /v3/rules API. The agent namespace scopes them through
policies that are attached to provider=nylas accounts. This surface manages
both inbound and outbound rules attached to those policies.

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

func loadAgentPolicyScope(ctx context.Context, client ports.NylasClient) (*agentPolicyScope, error) {
	policies, err := client.ListPolicies(ctx)
	if err != nil {
		return nil, common.WrapListError("policies", err)
	}

	accounts, err := client.ListAgentAccounts(ctx)
	if err != nil {
		return nil, common.WrapListError("agent accounts", err)
	}

	refsByPolicyID := buildPolicyAccountRefs(accounts)
	agentPolicies := filterPoliciesWithAgentAccounts(policies, refsByPolicyID)

	return &agentPolicyScope{
		AllPolicies:    policies,
		AgentPolicies:  agentPolicies,
		PolicyRefsByID: refsByPolicyID,
	}, nil
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
	if defaultPolicyID == "" {
		return nil, nil, common.NewUserError(
			"default agent account does not have a policy",
			"Pass --policy-id or attach a policy to the active provider=nylas account first",
		)
	}

	policy, err := client.GetPolicy(ctx, defaultPolicyID)
	if err != nil {
		return nil, nil, common.WrapGetError("policy", err)
	}

	return policy, []policyAgentAccountRef{{
		GrantID: account.ID,
		Email:   account.Email,
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

func buildRuleRefsByID(policies []domain.Policy, refsByPolicyID map[string][]policyAgentAccountRef) map[string][]rulePolicyRef {
	refsByRuleID := make(map[string][]rulePolicyRef)
	for _, policy := range policies {
		accounts := refsByPolicyID[policy.ID]
		if len(accounts) == 0 {
			continue
		}

		seen := make(map[string]struct{}, len(policy.Rules))
		for _, ruleID := range policy.Rules {
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

	refsByRuleID := buildRuleRefsByID(scope.AgentPolicies, scope.PolicyRefsByID)
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
		AllAgentPolicies:   scope.AgentPolicies,
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

func refreshPolicies(ctx context.Context, client ports.NylasClient, policies []domain.Policy) ([]domain.Policy, error) {
	refreshed := make([]domain.Policy, 0, len(policies))
	for _, policy := range policies {
		latest, err := client.GetPolicy(ctx, policy.ID)
		if err != nil {
			return nil, err
		}
		refreshed = append(refreshed, *latest)
	}
	return refreshed, nil
}

func policiesLeftEmptyByRuleRemoval(ctx context.Context, client interface {
	GetRule(context.Context, string) (*domain.Rule, error)
}, policies []domain.Policy, ruleID string) ([]domain.Policy, error) {
	blocking := make([]domain.Policy, 0)
	for _, policy := range policies {
		if !policyContainsRule(policy, ruleID) {
			continue
		}

		liveRemaining := false
		for _, candidate := range removeString(policy.Rules, ruleID) {
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
			blocking = append(blocking, policy)
		}
	}
	return blocking, nil
}

func attachRuleToPolicy(ctx context.Context, client ports.NylasClient, policy domain.Policy, ruleID string) error {
	updatedRules := appendUniqueString(policy.Rules, ruleID)
	if slices.Equal(updatedRules, policy.Rules) {
		return nil
	}

	_, err := client.UpdatePolicy(ctx, policy.ID, map[string]any{"rules": updatedRules})
	return err
}

func detachRuleFromPolicies(ctx context.Context, client ports.NylasClient, policies []domain.Policy, ruleID string) (func(context.Context) error, error) {
	originalRulesByPolicyID := make(map[string][]string)
	updatedPolicyIDs := make([]string, 0)

	for _, policy := range policies {
		if !policyContainsRule(policy, ruleID) {
			continue
		}

		originalRulesByPolicyID[policy.ID] = append([]string(nil), policy.Rules...)
		updatedRules := removeString(policy.Rules, ruleID)
		if _, err := client.UpdatePolicy(ctx, policy.ID, map[string]any{"rules": updatedRules}); err != nil {
			if rollbackErr := rollbackPolicyRuleUpdates(ctx, client, originalRulesByPolicyID, updatedPolicyIDs); rollbackErr != nil {
				return nil, fmt.Errorf("failed to detach rule from policy %s: %w (rollback failed: %v)", policy.ID, err, rollbackErr)
			}
			return nil, err
		}
		updatedPolicyIDs = append(updatedPolicyIDs, policy.ID)
	}

	return func(ctx context.Context) error {
		return rollbackPolicyRuleUpdates(ctx, client, originalRulesByPolicyID, updatedPolicyIDs)
	}, nil
}

func rollbackPolicyRuleUpdates(ctx context.Context, client ports.NylasClient, originalRulesByPolicyID map[string][]string, updatedPolicyIDs []string) error {
	var failures []string
	for _, policyID := range updatedPolicyIDs {
		if _, err := client.UpdatePolicy(ctx, policyID, map[string]any{"rules": originalRulesByPolicyID[policyID]}); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", policyID, err))
		}
	}

	if len(failures) > 0 {
		return fmt.Errorf("failed to rollback policy updates: %s", strings.Join(failures, "; "))
	}
	return nil
}

func printRuleSummary(rule domain.Rule, index int, refs []rulePolicyRef) {
	fmt.Printf("%d. %-32s %s\n", index+1, common.Cyan.Sprint(rule.Name), common.Dim.Sprint(rule.ID))
	if !rule.UpdatedAt.IsZero() {
		_, _ = common.Dim.Printf("   Updated: %s\n", common.FormatTimeAgo(rule.UpdatedAt.Time))
	}
	for _, ref := range refs {
		_, _ = common.Dim.Printf("   Policy: %s (%s)\n", ref.PolicyName, ref.PolicyID)
		for _, account := range ref.Accounts {
			_, _ = common.Dim.Printf("   Agent: %s (%s)\n", account.Email, account.GrantID)
		}
	}
}

func printRuleDetails(rule domain.Rule, refs []rulePolicyRef) {
	fmt.Printf("Rule:         %s\n", rule.Name)
	fmt.Printf("ID:           %s\n", rule.ID)
	if rule.Description != "" {
		fmt.Printf("Description:  %s\n", rule.Description)
	}
	if rule.Priority != nil {
		fmt.Printf("Priority:     %d\n", *rule.Priority)
	}
	if rule.Enabled != nil {
		fmt.Printf("Enabled:      %t\n", *rule.Enabled)
	}
	if rule.Trigger != "" {
		fmt.Printf("Trigger:      %s\n", rule.Trigger)
	}
	if rule.ApplicationID != "" {
		fmt.Printf("Application:  %s\n", rule.ApplicationID)
	}
	if rule.OrganizationID != "" {
		fmt.Printf("Organization: %s\n", rule.OrganizationID)
	}
	if !rule.CreatedAt.IsZero() {
		fmt.Printf("Created:      %s (%s)\n", rule.CreatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(rule.CreatedAt.Time))
	}
	if !rule.UpdatedAt.IsZero() {
		fmt.Printf("Updated:      %s (%s)\n", rule.UpdatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(rule.UpdatedAt.Time))
	}

	printRuleRefsSection(refs)
	printRuleMatchSection(rule.Match)
	printRuleActionsSection(rule.Actions)
	fmt.Println()
}

func printRuleRefsSection(refs []rulePolicyRef) {
	printPolicySectionHeader("Policies")
	if len(refs) == 0 {
		fmt.Println("  none")
		return
	}

	for _, ref := range refs {
		printPolicyField("Policy", fmt.Sprintf("%s (%s)", ref.PolicyName, ref.PolicyID))
		if len(ref.Accounts) == 0 {
			continue
		}
		for _, account := range ref.Accounts {
			printPolicyField("Agent", fmt.Sprintf("%s (%s)", account.Email, account.GrantID))
		}
	}
}

func printRuleMatchSection(match *domain.RuleMatch) {
	printPolicySectionHeader("Match")
	if match == nil {
		fmt.Println("  none")
		return
	}

	if match.Operator != "" {
		printPolicyField("Operator", match.Operator)
	}
	if len(match.Conditions) == 0 {
		fmt.Println("  Conditions: none")
		return
	}

	fmt.Println("  Conditions:")
	for i, condition := range match.Conditions {
		fmt.Printf("    %d. %s %s %s\n", i+1, condition.Field, condition.Operator, formatRuleValue(condition.Value))
	}
}

func printRuleActionsSection(actions []domain.RuleAction) {
	printPolicySectionHeader("Actions")
	if len(actions) == 0 {
		fmt.Println("  none")
		return
	}

	for i, action := range actions {
		if action.Value == nil {
			fmt.Printf("  %d. %s\n", i+1, action.Type)
			continue
		}
		fmt.Printf("  %d. %s => %s\n", i+1, action.Type, formatRuleValue(action.Value))
	}
}

func formatRuleValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "none"
	case string:
		return v
	case []string:
		return strings.Join(v, ", ")
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			parts = append(parts, formatRuleValue(item))
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", v)
	}
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
