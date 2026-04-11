package domain

import (
	"testing"
	"time"
)

// =============================================================================
// ScheduledMessage Tests
// =============================================================================

func TestScheduledMessage_Creation(t *testing.T) {
	tests := []struct {
		name   string
		msg    ScheduledMessage
		status string
	}{
		{
			name: "pending scheduled message",
			msg: ScheduledMessage{
				ScheduleID: "sched-123",
				Status:     "pending",
				CloseTime:  1704067200,
			},
			status: "pending",
		},
		{
			name: "sent scheduled message",
			msg: ScheduledMessage{
				ScheduleID: "sched-456",
				Status:     "sent",
				CloseTime:  1704060000,
			},
			status: "sent",
		},
		{
			name: "cancelled scheduled message",
			msg: ScheduledMessage{
				ScheduleID: "sched-789",
				Status:     "cancelled",
				CloseTime:  1704070000,
			},
			status: "cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.msg.Status != tt.status {
				t.Errorf("ScheduledMessage.Status = %q, want %q", tt.msg.Status, tt.status)
			}
		})
	}
}

// =============================================================================
// TrackingOptions Tests
// =============================================================================

func TestTrackingOptions_Creation(t *testing.T) {
	tests := []struct {
		name string
		opts TrackingOptions
	}{
		{
			name: "full tracking enabled",
			opts: TrackingOptions{
				Opens: true,
				Links: true,
				Label: "newsletter-2024-01",
			},
		},
		{
			name: "opens only",
			opts: TrackingOptions{
				Opens: true,
				Links: false,
			},
		},
		{
			name: "links only",
			opts: TrackingOptions{
				Opens: false,
				Links: true,
			},
		},
		{
			name: "no tracking",
			opts: TrackingOptions{
				Opens: false,
				Links: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the struct can be created
			_ = tt.opts
		})
	}
}

// =============================================================================
// MessageQueryParams Tests
// =============================================================================

func TestMessageQueryParams_Creation(t *testing.T) {
	unread := true
	starred := false
	hasAttachment := true

	params := MessageQueryParams{
		Limit:          100,
		Offset:         50,
		PageToken:      "token-123",
		Subject:        "important",
		From:           "boss@example.com",
		To:             "me@example.com",
		Cc:             "team@example.com",
		In:             []string{"inbox", "important"},
		Unread:         &unread,
		Starred:        &starred,
		ThreadID:       "thread-123",
		ReceivedBefore: 1704153600,
		ReceivedAfter:  1704067200,
		HasAttachment:  &hasAttachment,
		SearchQuery:    "project update",
		Fields:         "include_headers",
		MetadataPair:   "key1:value1",
	}

	if params.Limit != 100 {
		t.Errorf("MessageQueryParams.Limit = %d, want 100", params.Limit)
	}
	if params.Subject != "important" {
		t.Errorf("MessageQueryParams.Subject = %q, want %q", params.Subject, "important")
	}
	if params.Unread == nil || !*params.Unread {
		t.Error("MessageQueryParams.Unread should be true")
	}
	if len(params.In) != 2 {
		t.Errorf("MessageQueryParams.In length = %d, want 2", len(params.In))
	}
}

// =============================================================================
// ThreadQueryParams Tests
// =============================================================================

func TestThreadQueryParams_Creation(t *testing.T) {
	unread := true
	hasAttachment := true

	params := ThreadQueryParams{
		Limit:           50,
		PageToken:       "cursor-abc",
		Subject:         "meeting",
		From:            "sender@example.com",
		To:              "recipient@example.com",
		In:              []string{"inbox"},
		Unread:          &unread,
		LatestMsgBefore: 1704153600,
		LatestMsgAfter:  1704067200,
		HasAttachment:   &hasAttachment,
		SearchQuery:     "quarterly review",
	}

	if params.Limit != 50 {
		t.Errorf("ThreadQueryParams.Limit = %d, want 50", params.Limit)
	}
	if params.Subject != "meeting" {
		t.Errorf("ThreadQueryParams.Subject = %q, want %q", params.Subject, "meeting")
	}
	if params.HasAttachment == nil || !*params.HasAttachment {
		t.Error("ThreadQueryParams.HasAttachment should be true")
	}
}

