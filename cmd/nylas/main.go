// Package main is the entry point for the nylas CLI.
package main

import (
	"fmt"
	"os"

	"github.com/nylas/cli/internal/cli"
	"github.com/nylas/cli/internal/cli/admin"
	"github.com/nylas/cli/internal/cli/auth"
	"github.com/nylas/cli/internal/cli/calendar"
	"github.com/nylas/cli/internal/cli/contacts"
	"github.com/nylas/cli/internal/cli/demo"
	"github.com/nylas/cli/internal/cli/email"
	"github.com/nylas/cli/internal/cli/inbound"
	"github.com/nylas/cli/internal/cli/mcp"
	"github.com/nylas/cli/internal/cli/notetaker"
	"github.com/nylas/cli/internal/cli/scheduler"
	"github.com/nylas/cli/internal/cli/update"
	"github.com/nylas/cli/internal/cli/webhook"
)

func main() {
	// Add subcommands
	rootCmd := cli.GetRootCmd()
	rootCmd.AddCommand(auth.NewAuthCmd())
	rootCmd.AddCommand(email.NewEmailCmd())
	rootCmd.AddCommand(calendar.NewCalendarCmd())
	rootCmd.AddCommand(contacts.NewContactsCmd())
	rootCmd.AddCommand(scheduler.NewSchedulerCmd())
	rootCmd.AddCommand(admin.NewAdminCmd())
	rootCmd.AddCommand(webhook.NewWebhookCmd())
	rootCmd.AddCommand(notetaker.NewNotetakerCmd())
	rootCmd.AddCommand(inbound.NewInboundCmd())
	rootCmd.AddCommand(mcp.NewMCPCmd())
	rootCmd.AddCommand(demo.NewDemoCmd())
	rootCmd.AddCommand(update.NewUpdateCmd())

	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
