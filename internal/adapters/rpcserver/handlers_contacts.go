package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type contactListParams struct {
	GrantID   string `json:"grant_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	PageToken string `json:"page_token,omitempty"`
}

type contactListResult struct {
	Contacts   []domain.Contact `json:"contacts"`
	NextCursor string           `json:"next_cursor"`
	HasMore    bool             `json:"has_more"`
}

type contactGetParams struct {
	GrantID   string `json:"grant_id,omitempty"`
	ContactID string `json:"contact_id"`
}

func RegisterContactHandlers(d *Dispatcher, client ports.ContactClient, defaultGrant string) {
	d.Register("contact.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		resp, err := client.GetContactsWithCursor(ctx, grantID, &domain.ContactQueryParams{
			Limit:     p.Limit,
			PageToken: p.PageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("contact.list: %w", err)
		}

		return contactListResult{
			Contacts:   resp.Data,
			NextCursor: resp.Pagination.NextCursor,
			HasMore:    resp.Pagination.HasMore,
		}, nil
	})

	d.Register("contact.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ContactID == "" {
			return nil, NewRPCError(InvalidParams, "contact_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		contact, err := client.GetContact(ctx, grantID, p.ContactID)
		if err != nil {
			return nil, fmt.Errorf("contact.get: %w", err)
		}
		return contact, nil
	})
}