// =============================================================================
// UpdateMessageRequest Tests
// =============================================================================

func TestUpdateMessageRequest_Creation(t *testing.T) {
	unread := false
	starred := true

	req := UpdateMessageRequest{
		Unread:  &unread,
		Starred: &starred,
		Folders: []string{"archive", "important"},
	}

	if req.Unread == nil || *req.Unread {
		t.Error("UpdateMessageRequest.Unread should be false")
	}
	if req.Starred == nil || !*req.Starred {
		t.Error("UpdateMessageRequest.Starred should be true")
	}
	if len(req.Folders) != 2 {
		t.Errorf("UpdateMessageRequest.Folders length = %d, want 2", len(req.Folders))
	}
}

// =============================================================================
// CreateDraftRequest Tests
// =============================================================================

func TestCreateDraftRequest_Creation(t *testing.T) {
	req := CreateDraftRequest{
		Subject:     "Draft Email",
		Body:        "<p>Draft content</p>",
		SignatureID: "sig-123",
		To: []EmailParticipant{
			{Email: "to@example.com"},
		},
		Cc: []EmailParticipant{
			{Email: "cc@example.com"},
		},
		ReplyToMsgID: "orig-msg-123",
		Attachments: []Attachment{
			{Filename: "draft-attachment.pdf", Size: 500},
		},
		Metadata: map[string]string{
			"draft_type": "follow_up",
		},
	}

	if req.Subject != "Draft Email" {
		t.Errorf("CreateDraftRequest.Subject = %q, want %q", req.Subject, "Draft Email")
	}
	if len(req.To) != 1 {
		t.Errorf("CreateDraftRequest.To length = %d, want 1", len(req.To))
	}
	if req.ReplyToMsgID != "orig-msg-123" {
		t.Errorf("CreateDraftRequest.ReplyToMsgID = %q, want %q", req.ReplyToMsgID, "orig-msg-123")
	}
	if req.SignatureID != "sig-123" {
		t.Errorf("CreateDraftRequest.SignatureID = %q, want %q", req.SignatureID, "sig-123")
	}
}

func TestSignatureRequestTypes(t *testing.T) {
	t.Run("send draft request", func(t *testing.T) {
		req := SendDraftRequest{SignatureID: "sig-send"}
		if req.SignatureID != "sig-send" {
			t.Errorf("SendDraftRequest.SignatureID = %q, want %q", req.SignatureID, "sig-send")
		}
	})

	t.Run("create signature request", func(t *testing.T) {
		req := CreateSignatureRequest{
			Name: "Work",
			Body: "<p>Best regards</p>",
		}
		if req.Name != "Work" {
			t.Errorf("CreateSignatureRequest.Name = %q, want %q", req.Name, "Work")
		}
		if req.Body != "<p>Best regards</p>" {
			t.Errorf("CreateSignatureRequest.Body = %q, want %q", req.Body, "<p>Best regards</p>")
		}
	})

	t.Run("update signature request", func(t *testing.T) {
		name := "Updated"
		body := "<p>Updated</p>"
		req := UpdateSignatureRequest{
			Name: &name,
			Body: &body,
		}
		if req.Name == nil || *req.Name != "Updated" {
			t.Fatalf("UpdateSignatureRequest.Name = %v, want %q", req.Name, "Updated")
		}
		if req.Body == nil || *req.Body != "<p>Updated</p>" {
			t.Fatalf("UpdateSignatureRequest.Body = %v, want %q", req.Body, "<p>Updated</p>")
		}
	})

	t.Run("signature model", func(t *testing.T) {
		now := time.Now()
		signature := Signature{
			ID:        "sig-123",
			Name:      "Work",
			Body:      "<p>Best regards</p>",
			CreatedAt: now,
			UpdatedAt: now,
		}
		if signature.ID != "sig-123" {
			t.Errorf("Signature.ID = %q, want %q", signature.ID, "sig-123")
		}
		if signature.Name != "Work" {
			t.Errorf("Signature.Name = %q, want %q", signature.Name, "Work")
		}
	})
}

