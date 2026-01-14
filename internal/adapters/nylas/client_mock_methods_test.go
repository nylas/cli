package nylas_test

import (
	"context"
	"io"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test mock client implements interface

func TestMockClient_Messages(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	t.Run("GetMessages", func(t *testing.T) {
		mock.GetMessagesFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
			return []domain.Message{
				{ID: "msg-1", Subject: "Test 1"},
				{ID: "msg-2", Subject: "Test 2"},
			}, nil
		}

		messages, err := mock.GetMessages(ctx, "grant-123", 10)
		require.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.True(t, mock.GetMessagesCalled)
		assert.Equal(t, "grant-123", mock.LastGrantID)
	})

	t.Run("GetMessagesWithParams", func(t *testing.T) {
		unread := true
		params := &domain.MessageQueryParams{
			Limit:  5,
			Unread: &unread,
			From:   "sender@example.com",
		}

		mock.GetMessagesWithParamsFunc = func(ctx context.Context, grantID string, p *domain.MessageQueryParams) ([]domain.Message, error) {
			assert.Equal(t, params, p)
			return []domain.Message{{ID: "msg-1"}}, nil
		}

		messages, err := mock.GetMessagesWithParams(ctx, "grant-123", params)
		require.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.True(t, mock.GetMessagesWithParamsCalled)
	})

	t.Run("GetMessage", func(t *testing.T) {
		msg, err := mock.GetMessage(ctx, "grant-123", "msg-456")
		require.NoError(t, err)
		assert.Equal(t, "msg-456", msg.ID)
		assert.True(t, mock.GetMessageCalled)
		assert.Equal(t, "msg-456", mock.LastMessageID)
	})

	t.Run("SendMessage", func(t *testing.T) {
		req := &domain.SendMessageRequest{
			Subject: "Test Subject",
			Body:    "Test Body",
			To:      []domain.EmailParticipant{{Email: "recipient@example.com"}},
		}

		msg, err := mock.SendMessage(ctx, "grant-123", req)
		require.NoError(t, err)
		assert.Equal(t, "Test Subject", msg.Subject)
		assert.True(t, mock.SendMessageCalled)
	})

	t.Run("UpdateMessage", func(t *testing.T) {
		unread := false
		starred := true
		req := &domain.UpdateMessageRequest{
			Unread:  &unread,
			Starred: &starred,
		}

		msg, err := mock.UpdateMessage(ctx, "grant-123", "msg-456", req)
		require.NoError(t, err)
		assert.False(t, msg.Unread)
		assert.True(t, msg.Starred)
		assert.True(t, mock.UpdateMessageCalled)
	})

	t.Run("DeleteMessage", func(t *testing.T) {
		err := mock.DeleteMessage(ctx, "grant-123", "msg-456")
		require.NoError(t, err)
		assert.True(t, mock.DeleteMessageCalled)
	})
}

func TestMockClient_Threads(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	t.Run("GetThreads", func(t *testing.T) {
		mock.GetThreadsFunc = func(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error) {
			return []domain.Thread{
				{ID: "thread-1", Subject: "Thread 1"},
				{ID: "thread-2", Subject: "Thread 2"},
			}, nil
		}

		threads, err := mock.GetThreads(ctx, "grant-123", nil)
		require.NoError(t, err)
		assert.Len(t, threads, 2)
		assert.True(t, mock.GetThreadsCalled)
	})

	t.Run("GetThread", func(t *testing.T) {
		thread, err := mock.GetThread(ctx, "grant-123", "thread-456")
		require.NoError(t, err)
		assert.Equal(t, "thread-456", thread.ID)
		assert.True(t, mock.GetThreadCalled)
		assert.Equal(t, "thread-456", mock.LastThreadID)
	})

	t.Run("UpdateThread", func(t *testing.T) {
		unread := false
		req := &domain.UpdateMessageRequest{
			Unread: &unread,
		}

		thread, err := mock.UpdateThread(ctx, "grant-123", "thread-456", req)
		require.NoError(t, err)
		assert.False(t, thread.Unread)
		assert.True(t, mock.UpdateThreadCalled)
	})

	t.Run("DeleteThread", func(t *testing.T) {
		err := mock.DeleteThread(ctx, "grant-123", "thread-456")
		require.NoError(t, err)
		assert.True(t, mock.DeleteThreadCalled)
	})
}

