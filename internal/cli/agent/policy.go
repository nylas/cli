package agent

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

type policyAgentAccountRef struct {
	GrantID string `json:"grant_id"`
	Email   string `json:"email"`
}

func newPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage agent policies",
		Long: `Manage policies used by agent accounts.

Policies are backed by the /v3/policies API and can be attached to agent
accounts via policy_id in grant settings.

Examples:
  nylas agent policy list
  nylas agent policy get <policy-id>
  nylas agent policy read <policy-id>
  nylas agent policy create --name "Strict Policy"
  nylas agent policy update <policy-id> --name "Updated Policy"
  nylas agent policy delete <policy-id> --yes`,
	}

	cmd.AddCommand(newPolicyListCmd())
	cmd.AddCommand(newPolicyGetCmd())
	cmd.AddCommand(newPolicyReadCmd())
	cmd.AddCommand(newPolicyCreateCmd())
	cmd.AddCommand(newPolicyUpdateCmd())
	cmd.AddCommand(newPolicyDeleteCmd())

	return cmd
}

func buildPolicyAccountRefs(accounts []domain.AgentAccount) map[string][]policyAgentAccountRef {
	refsByPolicyID := make(map[string][]policyAgentAccountRef, len(accounts))
	for _, account := range accounts {
		policyID := strings.TrimSpace(account.Settings.PolicyID)
		if policyID == "" {
			continue
		}
		refsByPolicyID[policyID] = append(refsByPolicyID[policyID], policyAgentAccountRef{
			GrantID: account.ID,
			Email:   account.Email,
		})
	}

	for policyID, refs := range refsByPolicyID {
		slices.SortFunc(refs, func(a, b policyAgentAccountRef) int {
			if c := cmp.Compare(strings.ToLower(a.Email), strings.ToLower(b.Email)); c != 0 {
				return c
			}
			return cmp.Compare(a.GrantID, b.GrantID)
		})
		refsByPolicyID[policyID] = refs
	}

	return refsByPolicyID
}

func filterPoliciesWithAgentAccounts(policies []domain.Policy, refsByPolicyID map[string][]policyAgentAccountRef) []domain.Policy {
	filtered := make([]domain.Policy, 0, len(policies))
	for _, policy := range policies {
		if len(refsByPolicyID[policy.ID]) == 0 {
			continue
		}
		filtered = append(filtered, policy)
	}
	return filtered
}

func formatPolicyAgentAccounts(accounts []policyAgentAccountRef) string {
	if len(accounts) == 0 {
		return ""
	}

	formatted := make([]string, 0, len(accounts))
	for _, account := range accounts {
		switch {
		case account.Email != "" && account.GrantID != "":
			formatted = append(formatted, fmt.Sprintf("%s (%s)", account.Email, account.GrantID))
		case account.Email != "":
			formatted = append(formatted, account.Email)
		case account.GrantID != "":
			formatted = append(formatted, account.GrantID)
		}
	}

	return strings.Join(formatted, ", ")
}

func printPolicySummary(policy domain.Policy, index int, accounts []policyAgentAccountRef) {
	fmt.Printf("%d. %-32s %s\n", index+1, common.Cyan.Sprint(policy.Name), common.Dim.Sprint(policy.ID))
	if !policy.UpdatedAt.IsZero() {
		_, _ = common.Dim.Printf("   Updated: %s\n", common.FormatTimeAgo(policy.UpdatedAt.Time))
	}
	for _, account := range accounts {
		_, _ = common.Dim.Printf("   Agent: %s (%s)\n", account.Email, account.GrantID)
	}
}

func printPolicyDetails(policy domain.Policy) {
	fmt.Printf("Policy:       %s\n", policy.Name)
	fmt.Printf("ID:           %s\n", policy.ID)
	if policy.ApplicationID != "" {
		fmt.Printf("Application:  %s\n", policy.ApplicationID)
	}
	if policy.OrganizationID != "" {
		fmt.Printf("Organization: %s\n", policy.OrganizationID)
	}
	fmt.Printf("Rules:        %d\n", len(policy.Rules))
	if !policy.CreatedAt.IsZero() {
		fmt.Printf("Created:      %s (%s)\n", policy.CreatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(policy.CreatedAt.Time))
	}
	if !policy.UpdatedAt.IsZero() {
		fmt.Printf("Updated:      %s (%s)\n", policy.UpdatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(policy.UpdatedAt.Time))
	}

	printPolicyStringListSection("Rules", policy.Rules)
	printPolicyLimitsSection(policy.Limits)
	printPolicyOptionsSection(policy.Options)
	printPolicySpamDetectionSection(policy.SpamDetection)
	fmt.Println()
}

