package admin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newCallbackURIsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "callback-uris",
		Aliases: []string{"callbacks", "cb"},
		Short:   "Manage application callback URIs",
		Long:    "Manage OAuth callback URIs for your Nylas application.",
	}

	cmd.AddCommand(newCallbackURIListCmd())
	cmd.AddCommand(newCallbackURIShowCmd())
	cmd.AddCommand(newCallbackURICreateCmd())
	cmd.AddCommand(newCallbackURIUpdateCmd())
	cmd.AddCommand(newCallbackURIDeleteCmd())

	return cmd
}

func newCallbackURIListCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List callback URIs",
		Long:    "List all callback URIs for your application.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				uris, err := client.ListCallbackURIs(ctx)
				if err != nil {
					return struct{}{}, common.WrapListError("callback URIs", err)
				}

				if jsonOutput {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(uris)
				}

				if len(uris) == 0 {
					common.PrintEmptyState("callback URIs")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d callback URI(s):\n\n", len(uris))

				table := common.NewTable("ID", "URL", "PLATFORM")
				for _, uri := range uris {
					platform := uri.Platform
					if platform == "" {
						platform = "-"
					}
					table.AddRow(common.Cyan.Sprint(uri.ID), uri.URL, platform)
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

func newCallbackURIShowCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "show <uri-id>",
		Short: "Show callback URI details",
		Long:  "Show detailed information about a specific callback URI.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uriID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				uri, err := client.GetCallbackURI(ctx, uriID)
				if err != nil {
					return struct{}{}, common.WrapGetError("callback URI", err)
				}

				if jsonOutput {
					return struct{}{}, json.NewEncoder(cmd.OutOrStdout()).Encode(uri)
				}

				_, _ = common.Bold.Println("Callback URI Details")
				fmt.Printf("  ID:       %s\n", common.Cyan.Sprint(uri.ID))
				fmt.Printf("  URL:      %s\n", uri.URL)
				fmt.Printf("  Platform: %s\n", uri.Platform)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newCallbackURICreateCmd() *cobra.Command {
	var (
		url      string
		platform string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a callback URI",
		Long: `Create a new callback URI for your application.

Examples:
  nylas admin callback-uris create --url http://localhost:9007/callback
  nylas admin callback-uris create --url https://myapp.com/oauth/callback --platform web`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.CreateCallbackURIRequest{
					URL:      url,
					Platform: platform,
				}

				uri, err := client.CreateCallbackURI(ctx, req)
				if err != nil {
					return struct{}{}, common.WrapCreateError("callback URI", err)
				}

				_, _ = common.Green.Println("✓ Created callback URI")
				fmt.Printf("  ID:       %s\n", common.Cyan.Sprint(uri.ID))
				fmt.Printf("  URL:      %s\n", uri.URL)
				fmt.Printf("  Platform: %s\n", uri.Platform)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Callback URL (required)")
	cmd.Flags().StringVar(&platform, "platform", "web", "Platform (web, ios, android)")

	_ = cmd.MarkFlagRequired("url")

	return cmd
}

func newCallbackURIUpdateCmd() *cobra.Command {
	var (
		url      string
		platform string
	)

	cmd := &cobra.Command{
		Use:   "update <uri-id>",
		Short: "Update a callback URI",
		Long: `Update an existing callback URI.

Examples:
  nylas admin callback-uris update <uri-id> --url https://myapp.com/new-callback
  nylas admin callback-uris update <uri-id> --platform ios`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uriID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				req := &domain.UpdateCallbackURIRequest{}
				if url != "" {
					req.URL = &url
				}
				if platform != "" {
					req.Platform = &platform
				}

				uri, err := client.UpdateCallbackURI(ctx, uriID, req)
				if err != nil {
					return struct{}{}, common.WrapUpdateError("callback URI", err)
				}

				_, _ = common.Green.Printf("✓ Updated callback URI: %s\n", uri.ID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Callback URL")
	cmd.Flags().StringVar(&platform, "platform", "", "Platform (web, ios, android)")

	return cmd
}

func newCallbackURIDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uri-id>",
		Short: "Delete a callback URI",
		Long:  "Delete a callback URI permanently.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("Are you sure you want to delete callback URI %s? (y/N): ", args[0])
				var confirm string
				_, _ = fmt.Scanln(&confirm)
				if confirm != "y" && confirm != "Y" {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			uriID := args[0]
			_, err := common.WithClientNoGrant(func(ctx context.Context, client ports.NylasClient) (struct{}, error) {
				if err := client.DeleteCallbackURI(ctx, uriID); err != nil {
					return struct{}{}, common.WrapDeleteError("callback URI", err)
				}

				_, _ = common.Green.Printf("✓ Deleted callback URI: %s\n", uriID)

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}
