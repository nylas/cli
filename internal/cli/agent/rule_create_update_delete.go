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

func newRuleCreateCmd() *cobra.Command {
	var (
		data        string
		dataFile    string
		policyID    string
		jsonOutput  bool
		opts        rulePayloadOptions
		enableRule  bool
		disableRule bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a rule",
		Long: `Create a new rule and attach it to an agent policy.

Rules are created through /v3/rules, then attached to the selected policy. If
--policy-id is omitted, the CLI uses the policy attached to the current
default provider=nylas grant.

Examples:
  nylas agent rule create --name "Block Example" --condition from.domain,is,example.com --action block
  nylas agent rule create --name "Archive example.com" --condition from.domain,is,example.com --action archive --action mark_as_read
  nylas agent rule create --data-file rule.json
  nylas agent rule create --data-file rule.json --policy-id <policy-id>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.PrioritySet = cmd.Flags().Changed("priority")
			if err := assignRuleStateFlags(cmd, enableRule, disableRule, &opts); err != nil {
				return err
			}

			loaded, err := loadRulePayloadDetails(data, dataFile, opts, true)
			if err != nil {
				return err
			}
			return runRuleCreate(loaded.Payload, policyID, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Rule name")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Rule description")
	cmd.Flags().IntVar(&opts.Priority, "priority", 0, "Rule priority")
	cmd.Flags().BoolVar(&enableRule, "enabled", false, "Create the rule in an enabled state")
	cmd.Flags().BoolVar(&disableRule, "disabled", false, "Create the rule in a disabled state")
	cmd.Flags().StringVar(&opts.Trigger, "trigger", "", "Rule trigger (inbound or outbound; defaults to inbound when using flags)")
	cmd.Flags().StringVar(&opts.MatchOperator, "match-operator", "", "Match operator for the supplied conditions")
	cmd.Flags().StringArrayVar(&opts.Conditions, "condition", nil, "Match condition as field,operator,value (repeatable). For in_list, pass field,in_list,list-id-1,list-id-2")
	cmd.Flags().StringArrayVar(&opts.Actions, "action", nil, "Rule action as type or type=value (repeatable)")
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON request body")
	cmd.Flags().StringVar(&dataFile, "data-file", "", "Path to a JSON request body file")
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy ID to attach the created rule to")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runRuleCreate(payload map[string]any, policyID string, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		policy, accounts, err := resolveAgentPolicy(ctx, client, policyID)
		if err != nil {
			return struct{}{}, err
		}

		rule, err := client.CreateRule(ctx, payload)
		if err != nil {
			return struct{}{}, common.WrapCreateError("rule", err)
		}

		if err := attachRuleToPolicy(ctx, client, *policy, rule.ID); err != nil {
			cleanupErr := client.DeleteRule(ctx, rule.ID)
			if cleanupErr != nil {
				return struct{}{}, fmt.Errorf("failed to attach rule to policy: %w (cleanup failed: %v)", err, cleanupErr)
			}
			return struct{}{}, fmt.Errorf("failed to attach rule to policy: %w", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(rule)
		}

		printSuccess("Rule created successfully!")
		fmt.Println()
		printRuleDetails(*rule, []rulePolicyRef{{
			PolicyID:   policy.ID,
			PolicyName: policy.Name,
			Accounts:   accounts,
		}})
		return struct{}{}, nil
	})

	return err
}

func newRuleUpdateCmd() *cobra.Command {
	var (
		data        string
		dataFile    string
		policyID    string
		allRules    bool
		jsonOutput  bool
		opts        rulePayloadOptions
		enableRule  bool
		disableRule bool
	)

	cmd := &cobra.Command{
		Use:   "update <rule-id>",
		Short: "Update a rule",
		Long: `Update an existing rule.

By default, this validates that the rule belongs to the current default
provider=nylas policy. Use --policy-id to scope the validation to another
agent policy, or --all to search any agent policy.

Examples:
  nylas agent rule update <rule-id> --name "Updated Rule"
  nylas agent rule update <rule-id> --description "Archive vendor mail" --priority 20
  nylas agent rule update <rule-id> --condition from.domain,is,example.org --action mark_as_starred
  nylas agent rule update <rule-id> --data-file update.json
  nylas agent rule update <rule-id> --all --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if allRules && policyID != "" {
				return common.NewUserError("cannot combine --all with --policy-id", "Use either --all or --policy-id")
			}

			opts.PrioritySet = cmd.Flags().Changed("priority")
			if err := assignRuleStateFlags(cmd, enableRule, disableRule, &opts); err != nil {
				return err
			}

			loaded, err := loadRulePayloadDetails(data, dataFile, opts, false)
			if err != nil {
				return err
			}
			payload := loaded.Payload
			if len(payload) == 0 {
				return common.NewUserError(
					"rule update requires at least one field",
					"Use flags like --name/--condition/--action, or provide JSON with --data/--data-file",
				)
			}
			return runRuleUpdate(args[0], payload, loaded.PureJSON, policyID, allRules, jsonOutput)
		},
	}

	cmd.Flags().StringVar(&opts.Name, "name", "", "Updated rule name")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Updated rule description")
	cmd.Flags().IntVar(&opts.Priority, "priority", 0, "Updated rule priority")
	cmd.Flags().BoolVar(&enableRule, "enabled", false, "Set the rule to enabled")
	cmd.Flags().BoolVar(&disableRule, "disabled", false, "Set the rule to disabled")
	cmd.Flags().StringVar(&opts.Trigger, "trigger", "", "Updated rule trigger (inbound or outbound)")
	cmd.Flags().StringVar(&opts.MatchOperator, "match-operator", "", "Updated match operator")
	cmd.Flags().StringArrayVar(&opts.Conditions, "condition", nil, "Replace conditions with field,operator,value entries (repeatable). For in_list, pass field,in_list,list-id-1,list-id-2")
	cmd.Flags().StringArrayVar(&opts.Actions, "action", nil, "Replace actions with type or type=value entries (repeatable)")
	cmd.Flags().StringVar(&data, "data", "", "Inline JSON request body")
	cmd.Flags().StringVar(&dataFile, "data-file", "", "Path to a JSON request body file")
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy ID to scope the update to")
	cmd.Flags().BoolVar(&allRules, "all", false, "Search across all provider=nylas policies")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func runRuleUpdate(ruleID string, payload map[string]any, pureJSON bool, policyID string, allRules, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		scope, err := resolveScopedRule(ctx, client, ruleID, policyID, allRules)
		if err != nil {
			return struct{}{}, err
		}
		if scope.SharedOutsideAgent {
			return struct{}{}, common.NewUserError(
				"rule is shared with a non-agent policy",
				"Use the generic policy/rule surface to modify shared rules safely",
			)
		}

		if err := finalizeRuleUpdatePayload(payload, scope.Rule, pureJSON); err != nil {
			return struct{}{}, err
		}

		rule, err := client.UpdateRule(ctx, ruleID, payload)
		if err != nil {
			return struct{}{}, common.WrapUpdateError("rule", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(rule)
		}

		common.PrintUpdateSuccess("rule", rule.Name)
		fmt.Println()
		printRuleDetails(*rule, scope.SelectedRefs)
		return struct{}{}, nil
	})

	return err
}

