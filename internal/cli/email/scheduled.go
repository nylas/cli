package email

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newScheduledCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scheduled",
		Short: "Manage scheduled messages",
		Long:  "List, view, and cancel scheduled messages.",
	}

	cmd.AddCommand(newScheduledListCmd())
	cmd.AddCommand(newScheduledShowCmd())
	cmd.AddCommand(newScheduledCancelCmd())

	return cmd
}

func newScheduledListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [grant-id]",
		Short: "List scheduled messages",
		Long:  "List all messages that are scheduled to be sent.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				scheduled, err := client.ListScheduledMessages(ctx, grantID)
				if err != nil {
					return struct{}{}, common.WrapListError("scheduled messages", err)
				}

				// JSON output (including empty array)
				if common.IsStructuredOutput(cmd) {
					out := common.GetOutputWriter(cmd)
					return struct{}{}, out.Write(scheduled)
				}

				if len(scheduled) == 0 {
					common.PrintEmptyState("scheduled messages")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d scheduled message(s):\n\n", len(scheduled))

				for _, s := range scheduled {
					closeTime := time.Unix(s.CloseTime, 0)
					timeUntil := time.Until(closeTime)

					statusIcon := "⏳"
					switch s.Status {
					case "cancelled":
						statusIcon = "❌"
					case "sent":
						statusIcon = "✅"
					}

					fmt.Printf("%s  Schedule ID: %s\n", statusIcon, s.ScheduleID)
					fmt.Printf("   Status:      %s\n", s.Status)
					fmt.Printf("   Send at:     %s\n", closeTime.Format(common.DisplayDateTime))

					if timeUntil > 0 {
						fmt.Printf("   Time until:  %s\n", formatDuration(timeUntil))
					}
					fmt.Println()
				}

				return struct{}{}, nil
			})
			return err
		},
	}
}

func newScheduledShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <schedule-id> [grant-id]",
		Short: "Show scheduled message details",
		Long:  "Show details of a specific scheduled message.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			scheduleID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				scheduled, err := client.GetScheduledMessage(ctx, grantID, scheduleID)
				if err != nil {
					return struct{}{}, common.WrapGetError("scheduled message", err)
				}

				closeTime := time.Unix(scheduled.CloseTime, 0)
				timeUntil := time.Until(closeTime)

				fmt.Println("════════════════════════════════════════════════════════════")
				_, _ = common.BoldWhite.Printf("Scheduled Message: %s\n", scheduled.ScheduleID)
				fmt.Println("════════════════════════════════════════════════════════════")

				fmt.Printf("Status:      %s\n", scheduled.Status)
				fmt.Printf("Send at:     %s\n", closeTime.Format("Mon, Jan 2, 2006 3:04:05 PM MST"))

				if timeUntil > 0 {
					fmt.Printf("Time until:  %s\n", formatDuration(timeUntil))
				} else {
					fmt.Printf("Time since:  %s ago\n", formatDuration(-timeUntil))
				}

				return struct{}{}, nil
			})
			return err
		},
	}
}

func newScheduledCancelCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "cancel <schedule-id> [grant-id]",
		Short: "Cancel a scheduled message",
		Long:  "Cancel a message that is scheduled to be sent.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			scheduleID := args[0]
			remainingArgs := args[1:]

			_, err := common.WithClient(remainingArgs, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				// Get scheduled message info for confirmation
				if !force {
					scheduled, err := client.GetScheduledMessage(ctx, grantID, scheduleID)
					if err != nil {
						return struct{}{}, common.WrapGetError("scheduled message", err)
					}

					closeTime := time.Unix(scheduled.CloseTime, 0)

					fmt.Println("Cancel this scheduled message?")
					fmt.Printf("  Schedule ID: %s\n", scheduled.ScheduleID)
					fmt.Printf("  Status:      %s\n", scheduled.Status)
					fmt.Printf("  Send at:     %s\n", closeTime.Format(common.DisplayDateTime))
					fmt.Print("\n[y/N]: ")

					var confirm string
					_, _ = fmt.Scanln(&confirm) // Ignore error - empty string treated as "no"
					if confirm != "y" && confirm != "Y" && confirm != "yes" {
						fmt.Println("Cancelled.")
						return struct{}{}, nil
					}
				}

				err := client.CancelScheduledMessage(ctx, grantID, scheduleID)
				if err != nil {
					return struct{}{}, common.WrapCancelError("scheduled message", err)
				}

				printSuccess("Scheduled message %s cancelled", scheduleID)
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation")

	return cmd
}

// formatDuration formats a duration in a human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds", int(d.Seconds()))
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		mins := int(d.Minutes()) % 60
		if hours == 1 && mins == 0 {
			return "1 hour"
		}
		if mins == 0 {
			return fmt.Sprintf("%d hours", hours)
		}
		return fmt.Sprintf("%d hours %d minutes", hours, mins)
	}
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	if days == 1 && hours == 0 {
		return "1 day"
	}
	if hours == 0 {
		return fmt.Sprintf("%d days", days)
	}
	return fmt.Sprintf("%d days %d hours", days, hours)
}
