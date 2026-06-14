package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newGroupEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "group-events",
		Aliases: []string{"group-event", "ge"},
		Short:   "Manage Scheduler group events",
		Long: `Manage Scheduler group events under a Configuration.

Group events let multiple participants book a single shared time slot (for
example, a workshop or webinar). They live under a Scheduler Configuration, so
every command takes a configuration ID.

API reference: https://developer.nylas.com/docs/v3/scheduler/`,
	}

	cmd.AddCommand(newGroupEventsListCmd())
	cmd.AddCommand(newGroupEventCreateCmd())
	cmd.AddCommand(newGroupEventUpdateCmd())
	cmd.AddCommand(newGroupEventDeleteCmd())
	cmd.AddCommand(newGroupEventsImportCmd())

	return cmd
}

func newGroupEventsListCmd() *cobra.Command {
	var (
		calendarID string
		startStr   string
		endStr     string
	)

	cmd := &cobra.Command{
		Use:     "list <configuration-id> [grant-id]",
		Aliases: []string{"ls"},
		Short:   "List group events for a configuration",
		Long: `List group events under a Scheduler Configuration within a time window.

A calendar and time window are required by the API; --start/--end default to
now through 30 days ahead when omitted.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			configID := args[0]
			if calendarID == "" {
				return common.NewUserError("a calendar is required", "Pass --calendar <calendar-id> (e.g. primary).")
			}
			start, err := parseGroupEventTime("start", startStr)
			if err != nil {
				return err
			}
			end, err := parseGroupEventTime("end", endStr)
			if err != nil {
				return err
			}
			// The list endpoint requires a window; default to now .. +30d.
			if start == 0 {
				start = time.Now().Unix()
			}
			if end == 0 {
				end = time.Now().AddDate(0, 0, 30).Unix()
			}

			events, err := common.WithClient(args[1:], func(ctx context.Context, client ports.NylasClient, grantID string) ([]domain.GroupEvent, error) {
				return client.ListGroupEvents(ctx, grantID, configID, calendarID, start, end)
			})
			if err != nil {
				return common.WrapListError("group events", err)
			}

			if common.IsStructuredOutput(cmd) {
				return common.GetOutputWriter(cmd).Write(events)
			}
			if len(events) == 0 {
				common.PrintEmptyState("group events")
				return nil
			}
			fmt.Printf("Found %d group event(s):\n\n", len(events))
			for _, e := range events {
				printGroupEvent(e)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID to list events from (required, e.g. primary)")
	cmd.Flags().StringVar(&startStr, "start", "", "Window start (YYYY-MM-DD HH:MM, RFC3339, or Unix). Defaults to now.")
	cmd.Flags().StringVar(&endStr, "end", "", "Window end (YYYY-MM-DD HH:MM, RFC3339, or Unix). Defaults to +30 days.")

	return cmd
}

func newGroupEventCreateCmd() *cobra.Command {
	var (
		calendarID   string
		title        string
		capacity     int
		description  string
		location     string
		startStr     string
		endStr       string
		timezone     string
		participants []string
		organizer    string
	)

	cmd := &cobra.Command{
		Use:   "create <configuration-id> [grant-id]",
		Short: "Create a group event",
		Long: `Create a group event under a Scheduler Configuration.

A calendar, title, capacity, time window, and at least one participant are
required. If no participants are given, the API uses the event organizer.`,
		Example: `  nylas scheduler group-events create <config-id> \
    --calendar primary --title "Philosophy Workshop" --capacity 50 \
    --start "2026-07-01 18:00" --end "2026-07-01 19:00" --timezone America/New_York \
    --organizer "Nyla:nyla@example.com"`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			configID := args[0]

			start, err := parseGroupEventTime("start", startStr)
			if err != nil {
				return err
			}
			end, err := parseGroupEventTime("end", endStr)
			if err != nil {
				return err
			}
			if calendarID == "" || title == "" || start == 0 || end == 0 {
				return common.NewUserError(
					"missing required fields",
					"Provide --calendar, --title, --start, and --end (and usually --capacity).",
				)
			}

			parts := buildGroupParticipants(participants, organizer)
			req := &domain.CreateGroupEventRequest{
				CalendarID:   calendarID,
				Title:        title,
				Capacity:     capacity,
				Description:  description,
				Location:     location,
				Participants: parts,
				When: &domain.GroupEventWhen{
					StartTime:     start,
					EndTime:       end,
					StartTimezone: timezone,
					EndTimezone:   timezone,
				},
			}

			events, err := common.WithClient(args[1:], func(ctx context.Context, client ports.NylasClient, grantID string) ([]domain.GroupEvent, error) {
				return client.CreateGroupEvent(ctx, grantID, configID, req)
			})
			if err != nil {
				return common.WrapCreateError("group event", err)
			}
			return reportGroupEvents(cmd, events, "Group event created")
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (required)")
	cmd.Flags().StringVarP(&title, "title", "t", "", "Event title (required)")
	cmd.Flags().IntVar(&capacity, "capacity", 10, "Maximum number of attendees (1-500)")
	cmd.Flags().StringVar(&description, "description", "", "Event description")
	cmd.Flags().StringVar(&location, "location", "", "Event location")
	cmd.Flags().StringVar(&startStr, "start", "", "Start time (YYYY-MM-DD HH:MM, RFC3339, or Unix) (required)")
	cmd.Flags().StringVar(&endStr, "end", "", "End time (YYYY-MM-DD HH:MM, RFC3339, or Unix) (required)")
	cmd.Flags().StringVar(&timezone, "timezone", "", "IANA timezone for start/end (e.g. America/New_York)")
	cmd.Flags().StringArrayVarP(&participants, "participant", "p", nil, "Participant as '[name:]email' (repeatable)")
	cmd.Flags().StringVar(&organizer, "organizer", "", "Organizer participant as '[name:]email'")

	return cmd
}

func newGroupEventUpdateCmd() *cobra.Command {
	var (
		title       string
		capacity    int
		description string
		location    string
	)

	cmd := &cobra.Command{
		Use:   "update <configuration-id> <event-id> [grant-id]",
		Short: "Update a group event",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			configID, eventID := args[0], args[1]

			req := &domain.UpdateGroupEventRequest{
				Title:       title,
				Capacity:    capacity,
				Description: description,
				Location:    location,
			}
			if title == "" && capacity == 0 && description == "" && location == "" {
				return common.NewUserError(
					"nothing to update",
					"Pass at least one of --title, --capacity, --description, or --location.",
				)
			}

			events, err := common.WithClient(args[2:], func(ctx context.Context, client ports.NylasClient, grantID string) ([]domain.GroupEvent, error) {
				return client.UpdateGroupEvent(ctx, grantID, configID, eventID, req)
			})
			if err != nil {
				return common.WrapUpdateError("group event", err)
			}
			return reportGroupEvents(cmd, events, "Group event updated")
		},
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "New event title")
	cmd.Flags().IntVar(&capacity, "capacity", 0, "New maximum number of attendees (1-500)")
	cmd.Flags().StringVar(&description, "description", "", "New event description")
	cmd.Flags().StringVar(&location, "location", "", "New event location")

	return cmd
}

func newGroupEventDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <configuration-id> <event-id> [grant-id]",
		Aliases: []string{"rm"},
		Short:   "Delete a group event",
		Args:    cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			configID, eventID := args[0], args[1]
			_, err := common.WithClient(args[2:], func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				if err := client.DeleteGroupEvent(ctx, grantID, configID, eventID); err != nil {
					return struct{}{}, common.WrapDeleteError("group event", err)
				}
				common.PrintSuccess("Group event deleted")
				return struct{}{}, nil
			})
			return err
		},
	}
	return cmd
}

func newGroupEventsImportCmd() *cobra.Command {
	var (
		file       string
		calendarID string
		eventID    string
		capacity   int
	)

	cmd := &cobra.Command{
		Use:   "import <configuration-id>",
		Short: "Import existing provider events as group events",
		Long: `Import one or more existing calendar events into a Configuration as
group events.

Provide a JSON array of import items with --file, or import a single event
inline with --calendar and --event. The JSON file format is an array of:
  [{"calendar_id": "...", "event_id": "...", "capacity": 50}]

This endpoint is configuration-scoped and does not take a grant.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configID := args[0]

			items, err := loadImportItems(file, calendarID, eventID, capacity)
			if err != nil {
				return err
			}

			events, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) ([]domain.GroupEvent, error) {
				return client.ImportGroupEvents(ctx, configID, items)
			})
			if err != nil {
				return common.WrapCreateError("group events", err)
			}
			return reportGroupEvents(cmd, events, fmt.Sprintf("Imported %d group event(s)", len(events)))
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to a JSON array of import items")
	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (single-event import)")
	cmd.Flags().StringVarP(&eventID, "event", "e", "", "Provider event ID to import (single-event import)")
	cmd.Flags().IntVar(&capacity, "capacity", 0, "Capacity for the imported event (single-event import)")

	return cmd
}

