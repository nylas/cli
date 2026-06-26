package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type notetakerListParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.NotetakerQueryParams
}

type notetakerListResult struct {
	Notetakers []domain.Notetaker `json:"notetakers"`
}

type notetakerGetParams struct {
	GrantID     string `json:"grant_id,omitempty"`
	NotetakerID string `json:"notetaker_id"`
}

type notetakerCreateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateNotetakerRequest
}

type notetakerUpdateParams struct {
	GrantID     string `json:"grant_id,omitempty"`
	NotetakerID string `json:"notetaker_id"`
	domain.UpdateNotetakerRequest
}

type notetakerDeleteParams struct {
	GrantID     string `json:"grant_id,omitempty"`
	NotetakerID string `json:"notetaker_id"`
}

type notetakerLeaveParams struct {
	GrantID     string `json:"grant_id,omitempty"`
	NotetakerID string `json:"notetaker_id"`
}

type notetakerMediaParams struct {
	GrantID     string `json:"grant_id,omitempty"`
	NotetakerID string `json:"notetaker_id"`
}

type leftResult struct {
	Left bool `json:"left"`
}

func RegisterNotetakerHandlers(d *Dispatcher, client ports.NotetakerClient, defaultGrant string) {
	d.Register("notetaker.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p notetakerListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		notetakers, err := client.ListNotetakers(ctx, grantID, &p.NotetakerQueryParams)
		if err != nil {
			return nil, fmt.Errorf("notetaker.list: %w", err)
		}
		return notetakerListResult{Notetakers: notetakers}, nil
	})

	d.Register("notetaker.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p notetakerGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.NotetakerID == "" {
			return nil, NewRPCError(InvalidParams, "notetaker_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		notetaker, err := client.GetNotetaker(ctx, grantID, p.NotetakerID)
		if err != nil {
			return nil, fmt.Errorf("notetaker.get: %w", err)
		}
		return notetaker, nil
	})

	d.Register("notetaker.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p notetakerCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		notetaker, err := client.CreateNotetaker(ctx, grantID, &p.CreateNotetakerRequest)
		if err != nil {
			return nil, fmt.Errorf("notetaker.create: %w", err)
		}
		return notetaker, nil
	})

	d.Register("notetaker.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p notetakerUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.NotetakerID == "" {
			return nil, NewRPCError(InvalidParams, "notetaker_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		notetaker, err := client.UpdateNotetaker(ctx, grantID, p.NotetakerID, &p.UpdateNotetakerRequest)
		if err != nil {
			return nil, fmt.Errorf("notetaker.update: %w", err)
		}
		return notetaker, nil
	})

	d.Register("notetaker.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p notetakerDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.NotetakerID == "" {
			return nil, NewRPCError(InvalidParams, "notetaker_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteNotetaker(ctx, grantID, p.NotetakerID); err != nil {
			return nil, fmt.Errorf("notetaker.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("notetaker.leave", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p notetakerLeaveParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.NotetakerID == "" {
			return nil, NewRPCError(InvalidParams, "notetaker_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.LeaveNotetaker(ctx, grantID, p.NotetakerID); err != nil {
			return nil, fmt.Errorf("notetaker.leave: %w", err)
		}
		return leftResult{Left: true}, nil
	})

	d.Register("notetaker.media", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p notetakerMediaParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.NotetakerID == "" {
			return nil, NewRPCError(InvalidParams, "notetaker_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		media, err := client.GetNotetakerMedia(ctx, grantID, p.NotetakerID)
		if err != nil {
			return nil, fmt.Errorf("notetaker.media: %w", err)
		}
		return media, nil
	})
}
