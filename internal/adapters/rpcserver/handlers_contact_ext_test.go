package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeContactExtClient struct {
	ports.ContactClient

	getContactGroups   func(context.Context, string) ([]domain.ContactGroup, error)
	getContactGroup    func(context.Context, string, string) (*domain.ContactGroup, error)
	createContactGroup func(context.Context, string, *domain.CreateContactGroupRequest) (*domain.ContactGroup, error)
	updateContactGroup func(context.Context, string, string, *domain.UpdateContactGroupRequest) (*domain.ContactGroup, error)
	deleteContactGroup func(context.Context, string, string) error
	getContactPicture  func(context.Context, string, string, bool) (*domain.Contact, error)

	grantIDs []string
}

func (f *fakeContactExtClient) GetContactGroups(ctx context.Context, grantID string) ([]domain.ContactGroup, error) {
	f.grantIDs = append(f.grantIDs, grantID)
	if f.getContactGroups == nil {
		return nil, errors.New("unexpected GetContactGroups")
	}
	return f.getContactGroups(ctx, grantID)
}

func (f *fakeContactExtClient) GetContactGroup(ctx context.Context, grantID, groupID string) (*domain.ContactGroup, error) {
	if f.getContactGroup == nil {
		return nil, errors.New("unexpected GetContactGroup")
	}
	return f.getContactGroup(ctx, grantID, groupID)
}

func (f *fakeContactExtClient) CreateContactGroup(ctx context.Context, grantID string, req *domain.CreateContactGroupRequest) (*domain.ContactGroup, error) {
	if f.createContactGroup == nil {
		return nil, errors.New("unexpected CreateContactGroup")
	}
	return f.createContactGroup(ctx, grantID, req)
}

func (f *fakeContactExtClient) UpdateContactGroup(ctx context.Context, grantID, groupID string, req *domain.UpdateContactGroupRequest) (*domain.ContactGroup, error) {
	if f.updateContactGroup == nil {
		return nil, errors.New("unexpected UpdateContactGroup")
	}
	return f.updateContactGroup(ctx, grantID, groupID, req)
}

func (f *fakeContactExtClient) DeleteContactGroup(ctx context.Context, grantID, groupID string) error {
	if f.deleteContactGroup == nil {
		return errors.New("unexpected DeleteContactGroup")
	}
	return f.deleteContactGroup(ctx, grantID, groupID)
}

func (f *fakeContactExtClient) GetContactWithPicture(ctx context.Context, grantID, contactID string, includePicture bool) (*domain.Contact, error) {
	if f.getContactPicture == nil {
		return nil, errors.New("unexpected GetContactWithPicture")
	}
	return f.getContactPicture(ctx, grantID, contactID, includePicture)
}

func TestRegisterContactExtHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeContactExtClient
		assert       func(*testing.T, *fakeContactExtClient, rpcTestResponse)
	}{
		{
			name:         "contact.group.list returns groups",
			method:       "contact.group.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeContactExtClient{
				getContactGroups: func(_ context.Context, grantID string) ([]domain.ContactGroup, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want default-grant", grantID)
					}
					return []domain.ContactGroup{{ID: "grp-1", Name: "Friends"}}, nil
				},
			},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result contactGroupListResult
				unmarshalResult(t, resp, &result)
				if len(result.Groups) != 1 || result.Groups[0].ID != "grp-1" {
					t.Fatalf("groups = %+v, want one grp-1", result.Groups)
				}
			},
		},
		{
			name:         "contact.group.list without grant errors",
			method:       "contact.group.list",
			params:       `{}`,
			defaultGrant: "",
			client:       &fakeContactExtClient{},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "contact.group.get returns group",
			method:       "contact.group.get",
			params:       `{"group_id":"grp-9"}`,
			defaultGrant: "default-grant",
			client: &fakeContactExtClient{
				getContactGroup: func(_ context.Context, _, groupID string) (*domain.ContactGroup, error) {
					if groupID != "grp-9" {
						t.Fatalf("groupID = %q, want grp-9", groupID)
					}
					return &domain.ContactGroup{ID: "grp-9", Name: "Work"}, nil
				},
			},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var group domain.ContactGroup
				unmarshalResult(t, resp, &group)
				if group.ID != "grp-9" {
					t.Fatalf("group ID = %q, want grp-9", group.ID)
				}
			},
		},
		{
			name:         "contact.group.get missing group_id",
			method:       "contact.group.get",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeContactExtClient{},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "contact.group.create returns group",
			method:       "contact.group.create",
			params:       `{"name":"New Group"}`,
			defaultGrant: "default-grant",
			client: &fakeContactExtClient{
				createContactGroup: func(_ context.Context, _ string, req *domain.CreateContactGroupRequest) (*domain.ContactGroup, error) {
					if req.Name != "New Group" {
						t.Fatalf("name = %q, want New Group", req.Name)
					}
					return &domain.ContactGroup{ID: "grp-new", Name: req.Name}, nil
				},
			},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var group domain.ContactGroup
				unmarshalResult(t, resp, &group)
				if group.ID != "grp-new" {
					t.Fatalf("group ID = %q, want grp-new", group.ID)
				}
			},
		},
		{
			name:         "contact.group.update missing group_id",
			method:       "contact.group.update",
			params:       `{"name":"x"}`,
			defaultGrant: "default-grant",
			client:       &fakeContactExtClient{},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "contact.group.delete returns deleted",
			method:       "contact.group.delete",
			params:       `{"group_id":"grp-9"}`,
			defaultGrant: "default-grant",
			client: &fakeContactExtClient{
				deleteContactGroup: func(_ context.Context, _, groupID string) error {
					if groupID != "grp-9" {
						t.Fatalf("groupID = %q, want grp-9", groupID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:         "contact.getWithPicture passes include flag",
			method:       "contact.getWithPicture",
			params:       `{"contact_id":"c-1","include_picture":true}`,
			defaultGrant: "default-grant",
			client: &fakeContactExtClient{
				getContactPicture: func(_ context.Context, _, contactID string, includePicture bool) (*domain.Contact, error) {
					if contactID != "c-1" || !includePicture {
						t.Fatalf("args = %q/%v, want c-1/true", contactID, includePicture)
					}
					return &domain.Contact{ID: "c-1", Picture: "base64data"}, nil
				},
			},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var contact domain.Contact
				unmarshalResult(t, resp, &contact)
				if contact.ID != "c-1" || contact.Picture == "" {
					t.Fatalf("contact = %+v, want c-1 with picture", contact)
				}
			},
		},
		{
			name:         "contact.getWithPicture missing contact_id",
			method:       "contact.getWithPicture",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeContactExtClient{},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "client error surfaces as internal error",
			method:       "contact.group.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeContactExtClient{
				getContactGroups: func(context.Context, string) ([]domain.ContactGroup, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, _ *fakeContactExtClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterContactExtHandlers(d, tt.client, tt.defaultGrant)

			raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + tt.method + `","params":` + tt.params + `}`)
			got := d.Dispatch(context.Background(), raw)
			if got == nil {
				t.Fatal("Dispatch() = nil, want response")
			}
			var resp rpcTestResponse
			if err := json.Unmarshal(got, &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			tt.assert(t, tt.client, resp)
		})
	}
}
