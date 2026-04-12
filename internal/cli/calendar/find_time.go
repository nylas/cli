package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/utilities/scheduling"
	timezonesvc "github.com/nylas/cli/internal/adapters/utilities/timezone"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newFindTimeCmd() *cobra.Command {
	var (
		participants    []string
		participantTZs  []string
		duration        string
		workingStart    string
		workingEnd      string
		days            int
		excludeWeekends bool
	)

	cmd := &cobra.Command{
		Use:   "find-time",
		Short: "Find optimal meeting times across multiple timezones",
		Long: `Find optimal meeting times across multiple timezones.

Analyzes participant timezones and suggests meeting times with a 100-point scoring algorithm:
- Working Hours (40 pts): All participants within working hours
- Time Quality (25 pts): Quality of time for participants (morning/afternoon)
- Cultural (15 pts): Respects cultural norms (no Friday PM, no lunch hour)
- Weekday (10 pts): Prefers mid-week meetings
- Holiday (10 pts): Avoids holidays`,
		Example: `  # Find time for 2 participants
  nylas calendar find-time --participants alice@example.com,bob@example.com --duration 1h

  # Custom working hours and date range
  nylas calendar find-time \
    --participants alice@example.com,bob@example.com \
    --duration 1h \
    --working-start 09:00 \
    --working-end 17:00 \
    --days 7`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(participants) < 2 {
				return common.NewUserError(
					"at least 2 participants required",
					"Specify participants with --participants alice@example.com,bob@example.com",
				)
			}

			// Validate participant emails are not empty
			for i, email := range participants {
				email = strings.TrimSpace(email)
				if email == "" {
					return common.NewUserError(
						fmt.Sprintf("participant email at position %d cannot be empty", i+1),
						"Ensure all participant emails are valid",
					)
				}
				participants[i] = email
			}

			// Parse duration
			dur, err := common.ParseDuration(duration)
			if err != nil {
				return common.NewUserError(
					fmt.Sprintf("invalid duration: %s", duration),
					"Use formats like: 30m, 1h, 1h30m, 7d",
				)
			}
			workStart, err := parseWorkingTime(workingStart)
			if err != nil {
				return common.NewUserError(
					fmt.Sprintf("invalid working hours start: %s", workingStart),
					"Use format HH:MM (e.g., 09:00)",
				)
			}
			workEnd, err := parseWorkingTime(workingEnd)
			if err != nil {
				return common.NewUserError(
					fmt.Sprintf("invalid working hours end: %s", workingEnd),
					"Use format HH:MM (e.g., 17:00)",
				)
			}

			timezones, usedFallback, err := resolveParticipantTimezones(participants, participantTZs)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			// Find overlapping times
			slots, err := findMeetingSlots(ctx, timezones, dur, workingStart, workingEnd, days, excludeWeekends)
			if err != nil {
				return err
			}

			// Display results
			displayFindTimeResults(participants, timezones, slots, usedFallback, workStart, workEnd)

			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&participants, "participants", "p", nil, "Participant email addresses (comma-separated)")
	cmd.Flags().StringSliceVar(&participantTZs, "timezones", nil, "Participant IANA timezones, aligned with --participants order")
	cmd.Flags().StringVarP(&duration, "duration", "d", "1h", "Meeting duration (e.g., 30m, 1h, 1h30m)")
	cmd.Flags().StringVar(&workingStart, "working-start", "09:00", "Working hours start (HH:MM)")
	cmd.Flags().StringVar(&workingEnd, "working-end", "17:00", "Working hours end (HH:MM)")
	cmd.Flags().IntVar(&days, "days", 7, "Number of days to search")
	cmd.Flags().BoolVar(&excludeWeekends, "exclude-weekends", true, "Exclude weekends from search")

	_ = cmd.MarkFlagRequired("participants")

	return cmd
}

// findMeetingSlots finds overlapping meeting times across timezones.
func findMeetingSlots(
	ctx context.Context,
	timezones []string,
	duration time.Duration,
	workingStart, workingEnd string,
	days int,
	excludeWeekends bool,
) ([]scheduling.TimeSlot, error) {
	locations := make([]*time.Location, len(timezones))
	for i, tz := range timezones {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			return nil, common.NewUserError(
				fmt.Sprintf("invalid timezone: %s", tz),
				"Use IANA timezone IDs like 'America/Los_Angeles'",
			)
		}
		locations[i] = loc
	}

	workStart, err := parseWorkingTime(workingStart)
	if err != nil {
		return nil, common.NewUserError(
			fmt.Sprintf("invalid working hours start: %s", workingStart),
			"Use format HH:MM (e.g., 09:00)",
		)
	}

	workEnd, err := parseWorkingTime(workingEnd)
	if err != nil {
		return nil, common.NewUserError(
			fmt.Sprintf("invalid working hours end: %s", workingEnd),
			"Use format HH:MM (e.g., 17:00)",
		)
	}

	now := time.Now()
	startDate := now.Add(24 * time.Hour)
	endDate := startDate.AddDate(0, 0, max(days-1, 0))

	result, err := timezonesvc.NewService().FindMeetingTime(ctx, &domain.MeetingFinderRequest{
		TimeZones:         timezones,
		Duration:          duration,
		WorkingHoursStart: workingStart,
		WorkingHoursEnd:   workingEnd,
		DateRange: domain.DateRange{
			Start: startDate,
			End:   endDate,
		},
		ExcludeWeekends: excludeWeekends,
	})
	if err != nil {
		return nil, fmt.Errorf("find meeting slots: %w", err)
	}

	slots := make([]scheduling.TimeSlot, 0, len(result.Slots))
	for _, slot := range result.Slots {
		participants := make([]scheduling.ParticipantTime, len(timezones))
		for i, loc := range locations {
			localTime := slot.StartTime.In(loc)
			localMinutes := localTime.Hour()*60 + localTime.Minute()
			isWorking := localMinutes >= workStart && localMinutes < workEnd
			quality, icon := scheduling.GetQualityLabel(localTime, isWorking)
			participants[i] = scheduling.ParticipantTime{
				TimeZone:    timezones[i],
				LocalTime:   localTime,
				IsWorking:   isWorking,
				Quality:     quality,
				QualityIcon: icon,
			}
		}

		breakdown := scheduling.ScoreTimeSlot(slot.StartTime, slot.EndTime, participants)
		slots = append(slots, scheduling.TimeSlot{
			StartTime: slot.StartTime,
			EndTime:   slot.EndTime,
			Score:     breakdown.Total,
			Breakdown: breakdown,
		})
	}

	if len(slots) > 5 {
		slots = slots[:5]
	}

	return slots, nil
}

