package calendar

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

// newVirtualCmd creates the virtual calendar command.
func newVirtualCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "virtual",
		Short: "Manage virtual calendars",
		Long: `Virtual calendars allow scheduling without connecting to a third-party provider.
Perfect for conference rooms, equipment, or external contractors.`,
	}

	cmd.AddCommand(newVirtualListCmd())
	cmd.AddCommand(newVirtualCreateCmd())
	cmd.AddCommand(newVirtualShowCmd())
	cmd.AddCommand(newVirtualDeleteCmd())

	return cmd
}

// newVirtualListCmd creates the list virtual calendars command.
func newVirtualListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all virtual calendar grants",
		Long:  "List all virtual calendar accounts (grants) in your Nylas application.",
		Example: `  # List all virtual calendars
  nylas calendar virtual list

  # List in JSON format
  nylas calendar virtual list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			grants, err := client.ListVirtualCalendarGrants(ctx)
			if err != nil {
				return common.WrapFetchError("virtual calendar grants", err)
			}

			if jsonOutput {
				return json.NewEncoder(os.Stdout).Encode(grants)
			}

			table := common.NewTable("ID", "EMAIL", "STATUS", "CREATED")
			for _, grant := range grants {
				created := time.Unix(grant.CreatedAt, 0).Format(common.DateTimeFormat)
				table.AddRow(grant.ID, grant.Email, grant.GrantStatus, created)
			}
			table.Render()

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// newVirtualCreateCmd creates the create virtual calendar command.
func newVirtualCreateCmd() *cobra.Command {
	var (
		email      string
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new virtual calendar grant",
		Long: `Create a new virtual calendar account (grant).
The email can be any identifier - it doesn't need to be a real email address.`,
		Example: `  # Create a virtual calendar for a conference room
  nylas calendar virtual create --email conference-room-a@company.com

  # Create a virtual calendar for equipment
  nylas calendar virtual create --email projector-1@company.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if email == "" {
				return common.NewUserError("email is required", "Use --email to specify an identifier for the virtual calendar")
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			grant, err := client.CreateVirtualCalendarGrant(ctx, email)
			if err != nil {
				return common.WrapCreateError("virtual calendar grant", err)
			}

			if jsonOutput {
				return json.NewEncoder(os.Stdout).Encode(grant)
			}

			fmt.Printf("✓ Created virtual calendar grant\n")
			fmt.Printf("  ID:     %s\n", grant.ID)
			fmt.Printf("  Email:  %s\n", grant.Email)
			fmt.Printf("  Status: %s\n", grant.GrantStatus)
			fmt.Printf("\nYou can now create calendars using this grant ID:\n")
			fmt.Printf("  nylas calendar create %s --name \"My Calendar\"\n", grant.ID)

			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Email identifier for the virtual calendar (required)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	_ = cmd.MarkFlagRequired("email") // Hardcoded flag name, won't fail

	return cmd
}

// newVirtualShowCmd creates the show virtual calendar command.
func newVirtualShowCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <grant-id>",
		Short: "Show details of a virtual calendar grant",
		Long:  "Display detailed information about a specific virtual calendar grant.",
		Example: `  # Show virtual calendar details
  nylas calendar virtual show vcal-grant-123

  # Show in JSON format
  nylas calendar virtual show vcal-grant-123 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grantID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			grant, err := client.GetVirtualCalendarGrant(ctx, grantID)
			if err != nil {
				return common.WrapGetError("virtual calendar grant", err)
			}

			if jsonOutput {
				return json.NewEncoder(os.Stdout).Encode(grant)
			}

			fmt.Printf("Virtual Calendar Grant\n")
			fmt.Printf("  ID:       %s\n", grant.ID)
			fmt.Printf("  Provider: %s\n", grant.Provider)
			fmt.Printf("  Email:    %s\n", grant.Email)
			fmt.Printf("  Status:   %s\n", grant.GrantStatus)
			fmt.Printf("  Created:  %s\n", time.Unix(grant.CreatedAt, 0).Format(common.DisplayDateTime))
			fmt.Printf("  Updated:  %s\n", time.Unix(grant.UpdatedAt, 0).Format(common.DisplayDateTime))

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// newVirtualDeleteCmd creates the delete virtual calendar command.
func newVirtualDeleteCmd() *cobra.Command {
	var skipConfirm bool

	cmd := &cobra.Command{
		Use:   "delete <grant-id>",
		Short: "Delete a virtual calendar grant",
		Long:  "Delete a virtual calendar account and all its associated calendars and events.",
		Example: `  # Delete a virtual calendar (with confirmation)
  nylas calendar virtual delete vcal-grant-123

  # Delete without confirmation
  nylas calendar virtual delete vcal-grant-123 -y`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grantID := args[0]

			if !skipConfirm {
				fmt.Printf("Are you sure you want to delete virtual calendar grant %s? (y/N): ", grantID)
				var response string
				_, _ = fmt.Scanln(&response)
				if response != "y" && response != "Y" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if err := client.DeleteVirtualCalendarGrant(ctx, grantID); err != nil {
				return common.WrapDeleteError("virtual calendar grant", err)
			}

			fmt.Printf("✓ Deleted virtual calendar grant %s\n", grantID)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
