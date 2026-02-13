package chat

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for send email, events, contacts, and folders operations

func TestSendEmail_Success(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.SendMessageFunc = func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
		assert.Equal(t, "test@example.com", req.To[0].Email)
		assert.Equal(t, "Test Subject", req.Subject)
		assert.Equal(t, "Test body", req.Body)
		return &domain.Message{ID: "sent-123"}, nil
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "send_email",
		Args: map[string]any{
			"to":      "test@example.com",
			"subject": "Test Subject",
			"body":    "Test body",
		},
	})

	assert.Empty(t, result.Error)

	jsonData, err := json.Marshal(result.Data)
	require.NoError(t, err)

	var data map[string]string
	err = json.Unmarshal(jsonData, &data)
	require.NoError(t, err)

	assert.Equal(t, "sent-123", data["id"])
	assert.Equal(t, "sent", data["status"])
}

func TestSendEmail_MissingParams(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing to", map[string]any{"subject": "Test", "body": "Body"}},
		{"missing subject", map[string]any{"to": "test@example.com", "body": "Body"}},
		{"missing body", map[string]any{"to": "test@example.com", "subject": "Test"}},
		{"empty to", map[string]any{"to": "", "subject": "Test", "body": "Body"}},
		{"empty subject", map[string]any{"to": "test@example.com", "subject": "", "body": "Body"}},
		{"empty body", map[string]any{"to": "test@example.com", "subject": "Test", "body": ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant")

			result := executor.Execute(context.Background(), ToolCall{
				Name: "send_email",
				Args: tt.args,
			})

			assert.Contains(t, result.Error, "to, subject, and body are required")
		})
	}
}

func TestSendEmail_Error(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.SendMessageFunc = func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
		return nil, errors.New("send failed")
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "send_email",
		Args: map[string]any{"to": "test@example.com", "subject": "Test", "body": "Body"},
	})

	assert.Contains(t, result.Error, "send failed")
}

func TestListEvents_Success(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		wantCalID  string
		wantLimit  int
		events     []domain.Event
		wantEvents int
	}{
		{
			name:      "default calendar",
			args:      map[string]any{},
			wantCalID: "primary",
			wantLimit: 10,
			events: []domain.Event{
				{ID: "evt1", Title: "Meeting", When: domain.EventWhen{StartTime: 1707753600, EndTime: 1707757200}},
			},
			wantEvents: 1,
		},
		{
			name:      "custom calendar and limit",
			args:      map[string]any{"calendar_id": "work", "limit": float64(5)},
			wantCalID: "work",
			wantLimit: 5,
			events: []domain.Event{
				{ID: "evt2", Title: "Standup", When: domain.EventWhen{StartTime: 1707753600, EndTime: 1707755400}},
			},
			wantEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant")

			client.GetEventsFunc = func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
				assert.Equal(t, tt.wantCalID, calendarID)
				assert.Equal(t, tt.wantLimit, params.Limit)
				return tt.events, nil
			}

			result := executor.Execute(context.Background(), ToolCall{
				Name: "list_events",
				Args: tt.args,
			})

			assert.Empty(t, result.Error)

			data, err := json.Marshal(result.Data)
			require.NoError(t, err)

			var events []map[string]any
			err = json.Unmarshal(data, &events)
			require.NoError(t, err)
			assert.Len(t, events, tt.wantEvents)

			if tt.wantEvents > 0 {
				assert.NotEmpty(t, events[0]["id"])
				assert.NotEmpty(t, events[0]["title"])
			}
		})
	}
}

func TestListEvents_Error(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.GetEventsFunc = func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
		return nil, errors.New("calendar not found")
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "list_events",
		Args: map[string]any{},
	})

	assert.Contains(t, result.Error, "calendar not found")
}