// =============================================================================
// CreateFolderRequest Tests
// =============================================================================

func TestCreateFolderRequest_Creation(t *testing.T) {
	req := CreateFolderRequest{
		Name:            "Projects",
		ParentID:        "parent-123",
		BackgroundColor: "#0000ff",
		TextColor:       "#ffffff",
	}

	if req.Name != "Projects" {
		t.Errorf("CreateFolderRequest.Name = %q, want %q", req.Name, "Projects")
	}
	if req.ParentID != "parent-123" {
		t.Errorf("CreateFolderRequest.ParentID = %q, want %q", req.ParentID, "parent-123")
	}
}

// =============================================================================
// Pagination Tests
// =============================================================================

func TestPagination_Creation(t *testing.T) {
	tests := []struct {
		name       string
		pagination Pagination
		hasMore    bool
	}{
		{
			name: "has more pages",
			pagination: Pagination{
				NextCursor: "next-page-cursor",
				HasMore:    true,
			},
			hasMore: true,
		},
		{
			name: "last page",
			pagination: Pagination{
				NextCursor: "",
				HasMore:    false,
			},
			hasMore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pagination.HasMore != tt.hasMore {
				t.Errorf("Pagination.HasMore = %v, want %v", tt.pagination.HasMore, tt.hasMore)
			}
		})
	}
}

// =============================================================================
// SmartComposeRequest Tests
// =============================================================================

func TestSmartComposeRequest_Creation(t *testing.T) {
	req := SmartComposeRequest{
		Prompt: "Write a polite follow-up email to a client about a delayed project",
	}

	if req.Prompt == "" {
		t.Error("SmartComposeRequest.Prompt should not be empty")
	}
}

// =============================================================================
// SmartComposeSuggestion Tests
// =============================================================================

func TestSmartComposeSuggestion_Creation(t *testing.T) {
	suggestion := SmartComposeSuggestion{
		Suggestion: "Dear Client,\n\nI hope this email finds you well...",
	}

	if suggestion.Suggestion == "" {
		t.Error("SmartComposeSuggestion.Suggestion should not be empty")
	}
}

// =============================================================================
// TrackingData Tests
// =============================================================================

func TestTrackingData_Creation(t *testing.T) {
	now := time.Now()
	data := TrackingData{
		MessageID: "msg-123",
		Opens: []OpenEvent{
			{
				OpenedID:  "open-1",
				Timestamp: now.Add(-1 * time.Hour),
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
			},
		},
		Clicks: []ClickEvent{
			{
				ClickID:   "click-1",
				Timestamp: now.Add(-30 * time.Minute),
				URL:       "https://example.com/link",
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
				LinkIndex: 0,
			},
		},
		Replies: []ReplyEvent{
			{
				MessageID:     "reply-msg-1",
				Timestamp:     now,
				ThreadID:      "thread-123",
				RootMessageID: "msg-123",
			},
		},
	}

	if len(data.Opens) != 1 {
		t.Errorf("TrackingData.Opens length = %d, want 1", len(data.Opens))
	}
	if len(data.Clicks) != 1 {
		t.Errorf("TrackingData.Clicks length = %d, want 1", len(data.Clicks))
	}
	if len(data.Replies) != 1 {
		t.Errorf("TrackingData.Replies length = %d, want 1", len(data.Replies))
	}
	if data.Clicks[0].URL != "https://example.com/link" {
		t.Errorf("ClickEvent.URL = %q, want expected URL", data.Clicks[0].URL)
	}
}
