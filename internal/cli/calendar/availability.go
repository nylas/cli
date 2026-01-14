package calendar

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newAvailabilityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "availability",
		Aliases: []string{"avail", "freebusy"},
		Short:   "Check calendar availability",
		Long: `Check calendar availability and find free meeting times.

Use 'nylas calendar availability check' to see free/busy times for calendars.
Use 'nylas calendar availability find' to find available meeting slots.`,
	}

	cmd.AddCommand(newFreeBusyCmd())
	cmd.AddCommand(newFindSlotsCmd())

	return cmd
}

func newFreeBusyCmd() *cobra.Command {
	var (
		emails   []string
		start    string
		end      string
		duration string
		format   string
	)

	cmd := &cobra.Command{
		Use:   "check [grant-id]",
		Short: "Check free/busy status for calendars",
		Long: `Check free/busy status for one or more email addresses.

Shows busy time slots within the specified time range.`,
		Example: `  # Check your own availability for the next 24 hours
  nylas calendar availability check

  # Check availability for multiple people
  nylas calendar availability check --emails alice@example.com,bob@example.com

  # Check availability for a specific time range
  nylas calendar availability check --start "tomorrow 9am" --end "tomorrow 5pm"

  # Check availability for next week
  nylas calendar availability check --duration 7d`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := getClient()
			if err != nil {
				return common.NewUserError("Failed to initialize client: "+err.Error(),
					"Run 'nylas auth login' to authenticate")
			}

			grantID, err := getGrantID(args)
			if err != nil {
				return common.NewUserError("Failed to get grant: "+err.Error(),
					"Run 'nylas auth status' to check authentication")
			}

			// Parse time range
			now := time.Now()
			var startTime, endTime time.Time

			if start != "" {
				startTime, err = parseTimeInput(start)
				if err != nil {
					return common.WrapDateParseError("start", err)
				}
			} else {
				startTime = now
			}

			if end != "" {
				endTime, err = parseTimeInput(end)
				if err != nil {
					return common.WrapDateParseError("end", err)
				}
			} else if duration != "" {
				dur, err := common.ParseDuration(duration)
				if err != nil {
					return common.WrapDateParseError("duration", err)
				}
				endTime = startTime.Add(dur)
			} else {
				// Default to 24 hours
				endTime = startTime.Add(24 * time.Hour)
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// If no emails specified, get the grant's email
			if len(emails) == 0 {
				grant, err := c.GetGrant(ctx, grantID)
				if err != nil {
					return common.NewUserError("Failed to get grant details: "+err.Error(),
						"Run 'nylas auth status' to verify your authentication")
				}
				if grant.Email != "" {
					emails = []string{grant.Email}
				} else {
					return common.NewUserError("No email found for grant",
						"Please specify --emails flag with the email addresses to check")
				}
			}

			req := &domain.FreeBusyRequest{
				StartTime: startTime.Unix(),
				EndTime:   endTime.Unix(),
				Emails:    emails,
			}

			result, err := common.RunWithSpinnerResult("Checking availability...", func() (*domain.FreeBusyResponse, error) {
				return c.GetFreeBusy(ctx, grantID, req)
			})
			if err != nil {
				return common.NewUserError("Failed to get availability: "+err.Error(),
					"Check that the email addresses are valid")
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			case "yaml":
				return yaml.NewEncoder(os.Stdout).Encode(result)
			default:
				return displayFreeBusy(result, startTime, endTime)
			}
		},
	}

	cmd.Flags().StringSliceVarP(&emails, "emails", "e", nil, "Email addresses to check (comma-separated)")
	cmd.Flags().StringVarP(&start, "start", "s", "", "Start time (default: now)")
	cmd.Flags().StringVar(&end, "end", "", "End time")
	cmd.Flags().StringVarP(&duration, "duration", "d", "", "Duration from start (e.g., '8h', '1d', '7d')")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")

	return cmd
}

func newFindSlotsCmd() *cobra.Command {
	var (
		participants []string
		start        string
		end          string
		durationMins int
		intervalMins int
		format       string
	)

	cmd := &cobra.Command{
		Use:   "find",
		Short: "Find available meeting times",
		Long: `Find available meeting times across multiple participants.

This searches for time slots when all participants are free.`,
		Example: `  # Find 30-minute meeting slots with participants
  nylas calendar availability find --participants alice@example.com,bob@example.com --duration 30

  # Find 1-hour meeting slots for tomorrow
  nylas calendar availability find --participants alice@example.com --duration 60 --start "tomorrow 9am" --end "tomorrow 5pm"

  # Find slots in 15-minute intervals
  nylas calendar availability find --participants team@example.com --duration 30 --interval 15`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(participants) == 0 {
				return common.NewUserError("At least one participant is required",
					"Use --participants to specify email addresses")
			}

			c, err := getClient()
			if err != nil {
				return common.NewUserError("Failed to initialize client: "+err.Error(),
					"Run 'nylas auth login' to authenticate")
			}

			// Parse time range
			now := time.Now()
			var startTime, endTime time.Time

			if start != "" {
				startTime, err = parseTimeInput(start)
				if err != nil {
					return common.WrapDateParseError("start", err)
				}
			} else {
				// Default to next business hour
				startTime = now.Add(time.Hour).Truncate(time.Hour)
			}

			if end != "" {
				endTime, err = parseTimeInput(end)
				if err != nil {
					return common.WrapDateParseError("end", err)
				}
			} else {
				// Default to 7 days from start
				endTime = startTime.AddDate(0, 0, 7)
			}

			// Build participant list
			availParticipants := make([]domain.AvailabilityParticipant, len(participants))
			for i, email := range participants {
				availParticipants[i] = domain.AvailabilityParticipant{
					Email: strings.TrimSpace(email),
				}
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			req := &domain.AvailabilityRequest{
				StartTime:       startTime.Unix(),
				EndTime:         endTime.Unix(),
				DurationMinutes: durationMins,
				Participants:    availParticipants,
				IntervalMinutes: intervalMins,
			}

			result, err := common.RunWithSpinnerResult("Finding available times...", func() (*domain.AvailabilityResponse, error) {
				return c.GetAvailability(ctx, req)
			})
			if err != nil {
				return common.NewUserError("Failed to find availability: "+err.Error(),
					"Check that the participant email addresses are valid")
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			case "yaml":
				return yaml.NewEncoder(os.Stdout).Encode(result)
			default:
				return displayAvailableSlots(result, durationMins)
			}
		},
	}

	cmd.Flags().StringSliceVarP(&participants, "participants", "p", nil, "Participant email addresses (required)")
	cmd.Flags().StringVarP(&start, "start", "s", "", "Start time for search (default: next hour)")
	cmd.Flags().StringVar(&end, "end", "", "End time for search (default: 7 days from start)")
	cmd.Flags().IntVarP(&durationMins, "duration", "d", 30, "Meeting duration in minutes")
	cmd.Flags().IntVarP(&intervalMins, "interval", "i", 15, "Search interval in minutes")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json, yaml)")

	_ = cmd.MarkFlagRequired("participants")

	return cmd
}

