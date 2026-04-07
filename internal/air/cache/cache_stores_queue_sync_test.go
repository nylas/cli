package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ================================
// ENCRYPTION HELPER TESTS
// ================================

func TestOfflineQueueMarkFailed(t *testing.T) {
	db := setupTestDB(t)
	queue, err := NewOfflineQueue(db)
	if err != nil {
		t.Fatalf("NewOfflineQueue failed: %v", err)
	}

	// Enqueue an action
	payload := MarkReadPayload{EmailID: "email-1", Unread: false}
	if err := queue.Enqueue(ActionMarkRead, "email-1", payload); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Peek to get the ID
	action, _ := queue.Peek()

	// Mark as failed
	testErr := fmt.Errorf("test error")
	if err := queue.MarkFailed(action.ID, testErr); err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}

	// Verify attempts incremented and error recorded
	actions, _ := queue.List()
	if len(actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(actions))
	}
	if actions[0].Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", actions[0].Attempts)
	}
	if actions[0].LastError != "test error" {
		t.Errorf("LastError = %s, want 'test error'", actions[0].LastError)
	}
}

func TestOfflineQueueRemove(t *testing.T) {
	db := setupTestDB(t)
	queue, err := NewOfflineQueue(db)
	if err != nil {
		t.Fatalf("NewOfflineQueue failed: %v", err)
	}

	// Enqueue actions
	_ = queue.Enqueue(ActionMarkRead, "email-1", nil)
	_ = queue.Enqueue(ActionStar, "email-2", nil)

	// Get first action ID
	action, _ := queue.Peek()

	// Remove it
	if err := queue.Remove(action.ID); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify count
	count, _ := queue.Count()
	if count != 1 {
		t.Errorf("Count after Remove = %d, want 1", count)
	}
}

func TestOfflineQueueRemoveStale(t *testing.T) {
	db := setupTestDB(t)
	queue, err := NewOfflineQueue(db)
	if err != nil {
		t.Fatalf("NewOfflineQueue failed: %v", err)
	}

	// Enqueue an action
	_ = queue.Enqueue(ActionMarkRead, "email-1", nil)

	// Remove stale with short max age (should remove nothing since just created)
	removed, err := queue.RemoveStale(time.Hour)
	if err != nil {
		t.Fatalf("RemoveStale failed: %v", err)
	}
	if removed != 0 {
		t.Errorf("RemoveStale removed %d, want 0", removed)
	}

	// Manually insert an old action
	oldTime := time.Now().Add(-24 * time.Hour).Unix()
	_, err = db.Exec(`INSERT INTO offline_queue (type, resource_id, payload, created_at) VALUES (?, ?, ?, ?)`,
		ActionArchive, "old-email", "{}", oldTime)
	if err != nil {
		t.Fatalf("insert old action failed: %v", err)
	}

	// Remove stale with 1 hour max age
	removed, err = queue.RemoveStale(time.Hour)
	if err != nil {
		t.Fatalf("RemoveStale failed: %v", err)
	}
	if removed != 1 {
		t.Errorf("RemoveStale removed %d, want 1", removed)
	}
}

func TestOfflineQueueGetActionData(t *testing.T) {
	db := setupTestDB(t)
	queue, err := NewOfflineQueue(db)
	if err != nil {
		t.Fatalf("NewOfflineQueue failed: %v", err)
	}

	// Enqueue with payload
	payload := MarkReadPayload{EmailID: "email-123", Unread: true}
	_ = queue.Enqueue(ActionMarkRead, "email-123", payload)

	// Get the action
	action, _ := queue.Peek()

	// Parse payload
	var retrieved MarkReadPayload
	if err := action.GetActionData(&retrieved); err != nil {
		t.Fatalf("GetActionData failed: %v", err)
	}

	if retrieved.EmailID != "email-123" {
		t.Errorf("EmailID = %s, want email-123", retrieved.EmailID)
	}
	if !retrieved.Unread {
		t.Error("Unread should be true")
	}
}

