package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeEmailSubscriptions(t *testing.T) {
	newer := time.Now().Add(-time.Hour)
	older := newer.Add(-24 * time.Hour)
	messages := []domain.Message{
		{
			ID:   "new-message",
			Date: newer,
			From: []domain.EmailParticipant{{Name: "News\n\x1b[31m", Email: "latest@example.com"}},
			Headers: []domain.Header{
				{Name: "list-id", Value: "Example News <news.example.com>"},
				{Name: "LIST-UNSUBSCRIBE", Value: "<mailto:leave@example.com>, <https://example.com/unsubscribe?token=secret>"},
				{Name: "List-Unsubscribe-Post", Value: "List-Unsubscribe=One-Click"},
				{Name: "Authentication-Results", Value: "mx.example; dkim=pass header.i=@example.com header.s=news"},
				{Name: "DKIM-Signature", Value: "v=1; d=example.com; s=news; h=from:list-unsubscribe:list-unsubscribe-post:to"},
			},
		},
		{
			ID:   "old-message",
			Date: older,
			From: []domain.EmailParticipant{{Name: "Old sender", Email: "latest@example.com"}},
			Headers: []domain.Header{
				{Name: "List-ID", Value: "news.example.com"},
				{Name: "List-Unsubscribe", Value: "<https://example.com/old>"},
				{Name: "List-Unsubscribe-Post", Value: "List-Unsubscribe=One-Click"},
				{Name: "Authentication-Results", Value: "mx.example; dkim=pass header.d=example.com header.s=news"},
				{Name: "DKIM-Signature", Value: "v=1; d=example.com; s=news; h=from:list-unsubscribe:list-unsubscribe-post:to"},
			},
		},
		{
			Date: older,
			From: []domain.EmailParticipant{{Name: "Deals", Email: "deals@example.com"}},
			Headers: []domain.Header{
				{Name: "List-Unsubscribe", Value: "<mailto:unsubscribe@example.com>"},
			},
		},
		{
			Date:    newer,
			From:    []domain.EmailParticipant{{Email: "ordinary@example.com"}},
			Headers: []domain.Header{{Name: "List-ID", Value: "ordinary.example.com"}},
		},
		{
			Date: newer,
			From: []domain.EmailParticipant{{Name: "'Support' via Limitless", Email: "limitless@nylas.com"}},
			Headers: []domain.Header{
				{Name: "List-ID", Value: "<limitless.nylas.com>"},
				{Name: "List-Post", Value: "<mailto:limitless@nylas.com>"},
				{Name: "List-Unsubscribe", Value: "<https://groups.example.com/unsubscribe>"},
			},
		},
	}

	got := summarizeEmailSubscriptions(messages)
	require.Len(t, got, 2)

	assert.Equal(t, 2, got[0].Messages)
	assert.Equal(t, "news.example.com", got[0].ListID)
	assert.Equal(t, "latest@example.com", got[0].Email)
	assert.Equal(t, "Web", got[0].Method)
	assert.Equal(t, newer, got[0].LastSeen)
	assert.ElementsMatch(t, []string{"new-message", "old-message"}, got[0].messageIDs)
	assert.Equal(t, "Email", got[1].Method)
	assert.Equal(t, "deals@example.com", got[1].QuietField())

	for _, r := range got[0].Sender {
		assert.False(t, unicode.IsControl(r), "sender contains terminal control character")
	}
	encoded, err := json.Marshal(got)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "token=secret")
	assert.NotContains(t, string(encoded), "unsubscribe@example.com")
}

func TestSummarizeEmailSubscriptionsSeparatesSendersSharingListID(t *testing.T) {
	got := summarizeEmailSubscriptions([]domain.Message{
		{
			ID: "marketing", Date: time.Now(),
			From: []domain.EmailParticipant{{Email: "marketing@example.com"}},
			Headers: []domain.Header{
				{Name: "List-ID", Value: "shared.example.com"},
				{Name: "List-Unsubscribe", Value: "<https://example.com/marketing>"},
			},
		},
		{
			ID: "alerts", Date: time.Now().Add(-time.Hour),
			From: []domain.EmailParticipant{{Email: "alerts@example.com"}},
			Headers: []domain.Header{
				{Name: "List-ID", Value: "shared.example.com"},
				{Name: "List-Unsubscribe", Value: "<https://example.com/alerts>"},
			},
		},
	})

	require.Len(t, got, 2)
	byEmail := make(map[string]emailSubscription, len(got))
	for _, subscription := range got {
		byEmail[subscription.Email] = subscription
	}
	assert.Equal(t, []string{"marketing"}, byEmail["marketing@example.com"].messageIDs)
	assert.Equal(t, []string{"alerts"}, byEmail["alerts@example.com"].messageIDs)
}

