package agent

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

const agentDomainDashboardURL = "https://dashboard-v3.nylas.com/"

func newCreateCmd() *cobra.Command {
	var appPassword string
	var name string

	cmd := &cobra.Command{
		Use:   "create <email>",
		Short: "Create a new agent account",
		Long: `Create a new Nylas agent account.

This command always creates a provider=nylas grant. If the nylas connector
does not exist yet, it will be created automatically first. The API
automatically creates a default workspace and policy for the account.

To attach a custom policy after creation:
  nylas workspace update <workspace-id> --policy-id <policy-id>

Examples:
  nylas agent account create me@yourapp.nylas.email
  nylas agent account create support@yourapp.nylas.email --json
  nylas agent account create support@yourapp.nylas.email --name 'Support Bot'
  nylas agent account create debug@yourapp.nylas.email --app-password 'ValidAgentPass123ABC!'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(args[0], name, appPassword, common.IsJSON(cmd))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Optional display name for the agent account (1-256 characters)")
	cmd.Flags().StringVar(&appPassword, "app-password", "", "Optional IMAP/SMTP app password for mail-client access")

	return cmd
}

func runCreate(email, name, appPassword string, jsonOutput bool) error {
	email = normalizeAgentAccountEmail(email)
	if email == "" {
		common.PrintError("Email address cannot be empty")
		return common.NewInputError("email address cannot be empty")
	}
	if strings.Contains(email, " ") {
		common.PrintError("Email address should not contain spaces")
		return common.NewInputError("invalid email address - should not contain spaces")
	}
	name = strings.TrimSpace(name)
	if err := validateAgentName(name); err != nil {
		common.PrintError(err.Error())
		return err
	}
	if err := validateAgentAppPassword(appPassword); err != nil {
		common.PrintError(err.Error())
		return err
	}
	_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
		connector, err := ensureNylasConnector(ctx, client)
		if err != nil {
			return struct{}{}, common.WrapCreateError("nylas connector", err)
		}

		account, err := createAgentAccountWithFallback(ctx, client, email, name, appPassword)
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

func createAgentAccountWithFallback(ctx context.Context, client ports.AgentClient, email, name, appPassword string) (*domain.AgentAccount, error) {
	account, err := client.CreateAgentAccount(ctx, email, name, appPassword, "")
	if err == nil || appPassword == "" || !shouldRetryAgentCreateWithoutPassword(err) {
		return account, wrapAgentAccountCreateError(email, err)
	}

	existingAccount, lookupErr := findExistingAgentAccountByEmail(ctx, client, email)
	if lookupErr == nil && existingAccount != nil {
		// Apply the requested name when given, otherwise preserve the existing
		// account's name — the grant update replaces the full record, so an
		// empty name would clear it.
		effectiveName := name
		if effectiveName == "" {
			effectiveName = existingAccount.Name
		}
		updated, updateErr := client.UpdateAgentAccount(ctx, existingAccount.ID, email, effectiveName, appPassword)
		if updateErr == nil {
			if updated == nil {
				return existingAccount, nil
			}
			return updated, nil
		}

		return nil, fmt.Errorf("failed to set app password on existing agent account %s: %w", email, updateErr)
	}

	account, retryErr := client.CreateAgentAccount(ctx, email, name, "", "")
	if retryErr != nil {
		return nil, fmt.Errorf("failed to create agent account after retrying without app password: %w", wrapAgentAccountCreateError(email, retryErr))
	}

	// Re-send name so the password-setting update preserves it (the grant
	// update replaces the full record).
	updated, updateErr := client.UpdateAgentAccount(ctx, account.ID, email, name, appPassword)
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

func normalizeAgentAccountEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" || strings.Contains(email, "@") {
		return email
	}
	return email + "@nylas.email"
}

func findExistingAgentAccountByEmail(ctx context.Context, client ports.AgentClient, email string) (*domain.AgentAccount, error) {
	accounts, err := client.ListAgentAccounts(ctx)
	if err != nil {
		return nil, err
	}

	if account := findAgentAccountByEmail(accounts, email); account != nil {
		accountCopy := *account
		return &accountCopy, nil
	}

	defaultAccount := getConfiguredDefaultAgentAccount(ctx, client)
	if defaultAccount != nil && strings.EqualFold(defaultAccount.Email, email) {
		return defaultAccount, nil
	}

	return nil, nil
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

type agentAccountDomainErrorKind int

const (
	agentAccountDomainErrorNone agentAccountDomainErrorKind = iota
	agentAccountDomainErrorMissing
	agentAccountDomainErrorLimit
)

func wrapAgentAccountCreateError(email string, err error) error {
	if err == nil {
		return nil
	}

	kind, apiErr := classifyAgentAccountDomainError(err)
	if kind == agentAccountDomainErrorNone {
		return err
	}

	domainName := agentAccountDomainFromEmail(email)
	if domainName == "" {
		domainName = "the requested domain"
	}

	suggestions := []string{
		fmt.Sprintf("Create or register %q as an agent domain in the Nylas Dashboard: %s", domainName, agentDomainDashboardURL),
	}

	message := fmt.Sprintf("Cannot create agent account because domain %q is not registered", domainName)
	if kind == agentAccountDomainErrorLimit {
		message = "Maximum number of agent account domains reached"
		suggestions = append(suggestions, "Remove an unused domain or use an email address on one of your existing agent domains")
	} else {
		suggestions = append(suggestions, "Or use an email address on an agent domain already registered in the Dashboard")
		suggestions = append(suggestions, fmt.Sprintf("After registering the domain, retry: nylas agent account create %s", email))
	}

	requestID := ""
	if apiErr != nil {
		requestID = apiErr.RequestID
	}

	return &common.CLIError{
		Err:         err,
		Message:     message,
		Suggestions: suggestions,
		Code:        common.ErrCodeInvalidInput,
		RequestID:   requestID,
	}
}

func classifyAgentAccountDomainError(err error) (agentAccountDomainErrorKind, *domain.APIError) {
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		return agentAccountDomainErrorNone, nil
	}
	if apiErr.StatusCode != http.StatusBadRequest && apiErr.StatusCode != http.StatusUnprocessableEntity && apiErr.StatusCode != http.StatusNotFound {
		return agentAccountDomainErrorNone, apiErr
	}

	msg := strings.ToLower(strings.TrimSpace(apiErr.Message))
	if !strings.Contains(msg, "domain") {
		return agentAccountDomainErrorNone, apiErr
	}

	limitPhrases := []string{
		"maximum",
		"max",
		"limit",
	}
	for _, phrase := range limitPhrases {
		if strings.Contains(msg, phrase) {
			return agentAccountDomainErrorLimit, apiErr
		}
	}

	missingPhrases := []string{
		"not registered",
		"not found",
		"does not exist",
		"doesn't exist",
		"missing",
		"unknown",
	}
	for _, phrase := range missingPhrases {
		if strings.Contains(msg, phrase) {
			return agentAccountDomainErrorMissing, apiErr
		}
	}

	return agentAccountDomainErrorNone, apiErr
}

func agentAccountDomainFromEmail(email string) string {
	_, domainName, ok := strings.Cut(strings.TrimSpace(email), "@")
	if !ok {
		return ""
	}
	return strings.TrimSpace(domainName)
}

// validateAgentName enforces the grant name constraints (1-256 characters when
// set). An empty name is valid and omits the field from the create payload.
// Length is measured in Unicode characters (runes), not bytes, to match the
// documented "1-256 characters" limit for multi-byte names.
func validateAgentName(name string) error {
	if name == "" {
		return nil
	}
	if utf8.RuneCountInString(name) > 256 {
		return common.NewInputError("name must be 256 characters or fewer")
	}
	return nil
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
