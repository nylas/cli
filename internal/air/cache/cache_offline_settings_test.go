package cache

import (
	"testing"
	"time"
)

// ================================
// OFFLINE QUEUE TESTS
// ================================

func TestOfflineQueue(t *testing.T) {
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

	queue, err := NewOfflineQueue(db)
	if err != nil {
		t.Fatalf("NewOfflineQueue failed: %v", err)
	}

	// Test Enqueue
	payload := MarkReadPayload{EmailID: "email-123", Unread: false}
	if err := queue.Enqueue(ActionMarkRead, "email-123", payload); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Test Count
	count, err := queue.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Count = %d, want 1", count)
	}

	// Test HasPendingActions
	hasPending, err := queue.HasPendingActions()
	if err != nil {
		t.Fatalf("HasPendingActions failed: %v", err)
	}
	if !hasPending {
		t.Error("Should have pending actions")
	}

	// Test Peek
	peeked, err := queue.Peek()
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if peeked == nil {
		t.Fatal("Peek returned nil")
		return
	}
	if peeked.Type != ActionMarkRead {
		t.Errorf("Peeked Type = %s, want %s", peeked.Type, ActionMarkRead)
	}
	if peeked.ResourceID != "email-123" {
		t.Errorf("Peeked ResourceID = %s, want email-123", peeked.ResourceID)
	}

	// Count should still be 1 after peek
	count, _ = queue.Count()
	if count != 1 {
		t.Error("Peek should not remove item from queue")
	}

	// Test List
	actions, err := queue.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(actions) != 1 {
		t.Errorf("List returned %d actions, want 1", len(actions))
	}

	// Test Dequeue
	dequeued, err := queue.Dequeue()
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	if dequeued == nil {
		t.Fatal("Dequeue returned nil")
		return
	}
	if dequeued.Type != ActionMarkRead {
		t.Errorf("Dequeued Type = %s, want %s", dequeued.Type, ActionMarkRead)
	}

	// Queue should be empty now
	count, _ = queue.Count()
	if count != 0 {
		t.Error("Queue should be empty after dequeue")
	}
}

func TestOfflineQueueMultipleActions(t *testing.T) {
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

	queue, err := NewOfflineQueue(db)
	if err != nil {
		t.Fatalf("NewOfflineQueue failed: %v", err)
	}

	// Enqueue multiple actions
	actions := []struct {
		actionType ActionType
		resourceID string
		payload    any
	}{
		{ActionMarkRead, "email-1", MarkReadPayload{EmailID: "email-1", Unread: false}},
		{ActionStar, "email-2", StarPayload{EmailID: "email-2", Starred: true}},
		{ActionArchive, "email-3", nil},
	}

	for _, a := range actions {
		if err := queue.Enqueue(a.actionType, a.resourceID, a.payload); err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}
	}

	count, _ := queue.Count()
	if count != 3 {
		t.Errorf("Count = %d, want 3", count)
	}

	// Test RemoveByResourceID
	if err := queue.RemoveByResourceID("email-2"); err != nil {
		t.Fatalf("RemoveByResourceID failed: %v", err)
	}

	count, _ = queue.Count()
	if count != 2 {
		t.Errorf("Count after RemoveByResourceID = %d, want 2", count)
	}

	// Test Clear
	if err := queue.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	count, _ = queue.Count()
	if count != 0 {
		t.Error("Queue should be empty after Clear")
	}
}

// ================================
// SETTINGS TESTS
// ================================

