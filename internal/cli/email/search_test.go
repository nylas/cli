package email

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

type stubMessagesClient struct {
	getMessagesWithParamsFunc func(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error)
	getMessagesWithCursorFunc func(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error)

	getMessagesWithParamsCalls int
	getMessagesWithCursorCalls int
	pageTokens                 []string
	limits                     []int
}

func (s *stubMessagesClient) GetMessagesWithParams(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
	s.getMessagesWithParamsCalls++
	s.limits = append(s.limits, params.Limit)
	if s.getMessagesWithParamsFunc != nil {
		return s.getMessagesWithParamsFunc(ctx, grantID, params)
	}
	return nil, nil
}

func (s *stubMessagesClient) GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
	s.getMessagesWithCursorCalls++
	s.pageTokens = append(s.pageTokens, params.PageToken)
	s.limits = append(s.limits, params.Limit)
	if s.getMessagesWithCursorFunc != nil {
		return s.getMessagesWithCursorFunc(ctx, grantID, params)
	}
	return nil, nil
}

func TestFetchMessages(t *testing.T) {
	common.ResetLogger()
	common.InitLogger(false, true)
	defer common.ResetLogger()

	t.Run("uses direct fetch when maxItems is negative", func(t *testing.T) {
		expected := []domain.Message{{ID: "msg-1"}, {ID: "msg-2"}}
		client := &stubMessagesClient{
			getMessagesWithParamsFunc: func(_ context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error) {
				assert.Equal(t, "grant-123", grantID)
				assert.Equal(t, 50, params.Limit)
				return expected, nil
			},
		}

		params := &domain.MessageQueryParams{Limit: 50}
		messages, err := fetchMessages(context.Background(), client, "grant-123", params, -1)

		require.NoError(t, err)
		assert.Equal(t, expected, messages)
		assert.Equal(t, 1, client.getMessagesWithParamsCalls)
		assert.Zero(t, client.getMessagesWithCursorCalls)
	})

	t.Run("auto paginates when limit exceeds API maximum", func(t *testing.T) {
		client := &stubMessagesClient{
			getMessagesWithCursorFunc: func(_ context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
				assert.Equal(t, "grant-123", grantID)
				switch params.PageToken {
				case "":
					return &domain.MessageListResponse{
						Data: makeMessages(200, "page-1"),
						Pagination: domain.Pagination{
							NextCursor: "cursor-2",
						},
					}, nil
				case "cursor-2":
					return &domain.MessageListResponse{
						Data: makeMessages(100, "page-2"),
					}, nil
				default:
					t.Fatalf("unexpected page token %q", params.PageToken)
					return nil, nil
				}
			},
		}

		params := &domain.MessageQueryParams{Limit: common.MaxAPILimit}
		messages, err := fetchMessages(context.Background(), client, "grant-123", params, 250)

		require.NoError(t, err)
		assert.Len(t, messages, 250)
		assert.Equal(t, "page-1-0", messages[0].ID)
		assert.Equal(t, "page-2-49", messages[len(messages)-1].ID)
		assert.Equal(t, []string{"", "cursor-2"}, client.pageTokens)
		assert.Equal(t, []int{common.MaxAPILimit, common.MaxAPILimit}, client.limits)
		assert.Zero(t, client.getMessagesWithParamsCalls)
		assert.Equal(t, 2, client.getMessagesWithCursorCalls)
	})

	t.Run("returns pagination errors", func(t *testing.T) {
		client := &stubMessagesClient{
			getMessagesWithCursorFunc: func(_ context.Context, _ string, _ *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
				return nil, errors.New("boom")
			},
		}

		params := &domain.MessageQueryParams{Limit: common.MaxAPILimit}
		messages, err := fetchMessages(context.Background(), client, "grant-123", params, 250)

		require.Error(t, err)
		assert.Nil(t, messages)
		assert.Contains(t, err.Error(), "failed to fetch page 1")
	})
}

func makeMessages(count int, prefix string) []domain.Message {
	messages := make([]domain.Message, count)
	for i := range count {
		messages[i] = domain.Message{ID: prefix + "-" + strconv.Itoa(i)}
	}
	return messages
}
