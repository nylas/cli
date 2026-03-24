package config

import (
	"fmt"

	"github.com/spf13/cobra"

	adapterconfig "github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	authapp "github.com/nylas/cli/internal/app/auth"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

func newResetCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset all CLI configuration and credentials",
		Long: `Reset the Nylas CLI to a clean state by clearing all stored data:

  - API credentials (API key, client ID, client secret)
  - Dashboard session (login tokens, selected app)
  - Grants (authenticated email accounts)
  - Config file (reset to defaults)

After reset, run 'nylas init' to set up again.

To reset only part of the CLI:
  nylas auth config --reset    Reset API credentials only
  nylas dashboard logout       Log out of Dashboard only`,
		Example: `  # Reset with confirmation prompt
  nylas config reset

  # Reset without confirmation
  nylas config reset --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Println("This will remove all stored credentials, grants, and configuration.")
				fmt.Println()
				if !common.Confirm("Are you sure you want to reset the CLI?", false) {
					fmt.Println("Reset cancelled.")
					return nil
				}
				fmt.Println()
			}

			secretStore, err := keyring.NewSecretStore(adapterconfig.DefaultConfigDir())
			if err != nil {
				return fmt.Errorf("access secret store: %w", err)
			}

			// 1. Clear API credentials
			configSvc := authapp.NewConfigService(configStore, secretStore)
			if err := configSvc.ResetConfig(); err != nil {
				return fmt.Errorf("reset API config: %w", err)
			}
			_, _ = common.Green.Println("  ✓ API credentials cleared")

			// 2. Clear dashboard credentials
			clearDashboardCredentials(secretStore)
			_, _ = common.Green.Println("  ✓ Dashboard session cleared")

			// 3. Clear grants
			grantStore := keyring.NewGrantStore(secretStore)
			if err := grantStore.ClearGrants(); err != nil {
				return fmt.Errorf("clear grants: %w", err)
			}
			_, _ = common.Green.Println("  ✓ Grants cleared")

			// 4. Reset config file to defaults
			if err := configStore.Save(domain.DefaultConfig()); err != nil {
				return fmt.Errorf("reset config file: %w", err)
			}
			_, _ = common.Green.Println("  ✓ Config file reset")

			fmt.Println()
			_, _ = common.Green.Println("CLI has been reset.")
			fmt.Println()
			fmt.Println("Run 'nylas init' to set up again.")

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")

	return cmd
}

// clearDashboardCredentials removes all dashboard-related keys from the secret store.
func clearDashboardCredentials(secrets ports.SecretStore) {
	_ = secrets.Delete(ports.KeyDashboardUserToken)
	_ = secrets.Delete(ports.KeyDashboardOrgToken)
	_ = secrets.Delete(ports.KeyDashboardUserPublicID)
	_ = secrets.Delete(ports.KeyDashboardOrgPublicID)
	_ = secrets.Delete(ports.KeyDashboardDPoPKey)
	_ = secrets.Delete(ports.KeyDashboardAppID)
	_ = secrets.Delete(ports.KeyDashboardAppRegion)
}
