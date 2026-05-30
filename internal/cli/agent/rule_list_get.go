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
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List rules",
		Long: `List all rules from /v3/rules.

Shows which agent workspace has each rule attached.

Examples:
  nylas agent rule list
  nylas agent rule list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuleList(common.IsJSON(cmd))
		},
	}

	return cmd
}

func runRuleList(jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		rules, err := client.ListRules(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("rules", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(rules)
		}

		if len(rules) == 0 {
			common.PrintEmptyStateWithHint("rules", "Create one with: nylas agent rule create --name \"Rule Name\" --condition ... --action ...")
			return struct{}{}, nil
		}

		workspaceRefs := buildWorkspaceRuleRefs(ctx, client)

		_, _ = common.BoldWhite.Printf("Rules (%d)\n\n", len(rules))
		for i, rule := range rules {
			printRuleSummary(rule, i, workspaceRefs[rule.ID])
		}
		fmt.Println()
		return struct{}{}, nil
	})

	return err
}

func buildWorkspaceRuleRefs(ctx context.Context, client ports.NylasClient) map[string][]ruleWorkspaceRef {
	refs, _ := buildWorkspaceRuleRefsStrict(ctx, client)
	return refs
}

func buildWorkspaceRuleRefsStrict(ctx context.Context, client ports.NylasClient) (map[string][]ruleWorkspaceRef, error) {
	workspaces, err := client.ListWorkspaces(ctx)
	if err != nil {
		return nil, common.WrapListError("workspaces", err)
	}

	accounts, err := client.ListAgentAccounts(ctx)
	if err != nil {
		return nil, common.WrapListError("agent accounts", err)
	}
	grantByWorkspace := make(map[string]domain.AgentAccount)
	for _, account := range accounts {
		if wsID := strings.TrimSpace(account.WorkspaceID); wsID != "" {
			grantByWorkspace[wsID] = account
		}
	}

	refs := make(map[string][]ruleWorkspaceRef)
	for _, workspace := range workspaces {
		wsID := strings.TrimSpace(workspace.ID)
		if wsID == "" {
			continue
		}
		for _, ruleID := range workspace.RulesIDs {
			ruleID = strings.TrimSpace(ruleID)
			if ruleID == "" {
				continue
			}
			ref := ruleWorkspaceRef{
				WorkspaceID:   wsID,
				WorkspaceName: workspace.Name,
			}
			if account, ok := grantByWorkspace[wsID]; ok {
				ref.GrantID = account.ID
				ref.Email = account.Email
			}
			refs[ruleID] = append(refs[ruleID], ref)
		}
	}
	return refs, nil
}

func newRuleGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <rule-id>",
		Short: "Show a rule",
		Long: `Show details for a single rule.

Examples:
  nylas agent rule get <rule-id>
  nylas agent rule get <rule-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuleGet(args[0], common.IsJSON(cmd))
		},
	}

	return cmd
}

func newRuleReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <rule-id>",
		Short: "Read a rule",
		Long: `Read details for a single rule.

Examples:
  nylas agent rule read <rule-id>
  nylas agent rule read <rule-id> --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRuleGet(args[0], common.IsJSON(cmd))
		},
	}

	return cmd
}

func runRuleGet(ruleID string, jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		rule, err := client.GetRule(ctx, ruleID)
		if err != nil {
			return struct{}{}, common.WrapGetError("rule", err)
		}

		if jsonOutput {
			return struct{}{}, common.PrintJSON(rule)
		}

		workspaceRefs := buildWorkspaceRuleRefs(ctx, client)
		printRuleDetails(*rule, workspaceRefs[rule.ID])
		return struct{}{}, nil
	})

	return err
}
