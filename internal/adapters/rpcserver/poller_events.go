package rpcserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	eventPollLimit    = 50
	maxEventPollPages = 20
)

type EventPoller struct {
	incrementalState
	client     ports.CalendarClient
	grantID    string
	calendarID string
	notify     NotifyFunc
}

type eventUpdatedPayload struct {
	ID         string           `json:"id"`
	CalendarID string           `json:"calendar_id"`
	Title      string           `json:"title"`
	When       domain.EventWhen `json:"when"`
	Status     string           `json:"status"`
	UpdatedAt  int64            `json:"updated_at"`
}

func NewEventPoller(client ports.CalendarClient, grantID, calendarID string, since int64, notify NotifyFunc) *EventPoller {
	if calendarID == "" {
		calendarID = "primary"
	}
	return &EventPoller{
		incrementalState: incrementalState{cursor: since},
		client:           client,
		grantID:          grantID,
		calendarID:       calendarID,
		notify:           notify,
	}
}

func (p *EventPoller) PollOnce(ctx context.Context) error {
	return pollIncremental(ctx, &p.incrementalState, p.fetch, func(event domain.Event) int64 {
		return event.UpdatedAt.Unix()
	}, func(event domain.Event) string {
		return event.ID
	}, "event.updated", func(event domain.Event) any {
		return eventUpdatedPayload{
			ID:         event.ID,
			CalendarID: p.calendarID,
			Title:      event.Title,
			When:       event.When,
			Status:     event.Status,
			UpdatedAt:  event.UpdatedAt.Unix(),
		}
	}, p.notify)
}

func (p *EventPoller) fetch(ctx context.Context, queryAfter int64) ([]domain.Event, error) {
	var events []domain.Event
	pageToken := ""
	for page := range maxEventPollPages {
		resp, err := p.client.GetEventsWithCursor(ctx, p.grantID, p.calendarID, &domain.EventQueryParams{
			Limit:        eventPollLimit,
			PageToken:    pageToken,
			UpdatedAfter: queryAfter,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.New("event poll response is nil")
		}
		events = append(events, resp.Data...)
		if resp.Pagination.NextCursor == "" || !resp.Pagination.HasMore {
			break
		}
		if page == maxEventPollPages-1 {
			return nil, fmt.Errorf("event poll truncated at %d pages; not advancing cursor", maxEventPollPages)
		}
		pageToken = resp.Pagination.NextCursor
	}
	// ponytail: cap polling bursts at 20 pages; webhooks are the real fix for larger calendar spikes.
	return events, nil
}
