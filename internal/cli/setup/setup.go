package setup

import (
	"github.com/spf13/cobra"
)

// NewSetupCmd creates the "init" command for first-time CLI setup.
func NewSetupCmd() *cobra.Command {
	var opts wizardOpts

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Set up the Nylas CLI",
		Long: `Guided setup for first-time users.

This wizard walks you through:
  1. Creating or logging into your Nylas account
  2. Selecting or creating an application
  3. Generating and activating an API key
  4. Syncing existing email accounts

Already have an API key? Skip the wizard:
  nylas init --api-key <your-key>`,
		Example: `  # Interactive guided setup
  nylas init

  # Quick setup with existing API key
  nylas init --api-key nyl_abc123

  # Quick setup with region
  nylas init --api-key nyl_abc123 --region eu

  # Skip SSO provider menu
  nylas init --google`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWizard(opts)
		},
	}

	cmd.Flags().StringVar(&opts.apiKey, "api-key", "", "Nylas API key (skips interactive setup)")
	cmd.Flags().StringVarP(&opts.region, "region", "r", "us", "API region (us or eu)")
	cmd.Flags().BoolVar(&opts.google, "google", false, "Use Google SSO")
	cmd.Flags().BoolVar(&opts.microsoft, "microsoft", false, "Use Microsoft SSO")
	cmd.Flags().BoolVar(&opts.github, "github", false, "Use GitHub SSO")

	return cmd
}
