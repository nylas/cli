package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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

type overviewList struct {
	ID         string `json:"id"`
	Name       string `json:"name,omitempty"`
	Type       string `json:"type,omitempty"`
	ItemsCount int    `json:"items_count"`
	Missing    bool   `json:"missing"`
}

type overviewRule struct {
	ID      string         `json:"id"`
	Name    string         `json:"name,omitempty"`
	Trigger string         `json:"trigger,omitempty"`
	Enabled bool           `json:"enabled"`
	Missing bool           `json:"missing"`
	Lists   []overviewList `json:"lists,omitempty"`
}

type overviewPolicy struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Missing bool   `json:"missing"`
}

type overviewAccount struct {
	ID               string          `json:"id"`
	Email            string          `json:"email"`
	Status           string          `json:"status,omitempty"`
	WorkspaceID      string          `json:"workspace_id,omitempty"`
	WorkspaceName    string          `json:"workspace_name,omitempty"`
	WorkspaceMissing bool            `json:"workspace_missing"`
	AutoGroup        bool            `json:"auto_group"`
	Default          bool            `json:"default"`
	SharedWith       int             `json:"shared_with"`
	Policy           *overviewPolicy `json:"policy,omitempty"`
	Rules            []overviewRule  `json:"rules,omitempty"`
}

type agentOverview struct {
	Accounts       []overviewAccount `json:"accounts"`
	OrphanPolicies []overviewPolicy  `json:"orphan_policies,omitempty"`
	OrphanRules    []overviewRule    `json:"orphan_rules,omitempty"`
	UnusedLists    []overviewList    `json:"unused_lists,omitempty"`
	Totals         map[string]int    `json:"totals"`
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

		overview := buildAgentOverview(accounts, workspaces, policies, rules, lists)

		if jsonOutput {
			return struct{}{}, common.PrintJSON(overview)
		}

		printOverview(overview)
		return struct{}{}, nil
	})
	return err
}

// buildAgentOverview joins the five agent resource collections into one
// account-rooted tree, flagging references to resources that no longer exist.
func buildAgentOverview(
	accounts []domain.AgentAccount,
	workspaces []domain.Workspace,
	policies []domain.Policy,
	rules []domain.Rule,
	lists []domain.AgentList,
) *agentOverview {
	workspaceByID := make(map[string]domain.Workspace, len(workspaces))
	for _, ws := range workspaces {
		workspaceByID[ws.ID] = ws
	}
	policyByID := make(map[string]domain.Policy, len(policies))
	for _, p := range policies {
		policyByID[p.ID] = p
	}
	ruleByID := make(map[string]domain.Rule, len(rules))
	for _, r := range rules {
		ruleByID[r.ID] = r
	}
	listByID := make(map[string]domain.AgentList, len(lists))
	for _, l := range lists {
		listByID[l.ID] = l
	}

	accountsPerWorkspace := make(map[string]int, len(accounts))
	for _, acct := range accounts {
		if wsID := strings.TrimSpace(acct.WorkspaceID); wsID != "" {
			accountsPerWorkspace[wsID]++
		}
	}

	attachedPolicies := make(map[string]bool)
	attachedRules := make(map[string]bool)
	referencedLists := make(map[string]bool)
	for _, ws := range workspaces {
		if ws.PolicyID != "" {
			attachedPolicies[ws.PolicyID] = true
		}
		for _, id := range ws.RulesIDs {
			attachedRules[id] = true
		}
	}
	for _, acct := range accounts {
		if id := strings.TrimSpace(acct.Settings.PolicyID); id != "" {
			attachedPolicies[id] = true
		}
	}
	// A list counts as referenced when ANY existing rule names it, including
	// rules attached to no workspace — deleting such a list would still break
	// the rule the moment it is re-attached.
	for _, rule := range rules {
		for _, id := range ruleListIDs(rule) {
			referencedLists[id] = true
		}
	}

	overview := &agentOverview{
		Accounts: make([]overviewAccount, 0, len(accounts)),
		Totals: map[string]int{
			"accounts":   len(accounts),
			"workspaces": len(workspaces),
			"policies":   len(policies),
			"rules":      len(rules),
			"lists":      len(lists),
		},
	}

	for _, acct := range accounts {
		entry := overviewAccount{
			ID:          acct.ID,
			Email:       acct.Email,
			Status:      acct.GrantStatus,
			WorkspaceID: strings.TrimSpace(acct.WorkspaceID),
		}

		// The effective policy is the workspace's, falling back to the policy
		// attached on the account itself (same resolution as `agent policy`).
		effectivePolicyID := strings.TrimSpace(acct.Settings.PolicyID)

		if entry.WorkspaceID != "" {
			ws, ok := workspaceByID[entry.WorkspaceID]
			if !ok {
				entry.WorkspaceMissing = true
			} else {
				entry.WorkspaceName = ws.Name
				entry.AutoGroup = ws.AutoGroup
				entry.Default = ws.Default
				entry.SharedWith = accountsPerWorkspace[entry.WorkspaceID] - 1
				if wsPolicyID := strings.TrimSpace(ws.PolicyID); wsPolicyID != "" {
					effectivePolicyID = wsPolicyID
				}
				entry.Rules = resolveOverviewRules(ws.RulesIDs, ruleByID, listByID)
			}
		}
		entry.Policy = resolveOverviewPolicy(effectivePolicyID, policyByID)

		overview.Accounts = append(overview.Accounts, entry)
	}

	for _, p := range policies {
		if !attachedPolicies[p.ID] {
			overview.OrphanPolicies = append(overview.OrphanPolicies, overviewPolicy{ID: p.ID, Name: p.Name})
		}
	}
	for _, r := range rules {
		if !attachedRules[r.ID] {
			overview.OrphanRules = append(overview.OrphanRules, overviewRule{
				ID: r.ID, Name: r.Name, Trigger: r.Trigger, Enabled: r.Enabled == nil || *r.Enabled,
			})
		}
	}
	for _, l := range lists {
		if !referencedLists[l.ID] {
			overview.UnusedLists = append(overview.UnusedLists, overviewList{
				ID: l.ID, Name: l.Name, Type: l.Type, ItemsCount: l.ItemsCount,
			})
		}
	}

	return overview
}