// displayFindTimeResults displays the found meeting times.
func displayFindTimeResults(participants []string, timezones []string, slots []scheduling.TimeSlot, usedFallback bool, workStart, workEnd int) {
	fmt.Println("\n🌍 Multi-Timezone Meeting Finder")
	fmt.Println()

	// Show participants
	fmt.Println("Participants:")
	for i, email := range participants {
		if i < len(timezones) {
			fmt.Printf("  • %s: %s\n", email, timezones[i])
		}
	}
	if usedFallback {
		fmt.Printf("\nℹ️  No participant timezones were provided. Using %s for all participants.\n", timezones[0])
		fmt.Println("    Pass --timezones with one IANA timezone per participant for accurate local-time views.")
	}
	fmt.Println()

	// Show top suggestions
	if len(slots) == 0 {
		fmt.Println("❌ No suitable meeting times found")
		fmt.Println("Try expanding the date range or adjusting working hours")
		return
	}

	fmt.Printf("Top %d Suggested Times:\n\n", len(slots))

	for i, slot := range slots {
		color := scheduling.GetScoreColor(slot.Score)
		fmt.Printf("%d. %s %s (Score: %.0f/100)\n",
			i+1,
			color,
			slot.StartTime.Format("Monday, Jan 2, 3:04 PM MST"),
			slot.Score,
		)

		// Show time for each participant
		for j, tz := range timezones {
			loc, _ := time.LoadLocation(tz)
			localTime := slot.StartTime.In(loc)
			endTime := slot.EndTime.In(loc)

			var quality, icon string
			localMinutes := localTime.Hour()*60 + localTime.Minute()
			isWorking := localMinutes >= workStart && localMinutes < workEnd
			quality, icon = scheduling.GetQualityLabel(localTime, isWorking)

			email := "Participant"
			if j < len(participants) {
				parts := strings.Split(participants[j], "@")
				email = parts[0]
			}

			fmt.Printf("   %s: %s - %s %s (%s %s)\n",
				email,
				localTime.Format("3:04 PM"),
				endTime.Format("3:04 PM MST"),
				tz,
				quality,
				icon,
			)
		}

		// Show score breakdown
		fmt.Println()
		fmt.Println("   Score Breakdown:")
		fmt.Printf("   • Working Hours: %.0f/40 (%s)\n",
			slot.Breakdown.WorkingHours,
			getCheckMark(slot.Breakdown.WorkingHours >= 40),
		)
		fmt.Printf("   • Time Quality: %.0f/25\n", slot.Breakdown.TimeQuality)
		fmt.Printf("   • Cultural: %.0f/15\n", slot.Breakdown.Cultural)
		fmt.Printf("   • Weekday: %.0f/10\n", slot.Breakdown.Weekday)
		fmt.Printf("   • Holidays: %.0f/10\n", slot.Breakdown.Holiday)
		fmt.Println()
	}

	if len(slots) > 0 {
		fmt.Println("💡 Recommendation: Book option #1 for best overall experience")
	}
}

func resolveParticipantTimezones(participants, provided []string) ([]string, bool, error) {
	if len(provided) > 0 && len(provided) != len(participants) {
		return nil, false, common.NewUserError(
			fmt.Sprintf("got %d timezones for %d participants", len(provided), len(participants)),
			"Provide one timezone per participant, in the same order as --participants",
		)
	}

	if len(provided) == 0 {
		localTZ := getLocalTimeZone()
		timezones := make([]string, len(participants))
		for i := range participants {
			timezones[i] = localTZ
		}
		return timezones, true, nil
	}

	timezones := make([]string, len(provided))
	for i, tz := range provided {
		tz = strings.TrimSpace(tz)
		if err := validateTimeZone(tz); err != nil {
			return nil, false, err
		}
		timezones[i] = tz
	}

	return timezones, false, nil
}

// getCheckMark returns a checkmark or x based on condition.
func getCheckMark(condition bool) string {
	if condition {
		return "✓"
	}
	return "✗"
}

// parseWorkingTime parses a working time in HH:MM format.
func parseWorkingTime(s string) (int, error) {
	var hour, minute int
	_, err := fmt.Sscanf(s, "%d:%d", &hour, &minute)
	if err != nil {
		return 0, err
	}
	if hour < 0 || hour > 23 {
		return 0, fmt.Errorf("hour must be 0-23, got %d from input %q", hour, s)
	}
	if minute < 0 || minute > 59 {
		return 0, fmt.Errorf("minute must be 0-59, got %d from input %q", minute, s)
	}
	return hour*60 + minute, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
