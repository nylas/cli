package auth

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newLoginCmd() *cobra.Command {
	var provider string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with an email provider",
		Long: `Authenticate with an email provider via OAuth.

Supported providers:
  google     Google/Gmail
  microsoft  Microsoft/Outlook`,
		Example: `  # Login with Google (default)
  nylas auth login

  # Login with Google explicitly
  nylas auth login --provider google

  # Login with Microsoft/Outlook
  nylas auth login --provider microsoft`,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := parseLoginProvider(provider)
			if err != nil {
				return err
			}

			// Check if configured
			configSvc, _, _, err := createConfigService()
			if err != nil {
				return err
			}

			if !configSvc.IsConfigured() {
				return fmt.Errorf("nylas not configured - run 'nylas auth config' first")
			}

			// Create auth service
			authSvc, _, err := createAuthService()
			if err != nil {
				return err
			}

			fmt.Println("Opening browser for authentication...")
			fmt.Println("Complete the sign-in process in your browser.")

			ctx, cancel := common.CreateLongContext()
			defer cancel()

			grant, err := authSvc.Login(ctx, p)
			if err != nil {
				return err
			}

			_, _ = common.Green.Printf("\n✓ Successfully authenticated!\n")
			fmt.Printf("  Email:    %s\n", grant.Email)
			fmt.Printf("  Provider: %s\n", grant.Provider.DisplayName())
			fmt.Printf("  Grant ID: %s\n", grant.ID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "google", "Email provider (google, microsoft)")

	return cmd
}

func parseLoginProvider(provider string) (domain.Provider, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case string(domain.ProviderGoogle):
		return domain.ProviderGoogle, nil
	case string(domain.ProviderMicrosoft):
		return domain.ProviderMicrosoft, nil
	default:
		return "", common.NewUserError(
			fmt.Sprintf("invalid provider: %s (use 'google' or 'microsoft')", provider),
			"use 'google' or 'microsoft'",
		)
	}
}
