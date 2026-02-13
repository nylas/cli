// channels.go provides channel listing and management commands for Slack.

package slack

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// newChannelsCmd creates the channels command for managing Slack channels.
func newChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "channels",
		Aliases: []string{"ch", "channel"},
		Short:   "Manage Slack channels",
		Long:    `Commands for listing and managing Slack channels.`,
	}

	cmd.AddCommand(newChannelListCmd())
	cmd.AddCommand(newChannelInfoCmd())

	return cmd
}

// newChannelListCmd creates the list subcommand for listing channels.
func newChannelListCmd() *cobra.Command {
	var (
		channelTypes    []string
		excludeArchived bool
		limit           int
		showID          bool
		teamID          string
		fetchAll        bool
		allWorkspace    bool
		createdAfter    string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Slack channels you are a member of",
		Long: `List Slack channels you are a member of, including public, private, DMs, and group DMs.

Examples:
  # List your channels
  nylas slack channels list

  # List channels with IDs
  nylas slack channels list --id

  # List channels created in the last 24 hours
  nylas slack channels list --created-after 24h

  # List channels created in the last 7 days
  nylas slack channels list --created-after 7d

  # List all workspace channels (slower, may hit rate limits)
  nylas slack channels list --all-workspace

  # List only public channels
  nylas slack channels list --type public_channel

  # Exclude archived channels
  nylas slack channels list --exclude-archived`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getSlackClientOrError()
			if err != nil {
				return err
			}

			// Parse created-after duration if provided
			var createdAfterTime time.Time
			if createdAfter != "" {
				duration, parseErr := common.ParseDuration(createdAfter)
				if parseErr != nil {
					return common.NewUserError(
						"invalid duration format",
						"Use format like: 24h, 7d, 2w (hours, days, weeks)",
					)
				}
				createdAfterTime = time.Now().Add(-duration)
				// When filtering by date, we need to fetch all to filter client-side
				fetchAll = true
			}

			// Use longer timeout when fetching all channels
			ctx, cancel := common.CreateContext()
			if fetchAll {
				cancel() // Cancel the default context
				ctx, cancel = common.CreateLongContext()
			}
			defer cancel()

			// Auto-detect team_id from auth if not provided (needed for Enterprise Grid)
			if teamID == "" {
				authResp, authErr := client.TestAuth(ctx)
				if authErr == nil && authResp.TeamID != "" {
					teamID = authResp.TeamID
				}
			}

			// Create pagination fetcher
			fetcher := func(ctx context.Context, cursor string) (common.PageResult[domain.SlackChannel], error) {
				params := &domain.SlackChannelQueryParams{
					Types:           channelTypes,
					ExcludeArchived: excludeArchived,
					Limit:           limit,
					TeamID:          teamID,
					Cursor:          cursor,
				}

				var resp *domain.SlackChannelListResponse
				var fetchErr error
				if allWorkspace {
					resp, fetchErr = client.ListChannels(ctx, params)
				} else {
					resp, fetchErr = client.ListMyChannels(ctx, params)
				}
				if fetchErr != nil {
					return common.PageResult[domain.SlackChannel]{}, fetchErr
				}
				return common.PageResult[domain.SlackChannel]{
					Data:       resp.Channels,
					NextCursor: resp.NextCursor,
				}, nil
			}

			config := common.DefaultPaginationConfig()
			config.PageSize = limit
			if !fetchAll {
				config.MaxPages = 1
				config.ShowProgress = false
			}

			allChannels, err := common.FetchAllPages(ctx, config, fetcher)
			if err != nil {
				return common.WrapListError("channels", err)
			}

			// Filter by creation date if specified
			if !createdAfterTime.IsZero() {
				filtered := make([]domain.SlackChannel, 0)
				for _, ch := range allChannels {
					if ch.Created.After(createdAfterTime) {
						filtered = append(filtered, ch)
					}
				}
				allChannels = filtered
			}

			if len(allChannels) == 0 {
				common.PrintEmptyState("channels")
				return nil
			}

			// Handle structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				out := common.GetOutputWriter(cmd)
				return out.Write(allChannels)
			}

			printChannels(allChannels, showID)

			return nil
		},
	}

	cmd.Flags().StringSliceVar(&channelTypes, "type", nil, "Channel types: public_channel, private_channel, mpim, im")
	cmd.Flags().BoolVar(&excludeArchived, "exclude-archived", false, "Exclude archived channels")
	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Channels per page (max 1000)")
	cmd.Flags().BoolVar(&showID, "id", false, "Show channel IDs")
	cmd.Flags().StringVar(&teamID, "team", "", "Team/workspace ID (auto-detected for Enterprise Grid)")
	cmd.Flags().BoolVar(&fetchAll, "all", false, "Fetch all channels (paginate through all pages)")
	cmd.Flags().BoolVar(&allWorkspace, "all-workspace", false, "List all workspace channels (slower, may hit rate limits)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Show channels created after duration (e.g., 24h, 7d, 2w)")

	return cmd
}

// printChannels formats and prints a list of Slack channels to stdout.
func printChannels(channels []domain.SlackChannel, showID bool) {
	cyan := common.Cyan
	dim := common.Dim
	yellow := common.Yellow

	for _, ch := range channels {
		name := ch.ChannelDisplayName()

		if ch.IsPrivate && !ch.IsIM && !ch.IsMPIM {
			_, _ = yellow.Print("🔒 ")
		}

		_, _ = cyan.Print(name)

		if showID {
			_, _ = dim.Printf(" [%s]", ch.ID)
		}

		if ch.MemberCount > 0 {
			_, _ = dim.Printf(" (%d members)", ch.MemberCount)
		}

		typeLabel := ch.ChannelType()
		if typeLabel != "public" {
			_, _ = dim.Printf(" [%s]", typeLabel)
		}

		if ch.IsArchived {
			_, _ = dim.Print(" (archived)")
		}

		fmt.Println()

		if ch.Purpose != "" {
			_, _ = dim.Printf("  %s\n", common.Truncate(ch.Purpose, 60))
		}
	}
}
