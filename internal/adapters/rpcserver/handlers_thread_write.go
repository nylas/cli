package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type threadUpdateParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	ThreadID string `json:"thread_id"`
	domain.UpdateMessageRequest
}

type threadDeleteParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	ThreadID string `json:"thread_id"`
}

func RegisterThreadWriteHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("thread.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p threadUpdateParams
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

		thread, err := client.UpdateThread(ctx, grantID, p.ThreadID, &p.UpdateMessageRequest)
		if err != nil {
			return nil, fmt.Errorf("thread.update: %w", err)
		}
		return thread, nil
	})

	d.Register("thread.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p threadDeleteParams
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

		if err := client.DeleteThread(ctx, grantID, p.ThreadID); err != nil {
			return nil, fmt.Errorf("thread.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})
}
