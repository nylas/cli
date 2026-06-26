package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeThreadClient struct {
	ports.MessageClient

	getThreads           func(context.Context, string, *domain.ThreadQueryParams) ([]domain.Thread, error)
	getThreadsWithCursor func(context.Context, string, *domain.ThreadQueryParams) (*domain.ThreadListResponse, error)
	getThread            func(context.Context, string, string) (*domain.Thread, error)
}

func (f *fakeThreadClient) GetThreads(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
	if f.getThreads == nil {
		return nil, errors.New("unexpected GetThreads")
	}
	return f.getThreads(ctx, grantID, params)
}

func (f *fakeThreadClient) GetThreadsWithCursor(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
	if f.getThreadsWithCursor == nil {
		return nil, errors.New("unexpected GetThreadsWithCursor")
	}
	return f.getThreadsWithCursor(ctx, grantID, params)
}

func (f *fakeThreadClient) GetThread(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
	if f.getThread == nil {
		return nil, errors.New("unexpected GetThread")
	}
	return f.getThread(ctx, grantID, threadID)
}

type threadRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

func TestRegisterThreadHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeThreadClient
		assert       func(*testing.T, threadRPCResponse)
	}{
		{
			name:         "thread.list returns threads and next cursor",
			method:       "thread.list",
			params:       `{"limit":2}`,
			defaultGrant: "default-grant",
			client: &fakeThreadClient{
				getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "default-grant")
					}
					return &domain.ThreadListResponse{
						Data: []domain.Thread{
							{ID: "thread-1", Subject: "Hello"},
							{ID: "thread-2", Subject: "World"},
						},
						Pagination: domain.Pagination{NextCursor: "cursor-2", HasMore: true},
					}, nil
				},
			},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireNoThreadRPCError(t, resp)

				var result struct {
					Threads    []domain.Thread `json:"threads"`
					NextCursor string          `json:"next_cursor"`
					HasMore    bool            `json:"has_more"`
				}
				unmarshalThreadResult(t, resp, &result)
				if len(result.Threads) != 2 || result.Threads[0].ID != "thread-1" || result.Threads[1].ID != "thread-2" {
					t.Fatalf("threads = %#v, want thread-1 and thread-2", result.Threads)
				}
				if result.NextCursor != "cursor-2" {
					t.Fatalf("next_cursor = %q, want cursor-2", result.NextCursor)
				}
				if !result.HasMore {
					t.Fatal("has_more = false, want true")
				}
			},
		},
		{
			name:         "thread.list forwards query params and request grant",
			method:       "thread.list",
			params:       `{"grant_id":"request-grant","limit":25,"page_token":"cursor-1","latest_message_after":1710000000,"unread":false}`,
			defaultGrant: "default-grant",
			client: &fakeThreadClient{
				getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
					if grantID != "request-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "request-grant")
					}
					if params.Limit != 25 {
						t.Fatalf("Limit = %d, want 25", params.Limit)
					}
					if params.PageToken != "cursor-1" {
						t.Fatalf("PageToken = %q, want %q", params.PageToken, "cursor-1")
					}
					if params.LatestMsgAfter != 1710000000 {
						t.Fatalf("LatestMsgAfter = %d, want 1710000000", params.LatestMsgAfter)
					}
					if params.Unread == nil || *params.Unread {
						t.Fatalf("Unread = %#v, want pointer to false", params.Unread)
					}
					return &domain.ThreadListResponse{}, nil
				},
			},
			assert: requireNoThreadRPCError,
		},
		{
			name:   "thread.list leaves unread nil when omitted",
			method: "thread.list",
			params: `{"grant_id":"grant-1"}`,
			client: &fakeThreadClient{
				getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
					if params.Unread != nil {
						t.Fatalf("Unread = %#v, want nil", params.Unread)
					}
					return &domain.ThreadListResponse{}, nil
				},
			},
			assert: requireNoThreadRPCError,
		},
		{
			name:   "thread.list missing grant returns invalid params",
			method: "thread.list",
			params: `{}`,
			client: &fakeThreadClient{},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireThreadRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "thread.list malformed params returns invalid params",
			method: "thread.list",
			params: `{"limit":"bad"}`,
			client: &fakeThreadClient{},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireThreadRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "thread.get with thread_id returns the thread",
			method: "thread.get",
			params: `{"grant_id":"grant-1","thread_id":"thread-1"}`,
			client: &fakeThreadClient{
				getThread: func(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
					if grantID != "grant-1" {
						t.Fatalf("grantID = %q, want %q", grantID, "grant-1")
					}
					if threadID != "thread-1" {
						t.Fatalf("threadID = %q, want %q", threadID, "thread-1")
					}
					return &domain.Thread{ID: "thread-1", Subject: "Hello"}, nil
				},
			},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireNoThreadRPCError(t, resp)

				var thread domain.Thread
				unmarshalThreadResult(t, resp, &thread)
				if thread.ID != "thread-1" || thread.Subject != "Hello" {
					t.Fatalf("thread = %#v, want thread-1 Hello", thread)
				}
			},
		},
		{
			name:         "thread.get missing thread_id returns invalid params",
			method:       "thread.get",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "grant-1",
			client:       &fakeThreadClient{},
			assert: func(t *testing.T, resp threadRPCResponse) {
				requireThreadRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "client error maps to internal error",
			method:       "thread.get",
			params:       `{"thread_id":"thread-1"}`,
			defaultGrant: "default-grant",
			client: &fakeThreadClient{
				getThread: func(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
					return nil, clientErr
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
			RegisterThreadHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchThreadRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func TestRegisterThreadHandlers_WrapsClientErrors(t *testing.T) {
	clientErr := errors.New("client unavailable")
	d := NewDispatcher()
	RegisterThreadHandlers(d, &fakeThreadClient{
		getThreadsWithCursor: func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) (*domain.ThreadListResponse, error) {
			return nil, clientErr
		},
		getThread: func(ctx context.Context, grantID, threadID string) (*domain.Thread, error) {
			return nil, clientErr
		},
	}, "grant-1")

	tests := []struct {
		name   string
		method string
		params string
	}{
		{name: "list", method: "thread.list", params: `{}`},
		{name: "get", method: "thread.get", params: `{"thread_id":"thread-1"}`},
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

func dispatchThreadRequest(t *testing.T, d *Dispatcher, method, params string) threadRPCResponse {
	t.Helper()

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`)
	got := d.Dispatch(context.Background(), raw)
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}

	var resp threadRPCResponse
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	return resp
}

func requireNoThreadRPCError(t *testing.T, resp threadRPCResponse) {
	t.Helper()

	if resp.Error != nil {
		t.Fatalf("Error = %#v, want nil", resp.Error)
	}
}

func requireThreadRPCErrorCode(t *testing.T, resp threadRPCResponse, want int) {
	t.Helper()

	if resp.Error == nil {
		t.Fatal("Error = nil, want RPC error")
	}
	if resp.Error.Code != want {
		t.Fatalf("Error.Code = %d, want %d", resp.Error.Code, want)
	}
}

func unmarshalThreadResult(t *testing.T, resp threadRPCResponse, dest any) {
	t.Helper()

	if err := json.Unmarshal(resp.Result, dest); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
}
