package chat

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	nylas "github.com/nylas/cli/internal/adapters/nylas"
	slackAdapter "github.com/nylas/cli/internal/adapters/slack"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func newSlackExecutor(slackClient *slackAdapter.MockClient) *ToolExecutor {
	return NewToolExecutor(nylas.NewMockClient(), "test-grant", slackClient)
}

func TestSlackTools_WithoutSlackClient(t *testing.T) {
	client := nylas.NewMockClient()
	executor := NewToolExecutor(client, "test-grant", nil)

	slackTools := []string{
		"list_slack_channels",
		"read_slack_messages",
		"read_slack_thread",
		"search_slack",
		"send_slack_message",
		"list_slack_users",
	}

	for _, toolName := range slackTools {
		t.Run(toolName, func(t *testing.T) {
			result := executor.Execute(context.Background(), ToolCall{
				Name: toolName,
				Args: map[string]any{},
			})

			assert.Equal(t, toolName, result.Name)
			assert.Contains(t, result.Error, "Slack integration not configured")
		})
	}
}

func TestAvailableTools_WithSlack(t *testing.T) {
	toolsWithoutSlack := AvailableTools(false)
	toolsWithSlack := AvailableTools(true)

	// Should have 8 base tools without Slack
	assert.Equal(t, 8, len(toolsWithoutSlack))

	// Should have 8 + 6 = 14 tools with Slack
	assert.Equal(t, 14, len(toolsWithSlack))

	// Verify Slack tools are present
	toolNames := make(map[string]bool)
	for _, tool := range toolsWithSlack {
		toolNames[tool.Name] = true
	}

	slackTools := []string{
		"list_slack_channels",
		"read_slack_messages",
		"read_slack_thread",
		"search_slack",
		"send_slack_message",
		"list_slack_users",
	}

	for _, slackTool := range slackTools {
		assert.True(t, toolNames[slackTool], "expected Slack tool %s to be present", slackTool)
	}
}

func TestResolveSlackChannel(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		mockSetup   func(*slackAdapter.MockClient)
		expectID    string
		expectError bool
	}{
		{
			name:     "channel ID passthrough - C prefix",
			input:    "C123456789",
			expectID: "C123456789",
		},
		{
			name:     "channel ID passthrough - G prefix",
			input:    "G987654321",
			expectID: "G987654321",
		},
		{
			name:     "channel ID passthrough - D prefix",
			input:    "D111222333",
			expectID: "D111222333",
		},
		{
			name:  "channel name resolution with #",
			input: "#general",
			mockSetup: func(m *slackAdapter.MockClient) {
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					assert.True(t, params.ExcludeArchived)
					assert.Equal(t, 1000, params.Limit)
					return &domain.SlackChannelListResponse{
						Channels: []domain.SlackChannel{
							{ID: "C999", Name: "general", IsChannel: true},
							{ID: "C888", Name: "random", IsChannel: true},
						},
					}, nil
				}
			},
			expectID: "C999",
		},
		{
			name:  "channel name resolution without #",
			input: "random",
			mockSetup: func(m *slackAdapter.MockClient) {
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					return &domain.SlackChannelListResponse{
						Channels: []domain.SlackChannel{
							{ID: "C999", Name: "general", IsChannel: true},
							{ID: "C888", Name: "random", IsChannel: true},
						},
					}, nil
				}
			},
			expectID: "C888",
		},
		{
			name:  "channel name case insensitive",
			input: "#GENERAL",
			mockSetup: func(m *slackAdapter.MockClient) {
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					return &domain.SlackChannelListResponse{
						Channels: []domain.SlackChannel{
							{ID: "C999", Name: "general", IsChannel: true},
						},
					}, nil
				}
			},
			expectID: "C999",
		},
		{
			name:  "prefix match when exact match not found",
			input: "#incident-20260213-sync-latency",
			mockSetup: func(m *slackAdapter.MockClient) {
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					return &domain.SlackChannelListResponse{
						Channels: []domain.SlackChannel{
							{ID: "C999", Name: "general"},
							{ID: "C777", Name: "incident-20260213-sync-latency-us-prod-748"},
						},
					}, nil
				}
			},
			expectID: "C777",
		},
		{
			name:  "exact match preferred over prefix match",
			input: "#incident",
			mockSetup: func(m *slackAdapter.MockClient) {
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					return &domain.SlackChannelListResponse{
						Channels: []domain.SlackChannel{
							{ID: "C111", Name: "incident-20260213-foo"},
							{ID: "C222", Name: "incident"},
						},
					}, nil
				}
			},
			expectID: "C222",
		},
		{
			name:  "channel found on second page",
			input: "#incident-channel",
			mockSetup: func(m *slackAdapter.MockClient) {
				callCount := 0
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					callCount++
					if callCount == 1 {
						return &domain.SlackChannelListResponse{
							Channels:   []domain.SlackChannel{{ID: "C999", Name: "general"}},
							NextCursor: "page2",
						}, nil
					}
					return &domain.SlackChannelListResponse{
						Channels: []domain.SlackChannel{{ID: "C777", Name: "incident-channel"}},
					}, nil
				}
			},
			expectID: "C777",
		},
		{
			name:  "fallback to search when not in channel list",
			input: "#incident-20260213-prod",
			mockSetup: func(m *slackAdapter.MockClient) {
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					return &domain.SlackChannelListResponse{
						Channels: []domain.SlackChannel{{ID: "C999", Name: "general"}},
					}, nil
				}
				m.SearchMessagesFunc = func(ctx context.Context, query string, limit int) ([]domain.SlackMessage, error) {
					assert.Equal(t, "in:#incident-20260213-prod", query)
					return []domain.SlackMessage{
						{ID: "msg1", ChannelID: "C555"},
					}, nil
				}
			},
			expectID: "C555",
		},
		{
			name:  "channel not found after all fallbacks",
			input: "#nonexistent",
			mockSetup: func(m *slackAdapter.MockClient) {
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					return &domain.SlackChannelListResponse{
						Channels: []domain.SlackChannel{{ID: "C999", Name: "general"}},
					}, nil
				}
				m.SearchMessagesFunc = func(ctx context.Context, query string, limit int) ([]domain.SlackMessage, error) {
					return []domain.SlackMessage{}, nil
				}
			},
			expectError: true,
		},
		{
			name:  "API error",
			input: "#general",
			mockSetup: func(m *slackAdapter.MockClient) {
				m.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
					return nil, errors.New("API error")
				}
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := slackAdapter.NewMockClient()
			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			executor := newSlackExecutor(mock)
			channelID, err := executor.resolveSlackChannel(context.Background(), tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectID, channelID)
			}
		})
	}
}

