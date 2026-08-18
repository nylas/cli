package email

import (
	"context"
	"fmt"
	"net/mail"
	"net/netip"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

const (
	maxSubscriptionSelectors = 50
)

var blockedUnsubscribePrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("3fff::/20"),
}

type subscriptionSelector struct {
	email string
	list  string
}

type subscriptionActionResult struct {
	Sender      string `json:"-"`
	Name        string `json:"name,omitempty"`
	Email       string `json:"email,omitempty"`
	ListID      string `json:"list_id,omitempty"`
	Method      string `json:"method"`
	Destination string `json:"destination,omitempty"`
	Messages    int    `json:"messages"`
	Status      string `json:"status"`
}

func (r subscriptionActionResult) QuietField() string {
	if r.Email != "" {
		return r.Email
	}
	return r.ListID
}

type unsubscribeActions struct {
	openURL func(string) error
}

func newSubscriptionsUnsubscribeCmd() *cobra.Command {
	var limit int
	var since string
	var allFolders bool
	var dryRun bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "unsubscribe <email-or-list-id>...",
		Short: "Open unsubscribe actions for selected mailing lists",
		Long: `Open unsubscribe actions for selected senders or List-IDs in the active account.

The command reviews recent messages to find a consistent unsubscribe action.
Finish each action in the browser or mail composer it opens. The command never
deletes existing mail; use 'nylas email subscriptions cleanup' afterward.`,
		Example: `  # Preview one subscription without making changes
  nylas email subscriptions unsubscribe noreply@medium.com --dry-run

  # Unsubscribe from several senders
  nylas email subscriptions unsubscribe news@example.com updates.example.com

  # Skip the CLI confirmation, then finish the opened action
  nylas email subscriptions unsubscribe news@example.com --yes --json`,
		Args: func(_ *cobra.Command, args []string) error {
			_, err := parseSubscriptionSelectors(args)
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts, err := parseSubscriptionListOptions(limit, since, allFolders)
			if err != nil {
				return err
			}
			selectors, err := parseSubscriptionSelectors(args)
			if err != nil {
				return err
			}
			if !dryRun && common.IsStructuredOutput(cmd) && !yes {
				return common.NewInputError("structured unsubscribe output requires --yes or --dry-run")
			}

			_, err = common.WithClient(nil, func(ctx context.Context, client ports.NylasClient, grantID string) (struct{}, error) {
				subscriptions, fetchErr := fetchEmailSubscriptions(ctx, cmd, client, grantID, opts, time.Now())
				if fetchErr != nil {
					return struct{}{}, common.WrapFetchError("subscriptions", fetchErr)
				}
				selected, _, selectErr := selectEmailSubscriptions(subscriptions, selectors, true)
				if selectErr != nil {
					return struct{}{}, selectErr
				}

				if dryRun {
					return struct{}{}, common.WriteListWithColumns(cmd, plannedSubscriptionResults(selected), subscriptionActionColumns())
				}

				if !yes {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Selected subscriptions:")
					if writeErr := common.WriteListWithColumns(cmd, plannedSubscriptionResults(selected), subscriptionActionColumns()); writeErr != nil {
						return struct{}{}, writeErr
					}
					if !common.Confirm(fmt.Sprintf("Open unsubscribe actions for %d subscription(s)?", len(selected)), false) {
						_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
						return struct{}{}, nil
					}
				}

				browserClient := browser.NewDefaultBrowser()
				results, failures := executeSubscriptionActions(selected, unsubscribeActions{
					openURL: browserClient.Open,
				})
				if writeErr := common.WriteListWithColumns(cmd, results, subscriptionActionColumns()); writeErr != nil {
					return struct{}{}, writeErr
				}
				if failures > 0 {
					return struct{}{}, common.NewUserError(
						fmt.Sprintf("%d subscription operation(s) did not complete", failures),
						"Review the result table and retry failed subscriptions",
					)
				}
				return struct{}{}, nil
			})
			return err
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", defaultSubscriptionScanLimit, "Maximum messages to scan")
	cmd.Flags().StringVar(&since, "since", "90d", "Only scan messages received within this duration (for example, 30d or 12w)")
	cmd.Flags().BoolVar(&allFolders, "all-folders", false, "Scan all folders instead of only the inbox")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview unsubscribe actions without making changes")
	common.AddYesFlag(cmd, &yes)

	return cmd
}

func parseSubscriptionSelectors(args []string) ([]subscriptionSelector, error) {
	if len(args) < 1 || len(args) > maxSubscriptionSelectors {
		return nil, common.NewInputError("provide between 1 and 50 subscription emails or List-IDs")
	}

	selectors := make([]subscriptionSelector, 0, len(args))
	for _, arg := range args {
		value := strings.TrimSpace(arg)
		if value != arg || value == "" || len(value) > 254 {
			return nil, common.NewInputError("invalid subscription selector")
		}
		if strings.Contains(value, "@") {
			address, err := parseSubscriptionEmail(value)
			if err != nil {
				return nil, common.NewInputError("subscription email selector is invalid")
			}
			selectors = append(selectors, subscriptionSelector{email: strings.ToLower(address.Address)})
			continue
		}
		if len(value) > 160 || !validListIDSelector(value) {
			return nil, common.NewInputError("subscription List-ID selector is invalid")
		}
		selectors = append(selectors, subscriptionSelector{list: strings.ToLower(value)})
	}
	return selectors, nil
}

func validListIDSelector(value string) bool {
	runes := []rune(value)
	if len(runes) == 0 || runes[0] == '.' || runes[len(runes)-1] == '.' || strings.Contains(value, "..") {
		return false
	}
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune("!#$%&'*+-/=?^_`{|}~.", r) {
			continue
		}
		return false
	}
	return true
}

