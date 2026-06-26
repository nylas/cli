package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeContactClient struct {
	ports.ContactClient

	getContactsWithCursor func(context.Context, string, *domain.ContactQueryParams) (*domain.ContactListResponse, error)
	getContact            func(context.Context, string, string) (*domain.Contact, error)
	contactGrantIDs       []string
	contactParams         []domain.ContactQueryParams
}

func (f *fakeContactClient) GetContactsWithCursor(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
	f.contactGrantIDs = append(f.contactGrantIDs, grantID)
	if params != nil {
		f.contactParams = append(f.contactParams, *params)
	}
	if f.getContactsWithCursor == nil {
		return nil, errors.New("unexpected GetContactsWithCursor")
	}
	return f.getContactsWithCursor(ctx, grantID, params)
}

func (f *fakeContactClient) GetContact(ctx context.Context, grantID, contactID string) (*domain.Contact, error) {
	if f.getContact == nil {
		return nil, errors.New("unexpected GetContact")
	}
	return f.getContact(ctx, grantID, contactID)
}

func TestRegisterContactHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeContactClient
		assert       func(*testing.T, rpcTestResponse)
	}{
		{
			name:         "contact.list returns contacts and next cursor",
			method:       "contact.list",
			params:       `{"limit":2}`,
			defaultGrant: "default-grant",
			client: &fakeContactClient{
				getContactsWithCursor: func(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "default-grant")
					}
					return &domain.ContactListResponse{
						Data: []domain.Contact{
							{ID: "contact-1", GivenName: "Ada"},
							{ID: "contact-2", GivenName: "Grace"},
						},
						Pagination: domain.Pagination{NextCursor: "cursor-2", HasMore: true},
					}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result struct {
					Contacts   []domain.Contact `json:"contacts"`
					NextCursor string           `json:"next_cursor"`
					HasMore    bool             `json:"has_more"`
				}
				unmarshalResult(t, resp, &result)
				if len(result.Contacts) != 2 || result.Contacts[0].ID != "contact-1" || result.Contacts[1].ID != "contact-2" {
					t.Fatalf("contacts = %#v, want contact-1 and contact-2", result.Contacts)
				}
				if result.NextCursor != "cursor-2" {
					t.Fatalf("next_cursor = %q, want %q", result.NextCursor, "cursor-2")
				}
				if !result.HasMore {
					t.Fatal("has_more = false, want true")
				}
			},
		},
		{
			name:         "contact.list forwards query params and request grant",
			method:       "contact.list",
			params:       `{"grant_id":"request-grant","limit":25,"page_token":"cursor-1"}`,
			defaultGrant: "default-grant",
			client: &fakeContactClient{
				getContactsWithCursor: func(ctx context.Context, grantID string, params *domain.ContactQueryParams) (*domain.ContactListResponse, error) {
					if grantID != "request-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "request-grant")
					}
					if params.Limit != 25 {
						t.Fatalf("Limit = %d, want 25", params.Limit)
					}
					if params.PageToken != "cursor-1" {
						t.Fatalf("PageToken = %q, want %q", params.PageToken, "cursor-1")
					}
					return &domain.ContactListResponse{}, nil
				},
			},
			assert: requireNoRPCError,
		},
		{
			name:   "contact.list missing grant returns invalid params",
			method: "contact.list",
			params: `{}`,
			client: &fakeContactClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "contact.list malformed params returns invalid params",
			method: "contact.list",
			params: `{"limit":"nope"}`,
			client: &fakeContactClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "contact.get with contact_id returns the contact",
			method: "contact.get",
			params: `{"grant_id":"grant-1","contact_id":"contact-1"}`,
			client: &fakeContactClient{
				getContact: func(ctx context.Context, grantID, contactID string) (*domain.Contact, error) {
					if grantID != "grant-1" {
						t.Fatalf("grantID = %q, want %q", grantID, "grant-1")
					}
					if contactID != "contact-1" {
						t.Fatalf("contactID = %q, want %q", contactID, "contact-1")
					}
					return &domain.Contact{ID: "contact-1", GivenName: "Ada"}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var contact domain.Contact
				unmarshalResult(t, resp, &contact)
				if contact.ID != "contact-1" || contact.GivenName != "Ada" {
					t.Fatalf("contact = %#v, want contact-1 Ada", contact)
				}
			},
		},
		{
			name:         "contact.get missing contact_id returns invalid params",
			method:       "contact.get",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "grant-1",
			client:       &fakeContactClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "client error maps to internal error",
			method:       "contact.get",
			params:       `{"contact_id":"contact-1"}`,
			defaultGrant: "default-grant",
			client: &fakeContactClient{
				getContact: func(ctx context.Context, grantID, contactID string) (*domain.Contact, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "default-grant")
					}
					if contactID != "contact-1" {
						t.Fatalf("contactID = %q, want %q", contactID, "contact-1")
					}
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterContactHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchContactRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func dispatchContactRequest(t *testing.T, d *Dispatcher, method, params string) rpcTestResponse {
	t.Helper()

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`)
	got := d.Dispatch(context.Background(), raw)
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}

	var resp rpcTestResponse
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	return resp
}
