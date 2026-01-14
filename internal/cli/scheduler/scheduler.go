// Package scheduler provides scheduler-related CLI commands.
package scheduler

import (
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

var client ports.NylasClient

// NewSchedulerCmd creates the scheduler command group.
func NewSchedulerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "scheduler",
		Aliases: []string{"sched"},
		Short:   "Manage Nylas Scheduler",
		Long: `Manage Nylas Scheduler configurations, sessions, bookings, and pages.

The Nylas Scheduler allows you to create meeting booking workflows,
manage availability, and handle scheduling sessions.`,
	}

	cmd.AddCommand(newConfigurationsCmd())
	cmd.AddCommand(newSessionsCmd())
	cmd.AddCommand(newBookingsCmd())
	cmd.AddCommand(newPagesCmd())

	return cmd
}

// getClient creates and configures a Nylas client with caching.
// Delegates to common.GetNylasClient() for consistent credential handling.
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