func printPolicySectionHeader(title string) {
	fmt.Printf("\n%s:\n", title)
}

func printPolicyField(label, value string) {
	fmt.Printf("  %-25s %s\n", label+":", value)
}

func printPolicyStringListSection(title string, values []string) {
	printPolicySectionHeader(title)
	if len(values) == 0 {
		fmt.Println("  none")
		return
	}

	for i, value := range values {
		fmt.Printf("  %d. %s\n", i+1, value)
	}
}

func printPolicyValueList(label string, values []string) {
	if len(values) == 0 {
		printPolicyField(label, "none")
		return
	}

	printPolicyField(label, fmt.Sprintf("%d", len(values)))
	for _, value := range values {
		fmt.Printf("    - %s\n", value)
	}
}

func printPolicyLimitsSection(limits *domain.PolicyLimits) {
	printPolicySectionHeader("Limits")
	if limits == nil {
		fmt.Println("  none")
		return
	}

	printed := false
	if limits.LimitAttachmentSizeInBytes != nil {
		printPolicyField("Attachment size", formatPolicyBytes(*limits.LimitAttachmentSizeInBytes))
		printed = true
	}
	if limits.LimitAttachmentCount != nil {
		printPolicyField("Attachment count", fmt.Sprintf("%d", *limits.LimitAttachmentCount))
		printed = true
	}
	if limits.LimitAttachmentAllowedTypes != nil {
		printPolicyValueList("Allowed types", *limits.LimitAttachmentAllowedTypes)
		printed = true
	}
	if limits.LimitSizeTotalMimeInBytes != nil {
		printPolicyField("MIME size total", formatPolicyBytes(*limits.LimitSizeTotalMimeInBytes))
		printed = true
	}
	if limits.LimitStorageTotalInBytes != nil {
		printPolicyField("Storage total", formatPolicyBytes(*limits.LimitStorageTotalInBytes))
		printed = true
	}
	if limits.LimitCountDailyMessagePerGrant != nil {
		printPolicyField("Daily messages/grant", fmt.Sprintf("%d", *limits.LimitCountDailyMessagePerGrant))
		printed = true
	}
	if limits.LimitInboxRetentionPeriodInDays != nil {
		printPolicyField("Inbox retention", formatPolicyDays(*limits.LimitInboxRetentionPeriodInDays))
		printed = true
	}
	if limits.LimitSpamRetentionPeriodInDays != nil {
		printPolicyField("Spam retention", formatPolicyDays(*limits.LimitSpamRetentionPeriodInDays))
		printed = true
	}

	if !printed {
		fmt.Println("  none")
	}
}

func printPolicyOptionsSection(options *domain.PolicyOptions) {
	printPolicySectionHeader("Options")
	if options == nil {
		fmt.Println("  none")
		return
	}

	printed := false
	if options.AdditionalFolders != nil {
		printPolicyValueList("Additional folders", *options.AdditionalFolders)
		printed = true
	}
	if options.UseCidrAliasing != nil {
		printPolicyField("CIDR aliasing", fmt.Sprintf("%t", *options.UseCidrAliasing))
		printed = true
	}

	if !printed {
		fmt.Println("  none")
	}
}

func printPolicySpamDetectionSection(spamDetection *domain.PolicySpamDetection) {
	printPolicySectionHeader("Spam detection")
	if spamDetection == nil {
		fmt.Println("  none")
		return
	}

	printed := false
	if spamDetection.UseListDNSBL != nil {
		printPolicyField("Use DNSBL", fmt.Sprintf("%t", *spamDetection.UseListDNSBL))
		printed = true
	}
	if spamDetection.UseHeaderAnomalyDetection != nil {
		printPolicyField("Header anomaly detection", fmt.Sprintf("%t", *spamDetection.UseHeaderAnomalyDetection))
		printed = true
	}
	if spamDetection.SpamSensitivity != nil {
		printPolicyField("Spam sensitivity", fmt.Sprintf("%.2f", *spamDetection.SpamSensitivity))
		printed = true
	}

	if !printed {
		fmt.Println("  none")
	}
}

func formatPolicyBytes(size int64) string {
	return fmt.Sprintf("%s (%d bytes)", common.FormatSize(size), size)
}

func formatPolicyDays(days int) string {
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func loadPolicyPayload(data, dataFile, name string, requireName bool) (map[string]any, error) {
	payload, err := common.ReadJSONStringMap(data, dataFile)
	if err != nil {
		return nil, err
	}

	if name != "" {
		payload["name"] = name
	}

	if requireName {
		rawName, _ := payload["name"].(string)
		if rawName == "" {
			return nil, common.NewUserError("policy name is required", "Use --name or include a non-empty name in --data/--data-file")
		}
	}

	return payload, nil
}
