package rpcserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeDraftClient struct {
	ports.MessageClient

	getDrafts func(context.Context, string, int) ([]domain.Draft, error)
	getDraft  func(context.Context, string, string) (*domain.Draft, error)
}

func (f *fakeDraftClient) GetDrafts(ctx context.Context, grantID string, limit int) ([]domain.Draft, error) {
	if f.getDrafts == nil {
		return nil, errors.New("unexpected GetDrafts")
	}
	return f.getDrafts(ctx, grantID, limit)
}

func (f *fakeDraftClient) GetDraft(ctx context.Context, grantID, draftID string) (*domain.Draft, error) {
	if f.getDraft == nil {
		return nil, errors.New("unexpected GetDraft")
	}
	return f.getDraft(ctx, grantID, draftID)
}

type draftRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

func TestRegisterDraftHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeDraftClient
		assert       func(*testing.T, draftRPCResponse)
	}{
		{
			name:         "draft.list returns drafts",
			method:       "draft.list",
			params:       `{"limit":2}`,
			defaultGrant: "default-grant",
			client: &fakeDraftClient{
				getDrafts: func(ctx context.Context, grantID string, limit int) ([]domain.Draft, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want %q", grantID, "default-grant")
					}
					if limit != 2 {
						t.Fatalf("limit = %d, want 2", limit)
					}
					return []domain.Draft{
						{ID: "draft-1", Subject: "Hello"},
						{ID: "draft-2", Subject: "World"},
					}, nil
				},
			},
			assert: func(t *testing.T, resp draftRPCResponse) {
				requireNoDraftRPCError(t, resp)

				var result struct {
					Drafts []domain.Draft `json:"drafts"`
				}
				unmarshalDraftResult(t, resp, &result)
				if len(result.Drafts) != 2 || result.Drafts[0].ID != "draft-1" || result.Drafts[1].ID != "draft-2" {
					t.Fatalf("drafts = %#v, want draft-1 and draft-2", result.Drafts)
				}
			},
		},
		{
			name:   "draft.list missing grant returns invalid params",
			method: "draft.list",
			params: `{"grant_id":""}`,
			client: &fakeDraftClient{},
			assert: func(t *testing.T, resp draftRPCResponse) {
				requireDraftRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:         "draft.get missing draft_id returns invalid params",
			method:       "draft.get",
			params:       `{"grant_id":"grant-1"}`,
			defaultGrant: "grant-1",
			client:       &fakeDraftClient{},
			assert: func(t *testing.T, resp draftRPCResponse) {
				requireDraftRPCErrorCode(t, resp, InvalidParams)
			},
		},
		{
			name:   "draft.get with draft_id returns the draft",
			method: "draft.get",
			params: `{"grant_id":"grant-1","draft_id":"draft-1"}`,
			client: &fakeDraftClient{
				getDraft: func(ctx context.Context, grantID, draftID string) (*domain.Draft, error) {
					if grantID != "grant-1" {
						t.Fatalf("grantID = %q, want %q", grantID, "grant-1")
					}
					if draftID != "draft-1" {
						t.Fatalf("draftID = %q, want %q", draftID, "draft-1")
					}
					return &domain.Draft{ID: "draft-1", Subject: "Hello"}, nil
				},
			},
			assert: func(t *testing.T, resp draftRPCResponse) {
				requireNoDraftRPCError(t, resp)

				var draft domain.Draft
				unmarshalDraftResult(t, resp, &draft)
				if draft.ID != "draft-1" || draft.Subject != "Hello" {
					t.Fatalf("draft = %#v, want draft-1 Hello", draft)
				}
			},
		},
		{
			name:         "client error maps to internal error",
			method:       "draft.get",
			params:       `{"draft_id":"draft-1"}`,
			defaultGrant: "default-grant",
			client: &fakeDraftClient{
				getDraft: func(ctx context.Context, grantID, draftID string) (*domain.Draft, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp draftRPCResponse) {
				requireDraftRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterDraftHandlers(d, tt.client, tt.defaultGrant)

			resp := dispatchDraftRequest(t, d, tt.method, tt.params)
			tt.assert(t, resp)
		})
	}
}

func dispatchDraftRequest(t *testing.T, d *Dispatcher, method, params string) draftRPCResponse {
	t.Helper()

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + method + `","params":` + params + `}`)
	got := d.Dispatch(context.Background(), raw)
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}

	var resp draftRPCResponse
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.JSONRPC != "2.0" {
		t.Fatalf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}
	return resp
}

func requireNoDraftRPCError(t *testing.T, resp draftRPCResponse) {
	t.Helper()

	if resp.Error != nil {
		t.Fatalf("Error = %#v, want nil", resp.Error)
	}
}

func requireDraftRPCErrorCode(t *testing.T, resp draftRPCResponse, want int) {
	t.Helper()

	if resp.Error == nil {
		t.Fatal("Error = nil, want RPC error")
	}
	if resp.Error.Code != want {
		t.Fatalf("Error.Code = %d, want %d", resp.Error.Code, want)
	}
}

func unmarshalDraftResult(t *testing.T, resp draftRPCResponse, dest any) {
	t.Helper()

	if err := json.Unmarshal(resp.Result, dest); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
}
