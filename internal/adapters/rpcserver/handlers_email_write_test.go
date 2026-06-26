package rpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeEmailWriteClient struct {
	ports.MessageClient

	err error

	sendMessageResult   *domain.Message
	updateMessageResult *domain.Message
	createDraftResult   *domain.Draft
	updateDraftResult   *domain.Draft
	sendDraftResult     *domain.Message

	sendGrantID        string
	sendRequest        *domain.SendMessageRequest
	updateGrantID      string
	updateMessageID    string
	updateRequest      *domain.UpdateMessageRequest
	deleteGrantID      string
	deleteMessageID    string
	createDraftGrantID string
	createDraftRequest *domain.CreateDraftRequest
	updateDraftGrantID string
	updateDraftID      string
	updateDraftRequest *domain.CreateDraftRequest
	deleteDraftGrantID string
	deleteDraftID      string
	sendDraftGrantID   string
	sendDraftID        string
	sendDraftRequest   *domain.SendDraftRequest
}

func (f *fakeEmailWriteClient) SendMessage(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
	f.sendGrantID = grantID
	f.sendRequest = req
	return f.sendMessageResult, f.err
}

func (f *fakeEmailWriteClient) UpdateMessage(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error) {
	f.updateGrantID = grantID
	f.updateMessageID = messageID
	f.updateRequest = req
	return f.updateMessageResult, f.err
}

func (f *fakeEmailWriteClient) DeleteMessage(ctx context.Context, grantID, messageID string) error {
	f.deleteGrantID = grantID
	f.deleteMessageID = messageID
	return f.err
}

func (f *fakeEmailWriteClient) CreateDraft(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	f.createDraftGrantID = grantID
	f.createDraftRequest = req
	return f.createDraftResult, f.err
}

func (f *fakeEmailWriteClient) UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error) {
	f.updateDraftGrantID = grantID
	f.updateDraftID = draftID
	f.updateDraftRequest = req
	return f.updateDraftResult, f.err
}

func (f *fakeEmailWriteClient) DeleteDraft(ctx context.Context, grantID, draftID string) error {
	f.deleteDraftGrantID = grantID
	f.deleteDraftID = draftID
	return f.err
}

func (f *fakeEmailWriteClient) SendDraft(ctx context.Context, grantID, draftID string, req *domain.SendDraftRequest) (*domain.Message, error) {
	f.sendDraftGrantID = grantID
	f.sendDraftID = draftID
	f.sendDraftRequest = req
	return f.sendDraftResult, f.err
}

