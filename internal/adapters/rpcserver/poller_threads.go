package rpcserver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	threadPollLimit    = 50
	maxThreadPollPages = 20
)

type ThreadPoller struct {
	incrementalState
	client  ports.MessageClient
	grantID string
	notify  NotifyFunc
}

type threadUpdatedPayload struct {
	ID                        string `json:"id"`
	Subject                   string `json:"subject"`
	LatestMessageReceivedDate int64  `json:"latest_message_received_date"`
	Unread                    bool   `json:"unread"`
	MessageCount              int    `json:"message_count"`
}

// NewThreadPoller polls threads newer than since.
func NewThreadPoller(client ports.MessageClient, grantID string, since int64, notify NotifyFunc) *ThreadPoller {
	return &ThreadPoller{
		incrementalState: incrementalState{cursor: since},
		client:           client,
		grantID:          grantID,
		notify:           notify,
	}
}

// PollOnce runs one polling cycle.
func (p *ThreadPoller) PollOnce(ctx context.Context) error {
	return pollIncremental(ctx, &p.incrementalState, p.fetch, func(thread domain.Thread) int64 {
		return thread.LatestMessageRecvDate.Unix()
	}, func(thread domain.Thread) string {
		return thread.ID
	}, "thread.updated", func(thread domain.Thread) any {
		return threadUpdatedPayload{
			ID:                        thread.ID,
			Subject:                   thread.Subject,
			LatestMessageReceivedDate: thread.LatestMessageRecvDate.Unix(),
			Unread:                    thread.Unread,
			MessageCount:              len(thread.MessageIDs),
		}
	}, p.notify)
}

func (p *ThreadPoller) fetch(ctx context.Context, queryAfter int64) ([]domain.Thread, error) {
	var threads []domain.Thread
	pageToken := ""
	for page := range maxThreadPollPages {
		resp, err := p.client.GetThreadsWithCursor(ctx, p.grantID, &domain.ThreadQueryParams{
			Limit:          threadPollLimit,
			PageToken:      pageToken,
			LatestMsgAfter: queryAfter,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.New("thread poll response is nil")
		}
		threads = append(threads, resp.Data...)
		if resp.Pagination.NextCursor == "" || !resp.Pagination.HasMore {
			break
		}
		if page == maxThreadPollPages-1 {
			return nil, fmt.Errorf("thread poll truncated at %d pages; not advancing cursor", maxThreadPollPages)
		}
		pageToken = resp.Pagination.NextCursor
	}
	// ponytail: cap polling bursts at 20 pages; webhooks are the real fix for larger inbox spikes.
	return threads, nil
}

// Run polls until ctx is cancelled.
func (p *ThreadPoller) Run(ctx context.Context, interval time.Duration, onError func(error)) error {
	return runTicker(ctx, interval, onError, p.PollOnce)
}
