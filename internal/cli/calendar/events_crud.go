package calendar

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
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
		ignoreDSTWarning   bool
		ignoreWorkingHours bool
		lockTimezone       bool
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

			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Get calendar ID if not specified
				calID, err := GetDefaultCalendarID(ctx, client, grantID, calendarID, true)
				if err != nil {
					return struct{}{}, err
				}

				// Parse times
				when, err := parseEventTime(startTime, endTime, allDay)
				if err != nil {
					return struct{}{}, err
				}

				// Check for DST warnings (unless ignored or all-day event)
				if !ignoreDSTWarning && !allDay {
					eventStart := when.StartDateTime()
					eventTZ := eventStart.Location().String()
					if eventTZ == "Local" {
						eventTZ = getLocalTimeZone()
					}

					// Check for DST conflict
					dstWarning, err := checkDSTConflict(eventStart, eventTZ, when.EndDateTime().Sub(eventStart))
					if err == nil && dstWarning != nil {
						// Display DST warning
						if !confirmDSTConflict(dstWarning) {
							fmt.Println("Cancelled.")
							return struct{}{}, nil
						}
					}
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
							return struct{}{}, fmt.Errorf("event conflicts with break time")
						}

						// Check for working hours violations (soft warning - can override)
						violation := checkWorkingHoursViolation(eventStart, cfg)
						if violation != "" {
							// Get schedule for display
							weekday := strings.ToLower(eventStart.Weekday().String())
							schedule := cfg.WorkingHours.GetScheduleForDay(weekday)

							if !confirmWorkingHoursViolation(violation, eventStart, schedule) {
								fmt.Println("Cancelled.")
								return struct{}{}, nil
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

				// Set timezone lock in metadata if requested
				if lockTimezone && !allDay {
					if req.Metadata == nil {
						req.Metadata = make(map[string]string)
					}
					req.Metadata["timezone_locked"] = "true"
				}

				event, err := common.RunWithSpinnerResult("Creating event...", func() (*domain.Event, error) {
					return client.CreateEvent(ctx, grantID, calID, req)
				})
				if err != nil {
					return struct{}{}, common.WrapCreateError("event", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(event)
				}

				fmt.Printf("%s Event created successfully!\n\n", common.Green.Sprint("✓"))
				fmt.Printf("Title: %s\n", event.Title)
				fmt.Printf("When: %s\n", formatEventTime(event.When))
				if lockTimezone && !allDay {
					fmt.Printf("%s %s\n", common.Cyan.Sprint("🔒 Timezone locked:"), when.StartTimezone)
					fmt.Println("     This event will always display in this timezone, regardless of viewer's location.")
				}
				fmt.Printf("ID: %s\n", event.ID)

				return struct{}{}, nil
			})
			return err
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
	cmd.Flags().BoolVar(&ignoreDSTWarning, "ignore-dst-warning", false, "Skip DST conflict warnings")
	cmd.Flags().BoolVar(&ignoreWorkingHours, "ignore-working-hours", false, "Skip working hours validation")
	cmd.Flags().BoolVar(&lockTimezone, "lock-timezone", false, "Lock event to its timezone (always display in this timezone)")

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
			// Parse arguments
			resourceArgs, err := common.ParseResourceArgs(args, 1)
			if err != nil {
				return err
			}

			client, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Get calendar ID if not specified
			calendarID, err = GetDefaultCalendarID(ctx, client, resourceArgs.GrantID, calendarID, false)
			if err != nil {
				return err
			}

			// Wrap DeleteEvent to match the DeleteFunc signature
			deleteFunc := func(ctx context.Context, grantID, resourceID string) error {
				return client.DeleteEvent(ctx, grantID, calendarID, resourceID)
			}

			// Run delete with standard helpers
			return common.RunDelete(common.DeleteConfig{
				ResourceName: "event",
				ResourceID:   resourceArgs.ResourceID,
				GrantID:      resourceArgs.GrantID,
				Force:        force,
				DeleteFunc:   deleteFunc,
			})
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
			grantArgs := args[1:]

			_, err := common.WithClient(grantArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Get calendar ID if not specified
				calID, err := GetDefaultCalendarID(ctx, client, grantID, calendarID, false)
				if err != nil {
					return struct{}{}, err
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
						return struct{}{}, err
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
					return struct{}{}, common.NewUserError(
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
					return client.UpdateEvent(ctx, grantID, calID, eventID, req)
				})
				if err != nil {
					return struct{}{}, common.WrapUpdateError("event", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(event)
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

				return struct{}{}, nil
			})
			return err
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