func TestIsPostableDiscussionList(t *testing.T) {
	assert.True(t, isPostableDiscussionList([]domain.Header{{Name: "List-Post", Value: "<mailto:list@example.com>"}}))
	assert.False(t, isPostableDiscussionList([]domain.Header{{Name: "List-Post", Value: " no "}}))
	assert.False(t, isPostableDiscussionList(nil))
}

func TestSummarizeEmailSubscriptionsKeepsAnnouncementLists(t *testing.T) {
	got := summarizeEmailSubscriptions([]domain.Message{{
		Date: time.Now(),
		From: []domain.EmailParticipant{{Email: "announcements@example.com"}},
		Headers: []domain.Header{
			{Name: "List-Post", Value: "NO"},
			{Name: "List-Unsubscribe", Value: "<https://example.com/unsubscribe>"},
		},
	}})

	require.Len(t, got, 1)
	assert.Equal(t, "announcements@example.com", got[0].Email)
}

func TestSummarizeEmailSubscriptionsRejectsConflictingDestinations(t *testing.T) {
	got := summarizeEmailSubscriptions([]domain.Message{
		{
			ID: "legitimate", Date: time.Now().Add(-time.Hour),
			From:    []domain.EmailParticipant{{Email: "news@example.com"}},
			Headers: []domain.Header{{Name: "List-Unsubscribe", Value: "<https://example.com/unsubscribe?token=one>"}},
		},
		{
			ID: "spoofed-newer", Date: time.Now(),
			From:    []domain.EmailParticipant{{Email: "news@example.com"}},
			Headers: []domain.Header{{Name: "List-Unsubscribe", Value: "<https://attacker.example/unsubscribe>"}},
		},
	})

	require.Len(t, got, 1)
	assert.Equal(t, "Unsupported", got[0].Method)
	assert.Empty(t, got[0].actionTarget)
	assert.ElementsMatch(t, []string{"legitimate", "spoofed-newer"}, got[0].messageIDs)
}

func TestSubscriptionAction(t *testing.T) {
	tests := []struct {
		name        string
		unsubscribe string
		want        string
	}{
		{name: "HTTPS opens for review", unsubscribe: "<https://example.com/u>", want: "Web"},
		{name: "web preferred when web and mailto are present", unsubscribe: "<mailto:leave@example.com>, <https://example.com/u>", want: "Web"},
		{name: "email", unsubscribe: "<mailto:leave@example.com>", want: "Email"},
		{name: "plain HTTP is unsupported", unsubscribe: "<http://example.com/u>", want: "Unsupported"},
		{name: "unsupported", unsubscribe: "unsubscribe.example.com", want: "Unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := []domain.Header{{Name: "List-Unsubscribe", Value: tt.unsubscribe}}
			method, target := subscriptionAction(headers)
			assert.Equal(t, tt.want, method)
			if tt.want == "Unsupported" {
				assert.Empty(t, target)
			} else {
				assert.NotEmpty(t, target)
			}
		})
	}
}

