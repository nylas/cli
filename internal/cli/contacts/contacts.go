// Package contacts provides contacts-related CLI commands.
package contacts

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

var client ports.NylasClient

// NewContactsCmd creates the contacts command group.
func NewContactsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contacts",
		Aliases: []string{"contact"},
		Short:   "Manage contacts",
		Long: `Manage contacts from your connected accounts.

View contacts, create new contacts, update and delete contacts.`,
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

func getClient() (ports.NylasClient, error) {
	if client != nil {
		return client, nil
	}

	c, err := common.GetNylasClient()
	if err != nil {
		return nil, err
	}

	client = c
	return client, nil
}

// getGrantID gets the grant ID from args or default.
// Delegates to common.GetGrantID for consistent behavior across CLI commands.
func getGrantID(args []string) (string, error) {
	return common.GetGrantID(args)
}
