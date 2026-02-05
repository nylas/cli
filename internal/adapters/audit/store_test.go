package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestNewFileStore(t *testing.T) {
	tmpDir := t.TempDir()

	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	if store.Path() != tmpDir {
		t.Errorf("Path() = %q, want %q", store.Path(), tmpDir)
	}
}

func TestFileStore_SaveAndGetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	cfg := &domain.AuditConfig{
		Enabled:       true,
		Initialized:   true,
		Path:          tmpDir,
		RetentionDays: 30,
		MaxSizeMB:     50,
		Format:        "jsonl",
		LogRequestID:  true,
		LogAPIDetails: true,
		RotateDaily:   true,
	}

	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify config file was created
	configPath := filepath.Join(tmpDir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config.json was not created")
	}

	// Load config and verify
	loaded, err := store.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	if !loaded.Enabled {
		t.Error("Enabled should be true")
	}
	if loaded.RetentionDays != 30 {
		t.Errorf("RetentionDays = %d, want 30", loaded.RetentionDays)
	}
	if loaded.MaxSizeMB != 50 {
		t.Errorf("MaxSizeMB = %d, want 50", loaded.MaxSizeMB)
	}
}

func TestFileStore_Log(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	// Enable logging
	cfg := &domain.AuditConfig{
		Enabled:     true,
		Initialized: true,
		Path:        tmpDir,
		RotateDaily: true,
	}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Log an entry
	entry := &domain.AuditEntry{
		Timestamp:  time.Now(),
		Command:    "email list",
		Args:       []string{"--limit", "10"},
		GrantID:    "grant_123",
		GrantEmail: "test@example.com",
		Status:     domain.AuditStatusSuccess,
		Duration:   time.Second,
		RequestID:  "req_abc123",
	}

	if err := store.Log(entry); err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify entry was assigned an ID
	if entry.ID == "" {
		t.Error("Entry ID should be assigned")
	}

	// Verify log file was created
	logFile := filepath.Join(tmpDir, time.Now().Format("2006-01-02")+".jsonl")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestFileStore_LogDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	// Config with logging disabled
	cfg := &domain.AuditConfig{
		Enabled:     false,
		Initialized: true,
		Path:        tmpDir,
	}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Log should succeed but not write anything
	entry := &domain.AuditEntry{
		Command: "email list",
		Status:  domain.AuditStatusSuccess,
	}

	if err := store.Log(entry); err != nil {
		t.Fatalf("Log should not error when disabled: %v", err)
	}

	// Verify no log file was created
	files, _ := filepath.Glob(filepath.Join(tmpDir, "*.jsonl"))
	if len(files) > 0 {
		t.Error("No log files should be created when disabled")
	}
}

