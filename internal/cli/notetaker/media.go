package notetaker

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media <notetaker-id> [grant-id]",
		Short: "Get notetaker media (recording and transcript)",
		Long: `Retrieve media files from a completed notetaker session.

Returns URLs to download:
- Recording: Video/audio recording of the meeting
- Transcript: Text transcript of the meeting

Note: Media URLs have an expiration time. Download them promptly.`,
		Example: `  # Get media URLs
  nylas notetaker media abc123

  # Output as JSON
  nylas notetaker media abc123 --json`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			notetakerID := args[0]

			_, err := common.WithClient(args[1:], func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				media, err := client.GetNotetakerMedia(ctx, grantID, notetakerID)
				if err != nil {
					return struct{}{}, common.WrapGetError("notetaker media", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, common.PrintJSON(media)
				}

				if media.Recording == nil && media.Transcript == nil {
					fmt.Println("No media available yet.")
					fmt.Println("Media is generated after the meeting ends and processing completes.")
					return struct{}{}, nil
				}

				_, _ = common.BoldCyan.Println("Notetaker Media:")
				fmt.Println()

				if media.Recording != nil {
					_, _ = common.Green.Println("Recording:")
					fmt.Printf("  URL:  %s\n", media.Recording.URL)
					if media.Recording.ContentType != "" {
						_, _ = common.Dim.Printf("  Type: %s\n", media.Recording.ContentType)
					}
					if media.Recording.Size > 0 {
						_, _ = common.Dim.Printf("  Size: %s\n", common.FormatSize(media.Recording.Size))
					}
					if media.Recording.ExpiresAt > 0 {
						expires := time.Unix(media.Recording.ExpiresAt, 0)
						_, _ = common.Dim.Printf("  Expires: %s\n", expires.Local().Format(common.DisplayWeekdayFullWithTZ))
					}
					fmt.Println()
				}

				if media.Transcript != nil {
					_, _ = common.Green.Println("Transcript:")
					fmt.Printf("  URL:  %s\n", media.Transcript.URL)
					if media.Transcript.ContentType != "" {
						_, _ = common.Dim.Printf("  Type: %s\n", media.Transcript.ContentType)
					}
					if media.Transcript.Size > 0 {
						_, _ = common.Dim.Printf("  Size: %s\n", common.FormatSize(media.Transcript.Size))
					}
					if media.Transcript.ExpiresAt > 0 {
						expires := time.Unix(media.Transcript.ExpiresAt, 0)
						_, _ = common.Dim.Printf("  Expires: %s\n", expires.Local().Format(common.DisplayWeekdayFullWithTZ))
					}
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}
