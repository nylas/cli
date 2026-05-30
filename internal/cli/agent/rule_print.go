package agent

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

type ruleWorkspaceRef struct {
	WorkspaceID   string `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name,omitempty"`
	GrantID       string `json:"grant_id,omitempty"`
	Email         string `json:"email,omitempty"`
}

func printRuleSummary(rule domain.Rule, index int, refs []ruleWorkspaceRef) {
	fmt.Printf("%d. %-32s %s\n", index+1, common.Cyan.Sprint(rule.Name), common.Dim.Sprint(rule.ID))
	if !rule.UpdatedAt.IsZero() {
		_, _ = common.Dim.Printf("   Updated: %s\n", common.FormatTimeAgo(rule.UpdatedAt.Time))
	}
	for _, ref := range refs {
		if ref.Email != "" {
			_, _ = common.Dim.Printf("   Workspace: %s  Agent: %s\n", ref.WorkspaceID, ref.Email)
		} else {
			_, _ = common.Dim.Printf("   Workspace: %s\n", ref.WorkspaceID)
		}
	}
}

func printRuleDetails(rule domain.Rule, refs []ruleWorkspaceRef) {
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

	printRuleWorkspacesSection(refs)
	printRuleMatchSection(rule.Match)
	printRuleActionsSection(rule.Actions)
	fmt.Println()
}

func printRuleWorkspacesSection(refs []ruleWorkspaceRef) {
	printPolicySectionHeader("Workspaces")
	if len(refs) == 0 {
		fmt.Println("  none")
		return
	}

	for _, ref := range refs {
		printPolicyField("Workspace", ref.WorkspaceID)
		if ref.Email != "" {
			printPolicyField("Agent", fmt.Sprintf("%s (%s)", ref.Email, ref.GrantID))
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
		printPolicyField("Conditions", "none")
		return
	}

	printPolicyField("Conditions", fmt.Sprintf("%d", len(match.Conditions)))
	for _, cond := range match.Conditions {
		fmt.Printf("    - %s %s %v\n", cond.Field, cond.Operator, cond.Value)
	}
}

func printRuleActionsSection(actions []domain.RuleAction) {
	printPolicySectionHeader("Actions")
	if len(actions) == 0 {
		fmt.Println("  none")
		return
	}

	for _, action := range actions {
		if action.Value != nil {
			fmt.Printf("  - %s = %v\n", action.Type, action.Value)
		} else {
			fmt.Printf("  - %s\n", action.Type)
		}
	}
}