func TestFileStore_List(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	cfg := &domain.AuditConfig{
		Enabled:     true,
		Initialized: true,
		Path:        tmpDir,
		RotateDaily: true,
	}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Log multiple entries
	now := time.Now()
	for i := 0; i < 5; i++ {
		entry := &domain.AuditEntry{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Command:   "test command",
			Status:    domain.AuditStatusSuccess,
		}
		if err := store.Log(entry); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	ctx := context.Background()
	entries, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(entries) != 5 {
		t.Errorf("List returned %d entries, want 5", len(entries))
	}

	// Verify entries are sorted newest first
	for i := 1; i < len(entries); i++ {
		if entries[i].Timestamp.After(entries[i-1].Timestamp) {
			t.Error("Entries should be sorted newest first")
		}
	}
}

func TestFileStore_Query(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	cfg := &domain.AuditConfig{
		Enabled:     true,
		Initialized: true,
		Path:        tmpDir,
		RotateDaily: true,
	}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	now := time.Now()

	// Log entries with different commands and statuses
	testCases := []struct {
		command   string
		status    domain.AuditStatus
		requestID string
	}{
		{"email list", domain.AuditStatusSuccess, "req_1"},
		{"email send", domain.AuditStatusSuccess, "req_2"},
		{"email list", domain.AuditStatusError, "req_3"},
		{"calendar events", domain.AuditStatusSuccess, "req_4"},
	}

	for i, tc := range testCases {
		entry := &domain.AuditEntry{
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Command:   tc.command,
			Status:    tc.status,
			RequestID: tc.requestID,
		}
		if err := store.Log(entry); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	ctx := context.Background()

	t.Run("FilterByCommand", func(t *testing.T) {
		entries, err := store.Query(ctx, &domain.AuditQueryOptions{
			Command: "email",
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(entries) != 3 {
			t.Errorf("Expected 3 email entries, got %d", len(entries))
		}
	})

	t.Run("FilterByStatus", func(t *testing.T) {
		entries, err := store.Query(ctx, &domain.AuditQueryOptions{
			Status: "error",
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("Expected 1 error entry, got %d", len(entries))
		}
	})

	t.Run("FilterByRequestID", func(t *testing.T) {
		entries, err := store.Query(ctx, &domain.AuditQueryOptions{
			RequestID: "req_2",
			Limit:     10,
		})
		if err != nil {
			t.Fatalf("Query failed: %v", err)
		}
		if len(entries) != 1 {
			t.Errorf("Expected 1 entry, got %d", len(entries))
		}
		if entries[0].Command != "email send" {
			t.Errorf("Expected 'email send', got %q", entries[0].Command)
		}
	})
}

func TestFileStore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	cfg := &domain.AuditConfig{
		Enabled:     true,
		Initialized: true,
		Path:        tmpDir,
		RotateDaily: true,
	}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Log some entries
	for i := 0; i < 3; i++ {
		entry := &domain.AuditEntry{
			Command: "test",
			Status:  domain.AuditStatusSuccess,
		}
		if err := store.Log(entry); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	ctx := context.Background()

	// Clear logs
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify logs are cleared
	entries, err := store.List(ctx, 10)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}

	// Verify config still exists
	_, err = store.GetConfig()
	if err != nil {
		t.Error("Config should still exist after clear")
	}
}

func TestFileStore_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	cfg := &domain.AuditConfig{
		Enabled:     true,
		Initialized: true,
		Path:        tmpDir,
		RotateDaily: true,
	}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Log some entries
	for i := 0; i < 5; i++ {
		entry := &domain.AuditEntry{
			Command: "test command",
			Status:  domain.AuditStatusSuccess,
		}
		if err := store.Log(entry); err != nil {
			t.Fatalf("Log failed: %v", err)
		}
	}

	fileCount, totalSize, oldest, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats failed: %v", err)
	}

	if fileCount != 1 {
		t.Errorf("FileCount = %d, want 1", fileCount)
	}
	if totalSize == 0 {
		t.Error("TotalSize should be > 0")
	}
	if oldest == nil {
		t.Error("Oldest entry should not be nil")
	}
}

func TestFileStore_Cleanup(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileStore(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStore failed: %v", err)
	}

	cfg := &domain.AuditConfig{
		Enabled:       true,
		Initialized:   true,
		Path:          tmpDir,
		RetentionDays: 7,
		RotateDaily:   true,
	}
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Create old log files manually
	oldDate := time.Now().AddDate(0, 0, -10)
	oldFile := filepath.Join(tmpDir, oldDate.Format("2006-01-02")+".jsonl")
	if err := os.WriteFile(oldFile, []byte(`{"command":"old"}`+"\n"), 0600); err != nil {
		t.Fatalf("Failed to create old file: %v", err)
	}

	// Create recent log file
	recentDate := time.Now().AddDate(0, 0, -3)
	recentFile := filepath.Join(tmpDir, recentDate.Format("2006-01-02")+".jsonl")
	if err := os.WriteFile(recentFile, []byte(`{"command":"recent"}`+"\n"), 0600); err != nil {
		t.Fatalf("Failed to create recent file: %v", err)
	}

	ctx := context.Background()

	// Run cleanup
	if err := store.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup failed: %v", err)
	}

	// Verify old file was removed
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("Old file should have been removed")
	}

	// Verify recent file still exists
	if _, err := os.Stat(recentFile); os.IsNotExist(err) {
		t.Error("Recent file should still exist")
	}
}