func TestRegisterEmailWriteHandlers(t *testing.T) {
	unread := false
	starred := true

	tests := []struct {
		name   string
		method string
		params string
		client *fakeEmailWriteClient
		assert func(*testing.T, *fakeEmailWriteClient, rpcTestResponse)
	}{
		{
			name:   "email.send forwards request and returns message",
			method: "email.send",
			params: `{"grant_id":"grant-1","subject":"Hello","body":"World","to":[{"email":"ada@example.com","name":"Ada"}]}`,
			client: &fakeEmailWriteClient{
				sendMessageResult: &domain.Message{ID: "msg-1", Subject: "Hello"},
			},
			assert: func(t *testing.T, client *fakeEmailWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.sendGrantID != "grant-1" {
					t.Fatalf("grantID = %q, want grant-1", client.sendGrantID)
				}
				if client.sendRequest == nil {
					t.Fatal("sendRequest = nil, want request")
				}
				if client.sendRequest.Subject != "Hello" {
					t.Fatalf("Subject = %q, want Hello", client.sendRequest.Subject)
				}
				if len(client.sendRequest.To) != 1 || client.sendRequest.To[0].Email != "ada@example.com" {
					t.Fatalf("To = %#v, want ada@example.com", client.sendRequest.To)
				}

				var msg domain.Message
				unmarshalResult(t, resp, &msg)
				if msg.ID != "msg-1" || msg.Subject != "Hello" {
					t.Fatalf("message = %#v, want msg-1 Hello", msg)
				}
			},
		},
		{
			name:   "email.update forwards request and returns message",
			method: "email.update",
			params: `{"message_id":"msg-1","unread":false,"starred":true,"folders":["sent"]}`,
			client: &fakeEmailWriteClient{
				updateMessageResult: &domain.Message{ID: "msg-1", Starred: true},
			},
			assert: func(t *testing.T, client *fakeEmailWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.updateGrantID != "default-grant" || client.updateMessageID != "msg-1" {
					t.Fatalf("update args = %q/%q, want default-grant/msg-1", client.updateGrantID, client.updateMessageID)
				}
				if client.updateRequest == nil || client.updateRequest.Unread == nil || *client.updateRequest.Unread != unread {
					t.Fatalf("Unread = %#v, want false", client.updateRequest)
				}
				if client.updateRequest.Starred == nil || *client.updateRequest.Starred != starred {
					t.Fatalf("Starred = %#v, want true", client.updateRequest.Starred)
				}
				if len(client.updateRequest.Folders) != 1 || client.updateRequest.Folders[0] != "sent" {
					t.Fatalf("Folders = %#v, want sent", client.updateRequest.Folders)
				}
			},
		},
		{
			name:   "email.delete deletes and returns deleted",
			method: "email.delete",
			params: `{"grant_id":"grant-1","message_id":"msg-1"}`,
			client: &fakeEmailWriteClient{},
			assert: func(t *testing.T, client *fakeEmailWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.deleteGrantID != "grant-1" || client.deleteMessageID != "msg-1" {
					t.Fatalf("delete args = %q/%q, want grant-1/msg-1", client.deleteGrantID, client.deleteMessageID)
				}
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:   "draft.create forwards request and returns draft",
			method: "draft.create",
			params: `{"grant_id":"grant-1","subject":"Draft","body":"Body","to":[{"email":"grace@example.com"}]}`,
			client: &fakeEmailWriteClient{
				createDraftResult: &domain.Draft{ID: "draft-1", Subject: "Draft"},
			},
			assert: func(t *testing.T, client *fakeEmailWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.createDraftGrantID != "grant-1" {
					t.Fatalf("grantID = %q, want grant-1", client.createDraftGrantID)
				}
				if client.createDraftRequest == nil || client.createDraftRequest.Subject != "Draft" {
					t.Fatalf("createDraftRequest = %#v, want subject Draft", client.createDraftRequest)
				}

				var draft domain.Draft
				unmarshalResult(t, resp, &draft)
				if draft.ID != "draft-1" || draft.Subject != "Draft" {
					t.Fatalf("draft = %#v, want draft-1 Draft", draft)
				}
			},
		},
		{
			name:   "draft.update forwards request and returns draft",
			method: "draft.update",
			params: `{"draft_id":"draft-1","subject":"Updated","body":"Body","signature_id":"sig-1"}`,
			client: &fakeEmailWriteClient{
				updateDraftResult: &domain.Draft{ID: "draft-1", Subject: "Updated"},
			},
			assert: func(t *testing.T, client *fakeEmailWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.updateDraftGrantID != "default-grant" || client.updateDraftID != "draft-1" {
					t.Fatalf("update draft args = %q/%q, want default-grant/draft-1", client.updateDraftGrantID, client.updateDraftID)
				}
				if client.updateDraftRequest == nil || client.updateDraftRequest.SignatureID != "sig-1" {
					t.Fatalf("SignatureID = %#v, want sig-1", client.updateDraftRequest)
				}
			},
		},
		{
			name:   "draft.delete deletes and returns deleted",
			method: "draft.delete",
			params: `{"grant_id":"grant-1","draft_id":"draft-1"}`,
			client: &fakeEmailWriteClient{},
			assert: func(t *testing.T, client *fakeEmailWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.deleteDraftGrantID != "grant-1" || client.deleteDraftID != "draft-1" {
					t.Fatalf("delete draft args = %q/%q, want grant-1/draft-1", client.deleteDraftGrantID, client.deleteDraftID)
				}
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:   "draft.send forwards request and returns message",
			method: "draft.send",
			params: `{"draft_id":"draft-1","signature_id":"sig-1"}`,
			client: &fakeEmailWriteClient{
				sendDraftResult: &domain.Message{ID: "msg-1"},
			},
			assert: func(t *testing.T, client *fakeEmailWriteClient, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if client.sendDraftGrantID != "default-grant" || client.sendDraftID != "draft-1" {
					t.Fatalf("send draft args = %q/%q, want default-grant/draft-1", client.sendDraftGrantID, client.sendDraftID)
				}
				if client.sendDraftRequest == nil || client.sendDraftRequest.SignatureID != "sig-1" {
					t.Fatalf("SignatureID = %#v, want sig-1", client.sendDraftRequest)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterEmailWriteHandlers(d, tt.client, "default-grant")

			resp := dispatchEmailRequest(t, d, tt.method, tt.params)
			tt.assert(t, tt.client, resp)
		})
	}
}

func TestRegisterEmailWriteHandlers_InvalidParams(t *testing.T) {
	tests := []struct {
		name   string
		method string
		params string
	}{
		{name: "email.send missing grant", method: "email.send", params: `{}`},
		{name: "email.update missing message_id", method: "email.update", params: `{"grant_id":"grant-1"}`},
		{name: "email.delete missing message_id", method: "email.delete", params: `{"grant_id":"grant-1"}`},
		{name: "draft.update missing draft_id", method: "draft.update", params: `{"grant_id":"grant-1"}`},
		{name: "draft.delete missing draft_id", method: "draft.delete", params: `{"grant_id":"grant-1"}`},
		{name: "draft.send missing draft_id", method: "draft.send", params: `{"grant_id":"grant-1"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterEmailWriteHandlers(d, &fakeEmailWriteClient{}, "")

			resp := dispatchEmailRequest(t, d, tt.method, tt.params)
			requireRPCErrorCode(t, resp, InvalidParams)
		})
	}
}

func TestRegisterEmailWriteHandlers_ClientError(t *testing.T) {
	d := NewDispatcher()
	RegisterEmailWriteHandlers(d, &fakeEmailWriteClient{
		err: errors.New("client unavailable"),
	}, "grant-1")

	resp := dispatchEmailRequest(t, d, "email.delete", `{"message_id":"msg-1"}`)
	requireRPCErrorCode(t, resp, InternalError)
}
