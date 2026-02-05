// Package main is the entry point for the nylas CLI.
package main

import (
	"fmt"
	"os"

	"github.com/nylas/cli/internal/air"
	"github.com/nylas/cli/internal/cli"
	"github.com/nylas/cli/internal/cli/admin"
	"github.com/nylas/cli/internal/cli/ai"
	"github.com/nylas/cli/internal/cli/audit"
	"github.com/nylas/cli/internal/cli/auth"
	"github.com/nylas/cli/internal/cli/calendar"
	"github.com/nylas/cli/internal/cli/config"
	"github.com/nylas/cli/internal/cli/contacts"
	"github.com/nylas/cli/internal/cli/demo"
	"github.com/nylas/cli/internal/cli/email"
	"github.com/nylas/cli/internal/cli/inbound"
	"github.com/nylas/cli/internal/cli/mcp"
	"github.com/nylas/cli/internal/cli/notetaker"
	"github.com/nylas/cli/internal/cli/otp"
	"github.com/nylas/cli/internal/cli/scheduler"
	"github.com/nylas/cli/internal/cli/slack"
	"github.com/nylas/cli/internal/cli/timezone"
	"github.com/nylas/cli/internal/cli/update"
	"github.com/nylas/cli/internal/cli/webhook"
	"github.com/nylas/cli/internal/ui"
)

func main() {
	// Add subcommands
	rootCmd := cli.GetRootCmd()

	// Enable command typo suggestions (e.g., "Did you mean 'email'?")
	rootCmd.SuggestionsMinimumDistance = 2
	rootCmd.AddCommand(ai.NewAICmd())
	rootCmd.AddCommand(audit.NewAuditCmd())
	rootCmd.AddCommand(auth.NewAuthCmd())
	rootCmd.AddCommand(config.NewConfigCmd())
	rootCmd.AddCommand(otp.NewOTPCmd())
	rootCmd.AddCommand(email.NewEmailCmd())
	rootCmd.AddCommand(calendar.NewCalendarCmd())
	rootCmd.AddCommand(contacts.NewContactsCmd())
	rootCmd.AddCommand(scheduler.NewSchedulerCmd())
	rootCmd.AddCommand(admin.NewAdminCmd())
	rootCmd.AddCommand(webhook.NewWebhookCmd())
	rootCmd.AddCommand(notetaker.NewNotetakerCmd())
	rootCmd.AddCommand(inbound.NewInboundCmd())
	rootCmd.AddCommand(timezone.NewTimezoneCmd())
	rootCmd.AddCommand(mcp.NewMCPCmd())
	rootCmd.AddCommand(slack.NewSlackCmd())
	rootCmd.AddCommand(demo.NewDemoCmd())
	rootCmd.AddCommand(cli.NewTUICmd())
	rootCmd.AddCommand(ui.NewUICmd())
	rootCmd.AddCommand(air.NewAirCmd())
	rootCmd.AddCommand(update.NewUpdateCmd())

	if err := cli.Execute(); err != nil {
		cli.LogAuditError(err)
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
