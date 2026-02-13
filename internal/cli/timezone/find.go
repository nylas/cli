package timezone

import (
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func newFindMeetingCmd() *cobra.Command {
	var (
		zonesStr        string
		durationStr     string
		startHour       string
		endHour         string
		startDateStr    string
		endDateStr      string
		excludeWeekends bool
		jsonOut         bool
	)

	cmd := &cobra.Command{
		Use:   "find-meeting",
		Short: "Find overlapping meeting times across time zones",
		Long: `Find time slots where working hours overlap across multiple time zones.
This helps global teams schedule meetings at times that work for everyone.

The command finds times when all specified time zones are within their
working hours. Results are sorted by "quality score" based on how
convenient the time is for all participants.

Examples:
  # Find 1-hour meeting slot across 3 time zones
  nylas timezone find-meeting \
    --zones "America/New_York,Europe/London,Asia/Tokyo" \
    --duration 1h

  # Custom working hours (9 AM to 5 PM)
  nylas timezone find-meeting \
    --zones "PST,EST,IST" \
    --duration 30m \
    --start-hour 09:00 \
    --end-hour 17:00

  # Search specific date range
  nylas timezone find-meeting \
    --zones "America/Los_Angeles,Europe/Paris" \
    --duration 1h \
    --start-date 2026-01-15 \
    --end-date 2026-01-22

  # Exclude weekends
  nylas timezone find-meeting \
    --zones "PST,CST,EST" \
    --duration 1h \
    --exclude-weekends`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFindMeeting(zonesStr, durationStr, startHour, endHour,
				startDateStr, endDateStr, excludeWeekends, jsonOut)
		},
	}

	cmd.Flags().StringVar(&zonesStr, "zones", "", "Comma-separated list of time zones")
	cmd.Flags().StringVar(&durationStr, "duration", "1h", "Meeting duration (e.g., 30m, 1h, 1h30m)")
	cmd.Flags().StringVar(&startHour, "start-hour", "09:00", "Working hours start (HH:MM)")
	cmd.Flags().StringVar(&endHour, "end-hour", "17:00", "Working hours end (HH:MM)")
	cmd.Flags().StringVar(&startDateStr, "start-date", "", "Start date for search (YYYY-MM-DD, defaults to today)")
	cmd.Flags().StringVar(&endDateStr, "end-date", "", "End date for search (YYYY-MM-DD, defaults to 7 days from start)")
	cmd.Flags().BoolVar(&excludeWeekends, "exclude-weekends", false, "Exclude Saturday and Sunday")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	_ = cmd.MarkFlagRequired("zones")

	return cmd
}

func runFindMeeting(zonesStr, durationStr, startHour, endHour,
	startDateStr, endDateStr string, excludeWeekends, jsonOut bool) error {

	// Parse time zones
	zones := parseTimeZones(zonesStr)
	if len(zones) == 0 {
		return fmt.Errorf("at least one time zone required")
	}

	// Normalize zone names
	for i, zone := range zones {
		zones[i] = normalizeTimeZone(zone)
	}

	// Parse duration
	duration, err := common.ParseDuration(durationStr)
	if err != nil {
		return err
	}

	// Parse working hours
	start, end, err := parseWorkingHours(startHour, endHour)
	if err != nil {
		return err
	}

	// Parse date range
	var dateRange domain.DateRange
	if startDateStr == "" {
		dateRange.Start = time.Now()
	} else {
		dateRange.Start, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			return common.WrapDateParseError("start date", err)
		}
	}

	if endDateStr == "" {
		dateRange.End = dateRange.Start.AddDate(0, 0, 7) // 7 days from start
	} else {
		dateRange.End, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return common.WrapDateParseError("end date", err)
		}
	}

	// Create request
	req := &domain.MeetingFinderRequest{
		TimeZones:         zones,
		Duration:          duration,
		WorkingHoursStart: start,
		WorkingHoursEnd:   end,
		DateRange:         dateRange,
		ExcludeWeekends:   excludeWeekends,
	}

	// Find meeting times
	svc := getService()
	ctx, cancel := common.CreateContext()
	defer cancel()

	result, err := svc.FindMeetingTime(ctx, req)
	if err != nil {
		return fmt.Errorf("find meeting time: %w", err)
	}

	// Output
	if jsonOut {
		return common.PrintJSON(result)
	}

	// Human-readable output
	fmt.Printf("Meeting Time Finder\n\n")
	fmt.Printf("Time Zones: %s\n", zonesStr)
	fmt.Printf("Duration: %s\n", duration)
	fmt.Printf("Working Hours: %s - %s\n", start, end)
	fmt.Printf("Date Range: %s to %s\n",
		dateRange.Start.Format("2006-01-02"),
		dateRange.End.Format("2006-01-02"))
	if excludeWeekends {
		fmt.Printf("Excluding: Weekends\n")
	}
	fmt.Println()

	if len(result.Slots) == 0 {
		fmt.Println("❌ No overlapping time slots found")
		fmt.Println("\nSuggestions:")
		fmt.Println("  • Try a longer date range (--start-date and --end-date)")
		fmt.Println("  • Expand working hours (--start-hour and --end-hour)")
		fmt.Println("  • Reduce meeting duration (--duration)")
		return nil
	}

	fmt.Printf("✅ Found %d potential time slot(s):\n\n", len(result.Slots))

	// NOTE: The service implementation currently returns empty slots
	// This output formatting is ready for when the logic is implemented

	fmt.Println("⚠️  NOTE: Meeting time finder logic is not yet fully implemented.")
	fmt.Println("          The service will return available slots once the algorithm is complete.")
	fmt.Println("\nPlanned features:")
	fmt.Println("  • Identify overlapping working hours across all zones")
	fmt.Println("  • Calculate quality scores (middle of day = higher score)")
	fmt.Println("  • Filter by meeting duration")
	fmt.Println("  • Respect weekend exclusions")

	return nil
}
