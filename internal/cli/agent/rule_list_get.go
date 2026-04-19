package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newRuleListCmd() *cobra.Command {
	var (
		jsonOutput bool
		allRules   bool
		policyID   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List rules for the default agent policy",
		Long: `List rules for the current default agent policy.

By default, this command resolves the current default grant and lists the rules
attached to that provider=nylas account's policy. Use --policy-id to inspect a
specific agent policy, or --all to list every rule reachable from any
provider=nylas account policy.

Examples:
  nylas agent rule list
  nylas agent rule list --policy-id <policy-id>
  nylas agent rule list --all
  nylas agent rule list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRules && policyID != "" {
				return common.NewUserError("cannot combine --all with --policy-id", "Use either --all or --policy-id")
			}
			return runRuleList(jsonOutput, allRules, policyID)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&allRules, "all", false, "List all rules reachable from provider=nylas policies")
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy ID to scope the rule list to")

	return cmd
}

func runRuleList(jsonOutput, allRules bool, policyID string) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		if allRules {
			scope, err := loadAgentPolicyScope(ctx, client)
			if err != nil {
				return struct{}{}, err
			}

			refsByRuleID := buildRuleRefsByID(scope.AgentPolicies, scope.PolicyRefsByID)
			if len(refsByRuleID) == 0 {
				if jsonOutput {
					fmt.Println("[]")
					return struct{}{}, nil
				}
				common.PrintEmptyStateWithHint("rules attached to nylas agent policies", "Create a rule and attach it to a provider=nylas policy to see it here")
				return struct{}{}, nil
			}

			rules, err := client.ListRules(ctx)
			if err != nil {
				return struct{}{}, common.WrapListError("rules", err)
			}
			rules = filterRulesWithAgentPolicies(rules, refsByRuleID)

			if jsonOutput {
				return struct{}{}, common.PrintJSON(rules)
			}

			_, _ = common.BoldWhite.Printf("Rules (%d)\n\n", len(rules))
			for i, rule := range rules {
				printRuleSummary(rule, i, refsByRuleID[rule.ID])
			}
			fmt.Println()
			return struct{}{}, nil
		}

		scope, err := loadAgentPolicyScope(ctx, client)
		if err != nil {
			return struct{}{}, err
		}

		policy, accounts, err := resolveAgentPolicyFromScope(ctx, client, scope, policyID)
		if err != nil {
			return struct{}{}, err
		}

		ruleIDs := make([]string, 0, len(policy.Rules))
		for _, ruleID := range policy.Rules {
			ruleID = strings.TrimSpace(ruleID)
			if ruleID == "" {
				continue
			}
			ruleIDs = append(ruleIDs, ruleID)
		}

		if len(ruleIDs) == 0 {
			if jsonOutput {
				fmt.Println("[]")
				return struct{}{}, nil
			}
			common.PrintEmptyStateWithHint("rules on the selected agent policy", "Use 'nylas agent rule create --data-file rule.json' to add one")
			return struct{}{}, nil
		}

		allRulesList, err := client.ListRules(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("rules", err)
		}
		rules, ruleRefs := collectPolicyScopedRules(policy, accounts, allRulesList)
		if len(rules) == 0 {
			if jsonOutput {
				fmt.Println("[]")
				return struct{}{}, nil
			}
			common.PrintEmptyStateWithHint("rules on the selected agent policy", "Use 'nylas agent rule create --data-file rule.json' to add one")
			return struct{}{}, nil
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(rules)
		}

		_, _ = common.BoldWhite.Printf("Rules (%d)\n\n", len(rules))
		for i, rule := range rules {
			printRuleSummary(rule, i, ruleRefs[rule.ID])
		}
		fmt.Println()
		return struct{}{}, nil
	})

	return err
}

func collectPolicyScopedRules(policy *domain.Policy, accounts []policyAgentAccountRef, allRules []domain.Rule) ([]domain.Rule, map[string][]rulePolicyRef) {
	rulesByID := make(map[string]domain.Rule, len(allRules))
	for _, rule := range allRules {
		rulesByID[rule.ID] = rule
	}

	accountRefs := append([]policyAgentAccountRef(nil), accounts...)
	rules := make([]domain.Rule, 0, len(policy.Rules))
	ruleRefs := make(map[string][]rulePolicyRef, len(policy.Rules))

	for _, ruleID := range policy.Rules {
		ruleID = strings.TrimSpace(ruleID)
		if ruleID == "" {
			continue
		}

		rule, ok := rulesByID[ruleID]
		if !ok {
			continue
		}

		rules = append(rules, rule)
		if _, ok := ruleRefs[rule.ID]; ok {
			continue
		}
		ruleRefs[rule.ID] = []rulePolicyRef{{
			PolicyID:   policy.ID,
			PolicyName: policy.Name,
			Accounts:   accountRefs,
		}}
	}

	return rules, ruleRefs
}

func newRuleGetCmd() *cobra.Command {
	var (
		jsonOutput bool
		allRules   bool
		policyID   string
	)

	cmd := &cobra.Command{
		Use:   "get <rule-id>",
		Short: "Show a rule",
		Long: `Show details for a single rule.

By default, this validates that the rule is attached to the current default
agent policy. Use --policy-id to scope the lookup to another provider=nylas
policy, or --all to search any provider=nylas policy.

Examples:
  nylas agent rule get <rule-id>
  nylas agent rule get <rule-id> --policy-id <policy-id>
  nylas agent rule get <rule-id> --all
  nylas agent rule get <rule-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRules && policyID != "" {
				return common.NewUserError("cannot combine --all with --policy-id", "Use either --all or --policy-id")
			}
			return runRuleGet(args[0], jsonOutput, allRules, policyID)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&allRules, "all", false, "Search across all provider=nylas policies")
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy ID to scope the rule lookup to")

	return cmd
}

func newRuleReadCmd() *cobra.Command {
	var (
		jsonOutput bool
		allRules   bool
		policyID   string
	)

	cmd := &cobra.Command{
		Use:   "read <rule-id>",
		Short: "Read a rule",
		Long: `Read details for a single rule.

Examples:
  nylas agent rule read <rule-id>
  nylas agent rule read <rule-id> --policy-id <policy-id>
  nylas agent rule read <rule-id> --all
  nylas agent rule read <rule-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRules && policyID != "" {
				return common.NewUserError("cannot combine --all with --policy-id", "Use either --all or --policy-id")
			}
			return runRuleGet(args[0], jsonOutput, allRules, policyID)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVar(&allRules, "all", false, "Search across all provider=nylas policies")
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy ID to scope the rule lookup to")

	return cmd
}

func runRuleGet(ruleID string, jsonOutput, allRules bool, policyID string) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		scope, err := resolveScopedRule(ctx, client, ruleID, policyID, allRules)
		if err != nil {
			return struct{}{}, err
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(scope.Rule)
		}

		printRuleDetails(*scope.Rule, scope.SelectedRefs)
		return struct{}{}, nil
	})

	return err
}