func TestListSlackChannels(t *testing.T) {
	t.Run("success with default limit", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
			assert.Equal(t, 20, params.Limit)
			assert.True(t, params.ExcludeArchived)
			return &domain.SlackChannelListResponse{
				Channels: []domain.SlackChannel{
					{ID: "C123", Name: "general", IsChannel: true, MemberCount: 42, Topic: "General discussion"},
					{ID: "C456", Name: "random", IsChannel: true, MemberCount: 10},
					{ID: "D789", IsIM: true, MemberCount: 2},
				},
			}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.listSlackChannels(context.Background(), map[string]any{})

		assert.Empty(t, result.Error)
		assert.Equal(t, "list_slack_channels", result.Name)
		assert.NotNil(t, result.Data)
	})

	t.Run("success with custom limit", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
			assert.Equal(t, 50, params.Limit)
			return &domain.SlackChannelListResponse{Channels: []domain.SlackChannel{}}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.listSlackChannels(context.Background(), map[string]any{"limit": float64(50)})

		assert.Empty(t, result.Error)
	})

	t.Run("API error", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
			return nil, errors.New("API error")
		}

		executor := newSlackExecutor(mock)
		result := executor.listSlackChannels(context.Background(), map[string]any{})

		assert.Contains(t, result.Error, "failed to list channels")
	})
}

