package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakePollClient struct {
	ports.MessageClient
	pages    map[string][]domain.MessageListResponse
	errs     map[string]error
	grantIDs []string
	params   []domain.MessageQueryParams
}

func (f *fakePollClient) GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
	f.grantIDs = append(f.grantIDs, grantID)
	pageToken := ""
	if params != nil {
		pageToken = params.PageToken
		f.params = append(f.params, *params)
	}
	if err := f.errs[pageToken]; err != nil {
		return nil, err
	}
	if len(f.pages[pageToken]) == 0 {
		return &domain.MessageListResponse{}, nil
	}
	resp := f.pages[pageToken][0]
	f.pages[pageToken] = f.pages[pageToken][1:]
	if params != nil {
		resp.Data = filterMessagesAfter(resp.Data, params.ReceivedAfter)
	}
	return &resp, nil
}

type notifyCall struct {
	method string
	params any
}

func TestMessagePoller_PollOnce_DrainsPagesEmitsAllNewMessages(t *testing.T) {
	client := &fakePollClient{
		pages: map[string][]domain.MessageListResponse{
			"": {{
				Data:       pollMessages(175, 126),
				Pagination: domain.Pagination{NextCursor: "page-2", HasMore: true},
			}},
			"page-2": {{
				Data:       pollMessages(125, 106),
				Pagination: domain.Pagination{NextCursor: "page-3", HasMore: true},
			}},
			"page-3": {{
				Data: pollMessages(105, 101),
			}},
		},
	}

	var calls []notifyCall
	poller := NewMessagePoller(client, "grant-123", 100, func(method string, params any) error {
		calls = append(calls, notifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}

	var wantIDs []string
	for ts := int64(101); ts <= 175; ts++ {
		wantIDs = append(wantIDs, fmt.Sprintf("msg-%03d", ts))
	}
	assertNotifyIDs(t, calls, wantIDs)
	assertUniqueNotifyIDs(t, calls)
	assertQueries(t, client.params, []wantQuery{
		{receivedAfter: 99, pageToken: ""},
		{receivedAfter: 99, pageToken: "page-2"},
		{receivedAfter: 99, pageToken: "page-3"},
	})
	if !reflect.DeepEqual(client.grantIDs, []string{"grant-123", "grant-123", "grant-123"}) {
		t.Fatalf("grant IDs = %#v, want grant-123 for every page", client.grantIDs)
	}
	if poller.cursor != 175 {
		t.Fatalf("cursor = %d, want 175", poller.cursor)
	}
}

func TestMessagePoller_PollOnce_EmitsSameSecondBoundaryMessageOnce(t *testing.T) {
	base := int64(100)
	client := &fakePollClient{
		pages: map[string][]domain.MessageListResponse{
			"": {
				{Data: []domain.Message{pollMessage("boundary-a", 105)}},
				{Data: []domain.Message{
					pollMessage("boundary-a", 105),
					pollMessage("boundary-b", 105),
					pollMessage("newer-c", 106),
				}},
				{Data: []domain.Message{
					pollMessage("boundary-a", 105),
					pollMessage("boundary-b", 105),
					pollMessage("newer-c", 106),
				}},
			},
		},
	}

	var calls []notifyCall
	poller := NewMessagePoller(client, "grant-123", base, func(method string, params any) error {
		calls = append(calls, notifyCall{method: method, params: params})
		return nil
	})

	for i := 0; i < 3; i++ {
		if err := poller.PollOnce(context.Background()); err != nil {
			t.Fatalf("PollOnce() #%d error = %v", i+1, err)
		}
	}

	assertNotifyIDs(t, calls, []string{"boundary-a", "boundary-b", "newer-c"})
	assertUniqueNotifyIDs(t, calls)
	assertQueries(t, client.params, []wantQuery{
		{receivedAfter: 99, pageToken: ""},
		{receivedAfter: 104, pageToken: ""},
		{receivedAfter: 105, pageToken: ""},
	})
}

func TestMessagePoller_PollOnce_NoNewSecondPollEmitsNothing(t *testing.T) {
	client := &fakePollClient{
		pages: map[string][]domain.MessageListResponse{
			"": {
				{Data: []domain.Message{{
					ID:      "new-1",
					GrantID: "grant-123",
					Subject: "Hello",
					Snippet: "First line",
					From:    []domain.EmailParticipant{{Name: "Ada", Email: "ada@example.com"}},
					Date:    time.Unix(101, 0),
					Unread:  true,
					Folders: []string{"inbox"},
				}}},
				{},
			},
		},
	}

	var calls []notifyCall
	poller := NewMessagePoller(client, "grant-123", 100, func(method string, params any) error {
		calls = append(calls, notifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() second error = %v", err)
	}

	assertNotifyIDs(t, calls, []string{"new-1"})
	gotPayload, ok := calls[0].params.(messageReceivedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want messageReceivedPayload", calls[0].params)
	}
	wantPayload := messageReceivedPayload{
		ID:      "new-1",
		GrantID: "grant-123",
		Subject: "Hello",
		Snippet: "First line",
		From:    []domain.EmailParticipant{{Name: "Ada", Email: "ada@example.com"}},
		Date:    101,
		Unread:  true,
		Folders: []string{"inbox"},
	}
	if !reflect.DeepEqual(gotPayload, wantPayload) {
		t.Fatalf("payload = %#v, want %#v", gotPayload, wantPayload)
	}

	payloadJSON, err := json.Marshal(gotPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	wantJSON := `{"id":"new-1","grant_id":"grant-123","subject":"Hello","snippet":"First line","from":[{"name":"Ada","email":"ada@example.com"}],"date":101,"unread":true,"folders":["inbox"]}`
	if string(payloadJSON) != wantJSON {
		t.Fatalf("payload JSON = %s, want %s", payloadJSON, wantJSON)
	}
	assertQueries(t, client.params, []wantQuery{
		{receivedAfter: 99, pageToken: ""},
		{receivedAfter: 100, pageToken: ""},
	})
	if poller.cursor != 101 {
		t.Fatalf("cursor = %d, want 101", poller.cursor)
	}
}

func TestMessagePoller_PollOnce_ReturnsClientErrorFromLaterPage(t *testing.T) {
	clientErr := errors.New("api unavailable")
	client := &fakePollClient{
		pages: map[string][]domain.MessageListResponse{
			"": {{
				Data:       []domain.Message{pollMessage("new", 101)},
				Pagination: domain.Pagination{NextCursor: "page-2", HasMore: true},
			}},
		},
		errs: map[string]error{"page-2": clientErr},
	}
	called := false
	poller := NewMessagePoller(client, "grant-123", 100, func(method string, params any) error {
		called = true
		return nil
	})

	err := poller.PollOnce(context.Background())
	if !errors.Is(err, clientErr) {
		t.Fatalf("PollOnce() error = %v, want %v", err, clientErr)
	}
	if called {
		t.Fatal("notify was called on client error")
	}
	if poller.cursor != 100 {
		t.Fatalf("cursor = %d, want 100", poller.cursor)
	}
}

func TestMessagePoller_PollOnce_ReturnsErrorWhenPageDrainTruncates(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "more pages after cap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages := make(map[string][]domain.MessageListResponse, maxMessagePollPages)
			pageToken := ""
			for page := range maxMessagePollPages {
				nextToken := fmt.Sprintf("page-%d", page+1)
				pages[pageToken] = []domain.MessageListResponse{{
					Data:       []domain.Message{pollMessage(fmt.Sprintf("msg-%03d", page), int64(101+page))},
					Pagination: domain.Pagination{NextCursor: nextToken, HasMore: true},
				}}
				pageToken = nextToken
			}

			client := &fakePollClient{pages: pages}
			var calls []notifyCall
			poller := NewMessagePoller(client, "grant-123", 100, func(method string, params any) error {
				calls = append(calls, notifyCall{method: method, params: params})
				return nil
			})

			err := poller.PollOnce(context.Background())
			wantErr := fmt.Sprintf("message poll truncated at %d pages; not advancing cursor", maxMessagePollPages)
			if err == nil || err.Error() != wantErr {
				t.Fatalf("PollOnce() error = %v, want %q", err, wantErr)
			}
			if poller.cursor != 100 {
				t.Fatalf("cursor = %d, want 100", poller.cursor)
			}
			if len(calls) != 0 {
				t.Fatalf("notify calls = %#v, want none", calls)
			}
		})
	}
}

func TestMessagePoller_PollOnce_ReturnsNotifyError(t *testing.T) {
	notifyErr := errors.New("websocket closed")
	client := &fakePollClient{
		pages: map[string][]domain.MessageListResponse{
			"": {{Data: []domain.Message{pollMessage("new", 101)}}},
		},
	}
	poller := NewMessagePoller(client, "grant-123", 100, func(method string, params any) error {
		return notifyErr
	})

	err := poller.PollOnce(context.Background())
	if !errors.Is(err, notifyErr) {
		t.Fatalf("PollOnce() error = %v, want %v", err, notifyErr)
	}
}

func TestMessagePoller_Run_ReturnsContextErrorOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	poller := NewMessagePoller(&fakePollClient{}, "grant-123", 0, func(method string, params any) error {
		t.Fatal("notify should not be called")
		return nil
	})

	if err := poller.Run(ctx, time.Hour, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want %v", err, context.Canceled)
	}
}

func pollMessages(newest, oldest int64) []domain.Message {
	var messages []domain.Message
	for ts := newest; ts >= oldest; ts-- {
		messages = append(messages, pollMessage(fmt.Sprintf("msg-%03d", ts), ts))
	}
	return messages
}

func pollMessage(id string, unix int64) domain.Message {
	return domain.Message{
		ID:      id,
		GrantID: "grant-123",
		Date:    time.Unix(unix, 0),
	}
}

func filterMessagesAfter(messages []domain.Message, receivedAfter int64) []domain.Message {
	var filtered []domain.Message
	for _, msg := range messages {
		if msg.Date.Unix() > receivedAfter {
			filtered = append(filtered, msg)
		}
	}
	return filtered
}

func assertNotifyIDs(t *testing.T, calls []notifyCall, want []string) {
	t.Helper()
	var got []string
	for i, call := range calls {
		if call.method != "message.received" {
			t.Fatalf("notify call %d method = %q, want message.received", i, call.method)
		}
		payload, ok := call.params.(messageReceivedPayload)
		if !ok {
			t.Fatalf("notify call %d payload type = %T, want messageReceivedPayload", i, call.params)
		}
		got = append(got, payload.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notify IDs = %#v, want %#v", got, want)
	}
}

func assertUniqueNotifyIDs(t *testing.T, calls []notifyCall) {
	t.Helper()
	seen := make(map[string]struct{})
	for _, call := range calls {
		payload := call.params.(messageReceivedPayload)
		if _, ok := seen[payload.ID]; ok {
			t.Fatalf("duplicate notify ID %q", payload.ID)
		}
		seen[payload.ID] = struct{}{}
	}
}

type wantQuery struct {
	receivedAfter int64
	pageToken     string
}

func assertQueries(t *testing.T, got []domain.MessageQueryParams, want []wantQuery) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("query count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Limit != messagePollLimit || got[i].ReceivedAfter != want[i].receivedAfter || got[i].PageToken != want[i].pageToken {
			t.Fatalf("query %d = %+v, want limit %d received_after %d page_token %q", i, got[i], messagePollLimit, want[i].receivedAfter, want[i].pageToken)
		}
	}
}
