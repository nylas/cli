package rpcserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

type fakeEmailExtClient struct {
	ports.MessageClient

	getFolders         func(context.Context, string) ([]domain.Folder, error)
	getFolder          func(context.Context, string, string) (*domain.Folder, error)
	createFolder       func(context.Context, string, *domain.CreateFolderRequest) (*domain.Folder, error)
	updateFolder       func(context.Context, string, string, *domain.UpdateFolderRequest) (*domain.Folder, error)
	deleteFolder       func(context.Context, string, string) error
	listAttachments    func(context.Context, string, string) ([]domain.Attachment, error)
	getAttachment      func(context.Context, string, string, string) (*domain.Attachment, error)
	downloadAttachment func(context.Context, string, string, string) (io.ReadCloser, error)
	getSignatures      func(context.Context, string) ([]domain.Signature, error)
	getSignature       func(context.Context, string, string) (*domain.Signature, error)
	createSignature    func(context.Context, string, *domain.CreateSignatureRequest) (*domain.Signature, error)
	updateSignature    func(context.Context, string, string, *domain.UpdateSignatureRequest) (*domain.Signature, error)
	deleteSignature    func(context.Context, string, string) error
	listScheduled      func(context.Context, string) ([]domain.ScheduledMessage, error)
	getScheduled       func(context.Context, string, string) (*domain.ScheduledMessage, error)
	cancelScheduled    func(context.Context, string, string) error
	cleanMessages      func(context.Context, string, *domain.CleanMessagesRequest) ([]domain.CleanedMessage, error)
}

func (f *fakeEmailExtClient) GetFolders(ctx context.Context, grantID string) ([]domain.Folder, error) {
	if f.getFolders == nil {
		return nil, errors.New("unexpected GetFolders")
	}
	return f.getFolders(ctx, grantID)
}

func (f *fakeEmailExtClient) GetFolder(ctx context.Context, grantID, folderID string) (*domain.Folder, error) {
	if f.getFolder == nil {
		return nil, errors.New("unexpected GetFolder")
	}
	return f.getFolder(ctx, grantID, folderID)
}

func (f *fakeEmailExtClient) CreateFolder(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error) {
	if f.createFolder == nil {
		return nil, errors.New("unexpected CreateFolder")
	}
	return f.createFolder(ctx, grantID, req)
}

func (f *fakeEmailExtClient) UpdateFolder(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error) {
	if f.updateFolder == nil {
		return nil, errors.New("unexpected UpdateFolder")
	}
	return f.updateFolder(ctx, grantID, folderID, req)
}

func (f *fakeEmailExtClient) DeleteFolder(ctx context.Context, grantID, folderID string) error {
	if f.deleteFolder == nil {
		return errors.New("unexpected DeleteFolder")
	}
	return f.deleteFolder(ctx, grantID, folderID)
}

func (f *fakeEmailExtClient) ListAttachments(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error) {
	if f.listAttachments == nil {
		return nil, errors.New("unexpected ListAttachments")
	}
	return f.listAttachments(ctx, grantID, messageID)
}

func (f *fakeEmailExtClient) GetAttachment(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error) {
	if f.getAttachment == nil {
		return nil, errors.New("unexpected GetAttachment")
	}
	return f.getAttachment(ctx, grantID, messageID, attachmentID)
}

func (f *fakeEmailExtClient) DownloadAttachment(ctx context.Context, grantID, messageID, attachmentID string) (io.ReadCloser, error) {
	if f.downloadAttachment == nil {
		return nil, errors.New("unexpected DownloadAttachment")
	}
	return f.downloadAttachment(ctx, grantID, messageID, attachmentID)
}

func (f *fakeEmailExtClient) GetSignatures(ctx context.Context, grantID string) ([]domain.Signature, error) {
	if f.getSignatures == nil {
		return nil, errors.New("unexpected GetSignatures")
	}
	return f.getSignatures(ctx, grantID)
}

func (f *fakeEmailExtClient) GetSignature(ctx context.Context, grantID, signatureID string) (*domain.Signature, error) {
	if f.getSignature == nil {
		return nil, errors.New("unexpected GetSignature")
	}
	return f.getSignature(ctx, grantID, signatureID)
}

