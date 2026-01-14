package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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

			// Validate status
			if status != "yes" && status != "no" && status != "maybe" {
				return common.NewUserError(
					"invalid RSVP status",
					"Status must be 'yes', 'no', or 'maybe'",
				)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var grantID string
			if len(args) > 2 {
				grantID = args[2]
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

			req := &domain.SendRSVPRequest{
				Status:  status,
				Comment: comment,
			}

			err = common.RunWithSpinner("Sending RSVP...", func() error {
				return client.SendRSVP(ctx, grantID, calendarID, eventID, req)
			})
			if err != nil {
				return common.WrapSendError("RSVP", err)
			}

			statusText := map[string]string{
				"yes":   "accepted",
				"no":    "declined",
				"maybe": "tentatively accepted",
			}
			fmt.Printf("%s RSVP sent! You have %s the invitation.\n", common.Green.Sprint("✓"), statusText[status])

			return nil
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

func parseEventTime(startStr, endStr string, allDay bool) (*domain.EventWhen, error) {
	when := &domain.EventWhen{}

	// Try parsing as date first (YYYY-MM-DD)
	if allDay || len(startStr) <= 10 {
		startDate, err := time.Parse("2006-01-02", startStr)
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
		t, err := time.ParseInLocation(format, startStr, time.Local)
		if err == nil {
			startTime = t
			parsed = true
			break
		}
	}
	if !parsed {
		return nil, common.NewUserError(fmt.Sprintf("invalid start time format: %s", startStr), "use 'YYYY-MM-DD HH:MM' or 'YYYY-MM-DD'")
	}

	when.Object = "timespan"
	when.StartTime = startTime.Unix()

	if endStr != "" {
		var endTime time.Time
		for _, format := range formats {
			t, err := time.ParseInLocation(format, endStr, time.Local)
			if err == nil {
				endTime = t
				break
			}
		}
		if endTime.IsZero() {
			return nil, common.NewInputError(fmt.Sprintf("invalid end time format: %s", endStr))
		}
		when.EndTime = endTime.Unix()
	} else {
		// Default to 1 hour duration
		when.EndTime = startTime.Add(time.Hour).Unix()
	}

	return when, nil
}
