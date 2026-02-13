// auth.go provides authentication commands for Slack workspace connections.

package slack

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

// newAuthCmd creates the auth command for managing Slack authentication.
func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Slack authentication",
		Long: `Manage Slack authentication and stored credentials.

Get your User OAuth Token from:
  1. Go to https://api.slack.com/apps
  2. Create or select your app
  3. Go to "OAuth & Permissions"
  4. Add User Token Scopes (channels:read, channels:history, chat:write, etc.)
  5. Install to workspace
  6. Copy the "User OAuth Token" (starts with xoxp-)`,
	}

	cmd.AddCommand(newAuthSetCmd())
	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthRemoveCmd())

	return cmd
}

// newAuthSetCmd creates the set subcommand for storing a Slack token.
func newAuthSetCmd() *cobra.Command {
	var token string

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set Slack user token",
		Long: `Set your Slack user OAuth token for authentication.

The token should start with 'xoxp-' for user tokens.

Example:
  nylas slack auth set --token xoxp-1234567890-...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				return common.NewUserError(
					"--token is required",
					"Get your token from api.slack.com/apps -> OAuth & Permissions -> User OAuth Token",
				)
			}

			client, err := getSlackClient(token)
			if err != nil {
				return common.WrapCreateError("client", err)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			auth, err := client.TestAuth(ctx)
			if err != nil {
				return common.NewUserError("invalid token", "Check that your token is correct and has the required scopes")
			}

			if err := storeSlackToken(token); err != nil {
				return common.WrapSaveError("token", err)
			}

			_, _ = common.Green.Printf("✓ Authenticated as %s in workspace %s\n", auth.UserName, auth.TeamName)
			return nil
		},
	}

	cmd.Flags().StringVar(&token, "token", "", "Slack user OAuth token (xoxp-...)")
	_ = cmd.MarkFlagRequired("token")

	return cmd
}

// newAuthStatusCmd creates the status subcommand for showing auth state.
func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current Slack authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getSlackClientFromKeyring()
			if err != nil {
				_, _ = common.Yellow.Println("Not authenticated with Slack")
				fmt.Println("\nTo authenticate, run:")
				fmt.Println("  nylas slack auth set --token YOUR_TOKEN")
				fmt.Println("\nGet your token from: https://api.slack.com/apps")
				return nil
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			auth, err := client.TestAuth(ctx)
			if err != nil {
				return common.NewUserError("authentication failed", "Token may have expired or been revoked")
			}

			// Handle structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(auth)
			}

			_, _ = common.Green.Println("✓ Authenticated with Slack")
			fmt.Println()
			fmt.Printf("  User:      %s\n", common.Cyan.Sprint(auth.UserName))
			fmt.Printf("  Workspace: %s\n", auth.TeamName)
			_, _ = common.Dim.Printf("  User ID:   %s\n", auth.UserID)
			_, _ = common.Dim.Printf("  Team ID:   %s\n", auth.TeamID)

			return nil
		},
	}
}

// newAuthRemoveCmd creates the remove subcommand for deleting stored credentials.
func newAuthRemoveCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove stored Slack token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Print("Remove Slack authentication? [y/N]: ")
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			if err := removeSlackToken(); err != nil {
				return common.WrapDeleteError("token", err)
			}

			_, _ = common.Green.Println("✓ Slack token removed")
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "yes", "y", false, "Skip confirmation")

	return cmd
}
