package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteListCalendars_Error
// ============================================================================

func TestExecuteListCalendars_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getCalendarsFunc: func(_ context.Context, _ string) ([]domain.Calendar, error) {
			return nil, errors.New("api down")
		},
	})
	resp := s.executeListCalendars(ctx, map[string]any{})
	if !resp.IsError {
		t.Errorf("expected error response, got: %s", resp.Content[0].Text)
	}
}

// ============================================================================
// TestExecuteListFolders_Error
// ============================================================================

func TestExecuteListFolders_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getFoldersFunc: func(_ context.Context, _ string) ([]domain.Folder, error) {
			return nil, errors.New("api down")
		},
	})
	resp := s.executeListFolders(ctx, map[string]any{})
	if !resp.IsError {
		t.Errorf("expected error response, got: %s", resp.Content[0].Text)
	}
}

// ============================================================================
// TestExecuteListContacts_Error
// ============================================================================

func TestExecuteListContacts_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getContactsFunc: func(_ context.Context, _ string, _ *domain.ContactQueryParams) ([]domain.Contact, error) {
			return nil, errors.New("api down")
		},
	})
	resp := s.executeListContacts(ctx, map[string]any{})
	if !resp.IsError {
		t.Errorf("expected error response, got: %s", resp.Content[0].Text)
	}
}

// ============================================================================
// TestExecuteUpdateCalendar_OptionalFields
// ============================================================================

func TestExecuteUpdateCalendar_OptionalFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		updateCalendarFunc: func(_ context.Context, _, _ string, _ *domain.UpdateCalendarRequest) (*domain.Calendar, error) {
			return &domain.Calendar{ID: "cal1", Name: "New Name"}, nil
		},
	})

	args := map[string]any{
		"calendar_id": "cal1",
		"name":        "New Name",
		"description": "Desc",
		"location":    "NYC",
		"timezone":    "US/Eastern",
	}
	resp := s.executeUpdateCalendar(ctx, args)
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var result map[string]any
	unmarshalText(t, resp, &result)
	if result["status"] != "updated" {
		t.Errorf("status = %v, want updated", result["status"])
	}
}

// ============================================================================
// TestExecuteListEvents_AllFilters
// ============================================================================

func TestExecuteListEvents_AllFilters(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error)
		wantError bool
		wantCount int
	}{
		{
			name: "all optional filters applied",
			args: map[string]any{
				"calendar_id":      "cal1",
				"title":            "Standup",
				"start":            float64(1700000000),
				"end":              float64(1700003600),
				"expand_recurring": true,
				"show_cancelled":   true,
				"limit":            float64(5),
			},
			mockFn: func(_ context.Context, _, _ string, params *domain.EventQueryParams) ([]domain.Event, error) {
				if params.Title != "Standup" {
					return nil, errors.New("title not passed")
				}
				if params.Start != 1700000000 {
					return nil, errors.New("start not passed")
				}
				if params.End != 1700003600 {
					return nil, errors.New("end not passed")
				}
				if !params.ExpandRecurring {
					return nil, errors.New("expand_recurring not passed")
				}
				if !params.ShowCancelled {
					return nil, errors.New("show_cancelled not passed")
				}
				return []domain.Event{{ID: "ev1", Title: "Standup"}}, nil
			},
			wantCount: 1,
		},
		{
			name: "api error propagates",
			args: map[string]any{"calendar_id": "cal1"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.EventQueryParams) ([]domain.Event, error) {
				return nil, errors.New("fail")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getEventsFunc: tt.mockFn})
			resp := s.executeListEvents(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var items []map[string]any
			unmarshalText(t, resp, &items)
			if len(items) != tt.wantCount {
				t.Errorf("item count = %d, want %d", len(items), tt.wantCount)
			}
		})
	}
}

// ============================================================================
// TestExecuteGetEvent_Reminders
// ============================================================================

func TestExecuteGetEvent_Reminders(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getEventFunc: func(_ context.Context, _, _, _ string) (*domain.Event, error) {
			return &domain.Event{
				ID:    "ev1",
				Title: "Test",
				Reminders: &domain.Reminders{
					Overrides: []domain.Reminder{
						{ReminderMinutes: 10, ReminderMethod: "popup"},
					},
				},
			}, nil
		},
	})

	resp := s.executeGetEvent(ctx, map[string]any{"event_id": "ev1"})
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var result map[string]any
	unmarshalText(t, resp, &result)
	if _, ok := result["reminders"]; !ok {
		t.Error("reminders field should be present when event has reminders")
	}
}

// ============================================================================
// TestExecuteCreateEvent_OptionalFields
// ============================================================================

func TestExecuteCreateEvent_OptionalFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		createEventFunc: func(_ context.Context, _, _ string, req *domain.CreateEventRequest) (*domain.Event, error) {
			return &domain.Event{ID: "ev-new", Title: req.Title}, nil
		},
	})

	args := map[string]any{
		"title":            "Big Meeting",
		"start_time":       float64(1700000000),
		"end_time":         float64(1700003600),
		"participants":     []any{map[string]any{"email": "a@b.com", "name": "Alice"}},
		"busy":             true,
		"visibility":       "public",
		"conferencing_url": "https://zoom.us/j/123",
		"reminders":        []any{map[string]any{"minutes": float64(10), "method": "popup"}},
	}
	resp := s.executeCreateEvent(ctx, args)
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var result map[string]any
	unmarshalText(t, resp, &result)
	if result["status"] != "created" {
		t.Errorf("status = %v, want created", result["status"])
	}
}