func TestParseAndSelectSubscriptions(t *testing.T) {
	selectors, err := parseSubscriptionSelectors([]string{"NEWS@example.com", "digest.example.com"})
	require.NoError(t, err)
	assert.Equal(t, "news@example.com", selectors[0].email)
	assert.Equal(t, "digest.example.com", selectors[1].list)
	unusual, err := parseSubscriptionSelectors([]string{"digest+weekly/list.example.com"})
	require.NoError(t, err)
	assert.Equal(t, "digest+weekly/list.example.com", unusual[0].list)

	subscriptions := []emailSubscription{
		{Email: "news@example.com", ListID: "daily.example.com"},
		{Email: "news@example.com", ListID: "weekly.example.com"},
		{Email: "digest@example.com", ListID: "digest.example.com"},
	}
	selected, missing, err := selectEmailSubscriptions(subscriptions, selectors, true)
	require.NoError(t, err)
	assert.Len(t, selected, 3)
	assert.Empty(t, missing)

	sharedList := []emailSubscription{
		{Email: "marketing@example.com", ListID: "shared.example.com", messageIDs: []string{"marketing"}},
		{Email: "alerts@example.com", ListID: "shared.example.com", messageIDs: []string{"alerts"}},
	}
	selected, missing, err = selectEmailSubscriptions(sharedList, []subscriptionSelector{{email: "marketing@example.com"}}, true)
	require.NoError(t, err)
	require.Len(t, selected, 1)
	assert.Equal(t, []string{"marketing"}, selected[0].messageIDs)
	assert.Empty(t, missing)

	selected, missing, err = selectEmailSubscriptions(sharedList, []subscriptionSelector{{list: "shared.example.com"}}, true)
	require.NoError(t, err)
	require.Len(t, selected, 2)
	assert.ElementsMatch(t, []string{"marketing", "alerts"}, []string{selected[0].messageIDs[0], selected[1].messageIDs[0]})
	assert.Empty(t, missing)

	for _, args := range [][]string{
		nil,
		{"bad selector"},
		{"bad@example"},
		{"list:id"},
		{".bad"},
		{"bad."},
		{"bad..id"},
		{strings.Repeat("x", 255)},
		make([]string, maxSubscriptionSelectors+1),
	} {
		_, err := parseSubscriptionSelectors(args)
		assert.Error(t, err, "selectors: %q", args)
	}

	_, _, err = selectEmailSubscriptions(subscriptions, []subscriptionSelector{{email: "missing@example.com"}}, true)
	assert.Error(t, err)

	selected, missing, err = selectEmailSubscriptions(subscriptions, []subscriptionSelector{{email: "missing@example.com"}, {list: "digest.example.com"}}, false)
	require.NoError(t, err)
	assert.Equal(t, []emailSubscription{subscriptions[2]}, selected)
	assert.Equal(t, []string{"missing@example.com"}, missing)
}

func TestParseMailtoUnsubscribe(t *testing.T) {
	req, err := parseMailtoUnsubscribe("mailto:leave@example.com?subject=unsubscribe&body=Please%20remove%20me")
	require.NoError(t, err)
	require.Len(t, req.To, 1)
	assert.Equal(t, "leave@example.com", req.To[0].Email)
	assert.Equal(t, "unsubscribe", req.Subject)
	assert.Equal(t, "Please remove me", req.Body)

	for _, target := range []string{
		"mailto:one@example.com,two@example.com",
		"mailto:leave@example.com?cc=other@example.com",
		"mailto:leave@example.com?subject=ok%0d%0aBcc:evil@example.com",
		"javascript:alert(1)",
	} {
		_, err := parseMailtoUnsubscribe(target)
		assert.Error(t, err, "target: %q", target)
	}
}

func TestValidateHTTPSUnsubscribeTarget(t *testing.T) {
	_, err := validateHTTPSUnsubscribeTarget("https://example.com/unsubscribe?token=secret")
	require.NoError(t, err)

	for _, target := range []string{
		"http://example.com/unsubscribe",
		"https://user:pass@example.com/unsubscribe",
		"https://example.com:8443/unsubscribe",
		"https://localhost/unsubscribe",
		"https://127.0.0.1/unsubscribe",
		"https://127.1/unsubscribe",
		"https://0x7f.1/unsubscribe",
		"https://0177.0.0.1/unsubscribe",
		"https://10.0.0.1/unsubscribe",
		"https://[::1]/unsubscribe",
		"https://example.com./unsubscribe",
	} {
		_, err := validateHTTPSUnsubscribeTarget(target)
		assert.Error(t, err, "target: %q", target)
	}

	for _, address := range []string{"1.1.1.1", "2606:4700:4700::1111"} {
		assert.True(t, isPublicUnsubscribeIP(netip.MustParseAddr(address)), address)
	}
	for _, address := range []string{"10.0.0.1", "100.64.0.1", "192.0.2.1", "192.88.99.1", "127.0.0.1", "fc00::1", "fe80::1", "2001::1", "2001:db8::1", "2002::1", "3fff::1"} {
		assert.False(t, isPublicUnsubscribeIP(netip.MustParseAddr(address)), address)
	}

}

