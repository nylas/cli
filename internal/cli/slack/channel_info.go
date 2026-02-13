// channel_info.go provides the channel info subcommand for viewing channel details.

package slack

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
)

// newChannelInfoCmd creates the info subcommand for getting channel details.
func newChannelInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [channel-id]",
		Short: "Get detailed info about a channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withSlackClient(func(ctx context.Context, client ports.SlackClient) error {
				channelID := args[0]
				ch, err := client.GetChannel(ctx, channelID)
				if err != nil {
					return common.WrapGetError("channel", err)
				}

				// Handle structured output (JSON/YAML/quiet)
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return out.Write(ch)
				}

				_, _ = common.Cyan.Printf("Channel: #%s\n", ch.Name)
				fmt.Printf("  ID:           %s\n", ch.ID)
				fmt.Printf("  Is Channel:   %v\n", ch.IsChannel)
				fmt.Printf("  Is Private:   %v\n", ch.IsPrivate)
				fmt.Printf("  Is Archived:  %v\n", ch.IsArchived)
				fmt.Printf("  Is Member:    %v\n", ch.IsMember)
				fmt.Printf("  Is Shared:    %v\n", ch.IsShared)
				fmt.Printf("  Is OrgShared: %v\n", ch.IsOrgShared)
				fmt.Printf("  Is ExtShared: %v\n", ch.IsExtShared)
				fmt.Printf("  Is IM:        %v\n", ch.IsIM)
				fmt.Printf("  Is MPIM:      %v\n", ch.IsMPIM)
				fmt.Printf("  Is Group:     %v\n", ch.IsGroup)
				fmt.Printf("  Members:      %d\n", ch.MemberCount)
				if ch.Purpose != "" {
					_, _ = common.Dim.Printf("  Purpose:      %s\n", ch.Purpose)
				}
				if ch.Topic != "" {
					_, _ = common.Dim.Printf("  Topic:        %s\n", ch.Topic)
				}

				return nil
			})
		},
	}

	return cmd
}