// ================================
// ADDITIONAL SYNC STORE TESTS
// ================================

func TestSyncStoreUpdateCursor(t *testing.T) {
	db := setupTestDB(t)
	store := NewSyncStore(db)

	// Set initial state
	state := &SyncState{
		Resource: ResourceEmails,
		LastSync: time.Now().Add(-time.Hour),
		Cursor:   "cursor-1",
	}
	if err := store.Set(state); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Update cursor
	if err := store.UpdateCursor(ResourceEmails, "cursor-2"); err != nil {
		t.Fatalf("UpdateCursor failed: %v", err)
	}

	// Verify
	retrieved, _ := store.Get(ResourceEmails)
	if retrieved.Cursor != "cursor-2" {
		t.Errorf("Cursor = %s, want cursor-2", retrieved.Cursor)
	}
}

func TestSyncStoreMarkSynced(t *testing.T) {
	db := setupTestDB(t)
	store := NewSyncStore(db)

	// MarkSynced for new resource
	if err := store.MarkSynced(ResourceContacts); err != nil {
		t.Fatalf("MarkSynced failed: %v", err)
	}

	// Verify state exists
	state, err := store.Get(ResourceContacts)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if state == nil {
		t.Fatal("State should exist")
		return
	}
	if time.Since(state.LastSync) > time.Second {
		t.Error("LastSync should be recent")
	}

	// MarkSynced again (update existing) - uses Unix second precision
	if err := store.MarkSynced(ResourceContacts); err != nil {
		t.Fatalf("MarkSynced (2nd) failed: %v", err)
	}

	state2, _ := store.Get(ResourceContacts)
	// Verify state2 exists and LastSync is within last second
	if state2 == nil {
		t.Fatal("State should still exist after second MarkSynced")
		return
	}
	if time.Since(state2.LastSync) > time.Second {
		t.Error("LastSync should be recent after update")
	}
}

func TestSyncStoreDelete(t *testing.T) {
	db := setupTestDB(t)
	store := NewSyncStore(db)

	// Set state
	state := &SyncState{
		Resource: ResourceEvents,
		LastSync: time.Now(),
		Cursor:   "cursor-1",
	}
	_ = store.Set(state)

	// Delete
	if err := store.Delete(ResourceEvents); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	retrieved, err := store.Get(ResourceEvents)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved != nil {
		t.Error("State should be deleted")
	}
}

// ================================
// ADDITIONAL ATTACHMENT STORE TESTS
// ================================

func TestAttachmentStoreRemoveOrphaned(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(Config{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	db, err := mgr.GetDB("test@example.com")
	if err != nil {
		t.Fatalf("GetDB failed: %v", err)
	}

	store, err := NewAttachmentStore(db, tmpDir, 100)
	if err != nil {
		t.Fatalf("NewAttachmentStore failed: %v", err)
	}

	// Add a tracked attachment
	att := &CachedAttachment{ID: "att-tracked", EmailID: "email-1", Filename: "tracked.txt"}
	_ = store.Put(att, strings.NewReader("tracked content"))

	// Create orphan file directly in attachments directory
	attachDir := filepath.Join(tmpDir, "attachments")
	orphanPath := filepath.Join(attachDir, "orphan-file.txt")
	_ = os.WriteFile(orphanPath, []byte("orphan content"), 0600)

	// Remove orphaned
	removed, err := store.RemoveOrphaned()
	if err != nil {
		t.Fatalf("RemoveOrphaned failed: %v", err)
	}

	if removed != 1 {
		t.Errorf("RemoveOrphaned removed %d, want 1", removed)
	}

	// Verify orphan file is deleted
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Error("Orphan file should be deleted")
	}

	// Verify tracked attachment still exists
	retrieved, err := store.Get("att-tracked")
	if err != nil {
		t.Fatalf("Get tracked failed: %v", err)
	}
	if retrieved == nil {
		t.Error("Tracked attachment should still exist")
	}
}

// ================================
// ADDITIONAL SETTINGS TESTS
// ================================
