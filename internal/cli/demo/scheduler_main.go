package demo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
)

// newDemoSchedulerCmd creates the demo scheduler command with subcommands.
func newDemoSchedulerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scheduler",
		Short: "Explore scheduler features with sample data",
		Long:  "Demo scheduler commands showing sample configurations, bookings, and sessions.",
	}

	cmd.AddCommand(newDemoSchedulerConfigurationsCmd())
	cmd.AddCommand(newDemoSchedulerSessionsCmd())
	cmd.AddCommand(newDemoSchedulerBookingsCmd())
	cmd.AddCommand(newDemoSchedulerPagesCmd())

	return cmd
}

// ============================================================================
// CONFIGURATIONS COMMANDS
// ============================================================================

// newDemoSchedulerConfigurationsCmd creates the configurations subcommand group.
func newDemoSchedulerConfigurationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "configurations",
		Aliases: []string{"config", "configs"},
		Short:   "Manage scheduler configurations",
		Long:    "Demo commands for managing scheduler configurations.",
	}

	cmd.AddCommand(newDemoConfigListCmd())
	cmd.AddCommand(newDemoConfigShowCmd())
	cmd.AddCommand(newDemoConfigCreateCmd())
	cmd.AddCommand(newDemoConfigUpdateCmd())
	cmd.AddCommand(newDemoConfigDeleteCmd())

	return cmd
}

func newDemoConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List scheduler configurations",
		Example: `  # List sample scheduler configs
  nylas demo scheduler configurations list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := nylas.NewDemoClient()
			ctx := context.Background()

			configs, err := client.ListSchedulerConfigurations(ctx)
			if err != nil {
				return common.WrapListError("scheduler configs", err)
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Scheduler Configurations"))
			fmt.Println(common.Dim.Sprint("These are sample scheduling pages for demonstration purposes."))
			fmt.Println()
			fmt.Printf("Found %d configurations:\n\n", len(configs))

			for _, config := range configs {
				fmt.Printf("  %s %s\n", "📅", common.BoldWhite.Sprint(config.Name))
				fmt.Printf("    Slug: %s\n", config.Slug)
				fmt.Printf("    URL:  https://schedule.nylas.com/%s\n", config.Slug)
				_, _ = common.Dim.Printf("    ID:   %s\n", config.ID)
				fmt.Println()
			}

			fmt.Println(common.Dim.Sprint("To set up your own scheduler: nylas auth login"))

			return nil
		},
	}
}

func newDemoConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show [config-id]",
		Short: "Show configuration details",
		RunE: func(cmd *cobra.Command, args []string) error {
			configID := "config-demo-001"
			if len(args) > 0 {
				configID = args[0]
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Configuration Details"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Println("30-Minute Meeting")
			fmt.Printf("  ID:              %s\n", configID)
			fmt.Printf("  Slug:            30-min-meeting\n")
			fmt.Printf("  Duration:        30 minutes\n")
			fmt.Printf("  Buffer (before): 5 minutes\n")
			fmt.Printf("  Buffer (after):  10 minutes\n")
			fmt.Printf("  Location:        Zoom\n")
			fmt.Printf("  Timezone:        America/New_York\n")
			fmt.Println()
			fmt.Println("Availability:")
			fmt.Printf("  Monday-Friday:   9:00 AM - 5:00 PM\n")
			fmt.Printf("  Weekends:        Not available\n")
			fmt.Println(strings.Repeat("─", 50))

			return nil
		},
	}
}

func newDemoConfigCreateCmd() *cobra.Command {
	var name string
	var duration int
	var slug string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a scheduler configuration",
		Example: `  # Create a 30-minute meeting config
  nylas demo scheduler configurations create --name "Quick Chat" --duration 30

  # Create with custom slug
  nylas demo scheduler configurations create --name "Team Sync" --duration 60 --slug team-sync`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				name = "Demo Meeting"
			}
			if duration == 0 {
				duration = 30
			}
			if slug == "" {
				slug = "demo-meeting"
			}

			fmt.Println()
			fmt.Println(common.Dim.Sprint("📆 Demo Mode - Create Configuration"))
			fmt.Println()
			fmt.Println(strings.Repeat("─", 50))
			_, _ = common.BoldWhite.Printf("Name:     %s\n", name)
			fmt.Printf("Duration: %d minutes\n", duration)
			fmt.Printf("Slug:     %s\n", slug)
			fmt.Printf("URL:      https://schedule.nylas.com/%s\n", slug)
			fmt.Println(strings.Repeat("─", 50))
			fmt.Println()
			_, _ = common.Green.Println("✓ Configuration would be created (demo mode)")
			_, _ = common.Dim.Printf("  Config ID: config-demo-%d\n", time.Now().Unix())
			fmt.Println()
			fmt.Println(common.Dim.Sprint("To create real configurations: nylas auth login"))

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Configuration name")
	cmd.Flags().IntVar(&duration, "duration", 30, "Meeting duration in minutes")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")

	return cmd
}
