package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newConfigurationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "configurations",
		Aliases: []string{"config", "configs"},
		Short:   "Manage scheduler configurations",
		Long:    "Manage scheduler configurations (meeting types) for scheduling workflows.",
	}

	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigCreateCmd())
	cmd.AddCommand(newConfigUpdateCmd())
	cmd.AddCommand(newConfigDeleteCmd())

	return cmd
}

func newConfigListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List scheduler configurations",
		Long:    "List all scheduler configurations (meeting types).",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				configs, err := client.ListSchedulerConfigurations(ctx)
				if err != nil {
					return struct{}{}, common.WrapListError("configurations", err)
				}

				if jsonOutput {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(configs)
				}

				if len(configs) == 0 {
					common.PrintEmptyState("configurations")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d configuration(s):\n\n", len(configs))

				table := common.NewTable("NAME", "ID", "SLUG", "PARTICIPANTS")
				for _, cfg := range configs {
					participantCount := fmt.Sprintf("%d", len(cfg.Participants))
					table.AddRow(common.Cyan.Sprint(cfg.Name), cfg.ID, common.Green.Sprint(cfg.Slug), participantCount)
				}
				table.Render()

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <config-id>",
		Short: "Show scheduler configuration details",
		Long:  "Show detailed information about a specific scheduler configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			configID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				config, err := client.GetSchedulerConfiguration(ctx, configID)
				if err != nil {
					return struct{}{}, common.WrapGetError("configuration", err)
				}

				if jsonOutput {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(config)
				}

				formatConfigDetails(cmd.OutOrStdout(), config)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newConfigCreateCmd() *cobra.Command {
	var (
		name         string
		participants []string
		duration     int
		title        string
		description  string
		location     string
		jsonOutput   bool
	)
	flags := &configFlags{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a scheduler configuration",
		Long: `Create a new scheduler configuration (meeting type).

Use flags for common settings, or --file for full JSON config input.
When both are provided, flags override file values.`,
		Example: `  # Simple inline creation
  nylas scheduler configs create --name "Quick Chat" --title "Quick Chat" \
    --participants alice@co.com --duration 15

  # With availability settings
  nylas scheduler configs create --name "Product Demo" --title "Demo" \
    --participants alice@co.com --duration 30 --interval 15 \
    --buffer-before 5 --buffer-after 10 --conferencing-provider "Google Meet"

  # From a JSON file
  nylas scheduler configs create --file config.json

  # File as base, override specific values
  nylas scheduler configs create --file config.json --duration 60`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateConfigFlags(flags); err != nil {
				return err
			}

			if flags.file == "" {
				if len(participants) == 0 {
					return common.NewUserError("at least one participant email is required", "Use --participants to specify email addresses")
				}
				for i, p := range participants {
					p = strings.TrimSpace(p)
					if p == "" {
						return fmt.Errorf("participant email at position %d cannot be empty", i+1)
					}
					participants[i] = p
				}
			}

			req, err := buildCreateRequest(cmd, flags, name, participants, duration, title, description, location)
			if err != nil {
				return err
			}
			if err := validateCreateRequest(req); err != nil {
				return err
			}

			_, err = common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				config, err := client.CreateSchedulerConfiguration(ctx, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("configuration", err)
				}

				if jsonOutput {
					return struct{}{}, common.PrintJSON(config)
				}

				_, _ = common.Green.Printf("✓ Created configuration: %s\n", config.Name)
				fmt.Printf("  ID: %s\n", config.ID)
				if config.Slug != "" {
					fmt.Printf("  Slug: %s\n", config.Slug)
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Configuration name")
	cmd.Flags().StringSliceVar(&participants, "participants", []string{}, "Participant emails (comma-separated, first is organizer)")
	cmd.Flags().IntVar(&duration, "duration", 30, "Meeting duration in minutes")
	cmd.Flags().StringVar(&title, "title", "", "Event title")
	cmd.Flags().StringVar(&description, "description", "", "Event description")
	cmd.Flags().StringVar(&location, "location", "", "Event location")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	registerConfigFlags(cmd, flags)

	return cmd
}

func newConfigUpdateCmd() *cobra.Command {
	var (
		name        string
		duration    int
		title       string
		description string
		jsonOutput  bool
	)
	flags := &configFlags{}

	cmd := &cobra.Command{
		Use:   "update <config-id>",
		Short: "Update a scheduler configuration",
		Long: `Update an existing scheduler configuration.

Use flags to set specific fields, or --file for full JSON update.
When both are provided, flags override file values.`,
		Example: `  # Update specific fields
  nylas scheduler configs update abc123 --name "Updated Name" --duration 60

  # Add buffer times
  nylas scheduler configs update abc123 --buffer-before 5 --buffer-after 10

  # From a JSON file
  nylas scheduler configs update abc123 --file update.json

  # File as base, override specific values
  nylas scheduler configs update abc123 --file update.json --duration 45`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateConfigFlags(flags); err != nil {
				return err
			}

			configID := args[0]
			req, err := buildUpdateRequest(cmd, flags, name, duration, title, description)
			if err != nil {
				return err
			}
			if err := validateUpdateRequest(req); err != nil {
				return err
			}

			_, err = common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				config, err := client.UpdateSchedulerConfiguration(ctx, configID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("configuration", err)
				}

				if jsonOutput {
					return struct{}{}, common.PrintJSON(config)
				}

				common.PrintUpdateSuccess("configuration", config.Name)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Configuration name")
	cmd.Flags().IntVar(&duration, "duration", 0, "Meeting duration in minutes")
	cmd.Flags().StringVar(&title, "title", "", "Event title")
	cmd.Flags().StringVar(&description, "description", "", "Event description")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	registerConfigFlags(cmd, flags)

	return cmd
}

func newConfigDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <config-id>",
		Short: "Delete a scheduler configuration",
		Long:  "Delete a scheduler configuration permanently.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Are you sure you want to delete configuration %s? (y/N): ", args[0])
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			configID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				if err := client.DeleteSchedulerConfiguration(ctx, configID); err != nil {
					return struct{}{}, common.WrapDeleteError("configuration", err)
				}

				_, _ = common.Green.Printf("✓ Deleted configuration: %s\n", configID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
