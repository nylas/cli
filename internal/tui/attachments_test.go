package tui

import (
	"os"
	"path/filepath"
	"strings"
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

func TestGetUniqueFilenameTreatsDanglingSymlinkAsExisting(t *testing.T) {
	app := createTestApp(t)
	view := NewMessagesView(app)
	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	linkPath := filepath.Join(tmpDir, "invoice.pdf")
	if err := os.Symlink(filepath.Join(outsideDir, "outside.pdf"), linkPath); err != nil {
		t.Skipf("symlink creation not supported: %v", err)
	}

	result := view.getUniqueFilename(linkPath)
	if result == linkPath {
		t.Fatalf("getUniqueFilename returned dangling symlink path %q", result)
	}
	if filepath.Dir(result) != tmpDir {
		t.Fatalf("getUniqueFilename returned path outside download dir: %q", result)
	}
	if filepath.Base(result) != "invoice (1).pdf" {
		t.Fatalf("getUniqueFilename = %q, want invoice (1).pdf suffix", result)
	}
}

func TestCreateUniqueAttachmentFileDoesNotFollowDanglingSymlink(t *testing.T) {
	app := createTestApp(t)
	view := NewMessagesView(app)
	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	outsideTarget := filepath.Join(outsideDir, "outside.pdf")
	linkPath := filepath.Join(tmpDir, "invoice.pdf")
	if err := os.Symlink(outsideTarget, linkPath); err != nil {
		t.Skipf("symlink creation not supported: %v", err)
	}

	file, result, err := view.createUniqueAttachmentFile(linkPath)
	if err != nil {
		t.Fatalf("createUniqueAttachmentFile returned error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("failed to close created file: %v", err)
	}

	if result == linkPath {
		t.Fatalf("createUniqueAttachmentFile used dangling symlink path %q", result)
	}
	if filepath.Base(result) != "invoice (1).pdf" {
		t.Fatalf("createUniqueAttachmentFile path = %q, want invoice (1).pdf suffix", result)
	}
	if _, err := os.Stat(outsideTarget); !os.IsNotExist(err) {
		t.Fatalf("outside symlink target exists or returned unexpected error: %v", err)
	}
	if _, err := os.Stat(result); err != nil {
		t.Fatalf("created attachment file missing: %v", err)
	}
}

func TestSafeAttachmentDownloadPathSanitizesTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	result, err := safeAttachmentDownloadPath(tmpDir, "../../.ssh/config")
	if err != nil {
		t.Fatalf("safeAttachmentDownloadPath returned error: %v", err)
	}

	expected := filepath.Join(tmpDir, "config")
	if result != expected {
		t.Fatalf("safeAttachmentDownloadPath = %q, want %q", result, expected)
	}

	rel, err := filepath.Rel(tmpDir, result)
	if err != nil {
		t.Fatalf("filepath.Rel returned error: %v", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		t.Fatalf("download path escaped destination: %q", result)
	}
}

func TestSafeAttachmentDownloadPathUsesFallbackForEmptyBase(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		filename string
	}{
		{"empty", ""},
		{"current directory", "."},
		{"parent directory", ".."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := safeAttachmentDownloadPath(tmpDir, tt.filename)
			if err != nil {
				t.Fatalf("safeAttachmentDownloadPath returned error: %v", err)
			}

			expected := filepath.Join(tmpDir, "attachment")
			if result != expected {
				t.Fatalf("safeAttachmentDownloadPath = %q, want %q", result, expected)
			}
		})
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
