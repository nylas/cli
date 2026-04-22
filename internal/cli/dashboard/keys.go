package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func newAPIKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apikeys",
		Aliases: []string{"keys"},
		Short:   "Manage API keys for an application",
	}

	cmd.AddCommand(newAPIKeysListCmd())
	cmd.AddCommand(newAPIKeysCreateCmd())

	return cmd
}

// apiKeyRow is a flat struct for table output.
type apiKeyRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

var apiKeyColumns = []ports.Column{
	{Header: "ID", Field: "ID"},
	{Header: "NAME", Field: "Name"},
	{Header: "STATUS", Field: "Status"},
	{Header: "EXPIRES", Field: "ExpiresAt"},
	{Header: "CREATED", Field: "CreatedAt"},
}

func newAPIKeysListCmd() *cobra.Command {
	var (
		appID  string
		region string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List API keys for an application",
		Long: `List API keys for an application. Uses the active app if --app is not specified.
Set an active app with: nylas dashboard apps use <app-id> --region <region>`,
		Example: `  # Using active app
  nylas dashboard apps apikeys list

  # Explicit app
  nylas dashboard apps apikeys list --app <app-id> --region us`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedApp, resolvedRegion, err := getActiveApp(appID, region)
			if err != nil {
				return err
			}
			appID = resolvedApp
			region = resolvedRegion

			appSvc, err := createAppService()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			var keys []domain.GatewayAPIKey
			err = common.RunWithSpinner("Fetching API keys...", func() error {
				keys, err = appSvc.ListAPIKeys(ctx, appID, region)
				return err
			})
			if err != nil {
				return wrapDashboardError(err)
			}

			if len(keys) == 0 {
				createCmd := "nylas dashboard apps " +
					"apikeys create --app " + appID +
					" --region " + region
				fmt.Println("No API keys found.")
				fmt.Printf("\nCreate one with:\n  %s\n", createCmd)
				return nil
			}

			rows := toAPIKeyRows(keys)
			return common.WriteListWithColumns(cmd, rows, apiKeyColumns)
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "Application ID (overrides active app)")
	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (overrides active app)")

	return cmd
}

