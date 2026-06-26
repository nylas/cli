package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

type contactNotifyCall struct {
	method string
	params any
}

type contactPollScript map[string]domain.ContactListResponse

func TestContactPoller_PollOnce_FirstPollSeedsWithoutEmitting(t *testing.T) {
	client := scriptedFakeContactClient(t, []contactPollScript{
		{"": {Data: []domain.Contact{pollContact("contact-1", 1), pollContact("contact-2", 1)}}},
	}, nil)

	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if len(calls) != 0 {
		t.Fatalf("notify calls = %#v, want none", calls)
	}
	if !poller.seeded {
		t.Fatal("seeded = false, want true")
	}
	wantSeen := map[string]string{
		"contact-1": contactFingerprint(pollContact("contact-1", 1)),
		"contact-2": contactFingerprint(pollContact("contact-2", 1)),
	}
	if !reflect.DeepEqual(poller.seen, wantSeen) {
		t.Fatalf("seen = %#v, want both contacts", poller.seen)
	}
}

func TestContactPoller_PollOnce_SecondPollEmitsChangedContact(t *testing.T) {
	changed := pollContact("contact-1", 2)
	changed.GivenName = "Grace"

	client := scriptedFakeContactClient(t, []contactPollScript{
		{"": {Data: []domain.Contact{pollContact("contact-1", 1), pollContact("contact-2", 1)}}},
		{"": {Data: []domain.Contact{changed, pollContact("contact-2", 1)}}},
	}, nil)

	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() second error = %v", err)
	}

	assertContactNotifyIDs(t, calls, []string{"contact-1"})
	gotPayload := calls[0].params.(contactUpdatedPayload)
	wantPayload := contactUpdatedPayload{
		ID:        "contact-1",
		GivenName: "Grace",
		Surname:   "Lovelace",
		Emails:    []domain.ContactEmail{{Email: "ada@example.com", Type: "work"}},
		UpdatedAt: 2,
	}
	if !reflect.DeepEqual(gotPayload, wantPayload) {
		t.Fatalf("payload = %#v, want %#v", gotPayload, wantPayload)
	}

	payloadJSON, err := json.Marshal(gotPayload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	wantJSON := `{"id":"contact-1","given_name":"Grace","surname":"Lovelace","emails":[{"email":"ada@example.com","type":"work"}],"updated_at":2}`
	if string(payloadJSON) != wantJSON {
		t.Fatalf("payload JSON = %s, want %s", payloadJSON, wantJSON)
	}
}

func TestContactPoller_PollOnce_SecondPollEmitsNewContact(t *testing.T) {
	client := scriptedFakeContactClient(t, []contactPollScript{
		{"": {Data: []domain.Contact{pollContact("contact-1", 1)}}},
		{"": {Data: []domain.Contact{pollContact("contact-1", 1), pollContact("contact-2", 1)}}},
	}, nil)

	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() second error = %v", err)
	}

	assertContactNotifyIDs(t, calls, []string{"contact-2"})
}

func TestContactPoller_PollOnce_UnchangedContactsEmitNothing(t *testing.T) {
	client := scriptedFakeContactClient(t, []contactPollScript{
		{"": {Data: []domain.Contact{pollContact("contact-1", 1)}}},
		{"": {Data: []domain.Contact{pollContact("contact-1", 1)}}},
	}, nil)

	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() second error = %v", err)
	}

	if len(calls) != 0 {
		t.Fatalf("notify calls = %#v, want none", calls)
	}
}

func TestContactPoller_PollOnce_ContentFingerprintChangeDetection(t *testing.T) {
	contentChanged := pollContact("contact-1", 1)
	contentChanged.CompanyName = "Analytical Engines Ltd."

	timestampChanged := pollContact("contact-1", 2)

	tests := []struct {
		name       string
		secondPoll domain.Contact
		wantIDs    []string
	}{
		{
			name:       "content changes with unchanged updated_at emits update",
			secondPoll: contentChanged,
			wantIDs:    []string{"contact-1"},
		},
		{
			name:       "updated_at changes with identical content emits nothing",
			secondPoll: timestampChanged,
			wantIDs:    nil,
		},
		{
			name:       "same content and same updated_at emits nothing",
			secondPoll: pollContact("contact-1", 1),
			wantIDs:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := scriptedFakeContactClient(t, []contactPollScript{
				{"": {Data: []domain.Contact{pollContact("contact-1", 1)}}},
				{"": {Data: []domain.Contact{tt.secondPoll}}},
			}, nil)

			var calls []contactNotifyCall
			poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
				calls = append(calls, contactNotifyCall{method: method, params: params})
				return nil
			})

			if err := poller.PollOnce(context.Background()); err != nil {
				t.Fatalf("PollOnce() error = %v", err)
			}
			if err := poller.PollOnce(context.Background()); err != nil {
				t.Fatalf("PollOnce() second error = %v", err)
			}

			assertContactNotifyIDs(t, calls, tt.wantIDs)
		})
	}
}

