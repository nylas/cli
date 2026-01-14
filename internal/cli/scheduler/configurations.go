package scheduler

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
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
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			configs, err := client.ListSchedulerConfigurations(ctx)
			if err != nil {
				return common.WrapListError("configurations", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(configs)
			}

			if len(configs) == 0 {
				common.PrintEmptyState("configurations")
				return nil
			}

			fmt.Printf("Found %d configuration(s):\n\n", len(configs))

			table := common.NewTable("NAME", "ID", "SLUG", "PARTICIPANTS")
			for _, cfg := range configs {
				participantCount := fmt.Sprintf("%d", len(cfg.Participants))
				table.AddRow(common.Cyan.Sprint(cfg.Name), cfg.ID, common.Green.Sprint(cfg.Slug), participantCount)
			}
			table.Render()

			return nil
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
			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			config, err := client.GetSchedulerConfiguration(ctx, args[0])
			if err != nil {
				return common.WrapGetError("configuration", err)
			}

			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(config)
			}

			_, _ = common.Bold.Printf("Configuration: %s\n", config.Name)
			fmt.Printf("  ID: %s\n", common.Cyan.Sprint(config.ID))
			fmt.Printf("  Slug: %s\n", common.Green.Sprint(config.Slug))
			fmt.Printf("  Duration: %d minutes\n", config.Availability.DurationMinutes)

			if len(config.Participants) > 0 {
				fmt.Printf("\nParticipants (%d):\n", len(config.Participants))
				for i, p := range config.Participants {
					fmt.Printf("  %d. %s <%s>", i+1, p.Name, p.Email)
					if p.IsOrganizer {
						fmt.Printf(" %s", common.Green.Sprint("(Organizer)"))
					}
					fmt.Println()
				}
			}

			fmt.Printf("\nEvent Booking:\n")
			fmt.Printf("  Title: %s\n", config.EventBooking.Title)
			if config.EventBooking.Description != "" {
				fmt.Printf("  Description: %s\n", config.EventBooking.Description)
			}
			if config.EventBooking.Location != "" {
				fmt.Printf("  Location: %s\n", config.EventBooking.Location)
			}

			return nil
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
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a scheduler configuration",
		Long:  "Create a new scheduler configuration (meeting type).",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate participants
			if len(participants) == 0 {
				return common.NewUserError("at least one participant email is required", "Use --participant to specify email addresses")
			}
			for i, p := range participants {
				p = strings.TrimSpace(p)
				if p == "" {
					return fmt.Errorf("participant email at position %d cannot be empty", i+1)
				}
				participants[i] = p
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Build participants list
			var participantsList []domain.ConfigurationParticipant
			for i, email := range participants {
				participantsList = append(participantsList, domain.ConfigurationParticipant{
					Email:       email,
					IsOrganizer: i == 0, // First participant is organizer
				})
			}

			req := &domain.CreateSchedulerConfigurationRequest{
				Name:         name,
				Participants: participantsList,
				Availability: domain.AvailabilityRules{
					DurationMinutes: duration,
				},
				EventBooking: domain.EventBooking{
					Title:       title,
					Description: description,
					Location:    location,
				},
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			config, err := client.CreateSchedulerConfiguration(ctx, req)
			if err != nil {
				return common.WrapCreateError("configuration", err)
			}

			_, _ = common.Green.Printf("✓ Created configuration: %s\n", config.Name)
			fmt.Printf("  ID: %s\n", config.ID)
			if config.Slug != "" {
				fmt.Printf("  Slug: %s\n", config.Slug)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Configuration name (required)")
	cmd.Flags().StringSliceVar(&participants, "participants", []string{}, "Participant emails (comma-separated, first is organizer)")
	cmd.Flags().IntVar(&duration, "duration", 30, "Meeting duration in minutes")
	cmd.Flags().StringVar(&title, "title", "", "Event title (required)")
	cmd.Flags().StringVar(&description, "description", "", "Event description")
	cmd.Flags().StringVar(&location, "location", "", "Event location")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("participants")
	_ = cmd.MarkFlagRequired("title")

	return cmd
}

func newConfigUpdateCmd() *cobra.Command {
	var (
		name        string
		duration    int
		title       string
		description string
	)

	cmd := &cobra.Command{
		Use:   "update <config-id>",
		Short: "Update a scheduler configuration",
		Long:  "Update an existing scheduler configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			req := &domain.UpdateSchedulerConfigurationRequest{}

			if name != "" {
				req.Name = &name
			}

			if cmd.Flags().Changed("duration") {
				req.Availability = &domain.AvailabilityRules{
					DurationMinutes: duration,
				}
			}

			// Only set EventBooking fields that were explicitly changed
			if cmd.Flags().Changed("title") || cmd.Flags().Changed("description") {
				eventBooking := &domain.EventBooking{}
				if cmd.Flags().Changed("title") {
					eventBooking.Title = title
				}
				if cmd.Flags().Changed("description") {
					eventBooking.Description = description
				}
				req.EventBooking = eventBooking
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			config, err := client.UpdateSchedulerConfiguration(ctx, args[0], req)
			if err != nil {
				return common.WrapUpdateError("configuration", err)
			}

			_, _ = common.Green.Printf("✓ Updated configuration: %s\n", config.Name)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Configuration name")
	cmd.Flags().IntVar(&duration, "duration", 0, "Meeting duration in minutes")
	cmd.Flags().StringVar(&title, "title", "", "Event title")
	cmd.Flags().StringVar(&description, "description", "", "Event description")

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

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			if err := client.DeleteSchedulerConfiguration(ctx, args[0]); err != nil {
				return common.WrapDeleteError("configuration", err)
			}

			_, _ = common.Green.Printf("✓ Deleted configuration: %s\n", args[0])

			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
