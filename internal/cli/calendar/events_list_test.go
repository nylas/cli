package calendar

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCalendarClient struct {
	*nylas.MockClient
	getEventsWithCursorFunc func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error)
}

func (c *testCalendarClient) GetEventsWithCursor(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
	if c.getEventsWithCursorFunc != nil {
		return c.getEventsWithCursorFunc(ctx, grantID, calendarID, params)
	}
	return c.MockClient.GetEventsWithCursor(ctx, grantID, calendarID, params)
}

func TestFetchEvents(t *testing.T) {
	t.Run("uses direct fetch when maxItems is zero", func(t *testing.T) {
		client := &testCalendarClient{MockClient: nylas.NewMockClient()}
		client.GetEventsFunc = func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) ([]domain.Event, error) {
			assert.Equal(t, 25, params.Limit)
			return []domain.Event{{ID: "event-1"}}, nil
		}

		events, err := fetchEvents(context.Background(), client, "grant-123", "cal-123", &domain.EventQueryParams{Limit: 25}, 0)

		require.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, "event-1", events[0].ID)
	})

	t.Run("uses cursor pagination when maxItems is positive", func(t *testing.T) {
		client := &testCalendarClient{
			MockClient: nylas.NewMockClient(),
			getEventsWithCursorFunc: func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
				switch params.PageToken {
				case "":
					return &domain.EventListResponse{
						Data: []domain.Event{{ID: "event-1"}, {ID: "event-2"}},
						Pagination: domain.Pagination{
							NextCursor: "next",
						},
					}, nil
				case "next":
					return &domain.EventListResponse{
						Data: []domain.Event{{ID: "event-3"}},
					}, nil
				default:
					return nil, errors.New("unexpected cursor")
				}
			},
		}

		events, err := fetchEvents(context.Background(), client, "grant-123", "cal-123", &domain.EventQueryParams{Limit: 500}, 3)

		require.NoError(t, err)
		assert.Len(t, events, 3)
		assert.Equal(t, []string{"event-1", "event-2", "event-3"}, []string{events[0].ID, events[1].ID, events[2].ID})
	})

	t.Run("returns pagination errors", func(t *testing.T) {
		client := &testCalendarClient{
			MockClient: nylas.NewMockClient(),
			getEventsWithCursorFunc: func(ctx context.Context, grantID, calendarID string, params *domain.EventQueryParams) (*domain.EventListResponse, error) {
				return nil, errors.New("boom")
			},
		}

		events, err := fetchEvents(context.Background(), client, "grant-123", "cal-123", &domain.EventQueryParams{Limit: 500}, 3)

		require.Error(t, err)
		assert.Nil(t, events)
		assert.Contains(t, err.Error(), "failed to fetch page 1")
	})
}
