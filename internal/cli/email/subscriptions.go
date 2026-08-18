package email

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

const (
	defaultSubscriptionScanLimit = 1000
	maxSubscriptionScanLimit     = 10000
	maxSubscriptionScanAge       = 365 * 24 * time.Hour
	maxSubscriptionHeaderBytes   = 8192
)

type subscriptionListOptions struct {
	limit          int
	since          time.Duration
	allFolders     bool
	folder         string
	folderRequired bool
}

type emailSubscription struct {
	Sender       string    `json:"-"`
	Name         string    `json:"name,omitempty"`
	Email        string    `json:"email,omitempty"`
	ListID       string    `json:"list_id,omitempty"`
	List         string    `json:"-"`
	Messages     int       `json:"messages"`
	LastSeen     time.Time `json:"last_seen"`
	LastSeenAgo  string    `json:"-"`
	Method       string    `json:"method"`
	actionTarget string
	actionKey    string
	actionSeen   time.Time
	actionUnsafe bool
	messageIDs   []string
}

func (s emailSubscription) QuietField() string {
	if s.Email != "" {
		return s.Email
	}
	return s.ListID
}

func newSubscriptionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subscriptions",
		Aliases: []string{"subs"},
		Short:   "Discover email subscriptions",
		Long: `Discover mailing-list subscriptions from the active account.

Subscriptions are detected from standard List-Unsubscribe headers. Discovery is
read-only, omits postable discussion lists, and never follows or prints
unsubscribe links.`,
		Args: cobra.NoArgs,
	}

	cmd.AddCommand(newSubscriptionsListCmd(), newSubscriptionsUnsubscribeCmd(), newSubscriptionsCleanupCmd())
	return cmd
}