func TestExecuteSubscriptionActions(t *testing.T) {
	subscriptions := []emailSubscription{
		{Sender: "Web", Email: "web@example.com", Method: "Web", Messages: 1, actionTarget: "https://example.com/web", messageIDs: []string{"m2"}},
		{Sender: "Email", Email: "email@example.com", Method: "Email", Messages: 1, actionTarget: "mailto:leave@example.com?subject=unsubscribe", messageIDs: []string{"m3"}},
	}
	var opened []string
	results, failures := executeSubscriptionActions(subscriptions, unsubscribeActions{
		openURL: func(target string) error {
			opened = append(opened, target)
			return nil
		},
	})

	assert.Zero(t, failures)
	assert.Len(t, results, 2)
	assert.Equal(t, []string{"https://example.com/web", "mailto:leave@example.com?subject=unsubscribe"}, opened)
}

func TestExecuteSubscriptionCleanupPermanentlyDeletes(t *testing.T) {
	client := nylas.NewMockClient()
	var deleted []string
	client.DeleteMessagePermanentlyFunc = func(_ context.Context, _, messageID string) error {
		deleted = append(deleted, messageID)
		return nil
	}
	results, err := executeSubscriptionCleanup(context.Background(), client, "grant-123", []emailSubscription{{
		Email: "one@example.com", Method: "Web", actionTarget: "https://example.com/one", messageIDs: []string{"m1", "m2"},
	}}, true, nil)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, []string{"m1", "m2"}, deleted)
	assert.Contains(t, results[0].Status, "Permanently deleted 2/2")
	assert.False(t, client.DeleteMessageCalled)
}

func TestExecuteSubscriptionActionsFailureDoesNotDelete(t *testing.T) {
	results, failures := executeSubscriptionActions([]emailSubscription{{
		Email: "one@example.com", Method: "Web", actionTarget: "https://example.com/one", messageIDs: []string{"m1"},
	}}, unsubscribeActions{openURL: func(string) error { return fmt.Errorf("failed") }})

	assert.Equal(t, 1, failures)
	require.Len(t, results, 1)
	assert.Equal(t, "opening unsubscribe page failed", results[0].Status)
}

func TestExecuteSubscriptionCleanupStopsAndReportsFirstError(t *testing.T) {
	client := nylas.NewMockClient()
	var attempted []string
	client.DeleteMessageFunc = func(_ context.Context, _, messageID string) error {
		attempted = append(attempted, messageID)
		if messageID == "m2" {
			return fmt.Errorf("delete failed")
		}
		return nil
	}
	progress := 0
	results, err := executeSubscriptionCleanup(context.Background(), client, "grant-123", []emailSubscription{{
		Email: "one@example.com", Method: "Web", actionTarget: "https://example.com/one", messageIDs: []string{"m1", "m2", "m3"},
	}}, false, func() { progress++ })

	require.Error(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Status, "Moved 1/3 to Trash; stopped after an error")
	assert.Equal(t, []string{"m1", "m2"}, attempted)
	assert.Equal(t, 1, progress)
}

func TestExecuteSubscriptionCleanupTreatsMissingMessagesAsComplete(t *testing.T) {
	client := nylas.NewMockClient()
	client.DeleteMessagePermanentlyFunc = func(_ context.Context, _, _ string) error {
		return &domain.APIError{StatusCode: http.StatusNotFound}
	}

	results, err := executeSubscriptionCleanup(context.Background(), client, "grant-123", []emailSubscription{{
		Email: "one@example.com", messageIDs: []string{"already-gone"},
	}}, true, nil)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Permanently deleted 1/1", results[0].Status)
}

func TestSubscriptionCleanupErrorGuidesHardDeleteSetup(t *testing.T) {
	err := subscriptionCleanupError(&domain.APIError{StatusCode: http.StatusForbidden}, true)
	assert.Contains(t, err.Error(), "Permanent email cleanup is not enabled")

	err = subscriptionCleanupError(fmt.Errorf("network failed"), true)
	assert.Contains(t, err.Error(), "failed to delete subscription messages")

	err = subscriptionCleanupError(&domain.APIError{StatusCode: http.StatusUnauthorized}, true)
	assert.NotContains(t, err.Error(), "Permanent email cleanup is not enabled")
}

