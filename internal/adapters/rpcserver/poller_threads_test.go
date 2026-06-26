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
)

type threadNotifyCall struct {
	method string
	params any
}

func TestThreadPoller_PollOnce_EmitsNewThreads(t *testing.T) {
	client := &fakeThreadClient{
		getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
			if grantID != "grant-123" {
				t.Fatalf("grantID = %q, want grant-123", grantID)
			}
			assertThreadQuery(t, params, 99, "")
			return &domain.ThreadListResponse{Data: filterThreadsAfter([]domain.Thread{
				pollThread("thread-2", 102),
				{
					ID:                    "thread-1",
					Subject:               "Hello",
					LatestMessageRecvDate: time.Unix(101, 0),
					Unread:                true,
					MessageIDs:            []string{"msg-1", "msg-2"},
				},
			}, params.LatestMsgAfter)}, nil
		},
	}

	var calls []threadNotifyCall
	poller := NewThreadPoller(client, "grant-123", 100, func(method string, params any) error {
		calls = append(calls, threadNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}

	assertThreadNotifyIDs(t, calls, []string{"thread-1", "thread-2"})
	assertUniqueThreadNotifyIDs(t, calls)
	gotPayload, ok := calls[0].params.(threadUpdatedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want threadUpdatedPayload", calls[0].params)
	}
	wantPayload := threadUpdatedPayload{
		ID:                        "thread-1",
		Subject:                   "Hello",
		LatestMessageReceivedDate: 101,
		Unread:                    true,
		MessageCount:              2,
	}
	if !reflect.DeepEqual(gotPayload, wantPayload) {
		t.Fatalf("payload = %#v, want %#v", gotPayload, wantPayload)
	}

	payloadJSON, err := json.Marshal(gotPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	wantJSON := `{"id":"thread-1","subject":"Hello","latest_message_received_date":101,"unread":true,"message_count":2}`
	if string(payloadJSON) != wantJSON {
		t.Fatalf("payload JSON = %s, want %s", payloadJSON, wantJSON)
	}
	if poller.cursor != 102 {
		t.Fatalf("cursor = %d, want 102", poller.cursor)
	}
}

func TestThreadPoller_PollOnce_DrainsPagesEmitsAllNewThreads(t *testing.T) {
	pages := map[string]domain.ThreadListResponse{
		"": {
			Data:       pollThreads(175, 126),
			Pagination: domain.Pagination{NextCursor: "page-2", HasMore: true},
		},
		"page-2": {
			Data:       pollThreads(125, 106),
			Pagination: domain.Pagination{NextCursor: "page-3", HasMore: true},
		},
		"page-3": {
			Data: pollThreads(105, 101),
		},
	}
	var queries []domain.ThreadQueryParams
	client := &fakeThreadClient{
		getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
			queries = append(queries, *params)
			resp := pages[params.PageToken]
			resp.Data = filterThreadsAfter(resp.Data, params.LatestMsgAfter)
			return &resp, nil
		},
	}

	var calls []threadNotifyCall
	poller := NewThreadPoller(client, "grant-123", 100, func(method string, params any) error {
		calls = append(calls, threadNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}

	var wantIDs []string
	for ts := int64(101); ts <= 175; ts++ {
		wantIDs = append(wantIDs, fmt.Sprintf("thread-%03d", ts))
	}
	assertThreadNotifyIDs(t, calls, wantIDs)
	assertUniqueThreadNotifyIDs(t, calls)
	assertThreadQueries(t, queries, []wantThreadQuery{
		{latestAfter: 99, pageToken: ""},
		{latestAfter: 99, pageToken: "page-2"},
		{latestAfter: 99, pageToken: "page-3"},
	})
	if poller.cursor != 175 {
		t.Fatalf("cursor = %d, want 175", poller.cursor)
	}
}

func TestThreadPoller_PollOnce_EmitsSameSecondBoundaryThreadOnce(t *testing.T) {
	responses := [][]domain.Thread{
		{pollThread("boundary-a", 105)},
		{pollThread("boundary-a", 105), pollThread("boundary-b", 105), pollThread("newer-c", 106)},
		{pollThread("boundary-a", 105), pollThread("boundary-b", 105), pollThread("newer-c", 106)},
	}
	var queries []domain.ThreadQueryParams
	client := &fakeThreadClient{
		getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
			queries = append(queries, *params)
			threads := filterThreadsAfter(responses[0], params.LatestMsgAfter)
			responses = responses[1:]
			return &domain.ThreadListResponse{Data: threads}, nil
		},
	}

	var calls []threadNotifyCall
	poller := NewThreadPoller(client, "grant-123", 100, func(method string, params any) error {
		calls = append(calls, threadNotifyCall{method: method, params: params})
		return nil
	})

	for i := 0; i < 3; i++ {
		if err := poller.PollOnce(context.Background()); err != nil {
			t.Fatalf("PollOnce() #%d error = %v", i+1, err)
		}
	}

	assertThreadNotifyIDs(t, calls, []string{"boundary-a", "boundary-b", "newer-c"})
	assertUniqueThreadNotifyIDs(t, calls)
	assertThreadQueries(t, queries, []wantThreadQuery{
		{latestAfter: 99},
		{latestAfter: 104},
		{latestAfter: 105},
	})
}

func TestThreadPoller_PollOnce_NoNewThreadsEmitsNothing(t *testing.T) {
	client := &fakeThreadClient{
		getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
			assertThreadQuery(t, params, 99, "")
			return &domain.ThreadListResponse{}, nil
		},
	}
	poller := NewThreadPoller(client, "grant-123", 100, func(method string, params any) error {
		t.Fatal("notify should not be called")
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if poller.cursor != 100 {
		t.Fatalf("cursor = %d, want 100", poller.cursor)
	}
}

func TestThreadPoller_PollOnce_ReturnsClientError(t *testing.T) {
	clientErr := errors.New("api unavailable")
	client := &fakeThreadClient{
		getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
			return nil, clientErr
		},
	}
	called := false
	poller := NewThreadPoller(client, "grant-123", 100, func(method string, params any) error {
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

func pollThread(id string, unix int64) domain.Thread {
	return domain.Thread{
		ID:                    id,
		Subject:               id,
		LatestMessageRecvDate: time.Unix(unix, 0),
	}
}

func pollThreads(newest, oldest int64) []domain.Thread {
	var threads []domain.Thread
	for ts := newest; ts >= oldest; ts-- {
		threads = append(threads, pollThread(fmt.Sprintf("thread-%03d", ts), ts))
	}
	return threads
}

func filterThreadsAfter(threads []domain.Thread, latestMsgAfter int64) []domain.Thread {
	var filtered []domain.Thread
	for _, thread := range threads {
		if thread.LatestMessageRecvDate.Unix() > latestMsgAfter {
			filtered = append(filtered, thread)
		}
	}
	return filtered
}

func assertThreadNotifyIDs(t *testing.T, calls []threadNotifyCall, want []string) {
	t.Helper()
	var got []string
	for i, call := range calls {
		if call.method != "thread.updated" {
			t.Fatalf("notify call %d method = %q, want thread.updated", i, call.method)
		}
		payload, ok := call.params.(threadUpdatedPayload)
		if !ok {
			t.Fatalf("notify call %d payload type = %T, want threadUpdatedPayload", i, call.params)
		}
		got = append(got, payload.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notify IDs = %#v, want %#v", got, want)
	}
}

func assertUniqueThreadNotifyIDs(t *testing.T, calls []threadNotifyCall) {
	t.Helper()
	seen := make(map[string]struct{})
	for _, call := range calls {
		payload := call.params.(threadUpdatedPayload)
		if _, ok := seen[payload.ID]; ok {
			t.Fatalf("duplicate notify ID %q", payload.ID)
		}
		seen[payload.ID] = struct{}{}
	}
}

func assertThreadQuery(t *testing.T, got *domain.ThreadQueryParams, latestAfter int64, pageToken string) {
	t.Helper()
	if got == nil {
		t.Fatal("query params = nil")
	}
	if got.Limit != threadPollLimit || got.LatestMsgAfter != latestAfter || got.PageToken != pageToken {
		t.Fatalf("query = %+v, want limit %d latest_message_after %d page_token %q", got, threadPollLimit, latestAfter, pageToken)
	}
}

type wantThreadQuery struct {
	latestAfter int64
	pageToken   string
}

func assertThreadQueries(t *testing.T, got []domain.ThreadQueryParams, want []wantThreadQuery) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("query count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i, want := range want {
		assertThreadQuery(t, &got[i], want.latestAfter, want.pageToken)
	}
}