func resolveOverviewPolicy(policyID string, policyByID map[string]domain.Policy) *overviewPolicy {
	policyID = strings.TrimSpace(policyID)
	if policyID == "" {
		return nil
	}
	if p, ok := policyByID[policyID]; ok {
		return &overviewPolicy{ID: p.ID, Name: p.Name}
	}
	return &overviewPolicy{ID: policyID, Missing: true}
}

func resolveOverviewRules(ruleIDs []string, ruleByID map[string]domain.Rule, listByID map[string]domain.AgentList) []overviewRule {
	var out []overviewRule
	for _, id := range ruleIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		rule, ok := ruleByID[id]
		if !ok {
			out = append(out, overviewRule{ID: id, Missing: true})
			continue
		}

		entry := overviewRule{
			ID:      rule.ID,
			Name:    rule.Name,
			Trigger: rule.Trigger,
			Enabled: rule.Enabled == nil || *rule.Enabled,
		}
		for _, listID := range ruleListIDs(rule) {
			if list, ok := listByID[listID]; ok {
				entry.Lists = append(entry.Lists, overviewList{
					ID: list.ID, Name: list.Name, Type: list.Type, ItemsCount: list.ItemsCount,
				})
			} else {
				entry.Lists = append(entry.Lists, overviewList{ID: listID, Missing: true})
			}
		}
		out = append(out, entry)
	}
	return out
}

// ruleListIDs collects the list IDs referenced by a rule's in_list conditions.
func ruleListIDs(rule domain.Rule) []string {
	if rule.Match == nil {
		return nil
	}
	var ids []string
	seen := make(map[string]bool)
	for _, condition := range rule.Match.Conditions {
		for _, id := range conditionListIDs(condition) {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// conditionListIDs extracts list IDs from an in_list condition value. The API
// returns the value as an array of IDs, but a bare string is tolerated.
func conditionListIDs(condition domain.RuleCondition) []string {
	if condition.Operator != "in_list" {
		return nil
	}
	switch value := condition.Value.(type) {
	case string:
		if value = strings.TrimSpace(value); value != "" {
			return []string{value}
		}
	case []string:
		var ids []string
		for _, v := range value {
			if v = strings.TrimSpace(v); v != "" {
				ids = append(ids, v)
			}
		}
		return ids
	case []any:
		var ids []string
		for _, v := range value {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				ids = append(ids, strings.TrimSpace(s))
			}
		}
		return ids
	}
	return nil
}

func printOverview(overview *agentOverview) {
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

func printOverviewWorkspace(acct overviewAccount) {
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
		fmt.Println("    ├── (no policy attached)")
	} else if acct.Policy.Missing {
		fmt.Printf("    ├── ⚠ Policy %s no longer exists\n", acct.Policy.ID)
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

func printOverviewLeftovers(overview *agentOverview) {
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
