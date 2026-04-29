package agent

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

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
