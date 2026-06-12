package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/nylas/cli/internal/agentgraph"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newOverviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "overview",
		Aliases: []string{"tree"},
		Short:   "Show an overview of all agent resources",
		Long: `Show how agent accounts, workspaces, policies, rules, and lists fit together.

Renders one tree per agent account (workspace, attached policy, attached rules,
and the lists those rules reference), flags dangling references to deleted
resources, and reports unattached policies/rules and unused lists.

API reference: https://developer.nylas.com/docs/v3/agent-accounts/

Examples:
  nylas agent overview
  nylas agent overview --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOverview(common.IsJSON(cmd))
		},
	}
}

func runOverview(jsonOutput bool) error {
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		accounts, err := client.ListAgentAccounts(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("agent accounts", err)
		}
		workspaces, err := client.ListWorkspaces(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("workspaces", err)
		}
		policies, err := client.ListPolicies(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("policies", err)
		}
		rules, err := client.ListRules(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("rules", err)
		}
		lists, err := client.ListLists(ctx)
		if err != nil {
			return struct{}{}, common.WrapListError("lists", err)
		}

		overview := agentgraph.Build(accounts, workspaces, policies, rules, lists)

		if jsonOutput {
			return struct{}{}, common.PrintJSON(overview)
		}

		printOverview(overview)
		return struct{}{}, nil
	})
	return err
}

func printOverview(overview *agentgraph.Overview) {
	if len(overview.Accounts) == 0 {
		common.PrintInfo("No agent accounts found. Create one with: nylas agent account create <email>")
		return
	}

	_, _ = common.BoldWhite.Printf("Agent Overview (%d accounts)\n\n", len(overview.Accounts))

	for _, acct := range overview.Accounts {
		_, _ = common.Cyan.Printf("%s", acct.Email)
		if acct.Status != "" {
			fmt.Printf("  %s", acct.Status)
		}
		fmt.Println()

		switch {
		case acct.WorkspaceID == "":
			fmt.Println("└── (no workspace attached)")
		case acct.WorkspaceMissing:
			fmt.Printf("└── ⚠ Workspace %s no longer exists\n", acct.WorkspaceID)
		default:
			printOverviewWorkspace(acct)
		}
		fmt.Println()
	}

	printOverviewLeftovers(overview)
	printOverviewTotals(overview.Totals)
}

func printOverviewWorkspace(acct agentgraph.Account) {
	var traits []string
	if acct.Default {
		traits = append(traits, "default")
	}
	if acct.AutoGroup {
		traits = append(traits, "auto-group")
	}
	if acct.SharedWith > 0 {
		traits = append(traits, fmt.Sprintf("shared with %d other account(s)", acct.SharedWith))
	}
	suffix := ""
	if len(traits) > 0 {
		suffix = " (" + strings.Join(traits, ", ") + ")"
	}

	name := acct.WorkspaceName
	if name == "" {
		name = acct.WorkspaceID
	}
	fmt.Printf("└── Workspace: %s%s\n", name, suffix)

	if acct.Policy == nil {
		fmt.Println("    ├── (no policy attached — plan maximums apply)")
	} else if acct.Policy.Missing {
		fmt.Printf("    ├── ⚠ Policy %s no longer exists — plan maximums apply\n", acct.Policy.ID)
	} else {
		fmt.Printf("    ├── Policy: %s\n", acct.Policy.Name)
	}

	if len(acct.Rules) == 0 {
		fmt.Println("    └── Rules: (none attached)")
		return
	}

	fmt.Printf("    └── Rules (%d)\n", len(acct.Rules))
	for i, rule := range acct.Rules {
		branch := "├──"
		childIndent := "        │   "
		if i == len(acct.Rules)-1 {
			branch = "└──"
			childIndent = "            "
		}

		if rule.Missing {
			fmt.Printf("        %s ⚠ Rule %s no longer exists (detach it from the workspace)\n", branch, rule.ID)
			continue
		}

		state := ""
		if !rule.Enabled {
			state = " [disabled]"
		}
		fmt.Printf("        %s %s (%s)%s\n", branch, rule.Name, rule.Trigger, state)
		for _, list := range rule.Lists {
			if list.Missing {
				fmt.Printf("%s└── ⚠ List %s no longer exists\n", childIndent, list.ID)
			} else {
				fmt.Printf("%s└── List: %s (%s, %d items)\n", childIndent, list.Name, list.Type, list.ItemsCount)
			}
		}
	}
}

func printOverviewLeftovers(overview *agentgraph.Overview) {
	if len(overview.OrphanPolicies) == 0 && len(overview.OrphanRules) == 0 && len(overview.UnusedLists) == 0 {
		return
	}

	_, _ = common.BoldWhite.Println("Unattached resources")
	for _, p := range overview.OrphanPolicies {
		fmt.Printf("  Policy: %s  %s (attached to no workspace)\n", p.Name, p.ID)
	}
	for _, r := range overview.OrphanRules {
		fmt.Printf("  Rule:   %s  %s (attached to no workspace)\n", r.Name, r.ID)
	}
	for _, l := range overview.UnusedLists {
		fmt.Printf("  List:   %s  %s (referenced by no rule)\n", l.Name, l.ID)
	}
	fmt.Println()
}

func printOverviewTotals(totals map[string]int) {
	keys := make([]string, 0, len(totals))
	for k := range totals {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %d", k, totals[k]))
	}
	_, _ = common.Dim.Printf("Totals — %s\n", strings.Join(parts, ", "))
}
