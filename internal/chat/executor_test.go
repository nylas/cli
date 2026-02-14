package chat

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewToolExecutor(t *testing.T) {
	client := nylas.NewMockClient()
	grantID := "test-grant"

	executor := NewToolExecutor(client, grantID, nil)

	require.NotNil(t, executor)
	assert.Equal(t, grantID, executor.grantID)
	assert.Equal(t, client, executor.client)
	assert.Nil(t, executor.slack)
	assert.False(t, executor.HasSlack())
}

func TestExecute_UnknownTool(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant", nil)

	result := executor.Execute(context.Background(), ToolCall{
		Name: "unknown_tool",
		Args: map[string]any{},
	})

	assert.Equal(t, "unknown_tool", result.Name)
	assert.Contains(t, result.Error, "unknown tool: unknown_tool")
	assert.Nil(t, result.Data)
}

func TestExecute_Dispatcher(t *testing.T) {
	tests := []struct {
		name         string
		toolName     string
		wantDispatch bool
	}{
		{"list_emails", "list_emails", true},
		{"read_email", "read_email", true},
		{"search_emails", "search_emails", true},
		{"send_email", "send_email", true},
		{"list_events", "list_events", true},
		{"create_event", "create_event", true},
		{"list_contacts", "list_contacts", true},
		{"list_folders", "list_folders", true},
		{"invalid_tool", "invalid_tool", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant", nil)

			// Set up mock to avoid nil errors
			client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
				return []domain.Message{}, nil
			}
			client.GetMessageFunc = func(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
				return nil, errors.New("missing id")
			}
			client.SendMessageFunc = func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
				return nil, errors.New("missing required fields")
			}
			client.GetEventsFunc = func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
				return []domain.Event{}, nil
			}
			client.CreateEventFunc = func(ctx context.Context, grantID, calendarID string, req *domain.CreateEventRequest) (*domain.Event, error) {
				return nil, errors.New("missing required fields")
			}
			client.GetContactsFunc = func(ctx context.Context, grantID string, params *domain.ContactQueryParams) ([]domain.Contact, error) {
				return []domain.Contact{}, nil
			}
			client.GetFoldersFunc = func(ctx context.Context, grantID string) ([]domain.Folder, error) {
				return []domain.Folder{}, nil
			}

			result := executor.Execute(context.Background(), ToolCall{
				Name: tt.toolName,
				Args: map[string]any{},
			})

			assert.Equal(t, tt.toolName, result.Name)

			if !tt.wantDispatch {
				assert.Contains(t, result.Error, "unknown tool")
			}
		})
	}
}

func TestListEmails_Success(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		args     map[string]any
		messages []domain.Message
		wantLen  int
	}{
		{
			name: "default limit",
			args: map[string]any{},
			messages: []domain.Message{
				{ID: "msg1", Subject: "Test 1", Snippet: "snippet1", Date: now, Unread: true, From: []domain.EmailParticipant{{Email: "test@example.com", Name: "Test User"}}},
				{ID: "msg2", Subject: "Test 2", Snippet: "snippet2", Date: now, Unread: false, From: []domain.EmailParticipant{{Email: "other@example.com"}}},
			},
			wantLen: 2,
		},
		{
			name: "with limit",
			args: map[string]any{"limit": float64(5)},
			messages: []domain.Message{
				{ID: "msg1", Subject: "Test", Snippet: "snippet", Date: now, From: []domain.EmailParticipant{{Email: "test@example.com"}}},
			},
			wantLen: 1,
		},
		{
			name: "with filters",
			args: map[string]any{
				"subject": "Meeting",
				"from":    "boss@example.com",
				"unread":  true,
			},
			messages: []domain.Message{
				{ID: "msg1", Subject: "Meeting Notes", Snippet: "snippet", Date: now, Unread: true, From: []domain.EmailParticipant{{Email: "boss@example.com", Name: "Boss"}}},
			},
			wantLen: 1,
		},
		{
			name:     "empty from array",
			args:     map[string]any{},
			messages: []domain.Message{{ID: "msg1", Subject: "Test", Date: now, From: []domain.EmailParticipant{}}},
			wantLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant", nil)

			client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
				return tt.messages, nil
			}

			result := executor.Execute(context.Background(), ToolCall{
				Name: "list_emails",
				Args: tt.args,
			})

			assert.Empty(t, result.Error)
			assert.NotNil(t, result.Data)

			// The executor returns a custom struct slice, not []any
			// We can verify it's not nil and use JSON to validate structure
			data, err := json.Marshal(result.Data)
			require.NoError(t, err)

			var emails []map[string]any
			err = json.Unmarshal(data, &emails)
			require.NoError(t, err)
			assert.Len(t, emails, tt.wantLen)

			if tt.wantLen > 0 {
				assert.NotEmpty(t, emails[0]["id"])
				assert.NotEmpty(t, emails[0]["date"])
			}
		})
	}
}

