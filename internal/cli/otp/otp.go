// Package otp provides the otp subcommands.
package otp

import (
	"github.com/spf13/cobra"
)

// NewOTPCmd creates the otp command group.
func NewOTPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "otp",
		Short: "OTP management commands",
		Long: `Retrieve and manage OTP codes from email.

Commands:
  get       Get the latest OTP code
  watch     Watch for new OTP codes
  list      List configured accounts
  messages  Show recent messages (debug)

Guide: https://developer.nylas.com/docs/cookbook/cli/extract-otp-codes/`,
	}

	cmd.AddCommand(newGetCmd())
	cmd.AddCommand(newWatchCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newMessagesCmd())

	return cmd
}
