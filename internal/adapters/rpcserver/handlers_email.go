package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type emailListParams struct {
	GrantID       string `json:"grant_id,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	PageToken     string `json:"page_token,omitempty"`
	ReceivedAfter int64  `json:"received_after,omitempty"`
}

type emailListResult struct {
	Messages   []domain.Message `json:"messages"`
	NextCursor string           `json:"next_cursor"`
	HasMore    bool             `json:"has_more"`
}

type emailGetParams struct {
	GrantID   string `json:"grant_id,omitempty"`
	MessageID string `json:"message_id"`
}

func RegisterEmailHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("email.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p emailListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		resp, err := client.GetMessagesWithCursor(ctx, grantID, &domain.MessageQueryParams{
			Limit:         p.Limit,
			PageToken:     p.PageToken,
			ReceivedAfter: p.ReceivedAfter,
		})
		if err != nil {
			return nil, fmt.Errorf("email.list: %w", err)
		}

		return emailListResult{
			Messages:   resp.Data,
			NextCursor: resp.Pagination.NextCursor,
			HasMore:    resp.Pagination.HasMore,
		}, nil
	})

	d.Register("email.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p emailGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.MessageID == "" {
			return nil, NewRPCError(InvalidParams, "message_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		msg, err := client.GetMessage(ctx, grantID, p.MessageID)
		if err != nil {
			return nil, fmt.Errorf("email.get: %w", err)
		}
		return msg, nil
	})
}
