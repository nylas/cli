package rpcserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	messagePollLimit    = 50
	maxMessagePollPages = 20
)

// NotifyFunc emits a server->client notification.
type NotifyFunc func(method string, params any) error

type MessagePoller struct {
	incrementalState
	client  ports.MessageClient
	grantID string
	notify  NotifyFunc
}

type messageReceivedPayload struct {
	ID      string                    `json:"id"`
	GrantID string                    `json:"grant_id"`
	Subject string                    `json:"subject"`
	Snippet string                    `json:"snippet"`
	From    []domain.EmailParticipant `json:"from"`
	Date    int64                     `json:"date"`
	Unread  bool                      `json:"unread"`
	Folders []string                  `json:"folders"`
}

// NewMessagePoller polls messages newer than since.
func NewMessagePoller(client ports.MessageClient, grantID string, since int64, notify NotifyFunc) *MessagePoller {
	return &MessagePoller{
		incrementalState: incrementalState{cursor: since},
		client:           client,
		grantID:          grantID,
		notify:           notify,
	}
}

// PollOnce runs one polling cycle.
func (p *MessagePoller) PollOnce(ctx context.Context) error {
	return pollIncremental(ctx, &p.incrementalState, p.fetch, func(msg domain.Message) int64 {
		return msg.Date.Unix()
	}, func(msg domain.Message) string {
		return msg.ID
	}, "message.received", func(msg domain.Message) any {
		return messageReceivedPayload{
			ID:      msg.ID,
			GrantID: msg.GrantID,
			Subject: msg.Subject,
			Snippet: msg.Snippet,
			From:    msg.From,
			Date:    msg.Date.Unix(),
			Unread:  msg.Unread,
			Folders: msg.Folders,
		}
	}, p.notify)
}

func (p *MessagePoller) fetch(ctx context.Context, queryAfter int64) ([]domain.Message, error) {
	var messages []domain.Message
	pageToken := ""
	for page := range maxMessagePollPages {
		resp, err := p.client.GetMessagesWithCursor(ctx, p.grantID, &domain.MessageQueryParams{
			Limit:         messagePollLimit,
			PageToken:     pageToken,
			ReceivedAfter: queryAfter,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.New("message poll response is nil")
		}
		messages = append(messages, resp.Data...)
		if resp.Pagination.NextCursor == "" || !resp.Pagination.HasMore {
			break
		}
		if page == maxMessagePollPages-1 {
			return nil, fmt.Errorf("message poll truncated at %d pages; not advancing cursor", maxMessagePollPages)
		}
		pageToken = resp.Pagination.NextCursor
	}
	// ponytail: cap polling bursts at 20 pages; webhooks are the real fix for larger inbox spikes.
	return messages, nil
}
