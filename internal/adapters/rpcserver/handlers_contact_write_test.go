package rpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeContactWriteClient struct {
	ports.ContactClient

	createContact func(context.Context, string, *domain.CreateContactRequest) (*domain.Contact, error)
	updateContact func(context.Context, string, string, *domain.UpdateContactRequest) (*domain.Contact, error)
	deleteContact func(context.Context, string, string) error

	createGrantID string
	createReq     *domain.CreateContactRequest
	updateGrantID string
	updateID      string
	updateReq     *domain.UpdateContactRequest
	deleteGrantID string
	deleteID      string
}

func (f *fakeContactWriteClient) CreateContact(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
	f.createGrantID = grantID
	f.createReq = req
	if f.createContact == nil {
		return nil, errors.New("unexpected CreateContact")
	}
	return f.createContact(ctx, grantID, req)
}

func (f *fakeContactWriteClient) UpdateContact(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
	f.updateGrantID = grantID
	f.updateID = contactID
	f.updateReq = req
	if f.updateContact == nil {
		return nil, errors.New("unexpected UpdateContact")
	}
	return f.updateContact(ctx, grantID, contactID, req)
}

func (f *fakeContactWriteClient) DeleteContact(ctx context.Context, grantID, contactID string) error {
	f.deleteGrantID = grantID
	f.deleteID = contactID
	if f.deleteContact == nil {
		return errors.New("unexpected DeleteContact")
	}
	return f.deleteContact(ctx, grantID, contactID)
}

func TestRegisterContactWriteHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeContactWriteClient
		assert       func(*testing.T, *fakeContactWriteClient, rpcTestResponse)
	}{
		{
			name:         "contact.create forwards request",
			method:       "contact.create",
			params:       `{"grant_id":"grant-1","given_name":"Ada","surname":"Lovelace","emails":[{"email":"ada@example.com","type":"work"}]}`,
			defaultGrant: "default-grant",
			client: &fakeContactWriteClient{
				createContact: func(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
					return &domain.Contact{ID: "contact-1", GivenName: req.GivenName}, nil
				},
			},
			assert: func(t *testing.T, client *fakeContactWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.createGrantID != "grant-1" {
					t.Fatalf("createGrantID = %q, want grant-1", client.createGrantID)
				}
				if client.createReq == nil || client.createReq.GivenName != "Ada" || client.createReq.Surname != "Lovelace" {
					t.Fatalf("createReq = %#v, want Ada Lovelace", client.createReq)
				}
				if len(client.createReq.Emails) != 1 || client.createReq.Emails[0].Email != "ada@example.com" {
					t.Fatalf("createReq.Emails = %#v, want ada@example.com", client.createReq.Emails)
				}

				var contact domain.Contact
				unmarshalResult(t, resp, &contact)
				if contact.ID != "contact-1" || contact.GivenName != "Ada" {
					t.Fatalf("contact = %#v, want contact-1 Ada", contact)
				}
			},
		},
		{
			name:   "contact.create missing grant returns invalid params",
			method: "contact.create",
			params: `{"given_name":"Ada"}`,
			client: &fakeContactWriteClient{},
			assert: func(t *testing.T, client *fakeContactWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.createReq != nil {
					t.Fatalf("createReq = %#v, want nil", client.createReq)
				}
			},
		},
		{
			name:         "contact.update forwards embedded fields",
			method:       "contact.update",
			params:       `{"contact_id":"contact-1","given_name":"Grace"}`,
			defaultGrant: "default-grant",
			client: &fakeContactWriteClient{
				updateContact: func(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
					return &domain.Contact{ID: contactID, GivenName: *req.GivenName}, nil
				},
			},
			assert: func(t *testing.T, client *fakeContactWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.updateGrantID != "default-grant" || client.updateID != "contact-1" {
					t.Fatalf("update args = %q %q, want default-grant contact-1", client.updateGrantID, client.updateID)
				}
				if client.updateReq == nil || client.updateReq.GivenName == nil || *client.updateReq.GivenName != "Grace" {
					t.Fatalf("updateReq = %#v, want given_name Grace", client.updateReq)
				}
			},
		},
		{
			name:         "contact.update missing contact_id returns invalid params",
			method:       "contact.update",
			params:       `{"given_name":"Grace"}`,
			defaultGrant: "default-grant",
			client:       &fakeContactWriteClient{},
			assert: func(t *testing.T, client *fakeContactWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.updateReq != nil {
					t.Fatalf("updateReq = %#v, want nil", client.updateReq)
				}
			},
		},
		{
			name:         "contact.delete returns deleted",
			method:       "contact.delete",
			params:       `{"grant_id":"grant-1","contact_id":"contact-1"}`,
			defaultGrant: "default-grant",
			client: &fakeContactWriteClient{
				deleteContact: func(ctx context.Context, grantID, contactID string) error {
					return nil
				},
			},
			assert: func(t *testing.T, client *fakeContactWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.deleteGrantID != "grant-1" || client.deleteID != "contact-1" {
					t.Fatalf("delete args = %q %q, want grant-1 contact-1", client.deleteGrantID, client.deleteID)
				}

				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:         "contact.delete missing contact_id returns invalid params",
			method:       "contact.delete",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeContactWriteClient{},
			assert: func(t *testing.T, client *fakeContactWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
				if client.deleteID != "" {
					t.Fatalf("deleteID = %q, want empty", client.deleteID)
				}
			},
		},
		{
			name:         "client error maps to internal error",
			method:       "contact.create",
			params:       `{"given_name":"Ada"}`,
			defaultGrant: "default-grant",
			client: &fakeContactWriteClient{
				createContact: func(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, client *fakeContactWriteClient, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
				if client.createGrantID != "default-grant" {
					t.Fatalf("createGrantID = %q, want default-grant", client.createGrantID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterContactWriteHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchContactRequest(t, d, tt.method, tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}
