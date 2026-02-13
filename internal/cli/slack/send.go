// send.go provides commands for sending messages and replies to Slack channels.

package slack

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// newSendCmd creates the send command for sending messages to channels.
func newSendCmd() *cobra.Command {
	var (
		channelID   string
		channelName string
		text        string
		noConfirm   bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message to a channel",
		Long: `Send a message to a Slack channel as yourself.

The message will appear with your name and profile picture,
exactly as if you typed it in Slack.

Examples:
  # Send to channel
  nylas slack send --channel general --text "Hello team!"

  # Send without confirmation
  nylas slack send --channel general --text "Quick update" --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			resolvedChannelID := channelID
			if channelName != "" && channelID == "" {
				resolvedChannelID, err = resolveChannelName(ctx, client, channelName)
				if err != nil {
					return common.NewUserError(fmt.Sprintf("channel not found: %s", channelName), "Use --channel-id with the channel ID instead")
				}
			}

			if resolvedChannelID == "" {
				return common.NewUserError("channel is required", "Use --channel or --channel-id")
			}

			if text == "" {
				return common.NewUserError("message text is required", "Use --text")
			}

			if !noConfirm {
				fmt.Printf("Channel: %s\n", common.Cyan.Sprint(channelName))
				fmt.Printf("Message: %s\n\n", text)
				fmt.Print("Send this message? [y/N]: ")

				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))

				if confirm != "y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			msg, err := client.SendMessage(ctx, &domain.SlackSendMessageRequest{
				ChannelID: resolvedChannelID,
				Text:      text,
			})
			if err != nil {
				return common.WrapSendError("message", err)
			}

			_, _ = common.Green.Printf("✓ Message sent! ID: %s\n", msg.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&channelName, "channel", "c", "", "Channel name")
	cmd.Flags().StringVar(&channelID, "channel-id", "", "Channel ID")
	cmd.Flags().StringVarP(&text, "text", "t", "", "Message text")
	cmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation")

	return cmd
}

// newReplyCmd creates the reply command for replying to thread messages.
func newReplyCmd() *cobra.Command {
	var (
		channelID   string
		channelName string
		threadTS    string
		text        string
		broadcast   bool
		noConfirm   bool
	)

	cmd := &cobra.Command{
		Use:   "reply",
		Short: "Reply to a thread",
		Long: `Reply to a Slack thread as yourself.

Use the message timestamp/ID from 'nylas slack messages --id' as the thread ID.

Examples:
  # Reply to a thread
  nylas slack reply --channel general --thread 1234567890.123456 --text "Got it!"

  # Reply and also post to channel
  nylas slack reply --channel general --thread 1234567890.123456 --text "Update" --broadcast`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			resolvedChannelID := channelID
			if channelName != "" && channelID == "" {
				resolvedChannelID, err = resolveChannelName(ctx, client, channelName)
				if err != nil {
					return common.NewUserError(fmt.Sprintf("channel not found: %s", channelName), "Use --channel-id with the channel ID instead")
				}
			}

			if resolvedChannelID == "" {
				return common.NewUserError("channel is required", "")
			}
			if threadTS == "" {
				return common.NewUserError(
					"thread timestamp is required",
					"Use --thread with the message ID from 'nylas slack messages --id'",
				)
			}
			if text == "" {
				return common.NewUserError("message text is required", "Use --text")
			}

			if !noConfirm {
				fmt.Printf("Channel: %s\n", common.Cyan.Sprint(channelName))
				fmt.Printf("Thread:  %s\n", threadTS)
				fmt.Printf("Reply:   %s\n", text)
				if broadcast {
					fmt.Println("(Also posting to channel)")
				}
				fmt.Print("\nSend this reply? [y/N]: ")

				reader := bufio.NewReader(os.Stdin)
				confirm, _ := reader.ReadString('\n')
				confirm = strings.TrimSpace(strings.ToLower(confirm))

				if confirm != "y" && confirm != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			msg, err := client.SendMessage(ctx, &domain.SlackSendMessageRequest{
				ChannelID: resolvedChannelID,
				Text:      text,
				ThreadTS:  threadTS,
				Broadcast: broadcast,
			})
			if err != nil {
				return common.WrapSendError("reply", err)
			}

			_, _ = common.Green.Printf("✓ Reply sent! ID: %s\n", msg.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&channelName, "channel", "c", "", "Channel name")
	cmd.Flags().StringVar(&channelID, "channel-id", "", "Channel ID")
	cmd.Flags().StringVar(&threadTS, "thread", "", "Thread timestamp to reply to")
	cmd.Flags().StringVarP(&text, "text", "t", "", "Reply text")
	cmd.Flags().BoolVar(&broadcast, "broadcast", false, "Also post to channel")
	cmd.Flags().BoolVarP(&noConfirm, "yes", "y", false, "Skip confirmation")

	_ = cmd.MarkFlagRequired("thread")

	return cmd
}
