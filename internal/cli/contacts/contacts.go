// Package contacts provides contacts-related CLI commands.
package contacts

import (
	"github.com/spf13/cobra"
)

// NewContactsCmd creates the contacts command group.
func NewContactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contacts",
		Aliases: []string{"contact"},
		Short:   "Manage contacts",
		Long: `Manage contacts from your connected accounts.

View contacts, create new contacts, update and delete contacts.

API reference: https://developer.nylas.com/docs/v3/email/contacts/`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newGroupsCmd())
	cmd.AddCommand(newSearchCmd())
	cmd.AddCommand(newPhotoCmd())
	cmd.AddCommand(newSyncCmd())

	return cmd
}