func (f *fakeEmailExtClient) CreateSignature(ctx context.Context, grantID string, req *domain.CreateSignatureRequest) (*domain.Signature, error) {
	if f.createSignature == nil {
		return nil, errors.New("unexpected CreateSignature")
	}
	return f.createSignature(ctx, grantID, req)
}

func (f *fakeEmailExtClient) UpdateSignature(ctx context.Context, grantID, signatureID string, req *domain.UpdateSignatureRequest) (*domain.Signature, error) {
	if f.updateSignature == nil {
		return nil, errors.New("unexpected UpdateSignature")
	}
	return f.updateSignature(ctx, grantID, signatureID, req)
}

func (f *fakeEmailExtClient) DeleteSignature(ctx context.Context, grantID, signatureID string) error {
	if f.deleteSignature == nil {
		return errors.New("unexpected DeleteSignature")
	}
	return f.deleteSignature(ctx, grantID, signatureID)
}

func (f *fakeEmailExtClient) ListScheduledMessages(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error) {
	if f.listScheduled == nil {
		return nil, errors.New("unexpected ListScheduledMessages")
	}
	return f.listScheduled(ctx, grantID)
}

func (f *fakeEmailExtClient) GetScheduledMessage(ctx context.Context, grantID, scheduleID string) (*domain.ScheduledMessage, error) {
	if f.getScheduled == nil {
		return nil, errors.New("unexpected GetScheduledMessage")
	}
	return f.getScheduled(ctx, grantID, scheduleID)
}

func (f *fakeEmailExtClient) CancelScheduledMessage(ctx context.Context, grantID, scheduleID string) error {
	if f.cancelScheduled == nil {
		return errors.New("unexpected CancelScheduledMessage")
	}
	return f.cancelScheduled(ctx, grantID, scheduleID)
}

