package auth

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newAddCmd() *cobra.Command {
	var (
		email      string
		provider   string
		setDefault bool
	)

	cmd := &cobra.Command{
		Use:   "add <grant-id>",
		Short: "Manually add an existing grant",
		Long: `Manually add an existing Nylas grant to your local configuration.

This is useful when you have an existing grant ID from the Nylas dashboard
or from another system that you want to use with this CLI.

The email and provider are auto-detected from Nylas API, but can be overridden
with flags if needed.

Example:
  nylas auth add abc123-grant-id
  nylas auth add abc123-grant-id --default
  nylas auth add abc123-grant-id --email user@example.com --provider microsoft`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grantID := args[0]

			grantSvc, _, err := createGrantService()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Fetch grant info from Nylas API
			grant, err := grantSvc.FetchGrantFromNylas(ctx, grantID)
			if err != nil {
				return common.WrapFetchError("grant", err)
			}
			if !grant.IsValid() {
				return fmt.Errorf("grant %s is not valid (status: %s)", grantID, grant.GrantStatus)
			}

			// Use API values if not overridden by flags
			grantEmail := grant.Email
			if email != "" {
				grantEmail = email
			}

			grantProvider := grant.Provider
			if provider != "" {
				p, err := domain.ParseProvider(provider)
				if err != nil {
					return common.NewUserError(fmt.Sprintf("invalid provider: %s", provider), "use 'google' or 'microsoft'")
				}
				grantProvider = p
			}

			// Add the grant
			if err := grantSvc.AddGrant(grantID, grantEmail, grantProvider, setDefault); err != nil {
				return err
			}

			_, _ = common.Green.Printf("✓ Added grant %s\n", grantID)
			fmt.Printf("  Email:    %s\n", grantEmail)
			fmt.Printf("  Provider: %s\n", grantProvider.DisplayName())
			if setDefault {
				fmt.Println("  Set as default account")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&email, "email", "e", "", "Email address (auto-detected from Nylas)")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider (auto-detected from Nylas)")
	cmd.Flags().BoolVar(&setDefault, "default", false, "Set as default account")

	return cmd
}
