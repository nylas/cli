package calendar

import (
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newEventsShowCmd() *cobra.Command {
	var calendarID string

	cmd := &cobra.Command{
		Use:     "show <event-id> [grant-id]",
		Aliases: []string{"read", "get"},
		Short:   "Show event details",
		Long: `Display detailed information about a specific event.

Examples:
  nylas calendar events show <event-id>
  nylas calendar events show <event-id> --calendar <calendar-id>`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := args[0]

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 1 {
				grantID = args[1]
			} else {
				grantID, err = getGrantID(nil)
				if err != nil {
					return err
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get calendar ID if not specified
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
			}

			event, err := client.GetEvent(ctx, grantID, calendarID, eventID)
			if err != nil {
				return common.WrapGetError("event", err)
			}

			// Title
			fmt.Printf("%s\n\n", common.BoldCyan.Sprint(event.Title))

			// Time
			fmt.Printf("%s\n", common.Green.Sprint("When"))
			fmt.Printf("  %s\n\n", formatEventTime(event.When))

			// Location
			if event.Location != "" {
				fmt.Printf("%s\n", common.Green.Sprint("Location"))
				fmt.Printf("  %s\n\n", event.Location)
			}

			// Description
			if event.Description != "" {
				fmt.Printf("%s\n", common.Green.Sprint("Description"))
				fmt.Printf("  %s\n\n", event.Description)
			}

			// Organizer
			if event.Organizer != nil {
				fmt.Printf("%s\n", common.Green.Sprint("Organizer"))
				if event.Organizer.Name != "" {
					fmt.Printf("  %s <%s>\n\n", event.Organizer.Name, event.Organizer.Email)
				} else {
					fmt.Printf("  %s\n\n", event.Organizer.Email)
				}
			}

			// Participants
			if len(event.Participants) > 0 {
				fmt.Printf("%s\n", common.Green.Sprint("Participants"))
				for _, p := range event.Participants {
					status := formatParticipantStatus(p.Status)
					if p.Name != "" {
						fmt.Printf("  %s <%s> %s\n", p.Name, p.Email, status)
					} else {
						fmt.Printf("  %s %s\n", p.Email, status)
					}
				}
				fmt.Println()
			}

			// Conferencing
			if event.Conferencing != nil && event.Conferencing.Details != nil {
				fmt.Printf("%s\n", common.Green.Sprint("Video Conference"))
				if event.Conferencing.Provider != "" {
					fmt.Printf("  Provider: %s\n", event.Conferencing.Provider)
				}
				if event.Conferencing.Details.URL != "" {
					fmt.Printf("  URL: %s\n", event.Conferencing.Details.URL)
				}
				fmt.Println()
			}

			// Metadata
			fmt.Printf("%s\n", common.Green.Sprint("Details"))
			fmt.Printf("  Status: %s\n", event.Status)
			fmt.Printf("  Busy: %v\n", event.Busy)
			if event.Visibility != "" {
				fmt.Printf("  Visibility: %s\n", event.Visibility)
			}
			fmt.Printf("  ID: %s\n", common.Dim.Sprint(event.ID))
			fmt.Printf("  Calendar: %s\n", common.Dim.Sprint(event.CalendarID))

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (defaults to primary)")

	return cmd
}
