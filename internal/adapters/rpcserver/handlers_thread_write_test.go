package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeThreadWriteClient struct {
	ports.MessageClient

	updateThread func(context.Context, string, string, *domain.UpdateMessageRequest) (*domain.Thread, error)
	deleteThread func(context.Context, string, string) error
}

func (f *fakeThreadWriteClient) UpdateThread(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
	if f.updateThread == nil {
		return nil, errors.New("unexpected UpdateThread")
	}
	return f.updateThread(ctx, grantID, threadID, req)
}

func (f *fakeThreadWriteClient) DeleteThread(ctx context.Context, grantID, threadID string) error {
	if f.deleteThread == nil {
		return errors.New("unexpected DeleteThread")
	}
	return f.deleteThread(ctx, grantID, threadID)
}

func TestRegisterThreadWriteHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeThreadWriteClient
		assert       func(*testing.T, threadRPCResponse)
	}{
		{
			name:         "thread.update forwards update request",
			method:       "thread.update",
			params:       `{"grant_id":"grant-1","thread_id":"thread-1","unread":false,"starred":true,"folders":["inbox","important"]}`,
			defaultGrant: "default-grant",
			client: &fakeThreadWriteClient{
				updateThread: func(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
					if grantID != "grant-1" {
						t.Fatalf("grantID = %q, want grant-1", grantID)
					}
					if threadID != "thread-1" {
						t.Fatalf("threadID = %q, want thread-1", threadID)
					}
					if req.Unread == nil || *req.Unread {
						t.Fatalf("Unread = %#v, want pointer to false", req.Unread)
					}
					if req.Starred == nil || !*req.Starred {
						t.Fatalf("Starred = %#v, want pointer to true", req.Starred)
					}
					if len(req.Folders) != 2 || req.Folders[0] != "inbox" || req.Folders[1] != "important" {
						t.Fatalf("Folders = %#v, want inbox and important", req.Folders)
					}
					return &domain.Thread{ID: "thread-1", Unread: false, Starred: true}, nil
				},
			},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireNoThreadRPCError(t, resp)

				var thread domain.Thread
				unmarshalThreadResult(t, resp, &thread)
				if thread.ID != "thread-1" || thread.Unread || !thread.Starred {
					t.Fatalf("thread = %#v, want updated thread", thread)
				}
			},
		},
		{
			name:         "thread.delete deletes thread",
			method:       "thread.delete",
			params:       `{"thread_id":"thread-1"}`,
			defaultGrant: "default-grant",
			client: &fakeThreadWriteClient{
				deleteThread: func(ctx context.Context, grantID, threadID string) error {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want default-grant", grantID)
					}
					if threadID != "thread-1" {
						t.Fatalf("threadID = %q, want thread-1", threadID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireNoThreadRPCError(t, resp)

				var result deletedResult
				unmarshalThreadResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:   "thread.update missing grant returns invalid params",
			method: "thread.update",
			params: `{"thread_id":"thread-1"}`,
			client: &fakeThreadWriteClient{},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireThreadRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "thread.update missing thread_id returns invalid params",
			method:       "thread.update",
			params:       `{"unread":true}`,
			defaultGrant: "default-grant",
			client:       &fakeThreadWriteClient{},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireThreadRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "thread.delete missing thread_id returns invalid params",
			method:       "thread.delete",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeThreadWriteClient{},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireThreadRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "thread.update client error maps to internal error",
			method:       "thread.update",
			params:       `{"thread_id":"thread-1","unread":true}`,
			defaultGrant: "default-grant",
			client: &fakeThreadWriteClient{
				updateThread: func(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireThreadRPCErrorCode(t, resp, InternalError)
			},
		},
		{
			name:         "thread.delete client error maps to internal error",
			method:       "thread.delete",
			params:       `{"thread_id":"thread-1"}`,
			defaultGrant: "default-grant",
			client: &fakeThreadWriteClient{
				deleteThread: func(ctx context.Context, grantID, threadID string) error {
					return clientErr
				},
			},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireThreadRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterThreadWriteHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchThreadRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func TestRegisterThreadWriteHandlers_WrapsClientErrors(t *testing.T) {
	clientErr := errors.New("client unavailable")
	d := NewDispatcher()
	RegisterThreadWriteHandlers(d, &fakeThreadWriteClient{
		updateThread: func(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error) {
			return nil, clientErr
		},
		deleteThread: func(ctx context.Context, grantID, threadID string) error {
			return clientErr
		},
	}, "grant-1")

	tests := []struct {
		name   string
		method string
		params string
	}{
		{name: "update", method: "thread.update", params: `{"thread_id":"thread-1"}`},
		{name: "delete", method: "thread.delete", params: `{"thread_id":"thread-1"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := d.handlers[tt.method](context.Background(), json.RawMessage(tt.params))
			if !errors.Is(err, clientErr) {
				t.Fatalf("handler error = %v, want wrapped %v", err, clientErr)
			}
		})
	}
}