func selectEmailSubscriptions(subscriptions []emailSubscription, selectors []subscriptionSelector, requireAll bool) ([]emailSubscription, []string, error) {
	selected := make([]emailSubscription, 0, len(selectors))
	missing := make([]string, 0)
	seen := make(map[string]bool)
	for _, selector := range selectors {
		matched := false
		for _, subscription := range subscriptions {
			match := selector.email != "" && strings.EqualFold(selector.email, subscription.Email)
			match = match || selector.list != "" && strings.EqualFold(selector.list, subscription.ListID)
			if !match {
				continue
			}
			matched = true
			key := subscriptionIdentityKey(subscription.ListID, subscription.Email)
			if !seen[key] {
				selected = append(selected, subscription)
				seen[key] = true
			}
		}
		if !matched && requireAll {
			value := selector.email
			if value == "" {
				value = selector.list
			}
			return nil, nil, common.NewUserError(
				fmt.Sprintf("subscription not found: %s", value),
				"Run 'nylas email subscriptions list' or expand --since/--all-folders",
			)
		}
		if !matched {
			value := selector.email
			if value == "" {
				value = selector.list
			}
			missing = append(missing, value)
		}
	}
	return selected, missing, nil
}

func plannedSubscriptionResults(subscriptions []emailSubscription) []subscriptionActionResult {
	results := make([]subscriptionActionResult, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		status := map[string]string{
			"Web":         "Would open unsubscribe page",
			"Email":       "Would open unsubscribe email draft",
			"Unsupported": "No supported unsubscribe action",
		}[subscription.Method]
		results = append(results, newSubscriptionActionResult(subscription, status))
	}
	return results
}

func executeSubscriptionActions(
	subscriptions []emailSubscription,
	actions unsubscribeActions,
) ([]subscriptionActionResult, int) {
	results := make([]subscriptionActionResult, 0, len(subscriptions))
	failures := 0
	for _, subscription := range subscriptions {
		status, err := executeSubscriptionAction(subscription, actions)
		if err != nil {
			failures++
			results = append(results, newSubscriptionActionResult(subscription, err.Error()))
			continue
		}
		results = append(results, newSubscriptionActionResult(subscription, status))
	}
	return results, failures
}

