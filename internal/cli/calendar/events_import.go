package calendar

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newEventsImportCmd() *cobra.Command {
	var (
		calendarID string
		startStr   string
		endStr     string
		limit      int
	)

	cmd := &cobra.Command{
		Use:   "import [grant-id]",
		Short: "Bulk-export events from a calendar (for migration/backup)",
		Long: `Bulk-read events from a calendar over a time window.

Unlike "events list", import is built for migration and backup: it reads events
directly from the provider (including expanded recurring instances) for the
given window. A calendar is required. When --start/--end are omitted the API
defaults to now through one month ahead.

Use --json to capture the full event data for export or syncing. A single call
returns up to --limit events (max 500); raise --limit to export more.

API reference: https://developer.nylas.com/docs/v3/calendar/`,
		Example: `  # Export a year of events from the primary calendar as JSON
  nylas calendar events import --calendar primary \
    --start 2026-01-01 --end 2026-12-31 --json

  # Export from a specific calendar
  nylas calendar events import --calendar <calendar-id> --limit 200`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := parseImportTime("start", startStr)
			if err != nil {
				return err
			}
			end, err := parseImportTime("end", endStr)
			if err != nil {
				return err
			}

			_, err = common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				calID, err := GetDefaultCalendarID(ctx, client, grantID, calendarID, false)
				if err != nil {
					return struct{}{}, err
				}

				params := &domain.EventQueryParams{
					CalendarID: calID,
					Limit:      limit,
					Start:      start,
					End:        end,
				}

				events, err := client.ImportEvents(ctx, grantID, params)
				if err != nil {
					return struct{}{}, common.WrapListError("imported events", err)
				}

				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(events)
				}

				if len(events) == 0 {
					common.PrintEmptyState("events")
					return struct{}{}, nil
				}

				fmt.Printf("Imported %d event(s) from %s:\n\n", len(events), calID)
				for _, event := range events {
					title := event.Title
					if title == "" {
						title = "(no title)"
					}
					fmt.Printf("%s\n", common.Cyan.Sprint(title))
					fmt.Printf("  %s %s\n", common.Dim.Sprint("When:"), formatEventTime(event.When))
					fmt.Printf("  %s %s\n\n", common.Dim.Sprint("ID:"), common.Dim.Sprint(event.ID))
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID to import from (defaults to primary)")
	cmd.Flags().StringVar(&startStr, "start", "", "Start of the window (YYYY-MM-DD, 'YYYY-MM-DD HH:MM', or Unix). Defaults to now.")
	cmd.Flags().StringVar(&endStr, "end", "", "End of the window (YYYY-MM-DD, 'YYYY-MM-DD HH:MM', or Unix). Defaults to +1 month.")
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of events to import (max 500)")

	return cmd
}

// parseImportTime parses an import window bound into a Unix timestamp. An empty
// value returns 0 so the API default applies.
func parseImportTime(name, value string) (int64, error) {
	if value == "" {
		return 0, nil
	}
	if ts, err := strconv.ParseInt(value, 10, 64); err == nil && ts > 1000000000 {
		return ts, nil
	}
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02", time.RFC3339} {
		if t, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return t.Unix(), nil
		}
	}
	return 0, common.NewUserError(
		fmt.Sprintf("could not parse --%s value %q", name, value),
		"Use YYYY-MM-DD, 'YYYY-MM-DD HH:MM', or a Unix timestamp.",
	)
}
