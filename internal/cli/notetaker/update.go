package notetaker

import (
	"context"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		joinTime       string
		botName        string
		videoRecording bool
		audioRecording bool
		transcription  bool
	)

	cmd := &cobra.Command{
		Use:   "update <notetaker-id> [grant-id]",
		Short: "Update a scheduled notetaker",
		Long: `Update a scheduled notetaker before it joins its meeting.

You can change when the bot joins, its display name, and what it records
(video, audio, transcription). Only the options you pass are changed.

This applies to notetakers that haven't joined yet (scheduled state).

API reference: https://developer.nylas.com/docs/v3/notetaker/`,
		Example: `  # Reschedule when the bot joins
  nylas notetaker update <notetaker-id> --join-time "tomorrow 2pm"

  # Rename the bot and turn off video recording
  nylas notetaker update <notetaker-id> --bot-name "Recorder" --video-recording=false`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := args[0]

			req := &domain.UpdateNotetakerRequest{}
			if joinTime != "" {
				parsedTime, err := parseJoinTime(joinTime)
				if err != nil {
					return common.WrapDateParseError("join time", err)
				}
				req.JoinTime = parsedTime.Unix()
			}
			if botName != "" {
				req.Name = botName
			}

			// Only include recording toggles the user explicitly set.
			ms := &domain.NotetakerMeetingSettings{}
			settingsChanged := false
			if cmd.Flags().Changed("video-recording") {
				ms.VideoRecording = &videoRecording
				settingsChanged = true
			}
			if cmd.Flags().Changed("audio-recording") {
				ms.AudioRecording = &audioRecording
				settingsChanged = true
			}
			if cmd.Flags().Changed("transcription") {
				ms.Transcription = &transcription
				settingsChanged = true
			}
			if settingsChanged {
				req.MeetingSettings = ms
			}

			if req.JoinTime == 0 && req.Name == "" && !settingsChanged {
				return common.NewUserError(
					"nothing to update",
					"Pass at least one of --join-time, --bot-name, --video-recording, --audio-recording, or --transcription.",
				)
			}

			_, err := common.WithClient(args[1:], func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				notetaker, err := client.UpdateNotetaker(ctx, grantID, notetakerID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("notetaker", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(notetaker)
				}

				common.PrintSuccess("Notetaker updated")
				fmt.Printf("State: %s\n", formatState(notetaker.State))
				if !notetaker.JoinTime.IsZero() {
					fmt.Printf("Join:  %s\n", notetaker.JoinTime.Local().Format(common.DisplayWeekdayFullWithTZ))
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&joinTime, "join-time", "j", "", "When to join (e.g., '2024-01-15 14:00', 'tomorrow 9am', '30m')")
	cmd.Flags().StringVar(&botName, "bot-name", "", "New display name for the notetaker bot")
	cmd.Flags().BoolVar(&videoRecording, "video-recording", false, "Record the meeting's video")
	cmd.Flags().BoolVar(&audioRecording, "audio-recording", false, "Record the meeting's audio")
	cmd.Flags().BoolVar(&transcription, "transcription", false, "Transcribe the meeting's audio")

	return cmd
}
