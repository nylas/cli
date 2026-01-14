package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// Note: fatih/color import needed for *color.Color type in stateColor variables

// newDemoNotetakerCmd creates the demo notetaker command with subcommands.
func newDemoNotetakerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notetaker",
		Short: "Explore AI notetaker features with sample data",
		Long:  "Demo notetaker commands showing sample meeting recordings and transcripts.",
	}

	cmd.AddCommand(newDemoNotetakerListCmd())
	cmd.AddCommand(newDemoNotetakerShowCmd())
	cmd.AddCommand(newDemoNotetakerCreateCmd())
	cmd.AddCommand(newDemoNotetakerDeleteCmd())
	cmd.AddCommand(newDemoNotetakerMediaCmd())

	return cmd
}

// ============================================================================
// LIST COMMAND
// ============================================================================

// newDemoNotetakerListCmd lists sample notetakers.
func newDemoNotetakerListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sample notetakers",
		Long:  "Display a list of sample AI notetaker sessions.",
		Example: `  # List sample notetakers
  nylas demo notetaker list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			notetakers, err := client.ListNotetakers(ctx, "demo-grant", nil)
			if err != nil {
				return common.WrapListError("notetakers", err)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - Sample AI Notetakers"))
			fmt.Println(common.Dim.Sprint("These are sample notetaker sessions for demonstration purposes."))
			fmt.Println()
			fmt.Printf("Found %d notetakers:\n\n", len(notetakers))

			for _, nt := range notetakers {
				printDemoNotetaker(nt)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("To use AI notetakers on your meetings: nylas auth login"))

			return nil
		},
	}

	return cmd
}

// ============================================================================
// SHOW COMMAND
// ============================================================================

// newDemoNotetakerShowCmd shows a sample notetaker.
func newDemoNotetakerShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "show [notetaker-id]",
		Aliases: []string{"read"},
		Short:   "Show a sample notetaker session",
		Long:    "Display a sample notetaker session with details.",
		Example: `  # Show first sample notetaker
  nylas demo notetaker show

  # Show specific notetaker
  nylas demo notetaker show notetaker-001`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			notetakerID := "notetaker-001"
			if len(args) > 0 {
				notetakerID = args[0]
			}

			nt, err := client.GetNotetaker(ctx, "demo-grant", notetakerID)
			if err != nil {
				return common.WrapGetError("notetaker", err)
			}

			// Get media info
			media, _ := client.GetNotetakerMedia(ctx, "demo-grant", notetakerID)

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - Sample Notetaker Session"))
			fmt.Println()
			printDemoNotetakerFull(*nt, media)

			fmt.Println(common.Dim.Sprint("To use AI notetakers on your meetings: nylas auth login"))

			return nil
		},
	}

	return cmd
}

// ============================================================================
// CREATE COMMAND
// ============================================================================

// newDemoNotetakerCreateCmd simulates creating a notetaker.
func newDemoNotetakerCreateCmd() *cobra.Command {
	var meetingLink string
	var name string
	var joinAt string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Simulate creating a notetaker",
		Long: `Simulate creating an AI notetaker to join a meeting.

No actual notetaker is created - this is just a demonstration of the command flow.`,
		Example: `  # Create notetaker for a Zoom meeting
  nylas demo notetaker create --meeting-link "https://zoom.us/j/123456789"

  # Create with custom name
  nylas demo notetaker create --meeting-link "https://meet.google.com/abc-defg-hij" --name "Project Review Bot"

  # Schedule notetaker for later
  nylas demo notetaker create --meeting-link "https://zoom.us/j/123" --join-at "2024-01-15 10:00"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if meetingLink == "" {
				meetingLink = "https://zoom.us/j/123456789"
			}
			if name == "" {
				name = "Nylas Notetaker"
			}

			// Detect meeting provider
			var provider string
			if strings.Contains(meetingLink, "zoom.us") {
				provider = "Zoom"
			} else if strings.Contains(meetingLink, "meet.google.com") {
				provider = "Google Meet"
			} else if strings.Contains(meetingLink, "teams.microsoft.com") {
				provider = "Microsoft Teams"
			} else {
				provider = "Unknown"
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - Simulated Notetaker Creation"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Printf("Name:         %s\n", name)
			fmt.Printf("Meeting Link: %s\n", meetingLink)
			fmt.Printf("Provider:     %s\n", provider)
			if joinAt != "" {
				fmt.Printf("Join At:      %s\n", joinAt)
			} else {
				fmt.Printf("Join At:      %s\n", common.Green.Sprint("Immediately"))
			}
			fmt.Println()
			fmt.Println("Settings:")
			fmt.Printf("  Recording:     %s\n", common.Green.Sprint("Enabled"))
			fmt.Printf("  Transcription: %s\n", common.Green.Sprint("Enabled"))
			fmt.Printf("  Summary:       %s\n", common.Green.Sprint("Enabled"))
			fmt.Printf("  Action Items:  %s\n", common.Green.Sprint("Enabled"))
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()
			_, _ = common.Green.Println("✓ Notetaker would be created (demo mode - no actual notetaker created)")
			_, _ = common.Dim.Printf("  Notetaker ID: notetaker-demo-%d\n", time.Now().Unix())
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To create real notetakers, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&meetingLink, "meeting-link", "", "Video conference meeting link")
	cmd.Flags().StringVar(&name, "name", "", "Display name for the notetaker bot")
	cmd.Flags().StringVar(&joinAt, "join-at", "", "When to join the meeting (optional)")

	return cmd
}