func TestMockClient_Drafts(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	t.Run("GetDrafts", func(t *testing.T) {
		mock.GetDraftsFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Draft, error) {
			return []domain.Draft{
				{ID: "draft-1", Subject: "Draft 1"},
			}, nil
		}

		drafts, err := mock.GetDrafts(ctx, "grant-123", 10)
		require.NoError(t, err)
		assert.Len(t, drafts, 1)
		assert.True(t, mock.GetDraftsCalled)
	})

	t.Run("GetDraft", func(t *testing.T) {
		draft, err := mock.GetDraft(ctx, "grant-123", "draft-456")
		require.NoError(t, err)
		assert.Equal(t, "draft-456", draft.ID)
		assert.True(t, mock.GetDraftCalled)
	})

	t.Run("CreateDraft", func(t *testing.T) {
		req := &domain.CreateDraftRequest{
			Subject: "New Draft",
			Body:    "Draft body",
			To:      []domain.EmailParticipant{{Email: "to@example.com"}},
		}

		draft, err := mock.CreateDraft(ctx, "grant-123", req)
		require.NoError(t, err)
		assert.Equal(t, "New Draft", draft.Subject)
		assert.True(t, mock.CreateDraftCalled)
	})

	t.Run("UpdateDraft", func(t *testing.T) {
		req := &domain.CreateDraftRequest{
			Subject: "Updated Draft",
			Body:    "Updated body",
		}

		draft, err := mock.UpdateDraft(ctx, "grant-123", "draft-456", req)
		require.NoError(t, err)
		assert.Equal(t, "Updated Draft", draft.Subject)
		assert.True(t, mock.UpdateDraftCalled)
	})

	t.Run("DeleteDraft", func(t *testing.T) {
		err := mock.DeleteDraft(ctx, "grant-123", "draft-456")
		require.NoError(t, err)
		assert.True(t, mock.DeleteDraftCalled)
	})

	t.Run("SendDraft", func(t *testing.T) {
		msg, err := mock.SendDraft(ctx, "grant-123", "draft-456")
		require.NoError(t, err)
		assert.NotEmpty(t, msg.ID)
		assert.True(t, mock.SendDraftCalled)
	})
}

func TestMockClient_Folders(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	t.Run("GetFolders", func(t *testing.T) {
		folders, err := mock.GetFolders(ctx, "grant-123")
		require.NoError(t, err)
		assert.Len(t, folders, 3) // Default mock returns inbox, sent, drafts
		assert.True(t, mock.GetFoldersCalled)
	})

	t.Run("GetFolder", func(t *testing.T) {
		folder, err := mock.GetFolder(ctx, "grant-123", "folder-456")
		require.NoError(t, err)
		assert.Equal(t, "folder-456", folder.ID)
		assert.True(t, mock.GetFolderCalled)
	})

	t.Run("CreateFolder", func(t *testing.T) {
		req := &domain.CreateFolderRequest{
			Name: "New Folder",
		}

		folder, err := mock.CreateFolder(ctx, "grant-123", req)
		require.NoError(t, err)
		assert.Equal(t, "New Folder", folder.Name)
		assert.True(t, mock.CreateFolderCalled)
	})

	t.Run("UpdateFolder", func(t *testing.T) {
		req := &domain.UpdateFolderRequest{
			Name: "Renamed Folder",
		}

		folder, err := mock.UpdateFolder(ctx, "grant-123", "folder-456", req)
		require.NoError(t, err)
		assert.Equal(t, "Renamed Folder", folder.Name)
		assert.True(t, mock.UpdateFolderCalled)
	})

	t.Run("DeleteFolder", func(t *testing.T) {
		err := mock.DeleteFolder(ctx, "grant-123", "folder-456")
		require.NoError(t, err)
		assert.True(t, mock.DeleteFolderCalled)
	})
}

func TestMockClient_Attachments(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	t.Run("GetAttachment", func(t *testing.T) {
		attachment, err := mock.GetAttachment(ctx, "grant-123", "msg-789", "attach-456")
		require.NoError(t, err)
		assert.Equal(t, "attach-456", attachment.ID)
		assert.Equal(t, "test.pdf", attachment.Filename)
		assert.True(t, mock.GetAttachmentCalled)
	})

	t.Run("DownloadAttachment", func(t *testing.T) {
		reader, err := mock.DownloadAttachment(ctx, "grant-123", "msg-789", "attach-456")
		require.NoError(t, err)
		defer func() { _ = reader.Close() }()

		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		assert.Equal(t, "mock attachment content", string(content))
		assert.True(t, mock.DownloadAttachmentCalled)
	})
}

func TestMockClient_Grants(t *testing.T) {
	ctx := context.Background()
	mock := nylas.NewMockClient()

	t.Run("ExchangeCode", func(t *testing.T) {
		grant, err := mock.ExchangeCode(ctx, "auth-code", "http://localhost")
		require.NoError(t, err)
		assert.Equal(t, "mock-grant-id", grant.ID)
		assert.True(t, mock.ExchangeCodeCalled)
	})

	t.Run("ListGrants", func(t *testing.T) {
		mock.ListGrantsFunc = func(ctx context.Context) ([]domain.Grant, error) {
			return []domain.Grant{
				{ID: "grant-1", Email: "user1@example.com"},
				{ID: "grant-2", Email: "user2@example.com"},
			}, nil
		}

		grants, err := mock.ListGrants(ctx)
		require.NoError(t, err)
		assert.Len(t, grants, 2)
		assert.True(t, mock.ListGrantsCalled)
	})

	t.Run("GetGrant", func(t *testing.T) {
		grant, err := mock.GetGrant(ctx, "grant-123")
		require.NoError(t, err)
		assert.Equal(t, "grant-123", grant.ID)
		assert.True(t, mock.GetGrantCalled)
	})

	t.Run("RevokeGrant", func(t *testing.T) {
		err := mock.RevokeGrant(ctx, "grant-123")
		require.NoError(t, err)
		assert.True(t, mock.RevokeGrantCalled)
	})
}
