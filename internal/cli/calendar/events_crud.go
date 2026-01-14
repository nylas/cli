package calendar

import (
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newEventsCreateCmd() *cobra.Command {
	var (
		calendarID         string
		title              string
		description        string
		location           string
		startTime          string
		endTime            string
		allDay             bool
		participants       []string
		busy               bool
		free               bool
		ignoreWorkingHours bool
	)

	cmd := &cobra.Command{
		Use:   "create [grant-id]",
		Short: "Create a new event",
		Long: `Create a new calendar event.

Examples:
  # Create a simple event
  nylas calendar events create --title "Meeting" --start "2024-01-15 14:00" --end "2024-01-15 15:00"

  # Create an all-day event
  nylas calendar events create --title "Vacation" --start "2024-01-15" --all-day

  # Create event with participants
  nylas calendar events create --title "Team Sync" --start "2024-01-15 10:00" --end "2024-01-15 11:00" \
    --participant "alice@example.com" --participant "bob@example.com"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return common.NewUserError(
					"title is required",
					"Use --title to specify event title",
				)
			}
			if startTime == "" {
				return common.NewUserError(
					"start time is required",
					"Use --start to specify start time (e.g., '2024-01-15 14:00' or '2024-01-15' for all-day)",
				)
			}

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

			// Get calendar ID if not specified
			if calendarID == "" {
				calendars, err := client.GetCalendars(ctx, grantID)
				if err != nil {
					return common.WrapListError("calendars", err)
				}
				for _, cal := range calendars {
					if cal.IsPrimary && !cal.ReadOnly {
						calendarID = cal.ID
						break
					}
				}
				// Fallback to any writable calendar
				if calendarID == "" {
					for _, cal := range calendars {
						if !cal.ReadOnly {
							calendarID = cal.ID
							break
						}
					}
				}
				if calendarID == "" {
					return common.NewUserError(
						"no writable calendar found",
						"Specify a calendar with --calendar",
					)
				}
			}

			// Parse times
			when, err := parseEventTime(startTime, endTime, allDay)
			if err != nil {
				return err
			}

			// Check for working hours violations (unless ignored or all-day event)
			if !ignoreWorkingHours && !allDay {
				eventStart := when.StartDateTime()

				// Load config to get working hours settings
				configStore := config.NewDefaultFileStore()
				cfg, err := configStore.Load()
				if err == nil && cfg != nil {
					// Check for break violations first (hard block - cannot override)
					breakViolation := checkBreakViolation(eventStart, cfg)
					if breakViolation != "" {
						_, _ = common.BoldRed.Println("\n⛔ Break Time Conflict")
						fmt.Printf("\n%s\n\n", breakViolation)
						fmt.Println("Tip: Schedule the event outside of break times, or update your")
						fmt.Println("     break configuration in ~/.nylas/config.yaml")
						return fmt.Errorf("event conflicts with break time")
					}

					// Check for working hours violations (soft warning - can override)
					violation := checkWorkingHoursViolation(eventStart, cfg)
					if violation != "" {
						// Get schedule for display
						weekday := strings.ToLower(eventStart.Weekday().String())
						schedule := cfg.WorkingHours.GetScheduleForDay(weekday)

						if !confirmWorkingHoursViolation(violation, eventStart, schedule) {
							fmt.Println("Cancelled.")
							return nil
						}
					}
				}
			}

			// --free flag overrides --busy
			if free {
				busy = false
			}

			req := &domain.CreateEventRequest{
				Title:       title,
				Description: description,
				Location:    location,
				When:        *when,
				Busy:        busy,
			}

			// Add participants
			for _, email := range participants {
				req.Participants = append(req.Participants, domain.Participant{
					Person: domain.Person{Email: email},
				})
			}

			event, err := common.RunWithSpinnerResult("Creating event...", func() (*domain.Event, error) {
				return client.CreateEvent(ctx, grantID, calendarID, req)
			})
			if err != nil {
				return common.WrapCreateError("event", err)
			}

			fmt.Printf("%s Event created successfully!\n\n", common.Green.Sprint("✓"))
			fmt.Printf("Title: %s\n", event.Title)
			fmt.Printf("When: %s\n", formatEventTime(event.When))
			fmt.Printf("ID: %s\n", event.ID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (defaults to primary)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Event title (required)")
	cmd.Flags().StringVarP(&description, "description", "D", "", "Event description")
	cmd.Flags().StringVarP(&location, "location", "l", "", "Event location")
	cmd.Flags().StringVarP(&startTime, "start", "s", "", "Start time (e.g., '2024-01-15 14:00' or '2024-01-15')")
	cmd.Flags().StringVarP(&endTime, "end", "e", "", "End time (defaults to 1 hour after start)")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "Create an all-day event")
	cmd.Flags().StringArrayVarP(&participants, "participant", "p", nil, "Add participant email (can be used multiple times)")
	cmd.Flags().BoolVar(&busy, "busy", true, "Mark time as busy")
	cmd.Flags().BoolVar(&free, "free", false, "Mark time as free (not busy)")
	cmd.Flags().BoolVar(&ignoreWorkingHours, "ignore-working-hours", false, "Skip working hours validation")

	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("start")

	return cmd
}

func newEventsDeleteCmd() *cobra.Command {
	var (
		calendarID string
		force      bool
	)

	cmd := &cobra.Command{
		Use:     "delete <event-id> [grant-id]",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete an event",
		Long:    "Delete a calendar event by its ID.",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := args[0]

			if !force {
				fmt.Printf("Are you sure you want to delete event %s? [y/N] ", eventID)
				var confirm string
				_, _ = fmt.Scanln(&confirm) // Ignore error - empty string treated as "no"
				if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

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

			err = common.RunWithSpinner("Deleting event...", func() error {
				return client.DeleteEvent(ctx, grantID, calendarID, eventID)
			})
			if err != nil {
				return common.WrapDeleteError("event", err)
			}

			fmt.Printf("%s Event deleted successfully.\n", common.Green.Sprint("✓"))

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (defaults to primary)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}

func newEventsUpdateCmd() *cobra.Command {
	var (
		calendarID     string
		title          string
		description    string
		location       string
		startTime      string
		endTime        string
		allDay         bool
		participants   []string
		busy           bool
		free           bool
		visibility     string
		lockTimezone   bool
		unlockTimezone bool
	)

	cmd := &cobra.Command{
		Use:   "update <event-id> [grant-id]",
		Short: "Update an existing event",
		Long: `Update a calendar event.

Examples:
  # Update event title
  nylas calendar events update <event-id> --title "New Title"

  # Update event time
  nylas calendar events update <event-id> --start "2024-01-15 14:00" --end "2024-01-15 15:00"

  # Update location and description
  nylas calendar events update <event-id> --location "Conference Room A" --description "Weekly sync"`,
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

			req := &domain.UpdateEventRequest{}

			if cmd.Flags().Changed("title") {
				req.Title = &title
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			if cmd.Flags().Changed("location") {
				req.Location = &location
			}
			if cmd.Flags().Changed("busy") {
				req.Busy = &busy
			}
			// --free flag overrides --busy
			if free {
				f := false
				req.Busy = &f
			}
			if cmd.Flags().Changed("visibility") {
				req.Visibility = &visibility
			}

			// Handle time changes
			if cmd.Flags().Changed("start") {
				when, err := parseEventTime(startTime, endTime, allDay)
				if err != nil {
					return err
				}
				req.When = when
			}

			// Handle participants
			if len(participants) > 0 {
				for _, email := range participants {
					req.Participants = append(req.Participants, domain.Participant{
						Person: domain.Person{Email: email},
					})
				}
			}

			// Handle timezone locking/unlocking
			if lockTimezone && unlockTimezone {
				return common.NewUserError(
					"cannot use both --lock-timezone and --unlock-timezone",
					"Use only one flag to either lock or unlock timezone",
				)
			}

			if lockTimezone {
				if req.Metadata == nil {
					req.Metadata = make(map[string]string)
				}
				req.Metadata["timezone_locked"] = "true"
			} else if unlockTimezone {
				if req.Metadata == nil {
					req.Metadata = make(map[string]string)
				}
				req.Metadata["timezone_locked"] = "false"
			}

			event, err := common.RunWithSpinnerResult("Updating event...", func() (*domain.Event, error) {
				return client.UpdateEvent(ctx, grantID, calendarID, eventID, req)
			})
			if err != nil {
				return common.WrapUpdateError("event", err)
			}

			fmt.Printf("%s Event updated successfully!\n\n", common.Green.Sprint("✓"))
			fmt.Printf("Title: %s\n", event.Title)
			fmt.Printf("When: %s\n", formatEventTime(event.When))
			if lockTimezone {
				fmt.Printf("%s Timezone is now locked\n", common.Cyan.Sprint("🔒"))
			} else if unlockTimezone {
				fmt.Printf("%s Timezone lock removed\n", common.Cyan.Sprint("🔓"))
			}
			fmt.Printf("ID: %s\n", event.ID)

			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (defaults to primary)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Event title")
	cmd.Flags().StringVarP(&description, "description", "D", "", "Event description")
	cmd.Flags().StringVarP(&location, "location", "l", "", "Event location")
	cmd.Flags().StringVarP(&startTime, "start", "s", "", "Start time (e.g., '2024-01-15 14:00')")
	cmd.Flags().StringVarP(&endTime, "end", "e", "", "End time")
	cmd.Flags().BoolVar(&allDay, "all-day", false, "Set as all-day event")
	cmd.Flags().StringArrayVarP(&participants, "participant", "p", nil, "Set participant emails (replaces existing)")
	cmd.Flags().BoolVar(&busy, "busy", true, "Mark time as busy")
	cmd.Flags().BoolVar(&free, "free", false, "Mark time as free (not busy)")
	cmd.Flags().StringVar(&visibility, "visibility", "", "Event visibility (public, private, default)")
	cmd.Flags().BoolVar(&lockTimezone, "lock-timezone", false, "Lock event to its timezone")
	cmd.Flags().BoolVar(&unlockTimezone, "unlock-timezone", false, "Remove timezone lock from event")

	return cmd
}