// ============================================================================
// TestExecuteUpdateEvent_OptionalFields
// ============================================================================

func TestExecuteUpdateEvent_OptionalFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	happyMock := func(_ context.Context, _, _, _ string, _ *domain.UpdateEventRequest) (*domain.Event, error) {
		return &domain.Event{ID: "ev1", Title: "Updated"}, nil
	}

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "all optional fields via start_time/end_time",
			args: map[string]any{
				"event_id":     "ev1",
				"title":        "Updated",
				"description":  "New desc",
				"location":     "Room A",
				"visibility":   "private",
				"start_time":   float64(1700000000),
				"end_time":     float64(1700003600),
				"participants": []any{map[string]any{"email": "a@b.com"}},
				"busy":         true,
			},
		},
		{
			name: "start_date/end_date branch",
			args: map[string]any{
				"event_id":   "ev1",
				"start_date": "2024-01-15",
				"end_date":   "2024-01-16",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{updateEventFunc: happyMock})
			resp := s.executeUpdateEvent(ctx, tt.args)
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if result["status"] != "updated" {
				t.Errorf("status = %v, want updated", result["status"])
			}
		})
	}
}

// ============================================================================
// TestExecuteUpdateContact_OptionalFields
// ============================================================================

func TestExecuteUpdateContact_OptionalFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		updateContactFunc: func(_ context.Context, _, _ string, _ *domain.UpdateContactRequest) (*domain.Contact, error) {
			return &domain.Contact{ID: "c1", GivenName: "Alice"}, nil
		},
	})

	args := map[string]any{
		"contact_id":   "c1",
		"given_name":   "Alice",
		"surname":      "Smith",
		"nickname":     "Ali",
		"company_name": "Acme",
		"job_title":    "Engineer",
		"notes":        "VIP client",
	}
	resp := s.executeUpdateContact(ctx, args)
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var result map[string]any
	unmarshalText(t, resp, &result)
	if result["status"] != "updated" {
		t.Errorf("status = %v, want updated", result["status"])
	}
}

// ============================================================================
// TestExecuteListMessages_FolderIDFilter
// ============================================================================

func TestExecuteListMessages_FolderIDFilter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getMessagesWithParamsFunc: func(_ context.Context, _ string, params *domain.MessageQueryParams) ([]domain.Message, error) {
			if len(params.In) == 0 || params.In[0] != "inbox" {
				return nil, errors.New("folder_id not passed")
			}
			return []domain.Message{}, nil
		},
	})

	resp := s.executeListMessages(ctx, map[string]any{"folder_id": "inbox"})
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
}

// ============================================================================
// TestExecuteSendMessage_CcBccReplyTo
// ============================================================================

func TestExecuteSendMessage_CcBccReplyTo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		sendMessageFunc: func(_ context.Context, _ string, req *domain.SendMessageRequest) (*domain.Message, error) {
			if len(req.Cc) == 0 {
				return nil, errors.New("cc not passed")
			}
			if len(req.Bcc) == 0 {
				return nil, errors.New("bcc not passed")
			}
			if req.ReplyToMsgID != "orig-123" {
				return nil, errors.New("reply_to_message_id not passed")
			}
			return &domain.Message{ID: "sent1"}, nil
		},
	})

	args := map[string]any{
		"to":                  []any{map[string]any{"email": "a@b.com"}},
		"cc":                  []any{map[string]any{"email": "cc@b.com"}},
		"bcc":                 []any{map[string]any{"email": "bcc@b.com"}},
		"subject":             "Test",
		"body":                "Hello",
		"reply_to_message_id": "orig-123",
	}
	resp := s.executeSendMessage(ctx, args)
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var result map[string]any
	unmarshalText(t, resp, &result)
	if result["status"] != "sent" {
		t.Errorf("status = %v, want sent", result["status"])
	}
}

// ============================================================================
// TestExecuteGetMessage_AllParticipantFields
// ============================================================================

func TestExecuteGetMessage_AllParticipantFields(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	s := newMockServer(&mockNylasClient{
		getMessageFunc: func(_ context.Context, _, _ string) (*domain.Message, error) {
			return &domain.Message{
				ID:      "m1",
				Subject: "Hi",
				From:    []domain.EmailParticipant{{Name: "Alice", Email: "a@b.com"}},
				To:      []domain.EmailParticipant{{Email: "to@b.com"}},
				Cc:      []domain.EmailParticipant{{Email: "cc@b.com"}},
				Bcc:     []domain.EmailParticipant{{Email: "bcc@b.com"}},
				Folders: []string{"inbox"},
				Attachments: []domain.Attachment{
					{ID: "att1"},
				},
			}, nil
		},
	})

	resp := s.executeGetMessage(ctx, map[string]any{"message_id": "m1"})
	if resp.IsError {
		t.Fatalf("unexpected error: %s", resp.Content[0].Text)
	}
	var result map[string]any
	unmarshalText(t, resp, &result)
	if result["id"] != "m1" {
		t.Errorf("id = %v, want m1", result["id"])
	}
	if result["from"] != "Alice <a@b.com>" {
		t.Errorf("from = %v, want 'Alice <a@b.com>'", result["from"])
	}
	attachCount, _ := result["attachments"].(float64)
	if attachCount != 1 {
		t.Errorf("attachments = %v, want 1", result["attachments"])
	}
}