func TestContactPoller_PollOnce_SecondPollEmitsDeletedContact(t *testing.T) {
	client := scriptedFakeContactClient(t, []contactPollScript{
		{"": {Data: []domain.Contact{pollContact("contact-1", 1), pollContact("contact-2", 1)}}},
		{"": {Data: []domain.Contact{pollContact("contact-1", 1)}}},
	}, nil)

	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() second error = %v", err)
	}

	if len(calls) != 1 {
		t.Fatalf("notify calls = %#v, want exactly one deletion", calls)
	}
	assertContactDeletedCall(t, calls[0], "contact-2")
}

func TestContactPoller_PollOnce_FirstPollDoesNotEmitDeletes(t *testing.T) {
	client := scriptedFakeContactClient(t, []contactPollScript{
		{"": {Data: []domain.Contact{pollContact("contact-1", 1)}}},
	}, nil)

	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})
	poller.seen = map[string]string{"contact-2": contactFingerprint(pollContact("contact-2", 1))}

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}

	if len(calls) != 0 {
		t.Fatalf("notify calls = %#v, want none", calls)
	}
	if !reflect.DeepEqual(poller.seen, map[string]string{"contact-1": contactFingerprint(pollContact("contact-1", 1))}) {
		t.Fatalf("seen = %#v, want seeded snapshot only", poller.seen)
	}
}

func TestContactPoller_PollOnce_EmitsChangedAndDeletedContact(t *testing.T) {
	changed := pollContact("contact-1", 2)
	changed.JobTitle = "Mathematician"

	client := scriptedFakeContactClient(t, []contactPollScript{
		{"": {Data: []domain.Contact{pollContact("contact-1", 1), pollContact("contact-2", 1)}}},
		{"": {Data: []domain.Contact{changed}}},
	}, nil)

	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}
	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() second error = %v", err)
	}

	if len(calls) != 2 {
		t.Fatalf("notify calls = %#v, want update and deletion", calls)
	}
	assertContactNotifyIDs(t, calls[:1], []string{"contact-1"})
	assertContactDeletedCall(t, calls[1], "contact-2")
}

func TestContactPoller_PollOnce_DrainsPages(t *testing.T) {
	client := scriptedFakeContactClient(t, []contactPollScript{
		{
			"": {
				Data:       []domain.Contact{pollContact("contact-1", 1)},
				Pagination: domain.Pagination{NextCursor: "page-2", HasMore: true},
			},
			"page-2": {Data: []domain.Contact{pollContact("contact-2", 1)}},
		},
	}, nil)

	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})

	if err := poller.PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce() error = %v", err)
	}

	if len(calls) != 0 {
		t.Fatalf("notify calls = %#v, want none", calls)
	}
	assertContactQueries(t, client.contactParams, []string{"", "page-2"})
	if !reflect.DeepEqual(client.contactGrantIDs, []string{"grant-123", "grant-123"}) {
		t.Fatalf("grant IDs = %#v, want grant-123 for every page", client.contactGrantIDs)
	}
	wantSeen := map[string]string{
		"contact-1": contactFingerprint(pollContact("contact-1", 1)),
		"contact-2": contactFingerprint(pollContact("contact-2", 1)),
	}
	if !reflect.DeepEqual(poller.seen, wantSeen) {
		t.Fatalf("seen = %#v, want both pages", poller.seen)
	}
}

func TestContactPoller_PollOnce_ReturnsClientError(t *testing.T) {
	clientErr := errors.New("api unavailable")
	client := scriptedFakeContactClient(t, []contactPollScript{
		{
			"": {
				Data:       []domain.Contact{pollContact("contact-1", 1)},
				Pagination: domain.Pagination{NextCursor: "page-2", HasMore: true},
			},
		},
	}, map[string]error{"page-2": clientErr})

	called := false
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
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
	if poller.seeded {
		t.Fatal("seeded = true, want false")
	}
}

