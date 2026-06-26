package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type threadListParams struct {
	GrantID            string `json:"grant_id,omitempty"`
	Limit              int    `json:"limit,omitempty"`
	PageToken          string `json:"page_token,omitempty"`
	LatestMessageAfter int64  `json:"latest_message_after,omitempty"`
	Unread             *bool  `json:"unread,omitempty"`
}

type threadListResult struct {
	Threads    []domain.Thread `json:"threads"`
	NextCursor string          `json:"next_cursor"`
	HasMore    bool            `json:"has_more"`
}

type threadGetParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	ThreadID string `json:"thread_id"`
}

func RegisterThreadHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("thread.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p threadListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		resp, err := client.GetThreadsWithCursor(ctx, grantID, &domain.ThreadQueryParams{
			Limit:          p.Limit,
			PageToken:      p.PageToken,
			Unread:         p.Unread,
			LatestMsgAfter: p.LatestMessageAfter,
		})
		if err != nil {
			return nil, fmt.Errorf("thread.list: %w", err)
		}

		return threadListResult{
			Threads:    resp.Data,
			NextCursor: resp.Pagination.NextCursor,
			HasMore:    resp.Pagination.HasMore,
		}, nil
	})

	d.Register("thread.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p threadGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ThreadID == "" {
			return nil, NewRPCError(InvalidParams, "thread_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		thread, err := client.GetThread(ctx, grantID, p.ThreadID)
		if err != nil {
			return nil, fmt.Errorf("thread.get: %w", err)
		}
		return thread, nil
	})
}