func TestReadSlackMessages(t *testing.T) {
	t.Run("success with thread expansion", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
			return &domain.SlackChannelListResponse{
				Channels: []domain.SlackChannel{{ID: "C999", Name: "general"}},
			}, nil
		}
		mock.GetMessagesFunc = func(ctx context.Context, params *domain.SlackMessageQueryParams) (*domain.SlackMessageListResponse, error) {
			assert.Equal(t, "C999", params.ChannelID)
			assert.Equal(t, 500, params.Limit)
			return &domain.SlackMessageListResponse{
				Messages: []domain.SlackMessage{
					{ID: "1234.567", Username: "testuser", Text: "Hello world", Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), ReplyCount: 2},
					{ID: "1234.999", Username: "other", Text: "No thread", Timestamp: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC)},
				},
			}, nil
		}
		mock.GetThreadRepliesFunc = func(ctx context.Context, channelID, threadTS string, limit int) ([]domain.SlackMessage, error) {
			assert.Equal(t, "C999", channelID)
			assert.Equal(t, "1234.567", threadTS)
			return []domain.SlackMessage{
				{ID: "1234.567", Username: "testuser", Text: "Hello world", Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)},
				{ID: "1234.568", Username: "replier", Text: "Thread reply", Timestamp: time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC), IsReply: true},
			}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.readSlackMessages(context.Background(), map[string]any{"channel": "#general"})

		assert.Empty(t, result.Error)
		assert.NotNil(t, result.Data)
	})

	t.Run("success with channel ID", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.GetMessagesFunc = func(ctx context.Context, params *domain.SlackMessageQueryParams) (*domain.SlackMessageListResponse, error) {
			assert.Equal(t, "C123456789", params.ChannelID)
			return &domain.SlackMessageListResponse{
				Messages: []domain.SlackMessage{
					{ID: "1", Username: "user1", Text: "msg1", Timestamp: time.Now()},
					{ID: "2", Username: "user2", Text: "msg2", Timestamp: time.Now()},
					{ID: "3", Username: "user3", Text: "msg3", Timestamp: time.Now()},
					{ID: "4", Username: "user4", Text: "msg4", Timestamp: time.Now()},
					{ID: "5", Username: "user5", Text: "msg5", Timestamp: time.Now()},
				},
			}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.readSlackMessages(context.Background(), map[string]any{"channel": "C123456789"})

		assert.Empty(t, result.Error)
	})

	t.Run("search fallback when history returns few messages", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
			return &domain.SlackChannelListResponse{
				Channels: []domain.SlackChannel{{ID: "C555", Name: "incident-channel"}},
			}, nil
		}
		mock.GetMessagesFunc = func(ctx context.Context, params *domain.SlackMessageQueryParams) (*domain.SlackMessageListResponse, error) {
			// History returns only system messages
			return &domain.SlackMessageListResponse{
				Messages: []domain.SlackMessage{
					{ID: "1", Username: "slackbot", Text: "has joined the channel", Timestamp: time.Now()},
				},
			}, nil
		}
		mock.SearchMessagesFunc = func(ctx context.Context, query string, limit int) ([]domain.SlackMessage, error) {
			assert.Equal(t, "in:#incident-channel", query)
			return []domain.SlackMessage{
				{ID: "100", Username: "user1", Text: "Incident started", Timestamp: time.Now()},
				{ID: "101", Username: "user2", Text: "Investigating", Timestamp: time.Now()},
				{ID: "102", Username: "user1", Text: "Root cause found", Timestamp: time.Now()},
			}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.readSlackMessages(context.Background(), map[string]any{"channel": "#incident-channel"})

		assert.Empty(t, result.Error)
		assert.NotNil(t, result.Data)
	})

	t.Run("no search fallback when history has enough messages", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		searchCalled := false
		mock.GetMessagesFunc = func(ctx context.Context, params *domain.SlackMessageQueryParams) (*domain.SlackMessageListResponse, error) {
			msgs := make([]domain.SlackMessage, 10)
			for i := range 10 {
				msgs[i] = domain.SlackMessage{ID: fmt.Sprintf("%d", i), Username: "user", Text: "msg", Timestamp: time.Now()}
			}
			return &domain.SlackMessageListResponse{Messages: msgs}, nil
		}
		mock.SearchMessagesFunc = func(ctx context.Context, query string, limit int) ([]domain.SlackMessage, error) {
			searchCalled = true
			return nil, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.readSlackMessages(context.Background(), map[string]any{"channel": "C123456789"})

		assert.Empty(t, result.Error)
		assert.False(t, searchCalled, "search should not be called when history has enough messages")
	})

	t.Run("missing channel parameter", func(t *testing.T) {
		executor := newSlackExecutor(slackAdapter.NewMockClient())
		result := executor.readSlackMessages(context.Background(), map[string]any{})

		assert.Contains(t, result.Error, "missing required parameter: channel")
	})

	t.Run("channel resolution error", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
			return nil, errors.New("API error")
		}

		executor := newSlackExecutor(mock)
		result := executor.readSlackMessages(context.Background(), map[string]any{"channel": "#general"})

		assert.Contains(t, result.Error, "failed to list channels")
	})

	t.Run("API error", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.GetMessagesFunc = func(ctx context.Context, params *domain.SlackMessageQueryParams) (*domain.SlackMessageListResponse, error) {
			return nil, errors.New("API error")
		}

		executor := newSlackExecutor(mock)
		result := executor.readSlackMessages(context.Background(), map[string]any{"channel": "C123456789"})

		assert.Contains(t, result.Error, "failed to get messages")
	})
}

func TestBuildChannelSearchQuery(t *testing.T) {
	tests := []struct {
		channel   string
		channelID string
		want      string
	}{
		{"#general", "C999", "in:#general"},
		{"#GENERAL", "C999", "in:#general"},
		{"random", "C888", "in:#random"},
		{"C123456789", "C123456789", "in:<#C123456789>"},
		{"G987654321", "G987654321", "in:<#G987654321>"},
	}

	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			got := buildChannelSearchQuery(tt.channel, tt.channelID)
			assert.Equal(t, tt.want, got)
		})
	}
}
