package email

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var limit int
	var unread bool
	var starred bool
	var from string
	var folder string
	var showID bool
	var all bool
	var allFolders bool
	var maxItems int
	var metadataPair string

	cmd := &cobra.Command{
		Use:   "list [grant-id]",
		Short: "List recent emails",
		Long: `List recent emails from your inbox. Use grant-id or the default account.

By default, only shows messages from INBOX. Use --folder to specify a different
folder, or --all-folders to show messages from all folders.

Use --all to fetch all messages (paginated automatically).
Use --max to limit total messages when using --all.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			grantID, err := common.GetGrantID(args)
			if err != nil {
				return err
			}

			ctx, cancel := common.CreateContext()
			defer cancel()

			params := &domain.MessageQueryParams{
				Limit: limit,
			}

			if cmd.Flags().Changed("unread") {
				params.Unread = &unread
			}
			if cmd.Flags().Changed("starred") {
				params.Starred = &starred
			}
			if from != "" {
				params.From = from
			}
			if metadataPair != "" {
				params.MetadataPair = metadataPair
			}

			// Default to INBOX unless --all-folders is set or specific folder is provided
			if folder != "" {
				// Resolve folder name to ID if needed (for Microsoft accounts)
				resolvedFolder, err := resolveFolderName(ctx, client, grantID, folder)
				if err != nil {
					// API error - warn user but continue with literal name
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not resolve folder '%s': %v\n", folder, err)
					params.In = []string{folder}
				} else if resolvedFolder != "" {
					params.In = []string{resolvedFolder}
				} else {
					// Folder not found by name, use literal
					params.In = []string{folder}
				}
			} else if !allFolders {
				// Try to find inbox folder ID (works for both Google and Microsoft)
				inboxID, err := resolveFolderName(ctx, client, grantID, "INBOX")
				if err != nil {
					// API error - warn but fallback to literal INBOX
					_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not resolve INBOX folder: %v\n", err)
					params.In = []string{"INBOX"}
				} else if inboxID != "" {
					params.In = []string{inboxID}
				} else {
					// Fallback to INBOX (works for Google)
					params.In = []string{"INBOX"}
				}
			}

			var messages []domain.Message

			if all {
				// Use pagination to fetch all messages
				pageSize := 50 // Optimal page size for API
				if limit > 0 && limit < pageSize {
					pageSize = limit
				}
				params.Limit = pageSize

				fetcher := func(ctx context.Context, cursor string) (common.PageResult[domain.Message], error) {
					params.PageToken = cursor
					resp, err := client.GetMessagesWithCursor(ctx, grantID, params)
					if err != nil {
						return common.PageResult[domain.Message]{}, err
					}
					return common.PageResult[domain.Message]{
						Data:       resp.Data,
						NextCursor: resp.Pagination.NextCursor,
					}, nil
				}

				config := common.DefaultPaginationConfig()
				config.PageSize = pageSize
				config.MaxItems = maxItems

				messages, err = common.FetchAllPages(ctx, config, fetcher)
				if err != nil {
					return common.WrapFetchError("messages", err)
				}
			} else {
				// Standard single-page fetch
				messages, err = client.GetMessagesWithParams(ctx, grantID, params)
				if err != nil {
					return common.WrapGetError("messages", err)
				}
			}

			if len(messages) == 0 {
				common.PrintEmptyState("messages")
				return nil
			}

			fmt.Printf("Found %d messages:\n\n", len(messages))
			for i, msg := range messages {
				printMessageSummaryWithID(msg, i+1, showID)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of messages to fetch (per page with --all)")
	cmd.Flags().BoolVarP(&unread, "unread", "u", false, "Only show unread messages")
	cmd.Flags().BoolVarP(&starred, "starred", "s", false, "Only show starred messages")
	cmd.Flags().StringVarP(&from, "from", "f", "", "Filter by sender email")
	cmd.Flags().StringVar(&folder, "folder", "", "Filter by folder (e.g., INBOX, SENT, TRASH, or folder ID)")
	cmd.Flags().BoolVar(&allFolders, "all-folders", false, "Show messages from all folders (default: INBOX only)")
	cmd.Flags().BoolVar(&showID, "id", false, "Show message IDs")
	cmd.Flags().BoolVarP(&all, "all", "a", false, "Fetch all messages (paginated)")
	cmd.Flags().IntVar(&maxItems, "max", 0, "Maximum messages to fetch with --all (0=unlimited)")
	cmd.Flags().StringVar(&metadataPair, "metadata", "", "Filter by metadata (format: key:value, only key1-key5 supported)")

	return cmd
}

// resolveFolderName looks up a folder by name and returns its ID.
// This is needed for Microsoft accounts which use folder IDs, not names like "INBOX".
// For Google accounts, this will just return the original name if no match is found.
func resolveFolderName(ctx context.Context, client ports.NylasClient, grantID, folderName string) (string, error) {
	folders, err := client.GetFolders(ctx, grantID)
	if err != nil {
		return "", err
	}

	// Normalize the search name
	searchName := strings.ToLower(folderName)

	// Common folder name mappings
	nameAliases := map[string][]string{
		"inbox":     {"inbox"},
		"sent":      {"sent", "sent items", "sent mail"},
		"drafts":    {"drafts", "draft"},
		"trash":     {"trash", "deleted items", "deleted"},
		"spam":      {"spam", "junk", "junk email"},
		"archive":   {"archive", "all mail"},
		"outbox":    {"outbox"},
		"scheduled": {"scheduled"},
	}

	// Find matching aliases for the search name
	var searchAliases []string
	for key, aliases := range nameAliases {
		if key == searchName || contains(aliases, searchName) {
			searchAliases = aliases
			break
		}
	}
	if searchAliases == nil {
		searchAliases = []string{searchName}
	}

	// Search for matching folder
	for _, f := range folders {
		folderNameLower := strings.ToLower(f.Name)
		for _, alias := range searchAliases {
			if folderNameLower == alias {
				return f.ID, nil
			}
		}
	}

	// No match found - return empty (caller will use original name)
	return "", nil
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
