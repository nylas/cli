package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newEventsRSVPCmd() *cobra.Command {
	var (
		calendarID string
		comment    string
	)

	cmd := &cobra.Command{
		Use:   "rsvp <event-id> <status> [grant-id]",
		Short: "RSVP to an event invitation",
		Long: `Respond to an event invitation with your RSVP status.

Status options:
  - yes    Accept the invitation
  - no     Decline the invitation
  - maybe  Tentatively accept

Examples:
  # Accept an event invitation
  nylas calendar events rsvp <event-id> yes

  # Decline with a comment
  nylas calendar events rsvp <event-id> no --comment "I have a conflict"

  # Tentatively accept
  nylas calendar events rsvp <event-id> maybe`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID := args[0]
			status := strings.ToLower(args[1])
			grantArgs := args[2:]

			// Validate status
			if status != "yes" && status != "no" && status != "maybe" {
				return common.NewUserError(
					"invalid RSVP status",
					"Status must be 'yes', 'no', or 'maybe'",
				)
			}

			_, err := common.WithClient(grantArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Get calendar ID if not specified
				calID, err := GetDefaultCalendarID(ctx, client, grantID, calendarID, false)
				if err != nil {
					return struct{}{}, err
				}

				req := &domain.SendRSVPRequest{
					Status:  status,
					Comment: comment,
				}

				err = common.RunWithSpinner("Sending RSVP...", func() error {
					return client.SendRSVP(ctx, grantID, calID, eventID, req)
				})
				if err != nil {
					return struct{}{}, common.WrapSendError("RSVP", err)
				}

				statusText := map[string]string{
					"yes":   "accepted",
					"no":    "declined",
					"maybe": "tentatively accepted",
				}
				fmt.Printf("%s RSVP sent! You have %s the invitation.\n", common.Green.Sprint("✓"), statusText[status])

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&calendarID, "calendar", "c", "", "Calendar ID (defaults to primary)")
	cmd.Flags().StringVar(&comment, "comment", "", "Optional comment with your RSVP")

	return cmd
}

// Helper functions

func formatEventTime(when domain.EventWhen) string {
	if when.IsAllDay() {
		start := when.StartDateTime()
		end := when.EndDateTime()
		if start.Equal(end) || end.IsZero() {
			return start.Format("Mon, Jan 2, 2006") + " (all day)"
		}
		return fmt.Sprintf("%s - %s (all day)",
			start.Format("Mon, Jan 2, 2006"),
			end.Format("Mon, Jan 2, 2006"))
	}

	start := when.StartDateTime()
	end := when.EndDateTime()

	if start.Format("2006-01-02") == end.Format("2006-01-02") {
		// Same day
		return fmt.Sprintf("%s, %s - %s",
			start.Format("Mon, Jan 2, 2006"),
			start.Format("3:04 PM"),
			end.Format("3:04 PM"))
	}

	return fmt.Sprintf("%s - %s",
		start.Format("Mon, Jan 2, 2006 3:04 PM"),
		end.Format("Mon, Jan 2, 2006 3:04 PM"))
}

func formatParticipantStatus(status string) string {
	switch status {
	case "yes":
		return common.Green.Sprint("✓ accepted")
	case "no":
		return common.Red.Sprint("✗ declined")
	case "maybe":
		return common.Yellow.Sprint("? tentative")
	case "noreply":
		return common.Dim.Sprint("pending")
	default:
		return ""
	}
}

// parseEventTime parses start/end input into an EventWhen.
// Timed events are parsed in tz (an IANA timezone ID, defaulting to the system
// timezone when empty) and record it in StartTimezone/EndTimezone so the
// timestamps and zone always agree. All-day events take a date only.
func parseEventTime(startStr, endStr string, allDay bool, tz string) (*domain.EventWhen, error) {
	when := &domain.EventWhen{}

	// Try parsing as date first (YYYY-MM-DD)
	if allDay || len(startStr) <= 10 {
		startDate, err := time.Parse("2006-01-02", startStr)
		if err != nil && allDay {
			// Never fall through to a timed event when --all-day was requested
			return nil, common.NewUserError(
				fmt.Sprintf("invalid all-day start date: %s", startStr),
				"All-day events take a date only (YYYY-MM-DD). Remove --all-day to create a timed event.",
			)
		}
		if err == nil {
			when.Object = "date"
			when.Date = startDate.Format("2006-01-02")
			if endStr != "" {
				endDate, err := time.Parse("2006-01-02", endStr)
				if err != nil {
					return nil, common.NewUserError(fmt.Sprintf("invalid end date format: %s", endStr), "use YYYY-MM-DD")
				}
				if !endDate.Equal(startDate) {
					when.Object = "datespan"
					when.StartDate = when.Date
					when.Date = ""
					when.EndDate = endDate.Format("2006-01-02")
				}
			}
			return when, nil
		}
	}

	// Resolve the timezone for timed events: explicit value, else system zone
	if tz == "" {
		tz = getLocalTimeZone()
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, common.NewUserError(
			fmt.Sprintf("invalid timezone: %s", tz),
			"Use IANA timezone IDs like 'America/Los_Angeles'.\nRun 'nylas timezone list' to see available timezones.",
		)
	}

	// Try parsing as datetime
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}

	var startTime time.Time
	var parsed bool
	for _, format := range formats {
		t, err := time.ParseInLocation(format, startStr, loc)
		if err == nil {
			startTime = t
			parsed = true
			break
		}
	}
	if !parsed {
		return nil, common.NewUserError(fmt.Sprintf("invalid start time format: %s", startStr), "use 'YYYY-MM-DD HH:MM' or 'YYYY-MM-DD'")
	}
	if err := checkOffsetMatchesZone(startTime, loc, tz, "start"); err != nil {
		return nil, err
	}

	when.Object = "timespan"
	when.StartTime = startTime.Unix()
	when.StartTimezone = tz
	when.EndTimezone = tz

	if endStr != "" {
		var endTime time.Time
		for _, format := range formats {
			t, err := time.ParseInLocation(format, endStr, loc)
			if err == nil {
				endTime = t
				break
			}
		}
		if endTime.IsZero() {
			return nil, common.NewInputError(fmt.Sprintf("invalid end time format: %s", endStr))
		}
		if err := checkOffsetMatchesZone(endTime, loc, tz, "end"); err != nil {
			return nil, err
		}
		when.EndTime = endTime.Unix()
	} else {
		// Default to 1 hour duration
		when.EndTime = startTime.Add(time.Hour).Unix()
	}

	return when, nil
}

// checkOffsetMatchesZone rejects inputs whose explicit UTC offset (RFC3339)
// disagrees with the event timezone. ParseInLocation honors the input's
// offset over loc, so without this check the epoch would follow the offset
// while start_timezone/end_timezone record a different zone — the event
// would display at a different wall time than the user typed. Inputs without
// an offset parse in loc and always agree.
func checkOffsetMatchesZone(t time.Time, loc *time.Location, tz, field string) error {
	_, inputOffset := t.Zone()
	_, zoneOffset := t.In(loc).Zone()
	if inputOffset == zoneOffset {
		return nil
	}
	return common.NewUserError(
		fmt.Sprintf("%s time UTC offset %s does not match timezone %s (%s)",
			field, t.Format("-07:00"), tz, t.In(loc).Format("-07:00")),
		"Remove the offset from the input (e.g. 'YYYY-MM-DD HH:MM'), or pass a --timezone that matches it.",
	)
}