func TestPlannedSubscriptionResultsDoNotExposeTargetsOrAct(t *testing.T) {
	subscription := emailSubscription{
		Email:        "news@example.com",
		Method:       "Web",
		Messages:     2,
		actionTarget: "https://example.com/unsubscribe?token=secret",
		messageIDs:   []string{"m1", "m2"},
	}
	results := plannedSubscriptionResults([]emailSubscription{subscription})
	require.Len(t, results, 1)
	assert.Equal(t, "Would open unsubscribe page", results[0].Status)
	assert.Equal(t, "example.com", results[0].Destination)

	encoded, err := json.Marshal(results)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "token=secret")
	assert.NotContains(t, string(encoded), "messageIDs")

	permanentResults := plannedCleanupResults([]emailSubscription{subscription}, true)
	assert.Contains(t, permanentResults[0].Status, "Would permanently delete 2")
}

func TestMessageHeaderCapsExternalValues(t *testing.T) {
	value := messageHeader([]domain.Header{{Name: "List-Unsubscribe", Value: strings.Repeat("x", maxSubscriptionHeaderBytes+1)}}, "list-unsubscribe")
	assert.Len(t, value, maxSubscriptionHeaderBytes)
}

func TestParseSubscriptionListOptions(t *testing.T) {
	opts, err := parseSubscriptionListOptions(500, "12w", true)
	require.NoError(t, err)
	assert.Equal(t, 500, opts.limit)
	assert.Equal(t, 12*7*24*time.Hour, opts.since)
	assert.True(t, opts.allFolders)

	for _, input := range []struct {
		limit int
		since string
	}{
		{limit: 0, since: "90d"},
		{limit: 10001, since: "90d"},
		{limit: 10, since: "0d"},
		{limit: 10, since: "1ns"},
		{limit: 10, since: "366d"},
		{limit: 10, since: strings.Repeat("1", 17)},
	} {
		_, err := parseSubscriptionListOptions(input.limit, input.since, false)
		assert.Error(t, err)
	}
}

func TestFetchEmailSubscriptions(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	client := &testListClient{MockClient: nylas.NewMockClient()}
	client.GetGrantFunc = func(context.Context, string) (*domain.Grant, error) {
		return &domain.Grant{Provider: domain.ProviderGoogle}, nil
	}
	client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
		assert.Equal(t, "grant-123", grantID)
		return []domain.Folder{{ID: "inbox-id", Name: "Inbox"}}, nil
	}
	client.getMessagesWithCursorFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
		assert.Equal(t, "grant-123", grantID)
		assert.Equal(t, "include_headers", params.Fields)
		assert.Equal(t, now.Add(-90*24*time.Hour).Unix(), params.ReceivedAfter)
		assert.Equal(t, []string{"inbox-id"}, params.In)
		assert.Equal(t, 200, params.Limit)
		return &domain.MessageListResponse{Data: []domain.Message{{
			Date:    now,
			From:    []domain.EmailParticipant{{Email: "news@example.com"}},
			Headers: []domain.Header{{Name: "List-Unsubscribe", Value: "<https://example.com/u>"}},
		}}}, nil
	}

	got, err := fetchEmailSubscriptions(context.Background(), newSubscriptionsListCmd(), client, "grant-123", subscriptionListOptions{
		limit: 500,
		since: 90 * 24 * time.Hour,
	}, now)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "news@example.com", got[0].Email)
}

