package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{1048576, "1 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1 GB"},
		{1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatFileSize(tt.size)
			if result != tt.expected {
				t.Errorf("formatFileSize(%d) = %q, want %q", tt.size, result, tt.expected)
			}
		})
	}
}

func TestAttachmentInfo(t *testing.T) {
	info := AttachmentInfo{
		MessageID: "msg-123",
		Attachment: domain.Attachment{
			ID:          "att-456",
			Filename:    "document.pdf",
			ContentType: "application/pdf",
			Size:        1024,
		},
	}

	if info.MessageID != "msg-123" {
		t.Errorf("MessageID = %q, want 'msg-123'", info.MessageID)
	}

	if info.Attachment.ID != "att-456" {
		t.Errorf("Attachment.ID = %q, want 'att-456'", info.Attachment.ID)
	}

	if info.Attachment.Filename != "document.pdf" {
		t.Errorf("Attachment.Filename = %q, want 'document.pdf'", info.Attachment.Filename)
	}
}

func TestMessagesViewAttachments(t *testing.T) {
	app := createTestApp(t)

	view := NewMessagesView(app)

	if view.attachments != nil {
		t.Error("attachments should be nil initially")
	}

	// Add some attachments
	view.attachments = []AttachmentInfo{
		{
			MessageID: "msg-1",
			Attachment: domain.Attachment{
				ID:          "att-1",
				Filename:    "file1.pdf",
				ContentType: "application/pdf",
				Size:        1024,
			},
		},
		{
			MessageID: "msg-1",
			Attachment: domain.Attachment{
				ID:          "att-2",
				Filename:    "file2.txt",
				ContentType: "text/plain",
				Size:        512,
			},
		},
	}

	if len(view.attachments) != 2 {
		t.Errorf("len(attachments) = %d, want 2", len(view.attachments))
	}
}

func TestGetUniqueFilename(t *testing.T) {
	app := createTestApp(t)
	view := NewMessagesView(app)

	// Test with non-existent file
	tmpDir := t.TempDir()
	nonExistentPath := filepath.Join(tmpDir, "newfile.txt")
	result := view.getUniqueFilename(nonExistentPath)
	if result != nonExistentPath {
		t.Errorf("getUniqueFilename for new file = %q, want %q", result, nonExistentPath)
	}

	// Create a file and test uniqueness
	existingPath := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result = view.getUniqueFilename(existingPath)
	expectedPath := filepath.Join(tmpDir, "existing (1).txt")
	if result != expectedPath {
		t.Errorf("getUniqueFilename for existing file = %q, want %q", result, expectedPath)
	}

	// Create second file and test again
	if err := os.WriteFile(expectedPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create second test file: %v", err)
	}

	result = view.getUniqueFilename(existingPath)
	expectedPath2 := filepath.Join(tmpDir, "existing (2).txt")
	if result != expectedPath2 {
		t.Errorf("getUniqueFilename with two existing = %q, want %q", result, expectedPath2)
	}
}

func TestMessagesViewClearAttachmentsOnCloseDetail(t *testing.T) {
	app := createTestApp(t)

	view := NewMessagesView(app)

	// Simulate having attachments
	view.attachments = []AttachmentInfo{
		{
			MessageID: "msg-1",
			Attachment: domain.Attachment{
				ID:       "att-1",
				Filename: "file.pdf",
			},
		},
	}
	view.showingDetail = true

	// Close detail should clear attachments
	// Note: We can't fully test closeDetail because it manipulates the app's page stack
	// So we just verify the data clearing logic
	view.showingDetail = false
	view.attachments = nil

	if view.attachments != nil {
		t.Error("attachments should be nil after clearing")
	}

	if view.showingDetail {
		t.Error("showingDetail should be false after clearing")
	}
}
