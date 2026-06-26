package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type contactGroupListParams struct {
	GrantID string `json:"grant_id,omitempty"`
}

type contactGroupListResult struct {
	Groups []domain.ContactGroup `json:"groups"`
}

type contactGroupGetParams struct {
	GrantID string `json:"grant_id,omitempty"`
	GroupID string `json:"group_id"`
}

type contactGroupCreateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateContactGroupRequest
}

type contactGroupUpdateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	GroupID string `json:"group_id"`
	domain.UpdateContactGroupRequest
}

type contactGroupDeleteParams struct {
	GrantID string `json:"grant_id,omitempty"`
	GroupID string `json:"group_id"`
}

type contactGetPictureParams struct {
	GrantID        string `json:"grant_id,omitempty"`
	ContactID      string `json:"contact_id"`
	IncludePicture bool   `json:"include_picture,omitempty"`
}

// RegisterContactExtHandlers registers contact group CRUD and the
// picture-bearing contact read.
func RegisterContactExtHandlers(d *Dispatcher, client ports.ContactClient, defaultGrant string) {
	d.Register("contact.group.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactGroupListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		groups, err := client.GetContactGroups(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("contact.group.list: %w", err)
		}
		return contactGroupListResult{Groups: groups}, nil
	})

	d.Register("contact.group.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactGroupGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.GroupID == "" {
			return nil, NewRPCError(InvalidParams, "group_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		group, err := client.GetContactGroup(ctx, grantID, p.GroupID)
		if err != nil {
			return nil, fmt.Errorf("contact.group.get: %w", err)
		}
		return group, nil
	})

	d.Register("contact.group.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactGroupCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		group, err := client.CreateContactGroup(ctx, grantID, &p.CreateContactGroupRequest)
		if err != nil {
			return nil, fmt.Errorf("contact.group.create: %w", err)
		}
		return group, nil
	})

	d.Register("contact.group.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactGroupUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.GroupID == "" {
			return nil, NewRPCError(InvalidParams, "group_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		group, err := client.UpdateContactGroup(ctx, grantID, p.GroupID, &p.UpdateContactGroupRequest)
		if err != nil {
			return nil, fmt.Errorf("contact.group.update: %w", err)
		}
		return group, nil
	})

	d.Register("contact.group.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactGroupDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.GroupID == "" {
			return nil, NewRPCError(InvalidParams, "group_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteContactGroup(ctx, grantID, p.GroupID); err != nil {
			return nil, fmt.Errorf("contact.group.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})

	d.Register("contact.getWithPicture", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p contactGetPictureParams
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

		contact, err := client.GetContactWithPicture(ctx, grantID, p.ContactID, p.IncludePicture)
		if err != nil {
			return nil, fmt.Errorf("contact.getWithPicture: %w", err)
		}
		return contact, nil
	})
}
