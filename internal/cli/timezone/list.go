package timezone

import (
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		filter  string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all available IANA time zones",
		Long: `Display all available IANA time zones that can be used with timezone commands.

You can filter the list by region or zone name to find specific time zones.

Examples:
  # List all time zones
  nylas timezone list

  # Filter by region (America)
  nylas timezone list --filter America

  # Filter by city name
  nylas timezone list --filter Tokyo

  # Output as JSON
  nylas timezone list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(filter, jsonOut)
		},
	}

	cmd.Flags().StringVar(&filter, "filter", "", "Filter zones by name (case-insensitive)")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	return cmd
}

func runList(filter string, jsonOut bool) error {
	// Get all time zones
	svc := getService()
	ctx, cancel := common.CreateContext()
	defer cancel()

	zones, err := svc.ListTimeZones(ctx)
	if err != nil {
		return common.WrapListError("time zones", err)
	}

	// Filter if requested
	if filter != "" {
		filterLower := strings.ToLower(filter)
		filtered := make([]string, 0)

		for _, zone := range zones {
			if strings.Contains(strings.ToLower(zone), filterLower) {
				filtered = append(filtered, zone)
			}
		}

		zones = filtered
	}

	// Output
	if jsonOut {
		return common.PrintJSON(map[string]any{
			"zones": zones,
			"count": len(zones),
		})
	}

	// Human-readable output
	fmt.Printf("IANA Time Zones")
	if filter != "" {
		fmt.Printf(" (filtered by '%s')", filter)
	}
	fmt.Printf("\n\n")

	if len(zones) == 0 {
		common.PrintEmptyStateWithHint("zones", "try adjusting the filter")
		return nil
	}

	// Group by region
	grouped := groupZonesByRegion(zones)

	for region, regionZones := range grouped {
		fmt.Printf("═══ %s (%d) ═══\n", region, len(regionZones))

		// Get current time in each zone for display
		now := time.Now()

		for _, zone := range regionZones {
			// Get time zone info
			info, err := svc.GetTimeZoneInfo(ctx, zone, now)
			if err != nil {
				fmt.Printf("  • %s\n", zone)
				continue
			}

			// Format with current time and offset
			loc, _ := time.LoadLocation(zone)
			localTime := now.In(loc)

			fmt.Printf("  • %-40s %s  %s\n",
				zone,
				formatOffset(info.Offset),
				localTime.Format("15:04 MST"))
		}

		fmt.Println()
	}

	fmt.Printf("Total: %d time zone(s)\n", len(zones))

	return nil
}

// groupZonesByRegion groups zones by their region prefix.
func groupZonesByRegion(zones []string) map[string][]string {
	grouped := make(map[string][]string)

	for _, zone := range zones {
		parts := strings.Split(zone, "/")
		region := "Other"

		if len(parts) > 1 {
			region = parts[0]
		}

		grouped[region] = append(grouped[region], zone)
	}

	return grouped
}
