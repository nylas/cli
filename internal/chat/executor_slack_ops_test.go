package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	slackAdapter "github.com/nylas/cli/internal/adapters/slack"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestReadSlackThread(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.GetThreadRepliesFunc = func(ctx context.Context, channelID, threadTS string, limit int) ([]domain.SlackMessage, error) {
			assert.Equal(t, "C123456789", channelID)
			assert.Equal(t, "1234.567", threadTS)
			assert.Equal(t, 20, limit)
			return []domain.SlackMessage{
				{ID: "1234.567", Username: "user1", Text: "Original", Timestamp: time.Now(), IsReply: false},
				{ID: "1234.568", Username: "user2", Text: "Reply", Timestamp: time.Now(), IsReply: true},
			}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.readSlackThread(context.Background(), map[string]any{
			"channel":   "C123456789",
			"thread_ts": "1234.567",
		})

		assert.Empty(t, result.Error)
		assert.NotNil(t, result.Data)
	})

	t.Run("missing channel parameter", func(t *testing.T) {
		executor := newSlackExecutor(slackAdapter.NewMockClient())
		result := executor.readSlackThread(context.Background(), map[string]any{
			"thread_ts": "1234.567",
		})

		assert.Contains(t, result.Error, "missing required parameter: channel")
	})

	t.Run("missing thread_ts parameter", func(t *testing.T) {
		executor := newSlackExecutor(slackAdapter.NewMockClient())
		result := executor.readSlackThread(context.Background(), map[string]any{
			"channel": "C123456789",
		})

		assert.Contains(t, result.Error, "missing required parameter: thread_ts")
	})

	t.Run("API error", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.GetThreadRepliesFunc = func(ctx context.Context, channelID, threadTS string, limit int) ([]domain.SlackMessage, error) {
			return nil, errors.New("API error")
		}

		executor := newSlackExecutor(mock)
		result := executor.readSlackThread(context.Background(), map[string]any{
			"channel":   "C123456789",
			"thread_ts": "1234.567",
		})

		assert.Contains(t, result.Error, "failed to get thread replies")
	})
}

