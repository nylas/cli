package calendar

import (
	"github.com/spf13/cobra"
)

func newEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "events",
		Aliases: []string{"ev", "event"},
		Short:   "Manage calendar events",
		Long: `List, create, update, delete, and manage calendar events

API reference: https://developer.nylas.com/docs/reference/api/events/`,
	}

	cmd.AddCommand(newEventsListCmd())
	cmd.AddCommand(newEventsShowCmd())
	cmd.AddCommand(newEventsCreateCmd())
	cmd.AddCommand(newEventsUpdateCmd())
	cmd.AddCommand(newEventsDeleteCmd())
	cmd.AddCommand(newEventsRSVPCmd())

	return cmd
}
