package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type contactCreateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateContactRequest
}

type contactUpdateParams struct {
	GrantID   string `json:"grant_id,omitempty"`
	ContactID string `json:"contact_id"`
	domain.UpdateContactRequest
}

type contactDeleteParams struct {
	GrantID   string `json:"grant_id,omitempty"`
	ContactID string `json:"contact_id"`
}

func RegisterContactWriteHandlers(d *Dispatcher, client ports.ContactClient, defaultGrant string) {
	d.Register("contact.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		contact, err := client.CreateContact(ctx, grantID, &p.CreateContactRequest)
		if err != nil {
			return nil, fmt.Errorf("contact.create: %w", err)
		}
		return contact, nil
	})

	d.Register("contact.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactUpdateParams
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

		contact, err := client.UpdateContact(ctx, grantID, p.ContactID, &p.UpdateContactRequest)
		if err != nil {
			return nil, fmt.Errorf("contact.update: %w", err)
		}
		return contact, nil
	})

	d.Register("contact.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactDeleteParams
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

		if err := client.DeleteContact(ctx, grantID, p.ContactID); err != nil {
			return nil, fmt.Errorf("contact.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})
}
