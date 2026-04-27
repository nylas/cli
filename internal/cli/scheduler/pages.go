package scheduler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newPagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pages",
		Aliases: []string{"page"},
		Short:   "Manage scheduler pages",
		Long:    "Manage scheduler pages (public booking pages).",
	}

	cmd.AddCommand(newPageListCmd())
	cmd.AddCommand(newPageShowCmd())
	cmd.AddCommand(newPageCreateCmd())
	cmd.AddCommand(newPageUpdateCmd())
	cmd.AddCommand(newPageDeleteCmd())

	return cmd
}

func newPageListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List scheduler pages",
		Long:    "List all scheduler pages.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				pages, err := client.ListSchedulerPages(ctx)
				if err != nil {
					return struct{}{}, common.WrapListError("pages", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(pages)
				}

				if len(pages) == 0 {
					common.PrintEmptyState("pages")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d page(s):\n\n", len(pages))

				table := common.NewTable("NAME", "ID", "SLUG", "CONFIG ID")
				for _, page := range pages {
					table.AddRow(common.Cyan.Sprint(page.Name), page.ID, common.Green.Sprint(page.Slug), page.ConfigurationID)
				}
				table.Render()

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newPageShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <page-id>",
		Short: "Show scheduler page details",
		Long:  "Show detailed information about a specific scheduler page.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				page, err := client.GetSchedulerPage(ctx, pageID)
				if err != nil {
					return struct{}{}, common.WrapGetError("page", err)
				}

				if common.IsJSON(cmd) {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(page)
				}

				_, _ = common.Bold.Printf("Scheduler Page: %s\n", page.Name)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(page.ID))
				fmt.Printf("  Slug: %s\n", common.Green.Sprint(page.Slug))
				fmt.Printf("  Configuration ID: %s\n", page.ConfigurationID)
				if page.URL != "" {
					fmt.Printf("  URL: %s\n", common.Cyan.Sprint(page.URL))
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	return cmd
}

func newPageCreateCmd() *cobra.Command {
	var (
		name     string
		configID string
		slug     string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a scheduler page",
		Long:  "Create a new scheduler page (public booking page).",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.CreateSchedulerPageRequest{
					Name:            name,
					ConfigurationID: configID,
					Slug:            slug,
				}

				page, err := client.CreateSchedulerPage(ctx, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("page", err)
				}

				_, _ = common.Green.Printf("✓ Created scheduler page: %s\n", page.Name)
				fmt.Printf("  ID: %s\n", common.Cyan.Sprint(page.ID))
				if page.Slug != "" {
					fmt.Printf("  Slug: %s\n", common.Green.Sprint(page.Slug))
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Page name (required)")
	cmd.Flags().StringVar(&configID, "config-id", "", "Configuration ID (required)")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("config-id")

	return cmd
}

func newPageUpdateCmd() *cobra.Command {
	var (
		name string
		slug string
	)

	cmd := &cobra.Command{
		Use:   "update <page-id>",
		Short: "Update a scheduler page",
		Long:  "Update an existing scheduler page.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pageID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				req := &domain.UpdateSchedulerPageRequest{}

				if name != "" {
					req.Name = &name
				}

				if slug != "" {
					req.Slug = &slug
				}

				page, err := client.UpdateSchedulerPage(ctx, pageID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("page", err)
				}

				common.PrintUpdateSuccess("scheduler page", page.Name)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Page name")
	cmd.Flags().StringVar(&slug, "slug", "", "URL slug")

	return cmd
}

func newPageDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <page-id>",
		Short: "Delete a scheduler page",
		Long:  "Delete a scheduler page permanently.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Are you sure you want to delete page %s? (y/N): ", args[0])
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			pageID := args[0]
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				if err := client.DeleteSchedulerPage(ctx, pageID); err != nil {
					return struct{}{}, common.WrapDeleteError("page", err)
				}

				_, _ = common.Green.Printf("✓ Deleted scheduler page: %s\n", pageID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