func TestSettings(t *testing.T) {
	tmpDir := t.TempDir()

	// Test LoadSettings (creates default if not exists)
	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	// Verify defaults
	if !settings.Enabled {
		t.Error("Default Enabled should be true")
	}
	if settings.MaxSizeMB != 500 {
		t.Errorf("Default MaxSizeMB = %d, want 500", settings.MaxSizeMB)
	}
	if settings.TTLDays != 30 {
		t.Errorf("Default TTLDays = %d, want 30", settings.TTLDays)
	}
	if settings.Theme != "dark" {
		t.Errorf("Default Theme = %s, want 'dark'", settings.Theme)
	}

	// Test Update
	err = settings.Update(func(s *Settings) {
		s.MaxSizeMB = 1000
		s.Theme = "light"
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify update
	if settings.MaxSizeMB != 1000 {
		t.Errorf("Updated MaxSizeMB = %d, want 1000", settings.MaxSizeMB)
	}
	if settings.Theme != "light" {
		t.Errorf("Updated Theme = %s, want 'light'", settings.Theme)
	}

	// Test Get (returns copy)
	copy := settings.Get()
	if copy.MaxSizeMB != 1000 {
		t.Errorf("Get().MaxSizeMB = %d, want 1000", copy.MaxSizeMB)
	}

	// Test SetEnabled
	if err := settings.SetEnabled(false); err != nil {
		t.Fatalf("SetEnabled failed: %v", err)
	}
	if settings.Enabled {
		t.Error("Enabled should be false")
	}

	// Test SetMaxSize
	if err := settings.SetMaxSize(2000); err != nil {
		t.Fatalf("SetMaxSize failed: %v", err)
	}
	if settings.MaxSizeMB != 2000 {
		t.Errorf("MaxSizeMB = %d, want 2000", settings.MaxSizeMB)
	}

	// Test SetMaxSize minimum
	if err := settings.SetMaxSize(10); err != nil {
		t.Fatalf("SetMaxSize (min) failed: %v", err)
	}
	if settings.MaxSizeMB != 50 {
		t.Errorf("MaxSizeMB should be clamped to minimum 50, got %d", settings.MaxSizeMB)
	}

	// Test SetTheme
	if err := settings.SetTheme("system"); err != nil {
		t.Fatalf("SetTheme failed: %v", err)
	}
	if settings.Theme != "system" {
		t.Errorf("Theme = %s, want 'system'", settings.Theme)
	}

	// Test SetTheme invalid (should default to dark)
	if err := settings.SetTheme("invalid"); err != nil {
		t.Fatalf("SetTheme (invalid) failed: %v", err)
	}
	if settings.Theme != "dark" {
		t.Errorf("Invalid theme should default to 'dark', got %s", settings.Theme)
	}

	// Test Reset
	if err := settings.Reset(); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}
	if settings.MaxSizeMB != 500 {
		t.Errorf("Reset MaxSizeMB = %d, want 500", settings.MaxSizeMB)
	}
	if settings.Theme != "dark" {
		t.Errorf("Reset Theme = %s, want 'dark'", settings.Theme)
	}
}

func TestSettingsValidate(t *testing.T) {
	tmpDir := t.TempDir()
	settings, _ := LoadSettings(tmpDir)

	// Valid settings
	if err := settings.Validate(); err != nil {
		t.Errorf("Valid settings should not return error: %v", err)
	}

	// Test invalid MaxSizeMB
	settings.MaxSizeMB = 10
	if err := settings.Validate(); err == nil {
		t.Error("MaxSizeMB < 50 should fail validation")
	}
	settings.MaxSizeMB = 500

	// Test invalid TTLDays
	settings.TTLDays = 0
	if err := settings.Validate(); err == nil {
		t.Error("TTLDays < 1 should fail validation")
	}
	settings.TTLDays = 30

	// Test invalid SyncIntervalMinutes
	settings.SyncIntervalMinutes = 0
	if err := settings.Validate(); err == nil {
		t.Error("SyncIntervalMinutes < 1 should fail validation")
	}
}

func TestSettingsHelpers(t *testing.T) {
	tmpDir := t.TempDir()
	settings, _ := LoadSettings(tmpDir)

	// Test GetSyncInterval
	settings.SyncIntervalMinutes = 5
	interval := settings.GetSyncInterval()
	if interval != 5*time.Minute {
		t.Errorf("GetSyncInterval = %v, want 5m", interval)
	}

	// Test GetTTL
	settings.TTLDays = 30
	ttl := settings.GetTTL()
	if ttl != 30*24*time.Hour {
		t.Errorf("GetTTL = %v, want 720h", ttl)
	}

	// Test GetMaxSizeBytes
	settings.MaxSizeMB = 500
	maxBytes := settings.GetMaxSizeBytes()
	if maxBytes != 500*1024*1024 {
		t.Errorf("GetMaxSizeBytes = %d, want %d", maxBytes, 500*1024*1024)
	}

	// Test IsEncryptionEnabled
	settings.EncryptionEnabled = true
	if !settings.IsEncryptionEnabled() {
		t.Error("IsEncryptionEnabled should be true")
	}

	// Test IsCacheEnabled
	settings.Enabled = true
	if !settings.IsCacheEnabled() {
		t.Error("IsCacheEnabled should be true")
	}
}

// ================================
// UNIFIED SEARCH TESTS
// ================================

func TestUnifiedSearch(t *testing.T) {
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

	// Add test data
	emailStore := NewEmailStore(db)
	emails := []*CachedEmail{
		{ID: "e1", Subject: "Meeting notes", FromName: "Alice", FromEmail: "alice@test.com", Date: time.Now()},
		{ID: "e2", Subject: "Project update", FromName: "Bob", FromEmail: "bob@test.com", Date: time.Now()},
	}
	if err := emailStore.PutBatch(emails); err != nil {
		t.Fatalf("Put emails failed: %v", err)
	}

	eventStore := NewEventStore(db)
	now := time.Now()
	events := []*CachedEvent{
		{ID: "ev1", Title: "Team Meeting", StartTime: now, EndTime: now.Add(time.Hour)},
		{ID: "ev2", Title: "Project Review", StartTime: now.Add(2 * time.Hour), EndTime: now.Add(3 * time.Hour)},
	}
	if err := eventStore.PutBatch(events); err != nil {
		t.Fatalf("Put events failed: %v", err)
	}

	contactStore := NewContactStore(db)
	contacts := []*CachedContact{
		{ID: "c1", DisplayName: "Meeting Coordinator", Email: "coord@test.com"},
		{ID: "c2", DisplayName: "Project Manager", Email: "pm@test.com"},
	}
	if err := contactStore.PutBatch(contacts); err != nil {
		t.Fatalf("Put contacts failed: %v", err)
	}

	// Search for "Meeting"
	results, err := UnifiedSearch(db, "Meeting", 20)
	if err != nil {
		t.Fatalf("UnifiedSearch failed: %v", err)
	}

	// Should find: 1 email + 1 event + 1 contact = 3 results
	if len(results) != 3 {
		t.Errorf("UnifiedSearch returned %d results, want 3", len(results))
	}

	// Verify result types
	types := make(map[string]int)
	for _, r := range results {
		types[r.Type]++
	}

	if types["email"] != 1 {
		t.Errorf("Expected 1 email result, got %d", types["email"])
	}
	if types["event"] != 1 {
		t.Errorf("Expected 1 event result, got %d", types["event"])
	}
	if types["contact"] != 1 {
		t.Errorf("Expected 1 contact result, got %d", types["contact"])
	}

	// Search for "Project"
	results, err = UnifiedSearch(db, "Project", 20)
	if err != nil {
		t.Fatalf("UnifiedSearch (Project) failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("UnifiedSearch (Project) returned %d results, want 3", len(results))
	}
}

// ================================
// MANAGER STATS TESTS
// ================================

func TestManagerGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := NewManager(Config{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	email := "test@example.com"
	db, err := mgr.GetDB(email)
	if err != nil {
		t.Fatalf("GetDB failed: %v", err)
	}

	// Add some test data
	emailStore := NewEmailStore(db)
	emails := []*CachedEmail{
		{ID: "1", Subject: "Email 1", Date: time.Now()},
		{ID: "2", Subject: "Email 2", Date: time.Now()},
	}
	if err := emailStore.PutBatch(emails); err != nil {
		t.Fatalf("Put emails failed: %v", err)
	}

	eventStore := NewEventStore(db)
	now := time.Now()
	events := []*CachedEvent{
		{ID: "1", Title: "Event 1", StartTime: now, EndTime: now.Add(time.Hour)},
	}
	if err := eventStore.PutBatch(events); err != nil {
		t.Fatalf("Put events failed: %v", err)
	}

	// Get stats
	stats, err := mgr.GetStats(email)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.EmailCount != 2 {
		t.Errorf("EmailCount = %d, want 2", stats.EmailCount)
	}
	if stats.EventCount != 1 {
		t.Errorf("EventCount = %d, want 1", stats.EventCount)
	}
	if stats.SizeBytes == 0 {
		t.Error("SizeBytes should be > 0")
	}
}
