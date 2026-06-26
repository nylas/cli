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

func TestEventPoller_PollOnce_DrainsPagesEmitsAllNewEvents(t *testing.T) {
	client, queries := fakeEventPollPages(map[string][]domain.EventListResponse{
		"": {{
			Data:       pollEvents(175, 126),
			Pagination: domain.Pagination{NextCursor: "page-2", HasMore: true},
		}},
		"page-2": {{
			Data:       pollEvents(125, 106),
			Pagination: domain.Pagination{NextCursor: "page-3", HasMore: true},
		}},
		"page-3": {{
			Data: pollEvents(105, 101),
		}},
	}, nil)

	var calls []notifyCall
	poller := NewEventPoller(client, "grant-123", "cal-123", 100, func(method string, params any) error {
		calls = append(calls, notifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}

	var wantIDs []string
	for ts := int64(101); ts <= 175; ts++ {
		wantIDs = append(wantIDs, fmt.Sprintf("event-%03d", ts))
	}
	assertEventNotifyIDs(t, calls, wantIDs)
	assertUniqueEventNotifyIDs(t, calls)
	assertEventQueries(t, *queries, []eventWantQuery{
		{updatedAfter: 99, pageToken: "", calendarID: "cal-123"},
		{updatedAfter: 99, pageToken: "page-2", calendarID: "cal-123"},
		{updatedAfter: 99, pageToken: "page-3", calendarID: "cal-123"},
	})
	if poller.cursor != 175 {
		t.Fatalf("cursor = %d, want 175", poller.cursor)
	}
}

func TestEventPoller_PollOnce_EmitsSameSecondBoundaryEventOnce(t *testing.T) {
	client, queries := fakeEventPollPages(map[string][]domain.EventListResponse{
		"": {
			{Data: []domain.Event{pollEvent("boundary-a", 105)}},
			{Data: []domain.Event{
				pollEvent("boundary-a", 105),
				pollEvent("boundary-b", 105),
				pollEvent("newer-c", 106),
			}},
			{Data: []domain.Event{
				pollEvent("boundary-a", 105),
				pollEvent("boundary-b", 105),
				pollEvent("newer-c", 106),
			}},
		},
	}, nil)

	var calls []notifyCall
	poller := NewEventPoller(client, "grant-123", "", 100, func(method string, params any) error {
		calls = append(calls, notifyCall{method: method, params: params})
		return nil
	})

	for i := 0; i < 3; i++ {
		if err := poller.PollOnce(context.Background()); err != nil {
			t.Fatalf("PollOnce() #%d error = %v", i+1, err)
		}
	}

	assertEventNotifyIDs(t, calls, []string{"boundary-a", "boundary-b", "newer-c"})
	assertUniqueEventNotifyIDs(t, calls)
	assertEventQueries(t, *queries, []eventWantQuery{
		{updatedAfter: 99, pageToken: "", calendarID: "primary"},
		{updatedAfter: 104, pageToken: "", calendarID: "primary"},
		{updatedAfter: 105, pageToken: "", calendarID: "primary"},
	})
}

func TestEventPoller_PollOnce_NoNewSecondPollEmitsNothing(t *testing.T) {
	client, queries := fakeEventPollPages(map[string][]domain.EventListResponse{
		"": {
			{Data: []domain.Event{{
				ID:         "new-1",
				CalendarID: "cal-123",
				Title:      "Sync",
				When:       domain.EventWhen{StartTime: 1010, EndTime: 1020, Object: "timespan"},
				Status:     "confirmed",
				UpdatedAt:  time.Unix(101, 0),
			}}},
			{},
		},
	}, nil)

	var calls []notifyCall
	poller := NewEventPoller(client, "grant-123", "cal-123", 100, func(method string, params any) error {
		calls = append(calls, notifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() second error = %v", err)
	}

	assertEventNotifyIDs(t, calls, []string{"new-1"})
	gotPayload, ok := calls[0].params.(eventUpdatedPayload)
	if !ok {
		t.Fatalf("payload type = %T, want eventUpdatedPayload", calls[0].params)
	}
	wantPayload := eventUpdatedPayload{
		ID:         "new-1",
		CalendarID: "cal-123",
		Title:      "Sync",
		When:       domain.EventWhen{StartTime: 1010, EndTime: 1020, Object: "timespan"},
		Status:     "confirmed",
		UpdatedAt:  101,
	}
	if !reflect.DeepEqual(gotPayload, wantPayload) {
		t.Fatalf("payload = %#v, want %#v", gotPayload, wantPayload)
	}

	payloadJSON, err := json.Marshal(gotPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	wantJSON := `{"id":"new-1","calendar_id":"cal-123","title":"Sync","when":{"start_time":1010,"end_time":1020,"object":"timespan"},"status":"confirmed","updated_at":101}`
	if string(payloadJSON) != wantJSON {
		t.Fatalf("payload JSON = %s, want %s", payloadJSON, wantJSON)
	}
	assertEventQueries(t, *queries, []eventWantQuery{
		{updatedAfter: 99, pageToken: "", calendarID: "cal-123"},
		{updatedAfter: 100, pageToken: "", calendarID: "cal-123"},
	})
}

func TestEventPoller_PollOnce_ReturnsClientErrorFromLaterPage(t *testing.T) {
	clientErr := errors.New("api unavailable")
	client, _ := fakeEventPollPages(map[string][]domain.EventListResponse{
		"": {{
			Data:       []domain.Event{pollEvent("new", 101)},
			Pagination: domain.Pagination{NextCursor: "page-2", HasMore: true},
		}},
	}, map[string]error{"page-2": clientErr})
	called := false
	poller := NewEventPoller(client, "grant-123", "cal-123", 100, func(method string, params any) error {
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

func TestEventPoller_PollOnce_ReturnsErrorWhenPageDrainTruncates(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "more pages after cap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pages := make(map[string][]domain.EventListResponse, maxEventPollPages)
			pageToken := ""
			for page := range maxEventPollPages {
				nextToken := fmt.Sprintf("page-%d", page+1)
				pages[pageToken] = []domain.EventListResponse{{
					Data:       []domain.Event{pollEvent(fmt.Sprintf("event-%03d", page), int64(101+page))},
					Pagination: domain.Pagination{NextCursor: nextToken, HasMore: true},
				}}
				pageToken = nextToken
			}
			client, _ := fakeEventPollPages(pages, nil)

			var calls []notifyCall
			poller := NewEventPoller(client, "grant-123", "cal-123", 100, func(method string, params any) error {
				calls = append(calls, notifyCall{method: method, params: params})
				return nil
			})

			err := poller.PollOnce(context.Background())
			wantErr := fmt.Sprintf("event poll truncated at %d pages; not advancing cursor", maxEventPollPages)
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

func fakeEventPollPages(pages map[string][]domain.EventListResponse, errs map[string]error) (*fakeCalendarClient, *[]eventWantQuery) {
	var queries []eventWantQuery
	return &fakeCalendarClient{
		getEventsWithCursor: func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
			pageToken := ""
			if params != nil {
				pageToken = params.PageToken
				queries = append(queries, eventWantQuery{
					updatedAfter: params.UpdatedAfter,
					pageToken:    params.PageToken,
					calendarID:   calendarID,
				})
			}
			if err := errs[pageToken]; err != nil {
				return nil, err
			}
			if len(pages[pageToken]) == 0 {
				return &domain.EventListResponse{}, nil
			}
			resp := pages[pageToken][0]
			pages[pageToken] = pages[pageToken][1:]
			if params != nil {
				resp.Data = filterEventsAfter(resp.Data, params.UpdatedAfter)
			}
			return &resp, nil
		},
	}, &queries
}

func pollEvents(newest, oldest int64) []domain.Event {
	var events []domain.Event
	for ts := newest; ts >= oldest; ts-- {
		events = append(events, pollEvent(fmt.Sprintf("event-%03d", ts), ts))
	}
	return events
}

func pollEvent(id string, unix int64) domain.Event {
	return domain.Event{
		ID:        id,
		Title:     id,
		UpdatedAt: time.Unix(unix, 0),
	}
}

func filterEventsAfter(events []domain.Event, updatedAfter int64) []domain.Event {
	var filtered []domain.Event
	for _, event := range events {
		if event.UpdatedAt.Unix() > updatedAfter {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func assertEventNotifyIDs(t *testing.T, calls []notifyCall, want []string) {
	t.Helper()
	var got []string
	for i, call := range calls {
		if call.method != "event.updated" {
			t.Fatalf("notify call %d method = %q, want event.updated", i, call.method)
		}
		payload, ok := call.params.(eventUpdatedPayload)
		if !ok {
			t.Fatalf("notify call %d payload type = %T, want eventUpdatedPayload", i, call.params)
		}
		got = append(got, payload.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notify IDs = %#v, want %#v", got, want)
	}
}

func assertUniqueEventNotifyIDs(t *testing.T, calls []notifyCall) {
	t.Helper()
	seen := make(map[string]struct{})
	for _, call := range calls {
		payload := call.params.(eventUpdatedPayload)
		if _, ok := seen[payload.ID]; ok {
			t.Fatalf("duplicate notify ID %q", payload.ID)
		}
		seen[payload.ID] = struct{}{}
	}
}

type eventWantQuery struct {
	updatedAfter int64
	pageToken    string
	calendarID   string
}

func assertEventQueries(t *testing.T, got []eventWantQuery, want []eventWantQuery) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("query count = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].updatedAfter != want[i].updatedAfter || got[i].pageToken != want[i].pageToken || got[i].calendarID != want[i].calendarID {
			t.Fatalf("query %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