func executeSubscriptionAction(
	subscription emailSubscription,
	actions unsubscribeActions,
) (string, error) {
	switch subscription.Method {
	case "Web":
		if actions.openURL == nil || actions.openURL(subscription.actionTarget) != nil {
			return "", fmt.Errorf("opening unsubscribe page failed")
		}
		return "Opened unsubscribe page", nil
	case "Email":
		if _, err := parseMailtoUnsubscribe(subscription.actionTarget); err != nil {
			return "", fmt.Errorf("invalid email unsubscribe action")
		}
		if actions.openURL == nil || actions.openURL(subscription.actionTarget) != nil {
			return "", fmt.Errorf("opening unsubscribe email draft failed")
		}
		return "Opened unsubscribe email draft", nil
	default:
		return "", fmt.Errorf("unsupported unsubscribe action")
	}
}

func newSubscriptionActionResult(subscription emailSubscription, status string) subscriptionActionResult {
	return subscriptionActionResult{
		Sender:      subscription.Sender,
		Name:        subscription.Name,
		Email:       subscription.Email,
		ListID:      subscription.ListID,
		Method:      subscription.Method,
		Destination: subscriptionDestination(subscription.Method, subscription.actionTarget),
		Messages:    subscription.Messages,
		Status:      status,
	}
}

func subscriptionActionColumns() []ports.Column {
	return []ports.Column{
		{Header: "SENDER", Field: "Sender", Width: 38},
		{Header: "METHOD", Field: "Method", Width: 12},
		{Header: "DESTINATION", Field: "Destination", Width: 28},
		{Header: "MESSAGES", Field: "Messages", Width: 0},
		{Header: "STATUS", Field: "Status", Width: 48},
	}
}

func selectedMessageCount(subscriptions []emailSubscription) int {
	total := 0
	for _, subscription := range subscriptions {
		total += len(subscription.messageIDs)
	}
	return total
}

func subscriptionAction(headers []domain.Header) (string, string) {
	targets := parseUnsubscribeTargets(messageHeader(headers, "List-Unsubscribe"))
	var webTarget, emailTarget string
	for _, target := range targets {
		if webTarget == "" {
			if _, err := validateHTTPSUnsubscribeTarget(target); err == nil {
				webTarget = target
			}
		}
		if emailTarget == "" && strings.HasPrefix(strings.ToLower(target), "mailto:") {
			if _, err := parseMailtoUnsubscribe(target); err == nil {
				emailTarget = target
			}
		}
	}

	if webTarget != "" {
		return "Web", webTarget
	}
	if emailTarget != "" {
		return "Email", emailTarget
	}
	return "Unsupported", ""
}

func subscriptionActionKey(method, target string) string {
	destination := strings.ToLower(subscriptionDestination(method, target))
	if destination == "" {
		return ""
	}
	if method == "Email" {
		return "email:" + destination
	}
	return "web:" + destination
}

func subscriptionDestination(method, target string) string {
	switch method {
	case "Web":
		u, err := validateHTTPSUnsubscribeTarget(target)
		if err == nil {
			return strings.ToLower(u.Hostname())
		}
	case "Email":
		req, err := parseMailtoUnsubscribe(target)
		if err == nil && len(req.To) == 1 {
			return strings.ToLower(req.To[0].Email)
		}
	}
	return ""
}

func parseUnsubscribeTargets(value string) []string {
	targets := make([]string, 0, 2)
	for len(targets) < 10 {
		open := strings.IndexByte(value, '<')
		if open < 0 {
			break
		}
		value = value[open+1:]
		close := strings.IndexByte(value, '>')
		if close < 0 {
			break
		}
		target := strings.TrimSpace(value[:close])
		if target != "" {
			targets = append(targets, target)
		}
		value = value[close+1:]
	}
	return targets
}