func TestCreateEvent_Success(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.CreateEventFunc = func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
		assert.Equal(t, "primary", calendarID)
		assert.Equal(t, "Team Meeting", req.Title)
		assert.Equal(t, "Discuss Q1 goals", req.Description)
		assert.Greater(t, req.When.EndTime, req.When.StartTime)
		return &domain.Event{ID: "evt-123", Title: "Team Meeting"}, nil
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "create_event",
		Args: map[string]any{
			"title":       "Team Meeting",
			"start_time":  "2026-02-12T14:00:00Z",
			"end_time":    "2026-02-12T15:00:00Z",
			"description": "Discuss Q1 goals",
		},
	})

	assert.Empty(t, result.Error)

	jsonData, err := json.Marshal(result.Data)
	require.NoError(t, err)

	var data map[string]string
	err = json.Unmarshal(jsonData, &data)
	require.NoError(t, err)

	assert.Equal(t, "evt-123", data["id"])
	assert.Equal(t, "Team Meeting", data["title"])
}

func TestCreateEvent_CustomCalendar(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.CreateEventFunc = func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
		assert.Equal(t, "work-calendar", calendarID)
		return &domain.Event{ID: "evt-123", Title: "Test"}, nil
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "create_event",
		Args: map[string]any{
			"title":       "Test",
			"start_time":  "2026-02-12T14:00:00Z",
			"end_time":    "2026-02-12T15:00:00Z",
			"calendar_id": "work-calendar",
		},
	})

	assert.Empty(t, result.Error)
}

func TestCreateEvent_MissingParams(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing title", map[string]any{"start_time": "2026-02-12T14:00:00Z", "end_time": "2026-02-12T15:00:00Z"}},
		{"missing start_time", map[string]any{"title": "Test", "end_time": "2026-02-12T15:00:00Z"}},
		{"missing end_time", map[string]any{"title": "Test", "start_time": "2026-02-12T14:00:00Z"}},
		{"empty title", map[string]any{"title": "", "start_time": "2026-02-12T14:00:00Z", "end_time": "2026-02-12T15:00:00Z"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant")

			result := executor.Execute(context.Background(), ToolCall{
				Name: "create_event",
				Args: tt.args,
			})

			assert.Contains(t, result.Error, "title, start_time, and end_time are required")
		})
	}
}

func TestCreateEvent_InvalidTimeFormat(t *testing.T) {
	tests := []struct {
		name      string
		startTime string
		endTime   string
		wantError string
	}{
		{"invalid start", "invalid-time", "2026-02-12T15:00:00Z", "invalid start_time"},
		{"invalid end", "2026-02-12T14:00:00Z", "invalid-time", "invalid end_time"},
		{"wrong format", "2026-02-12 14:00:00", "2026-02-12 15:00:00", "invalid start_time"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant")

			result := executor.Execute(context.Background(), ToolCall{
				Name: "create_event",
				Args: map[string]any{
					"title":      "Test",
					"start_time": tt.startTime,
					"end_time":   tt.endTime,
				},
			})

			assert.Contains(t, result.Error, tt.wantError)
		})
	}
}

func TestCreateEvent_Error(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.CreateEventFunc = func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
		return nil, errors.New("calendar permission denied")
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "create_event",
		Args: map[string]any{
			"title":      "Test",
			"start_time": "2026-02-12T14:00:00Z",
			"end_time":   "2026-02-12T15:00:00Z",
		},
	})

	assert.Contains(t, result.Error, "calendar permission denied")
}