func TestListEmails_FromFormatting(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		from     []domain.EmailParticipant
		wantFrom string
	}{
		{
			name:     "with name",
			from:     []domain.EmailParticipant{{Email: "test@example.com", Name: "Test User"}},
			wantFrom: "Test User <test@example.com>",
		},
		{
			name:     "without name",
			from:     []domain.EmailParticipant{{Email: "test@example.com"}},
			wantFrom: "test@example.com",
		},
		{
			name:     "empty from",
			from:     []domain.EmailParticipant{},
			wantFrom: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant", nil)

			client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
				return []domain.Message{{ID: "msg1", Subject: "Test", Date: now, From: tt.from}}, nil
			}

			result := executor.listEmails(context.Background(), map[string]any{})

			require.Empty(t, result.Error)

			data, err := json.Marshal(result.Data)
			require.NoError(t, err)

			var emails []map[string]any
			err = json.Unmarshal(data, &emails)
			require.NoError(t, err)
			require.Len(t, emails, 1)

			assert.Equal(t, tt.wantFrom, emails[0]["from"])
		})
	}
}

func TestListEmails_Error(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant", nil)

	client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
		return nil, errors.New("API error")
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "list_emails",
		Args: map[string]any{},
	})

	assert.Equal(t, "list_emails", result.Name)
	assert.Contains(t, result.Error, "API error")
	assert.Nil(t, result.Data)
}

func TestReadEmail_Success(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		message *domain.Message
		wantLen int
	}{
		{
			name: "normal email",
			message: &domain.Message{
				ID:      "msg1",
				Subject: "Test Subject",
				Body:    "Email body",
				Date:    now,
				From:    []domain.EmailParticipant{{Email: "sender@example.com", Name: "Sender"}},
				To:      []domain.EmailParticipant{{Email: "recipient@example.com"}},
			},
			wantLen: len("Email body"),
		},
		{
			name: "long body truncation",
			message: &domain.Message{
				ID:      "msg2",
				Subject: "Long Email",
				Body:    strings.Repeat("a", 6000),
				Date:    now,
				From:    []domain.EmailParticipant{{Email: "sender@example.com"}},
			},
			wantLen: 5000 + len("\n... [truncated]"),
		},
		{
			name: "exactly 5000 chars",
			message: &domain.Message{
				ID:      "msg3",
				Subject: "Edge Case",
				Body:    strings.Repeat("a", 5000),
				Date:    now,
				From:    []domain.EmailParticipant{{Email: "sender@example.com"}},
			},
			wantLen: 5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant", nil)

			client.GetMessageFunc = func(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
				return tt.message, nil
			}

			result := executor.Execute(context.Background(), ToolCall{
				Name: "read_email",
				Args: map[string]any{"id": "msg1"},
			})

			assert.Empty(t, result.Error)
			require.NotNil(t, result.Data)

			data, err := json.Marshal(result.Data)
			require.NoError(t, err)

			var email map[string]any
			err = json.Unmarshal(data, &email)
			require.NoError(t, err)

			assert.Equal(t, tt.message.ID, email["id"])
			assert.Equal(t, tt.message.Subject, email["subject"])
			assert.Len(t, email["body"].(string), tt.wantLen)

			if len(tt.message.Body) > 5000 {
				assert.True(t, strings.HasSuffix(email["body"].(string), "\n... [truncated]"))
			}
		})
	}
}

func TestReadEmail_MissingID(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"no id", map[string]any{}},
		{"empty id", map[string]any{"id": ""}},
		{"wrong type", map[string]any{"id": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant", nil)

			result := executor.Execute(context.Background(), ToolCall{
				Name: "read_email",
				Args: tt.args,
			})

			assert.Equal(t, "read_email", result.Name)
			assert.Contains(t, result.Error, "id parameter is required")
			assert.Nil(t, result.Data)
		})
	}
}

func TestReadEmail_Error(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant", nil)

	client.GetMessageFunc = func(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
		return nil, errors.New("message not found")
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "read_email",
		Args: map[string]any{"id": "nonexistent"},
	})

	assert.Contains(t, result.Error, "message not found")
}

func TestSearchEmails_Success(t *testing.T) {
	now := time.Now()
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant", nil)

	client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
		assert.Equal(t, "budget report", params.Subject)
		assert.Equal(t, 20, params.Limit)
		return []domain.Message{
			{ID: "msg1", Subject: "Budget Report", Snippet: "Q1 budget", Date: now, From: []domain.EmailParticipant{{Email: "finance@example.com"}}},
		}, nil
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "search_emails",
		Args: map[string]any{"query": "budget report", "limit": float64(20)},
	})

	assert.Empty(t, result.Error)

	data, err := json.Marshal(result.Data)
	require.NoError(t, err)

	var emails []map[string]any
	err = json.Unmarshal(data, &emails)
	require.NoError(t, err)
	assert.Len(t, emails, 1)
}

func TestSearchEmails_MissingQuery(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
	}{
		{"no query", map[string]any{}},
		{"empty query", map[string]any{"query": ""}},
		{"wrong type", map[string]any{"query": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			executor := NewToolExecutor(client, "test-grant", nil)

			result := executor.Execute(context.Background(), ToolCall{
				Name: "search_emails",
				Args: tt.args,
			})

			assert.Contains(t, result.Error, "query parameter is required")
		})
	}
}

func TestSearchEmails_Error(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant", nil)

	client.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
		return nil, errors.New("search failed")
	}

	result := executor.Execute(context.Background(), ToolCall{
		Name: "search_emails",
		Args: map[string]any{"query": "test"},
	})

	assert.Contains(t, result.Error, "search failed")
}
