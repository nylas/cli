// Package cli provides the command-line interface.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/cli/setup"
)

var rootCmd = &cobra.Command{
	Use:     "nylas",
	Short:   "Nylas CLI - Email, calendar, and contacts from your terminal",
	Version: Version,
	Long: `Quick start:
  nylas init             Guided setup (first time)
  nylas email list       List recent emails
  nylas calendar events  Upcoming events
  nylas contacts list    List contacts

Documentation: https://cli.nylas.com/`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if setup.IsFirstRun() {
			printWelcome()
			return nil
		}
		printHelpHeader()
		return cmd.Help()
	},
}

// printHelpHeader prints the branded ASCII art header.
func printHelpHeader() {
	fmt.Println()
	_, _ = common.BoldCyan.Println("  ┳┓      ┓       ┏┓┓ ┳")
	_, _ = common.BoldCyan.Println("  ┃┃┓┏┃┏┓┏┃  ╺━╸  ┃ ┃ ┃")
	_, _ = common.BoldCyan.Println("  ┛┗┗┫┗┗┻┛┗       ┗┛┗┛┻")
	_, _ = common.BoldCyan.Println("     ┛")
	fmt.Println()
}

// printWelcome displays the first-run welcome message.
func printWelcome() {
	// Banner
	fmt.Println()
	_, _ = common.Dim.Println("  ╭──────────────────────────────────────────╮")
	_, _ = common.Dim.Println("  │                                          │")
	fmt.Print("  ")
	_, _ = common.Dim.Print("│")
	fmt.Print("   ")
	_, _ = common.BoldCyan.Print("◈  N Y L A S   C L I")
	fmt.Print("                  ")
	_, _ = common.Dim.Println("│")
	_, _ = common.Dim.Println("  │                                          │")
	fmt.Print("  ")
	_, _ = common.Dim.Print("│")
	fmt.Print("   Email, calendar, and contacts          ")
	_, _ = common.Dim.Println("│")
	fmt.Print("  ")
	_, _ = common.Dim.Print("│")
	fmt.Print("   from your terminal.                    ")
	_, _ = common.Dim.Println("│")
	_, _ = common.Dim.Println("  │                                          │")
	_, _ = common.Dim.Println("  ╰──────────────────────────────────────────╯")

	// Getting started
	fmt.Println()
	_, _ = common.Bold.Println("  Get started in under a minute:")
	fmt.Println()
	fmt.Print("    ")
	_, _ = common.BoldCyan.Print("❯ nylas init")
	fmt.Println("                Guided setup")
	fmt.Print("    ")
	_, _ = common.Dim.Println("  nylas init --api-key      Quick setup with existing key")

	// Capabilities box
	fmt.Println()
	_, _ = common.Dim.Print("  ╭─")
	_, _ = common.Bold.Print(" What you can do ")
	_, _ = common.Dim.Println("────────────────────────╮")
	_, _ = common.Dim.Println("  │                                          │")
	printCapability("email", "Send, search, and read")
	printCapability("calendar", "Events and availability")
	printCapability("contacts", "People and groups")
	printCapability("webhook", "Real-time notifications")
	printCapability("ai", "Chat with your data")
	_, _ = common.Dim.Println("  │                                          │")
	_, _ = common.Dim.Println("  ╰──────────────────────────────────────────╯")

	// Footer
	fmt.Println()
	fmt.Print("  ")
	_, _ = common.Dim.Print("nylas --help")
	fmt.Println("              All commands")
	fmt.Print("  ")
	_, _ = common.Dim.Println("https://cli.nylas.com     Documentation")
	fmt.Println()
}

// printCapability prints a single capability row inside the box.
func printCapability(name, desc string) {
	fmt.Print("  ")
	_, _ = common.Dim.Print("│")
	fmt.Print("  ")
	_, _ = common.Cyan.Printf("%-12s", name)
	fmt.Printf("%-28s", desc)
	_, _ = common.Dim.Println("│")
}

func init() {
	// Global output flags (format, json, quiet, wide, no-color)
	rootCmd.PersistentFlags().String("format", "", "Output format: table, json, yaml")
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Quiet mode - only output essential data (IDs)")
	rootCmd.PersistentFlags().BoolP("wide", "w", false, "Wide output - show full IDs without truncation")
	rootCmd.PersistentFlags().Bool("no-color", false, "Disable color output")

	// Other global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().String("config", "", "Custom config file path")

	rootCmd.AddCommand(newVersionCmd())

	// Initialize audit logging hooks
	initAuditHooks(rootCmd)
}

// GetRootCmd returns the root command for adding subcommands.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// Execute runs the CLI.
func Execute() error {
	return rootCmd.Execute()
}