func displayFreeBusy(result *domain.FreeBusyResponse, startTime, endTime time.Time) error {
	fmt.Printf("Free/Busy Status: %s - %s\n",
		startTime.Format("Mon Jan 2 3:04 PM"),
		endTime.Format("Mon Jan 2 3:04 PM"))
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()

	for _, cal := range result.Data {
		fmt.Printf("📧 %s\n", cal.Email)

		if len(cal.TimeSlots) == 0 {
			fmt.Printf("   %s\n", common.Green.Sprint("✓ Free during this period"))
		} else {
			fmt.Println("   Busy times:")
			for _, slot := range cal.TimeSlots {
				start := time.Unix(slot.StartTime, 0)
				end := time.Unix(slot.EndTime, 0)

				if start.Day() == end.Day() {
					fmt.Printf("   %s %s - %s\n",
						common.Red.Sprint("●"),
						start.Format("Mon Jan 2 3:04 PM"),
						end.Format("3:04 PM"))
				} else {
					fmt.Printf("   %s %s - %s\n",
						common.Red.Sprint("●"),
						start.Format("Mon Jan 2 3:04 PM"),
						end.Format("Mon Jan 2 3:04 PM"))
				}
			}
		}
		fmt.Println()
	}

	return nil
}

func displayAvailableSlots(result *domain.AvailabilityResponse, durationMins int) error {
	if len(result.Data.TimeSlots) == 0 {
		common.PrintEmptyStateWithHint("available time slots", "Try expanding the search range or reducing the meeting duration")
		return nil
	}

	fmt.Printf("Available %d-minute Meeting Slots\n", durationMins)
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	// Group by day
	currentDay := ""
	for i, slot := range result.Data.TimeSlots {
		start := time.Unix(slot.StartTime, 0)
		end := time.Unix(slot.EndTime, 0)

		day := start.Format("Mon, Jan 2")
		if day != currentDay {
			if currentDay != "" {
				fmt.Println()
			}
			fmt.Printf("📅 %s\n", day)
			currentDay = day
		}

		fmt.Printf("   %d. %s - %s\n", i+1,
			start.Format("3:04 PM"),
			end.Format("3:04 PM"))
	}

	fmt.Printf("\nFound %d available slots\n", len(result.Data.TimeSlots))
	return nil
}

func parseTimeInput(input string) (time.Time, error) {
	now := time.Now()
	input = strings.TrimSpace(input)
	lower := strings.ToLower(input)

	// Handle "tomorrow" keyword
	if strings.HasPrefix(lower, "tomorrow") {
		tomorrow := now.AddDate(0, 0, 1)
		rest := strings.TrimPrefix(lower, "tomorrow")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, now.Location()), nil
		}
		if t, err := common.ParseTimeOfDay(rest); err == nil {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), nil
		}
	}

	// Handle "today" keyword
	if strings.HasPrefix(lower, "today") {
		rest := strings.TrimPrefix(lower, "today")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return now, nil
		}
		if t, err := common.ParseTimeOfDay(rest); err == nil {
			return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), nil
		}
	}

	// Try standard formats (most specific first)
	formats := []string{
		time.RFC3339,           // "2006-01-02T15:04:05Z07:00"
		"2006-01-02T15:04:05Z", // ISO8601 UTC
		"2006-01-02T15:04:05",  // ISO8601 with seconds
		"2006-01-02T15:04",     // ISO8601 without seconds
		"2006-01-02 15:04:05",  // Space separator with seconds
		"2006-01-02 15:04",     // Space separator
		"2006-01-02 3:04pm",    // 12-hour format
		"Jan 2 15:04",          // Month name
		"Jan 2 3:04pm",         // Month name 12-hour
	}

	for _, format := range formats {
		if t, err := time.ParseInLocation(format, input, now.Location()); err == nil {
			if t.Year() == 0 {
				t = time.Date(now.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
			}
			return t, nil
		}
	}

	// Try just time of day
	if t, err := common.ParseTimeOfDay(lower); err == nil {
		result := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if result.Before(now) {
			result = result.AddDate(0, 0, 1)
		}
		return result, nil
	}

	return time.Time{}, fmt.Errorf("could not parse time: %s", input)
}
