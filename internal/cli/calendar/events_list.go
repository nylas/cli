package calendar

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newEventsListCmd() *cobra.Command {
	var (
		calendarID string
		limit      int
		days       int
		showAll    bool
		targetTZ   string
		showTZ     bool
	)

	cmd := &cobra.Command{
		Use:     "list [grant-id]",
		Aliases: []string{"ls"},
		Short:   "List calendar events",
		Long: `List events from the specified calendar or primary calendar.

Examples:
  # List events in your local timezone
  nylas calendar events list

  # List events converted to a specific timezone
  nylas calendar events list --timezone America/Los_Angeles

  # List events with timezone abbreviations shown
  nylas calendar events list --show-tz`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Auto-detect timezone if not specified
			if targetTZ == "" && cmd.Flags().Changed("timezone") {
				// User explicitly set --timezone="" to clear
				targetTZ = ""
			} else if targetTZ == "" {
				// Default to local timezone for conversion display
				targetTZ = getLocalTimeZone()
			}

			// Validate timezone if specified
			if targetTZ != "" {
				if err := validateTimeZone(targetTZ); err != nil {
					return err
				}
			}

			// Auto-paginate when limit exceeds API maximum
			maxItems := 0
			if limit > common.MaxAPILimit {
				maxItems = limit
				limit = common.MaxAPILimit
			}

			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// If no calendar specified, try to get the primary calendar
				calID, err := GetDefaultCalendarID(ctx, client, grantID, calendarID, false)
				if err != nil {
					return struct{}{}, err
				}

				params := &domain.EventQueryParams{
					Limit:   limit,
					OrderBy: "start", // Sort by start time ascending
				}

				// Set time range if days specified
				if days > 0 {
					now := time.Now()
					params.Start = now.Unix()
					params.End = now.AddDate(0, 0, days).Unix()
				}

				if showAll {
					params.ShowCancelled = true
				}

				var events []domain.Event
				if maxItems > 0 {
					// Paginated fetch for large limits
					pageSize := min(limit, common.MaxAPILimit)
					params.Limit = pageSize

					fetcher := func(ctx context.Context, cursor string) (common.PageResult[domain.Event], error) {
						params.PageToken = cursor
						resp, err := client.GetEventsWithCursor(ctx, grantID, calID, params)
						if err != nil {
							return common.PageResult[domain.Event]{}, err
						}
						return common.PageResult[domain.Event]{
							Data:       resp.Data,
							NextCursor: resp.Pagination.NextCursor,
						}, nil
					}

					config := common.DefaultPaginationConfig()
					config.PageSize = pageSize
					config.MaxItems = maxItems

					events, err = common.FetchAllPages(ctx, config, fetcher)
					if err != nil {
						return struct{}{}, common.WrapListError("events", err)
					}
				} else {
					// Standard single-page fetch
					events, err = client.GetEvents(ctx, grantID, calID, params)
					if err != nil {
						return struct{}{}, common.WrapListError("events", err)
					}
				}

				// JSON output (including empty array)
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(events)
				}

				if len(events) == 0 {
					common.PrintEmptyState("events")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d event(s):\n\n", len(events))

				for _, event := range events {
					// Title with timezone badge (if showing timezone info)
					fmt.Printf("%s", common.Cyan.Sprint(event.Title))
					if showTZ && !event.When.IsAllDay() {
						// Get event's original timezone
						start := event.When.StartDateTime()
						originalTZ := start.Location().String()
						if originalTZ == "Local" {
							originalTZ = getLocalTimeZone()
						}

						// Add colored timezone badge
						badge := formatTimezoneBadge(originalTZ, true) // Use abbreviation
						fmt.Printf(" %s", common.Blue.Sprint(badge))
					}
					fmt.Println()

					// Time (with timezone conversion if requested)
					timeDisplay, err := formatEventTimeWithTZ(&event, targetTZ)
					if err != nil {
						fmt.Printf("  %s %s (timezone conversion error: %v)\n",
							common.Dim.Sprint("When:"),
							formatEventTime(event.When),
							err)
					} else {
						if timeDisplay.ShowConversion {
							// Show converted time prominently
							fmt.Printf("  %s %s", common.Dim.Sprint("When:"), timeDisplay.ConvertedTime)
							if showTZ {
								fmt.Printf(" %s", common.BoldBlue.Sprint(timeDisplay.ConvertedTimezone))
							}
							fmt.Println()
							// Show original time as reference
							fmt.Printf("       %s %s",
								common.Dim.Sprint("(Original:"),
								common.Dim.Sprint(timeDisplay.OriginalTime))
							if showTZ {
								fmt.Printf(" %s", common.Dim.Sprint(timeDisplay.OriginalTimezone))
							}
							fmt.Printf("%s\n", common.Dim.Sprint(")"))
						} else {
							// No conversion - show original time
							fmt.Printf("  %s %s", common.Dim.Sprint("When:"), timeDisplay.OriginalTime)
							if showTZ && timeDisplay.OriginalTimezone != "" {
								fmt.Printf(" %s", common.BoldBlue.Sprint(timeDisplay.OriginalTimezone))
							}
							fmt.Println()
						}
					}

					// Location
					if event.Location != "" {
						fmt.Printf("  %s %s\n", common.Dim.Sprint("Location:"), event.Location)
					}

					// Status
					statusColor := common.Green
					switch event.Status {
					case "cancelled":
						statusColor = common.Red
					case "tentative":
						statusColor = common.Yellow
					}
					if event.Status != "" {
						fmt.Printf("  %s %s\n", common.Dim.Sprint("Status:"), statusColor.Sprint(event.Status))
					}

					// Participants count
					if len(event.Participants) > 0 {
						fmt.Printf("  %s %d participant(s)\n", common.Dim.Sprint("Guests:"), len(event.Participants))
					}

					// ID
					fmt.Printf("  %s %s\n", common.Dim.Sprint("ID:"), common.Dim.Sprint(event.ID))
					fmt.Println()
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (defaults to primary)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Maximum number of events to show (auto-paginates if >200)")
	cmd.Flags().IntVarP(&days, "days", "d", 7, "Show events for the next N days (0 for no limit)")
	cmd.Flags().BoolVar(&showAll, "show-cancelled", false, "Include cancelled events")
	cmd.Flags().StringVar(&targetTZ, "timezone", "", "Display times in this timezone (e.g., America/Los_Angeles). Defaults to local timezone.")
	cmd.Flags().BoolVar(&showTZ, "show-tz", false, "Show timezone abbreviations (e.g., PST, EST)")

	return cmd
}
