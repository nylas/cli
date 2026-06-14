package calendar

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newResourcesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resources [grant-id]",
		Aliases: []string{"rooms"},
		Short:   "List bookable room and equipment resources",
		Long: `List the room and equipment resources you can book for meetings.

Each resource's email address doubles as a calendar ID, so you can add it as a
participant when creating events or pass it to availability/free-busy checks.

API reference: https://developer.nylas.com/docs/v3/calendar/`,
		Example: `  # List bookable rooms
  nylas calendar resources

  # JSON output
  nylas calendar resources --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resources, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) ([]domain.RoomResource, error) {
				return client.ListRoomResources(ctx, grantID)
			})
			if err != nil {
				return common.WrapListError("room resources", err)
			}

			if len(resources) == 0 {
				if !common.IsStructuredOutput(cmd) {
					common.PrintEmptyState("room resources")
				}
				return nil
			}

			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(resources)
			}

			if !common.IsQuiet() {
				fmt.Printf("Found %d room resource(s):\n\n", len(resources))
			}

			normalCols := []ports.Column{
				{Header: "Name", Field: "Name", Width: 24},
				{Header: "Email", Field: "Email", Width: 36},
				{Header: "Capacity", Field: "Capacity"},
				{Header: "Building", Field: "Building"},
				{Header: "Floor", Field: "FloorName"},
			}
			wideCols := []ports.Column{
				{Header: "Name", Field: "Name"},
				{Header: "Email", Field: "Email", Width: -1},
				{Header: "Capacity", Field: "Capacity"},
				{Header: "Building", Field: "Building"},
				{Header: "Floor", Field: "FloorName"},
			}

			return common.WriteListWithWideColumns(cmd, resources, normalCols, wideCols)
		},
	}

	return cmd
}