// ============================================================================
// DELETE COMMAND
// ============================================================================

// newDemoNotetakerDeleteCmd simulates deleting/cancelling a notetaker.
func newDemoNotetakerDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [notetaker-id]",
		Short: "Simulate deleting/cancelling a notetaker",
		Long:  "Simulate deleting or cancelling an AI notetaker session.",
		Example: `  # Delete/cancel a notetaker
  nylas demo notetaker delete notetaker-001

  # Force delete without confirmation
  nylas demo notetaker delete notetaker-001 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := "notetaker-demo-123"
			if len(args) > 0 {
				notetakerID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - Simulated Notetaker Deletion"))
			fmt.Println()

			if !force {
				_, _ = common.Yellow.Println("⚠ Would prompt for confirmation in real mode")
			}

			fmt.Printf("Notetaker ID: %s\n", notetakerID)
			fmt.Println()
			_, _ = common.Green.Println("✓ Notetaker would be cancelled/deleted (demo mode - no actual deletion)")
			fmt.Println("  If the notetaker was in a meeting, it would leave immediately")
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To manage real notetakers, connect your account: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// printDemoNotetaker prints a notetaker summary.
func printDemoNotetaker(nt domain.Notetaker) {
	// State icon and color
	var stateIcon string
	var stateColor *color.Color

	switch nt.State {
	case domain.NotetakerStateComplete:
		stateIcon = "✓"
		stateColor = common.Green
	case domain.NotetakerStateAttending:
		stateIcon = "●"
		stateColor = common.Cyan
	case domain.NotetakerStateScheduled:
		stateIcon = "○"
		stateColor = common.Yellow
	default:
		stateIcon = "?"
		stateColor = common.Dim
	}

	title := nt.MeetingTitle
	if title == "" {
		title = "Untitled Meeting"
	}

	fmt.Printf("  %s %s\n", stateColor.Sprint(stateIcon), common.BoldWhite.Sprint(title))
	fmt.Printf("    State: %s\n", stateColor.Sprint(string(nt.State)))
	fmt.Printf("    Link:  %s\n", common.Dim.Sprint(nt.MeetingLink))

	if !nt.JoinTime.IsZero() {
		fmt.Printf("    Join:  %s\n", nt.JoinTime.Format("Jan 2, 2006 3:04 PM"))
	}

	_, _ = common.Dim.Printf("    ID:    %s\n", nt.ID)
	fmt.Println()
}

// printDemoNotetakerFull prints full notetaker details.
func printDemoNotetakerFull(nt domain.Notetaker, media *domain.MediaData) {
	title := nt.MeetingTitle
	if title == "" {
		title = "Untitled Meeting"
	}

	// State icon and color
	var stateColor *color.Color
	switch nt.State {
	case domain.NotetakerStateComplete:
		stateColor = common.Green
	case domain.NotetakerStateAttending:
		stateColor = common.Cyan
	case domain.NotetakerStateScheduled:
		stateColor = common.Yellow
	default:
		stateColor = common.Dim
	}

	fmt.Println("────────────────────────────────────────────────────")
	_, _ = common.BoldWhite.Printf("Meeting: %s\n", title)
	fmt.Printf("State:   %s\n", stateColor.Sprint(string(nt.State)))
	fmt.Printf("Link:    %s\n", nt.MeetingLink)
	fmt.Printf("ID:      %s\n", nt.ID)

	if !nt.JoinTime.IsZero() {
		fmt.Printf("Join:    %s\n", nt.JoinTime.Format("Jan 2, 2006 3:04 PM"))
	}

	fmt.Printf("Created: %s\n", nt.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", nt.UpdatedAt.Format(time.RFC3339))

	if media != nil {
		fmt.Println("\n📁 Media Files:")

		if media.Recording != nil {
			fmt.Printf("  Recording:  %s\n", media.Recording.ContentType)
			fmt.Printf("              Size: %s\n", formatDemoSize(media.Recording.Size))
			_, _ = common.Dim.Printf("              URL: %s\n", media.Recording.URL)
		}

		if media.Transcript != nil {
			fmt.Printf("  Transcript: %s\n", media.Transcript.ContentType)
			fmt.Printf("              Size: %s\n", formatDemoSize(media.Transcript.Size))
			_, _ = common.Dim.Printf("              URL: %s\n", media.Transcript.URL)
		}
	}

	fmt.Println("────────────────────────────────────────────────────")
	fmt.Println()
}

// formatDemoSize formats a file size in bytes to a human-readable string.
func formatDemoSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
