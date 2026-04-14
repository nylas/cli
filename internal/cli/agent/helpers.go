package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func printError(format string, args ...any) {
	common.PrintError(format, args...)
}

func printSuccess(format string, args ...any) {
	common.PrintSuccess(format, args...)
}

func formatStatus(status string) string {
	return common.FormatGrantStatus(status)
}

func printAgentSummary(account domain.AgentAccount, index int) {
	createdStr := common.FormatTimeAgo(account.CreatedAt.Time)
	fmt.Printf("%d. %-40s %s  %s\n",
		index+1,
		common.Cyan.Sprint(account.Email),
		common.Dim.Sprint(createdStr),
		formatStatus(account.GrantStatus),
	)
	_, _ = common.Dim.Printf("   ID: %s\n", account.ID)
}

func printAgentDetails(account domain.AgentAccount) {
	fmt.Println(strings.Repeat("─", 60))
	_, _ = common.BoldWhite.Printf("Agent Account: %s\n", account.Email)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("ID:           %s\n", account.ID)
	fmt.Printf("Provider:     %s\n", account.Provider.DisplayName())
	fmt.Printf("Email:        %s\n", account.Email)
	fmt.Printf("Status:       %s\n", formatStatus(account.GrantStatus))
	if account.CredentialID != "" {
		fmt.Printf("Credential:   %s\n", account.CredentialID)
	}
	if account.Settings.PolicyID != "" {
		fmt.Printf("Policy ID:    %s\n", account.Settings.PolicyID)
	}
	fmt.Printf("Blocked:      %t\n", account.Blocked)
	if !account.CreatedAt.IsZero() {
		fmt.Printf("Created:      %s (%s)\n", account.CreatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(account.CreatedAt.Time))
	}
	if !account.UpdatedAt.IsZero() {
		fmt.Printf("Updated:      %s (%s)\n", account.UpdatedAt.Format(common.DisplayDateTime), common.FormatTimeAgo(account.UpdatedAt.Time))
	}
	fmt.Println()
}

func saveGrantLocally(grantID, email string) {
	common.SaveGrantLocally(grantID, email, domain.ProviderNylas)
}

func removeGrantLocally(grantID string) {
	common.RemoveGrantLocally(grantID)
}

func ensureNylasConnector(ctx context.Context, client ports.NylasClient) (*domain.Connector, error) {
	connectors, err := client.ListConnectors(ctx)
	if err != nil {
		return nil, err
	}

	for _, connector := range connectors {
		if connector.Provider == string(domain.ProviderNylas) {
			return &connector, nil
		}
	}

	connector, err := client.CreateConnector(ctx, &domain.CreateConnectorRequest{
		Name:     "nylas",
		Provider: string(domain.ProviderNylas),
	})
	if err == nil {
		return connector, nil
	}

	// Retry discovery once in case another process created it concurrently.
	connectors, listErr := client.ListConnectors(ctx)
	if listErr == nil {
		for _, connector := range connectors {
			if connector.Provider == string(domain.ProviderNylas) {
				return &connector, nil
			}
		}
	}

	return nil, err
}

func resolveAgentID(ctx context.Context, client ports.NylasClient, identifier string) (string, error) {
	if !strings.Contains(identifier, "@") {
		return identifier, nil
	}

	accounts, err := client.ListAgentAccounts(ctx)
	if err != nil {
		return "", err
	}
	for _, account := range accounts {
		if strings.EqualFold(account.Email, identifier) {
			return account.ID, nil
		}
	}

	return "", common.NewUserError("agent account not found", fmt.Sprintf("No agent account found for email %s", identifier))
}

func getAgentIdentifier(args []string) (string, error) {
	if len(args) > 0 {
		return strings.TrimSpace(args[0]), nil
	}

	if envID := os.Getenv("NYLAS_AGENT_GRANT_ID"); envID != "" {
		return envID, nil
	}

	return "", common.NewUserError("agent ID required", "Provide an agent ID/email or set NYLAS_AGENT_GRANT_ID")
}
