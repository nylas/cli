package calendar

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list [grant-id]",
		Aliases: []string{"ls"},
		Short:   "List calendars",
		Long:    "List all calendars for the specified grant or default account.",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use generic WithClient helper to reduce boilerplate
			calendars, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) ([]domain.Calendar, error) {
				return client.GetCalendars(ctx, grantID)
			})
			if err != nil {
				return common.WrapListError("calendars", err)
			}

			if len(calendars) == 0 {
				if !common.IsStructuredOutput(cmd) {
					common.PrintEmptyState("calendars")
				}
				return nil
			}

			// Check if using structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(calendars)
			}

			// Table output with header
			if !common.IsQuiet() {
				fmt.Printf("Found %d calendar(s):\n\n", len(calendars))
			}

			// Use structured output system with wide mode support
			normalCols := []ports.Column{
				{Header: "Name", Field: "Name", Width: 20},
				{Header: "ID", Field: "ID", Width: 40},
				{Header: "Primary", Field: "IsPrimary"},
				{Header: "Read-Only", Field: "ReadOnly"},
			}

			wideCols := []ports.Column{
				{Header: "Name", Field: "Name"},
				{Header: "ID", Field: "ID", Width: -1}, // Full width, no truncation
				{Header: "Primary", Field: "IsPrimary"},
				{Header: "Read-Only", Field: "ReadOnly"},
			}

			return common.WriteListWithWideColumns(cmd, calendars, normalCols, wideCols)
		},
	}

	return cmd
}
