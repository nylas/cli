// Package scheduler provides scheduler-related CLI commands.
package scheduler

import (
	"github.com/spf13/cobra"
)

// NewSchedulerCmd creates the scheduler command group.
func NewSchedulerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scheduler",
		Aliases: []string{"sched"},
		Short:   "Manage Nylas Scheduler",
		Long: `Manage Nylas Scheduler configurations, sessions, and bookings.

The Nylas Scheduler allows you to create meeting booking workflows,
manage availability, and handle scheduling sessions.

API reference: https://developer.nylas.com/docs/v3/scheduler/`,
	}

	cmd.AddCommand(newConfigurationsCmd())
	cmd.AddCommand(newSessionsCmd())
	cmd.AddCommand(newBookingsCmd())

	return cmd
}