func parseMailtoUnsubscribe(target string) (*domain.SendMessageRequest, error) {
	if len(target) > maxSubscriptionHeaderBytes {
		return nil, fmt.Errorf("mailto target is too long")
	}
	u, err := url.Parse(target)
	if err != nil || !strings.EqualFold(u.Scheme, "mailto") || u.Host != "" || u.Fragment != "" {
		return nil, fmt.Errorf("invalid mailto target")
	}
	addressValue := u.Opaque
	if addressValue == "" {
		addressValue = strings.TrimPrefix(u.Path, "/")
	}
	addressValue, err = url.PathUnescape(addressValue)
	if err != nil || len(addressValue) > 254 {
		return nil, fmt.Errorf("invalid mailto address")
	}
	address, err := parseSubscriptionEmail(addressValue)
	if err != nil {
		return nil, fmt.Errorf("invalid mailto address")
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(query) > 2 {
		return nil, fmt.Errorf("invalid mailto query")
	}
	for key, values := range query {
		if (key != "subject" && key != "body") || len(values) != 1 {
			return nil, fmt.Errorf("unsupported mailto query")
		}
	}
	subject, body := query.Get("subject"), query.Get("body")
	if len(subject) > 998 || strings.ContainsAny(subject, "\r\n") || containsUnsafeMailtoControl(subject) || len(body) > 16*1024 || containsUnsafeMailtoControl(body) {
		return nil, fmt.Errorf("unsafe mailto content")
	}

	return &domain.SendMessageRequest{
		To:      []domain.EmailParticipant{{Email: address.Address}},
		Subject: subject,
		Body:    body,
	}, nil
}

func parseSubscriptionEmail(value string) (*mail.Address, error) {
	address, err := mail.ParseAddress(value)
	if err != nil || address.Name != "" || !strings.EqualFold(address.Address, value) {
		return nil, fmt.Errorf("invalid email address")
	}
	at := strings.LastIndexByte(address.Address, '@')
	if at < 1 || at == len(address.Address)-1 || !validUnsubscribeHostname(address.Address[at+1:]) {
		return nil, fmt.Errorf("invalid email address")
	}
	return address, nil
}

func containsUnsafeMailtoControl(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) && r != '\r' && r != '\n' && r != '\t' {
			return true
		}
	}
	return false
}

func validateHTTPSUnsubscribeTarget(target string) (*url.URL, error) {
	if len(target) == 0 || len(target) > maxSubscriptionHeaderBytes {
		return nil, fmt.Errorf("invalid unsubscribe URL")
	}
	u, err := url.ParseRequestURI(target)
	if err != nil || !u.IsAbs() || !strings.EqualFold(u.Scheme, "https") || u.User != nil || u.Hostname() == "" || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") {
		return nil, fmt.Errorf("invalid unsubscribe URL")
	}
	host := u.Hostname()
	if address, parseErr := netip.ParseAddr(host); parseErr == nil {
		if address.Zone() != "" || !isPublicUnsubscribeIP(address) {
			return nil, fmt.Errorf("unsafe unsubscribe URL")
		}
	} else if !validUnsubscribeHostname(host) {
		return nil, fmt.Errorf("invalid unsubscribe hostname")
	}
	return u, nil
}

func validUnsubscribeHostname(host string) bool {
	if len(host) > 253 || !strings.Contains(host, ".") || strings.HasSuffix(host, ".") || isIPv4NumberHostname(host) {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func isIPv4NumberHostname(host string) bool {
	parts := strings.Split(host, ".")
	if len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		digits := part
		hex := strings.HasPrefix(part, "0x") || strings.HasPrefix(part, "0X")
		if hex {
			digits = part[2:]
		}
		if digits == "" {
			return false
		}
		for _, r := range digits {
			if r >= '0' && r <= '9' || hex && (r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
				continue
			}
			return false
		}
	}
	return true
}

func isPublicUnsubscribeIP(address netip.Addr) bool {
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range blockedUnsubscribePrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}
