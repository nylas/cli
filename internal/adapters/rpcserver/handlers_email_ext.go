package rpcserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// maxAttachmentDownloadBytes caps email.attachment.download: the bytes are
// base64-encoded into one JSON-RPC response, so the whole attachment is held in
// memory. 30 MiB comfortably covers provider attachment limits.
const maxAttachmentDownloadBytes = 30 << 20

type emailGrantParams struct {
	GrantID string `json:"grant_id,omitempty"`
}

type folderListResult struct {
	Folders []domain.Folder `json:"folders"`
}

type folderGetParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	FolderID string `json:"folder_id"`
}

type folderCreateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateFolderRequest
}

type folderUpdateParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	FolderID string `json:"folder_id"`
	domain.UpdateFolderRequest
}

type folderDeleteParams struct {
	GrantID  string `json:"grant_id,omitempty"`
	FolderID string `json:"folder_id"`
}

type attachmentListParams struct {
	GrantID   string `json:"grant_id,omitempty"`
	MessageID string `json:"message_id"`
}

type attachmentListResult struct {
	Attachments []domain.Attachment `json:"attachments"`
}

type attachmentGetParams struct {
	GrantID      string `json:"grant_id,omitempty"`
	MessageID    string `json:"message_id"`
	AttachmentID string `json:"attachment_id"`
}

type attachmentDownloadResult struct {
	Content string `json:"content"` // base64-encoded attachment bytes
	Size    int    `json:"size"`
}

type signatureListResult struct {
	Signatures []domain.Signature `json:"signatures"`
}

type signatureGetParams struct {
	GrantID     string `json:"grant_id,omitempty"`
	SignatureID string `json:"signature_id"`
}

type signatureCreateParams struct {
	GrantID string `json:"grant_id,omitempty"`
	domain.CreateSignatureRequest
}

type signatureUpdateParams struct {
	GrantID     string `json:"grant_id,omitempty"`
	SignatureID string `json:"signature_id"`
	domain.UpdateSignatureRequest
}

type signatureDeleteParams struct {
	GrantID     string `json:"grant_id,omitempty"`
	SignatureID string `json:"signature_id"`
}

type scheduledListResult struct {
	Scheduled []domain.ScheduledMessage `json:"scheduled"`
}

type scheduledGetParams struct {
	GrantID    string `json:"grant_id,omitempty"`
	ScheduleID string `json:"schedule_id"`
}

// cleanParams uses message_ids (plural) for the RPC contract; the embedded
// domain request tags the same slice message_id, which is confusing over the
// wire, so the IDs are accepted explicitly and copied into the request.
type cleanParams struct {
	GrantID                 string   `json:"grant_id,omitempty"`
	MessageIDs              []string `json:"message_ids"`
	IgnoreLinks             *bool    `json:"ignore_links,omitempty"`
	IgnoreImages            *bool    `json:"ignore_images,omitempty"`
	IgnoreTables            *bool    `json:"ignore_tables,omitempty"`
	ImagesAsMarkdown        *bool    `json:"images_as_markdown,omitempty"`
	RemoveConclusionPhrases *bool    `json:"remove_conclusion_phrases,omitempty"`
}

type cleanResult struct {
	Messages []domain.CleanedMessage `json:"messages"`
}

type cancelledResult struct {
	Cancelled bool `json:"cancelled"`
}

// RegisterEmailExtHandlers registers folder, attachment, signature, scheduled
// message, and message-clean methods.
func RegisterEmailExtHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	registerEmailFolderHandlers(d, client, defaultGrant)
	registerEmailAttachmentHandlers(d, client, defaultGrant)
	registerEmailSignatureHandlers(d, client, defaultGrant)
	registerEmailScheduledHandlers(d, client, defaultGrant)

	d.Register("email.clean", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p cleanParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		if len(p.MessageIDs) == 0 {
			return nil, NewRPCError(InvalidParams, "message_ids required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		messages, err := client.CleanMessages(ctx, grantID, &domain.CleanMessagesRequest{
			MessageIDs:              p.MessageIDs,
			IgnoreLinks:             p.IgnoreLinks,
			IgnoreImages:            p.IgnoreImages,
			IgnoreTables:            p.IgnoreTables,
			ImagesAsMarkdown:        p.ImagesAsMarkdown,
			RemoveConclusionPhrases: p.RemoveConclusionPhrases,
		})
		if err != nil {
			return nil, fmt.Errorf("email.clean: %w", err)
		}
		return cleanResult{Messages: messages}, nil
	})
}

func registerEmailFolderHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("email.folder.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p emailGrantParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		folders, err := client.GetFolders(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("email.folder.list: %w", err)
		}
		return folderListResult{Folders: folders}, nil
	})

	d.Register("email.folder.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p folderGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.FolderID == "" {
			return nil, NewRPCError(InvalidParams, "folder_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		folder, err := client.GetFolder(ctx, grantID, p.FolderID)
		if err != nil {
			return nil, fmt.Errorf("email.folder.get: %w", err)
		}
		return folder, nil
	})

	d.Register("email.folder.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p folderCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		folder, err := client.CreateFolder(ctx, grantID, &p.CreateFolderRequest)
		if err != nil {
			return nil, fmt.Errorf("email.folder.create: %w", err)
		}
		return folder, nil
	})

	d.Register("email.folder.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p folderUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.FolderID == "" {
			return nil, NewRPCError(InvalidParams, "folder_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		folder, err := client.UpdateFolder(ctx, grantID, p.FolderID, &p.UpdateFolderRequest)
		if err != nil {
			return nil, fmt.Errorf("email.folder.update: %w", err)
		}
		return folder, nil
	})

	d.Register("email.folder.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p folderDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.FolderID == "" {
			return nil, NewRPCError(InvalidParams, "folder_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteFolder(ctx, grantID, p.FolderID); err != nil {
			return nil, fmt.Errorf("email.folder.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})
}

func registerEmailAttachmentHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("email.attachment.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p attachmentListParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.MessageID == "" {
			return nil, NewRPCError(InvalidParams, "message_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		attachments, err := client.ListAttachments(ctx, grantID, p.MessageID)
		if err != nil {
			return nil, fmt.Errorf("email.attachment.list: %w", err)
		}
		return attachmentListResult{Attachments: attachments}, nil
	})

	d.Register("email.attachment.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p attachmentGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.MessageID == "" {
			return nil, NewRPCError(InvalidParams, "message_id required", nil)
		}
		if p.AttachmentID == "" {
			return nil, NewRPCError(InvalidParams, "attachment_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		attachment, err := client.GetAttachment(ctx, grantID, p.MessageID, p.AttachmentID)
		if err != nil {
			return nil, fmt.Errorf("email.attachment.get: %w", err)
		}
		return attachment, nil
	})

	d.Register("email.attachment.download", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p attachmentGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.MessageID == "" {
			return nil, NewRPCError(InvalidParams, "message_id required", nil)
		}
		if p.AttachmentID == "" {
			return nil, NewRPCError(InvalidParams, "attachment_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		body, err := client.DownloadAttachment(ctx, grantID, p.MessageID, p.AttachmentID)
		if err != nil {
			return nil, fmt.Errorf("email.attachment.download: %w", err)
		}
		defer func() { _ = body.Close() }()

		// Cap the in-memory buffer: the whole attachment is base64-encoded into a
		// single JSON-RPC response, so an oversized attachment would balloon heap.
		data, err := io.ReadAll(io.LimitReader(body, maxAttachmentDownloadBytes+1))
		if err != nil {
			return nil, fmt.Errorf("email.attachment.download: read body: %w", err)
		}
		if len(data) > maxAttachmentDownloadBytes {
			return nil, NewRPCError(InvalidParams, "attachment exceeds maximum download size", nil)
		}
		return attachmentDownloadResult{
			Content: base64.StdEncoding.EncodeToString(data),
			Size:    len(data),
		}, nil
	})
}

func registerEmailSignatureHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("email.signature.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p emailGrantParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		signatures, err := client.GetSignatures(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("email.signature.list: %w", err)
		}
		return signatureListResult{Signatures: signatures}, nil
	})

	d.Register("email.signature.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p signatureGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.SignatureID == "" {
			return nil, NewRPCError(InvalidParams, "signature_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		signature, err := client.GetSignature(ctx, grantID, p.SignatureID)
		if err != nil {
			return nil, fmt.Errorf("email.signature.get: %w", err)
		}
		return signature, nil
	})

	d.Register("email.signature.create", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p signatureCreateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		signature, err := client.CreateSignature(ctx, grantID, &p.CreateSignatureRequest)
		if err != nil {
			return nil, fmt.Errorf("email.signature.create: %w", err)
		}
		return signature, nil
	})

	d.Register("email.signature.update", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p signatureUpdateParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.SignatureID == "" {
			return nil, NewRPCError(InvalidParams, "signature_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		signature, err := client.UpdateSignature(ctx, grantID, p.SignatureID, &p.UpdateSignatureRequest)
		if err != nil {
			return nil, fmt.Errorf("email.signature.update: %w", err)
		}
		return signature, nil
	})

	d.Register("email.signature.delete", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p signatureDeleteParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.SignatureID == "" {
			return nil, NewRPCError(InvalidParams, "signature_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.DeleteSignature(ctx, grantID, p.SignatureID); err != nil {
			return nil, fmt.Errorf("email.signature.delete: %w", err)
		}
		return deletedResult{Deleted: true}, nil
	})
}

func registerEmailScheduledHandlers(d *Dispatcher, client ports.MessageClient, defaultGrant string) {
	d.Register("email.scheduled.list", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p emailGrantParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		scheduled, err := client.ListScheduledMessages(ctx, grantID)
		if err != nil {
			return nil, fmt.Errorf("email.scheduled.list: %w", err)
		}
		return scheduledListResult{Scheduled: scheduled}, nil
	})

	d.Register("email.scheduled.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p scheduledGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ScheduleID == "" {
			return nil, NewRPCError(InvalidParams, "schedule_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		scheduled, err := client.GetScheduledMessage(ctx, grantID, p.ScheduleID)
		if err != nil {
			return nil, fmt.Errorf("email.scheduled.get: %w", err)
		}
		return scheduled, nil
	})

	d.Register("email.scheduled.cancel", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p scheduledGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}
		if p.ScheduleID == "" {
			return nil, NewRPCError(InvalidParams, "schedule_id required", nil)
		}

		grantID, err := resolveGrant(p.GrantID, defaultGrant)
		if err != nil {
			return nil, err
		}

		if err := client.CancelScheduledMessage(ctx, grantID, p.ScheduleID); err != nil {
			return nil, fmt.Errorf("email.scheduled.cancel: %w", err)
		}
		return cancelledResult{Cancelled: true}, nil
	})
}
