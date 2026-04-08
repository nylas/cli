package cache

import (
	"testing"
	"time"
)

// ================================
// ENCRYPTION HELPER TESTS
// ================================

func TestEmailStoreList(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmailStore(db)

	// Add test emails
	now := time.Now()
	emails := []*CachedEmail{
		{ID: "1", FolderID: "inbox", Subject: "Email 1", Unread: true, Starred: false, Date: now.Add(-3 * time.Hour)},
		{ID: "2", FolderID: "inbox", Subject: "Email 2", Unread: false, Starred: true, Date: now.Add(-2 * time.Hour)},
		{ID: "3", FolderID: "sent", Subject: "Email 3", Unread: false, Starred: false, Date: now.Add(-1 * time.Hour)},
		{ID: "4", ThreadID: "thread-1", FolderID: "inbox", Subject: "Email 4", Unread: true, Starred: false, Date: now},
	}

	if err := store.PutBatch(emails); err != nil {
		t.Fatalf("PutBatch failed: %v", err)
	}

	// Test List with FolderID filter
	list, err := store.List(ListOptions{FolderID: "inbox", Limit: 10})
	if err != nil {
		t.Fatalf("List (folder) failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("List (folder) returned %d, want 3", len(list))
	}

	// Test List with UnreadOnly filter
	list, err = store.List(ListOptions{UnreadOnly: true, Limit: 10})
	if err != nil {
		t.Fatalf("List (unread) failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List (unread) returned %d, want 2", len(list))
	}

	// Test List with StarredOnly filter
	list, err = store.List(ListOptions{StarredOnly: true, Limit: 10})
	if err != nil {
		t.Fatalf("List (starred) failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List (starred) returned %d, want 1", len(list))
	}

	// Test List with ThreadID filter
	list, err = store.List(ListOptions{ThreadID: "thread-1", Limit: 10})
	if err != nil {
		t.Fatalf("List (thread) failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List (thread) returned %d, want 1", len(list))
	}

	// Test List with Since/Before filters
	// Since 2.5 hours ago should include emails at -2h, -1h, and now (3 total)
	list, err = store.List(ListOptions{Since: now.Add(-2*time.Hour - 30*time.Minute), Limit: 10})
	if err != nil {
		t.Fatalf("List (since) failed: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("List (since) returned %d, want 3", len(list))
	}

	// Test List with Offset
	list, err = store.List(ListOptions{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("List (offset) failed: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List (offset) returned %d, want 2", len(list))
	}
}

func TestEmailStoreCountUnread(t *testing.T) {
	db := setupTestDB(t)
	store := NewEmailStore(db)

	// Add emails with different read status
	emails := []*CachedEmail{
		{ID: "1", Subject: "Unread 1", Unread: true, Date: time.Now()},
		{ID: "2", Subject: "Unread 2", Unread: true, Date: time.Now()},
		{ID: "3", Subject: "Read 1", Unread: false, Date: time.Now()},
	}

	if err := store.PutBatch(emails); err != nil {
		t.Fatalf("PutBatch failed: %v", err)
	}

	count, err := store.CountUnread()
	if err != nil {
		t.Fatalf("CountUnread failed: %v", err)
	}
	if count != 2 {
		t.Errorf("CountUnread = %d, want 2", count)
	}
}

// ================================
// ADDITIONAL CONTACT STORE TESTS
// ================================

func TestContactStoreGetByEmail(t *testing.T) {
	db := setupTestDB(t)
	store := NewContactStore(db)

	// Add test contacts
	contacts := []*CachedContact{
		{ID: "1", DisplayName: "Alice", Email: "alice@example.com"},
		{ID: "2", DisplayName: "Bob", Email: "bob@example.com"},
	}

	if err := store.PutBatch(contacts); err != nil {
		t.Fatalf("PutBatch failed: %v", err)
	}

	// Test GetByEmail
	contact, err := store.GetByEmail("alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail failed: %v", err)
	}
	if contact == nil {
		t.Fatal("GetByEmail returned nil")
		return
	}
	if contact.DisplayName != "Alice" {
		t.Errorf("DisplayName = %s, want Alice", contact.DisplayName)
	}

	// Test GetByEmail - not found (returns sql.ErrNoRows)
	notFound, err := store.GetByEmail("nonexistent@example.com")
	if err == nil {
		t.Error("GetByEmail should return error for non-existent email")
	}
	if notFound != nil {
		t.Error("GetByEmail should return nil for non-existent email")
	}
}

func TestContactStoreListGroups(t *testing.T) {
	db := setupTestDB(t)
	store := NewContactStore(db)

	// Add contacts with groups
	contacts := []*CachedContact{
		{ID: "1", DisplayName: "Alice", Email: "alice@example.com", Groups: []string{"Work", "Friends"}},
		{ID: "2", DisplayName: "Bob", Email: "bob@example.com", Groups: []string{"Work"}},
		{ID: "3", DisplayName: "Charlie", Email: "charlie@example.com", Groups: []string{"Family"}},
		{ID: "4", DisplayName: "Dave", Email: "dave@example.com", Groups: []string{}},
	}

	if err := store.PutBatch(contacts); err != nil {
		t.Fatalf("PutBatch failed: %v", err)
	}

	// Test ListGroups
	groups, err := store.ListGroups()
	if err != nil {
		t.Fatalf("ListGroups failed: %v", err)
	}

	if len(groups) != 3 {
		t.Errorf("ListGroups returned %d groups, want 3", len(groups))
	}

	// Verify groups
	groupSet := make(map[string]bool)
	for _, g := range groups {
		groupSet[g] = true
	}
	if !groupSet["Work"] {
		t.Error("Work group not found")
	}
	if !groupSet["Friends"] {
		t.Error("Friends group not found")
	}
	if !groupSet["Family"] {
		t.Error("Family group not found")
	}
}

// ================================
// ADDITIONAL EVENT STORE TESTS
// ================================
