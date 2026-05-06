//go:build integration

package tui

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestIntegrationSafeAttachmentDownloadPathStaysInDestination(t *testing.T) {
	downloadDir := t.TempDir()

	result, err := safeAttachmentDownloadPath(downloadDir, "../../../Library/LaunchAgents/com.example.plist")
	if err != nil {
		t.Fatalf("safeAttachmentDownloadPath returned error: %v", err)
	}

	if filepath.Base(result) != "com.example.plist" {
		t.Fatalf("download path base = %q, want %q", filepath.Base(result), "com.example.plist")
	}

	rel, err := filepath.Rel(downloadDir, result)
	if err != nil {
		t.Fatalf("filepath.Rel returned error: %v", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		t.Fatalf("download path escaped destination: %q", result)
	}
}