// loadImportItems builds the import payload from a JSON file or inline flags.
func loadImportItems(file, calendarID, eventID string, capacity int) ([]domain.ImportGroupEventItem, error) {
	if file != "" {
		data, err := os.ReadFile(file) //nolint:gosec // user-supplied path is expected
		if err != nil {
			return nil, common.NewUserError(fmt.Sprintf("could not read --file %q: %v", file, err), "Provide a readable JSON file.")
		}
		var items []domain.ImportGroupEventItem
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, common.NewUserError(
				fmt.Sprintf("invalid JSON in --file %q: %v", file, err),
				`Expected a JSON array like [{"calendar_id":"...","event_id":"..."}].`,
			)
		}
		if len(items) == 0 {
			return nil, common.NewUserError("no events to import", "The JSON array is empty.")
		}
		return items, nil
	}

	if calendarID == "" || eventID == "" {
		return nil, common.NewUserError(
			"no events to import",
			"Provide --file with a JSON array, or --calendar and --event for a single import.",
		)
	}
	return []domain.ImportGroupEventItem{
		{CalendarID: calendarID, EventID: eventID, Capacity: capacity},
	}, nil
}

// buildGroupParticipants converts repeatable "[name:]email" flags into
// participants, marking the organizer entry when provided.
func buildGroupParticipants(participants []string, organizer string) []domain.GroupEventParticipant {
	out := make([]domain.GroupEventParticipant, 0, len(participants)+1)
	for _, p := range participants {
		if pp, ok := parseGroupParticipant(p, false); ok {
			out = append(out, pp)
		}
	}
	if organizer != "" {
		if pp, ok := parseGroupParticipant(organizer, true); ok {
			out = append(out, pp)
		}
	}
	return out
}

