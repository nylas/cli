package domain

import (
	"testing"
	"time"
)

// =============================================================================
// Thread Tests
// =============================================================================

func TestThread_Creation(t *testing.T) {
	now := time.Now()
	thread := Thread{
		ID:                    "thread-123",
		GrantID:               "grant-456",
		HasAttachments:        true,
		HasDrafts:             false,
		Starred:               true,
		Unread:                true,
		EarliestMessageDate:   now.AddDate(0, 0, -7),
		LatestMessageRecvDate: now.AddDate(0, 0, -1),
		LatestMessageSentDate: now.AddDate(0, 0, -2),
		Participants: []EmailParticipant{
			{Name: "John Doe", Email: "john@example.com"},
			{Name: "Jane Smith", Email: "jane@example.com"},
		},
		MessageIDs: []string{"msg-1", "msg-2", "msg-3"},
		DraftIDs:   []string{},
		FolderIDs:  []string{"inbox"},
		Snippet:    "Here's the latest update on the project...",
		Subject:    "Project Update",
	}

	if thread.ID != "thread-123" {
		t.Errorf("Thread.ID = %q, want %q", thread.ID, "thread-123")
	}
	if !thread.HasAttachments {
		t.Error("Thread.HasAttachments should be true")
	}
	if !thread.Starred {
		t.Error("Thread.Starred should be true")
	}
	if len(thread.Participants) != 2 {
		t.Errorf("Thread.Participants length = %d, want 2", len(thread.Participants))
	}
	if len(thread.MessageIDs) != 3 {
		t.Errorf("Thread.MessageIDs length = %d, want 3", len(thread.MessageIDs))
	}
}

// =============================================================================
// Draft Tests
// =============================================================================

func TestDraft_Creation(t *testing.T) {
	now := time.Now()
	draft := Draft{
		ID:      "draft-123",
		GrantID: "grant-456",
		Subject: "Meeting Follow-up",
		Body:    "<p>Thanks for the meeting today!</p>",
		From: []EmailParticipant{
			{Name: "Sender", Email: "sender@example.com"},
		},
		To: []EmailParticipant{
			{Name: "Recipient", Email: "recipient@example.com"},
		},
		Cc: []EmailParticipant{
			{Name: "CC Person", Email: "cc@example.com"},
		},
		ReplyToMsgID: "original-msg-123",
		ThreadID:     "thread-456",
		Attachments: []Attachment{
			{Filename: "report.pdf", ContentType: "application/pdf", Size: 1024},
		},
		CreatedAt: now.Add(-1 * time.Hour),
		UpdatedAt: now,
	}

	if draft.Subject != "Meeting Follow-up" {
		t.Errorf("Draft.Subject = %q, want %q", draft.Subject, "Meeting Follow-up")
	}
	if len(draft.To) != 1 {
		t.Errorf("Draft.To length = %d, want 1", len(draft.To))
	}
	if len(draft.Cc) != 1 {
		t.Errorf("Draft.Cc length = %d, want 1", len(draft.Cc))
	}
	if len(draft.Attachments) != 1 {
		t.Errorf("Draft.Attachments length = %d, want 1", len(draft.Attachments))
	}
}

// =============================================================================
// Folder Tests
// =============================================================================

func TestFolder_Creation(t *testing.T) {
	folder := Folder{
		ID:              "folder-123",
		GrantID:         "grant-456",
		Name:            "Important",
		SystemFolder:    "",
		ParentID:        "parent-folder",
		BackgroundColor: "#ff0000",
		TextColor:       "#ffffff",
		TotalCount:      150,
		UnreadCount:     12,
		ChildIDs:        []string{"child-1", "child-2"},
		Attributes:      []string{"user_created"},
	}

	if folder.Name != "Important" {
		t.Errorf("Folder.Name = %q, want %q", folder.Name, "Important")
	}
	if folder.TotalCount != 150 {
		t.Errorf("Folder.TotalCount = %d, want 150", folder.TotalCount)
	}
	if folder.UnreadCount != 12 {
		t.Errorf("Folder.UnreadCount = %d, want 12", folder.UnreadCount)
	}
	if len(folder.ChildIDs) != 2 {
		t.Errorf("Folder.ChildIDs length = %d, want 2", len(folder.ChildIDs))
	}
}

func TestSystemFolderConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"inbox", FolderInbox, "inbox"},
		{"sent", FolderSent, "sent"},
		{"drafts", FolderDrafts, "drafts"},
		{"trash", FolderTrash, "trash"},
		{"spam", FolderSpam, "spam"},
		{"archive", FolderArchive, "archive"},
		{"all", FolderAll, "all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("Folder constant = %q, want %q", tt.constant, tt.want)
			}
		})
	}
}

// =============================================================================
// Attachment Tests
// =============================================================================

func TestAttachment_Creation(t *testing.T) {
	attachment := Attachment{
		ID:          "attach-123",
		GrantID:     "grant-456",
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Size:        2048,
		ContentID:   "cid:image001",
		IsInline:    true,
		Content:     []byte{0x25, 0x50, 0x44, 0x46}, // PDF magic bytes
	}

	if attachment.Filename != "document.pdf" {
		t.Errorf("Attachment.Filename = %q, want %q", attachment.Filename, "document.pdf")
	}
	if attachment.ContentType != "application/pdf" {
		t.Errorf("Attachment.ContentType = %q, want %q", attachment.ContentType, "application/pdf")
	}
	if attachment.Size != 2048 {
		t.Errorf("Attachment.Size = %d, want 2048", attachment.Size)
	}
	if !attachment.IsInline {
		t.Error("Attachment.IsInline should be true")
	}
	if len(attachment.Content) != 4 {
		t.Errorf("Attachment.Content length = %d, want 4", len(attachment.Content))
	}
}

// =============================================================================
// SendMessageRequest Tests
// =============================================================================

func TestSendMessageRequest_Creation(t *testing.T) {
	req := SendMessageRequest{
		Subject: "Test Email",
		Body:    "<p>Hello World</p>",
		From: []EmailParticipant{
			{Name: "Sender", Email: "sender@example.com"},
		},
		To: []EmailParticipant{
			{Name: "Recipient", Email: "recipient@example.com"},
		},
		Cc: []EmailParticipant{
			{Name: "CC", Email: "cc@example.com"},
		},
		Bcc: []EmailParticipant{
			{Name: "BCC", Email: "bcc@example.com"},
		},
		ReplyTo: []EmailParticipant{
			{Name: "Reply To", Email: "reply@example.com"},
		},
		ReplyToMsgID: "msg-123",
		TrackingOpts: &TrackingOptions{
			Opens: true,
			Links: true,
			Label: "campaign-123",
		},
		Attachments: []Attachment{
			{Filename: "file.txt", ContentType: "text/plain", Size: 100},
		},
		SendAt: 1704067200,
		Metadata: map[string]string{
			"campaign_id": "camp-123",
		},
	}

	if req.Subject != "Test Email" {
		t.Errorf("SendMessageRequest.Subject = %q, want %q", req.Subject, "Test Email")
	}
	if len(req.To) != 1 {
		t.Errorf("SendMessageRequest.To length = %d, want 1", len(req.To))
	}
	if req.TrackingOpts == nil {
		t.Fatal("SendMessageRequest.TrackingOpts should not be nil")
	}
	if !req.TrackingOpts.Opens {
		t.Error("TrackingOptions.Opens should be true")
	}
	if req.SendAt != 1704067200 {
		t.Errorf("SendMessageRequest.SendAt = %d, want 1704067200", req.SendAt)
	}
}
