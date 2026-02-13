package timezone

import (
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newDSTCmd() *cobra.Command {
	var (
		zone    string
		year    int
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "dst",
		Short: "Show DST (Daylight Saving Time) transitions for a time zone",
		Long: `Display when DST begins and ends for a specific time zone in a given year.

This helps identify when clocks "spring forward" or "fall back" and
warns about potential scheduling issues around DST transitions.

Note: Not all time zones observe DST. Some regions (e.g., Arizona, Hawaii)
stay on standard time year-round.

Examples:
  # Check DST transitions for New York in 2026
  nylas timezone dst --zone America/New_York --year 2026

  # Check for current year (default)
  nylas timezone dst --zone Europe/London

  # Check a zone that doesn't observe DST
  nylas timezone dst --zone America/Phoenix

  # Output as JSON
  nylas timezone dst --zone PST --year 2026 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDST(zone, year, jsonOut)
		},
	}

	currentYear := time.Now().Year()
	cmd.Flags().StringVar(&zone, "zone", "", "Time zone (IANA name or abbreviation)")
	cmd.Flags().IntVar(&year, "year", currentYear, "Year to check")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	_ = cmd.MarkFlagRequired("zone")

	return cmd
}

func runDST(zone string, year int, jsonOut bool) error {
	// Normalize time zone name
	zone = normalizeTimeZone(zone)

	// Get DST transitions
	svc := getService()
	ctx, cancel := common.CreateContext()
	defer cancel()

	transitions, err := svc.GetDSTTransitions(ctx, zone, year)
	if err != nil {
		return common.WrapGetError("DST transitions", err)
	}

	// Output
	if jsonOut {
		return common.PrintJSON(map[string]any{
			"zone":        zone,
			"year":        year,
			"transitions": transitions,
			"count":       len(transitions),
		})
	}

	// Human-readable output
	fmt.Printf("DST Transitions for %s in %d\n\n", zone, year)

	if len(transitions) == 0 {
		fmt.Println("❌ No DST transitions found")
		fmt.Println("\nThis time zone likely does not observe Daylight Saving Time.")
		fmt.Println("It stays on standard time throughout the year.")
		fmt.Println("\nExamples of non-DST zones:")
		fmt.Println("  • America/Phoenix (Arizona)")
		fmt.Println("  • Pacific/Honolulu (Hawaii)")
		fmt.Println("  • Asia/Tokyo (Japan)")
		fmt.Println("  • Asia/Kolkata (India)")
		return nil
	}

	fmt.Printf("Found %d transition(s):\n\n", len(transitions))

	// Create table
	headers := []string{"Date", "Time", "Direction", "Name", "Offset"}
	rows := make([][]string, len(transitions))

	for i, t := range transitions {
		var direction string
		var emoji string

		if t.Direction == "forward" {
			direction = "Spring Forward"
			emoji = "⏰"
		} else {
			direction = "Fall Back"
			emoji = "🕐"
		}

		rows[i] = []string{
			emoji + " " + t.Date.Format("2006-01-02"),
			t.Date.Format("15:04:05"),
			direction,
			t.Name,
			formatOffset(t.Offset),
		}
	}

	printTable(headers, rows)

	fmt.Println("\nLegend:")
	fmt.Println("  ⏰ Spring Forward: Clocks move ahead (lose 1 hour)")
	fmt.Println("  🕐 Fall Back: Clocks move back (gain 1 hour)")

	// Show warnings for upcoming transitions
	now := time.Now()
	for _, t := range transitions {
		if t.Date.After(now) && t.Date.Before(now.AddDate(0, 0, 30)) {
			fmt.Printf("\n⚠️  WARNING: DST transition in %d days (%s)\n",
				int(t.Date.Sub(now).Hours()/24),
				t.Date.Format("January 2"))
			fmt.Println("   Be mindful when scheduling meetings around this date.")
			break
		}
	}

	return nil
}
