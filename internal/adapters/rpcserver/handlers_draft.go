package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/ports"
)

type draftListParams struct {
	GrantID string `json:"grant_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type draftGetParams struct {
	GrantID string `json:"grant_id,omitempty"`
	DraftID string `json:"draft_id"`
}

func RegisterDraftHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("draft.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p draftListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		drafts, err := client.GetDrafts(ctx, grantID, p.Limit)
		if err != nil {
			return nil, fmt.Errorf("draft.list: %w", err)
		}

		return map[string]interface{}{"drafts": drafts}, nil
	})

	d.Register("draft.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p draftGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.DraftID == "" {
			return nil, NewRPCError(InvalidParams, "draft_id is required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		draft, err := client.GetDraft(ctx, grantID, p.DraftID)
		if err != nil {
			return nil, fmt.Errorf("draft.get: %w", err)
		}
		return draft, nil
	})
}
