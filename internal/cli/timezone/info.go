package timezone

import (
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var (
		zone    string
		timeStr string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "info [ZONE]",
		Short: "Get detailed information about a time zone",
		Long: `Display comprehensive information about a specific time zone including
current time, offset, DST status, and upcoming transitions.

Examples:
  # Get info for New York
  nylas timezone info America/New_York

  # Using the zone flag
  nylas timezone info --zone America/Los_Angeles

  # Check info at a specific time
  nylas timezone info --zone Europe/London --time "2026-06-01T12:00:00Z"

  # Use abbreviations
  nylas timezone info PST

  # Output as JSON
  nylas timezone info --zone IST --json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Allow zone as positional argument or flag
			if len(args) > 0 && zone == "" {
				zone = args[0]
			}

			if zone == "" {
				return fmt.Errorf("time zone required (use argument or --zone flag)")
			}

			return runInfo(zone, timeStr, jsonOut)
		},
	}

	cmd.Flags().StringVar(&zone, "zone", "", "Time zone (IANA name or abbreviation)")
	cmd.Flags().StringVar(&timeStr, "time", "", "Time to check (RFC3339 format, defaults to now)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

func runInfo(zone, timeStr string, jsonOut bool) error {
	// Normalize time zone name
	originalZone := zone
	zone = normalizeTimeZone(zone)

	// Parse time or use current time
	var checkTime time.Time
	var err error

	if timeStr == "" {
		checkTime = time.Now()
	} else {
		checkTime, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return common.NewUserError("invalid time format", "use RFC3339")
		}
	}

	// Get time zone info
	svc := getService()
	ctx, cancel := common.CreateContext()
	defer cancel()

	info, err := svc.GetTimeZoneInfo(ctx, zone, checkTime)
	if err != nil {
		return common.WrapGetError("time zone info", err)
	}

	// Get current time in the zone
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return fmt.Errorf("load location: %w", err)
	}

	localTime := checkTime.In(loc)

	// Output
	if jsonOut {
		return common.PrintJSON(map[string]any{
			"zone":           info.Name,
			"abbreviation":   info.Abbreviation,
			"offset":         formatOffset(info.Offset),
			"offset_seconds": info.Offset,
			"is_dst":         info.IsDST,
			"local_time":     localTime.Format(time.RFC3339),
			"next_dst":       info.NextDST,
		})
	}

	// Human-readable output
	fmt.Printf("Time Zone Information\n\n")

	// Show if abbreviation was expanded
	if originalZone != zone {
		fmt.Printf("Zone: %s (expanded from '%s')\n", zone, originalZone)
	} else {
		fmt.Printf("Zone: %s\n", zone)
	}

	fmt.Printf("Abbreviation: %s\n", info.Abbreviation)
	fmt.Printf("Current Time: %s\n", formatTime(localTime, true))
	fmt.Printf("UTC Offset: %s (%d seconds)\n", formatOffset(info.Offset), info.Offset)

	if info.IsDST {
		fmt.Printf("DST Status: ✓ Currently observing Daylight Saving Time\n")
	} else {
		fmt.Printf("DST Status: ✗ Currently on Standard Time\n")
	}

	// Show next DST transition if available
	if info.NextDST != nil {
		daysUntil := int(info.NextDST.Sub(checkTime).Hours() / 24)

		fmt.Printf("\nNext DST Transition:\n")
		fmt.Printf("  Date: %s\n", info.NextDST.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("  Days Until: %d\n", daysUntil)

		if info.IsDST {
			fmt.Printf("  Change: Fall Back (DST ends, gain 1 hour)\n")
		} else {
			fmt.Printf("  Change: Spring Forward (DST begins, lose 1 hour)\n")
		}

		// Warning for upcoming transitions
		if daysUntil <= 30 && daysUntil > 0 {
			fmt.Printf("\n⚠️  WARNING: DST transition in %d days\n", daysUntil)
		}
	} else {
		fmt.Printf("\nNext DST Transition: None found in next 365 days\n")
		fmt.Printf("  (This zone may not observe DST)\n")
	}

	// Show comparison with UTC
	fmt.Printf("\nUTC Comparison:\n")
	utcTime := checkTime.UTC()
	fmt.Printf("  UTC Time: %s\n", formatTime(utcTime, true))

	hoursDiff := info.Offset / 3600
	if hoursDiff > 0 {
		fmt.Printf("  Difference: %d hour(s) ahead of UTC\n", hoursDiff)
	} else if hoursDiff < 0 {
		fmt.Printf("  Difference: %d hour(s) behind UTC\n", -hoursDiff)
	} else {
		fmt.Printf("  Difference: Same as UTC\n")
	}

	return nil
}
