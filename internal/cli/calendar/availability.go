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
			// Parse time range
			now := time.Now()
			var startTime, endTime time.Time
			var err error

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

			_, err = common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// If no emails specified, get the grant's email
				emailList := emails
				if len(emailList) == 0 {
					grant, err := client.GetGrant(ctx, grantID)
					if err != nil {
						return struct{}{}, common.NewUserError(fmt.Sprintf("Failed to get grant details: %v", err),
							"Run 'nylas auth status' to verify your authentication")
					}
					if grant.Email != "" {
						emailList = []string{grant.Email}
					} else {
						return struct{}{}, common.NewUserError("No email found for grant",
							"Please specify --emails flag with the email addresses to check")
					}
				}

				req := &domain.FreeBusyRequest{
					StartTime: startTime.Unix(),
					EndTime:   endTime.Unix(),
					Emails:    emailList,
				}

				result, err := common.RunWithSpinnerResult("Checking availability...", func() (*domain.FreeBusyResponse, error) {
					return client.GetFreeBusy(ctx, grantID, req)
				})
				if err != nil {
					return struct{}{}, common.NewUserError(fmt.Sprintf("Failed to get availability: %v", err),
						"Check that the email addresses are valid")
				}

				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(result)
				}
				return struct{}{}, displayFreeBusy(result, startTime, endTime)
			})
			return err
		},
	}

	cmd.Flags().StringSliceVarP(&emails, "emails", "e", nil, "Email addresses to check (comma-separated)")
	cmd.Flags().StringVarP(&start, "start", "s", "", "Start time (default: now)")
	cmd.Flags().StringVar(&end, "end", "", "End time")
	cmd.Flags().StringVarP(&duration, "duration", "d", "", "Duration from start (e.g., '8h', '1d', '7d')")

	return cmd
}

func newFindSlotsCmd() *cobra.Command {
	var (
		participants []string
		start        string
		end          string
		durationMins int
		intervalMins int
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

			// Parse time range
			now := time.Now()
			var startTime, endTime time.Time
			var err error

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

			_, err = common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.AvailabilityRequest{
					StartTime:       startTime.Unix(),
					EndTime:         endTime.Unix(),
					DurationMinutes: durationMins,
					Participants:    availParticipants,
					IntervalMinutes: intervalMins,
				}

				result, err := common.RunWithSpinnerResult("Finding available times...", func() (*domain.AvailabilityResponse, error) {
					return client.GetAvailability(ctx, req)
				})
				if err != nil {
					return struct{}{}, common.NewUserError(fmt.Sprintf("Failed to find availability: %v", err),
						"Check that the participant email addresses are valid")
				}

				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(result)
				}
				return struct{}{}, displayAvailableSlots(result, durationMins)
			})
			return err
		},
	}

	cmd.Flags().StringSliceVarP(&participants, "participants", "p", nil, "Participant email addresses (required)")
	cmd.Flags().StringVarP(&start, "start", "s", "", "Start time for search (default: next hour)")
	cmd.Flags().StringVar(&end, "end", "", "End time for search (default: 7 days from start)")
	cmd.Flags().IntVarP(&durationMins, "duration", "d", 30, "Meeting duration in minutes")
	cmd.Flags().IntVarP(&intervalMins, "interval", "i", 15, "Search interval in minutes")

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
	return common.ParseHumanTime(input, common.ParseHumanTimeOpts{})
}
