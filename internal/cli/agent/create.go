package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var appPassword string
	var policyID string

	cmd := &cobra.Command{
		Use:   "create <email>",
		Short: "Create a new agent account",
		Long: `Create a new Nylas agent account.

This command always creates a provider=nylas grant. If the nylas connector
does not exist yet, it will be created automatically first.

Examples:
  nylas agent account create me@yourapp.nylas.email
  nylas agent account create support@yourapp.nylas.email --json
  nylas agent account create debug@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'
  nylas agent account create routed@yourapp.nylas.email --policy-id <policy-id>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args[0], appPassword, policyID, common.IsJSON(cmd))
		},
	}

	cmd.Flags().StringVar(&appPassword, "app-password", "", "Optional IMAP/SMTP app password for mail-client access")
	cmd.Flags().StringVar(&policyID, "policy-id", "", "Optional policy ID to attach to the created agent account")

	return cmd
}

func runCreate(email, appPassword, policyID string, jsonOutput bool) error {
	email = strings.TrimSpace(email)
	if email == "" {
		common.PrintError("Email address cannot be empty")
		return common.NewInputError("email address cannot be empty")
	}
	if strings.Contains(email, " ") {
		common.PrintError("Email address should not contain spaces")
		return common.NewInputError("invalid email address - should not contain spaces")
	}
	if err := validateAgentAppPassword(appPassword); err != nil {
		common.PrintError(err.Error())
		return err
	}
	policyID = strings.TrimSpace(policyID)

	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		connector, err := ensureNylasConnector(ctx, client)
		if err != nil {
			return struct{}{}, common.WrapCreateError("nylas connector", err)
		}

		account, err := createAgentAccountWithFallback(ctx, client, email, appPassword, policyID)
		if err != nil {
			return struct{}{}, common.WrapCreateError("agent account", err)
		}

		saveGrantLocally(account.ID, account.Email)

		if jsonOutput {
			return struct{}{}, common.PrintJSON(account)
		}

		common.PrintSuccess("Agent account created successfully!")
		fmt.Println()
		printAgentDetails(*account)
		if connector != nil {
			_, _ = common.Dim.Printf("Connector: %s\n", formatConnectorSummary(*connector))
		}

		fmt.Println()
		_, _ = common.Dim.Println("Next steps:")
		_, _ = common.Dim.Printf("  1. List agent accounts: nylas agent account list\n")
		_, _ = common.Dim.Printf("  2. Check connector status: nylas agent status\n")
		_, _ = common.Dim.Printf("  3. Show this account: nylas agent account get %s\n", account.ID)
		_, _ = common.Dim.Printf("  4. Delete this account: nylas agent account delete %s\n", account.ID)

		return struct{}{}, nil
	})

	return err
}

func createAgentAccountWithFallback(ctx context.Context, client ports.AgentClient, email, appPassword, policyID string) (*domain.AgentAccount, error) {
	account, err := client.CreateAgentAccount(ctx, email, appPassword, policyID)
	if err == nil || appPassword == "" || !shouldRetryAgentCreateWithoutPassword(err) {
		return account, err
	}

	existingAccount, lookupErr := findExistingAgentAccountByEmail(ctx, client, email)
	if lookupErr == nil && existingAccount != nil {
		if err := validateExistingAgentAccountPolicy(existingAccount, policyID); err != nil {
			return nil, err
		}

		updated, updateErr := client.UpdateAgentAccount(ctx, existingAccount.ID, email, appPassword)
		if updateErr == nil {
			return updated, nil
		}

		return nil, fmt.Errorf("failed to set app password on existing agent account %s: %w", email, updateErr)
	}

	account, retryErr := client.CreateAgentAccount(ctx, email, "", policyID)
	if retryErr != nil {
		return nil, fmt.Errorf("failed to create agent account after retrying without app password: %w", retryErr)
	}

	updated, updateErr := client.UpdateAgentAccount(ctx, account.ID, email, appPassword)
	if updateErr == nil {
		return updated, nil
	}

	if lookupErr != nil {
		return nil, fmt.Errorf(
			"created agent account %s but failed to set app password: %w (existing-account lookup before retry also failed: %v)",
			account.ID,
			updateErr,
			lookupErr,
		)
	}

	return nil, fmt.Errorf(
		"created agent account %s but failed to set app password; run 'nylas agent account update %s --app-password <password>' to finish setup: %w",
		account.ID,
		account.ID,
		updateErr,
	)
}

func findExistingAgentAccountByEmail(ctx context.Context, client ports.AgentClient, email string) (*domain.AgentAccount, error) {
	accounts, err := client.ListAgentAccounts(ctx)
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		if strings.EqualFold(account.Email, email) {
			accountCopy := account
			return &accountCopy, nil
		}
	}

	return nil, nil
}

func validateExistingAgentAccountPolicy(account *domain.AgentAccount, requestedPolicyID string) error {
	if account == nil {
		return nil
	}

	requestedPolicyID = strings.TrimSpace(requestedPolicyID)
	if requestedPolicyID == "" {
		return nil
	}

	currentPolicyID := strings.TrimSpace(account.Settings.PolicyID)
	if currentPolicyID == requestedPolicyID {
		return nil
	}
	if currentPolicyID == "" {
		return common.NewUserError(
			"existing agent account is not attached to the requested policy",
			fmt.Sprintf("Agent account %s already exists without a policy; create fallback cannot attach it to policy %s. Attach the policy separately, then run 'nylas agent account update %s --app-password <password>'.", account.Email, requestedPolicyID, account.ID),
		)
	}

	return common.NewUserError(
		"existing agent account is attached to a different policy",
		fmt.Sprintf("Agent account %s already exists on policy %s; create fallback cannot change it to policy %s. Update the policy assignment separately, then run 'nylas agent account update %s --app-password <password>'.", account.Email, currentPolicyID, requestedPolicyID, account.ID),
	)
}

func shouldRetryAgentCreateWithoutPassword(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode != http.StatusBadRequest && apiErr.StatusCode != http.StatusUnprocessableEntity {
		return false
	}

	msg := strings.ToLower(strings.TrimSpace(apiErr.Message))
	if !strings.Contains(msg, "app_password") && !strings.Contains(msg, "app password") {
		return false
	}

	phrases := []string{
		"unknown field",
		"unexpected field",
		"field not allowed",
		"field is not allowed",
		"extra field",
		"extra fields",
		"extra fields not permitted",
		"additional property",
		"additional properties",
		"unrecognized field",
		"unknown parameter",
		"unexpected parameter",
		"unsupported field",
		"unsupported parameter",
	}
	for _, phrase := range phrases {
		if strings.Contains(msg, phrase) {
			return true
		}
	}

	if (strings.Contains(msg, "not permitted") || strings.Contains(msg, "not allowed")) &&
		(strings.Contains(msg, "field") || strings.Contains(msg, "fields") ||
			strings.Contains(msg, "property") || strings.Contains(msg, "properties") ||
			strings.Contains(msg, "parameter") || strings.Contains(msg, "parameters")) {
		return true
	}

	return false
}

func validateAgentAppPassword(appPassword string) error {
	if appPassword == "" {
		return nil
	}
	if len(appPassword) < 18 || len(appPassword) > 40 {
		return common.NewInputError("app password must be between 18 and 40 characters")
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range appPassword {
		if r < 33 || r > 126 {
			return common.NewInputError("app password must use printable ASCII characters only and cannot contain spaces")
		}
		switch {
		case 'A' <= r && r <= 'Z':
			hasUpper = true
		case 'a' <= r && r <= 'z':
			hasLower = true
		case '0' <= r && r <= '9':
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return common.NewInputError("app password must include at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}

func formatConnectorSummary(connector domain.Connector) string {
	if connector.ID == "" {
		return connector.Provider
	}
	return fmt.Sprintf("%s (%s)", connector.Provider, connector.ID)
}