func TestListContacts_Success(t *testing.T) {
	tests := []struct {
		name         string
		args         map[string]any
		contacts     []domain.Contact
		wantLimit    int
		wantEmail    string
		wantContacts int
	}{
		{
			name:      "default limit",
			args:      map[string]any{},
			wantLimit: 10,
			contacts: []domain.Contact{
				{ID: "c1", GivenName: "John", Surname: "Doe", Emails: []domain.ContactEmail{{Email: "john@example.com"}}},
				{ID: "c2", GivenName: "Jane", Emails: []domain.ContactEmail{{Email: "jane@example.com"}}},
			},
			wantContacts: 2,
		},
		{
			name:      "with query",
			args:      map[string]any{"query": "john@example.com", "limit": float64(5)},
			wantLimit: 5,
			wantEmail: "john@example.com",
			contacts: []domain.Contact{
				{ID: "c1", GivenName: "John", Surname: "Doe", Emails: []domain.ContactEmail{{Email: "john@example.com"}}},
			},
			wantContacts: 1,
		},
		{
			name:         "no emails",
			args:         map[string]any{},
			wantLimit:    10,
			contacts:     []domain.Contact{{ID: "c1", GivenName: "Test", Emails: []domain.ContactEmail{}}},
			wantContacts: 1,
		},
		{
			name:      "surname only",
			args:      map[string]any{},
			wantLimit: 10,
			contacts: []domain.Contact{
				{ID: "c1", Surname: "Smith", Emails: []domain.ContactEmail{{Email: "smith@example.com"}}},
			},
			wantContacts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant")

			client.GetContactsFunc = func(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
				assert.Equal(t, tt.wantLimit, params.Limit)
				if tt.wantEmail != "" {
					assert.Equal(t, tt.wantEmail, params.Email)
				}
				return tt.contacts, nil
			}

			result := executor.Execute(context.Background(), ToolCall{
				Name: "list_contacts",
				Args: tt.args,
			})

			assert.Empty(t, result.Error)

			data, err := json.Marshal(result.Data)
			require.NoError(t, err)

			var contacts []map[string]any
			err = json.Unmarshal(data, &contacts)
			require.NoError(t, err)
			assert.Len(t, contacts, tt.wantContacts)

			if tt.wantContacts > 0 {
				assert.NotEmpty(t, contacts[0]["id"])
			}
		})
	}
}

func TestListContacts_NameFormatting(t *testing.T) {
	tests := []struct {
		name      string
		contact   domain.Contact
		wantName  string
		wantEmail string
	}{
		{
			name:      "full name",
			contact:   domain.Contact{ID: "c1", GivenName: "John", Surname: "Doe", Emails: []domain.ContactEmail{{Email: "john@example.com"}}},
			wantName:  "John Doe",
			wantEmail: "john@example.com",
		},
		{
			name:      "given name only",
			contact:   domain.Contact{ID: "c2", GivenName: "Jane", Emails: []domain.ContactEmail{{Email: "jane@example.com"}}},
			wantName:  "Jane",
			wantEmail: "jane@example.com",
		},
		{
			name:      "no name",
			contact:   domain.Contact{ID: "c3", Emails: []domain.ContactEmail{{Email: "noname@example.com"}}},
			wantName:  "",
			wantEmail: "noname@example.com",
		},
		{
			name:      "no email",
			contact:   domain.Contact{ID: "c4", GivenName: "Test", Emails: []domain.ContactEmail{}},
			wantName:  "Test",
			wantEmail: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant")

			client.GetContactsFunc = func(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
				return []domain.Contact{tt.contact}, nil
			}

			result := executor.listContacts(context.Background(), map[string]any{})

			require.Empty(t, result.Error)

			data, err := json.Marshal(result.Data)
			require.NoError(t, err)

			var contacts []map[string]any
			err = json.Unmarshal(data, &contacts)
			require.NoError(t, err)
			require.Len(t, contacts, 1)

			assert.Equal(t, tt.wantName, contacts[0]["name"])
			assert.Equal(t, tt.wantEmail, contacts[0]["email"])
		})
	}
}

func TestListContacts_Error(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.GetContactsFunc = func(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
		return nil, errors.New("contacts not available")
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "list_contacts",
		Args: map[string]any{},
	})

	assert.Contains(t, result.Error, "contacts not available")
}

func TestListFolders_Success(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
		return []domain.Folder{
			{ID: "f1", Name: "Inbox"},
			{ID: "f2", Name: "Sent"},
			{ID: "f3", Name: "Archive"},
		}, nil
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "list_folders",
		Args: map[string]any{},
	})

	assert.Empty(t, result.Error)

	data, err := json.Marshal(result.Data)
	require.NoError(t, err)

	var folders []map[string]any
	err = json.Unmarshal(data, &folders)
	require.NoError(t, err)
	assert.Len(t, folders, 3)

	assert.Equal(t, "f1", folders[0]["id"])
	assert.Equal(t, "Inbox", folders[0]["name"])
}

func TestListFolders_Error(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant")

	client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
		return nil, errors.New("folders not accessible")
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "list_folders",
		Args: map[string]any{},
	})

	assert.Contains(t, result.Error, "folders not accessible")
}
