package calendar

import (
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newEventsListCmd() *cobra.Command {
	var (
		calendarID string
		limit      int
		days       int
		showAll    bool
	)

	cmd := &cobra.Command{
		Use:     "list [grant-id]",
		Aliases: []string{"ls"},
		Short:   "List calendar events",
		Long: `List events from the specified calendar or primary calendar.

Examples:
  nylas calendar events list
  nylas calendar events list --days 14
  nylas calendar events list --limit 20`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := getGrantID(args)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// If no calendar specified, try to get the primary calendar
			if calendarID == "" {
				calendars, err := client.GetCalendars(ctx, grantID)
				if err != nil {
					return common.WrapListError("calendars", err)
				}
				for _, cal := range calendars {
					if cal.IsPrimary {
						calendarID = cal.ID
						break
					}
				}
				if calendarID == "" && len(calendars) > 0 {
					calendarID = calendars[0].ID
				}
				if calendarID == "" {
					return common.NewUserError(
						"no calendars found",
						"Connect a calendar account with: nylas auth login",
					)
				}
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

			events, err := client.GetEvents(ctx, grantID, calendarID, params)
			if err != nil {
				return common.WrapListError("events", err)
			}

			if len(events) == 0 {
				common.PrintEmptyState("events")
				return nil
			}

			fmt.Printf("Found %d event(s):\n\n", len(events))

			for _, event := range events {
				// Title
				fmt.Printf("%s\n", common.Cyan.Sprint(event.Title))

				// Time
				fmt.Printf("  %s %s\n", common.Dim.Sprint("When:"), formatEventTime(event.When))

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

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (defaults to primary)")
	cmd.Flags().IntVarP(&limit, "limit", "n", 10, "Maximum number of events to show")
	cmd.Flags().IntVarP(&days, "days", "d", 7, "Show events for the next N days (0 for no limit)")
	cmd.Flags().BoolVar(&showAll, "show-cancelled", false, "Include cancelled events")

	return cmd
}