func TestContactPoller_PollOnce_ReturnsErrorWithoutCommitWhenPageCapTruncates(t *testing.T) {
	pages := make(contactPollScript, maxContactPollPages)
	pageToken := ""
	for i := range maxContactPollPages {
		nextCursor := fmt.Sprintf("page-%02d", i+1)
		pages[pageToken] = domain.ContactListResponse{
			Data:       []domain.Contact{pollContact(fmt.Sprintf("contact-%02d", i+1), int64(i+1))},
			Pagination: domain.Pagination{NextCursor: nextCursor, HasMore: true},
		}
		pageToken = nextCursor
	}

	client := scriptedFakeContactClient(t, []contactPollScript{pages}, nil)
	var calls []contactNotifyCall
	poller := NewContactPoller(client, "grant-123", func(method string, params any) error {
		calls = append(calls, contactNotifyCall{method: method, params: params})
		return nil
	})
	poller.seeded = true
	poller.seen = map[string]string{"keep": "fingerprint"}

	err := poller.PollOnce(context.Background())
	if err == nil {
		t.Fatal("PollOnce() error = nil, want truncation error")
	}
	if err.Error() != "contact poll truncated at 500 pages; not committing snapshot" {
		t.Fatalf("PollOnce() error = %q, want truncation error", err)
	}
	if len(calls) != 0 {
		t.Fatalf("notify calls = %#v, want none", calls)
	}
	if !poller.seeded {
		t.Fatal("seeded = false, want unchanged true")
	}
	if !reflect.DeepEqual(poller.seen, map[string]string{"keep": "fingerprint"}) {
		t.Fatalf("seen = %#v, want unchanged", poller.seen)
	}
}

func scriptedFakeContactClient(t *testing.T, polls []contactPollScript, errs map[string]error) *fakeContactClient {
	t.Helper()

	poll := 0
	return &fakeContactClient{
		getContactsWithCursor: func(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
			pageToken := ""
			if params != nil {
				pageToken = params.PageToken
			}
			if err := errs[pageToken]; err != nil {
				return nil, err
			}
			if poll >= len(polls) {
				t.Fatalf("unexpected poll %d page %q", poll+1, pageToken)
			}
			resp, ok := polls[poll][pageToken]
			if !ok {
				t.Fatalf("unexpected poll %d page %q", poll+1, pageToken)
			}
			if resp.Pagination.NextCursor == "" || !resp.Pagination.HasMore {
				poll++
			}
			return &resp, nil
		},
	}
}

func pollContact(id string, updatedAt int64) domain.Contact {
	return domain.Contact{
		ID:        id,
		GivenName: "Ada",
		Surname:   "Lovelace",
		Emails:    []domain.ContactEmail{{Email: "ada@example.com", Type: "work"}},
		UpdatedAt: updatedAt,
	}
}

func assertContactNotifyIDs(t *testing.T, calls []contactNotifyCall, want []string) {
	t.Helper()

	var got []string
	for i, call := range calls {
		if call.method != "contact.updated" {
			t.Fatalf("notify call %d method = %q, want contact.updated", i, call.method)
		}
		payload, ok := call.params.(contactUpdatedPayload)
		if !ok {
			t.Fatalf("notify call %d payload type = %T, want contactUpdatedPayload", i, call.params)
		}
		got = append(got, payload.ID)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("notify IDs = %#v, want %#v", got, want)
	}
}

func assertContactDeletedCall(t *testing.T, call contactNotifyCall, wantID string) {
	t.Helper()

	if call.method != "contact.deleted" {
		t.Fatalf("notify method = %q, want contact.deleted", call.method)
	}
	payload, ok := call.params.(map[string]string)
	if !ok {
		t.Fatalf("notify payload type = %T, want map[string]string", call.params)
	}
	if !reflect.DeepEqual(payload, map[string]string{"id": wantID}) {
		t.Fatalf("notify payload = %#v, want id %q", payload, wantID)
	}
}

func assertContactQueries(t *testing.T, got []domain.ContactQueryParams, wantPageTokens []string) {
	t.Helper()

	if len(got) != len(wantPageTokens) {
		t.Fatalf("query count = %d, want %d: %#v", len(got), len(wantPageTokens), got)
	}
	for i, pageToken := range wantPageTokens {
		if got[i].Limit != contactPollLimit || got[i].PageToken != pageToken {
			t.Fatalf("query %d = %+v, want limit %d page_token %q", i, got[i], contactPollLimit, pageToken)
		}
	}
}
