package timezone

import (
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newConvertCmd() *cobra.Command {
	var (
		fromZone string
		toZone   string
		timeStr  string
		jsonOut  bool
	)

	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert time between time zones",
		Long: `Convert a specific time (or current time) from one time zone to another.

Supports IANA time zone names (e.g., America/New_York, Europe/London, Asia/Tokyo)
and common abbreviations (PST, EST, IST, etc.).

Examples:
  # Convert current time from PST to IST
  nylas timezone convert --from America/Los_Angeles --to Asia/Kolkata

  # Convert specific time
  nylas timezone convert --from UTC --to America/New_York --time "2025-01-01T12:00:00Z"

  # Use abbreviations
  nylas timezone convert --from PST --to EST

  # Output as JSON
  nylas timezone convert --from PST --to IST --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConvert(fromZone, toZone, timeStr, jsonOut)
		},
	}

	cmd.Flags().StringVar(&fromZone, "from", "", "Source time zone (IANA name or abbreviation)")
	cmd.Flags().StringVar(&toZone, "to", "", "Target time zone (IANA name or abbreviation)")
	cmd.Flags().StringVar(&timeStr, "time", "", "Time to convert (RFC3339 format, defaults to now)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runConvert(fromZone, toZone, timeStr string, jsonOut bool) error {
	// Normalize time zone names
	fromZone = normalizeTimeZone(fromZone)
	toZone = normalizeTimeZone(toZone)

	// Parse time or use current time
	var inputTime time.Time
	var err error

	if timeStr == "" {
		inputTime = time.Now()
	} else {
		inputTime, err = time.Parse(time.RFC3339, timeStr)
		if err != nil {
			return common.NewInputError("invalid time format (use RFC3339, e.g., 2025-01-01T12:00:00Z)")
		}
	}

	// Create service and convert
	svc := getService()
	ctx, cancel := common.CreateContext()
	defer cancel()

	converted, err := svc.ConvertTime(ctx, fromZone, toZone, inputTime)
	if err != nil {
		return common.WrapError(err)
	}

	// Get time zone info for both zones
	fromInfo, err := svc.GetTimeZoneInfo(ctx, fromZone, inputTime)
	if err != nil {
		return common.WrapGetError("source zone info", err)
	}

	toInfo, err := svc.GetTimeZoneInfo(ctx, toZone, converted)
	if err != nil {
		return common.WrapGetError("target zone info", err)
	}

	// Output
	if jsonOut {
		return common.PrintJSON(map[string]any{
			"from": map[string]any{
				"zone":   fromZone,
				"time":   inputTime.Format(time.RFC3339),
				"abbr":   fromInfo.Abbreviation,
				"offset": formatOffset(fromInfo.Offset),
				"is_dst": fromInfo.IsDST,
			},
			"to": map[string]any{
				"zone":   toZone,
				"time":   converted.Format(time.RFC3339),
				"abbr":   toInfo.Abbreviation,
				"offset": formatOffset(toInfo.Offset),
				"is_dst": toInfo.IsDST,
			},
		})
	}

	// Human-readable output
	fmt.Printf("Time Zone Conversion\n\n")

	fmt.Printf("From: %s (%s)\n", fromZone, fromInfo.Abbreviation)
	fmt.Printf("  Time:   %s\n", formatTime(inputTime, false))
	fmt.Printf("  Offset: %s\n", formatOffset(fromInfo.Offset))
	if fromInfo.IsDST {
		fmt.Printf("  DST:    Yes (Daylight Saving Time)\n")
	} else {
		fmt.Printf("  DST:    No (Standard Time)\n")
	}

	fmt.Println()

	fmt.Printf("To: %s (%s)\n", toZone, toInfo.Abbreviation)
	fmt.Printf("  Time:   %s\n", formatTime(converted, false))
	fmt.Printf("  Offset: %s\n", formatOffset(toInfo.Offset))
	if toInfo.IsDST {
		fmt.Printf("  DST:    Yes (Daylight Saving Time)\n")
	} else {
		fmt.Printf("  DST:    No (Standard Time)\n")
	}

	fmt.Println()

	// Show time difference
	offsetDiff := toInfo.Offset - fromInfo.Offset
	hoursDiff := offsetDiff / 3600
	if hoursDiff > 0 {
		fmt.Printf("Time Difference: %s is %d hour(s) ahead of %s\n",
			toZone, hoursDiff, fromZone)
	} else if hoursDiff < 0 {
		fmt.Printf("Time Difference: %s is %d hour(s) behind %s\n",
			toZone, -hoursDiff, fromZone)
	} else {
		fmt.Printf("Time Difference: Both time zones have the same offset\n")
	}

	return nil
}
