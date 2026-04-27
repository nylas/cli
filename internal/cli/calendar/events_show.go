package calendar

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newEventsShowCmd() *cobra.Command {
	var (
		calendarID string
		targetTZ   string
		showTZ     bool
	)

	cmd := &cobra.Command{
		Use:     "show <event-id> [grant-id]",
		Aliases: []string{"read", "get"},
		Short:   "Show event details",
		Long: `Display detailed information about a specific event.

Examples:
  # Show event in local timezone
  nylas calendar events show <event-id>

  # Show event in a specific timezone
  nylas calendar events show <event-id> --timezone Europe/London

  # Show event with timezone abbreviations
  nylas calendar events show <event-id> --show-tz`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := args[0]
			grantArgs := args[1:]

			// Auto-detect timezone if not specified
			if targetTZ == "" && !cmd.Flags().Changed("timezone") {
				targetTZ = getLocalTimeZone()
			}

			// Validate timezone if specified
			if targetTZ != "" {
				if err := validateTimeZone(targetTZ); err != nil {
					return err
				}
			}

			_, err := common.WithClient(grantArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Get calendar ID if not specified
				calID, err := GetDefaultCalendarID(ctx, client, grantID, calendarID, false)
				if err != nil {
					return struct{}{}, err
				}

				event, err := client.GetEvent(ctx, grantID, calID, eventID)
				if err != nil {
					return struct{}{}, common.WrapGetError("event", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(event)
				}

				// Title
				fmt.Printf("%s\n\n", common.BoldCyan.Sprint(event.Title))

				// Time (with timezone conversion if requested)
				fmt.Printf("%s\n", common.Green.Sprint("When"))
				timeDisplay, err := formatEventTimeWithTZ(event, targetTZ)
				if err != nil {
					fmt.Printf("  %s (timezone conversion error: %v)\n\n",
						formatEventTime(event.When),
						err)
				} else {
					if timeDisplay.ShowConversion {
						// Show converted time prominently
						fmt.Printf("  %s", timeDisplay.ConvertedTime)
						if showTZ {
							fmt.Printf(" %s", common.Blue.Sprint(timeDisplay.ConvertedTimezone))
						}
						fmt.Println()
						// Show original time as reference
						fmt.Printf("  %s %s",
							common.Dim.Sprint("(Original:"),
							common.Dim.Sprint(timeDisplay.OriginalTime))
						if showTZ {
							fmt.Printf(" %s", common.Dim.Sprint(timeDisplay.OriginalTimezone))
						}
						fmt.Printf("%s\n\n", common.Dim.Sprint(")"))
					} else {
						// No conversion - show original time
						fmt.Printf("  %s", timeDisplay.OriginalTime)
						if showTZ && timeDisplay.OriginalTimezone != "" {
							fmt.Printf(" %s", common.Blue.Sprint(timeDisplay.OriginalTimezone))
						}
						fmt.Printf("\n\n")
					}
				}

				// DST Warning (if applicable)
				if !event.When.IsAllDay() {
					start := event.When.StartDateTime()
					eventTZ := start.Location().String()
					if eventTZ == "Local" {
						eventTZ = getLocalTimeZone()
					}

					dstWarning := checkDSTWarning(start, eventTZ)
					if dstWarning != "" {
						fmt.Printf("  %s\n\n", common.Yellow.Sprint(dstWarning))
					}
				}

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

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (defaults to primary)")
	cmd.Flags().StringVar(&targetTZ, "timezone", "", "Display times in this timezone (e.g., America/Los_Angeles). Defaults to local timezone.")
	cmd.Flags().BoolVar(&showTZ, "show-tz", false, "Show timezone abbreviations (e.g., PST, EST)")

	return cmd
}
