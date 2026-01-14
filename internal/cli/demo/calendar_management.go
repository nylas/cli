package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
)

func newDemoCalendarsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendars",
		Short: "List sample calendars",
		Long:  "Display a list of sample calendars.",
		Example: `  # List sample calendars
  nylas demo calendar calendars`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			calendars, err := client.GetCalendars(ctx, "demo-grant")
			if err != nil {
				return common.WrapListError("calendars", err)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📅 Demo Mode - Sample Calendars"))
			fmt.Println()

			for _, cal := range calendars {
				primary := ""
				if cal.IsPrimary {
					primary = common.Green.Sprint(" (primary)")
				}
				fmt.Printf("  %s %s%s\n", cal.HexColor, common.BoldWhite.Sprint(cal.Name), primary)
				_, _ = common.Dim.Printf("    ID: %s\n", cal.ID)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To connect your real calendar: nylas auth login"))

			return nil
		},
	}

	return cmd
}

// newDemoCalendarShowCmd shows a sample calendar.
func newDemoCalendarShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [calendar-id]",
		Short: "Show sample calendar details",
		Long:  "Display details of a sample calendar.",
		Example: `  # Show primary calendar
  nylas demo calendar show primary

  # Show specific calendar
  nylas demo calendar show cal-work-123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			calID := "primary"
			if len(args) > 0 {
				calID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📅 Demo Mode - Calendar Details"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))

			// Show sample calendar details
			_, _ = common.BoldWhite.Println("Work Calendar")
			fmt.Printf("  ID:          %s\n", calID)
			fmt.Printf("  Owner:       demo@example.com\n")
			fmt.Printf("  Timezone:    America/New_York\n")
			fmt.Printf("  Color:       %s\n", common.Cyan.Sprint("●"))
			fmt.Printf("  Primary:     %s\n", common.Green.Sprint("Yes"))
			fmt.Printf("  Read-only:   No\n")
			fmt.Printf("  Description: Work meetings and appointments\n")

			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To view your real calendars: nylas auth login"))

			return nil
		},
	}

	return cmd
}

// ============================================================================
// EVENT MANAGEMENT COMMANDS
// ============================================================================

// newDemoCalendarListCmd lists sample calendar events.
func newDemoCalendarListCmd() *cobra.Command {
	var limit int
	var showID bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sample calendar events",
		Long:  "Display a list of realistic sample calendar events.",
		Example: `  # List sample events
  nylas demo calendar list

  # List with IDs shown
  nylas demo calendar list --id`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			events, err := client.GetEvents(ctx, "demo-grant", "primary", nil)
			if err != nil {
				return common.WrapListError("events", err)
			}

			if limit > 0 && limit < len(events) {
				events = events[:limit]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📅 Demo Mode - Sample Events"))
			fmt.Println(common.Dim.Sprint("These are sample events for demonstration purposes."))
			fmt.Println()
			fmt.Printf("Found %d events:\n\n", len(events))

			for _, event := range events {
				printDemoEvent(event, showID)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To connect your real calendar: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of events to show")
	cmd.Flags().BoolVar(&showID, "id", false, "Show event IDs")

	return cmd
}

// newDemoCalendarCreateCmd simulates creating a calendar event.
func newDemoCalendarCreateCmd() *cobra.Command {
	var title string
	var startTime string
	var duration int
	var location string
	var attendees []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Simulate creating a calendar event",
		Long: `Simulate creating a calendar event to see how the create command works.

No actual event is created - this is just a demonstration of the command flow.`,
		Example: `  # Simulate creating an event
  nylas demo calendar create --title "Team Meeting" --start "2024-01-15 10:00" --duration 60

  # With location and attendees
  nylas demo calendar create --title "Lunch" --start "tomorrow 12:00" --location "Downtown Cafe" --attendee "john@example.com"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				title = "Demo Meeting"
			}
			if startTime == "" {
				startTime = time.Now().Add(1 * time.Hour).Format("Jan 2, 2006 3:04 PM")
			}
			if duration == 0 {
				duration = 30
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📅 Demo Mode - Simulated Event Creation"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Printf("Title:    %s\n", title)
			fmt.Printf("Start:    %s\n", startTime)
			fmt.Printf("Duration: %d minutes\n", duration)
			if location != "" {
				fmt.Printf("Location: %s\n", location)
			}
			if len(attendees) > 0 {
				fmt.Printf("Attendees: %s\n", strings.Join(attendees, ", "))
			}
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()
			_, _ = common.Green.Println("✓ Event would be created (demo mode - no actual event created)")
			_, _ = common.Dim.Printf("  Event ID: evt-demo-%d\n", time.Now().Unix())
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To create real events, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Event title")
	cmd.Flags().StringVar(&startTime, "start", "", "Start time")
	cmd.Flags().IntVar(&duration, "duration", 30, "Duration in minutes")
	cmd.Flags().StringVar(&location, "location", "", "Event location")
	cmd.Flags().StringArrayVar(&attendees, "attendee", nil, "Attendee email (can be repeated)")

	return cmd
}

// newDemoCalendarUpdateCmd simulates updating a calendar event.
func newDemoCalendarUpdateCmd() *cobra.Command {
	var title string
	var startTime string
	var location string

	cmd := &cobra.Command{
		Use:   "update [event-id]",
		Short: "Simulate updating a calendar event",
		Long:  "Simulate updating a calendar event to see how the update command works.",
		Example: `  # Update event title
  nylas demo calendar update evt-123 --title "Updated Meeting"

  # Update location
  nylas demo calendar update evt-123 --location "Conference Room B"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := "evt-demo-123"
			if len(args) > 0 {
				eventID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📅 Demo Mode - Simulated Event Update"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.Dim.Printf("Event ID: %s\n", eventID)
			fmt.Println()
			_, _ = common.BoldWhite.Println("Changes:")
			if title != "" {
				fmt.Printf("  Title:    %s\n", title)
			}
			if startTime != "" {
				fmt.Printf("  Start:    %s\n", startTime)
			}
			if location != "" {
				fmt.Printf("  Location: %s\n", location)
			}
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()
			_, _ = common.Green.Println("✓ Event would be updated (demo mode - no actual changes made)")
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To update real events, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New event title")
	cmd.Flags().StringVar(&startTime, "start", "", "New start time")
	cmd.Flags().StringVar(&location, "location", "", "New event location")

	return cmd
}

// newDemoCalendarDeleteCmd simulates deleting a calendar event.
func newDemoCalendarDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [event-id]",
		Short: "Simulate deleting a calendar event",
		Long:  "Simulate deleting a calendar event to see how the delete command works.",
		Example: `  # Delete an event
  nylas demo calendar delete evt-123

  # Force delete without confirmation
  nylas demo calendar delete evt-123 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := "evt-demo-123"
			if len(args) > 0 {
				eventID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📅 Demo Mode - Simulated Event Deletion"))
			fmt.Println()

			if !force {
				_, _ = common.Yellow.Println("⚠ Would prompt for confirmation in real mode")
			}

			fmt.Printf("Event ID: %s\n", eventID)
			fmt.Println()
			_, _ = common.Green.Println("✓ Event would be deleted (demo mode - no actual deletion)")
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To delete real events, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

// ============================================================================
// EVENTS SUBCOMMAND GROUP
// ============================================================================

// newDemoEventsCmd creates the events subcommand group.
