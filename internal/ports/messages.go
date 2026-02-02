package ports

import (
	"context"
	"io"

	"github.com/nylas/cli/internal/domain"
)

// MessageClient defines the interface for message, draft, thread, folder, and attachment operations.
type MessageClient interface {
	// ================================
	// MESSAGE OPERATIONS
	// ================================

	// GetMessages retrieves messages with optional limit.
	GetMessages(ctx context.Context, grantID string, limit int) ([]domain.Message, error)

	// GetMessagesWithParams retrieves messages with query parameters.
	GetMessagesWithParams(ctx context.Context, grantID string, params *domain.MessageQueryParams) ([]domain.Message, error)

	// GetMessagesWithCursor retrieves messages with cursor-based pagination.
	GetMessagesWithCursor(ctx context.Context, grantID string, params *domain.MessageQueryParams) (*domain.MessageListResponse, error)

	// GetMessage retrieves a specific message.
	GetMessage(ctx context.Context, grantID, messageID string) (*domain.Message, error)

	// GetMessageWithFields retrieves a message with optional field selection (e.g., "raw_mime").
	GetMessageWithFields(ctx context.Context, grantID, messageID string, fields string) (*domain.Message, error)

	// SendMessage sends a new message.
	SendMessage(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error)

	// SendRawMessage sends a raw RFC 822 MIME message.
	SendRawMessage(ctx context.Context, grantID string, rawMIME []byte) (*domain.Message, error)

	// UpdateMessage updates an existing message.
	UpdateMessage(ctx context.Context, grantID, messageID string, req *domain.UpdateMessageRequest) (*domain.Message, error)

	// DeleteMessage deletes a message.
	DeleteMessage(ctx context.Context, grantID, messageID string) error

	// ================================
	// SCHEDULED MESSAGE OPERATIONS
	// ================================

	// ListScheduledMessages retrieves all scheduled messages.
	ListScheduledMessages(ctx context.Context, grantID string) ([]domain.ScheduledMessage, error)

	// GetScheduledMessage retrieves a specific scheduled message.
	GetScheduledMessage(ctx context.Context, grantID, scheduleID string) (*domain.ScheduledMessage, error)

	// CancelScheduledMessage cancels a scheduled message.
	CancelScheduledMessage(ctx context.Context, grantID, scheduleID string) error

	// ================================
	// SMART COMPOSE OPERATIONS
	// ================================

	// SmartCompose generates AI-powered message suggestions.
	SmartCompose(ctx context.Context, grantID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error)

	// SmartComposeReply generates AI-powered reply suggestions.
	SmartComposeReply(ctx context.Context, grantID, messageID string, req *domain.SmartComposeRequest) (*domain.SmartComposeSuggestion, error)

	// ================================
	// THREAD OPERATIONS
	// ================================

	// GetThreads retrieves threads with query parameters.
	GetThreads(ctx context.Context, grantID string, params *domain.ThreadQueryParams) ([]domain.Thread, error)

	// GetThread retrieves a specific thread.
	GetThread(ctx context.Context, grantID, threadID string) (*domain.Thread, error)

	// UpdateThread updates a thread.
	UpdateThread(ctx context.Context, grantID, threadID string, req *domain.UpdateMessageRequest) (*domain.Thread, error)

	// DeleteThread deletes a thread.
	DeleteThread(ctx context.Context, grantID, threadID string) error

	// ================================
	// DRAFT OPERATIONS
	// ================================

	// GetDrafts retrieves drafts with optional limit.
	GetDrafts(ctx context.Context, grantID string, limit int) ([]domain.Draft, error)

	// GetDraft retrieves a specific draft.
	GetDraft(ctx context.Context, grantID, draftID string) (*domain.Draft, error)

	// CreateDraft creates a new draft.
	CreateDraft(ctx context.Context, grantID string, req *domain.CreateDraftRequest) (*domain.Draft, error)

	// UpdateDraft updates an existing draft.
	UpdateDraft(ctx context.Context, grantID, draftID string, req *domain.CreateDraftRequest) (*domain.Draft, error)

	// DeleteDraft deletes a draft.
	DeleteDraft(ctx context.Context, grantID, draftID string) error

	// SendDraft sends a draft as a message.
	SendDraft(ctx context.Context, grantID, draftID string) (*domain.Message, error)

	// ================================
	// FOLDER OPERATIONS
	// ================================

	// GetFolders retrieves all folders.
	GetFolders(ctx context.Context, grantID string) ([]domain.Folder, error)

	// GetFolder retrieves a specific folder.
	GetFolder(ctx context.Context, grantID, folderID string) (*domain.Folder, error)

	// CreateFolder creates a new folder.
	CreateFolder(ctx context.Context, grantID string, req *domain.CreateFolderRequest) (*domain.Folder, error)

	// UpdateFolder updates an existing folder.
	UpdateFolder(ctx context.Context, grantID, folderID string, req *domain.UpdateFolderRequest) (*domain.Folder, error)

	// DeleteFolder deletes a folder.
	DeleteFolder(ctx context.Context, grantID, folderID string) error

	// ================================
	// ATTACHMENT OPERATIONS
	// ================================

	// ListAttachments retrieves all attachments for a message.
	ListAttachments(ctx context.Context, grantID, messageID string) ([]domain.Attachment, error)

	// GetAttachment retrieves a specific attachment.
	GetAttachment(ctx context.Context, grantID, messageID, attachmentID string) (*domain.Attachment, error)

	// DownloadAttachment downloads attachment content.
	DownloadAttachment(ctx context.Context, grantID, messageID, attachmentID string) (io.ReadCloser, error)
}