func TestFetchEmailSubscriptionsAllFolders(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	client := &testListClient{MockClient: nylas.NewMockClient()}
	client.GetGrantFunc = func(context.Context, string) (*domain.Grant, error) {
		return &domain.Grant{Provider: domain.ProviderMicrosoft}, nil
	}
	client.getMessagesWithCursorFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
		assert.Nil(t, params.In)
		assert.Equal(t, 25, params.Limit)
		return &domain.MessageListResponse{}, nil
	}

	got, err := fetchEmailSubscriptions(context.Background(), newSubscriptionsListCmd(), client, "grant-123", subscriptionListOptions{
		limit:      25,
		since:      30 * 24 * time.Hour,
		allFolders: true,
	}, now)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestFetchEmailSubscriptionsRejectsUnsupportedProvider(t *testing.T) {
	client := &testListClient{MockClient: nylas.NewMockClient()}
	client.GetGrantFunc = func(context.Context, string) (*domain.Grant, error) {
		return &domain.Grant{Provider: domain.ProviderIMAP}, nil
	}

	_, err := fetchEmailSubscriptions(context.Background(), newSubscriptionsListCmd(), client, "grant-123", subscriptionListOptions{
		limit: 25,
		since: 30 * 24 * time.Hour,
	}, time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable for this provider")
	assert.False(t, client.GetMessagesWithParamsCalled)
}

func TestSubscriptionsCommand(t *testing.T) {
	cmd := newSubscriptionsCmd()
	assert.Equal(t, "subscriptions", cmd.Use)
	assert.Contains(t, cmd.Aliases, "subs")

	list, _, err := cmd.Find([]string{"list"})
	require.NoError(t, err)
	assert.Equal(t, "list", list.Use)
	assert.NotNil(t, list.Flags().Lookup("limit"))
	assert.Equal(t, "1000", list.Flags().Lookup("limit").DefValue)
	assert.NotNil(t, list.Flags().Lookup("since"))
	assert.NotContains(t, list.Use, "grant")
	assert.Error(t, list.Args(list, []string{"grant-123"}))

	unsubscribe, _, err := cmd.Find([]string{"unsubscribe"})
	require.NoError(t, err)
	assert.Equal(t, "unsubscribe <email-or-list-id>...", unsubscribe.Use)
	assert.NotContains(t, unsubscribe.Use, "grant")
	for _, flag := range []string{"limit", "since", "all-folders", "dry-run", "yes"} {
		assert.NotNil(t, unsubscribe.Flags().Lookup(flag), flag)
	}
	assert.Nil(t, unsubscribe.Flags().Lookup("delete-emails"))
	assert.Nil(t, unsubscribe.Flags().Lookup("permanent"))
	assert.Error(t, unsubscribe.Args(unsubscribe, nil))
	assert.NoError(t, unsubscribe.Args(unsubscribe, []string{"news@example.com"}))

	cleanup, _, err := cmd.Find([]string{"cleanup"})
	require.NoError(t, err)
	assert.Equal(t, "cleanup <email-or-list-id>...", cleanup.Use)
	for _, flag := range []string{"limit", "since", "all-folders", "permanent", "dry-run", "yes"} {
		assert.NotNil(t, cleanup.Flags().Lookup(flag), flag)
	}
}

func TestSubscriptionsCleanupCommandPermanentlyDeletesWithoutUnsubscribing(t *testing.T) {
	var folderGets, messageGets, deletes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test":
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "grant-test", "provider": "google"}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/folders":
			folderGets++
			if folderGets == 1 {
				assert.Empty(t, r.URL.Query().Get("page_token"))
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data":        []map[string]any{{"id": "custom-trash", "name": "Trash"}},
					"next_cursor": "cursor-2",
				})
				return
			}
			assert.Equal(t, "cursor-2", r.URL.Query().Get("page_token"))
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{"id": "trash-id", "name": "Papierkorb", "attributes": []string{"\\Trash"}},
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/v3/grants/grant-test/messages":
			messageGets++
			assert.Equal(t, "trash-id", r.URL.Query().Get("in"))
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{
				"id": "m1", "date": time.Now().Unix(),
				"from":    []map[string]string{{"email": "news@example.com"}},
				"headers": []map[string]string{{"name": "List-Unsubscribe", "value": "<mailto:leave@example.com>"}},
			}}})
		case r.Method == http.MethodDelete && r.URL.Path == "/v3/grants/grant-test/messages/m1":
			deletes++
			assert.Equal(t, "true", r.URL.Query().Get("hard_delete"))
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s request to %s", r.Method, r.URL.String())
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	defer server.Close()

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("NYLAS_API_KEY", "test-api-key")
	t.Setenv("NYLAS_GRANT_ID", "grant-test")
	t.Setenv("NYLAS_API_BASE_URL", server.URL)

	root := &cobra.Command{Use: "test", SilenceErrors: true, SilenceUsage: true}
	common.AddOutputFlags(root)
	root.AddCommand(newSubscriptionsCmd())
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"subscriptions", "cleanup", "news@example.com", "missing@example.com", "--permanent", "--yes", "--json"})

	require.NoError(t, root.Execute(), stderr.String())
	assert.Equal(t, 2, folderGets)
	assert.Equal(t, 1, messageGets)
	assert.Equal(t, 1, deletes)
	assert.Contains(t, stdout.String(), "Permanently deleted 1/1")
	assert.Contains(t, stderr.String(), `subscription "missing@example.com" was not found in Trash with --since 90d and --limit 1000`)
}