func finalizeRuleUpdatePayload(payload map[string]any, existingRule *domain.Rule, pureJSON bool) error {
	if pureJSON {
		return nil
	}

	preserveRuleMatchOperator(payload, existingRule)
	return validateRulePayload(payload, existingRule)
}

func newRuleDeleteCmd() *cobra.Command {
	var (
		yes      bool
		policyID string
		allRules bool
	)

	cmd := &cobra.Command{
		Use:   "delete <rule-id>",
		Short: "Delete a rule",
		Long: `Delete a rule and detach it from agent policies.

Examples:
  nylas agent rule delete <rule-id> --yes
  nylas agent rule delete <rule-id> --all --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				return common.NewUserError("deletion requires confirmation", "Re-run with --yes to delete the rule")
			}
			if allRules && policyID != "" {
				return common.NewUserError("cannot combine --all with --policy-id", "Use either --all or --policy-id")
			}
			return runRuleDelete(args[0], policyID, allRules)
		},
	}

	common.AddYesFlag(cmd, &yes)
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Policy ID to scope the delete to")
	cmd.Flags().BoolVar(&allRules, "all", false, "Search across all provider=nylas policies")

	return cmd
}

func runRuleDelete(ruleID, policyID string, allRules bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		scope, err := resolveScopedRule(ctx, client, ruleID, policyID, allRules)
		if err != nil {
			return struct{}{}, err
		}
		if scope.SharedOutsideAgent {
			return struct{}{}, common.NewUserError(
				"rule is shared with a non-agent policy",
				"Use the generic policy/rule surface to delete shared rules safely",
			)
		}

		latestPolicies, err := refreshPolicies(ctx, client, scope.AllAgentPolicies)
		if err != nil {
			return struct{}{}, common.WrapGetError("policy", err)
		}

		blockingPolicies, err := policiesLeftEmptyByRuleRemoval(ctx, client, latestPolicies, ruleID)
		if err != nil {
			return struct{}{}, common.WrapGetError("rule", err)
		}
		if len(blockingPolicies) > 0 {
			policyNames := make([]string, 0, len(blockingPolicies))
			for _, policy := range blockingPolicies {
				policyNames = append(policyNames, policy.Name)
			}
			return struct{}{}, common.NewUserError(
				"cannot delete the last rule from an agent policy",
				fmt.Sprintf("Attach another rule to %s before deleting %q", strings.Join(policyNames, ", "), scope.Rule.Name),
			)
		}

		rollback, err := detachRuleFromPolicies(ctx, client, latestPolicies, ruleID)
		if err != nil {
			return struct{}{}, fmt.Errorf("failed to detach rule from agent policies: %w", err)
		}

		if err := client.DeleteRule(ctx, ruleID); err != nil {
			if rollbackErr := rollback(ctx); rollbackErr != nil {
				return struct{}{}, fmt.Errorf("failed to delete rule: %w (rollback failed: %v)", err, rollbackErr)
			}
			return struct{}{}, common.WrapDeleteError("rule", err)
		}

		common.PrintSuccess("Rule deleted")
		return struct{}{}, nil
	})

	return err
}

func assignRuleStateFlags(cmd *cobra.Command, enableRule, disableRule bool, opts *rulePayloadOptions) error {
	enabledChanged := cmd.Flags().Changed("enabled")
	disabledChanged := cmd.Flags().Changed("disabled")
	if enabledChanged && !enableRule {
		return common.NewUserError("invalid --enabled value", "Use --enabled or omit the flag")
	}
	if disabledChanged && !disableRule {
		return common.NewUserError("invalid --disabled value", "Use --disabled or omit the flag")
	}

	opts.EnabledSet = enableRule
	opts.DisabledSet = disableRule
	return nil
}