// parseGroupParticipant parses "[name:]email" into a participant.
func parseGroupParticipant(s string, isOrganizer bool) (domain.GroupEventParticipant, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return domain.GroupEventParticipant{}, false
	}
	name, email := "", s
	if idx := strings.LastIndex(s, ":"); idx >= 0 {
		name = strings.TrimSpace(s[:idx])
		email = strings.TrimSpace(s[idx+1:])
	}
	if email == "" {
		return domain.GroupEventParticipant{}, false
	}
	return domain.GroupEventParticipant{Name: name, Email: email, IsOrganizer: isOrganizer}, true
}

// parseGroupEventTime parses a time bound into a Unix timestamp. Empty returns 0.
func parseGroupEventTime(name, value string) (int64, error) {
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
		"Use YYYY-MM-DD HH:MM, RFC3339, or a Unix timestamp.",
	)
}

func reportGroupEvents(cmd *cobra.Command, events []domain.GroupEvent, successMsg string) error {
	if common.IsStructuredOutput(cmd) {
		return common.GetOutputWriter(cmd).Write(events)
	}
	common.PrintSuccess(successMsg)
	for _, e := range events {
		printGroupEvent(e)
	}
	return nil
}

func printGroupEvent(e domain.GroupEvent) {
	title := e.Title
	if title == "" {
		title = "(no title)"
	}
	fmt.Printf("%s\n", common.Cyan.Sprint(title))
	if e.ID != "" {
		fmt.Printf("  %s %s\n", common.Dim.Sprint("ID:"), e.ID)
	}
	if e.Capacity > 0 {
		fmt.Printf("  %s %d\n", common.Dim.Sprint("Capacity:"), e.Capacity)
	}
	if e.When != nil && e.When.StartTime > 0 {
		fmt.Printf("  %s %s\n", common.Dim.Sprint("Start:"), time.Unix(e.When.StartTime, 0).Format(time.RFC1123))
	}
	if e.Location != "" {
		fmt.Printf("  %s %s\n", common.Dim.Sprint("Location:"), e.Location)
	}
	if len(e.Participants) > 0 {
		fmt.Printf("  %s %d\n", common.Dim.Sprint("Participants:"), len(e.Participants))
	}
	fmt.Println()
}
