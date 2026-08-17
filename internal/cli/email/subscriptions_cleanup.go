package email

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

func newSubscriptionsCleanupCmd() *cobra.Command {
	var limit int
	var since string
	var allFolders bool
	var permanent bool
	var dryRun bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "cleanup <email-or-list-id>...",
		Short: "Delete messages from selected subscriptions",
		Long: `Delete messages from selected subscriptions in the active account.

By default, matching inbox messages move to Trash. --permanent irreversibly
deletes matching messages from Trash; combine it with --all-folders only when
you intend to delete matching mail everywhere. This command never unsubscribes.`,
		Example: `  # Preview moving matching inbox messages to Trash
  nylas email subscriptions cleanup news@example.com --dry-run

  # Move matching messages from all folders to Trash
  nylas email subscriptions cleanup news@example.com --all-folders

  # Permanently remove matching messages already in Trash
  nylas email subscriptions cleanup news@example.com --permanent

  # Permanently remove matching messages from every folder
  nylas email subscriptions cleanup news@example.com --permanent --all-folders`,
		Args: func(_ *cobra.Command, args []string) error {
			_, err := parseSubscriptionSelectors(args)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := parseSubscriptionListOptions(limit, since, allFolders)
			if err != nil {
				return err
			}
			if permanent && !allFolders {
				opts.folder = "TRASH"
			}
			selectors, err := parseSubscriptionSelectors(args)
			if err != nil {
				return err
			}
			if !dryRun && common.IsStructuredOutput(cmd) && !yes {
				return common.NewInputError("structured cleanup output requires --yes or --dry-run")
			}

			_, err = common.WithClient(nil, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				subscriptions, fetchErr := fetchEmailSubscriptions(ctx, cmd, client, grantID, opts, time.Now())
				if fetchErr != nil {
					return struct{}{}, common.WrapFetchError("subscriptions", fetchErr)
				}
				selected, missing, selectErr := selectEmailSubscriptions(subscriptions, selectors, false)
				if selectErr != nil {
					return struct{}{}, selectErr
				}
				scope := "Inbox"
				advice := "Increase --since or --limit, or use --all-folders, if needed."
				if permanent {
					scope = "Trash"
				}
				if allFolders {
					scope = "all folders"
					advice = "Increase --since or --limit if needed."
				}
				for _, selector := range missing {
					_, _ = fmt.Fprintf(
						cmd.ErrOrStderr(),
						"Warning: subscription %q was not found in %s with --since %s and --limit %d; it may already be clean. %s\n",
						selector, scope, since, limit, advice,
					)
				}
				if len(selected) == 0 {
					if common.IsStructuredOutput(cmd) {
						return struct{}{}, common.WriteListWithColumns(cmd, []subscriptionActionResult{}, subscriptionActionColumns())
					}
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No matching messages found. Cleanup may already be complete.")
					return struct{}{}, nil
				}

				if dryRun {
					return struct{}{}, common.WriteListWithColumns(cmd, plannedCleanupResults(selected, permanent), subscriptionActionColumns())
				}

				if !yes {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Selected subscriptions:")
					if writeErr := common.WriteListWithColumns(cmd, selected, subscriptionColumns()); writeErr != nil {
						return struct{}{}, writeErr
					}
					prompt := fmt.Sprintf("Move %d matching message(s) to Trash?", selectedMessageCount(selected))
					if permanent {
						scope := "from Trash"
						if allFolders {
							scope = "from all folders"
						}
						prompt = fmt.Sprintf("Permanently delete %d matching message(s) %s? This cannot be undone.", selectedMessageCount(selected), scope)
					}
					if !common.Confirm(prompt, false) {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
						return struct{}{}, nil
					}
				}

				counter := common.NewCounter("Cleaning messages")
				results, cleanupErr := executeSubscriptionCleanup(context.WithoutCancel(ctx), client, grantID, selected, permanent, counter.Increment)
				counter.Finish()
				if writeErr := common.WriteListWithColumns(cmd, results, subscriptionActionColumns()); writeErr != nil {
					return struct{}{}, writeErr
				}
				if cleanupErr != nil {
					return struct{}{}, subscriptionCleanupError(cleanupErr, permanent)
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", defaultSubscriptionScanLimit, "Maximum messages to scan")
	cmd.Flags().StringVar(&since, "since", "90d", "Only scan messages received within this duration (for example, 30d or 12w)")
	cmd.Flags().BoolVar(&allFolders, "all-folders", false, "Scan all folders instead of only the inbox (or Trash with --permanent)")
	cmd.Flags().BoolVar(&permanent, "permanent", false, "Permanently delete matches; scans Trash unless --all-folders is set")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview cleanup actions without making changes")
	common.AddYesFlag(cmd, &yes)

	return cmd
}

func plannedCleanupResults(subscriptions []emailSubscription, permanent bool) []subscriptionActionResult {
	results := make([]subscriptionActionResult, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		status := fmt.Sprintf("Would move %d to Trash", len(subscription.messageIDs))
		if permanent {
			status = fmt.Sprintf("Would permanently delete %d", len(subscription.messageIDs))
		}
		results = append(results, newSubscriptionActionResult(subscription, status))
	}
	return results
}

func executeSubscriptionCleanup(
	ctx context.Context,
	client ports.NylasClient,
	grantID string,
	subscriptions []emailSubscription,
	permanent bool,
	progress func(),
) ([]subscriptionActionResult, error) {
	deleteMessage := client.DeleteMessage
	statusFormat := "Moved %d/%d to Trash"
	if permanent {
		deleteMessage = client.DeleteMessagePermanently
		statusFormat = "Permanently deleted %d/%d"
	}

	results := make([]subscriptionActionResult, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		deleted := 0
		for _, messageID := range subscription.messageIDs {
			if messageID == "" || len(messageID) > 1024 {
				results = append(results, newSubscriptionActionResult(subscription, fmt.Sprintf(statusFormat+"; stopped after an error", deleted, len(subscription.messageIDs))))
				return results, fmt.Errorf("invalid message identifier")
			}
			err := deleteMessage(ctx, grantID, messageID)
			if err != nil && !apiErrorHasStatus(err, 404) {
				results = append(results, newSubscriptionActionResult(subscription, fmt.Sprintf(statusFormat+"; stopped after an error", deleted, len(subscription.messageIDs))))
				return results, err
			}
			deleted++
			if progress != nil {
				progress()
			}
		}
		results = append(results, newSubscriptionActionResult(subscription, fmt.Sprintf(statusFormat, deleted, len(subscription.messageIDs))))
	}
	return results, nil
}

func subscriptionCleanupError(err error, permanent bool) error {
	if permanent && (apiErrorHasStatus(err, 400) || apiErrorHasStatus(err, 403)) {
		return common.NewUserError(
			"Permanent email cleanup is not enabled for this grant",
			"Enable hard delete in Nylas Dashboard > Customizations > API, verify the provider write scope, then re-authenticate the grant",
		)
	}
	return common.WrapDeleteError("subscription messages", err)
}

func apiErrorHasStatus(err error, status int) bool {
	var apiErr *domain.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}
