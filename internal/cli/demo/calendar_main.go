package demo

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

func newDemoCalendarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendar",
		Short: "Explore calendar features with sample data",
		Long:  "Demo calendar commands showing sample events and simulated operations.",
	}

	// Calendar management
	cmd.AddCommand(newDemoCalendarsListCmd())
	cmd.AddCommand(newDemoCalendarShowCmd())

	// Event management
	cmd.AddCommand(newDemoCalendarListCmd())
	cmd.AddCommand(newDemoCalendarCreateCmd())
	cmd.AddCommand(newDemoCalendarUpdateCmd())
	cmd.AddCommand(newDemoCalendarDeleteCmd())

	// Events subcommand group
	cmd.AddCommand(newDemoEventsCmd())

	return cmd
}

// ============================================================================
// CALENDAR MANAGEMENT COMMANDS
// ============================================================================

// newDemoCalendarsListCmd lists sample calendars.
func printDemoEvent(event domain.Event, showID bool) {
	startTime := time.Unix(event.When.StartTime, 0)
	endTime := time.Unix(event.When.EndTime, 0)

	// Format time range
	var timeStr string
	if startTime.Day() == endTime.Day() {
		timeStr = fmt.Sprintf("%s - %s",
			startTime.Format("Jan 2, 3:04 PM"),
			endTime.Format("3:04 PM"))
	} else {
		timeStr = fmt.Sprintf("%s - %s",
			startTime.Format("Jan 2, 3:04 PM"),
			endTime.Format("Jan 2, 3:04 PM"))
	}

	// Status indicator
	statusColor := common.Green
	if event.Status == "cancelled" {
		statusColor = common.Red
	}

	fmt.Printf("  %s %s\n", statusColor.Sprint("●"), common.BoldWhite.Sprint(event.Title))
	fmt.Printf("    %s\n", common.Dim.Sprint(timeStr))

	if event.Location != "" {
		fmt.Printf("    📍 %s\n", event.Location)
	}

	if event.Conferencing != nil && event.Conferencing.Details != nil && event.Conferencing.Details.URL != "" {
		fmt.Printf("    🔗 %s\n", common.Dim.Sprint(event.Conferencing.Details.URL))
	}

	if len(event.Participants) > 0 {
		names := make([]string, 0, len(event.Participants))
		for _, p := range event.Participants {
			if p.Name != "" {
				names = append(names, p.Name)
			} else if p.Email != "" {
				names = append(names, p.Email)
			}
		}
		if len(names) > 0 {
			fmt.Printf("    👥 %s\n", strings.Join(names, ", "))
		}
	}

	if showID {
		_, _ = common.Dim.Printf("    ID: %s\n", event.ID)
	}

	fmt.Println()
}
