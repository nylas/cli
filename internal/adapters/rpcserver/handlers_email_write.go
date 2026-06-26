package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type emailSendParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.SendMessageRequest
}

type emailUpdateParams struct {
	GrantID   string `json:"grant_id,omitempty"`
	MessageID string `json:"message_id"`
	domain.UpdateMessageRequest
}

type emailDeleteParams struct {
	GrantID   string `json:"grant_id,omitempty"`
	MessageID string `json:"message_id"`
}

type draftCreateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateDraftRequest
}

type draftUpdateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	DraftID string `json:"draft_id"`
	domain.CreateDraftRequest
}

type draftDeleteParams struct {
	GrantID string `json:"grant_id,omitempty"`
	DraftID string `json:"draft_id"`
}

type draftSendParams struct {
	GrantID string `json:"grant_id,omitempty"`
	DraftID string `json:"draft_id"`
	domain.SendDraftRequest
}

func RegisterEmailWriteHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("email.send", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p emailSendParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		msg, err := client.SendMessage(ctx, grantID, &p.SendMessageRequest)
		if err != nil {
			return nil, fmt.Errorf("email.send: %w", err)
		}
		return msg, nil
	})

	d.Register("email.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p emailUpdateParams
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

		msg, err := client.UpdateMessage(ctx, grantID, p.MessageID, &p.UpdateMessageRequest)
		if err != nil {
			return nil, fmt.Errorf("email.update: %w", err)
		}
		return msg, nil
	})

	d.Register("email.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p emailDeleteParams
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

		if err := client.DeleteMessage(ctx, grantID, p.MessageID); err != nil {
			return nil, fmt.Errorf("email.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("draft.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p draftCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		draft, err := client.CreateDraft(ctx, grantID, &p.CreateDraftRequest)
		if err != nil {
			return nil, fmt.Errorf("draft.create: %w", err)
		}
		return draft, nil
	})

	d.Register("draft.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p draftUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.DraftID == "" {
			return nil, NewRPCError(InvalidParams, "draft_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		draft, err := client.UpdateDraft(ctx, grantID, p.DraftID, &p.CreateDraftRequest)
		if err != nil {
			return nil, fmt.Errorf("draft.update: %w", err)
		}
		return draft, nil
	})

	d.Register("draft.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p draftDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.DraftID == "" {
			return nil, NewRPCError(InvalidParams, "draft_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteDraft(ctx, grantID, p.DraftID); err != nil {
			return nil, fmt.Errorf("draft.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("draft.send", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p draftSendParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.DraftID == "" {
			return nil, NewRPCError(InvalidParams, "draft_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		msg, err := client.SendDraft(ctx, grantID, p.DraftID, &p.SendDraftRequest)
		if err != nil {
			return nil, fmt.Errorf("draft.send: %w", err)
		}
		return msg, nil
	})
}
