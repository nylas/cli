package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeEmailClient struct {
	ports.MessageClient

	getMessagesWithCursor func(context.Context, string, *domain.MessageQueryParams) (*domain.MessageListResponse, error)
	getMessage            func(context.Context, string, string) (*domain.Message, error)
}

func (f *fakeEmailClient) GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
	if f.getMessagesWithCursor == nil {
		return nil, errors.New("unexpected GetMessagesWithCursor")
	}
	return f.getMessagesWithCursor(ctx, grantID, params)
}

func (f *fakeEmailClient) GetMessage(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
	if f.getMessage == nil {
		return nil, errors.New("unexpected GetMessage")
	}
	return f.getMessage(ctx, grantID, messageID)
}

type rpcTestResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

func TestRegisterEmailHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeEmailClient
		assert       func(*testing.T, rpcTestResponse)
	}{
		{
			name:         "email.list returns messages and next cursor",
			method:       "email.list",
			params:       `{"limit":2}`,
			defaultGrant: "default-grant",
			client: &fakeEmailClient{
				getMessagesWithCursor: func(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "default-grant")
					}
					return &domain.MessageListResponse{
						Data: []domain.Message{
							{ID: "msg-1", Subject: "Hello"},
							{ID: "msg-2", Subject: "World"},
						},
						Pagination: domain.Pagination{NextCursor: "cursor-2", HasMore: true},
					}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var result struct {
					Messages   []domain.Message `json:"messages"`
					NextCursor string           `json:"next_cursor"`
					HasMore    bool             `json:"has_more"`
				}
				unmarshalResult(t, resp, &result)
				if len(result.Messages) != 2 || result.Messages[0].ID != "msg-1" || result.Messages[1].ID != "msg-2" {
					t.Fatalf("messages = %#v, want msg-1 and msg-2", result.Messages)
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
			name:         "email.list forwards query params and request grant",
			method:       "email.list",
			params:       `{"grant_id":"request-grant","limit":25,"page_token":"cursor-1","received_after":1710000000}`,
			defaultGrant: "default-grant",
			client: &fakeEmailClient{
				getMessagesWithCursor: func(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error) {
					if grantID != "request-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "request-grant")
					}
					if params.Limit != 25 {
						t.Fatalf("Limit = %d, want 25", params.Limit)
					}
					if params.PageToken != "cursor-1" {
						t.Fatalf("PageToken = %q, want %q", params.PageToken, "cursor-1")
					}
					if params.ReceivedAfter != 1710000000 {
						t.Fatalf("ReceivedAfter = %d, want 1710000000", params.ReceivedAfter)
					}
					return &domain.MessageListResponse{}, nil
				},
			},
			assert: requireNoRPCError,
		},
		{
			name:   "email.list missing grant returns invalid params",
			method: "email.list",
			params: `{}`,
			client: &fakeEmailClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "email.get with message_id returns the message",
			method: "email.get",
			params: `{"grant_id":"grant-1","message_id":"msg-1"}`,
			client: &fakeEmailClient{
				getMessage: func(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
					if grantID != "grant-1" {
						t.Fatalf("grantID = %q, want %q", grantID, "grant-1")
					}
					if messageID != "msg-1" {
						t.Fatalf("messageID = %q, want %q", messageID, "msg-1")
					}
					return &domain.Message{ID: "msg-1", Subject: "Hello"}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)

				var msg domain.Message
				unmarshalResult(t, resp, &msg)
				if msg.ID != "msg-1" || msg.Subject != "Hello" {
					t.Fatalf("message = %#v, want msg-1 Hello", msg)
				}
			},
		},
		{
			name:         "email.get missing message_id returns invalid params",
			method:       "email.get",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "grant-1",
			client:       &fakeEmailClient{},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "client error maps to internal error",
			method:       "email.get",
			params:       `{"message_id":"msg-1"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailClient{
				getMessage: func(ctx context.Context, grantID, messageID string) (*domain.Message, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "default-grant")
					}
					if messageID != "msg-1" {
						t.Fatalf("messageID = %q, want %q", messageID, "msg-1")
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
			RegisterEmailHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchEmailRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func dispatchEmailRequest(t *testing.T, d *Dispatcher, method, params string) rpcTestResponse {
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

func requireNoRPCError(t *testing.T, resp rpcTestResponse) {
	t.Helper()

	if resp.Error != nil {
		t.Fatalf("Error = %#v, want nil", resp.Error)
	}
}

func requireRPCErrorCode(t *testing.T, resp rpcTestResponse, want int) {
	t.Helper()

	if resp.Error == nil {
		t.Fatal("Error = nil, want RPC error")
	}
	if resp.Error.Code != want {
		t.Fatalf("Error.Code = %d, want %d", resp.Error.Code, want)
	}
}

func unmarshalResult(t *testing.T, resp rpcTestResponse, dest any) {
	t.Helper()

	if err := json.Unmarshal(resp.Result, dest); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
}
