// Package calendar provides calendar-related CLI commands.
package calendar

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

var client ports.NylasClient

// NewCalendarCmd creates the calendar command group.
func NewCalendarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "calendar",
		Aliases: []string{"cal"},
		Short:   "Manage calendars and events",
		Long: `Manage calendars and events from your connected accounts.

View calendars, list events, create new events, and more.`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newEventsCmd())
	cmd.AddCommand(newAvailabilityCmd())
	cmd.AddCommand(newVirtualCmd())
	cmd.AddCommand(newRecurringCmd())
	cmd.AddCommand(newFindTimeCmd())

	return cmd
}

func getClient() (ports.NylasClient, error) {
	if client != nil {
		return client, nil
	}

	// Use common helper that supports environment variables
	c, err := common.GetNylasClient()
	if err != nil {
		return nil, err
	}

	client = c
	return client, nil
}

func getGrantID(args []string) (string, error) {
	// Use common helper that supports environment variables
	return common.GetGrantID(args)
}
