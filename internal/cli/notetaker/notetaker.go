package notetaker

import (
	"github.com/spf13/cobra"
)

// NewNotetakerCmd creates the notetaker command and its subcommands.
func NewNotetakerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "notetaker",
		Aliases: []string{"nt", "bot"},
		Short:   "Manage Nylas Notetaker bots",
		Long: `Manage Nylas Notetaker bots for meeting recording and transcription.

Notetaker bots can join video meetings (Zoom, Google Meet, Teams) to:
- Record the meeting
- Generate transcripts
- Provide meeting summaries

Use subcommands to create, list, show, delete notetakers and retrieve media.

API reference: https://developer.nylas.com/docs/v3/notetaker/`,
		Example: `  # List all notetakers
  nylas notetaker list

  # Create a notetaker to join a meeting
  nylas notetaker create --meeting-link "https://zoom.us/j/123456789"

  # Show notetaker details
  nylas notetaker show <notetaker-id>

  # Get recording/transcript
  nylas notetaker media <notetaker-id>

  # Delete/cancel a notetaker
  nylas notetaker delete <notetaker-id>`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newLeaveCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newMediaCmd())

	return cmd
}

// createContext creates a context with timeout.