func newAPIKeysCreateCmd() *cobra.Command {
	var (
		appID     string
		region    string
		name      string
		expiresIn int
		delivery  string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an API key for an application",
		Long: `Create a new API key for an application. Uses the active app if --app is not specified.

After creation, you choose what to do with the key:
  1. Activate it — store in CLI keyring as the active API key (recommended)
  2. Copy to clipboard — for use in other tools
  3. Save to file — for handoff or scripts

Set an active app with: nylas dashboard apps use <app-id> --region <region>`,
		Example: `  # Using active app (simplest)
  nylas dashboard apps apikeys create

  # With a custom name
  nylas dashboard apps apikeys create --name "My key"

  # Explicit app
  nylas dashboard apps apikeys create --app <app-id> --region us

  # Non-interactive delivery
  nylas dashboard apps apikeys create --delivery activate

  # Create with custom expiration (days)
  nylas dashboard apps apikeys create --expires 30`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedApp, resolvedRegion, err := getActiveApp(appID, region)
			if err != nil {
				return err
			}
			appID = resolvedApp
			region = resolvedRegion

			if name == "" {
				name = "CLI-" + time.Now().Format("20060102-150405")
			}
			if err := validateAPIKeyDelivery(delivery); err != nil {
				return err
			}
			if !isInteractive() && delivery == "" {
				return dashboardError(
					"API key delivery requires an explicit choice in non-interactive runs",
					"Pass --delivery activate, --delivery clipboard, or --delivery file",
				)
			}

			appSvc, err := createAppService()
			if err != nil {
				return wrapDashboardError(err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			var key *domain.GatewayCreatedAPIKey
			err = common.RunWithSpinner("Creating API key...", func() error {
				key, err = appSvc.CreateAPIKey(ctx, appID, region, name, expiresIn)
				return err
			})
			if err != nil {
				return wrapDashboardError(err)
			}

			_, _ = common.Green.Println("✓ API key created")
			fmt.Printf("  ID:   %s\n", key.ID)
			fmt.Printf("  Name: %s\n", key.Name)

			return handleAPIKeyDelivery(key.APIKey, appID, region, delivery)
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "Application ID (overrides active app)")
	cmd.Flags().StringVarP(&region, "region", "r", "", "Region (overrides active app)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "API key name (default: CLI-<timestamp>)")
	cmd.Flags().IntVar(&expiresIn, "expires", 0, "Expiration in days (default: no expiration)")
	cmd.Flags().StringVar(&delivery, "delivery", "", "API key delivery method (activate, clipboard, or file)")

	return cmd
}

// handleAPIKeyDelivery prompts the user to choose how to handle the newly created key.
// The API key is never printed to stdout to prevent leaking it in terminal history or logs.
func handleAPIKeyDelivery(apiKey, appID, region, delivery string) error {
	choice := delivery
	if choice == "" {
		if !isInteractive() {
			return dashboardError(
				"API key delivery requires an explicit choice in non-interactive runs",
				"Pass --delivery activate, --delivery clipboard, or --delivery file",
			)
		}
		selected, err := common.Select("What would you like to do with this API key?", []common.SelectOption[string]{
			{Label: "Activate for this CLI (recommended)", Value: "activate"},
			{Label: "Copy to clipboard", Value: "clipboard"},
			{Label: "Save to file", Value: "file"},
		})
		if err != nil {
			return wrapDashboardError(err)
		}
		choice = selected
	}

	switch choice {
	case "activate":
		if err := activateAPIKey(apiKey, appID, region, ""); err != nil {
			_, _ = common.Yellow.Printf("  Could not activate: %v\n", err)
			return nil
		}
		_, _ = common.Green.Println("✓ API key activated — CLI is ready to use")
		_, _ = common.Dim.Println("  Try: nylas auth status")

	case "clipboard":
		if err := common.CopyToClipboard(apiKey); err != nil {
			_, _ = common.Yellow.Printf("  Clipboard unavailable: %v\n", err)
			_, _ = common.Dim.Println("  Falling back to file save")
			return saveSecretToFile(apiKey, "nylas-api-key.txt", "API key")
		}
		_, _ = common.Green.Println("✓ API key copied to clipboard")

	case "file":
		return saveSecretToFile(apiKey, "nylas-api-key.txt", "API key")

	default:
		return dashboardError(
			"invalid API key delivery method",
			"Use --delivery activate, --delivery clipboard, or --delivery file",
		)
	}

	return nil
}

// handleSecretDelivery prompts the user to choose how to receive a secret.
// Secrets are never printed to stdout to prevent leaking in terminal history or logs.
func handleSecretDelivery(secret, label, delivery string) error {
	choice := delivery
	if choice == "" {
		if !isInteractive() {
			return dashboardError(
				"client secret delivery requires an explicit choice in non-interactive runs",
				"Pass --secret-delivery clipboard or --secret-delivery file",
			)
		}
		selected, err := common.Select(fmt.Sprintf("How would you like to receive the %s?", label), []common.SelectOption[string]{
			{Label: "Copy to clipboard (recommended)", Value: "clipboard"},
			{Label: "Save to file", Value: "file"},
		})
		if err != nil {
			return wrapDashboardError(err)
		}
		choice = selected
	}

	switch choice {
	case "clipboard":
		if err := common.CopyToClipboard(secret); err != nil {
			_, _ = common.Yellow.Printf("  Clipboard unavailable: %v\n", err)
			_, _ = common.Dim.Println("  Falling back to file save")
			return saveSecretToFile(secret, "nylas-client-secret.txt", label)
		}
		_, _ = common.Green.Printf("✓ %s copied to clipboard\n", label)

	case "file":
		return saveSecretToFile(secret, "nylas-client-secret.txt", label)

	default:
		return dashboardError(
			"invalid secret delivery method",
			"Use --secret-delivery clipboard or --secret-delivery file",
		)
	}

	return nil
}

func validateAPIKeyDelivery(delivery string) error {
	switch delivery {
	case "", "activate", "clipboard", "file":
		return nil
	default:
		return dashboardError(
			"invalid API key delivery method",
			"Use --delivery activate, --delivery clipboard, or --delivery file",
		)
	}
}

func validateSecretDelivery(delivery string) error {
	switch delivery {
	case "", "clipboard", "file":
		return nil
	default:
		return dashboardError(
			"invalid secret delivery method",
			"Use --secret-delivery clipboard or --secret-delivery file",
		)
	}
}

// saveSecretToFile writes a secret to a temp file with restrictive permissions.
func saveSecretToFile(secret, filename, label string) error {
	keyFile, err := writeSecretTempFile(secret, filename)
	if err != nil {
		return wrapDashboardError(fmt.Errorf("failed to write file: %w", err))
	}
	_, _ = common.Green.Printf("✓ %s saved to: %s\n", label, keyFile)
	_, _ = common.Dim.Println("  Read it, then delete the file")
	return nil
}

func writeSecretTempFile(secret, filename string) (string, error) {
	pattern := tempSecretPattern(filename)
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	if err := file.Chmod(0o600); err != nil {
		return "", err
	}
	if _, err := file.WriteString(secret + "\n"); err != nil {
		return "", err
	}

	return file.Name(), nil
}

func tempSecretPattern(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		name = "nylas-secret"
	}
	return name + "-*" + ext
}

// activateAPIKey stores the API key and configures the CLI to use it.
func activateAPIKey(apiKey, clientID, region, orgID string) error {
	if strings.TrimSpace(clientID) == "" {
		return fmt.Errorf("client ID is required to activate an API key")
	}

	configStore := config.NewDefaultFileStore()
	secretStore, err := keyring.NewSecretStore(config.DefaultConfigDir())
	if err != nil {
		return err
	}

	configSvc := authapp.NewConfigService(configStore, secretStore)
	return configSvc.SetupConfig(region, clientID, "", apiKey, orgID)
}

// toAPIKeyRows converts API keys to flat display rows.
func toAPIKeyRows(keys []domain.GatewayAPIKey) []apiKeyRow {
	rows := make([]apiKeyRow, len(keys))
	for i, k := range keys {
		rows[i] = apiKeyRow{
			ID:        k.ID,
			Name:      k.Name,
			Status:    k.Status,
			ExpiresAt: formatEpoch(k.ExpiresAt),
			CreatedAt: formatEpoch(k.CreatedAt),
		}
	}
	return rows
}

// formatEpoch formats a Unix epoch (seconds) as a human-readable date.
func formatEpoch(epoch float64) string {
	if epoch == 0 {
		return "-"
	}
	t := time.Unix(int64(epoch), 0)
	return t.Format("2006-01-02")
}