func TestSearchSlack(t *testing.T) {
	t.Run("success with default limit", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.SearchMessagesFunc = func(ctx context.Context, query string, limit int) ([]domain.SlackMessage, error) {
			assert.Equal(t, "test query", query)
			assert.Equal(t, 10, limit)
			return []domain.SlackMessage{
				{
					ID:        "1234.567",
					ChannelID: "C999",
					Username:  "testuser",
					Text:      "Result matching test query",
					Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.searchSlack(context.Background(), map[string]any{
			"query": "test query",
		})

		assert.Empty(t, result.Error)
		assert.NotNil(t, result.Data)
	})

	t.Run("success with custom limit", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.SearchMessagesFunc = func(ctx context.Context, query string, limit int) ([]domain.SlackMessage, error) {
			assert.Equal(t, 25, limit)
			return []domain.SlackMessage{}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.searchSlack(context.Background(), map[string]any{
			"query": "test",
			"limit": float64(25),
		})

		assert.Empty(t, result.Error)
	})

	t.Run("missing query parameter", func(t *testing.T) {
		executor := newSlackExecutor(slackAdapter.NewMockClient())
		result := executor.searchSlack(context.Background(), map[string]any{})

		assert.Contains(t, result.Error, "missing required parameter: query")
	})

	t.Run("API error", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.SearchMessagesFunc = func(ctx context.Context, query string, limit int) ([]domain.SlackMessage, error) {
			return nil, errors.New("API error")
		}

		executor := newSlackExecutor(mock)
		result := executor.searchSlack(context.Background(), map[string]any{
			"query": "test",
		})

		assert.Contains(t, result.Error, "failed to search messages")
	})
}

func TestSendSlackMessage(t *testing.T) {
	t.Run("success with channel name", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
			return &domain.SlackChannelListResponse{
				Channels: []domain.SlackChannel{{ID: "C999", Name: "general"}},
			}, nil
		}
		mock.SendMessageFunc = func(ctx context.Context, req *domain.SlackSendMessageRequest) (*domain.SlackMessage, error) {
			assert.Equal(t, "C999", req.ChannelID)
			assert.Equal(t, "Hello world", req.Text)
			assert.Empty(t, req.ThreadTS)
			return &domain.SlackMessage{ID: "1234.567", ChannelID: req.ChannelID, Text: req.Text}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.sendSlackMessage(context.Background(), map[string]any{
			"channel": "#general",
			"text":    "Hello world",
		})

		assert.Empty(t, result.Error)
		assert.NotNil(t, result.Data)
	})

	t.Run("success with thread_ts", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.SendMessageFunc = func(ctx context.Context, req *domain.SlackSendMessageRequest) (*domain.SlackMessage, error) {
			assert.Equal(t, "C123456789", req.ChannelID)
			assert.Equal(t, "Thread reply", req.Text)
			assert.Equal(t, "1234.000", req.ThreadTS)
			return &domain.SlackMessage{ID: "1234.567", ThreadTS: "1234.000"}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.sendSlackMessage(context.Background(), map[string]any{
			"channel":   "C123456789",
			"text":      "Thread reply",
			"thread_ts": "1234.000",
		})

		assert.Empty(t, result.Error)
	})

	t.Run("missing channel parameter", func(t *testing.T) {
		executor := newSlackExecutor(slackAdapter.NewMockClient())
		result := executor.sendSlackMessage(context.Background(), map[string]any{"text": "Hello"})

		assert.Contains(t, result.Error, "missing required parameter: channel")
	})

	t.Run("missing text parameter", func(t *testing.T) {
		executor := newSlackExecutor(slackAdapter.NewMockClient())
		result := executor.sendSlackMessage(context.Background(), map[string]any{"channel": "C123456789"})

		assert.Contains(t, result.Error, "missing required parameter: text")
	})

	t.Run("channel resolution error", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListMyChannelsFunc = func(ctx context.Context, params *domain.SlackChannelQueryParams) (*domain.SlackChannelListResponse, error) {
			return nil, errors.New("API error")
		}

		executor := newSlackExecutor(mock)
		result := executor.sendSlackMessage(context.Background(), map[string]any{
			"channel": "#general",
			"text":    "Hello",
		})

		assert.Contains(t, result.Error, "failed to list channels")
	})

	t.Run("API error", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.SendMessageFunc = func(ctx context.Context, req *domain.SlackSendMessageRequest) (*domain.SlackMessage, error) {
			return nil, errors.New("API error")
		}

		executor := newSlackExecutor(mock)
		result := executor.sendSlackMessage(context.Background(), map[string]any{
			"channel": "C123456789",
			"text":    "Hello",
		})

		assert.Contains(t, result.Error, "failed to send message")
	})
}

func TestListSlackUsers(t *testing.T) {
	t.Run("success with default limit", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListUsersFunc = func(ctx context.Context, limit int, cursor string) (*domain.SlackUserListResponse, error) {
			assert.Equal(t, 20, limit)
			assert.Empty(t, cursor)
			return &domain.SlackUserListResponse{
				Users: []domain.SlackUser{
					{ID: "U123", Name: "user1", DisplayName: "User One", Title: "Engineer", IsBot: false},
					{ID: "U456", Name: "bot", RealName: "Bot User", IsBot: true},
				},
			}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.listSlackUsers(context.Background(), map[string]any{})

		assert.Empty(t, result.Error)
		assert.NotNil(t, result.Data)
	})

	t.Run("success with custom limit", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListUsersFunc = func(ctx context.Context, limit int, cursor string) (*domain.SlackUserListResponse, error) {
			assert.Equal(t, 100, limit)
			return &domain.SlackUserListResponse{Users: []domain.SlackUser{}}, nil
		}

		executor := newSlackExecutor(mock)
		result := executor.listSlackUsers(context.Background(), map[string]any{"limit": float64(100)})

		assert.Empty(t, result.Error)
	})

	t.Run("API error", func(t *testing.T) {
		mock := slackAdapter.NewMockClient()
		mock.ListUsersFunc = func(ctx context.Context, limit int, cursor string) (*domain.SlackUserListResponse, error) {
			return nil, errors.New("API error")
		}

		executor := newSlackExecutor(mock)
		result := executor.listSlackUsers(context.Background(), map[string]any{})

		assert.Contains(t, result.Error, "failed to list users")
	})
}
