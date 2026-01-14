package contacts

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <contact-id> [grant-id]",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a contact",
		Long:    "Delete a contact by its ID.",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			contactID := args[0]

			if !force {
				fmt.Printf("Are you sure you want to delete contact %s? [y/N] ", contactID)
				var confirm string
				_, _ = fmt.Scanln(&confirm) // Ignore error - empty string treated as "no"
				if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			err = common.RunWithSpinner("Deleting contact...", func() error {
				return client.DeleteContact(ctx, grantID, contactID)
			})
			if err != nil {
				return common.WrapDeleteError("contact", err)
			}

			fmt.Printf("%s Contact deleted successfully.\n", common.Green.Sprint("✓"))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
