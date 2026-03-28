package email

import (
	"context"
	"fmt"
	"io"
	"slices"
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
		Example: `  # List recent emails from inbox
  nylas email list

  # List only unread emails
  nylas email list --unread

  # List emails from a specific sender
  nylas email list --from boss@company.com

  # List emails from a specific folder
  nylas email list --folder SENT

  # Fetch all emails with pagination
  nylas email list --all --max 500`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := listOptions{
				limit:        limit,
				unread:       unread,
				starred:      starred,
				from:         from,
				folder:       folder,
				allFolders:   allFolders,
				all:          all,
				maxItems:     maxItems,
				metadataPair: metadataPair,
			}

			// Check if we should use structured output (JSON/YAML/quiet)
			if common.IsStructuredOutput(cmd) {
				return runListStructured(cmd, args, opts)
			}

			// Traditional formatted output
			_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				messages, err := fetchListMessages(ctx, cmd, client, grantID, opts)
				if err != nil {
					return struct{}{}, common.WrapFetchError("messages", err)
				}

				if len(messages) == 0 {
					common.PrintEmptyState("messages")
					return struct{}{}, nil
				}

				fmt.Printf("Found %d messages:\n\n", len(messages))
				for i, msg := range messages {
					printMessageSummaryWithID(msg, i+1, showID)
				}

				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "l", 10, "Number of messages to fetch (auto-paginates if >200)")
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

type listOptions struct {
	limit        int
	unread       bool
	starred      bool
	from         string
	folder       string
	allFolders   bool
	all          bool
	maxItems     int
	metadataPair string
}

func resolveListPagination(limit int, all bool, maxItems int) (int, int) {
	pag := common.SetupPagination(limit, all, maxItems)
	limit = pag.Limit

	switch pag.Mode {
	case common.PaginateSinglePage:
		return limit, -1 // fetchMessages uses -1 for single-page
	case common.PaginateAll:
		return limit, 0 // fetchMessages uses 0 for unlimited
	case common.PaginateWithCap:
		return limit, pag.MaxItems
	default:
		return limit, -1
	}
}

func fetchListMessages(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, grantID string, opts listOptions) ([]domain.Message, error) {
	limit, maxItems := resolveListPagination(opts.limit, opts.all, opts.maxItems)

	params := &domain.MessageQueryParams{
		Limit: limit,
	}

	if cmd.Flags().Changed("unread") {
		params.Unread = &opts.unread
	}
	if cmd.Flags().Changed("starred") {
		params.Starred = &opts.starred
	}
	if opts.from != "" {
		params.From = opts.from
	}
	if opts.metadataPair != "" {
		params.MetadataPair = opts.metadataPair
	}

	applyListFolderFilter(ctx, cmd.ErrOrStderr(), client, grantID, params, opts.folder, opts.allFolders)

	return fetchMessages(ctx, client, grantID, params, maxItems)
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
		if key == searchName || slices.Contains(aliases, searchName) {
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

func applyListFolderFilter(ctx context.Context, stderr io.Writer, client ports.NylasClient, grantID string, params *domain.MessageQueryParams, folder string, allFolders bool) {
	if folder != "" {
		// Resolve folder name to ID if needed (for Microsoft accounts)
		resolvedFolder, err := resolveFolderName(ctx, client, grantID, folder)
		if err != nil {
			// API error - warn user but continue with literal name
			_, _ = fmt.Fprintf(stderr, "Warning: could not resolve folder '%s': %v\n", folder, err)
			params.In = []string{folder}
			return
		}
		if resolvedFolder != "" {
			params.In = []string{resolvedFolder}
			return
		}

		// Folder not found by name, use literal
		params.In = []string{folder}
		return
	}

	if allFolders {
		return
	}

	// Try to find inbox folder ID (works for both Google and Microsoft)
	inboxID, err := resolveFolderName(ctx, client, grantID, "INBOX")
	if err != nil {
		// API error - warn but fallback to literal INBOX
		_, _ = fmt.Fprintf(stderr, "Warning: could not resolve INBOX folder: %v\n", err)
		params.In = []string{"INBOX"}
		return
	}
	if inboxID != "" {
		params.In = []string{inboxID}
		return
	}

	// Fallback to INBOX (works for Google)
	params.In = []string{"INBOX"}
}

// runListStructured handles structured output (JSON/YAML/quiet) for the list command.
func runListStructured(cmd *cobra.Command, args []string, opts listOptions) error {
	_, err := common.WithClient(args, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
		return struct{}{}, writeListStructured(ctx, cmd, client, grantID, opts)
	})
	return err
}

func writeListStructured(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, grantID string, opts listOptions) error {
	messages, err := fetchListMessages(ctx, cmd, client, grantID, opts)
	if err != nil {
		return common.WrapFetchError("messages", err)
	}

	out := common.GetOutputWriter(cmd)
	return out.Write(messages)
}