func (f *fakeEmailExtClient) CleanMessages(ctx context.Context, grantID string, req *domain.CleanMessagesRequest) ([]domain.CleanedMessage, error) {
	if f.cleanMessages == nil {
		return nil, errors.New("unexpected CleanMessages")
	}
	return f.cleanMessages(ctx, grantID, req)
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (t *trackingReadCloser) Close() error {
	t.closed = true
	return nil
}

func TestRegisterEmailExtHandlers(t *testing.T) {
	clientErr := errors.New("client unavailable")

	tests := []struct {
		name         string
		method       string
		params       string
		defaultGrant string
		client       *fakeEmailExtClient
		assert       func(*testing.T, rpcTestResponse)
	}{
		{
			name:         "email.folder.list returns folders",
			method:       "email.folder.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				getFolders: func(_ context.Context, grantID string) ([]domain.Folder, error) {
					if grantID != "default-grant" {
						t.Fatalf("grantID = %q, want default-grant", grantID)
					}
					return []domain.Folder{{ID: "fld-1", Name: "Inbox"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result folderListResult
				unmarshalResult(t, resp, &result)
				if len(result.Folders) != 1 || result.Folders[0].ID != "fld-1" {
					t.Fatalf("folders = %+v, want one fld-1", result.Folders)
				}
			},
		},
		{
			name:         "email.folder.get missing folder_id",
			method:       "email.folder.get",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "email.folder.create returns folder",
			method:       "email.folder.create",
			params:       `{"name":"Receipts"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				createFolder: func(_ context.Context, _ string, req *domain.CreateFolderRequest) (*domain.Folder, error) {
					if req.Name != "Receipts" {
						t.Fatalf("name = %q, want Receipts", req.Name)
					}
					return &domain.Folder{ID: "fld-new", Name: req.Name}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var folder domain.Folder
				unmarshalResult(t, resp, &folder)
				if folder.ID != "fld-new" {
					t.Fatalf("folder ID = %q, want fld-new", folder.ID)
				}
			},
		},
		{
			name:         "email.folder.update missing folder_id",
			method:       "email.folder.update",
			params:       `{"name":"x"}`,
			defaultGrant: "default-grant",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "email.folder.delete returns deleted",
			method:       "email.folder.delete",
			params:       `{"folder_id":"fld-1"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				deleteFolder: func(_ context.Context, _, folderID string) error {
					if folderID != "fld-1" {
						t.Fatalf("folderID = %q, want fld-1", folderID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:         "email.attachment.list missing message_id",
			method:       "email.attachment.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "email.attachment.list returns attachments",
			method:       "email.attachment.list",
			params:       `{"message_id":"msg-1"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				listAttachments: func(_ context.Context, _, messageID string) ([]domain.Attachment, error) {
					if messageID != "msg-1" {
						t.Fatalf("messageID = %q, want msg-1", messageID)
					}
					return []domain.Attachment{{ID: "att-1"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result attachmentListResult
				unmarshalResult(t, resp, &result)
				if len(result.Attachments) != 1 || result.Attachments[0].ID != "att-1" {
					t.Fatalf("attachments = %+v, want one att-1", result.Attachments)
				}
			},
		},
		{
			name:         "email.attachment.get missing attachment_id",
			method:       "email.attachment.get",
			params:       `{"message_id":"msg-1"}`,
			defaultGrant: "default-grant",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "email.attachment.download base64-encodes bytes",
			method:       "email.attachment.download",
			params:       `{"message_id":"msg-1","attachment_id":"att-1"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				downloadAttachment: func(_ context.Context, _, messageID, attachmentID string) (io.ReadCloser, error) {
					if messageID != "msg-1" || attachmentID != "att-1" {
						t.Fatalf("args = %q/%q, want msg-1/att-1", messageID, attachmentID)
					}
					return io.NopCloser(strings.NewReader("hello world")), nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result attachmentDownloadResult
				unmarshalResult(t, resp, &result)
				want := base64.StdEncoding.EncodeToString([]byte("hello world"))
				if result.Content != want {
					t.Fatalf("content = %q, want %q", result.Content, want)
				}
				if result.Size != len("hello world") {
					t.Fatalf("size = %d, want %d", result.Size, len("hello world"))
				}
			},
		},
		{
			name:         "email.signature.list returns signatures",
			method:       "email.signature.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				getSignatures: func(context.Context, string) ([]domain.Signature, error) {
					return []domain.Signature{{ID: "sig-1", Name: "Default"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result signatureListResult
				unmarshalResult(t, resp, &result)
				if len(result.Signatures) != 1 || result.Signatures[0].ID != "sig-1" {
					t.Fatalf("signatures = %+v, want one sig-1", result.Signatures)
				}
			},
		},
		{
			name:         "email.signature.get missing signature_id",
			method:       "email.signature.get",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "email.signature.create returns signature",
			method:       "email.signature.create",
			params:       `{"name":"Sig"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				createSignature: func(context.Context, string, *domain.CreateSignatureRequest) (*domain.Signature, error) {
					return &domain.Signature{ID: "sig-new"}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var sig domain.Signature
				unmarshalResult(t, resp, &sig)
				if sig.ID != "sig-new" {
					t.Fatalf("signature ID = %q, want sig-new", sig.ID)
				}
			},
		},
		{
			name:         "email.signature.update missing signature_id",
			method:       "email.signature.update",
			params:       `{"name":"x"}`,
			defaultGrant: "default-grant",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "email.signature.delete returns deleted",
			method:       "email.signature.delete",
			params:       `{"signature_id":"sig-1"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				deleteSignature: func(_ context.Context, _, signatureID string) error {
					if signatureID != "sig-1" {
						t.Fatalf("signatureID = %q, want sig-1", signatureID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result deletedResult
				unmarshalResult(t, resp, &result)
				if !result.Deleted {
					t.Fatal("deleted = false, want true")
				}
			},
		},
		{
			name:         "email.scheduled.list returns scheduled",
			method:       "email.scheduled.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				listScheduled: func(context.Context, string) ([]domain.ScheduledMessage, error) {
					return []domain.ScheduledMessage{{ScheduleID: "sch-1", Status: "pending"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result scheduledListResult
				unmarshalResult(t, resp, &result)
				if len(result.Scheduled) != 1 || result.Scheduled[0].ScheduleID != "sch-1" {
					t.Fatalf("scheduled = %+v, want one sch-1", result.Scheduled)
				}
			},
		},
		{
			name:         "email.scheduled.get missing schedule_id",
			method:       "email.scheduled.get",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "email.scheduled.cancel returns canceled",
			method:       "email.scheduled.cancel",
			params:       `{"schedule_id":"sch-1"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				cancelScheduled: func(_ context.Context, _, scheduleID string) error {
					if scheduleID != "sch-1" {
						t.Fatalf("scheduleID = %q, want sch-1", scheduleID)
					}
					return nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result cancelledResult
				unmarshalResult(t, resp, &result)
				if !result.Cancelled {
					t.Fatal("cancelled = false, want true")
				}
			},
		},
		{
			name:         "email.clean maps message_ids and returns cleaned messages",
			method:       "email.clean",
			params:       `{"message_ids":["msg-1","msg-2"]}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				cleanMessages: func(_ context.Context, _ string, req *domain.CleanMessagesRequest) ([]domain.CleanedMessage, error) {
					if len(req.MessageIDs) != 2 || req.MessageIDs[0] != "msg-1" {
						t.Fatalf("req.MessageIDs = %v, want [msg-1 msg-2]", req.MessageIDs)
					}
					return []domain.CleanedMessage{{ID: "msg-1", Conversation: "clean text"}}, nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				var result cleanResult
				unmarshalResult(t, resp, &result)
				if len(result.Messages) != 1 || result.Messages[0].ID != "msg-1" {
					t.Fatalf("messages = %+v, want one msg-1", result.Messages)
				}
			},
		},
		{
			name:         "email.clean without message_ids errors",
			method:       "email.clean",
			params:       `{}`,
			defaultGrant: "default-grant",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "email.attachment.download rejects oversized attachment",
			method:       "email.attachment.download",
			params:       `{"message_id":"msg-1","attachment_id":"att-big"}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				downloadAttachment: func(context.Context, string, string, string) (io.ReadCloser, error) {
					// One byte past the cap.
					return io.NopCloser(strings.NewReader(strings.Repeat("a", maxAttachmentDownloadBytes+1))), nil
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
		{
			name:         "client error surfaces as internal error",
			method:       "email.folder.list",
			params:       `{}`,
			defaultGrant: "default-grant",
			client: &fakeEmailExtClient{
				getFolders: func(context.Context, string) ([]domain.Folder, error) {
					return nil, clientErr
				},
			},
			assert: func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InternalError) },
		},
		{
			name:         "missing default grant errors",
			method:       "email.folder.list",
			params:       `{}`,
			defaultGrant: "",
			client:       &fakeEmailExtClient{},
			assert:       func(t *testing.T, resp rpcTestResponse) { requireRPCErrorCode(t, resp, InvalidParams) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterEmailExtHandlers(d, tt.client, tt.defaultGrant)

			raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"` + tt.method + `","params":` + tt.params + `}`)
			got := d.Dispatch(context.Background(), raw)
			if got == nil {
				t.Fatal("Dispatch() = nil, want response")
			}
			var resp rpcTestResponse
			if err := json.Unmarshal(got, &resp); err != nil {
				t.Fatalf("unmarshal response: %v", err)
			}
			tt.assert(t, resp)
		})
	}
}

func TestRegisterEmailExtHandlers_AttachmentDownloadClosesBody(t *testing.T) {
	tracker := &trackingReadCloser{Reader: strings.NewReader("data")}
	client := &fakeEmailExtClient{
		downloadAttachment: func(context.Context, string, string, string) (io.ReadCloser, error) {
			return tracker, nil
		},
	}

	d := NewDispatcher()
	RegisterEmailExtHandlers(d, client, "default-grant")

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"email.attachment.download","params":{"message_id":"msg-1","attachment_id":"att-1"}}`)
	got := d.Dispatch(context.Background(), raw)
	if got == nil {
		t.Fatal("Dispatch() = nil, want response")
	}
	var resp rpcTestResponse
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	requireNoRPCError(t, resp)
	if !tracker.closed {
		t.Fatal("download body was not closed")
	}
}
