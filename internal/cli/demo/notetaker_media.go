package demo

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

// ============================================================================
// MEDIA COMMAND
// ============================================================================

// newDemoNotetakerMediaCmd creates the media subcommand group.
func newDemoNotetakerMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "Access notetaker media (recordings, transcripts)",
		Long:  "Demo commands for accessing notetaker recordings and transcripts.",
	}

	cmd.AddCommand(newDemoMediaShowCmd())
	cmd.AddCommand(newDemoMediaDownloadCmd())
	cmd.AddCommand(newDemoMediaTranscriptCmd())
	cmd.AddCommand(newDemoMediaSummaryCmd())
	cmd.AddCommand(newDemoMediaActionItemsCmd())

	return cmd
}

func newDemoMediaShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [notetaker-id]",
		Short: "Show available media for a notetaker",
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := "notetaker-demo-001"
			if len(args) > 0 {
				notetakerID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - Notetaker Media"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Printf("Notetaker: %s\n", notetakerID)
			fmt.Println()

			fmt.Println("📁 Available Media:")
			fmt.Println()
			fmt.Printf("  %s Recording\n", common.Green.Sprint("●"))
			fmt.Printf("    Format:   MP4\n")
			fmt.Printf("    Size:     245.6 MB\n")
			fmt.Printf("    Duration: 45:32\n")
			_, _ = common.Dim.Printf("    URL:      https://media.example.com/recordings/%s.mp4\n", notetakerID)
			fmt.Println()

			fmt.Printf("  %s Transcript\n", common.Green.Sprint("●"))
			fmt.Printf("    Format:   VTT\n")
			fmt.Printf("    Size:     128.4 KB\n")
			_, _ = common.Dim.Printf("    URL:      https://media.example.com/transcripts/%s.vtt\n", notetakerID)
			fmt.Println()

			fmt.Printf("  %s Summary\n", common.Green.Sprint("●"))
			fmt.Printf("    Format:   JSON\n")
			fmt.Printf("    Size:     4.2 KB\n")
			fmt.Println()

			fmt.Printf("  %s Action Items\n", common.Green.Sprint("●"))
			fmt.Printf("    Count:    5 items\n")

			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}
}

func newDemoMediaDownloadCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "download [notetaker-id]",
		Short: "Download notetaker recording",
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := "notetaker-demo-001"
			if len(args) > 0 {
				notetakerID = args[0]
			}

			if output == "" {
				output = fmt.Sprintf("%s-recording.mp4", notetakerID)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - Download Recording"))
			fmt.Println()
			fmt.Printf("Notetaker: %s\n", notetakerID)
			fmt.Printf("Output:    %s\n", output)
			fmt.Println()
			_, _ = common.Green.Println("✓ Recording would be downloaded (demo mode)")
			fmt.Printf("  Size: 245.6 MB\n")
			fmt.Printf("  Duration: 45:32\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")

	return cmd
}

func newDemoMediaTranscriptCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "transcript [notetaker-id]",
		Short: "Get meeting transcript",
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := "notetaker-demo-001"
			if len(args) > 0 {
				notetakerID = args[0]
			}

			if format == "" {
				format = "text"
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - Meeting Transcript"))
			fmt.Println()
			fmt.Printf("Notetaker: %s\n", notetakerID)
			fmt.Printf("Format:    %s\n", format)
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()

			// Sample transcript
			fmt.Println("[00:00:05] John: Good morning everyone, let's get started with the standup.")
			fmt.Println()
			fmt.Println("[00:00:12] Sarah: Sure! Yesterday I finished the authentication module.")
			fmt.Println("                 Today I'm moving on to the API integration.")
			fmt.Println()
			fmt.Println("[00:00:28] Mike: I'm still working on the database optimization.")
			fmt.Println("                Should be done by end of day.")
			fmt.Println()
			fmt.Println("[00:00:45] John: Great progress team! Any blockers?")
			fmt.Println()
			_, _ = common.Dim.Println("... (transcript continues)")
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "", "Output format (text, vtt, json)")

	return cmd
}

func newDemoMediaSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "summary [notetaker-id]",
		Short: "Get AI-generated meeting summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := "notetaker-demo-001"
			if len(args) > 0 {
				notetakerID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - AI Meeting Summary"))
			fmt.Println()
			fmt.Printf("Notetaker: %s\n", notetakerID)
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Println("Meeting Summary")
			fmt.Println()

			fmt.Println("📋 Overview:")
			fmt.Println("   The team conducted their daily standup meeting to discuss")
			fmt.Println("   progress on the current sprint. All team members provided")
			fmt.Println("   updates on their tasks and no major blockers were identified.")
			fmt.Println()

			fmt.Println("👥 Participants:")
			fmt.Println("   • John (Meeting Host)")
			fmt.Println("   • Sarah")
			fmt.Println("   • Mike")
			fmt.Println()

			fmt.Println("📝 Key Points:")
			fmt.Println("   • Authentication module completed")
			fmt.Println("   • API integration work starting today")
			fmt.Println("   • Database optimization in progress")
			fmt.Println("   • Sprint on track for completion")
			fmt.Println()

			fmt.Println("⏱️  Duration: 15 minutes")
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}
}

func newDemoMediaActionItemsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "action-items [notetaker-id]",
		Short: "Get AI-extracted action items",
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := "notetaker-demo-001"
			if len(args) > 0 {
				notetakerID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("🤖 Demo Mode - AI Action Items"))
			fmt.Println()
			fmt.Printf("Notetaker: %s\n", notetakerID)
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Println("Action Items")
			fmt.Println()

			items := []struct {
				assignee string
				task     string
				due      string
			}{
				{"Sarah", "Complete API integration for user endpoints", "Today"},
				{"Mike", "Finish database optimization", "End of day"},
				{"John", "Review Sarah's authentication PR", "Tomorrow"},
				{"Sarah", "Write unit tests for auth module", "Tomorrow"},
				{"Mike", "Document database schema changes", "This week"},
			}

			for i, item := range items {
				fmt.Printf("  %s %s\n", common.Cyan.Sprintf("%d.", i+1), common.BoldWhite.Sprint(item.task))
				fmt.Printf("     Assignee: %s\n", item.assignee)
				fmt.Printf("     Due:      %s\n", item.due)
				fmt.Println()
			}

			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}
}
