package inbound

import (
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// getInboxID gets the inbox ID from args or environment variable.
func getInboxID(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	// Try to get from environment variable
	if envID := os.Getenv("NYLAS_INBOUND_GRANT_ID"); envID != "" {
		return envID, nil
	}

	return "", common.NewUserError("inbox ID required", "Provide as argument or set NYLAS_INBOUND_GRANT_ID environment variable")
}

// printError prints an error message in red.
// Delegates to common.PrintError for consistent error formatting.
func printError(format string, args ...any) {
	common.PrintError(format, args...)
}

// printSuccess prints a success message in green.
// Delegates to common.PrintSuccess for consistent success formatting.
func printSuccess(format string, args ...any) {
	common.PrintSuccess(format, args...)
}

// printInboxSummary prints a single-line inbox summary.
func printInboxSummary(inbox domain.InboundInbox, index int) {
	status := common.Green.Sprint("active")
	if inbox.GrantStatus != "valid" {
		status = common.Yellow.Sprint(inbox.GrantStatus)
	}

	createdStr := common.FormatTimeAgo(inbox.CreatedAt.Time)

	fmt.Printf("%d. %-40s %s  %s\n",
		index+1,
		common.Cyan.Sprint(inbox.Email),
		common.Dim.Sprint(createdStr),
		status,
	)
	_, _ = common.Dim.Printf("   ID: %s\n", inbox.ID)
}

// printInboxDetails prints detailed inbox information.
func printInboxDetails(inbox domain.InboundInbox) {
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Printf("Inbox: %s\n", inbox.Email)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("ID:          %s\n", inbox.ID)
	fmt.Printf("Email:       %s\n", inbox.Email)
	fmt.Printf("Status:      %s\n", formatStatus(inbox.GrantStatus))
	fmt.Printf("Created:     %s (%s)\n", inbox.CreatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(inbox.CreatedAt.Time))
	if !inbox.UpdatedAt.IsZero() {
		fmt.Printf("Updated:     %s (%s)\n", inbox.UpdatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(inbox.UpdatedAt.Time))
	}
	fmt.Println()
}

// formatStatus formats the grant status with color.
func formatStatus(status string) string {
	switch status {
	case "valid":
		return common.Green.Sprint("active")
	case "invalid":
		return common.Red.Sprint("invalid")
	default:
		return common.Yellow.Sprint(status)
	}
}

// saveGrantLocally saves the inbound inbox grant to the local keyring store.
func saveGrantLocally(grantID, email string) {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return
	}
	grantStore := keyring.NewGrantStore(secretStore)
	_ = grantStore.SaveGrant(domain.GrantInfo{
		ID:       grantID,
		Email:    email,
		Provider: domain.ProviderInbox,
	})
}

// removeGrantLocally removes the inbound inbox grant from the local keyring store.
func removeGrantLocally(grantID string) {
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return
	}
	grantStore := keyring.NewGrantStore(secretStore)
	_ = grantStore.DeleteGrant(grantID)
}

// printInboundMessageSummary prints an inbound message summary.
func printInboundMessageSummary(msg domain.InboundMessage, _ int) {
	status := " "
	if msg.Unread {
		status = common.Cyan.Sprint("●")
	}

	star := " "
	if msg.Starred {
		star = common.Yellow.Sprint("★")
	}

	from := common.FormatParticipants(msg.From)
	if len(from) > 20 {
		from = from[:17] + "..."
	}

	subject := msg.Subject
	if len(subject) > 40 {
		subject = subject[:37] + "..."
	}

	dateStr := common.FormatTimeAgo(msg.Date)
	if len(dateStr) > 12 {
		dateStr = msg.Date.Format("Jan 2")
	}

	fmt.Printf("%s %s %-20s %-40s %s\n", status, star, from, subject, common.Dim.Sprint(dateStr))
	_, _ = common.Dim.Printf("      ID: %s\n", msg.ID)
}
