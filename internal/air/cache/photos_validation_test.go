package cache

import (
	"strings"
	"testing"
)

// TestValidateContactID_Rejects covers IDs that must NOT be allowed to flow
// into filepath.Join(basePath, contactID). A photo store that accepts these
// would let a hostile API response punch out of the cache directory or
// corrupt unrelated files.
func TestValidateContactID_Rejects(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"dot", "."},
		{"dotdot", ".."},
		{"slash traversal", "../etc/passwd"},
		{"backslash traversal", "..\\windows\\system32"},
		{"forward slash", "abc/def"},
		{"backslash", "abc\\def"},
		{"null byte", "abc\x00def"},
		{"control byte", "abc\x01def"},
		{"DEL byte", "abc\x7fdef"},
		{"contains dotdot", "foo..bar"},
		{"long ID", strings.Repeat("a", 129)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := validateContactID(tc.id); err == nil {
				t.Errorf("expected error for %q, got nil", tc.id)
			}
		})
	}
}

func TestValidateContactID_Accepts(t *testing.T) {
	t.Parallel()

	cases := []string{
		"abc123",
		"contact-uuid-123",
		"USER_42",
		"a",
		strings.Repeat("a", 128), // boundary
	}
	for _, id := range cases {
		if err := validateContactID(id); err != nil {
			t.Errorf("expected %q to validate, got %v", id, err)
		}
	}
}

func TestPhotoStore_Put_RejectsTraversal(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	tmpDir := t.TempDir()
	store, err := NewPhotoStore(db, tmpDir, DefaultPhotoTTL)
	if err != nil {
		t.Fatalf("NewPhotoStore: %v", err)
	}

	if err := store.Put("../escape", "image/png", []byte{0x89, 0x50, 0x4E, 0x47}); err == nil {
		t.Fatal("Put should reject path-traversal contactID")
	}
}

func TestPhotoStore_Get_RejectsTraversal(t *testing.T) {
	t.Parallel()

	db := setupTestDB(t)
	tmpDir := t.TempDir()
	store, err := NewPhotoStore(db, tmpDir, DefaultPhotoTTL)
	if err != nil {
		t.Fatalf("NewPhotoStore: %v", err)
	}
	if _, _, err := store.Get("..\\escape"); err == nil {
		t.Fatal("Get should reject path-traversal contactID")
	}
}