func newSubscriptionsListCmd() *cobra.Command {
	var limit int
	var since string
	var allFolders bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List detected email subscriptions",
		Long: `List subscriptions detected in recent messages for the active account.

By default, the last 90 days of inbox messages are scanned. Use --all-folders
to include archived and automatically filed messages.`,
		Example: `  # List subscriptions for the active account
  nylas email subscriptions list

  # Scan a longer period and include archived mail
  nylas email subscriptions list --since 180d --all-folders

  # Produce machine-readable output
  nylas email subscriptions list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts, err := parseSubscriptionListOptions(limit, since, allFolders)
			if err != nil {
				return err
			}

			_, err = common.WithClient(nil, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				subscriptions, fetchErr := fetchEmailSubscriptions(ctx, cmd, client, grantID, opts, time.Now())
				if fetchErr != nil {
					return struct{}{}, common.WrapFetchError("subscriptions", fetchErr)
				}

				if len(subscriptions) == 0 && !common.IsStructuredOutput(cmd) {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No subscriptions found.")
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Try a longer --since period or --all-folders.")
					return struct{}{}, nil
				}

				return struct{}{}, common.WriteListWithColumns(cmd, subscriptions, subscriptionColumns())
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", defaultSubscriptionScanLimit, "Maximum messages to scan")
	cmd.Flags().StringVar(&since, "since", "90d", "Only scan messages received within this duration (for example, 30d or 12w)")
	cmd.Flags().BoolVar(&allFolders, "all-folders", false, "Scan all folders instead of only the inbox")

	return cmd
}

func parseSubscriptionListOptions(limit int, since string, allFolders bool) (subscriptionListOptions, error) {
	if limit < 1 || limit > maxSubscriptionScanLimit {
		return subscriptionListOptions{}, common.NewInputError("--limit must be between 1 and 10000")
	}
	if len(since) > 16 {
		return subscriptionListOptions{}, common.NewInputError("invalid --since duration")
	}

	duration, err := common.ParseDuration(since)
	if err != nil || duration < time.Second || duration > maxSubscriptionScanAge {
		return subscriptionListOptions{}, common.NewInputError("--since must be a duration between 1 second and 365 days")
	}

	return subscriptionListOptions{limit: limit, since: duration, allFolders: allFolders}, nil
}

func fetchEmailSubscriptions(ctx context.Context, cmd *cobra.Command, client ports.NylasClient, grantID string, opts subscriptionListOptions, now time.Time) ([]emailSubscription, error) {
	grant, err := client.GetGrant(ctx, grantID)
	if err != nil {
		return nil, err
	}
	if grant == nil || (grant.Provider != domain.ProviderGoogle && grant.Provider != domain.ProviderMicrosoft) {
		return nil, common.NewUserError(
			"Email subscription discovery is unavailable for this provider",
			"Use a Google or Microsoft grant; other providers do not expose List-Unsubscribe headers through Nylas",
		)
	}

	params := &domain.MessageQueryParams{
		Limit:         opts.limit,
		ReceivedAfter: now.Add(-opts.since).Unix(),
		Fields:        "include_headers",
	}
	if err := applyListFolderFilter(ctx, cmd.ErrOrStderr(), client, grantID, params, opts.folder, opts.allFolders, opts.folderRequired); err != nil {
		return nil, err
	}

	messages, err := fetchMessages(ctx, client, grantID, params, opts.limit)
	if err != nil {
		return nil, err
	}
	return summarizeEmailSubscriptions(messages), nil
}

func summarizeEmailSubscriptions(messages []domain.Message) []emailSubscription {
	byKey := make(map[string]emailSubscription)
	for _, message := range messages {
		if isPostableDiscussionList(message.Headers) {
			continue
		}
		unsubscribe := messageHeader(message.Headers, "List-Unsubscribe")
		if unsubscribe == "" {
			continue
		}

		listID := parseListID(messageHeader(message.Headers, "List-ID"))
		name, email := "", ""
		if len(message.From) > 0 {
			name = safeSubscriptionText(message.From[0].Name, 100)
			email = safeSubscriptionText(message.From[0].Email, 254)
		}

		key := subscriptionIdentityKey(listID, email)
		if key == "sender:" {
			continue
		}

		method, target := subscriptionAction(message.Headers)
		actionKey := subscriptionActionKey(method, target)
		current, exists := byKey[key]
		current.Messages++
		if message.ID != "" {
			current.messageIDs = append(current.messageIDs, message.ID)
		}
		if actionKey != "" {
			switch {
			case current.actionKey == "":
				current.actionKey = actionKey
			case current.actionKey != actionKey:
				current.actionUnsafe = true
			}
			if !current.actionUnsafe && (current.actionSeen.IsZero() || message.Date.After(current.actionSeen)) {
				current.Method = method
				current.actionTarget = target
				current.actionSeen = message.Date
			}
			if current.actionKey == actionKey && (current.Method == "Web" || method == "Web") {
				current.Method = "Web"
			}
		} else if !exists {
			current.Method = "Unsupported"
		}
		if !exists || message.Date.After(current.LastSeen) {
			current.Name = name
			current.Email = email
			current.ListID = listID
			current.LastSeen = message.Date
		}
		if current.actionUnsafe {
			current.Method = "Unsupported"
			current.actionTarget = ""
		}
		byKey[key] = current
	}

	subscriptions := make([]emailSubscription, 0, len(byKey))
	for _, subscription := range byKey {
		subscription.Sender = subscription.Email
		if subscription.Name != "" && subscription.Email != "" {
			subscription.Sender = fmt.Sprintf("%s <%s>", subscription.Name, subscription.Email)
		} else if subscription.Name != "" {
			subscription.Sender = subscription.Name
		}
		subscription.List = subscription.ListID
		if subscription.List == "" {
			subscription.List = "—"
		}
		subscription.LastSeenAgo = common.FormatTimeAgo(subscription.LastSeen)
		subscriptions = append(subscriptions, subscription)
	}

	sort.Slice(subscriptions, func(i, j int) bool {
		if subscriptions[i].Messages != subscriptions[j].Messages {
			return subscriptions[i].Messages > subscriptions[j].Messages
		}
		if !subscriptions[i].LastSeen.Equal(subscriptions[j].LastSeen) {
			return subscriptions[i].LastSeen.After(subscriptions[j].LastSeen)
		}
		return subscriptions[i].Sender < subscriptions[j].Sender
	})
	return subscriptions
}

func subscriptionIdentityKey(listID, email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	if listID == "" {
		return "sender:" + email
	}
	return "list:" + strings.ToLower(strings.TrimSpace(listID)) + "\x00sender:" + email
}

func isPostableDiscussionList(headers []domain.Header) bool {
	listPost := messageHeader(headers, "List-Post")
	return listPost != "" && !strings.EqualFold(strings.TrimSpace(listPost), "NO")
}

func subscriptionColumns() []ports.Column {
	return []ports.Column{
		{Header: "SENDER", Field: "Sender", Width: 40},
		{Header: "LIST", Field: "List", Width: 32},
		{Header: "MESSAGES", Field: "Messages", Width: 0},
		{Header: "LAST SEEN", Field: "LastSeenAgo", Width: 16},
		{Header: "METHOD", Field: "Method", Width: 12},
	}
}

func messageHeader(headers []domain.Header, name string) string {
	for _, header := range headers {
		if strings.EqualFold(strings.TrimSpace(header.Name), name) {
			value := strings.TrimSpace(header.Value)
			if len(value) > maxSubscriptionHeaderBytes {
				value = value[:maxSubscriptionHeaderBytes]
			}
			return value
		}
	}
	return ""
}

func parseListID(value string) string {
	if open := strings.LastIndex(value, "<"); open >= 0 {
		if close := strings.Index(value[open+1:], ">"); close >= 0 {
			value = value[open+1 : open+1+close]
		}
	}
	return safeSubscriptionText(strings.Trim(value, "<> \t"), 160)
}

func safeSubscriptionText(value string, maxRunes int) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, strings.TrimSpace(value))

	runes := []rune(value)
	if len(runes) > maxRunes {
		runes = runes[:maxRunes]
	}
	return strings.TrimSpace(string(runes))
}
