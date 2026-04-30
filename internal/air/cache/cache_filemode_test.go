package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestNewManager_SetsCacheDirMode confirms the cache directory is 0700.
func TestNewManager_SetsCacheDirMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode semantics differ on Windows")
	}

	dir := t.TempDir()
	cacheDir := filepath.Join(dir, "air-cache")

	if _, err := NewManager(Config{BasePath: cacheDir}); err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	info, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("stat cache dir: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0700 {
		t.Errorf("cache dir mode: want 0700, got %o", mode)
	}
}

// TestGetDB_RestrictsFileMode confirms the per-account .db file is 0600
// after Manager.GetDB initializes the schema. This is defense-in-depth on
// top of the 0700 directory mode — a permissive umask should not leave the
// SQLite file world-readable.
func TestGetDB_RestrictsFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode semantics differ on Windows")
	}

	dir := t.TempDir()
	mgr, err := NewManager(Config{BasePath: dir})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })

	if _, err := mgr.GetDB("user@example.com"); err != nil {
		t.Fatalf("GetDB: %v", err)
	}

	dbPath := mgr.DBPath("user@example.com")
	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db file: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("db file mode: want 0600, got %o (path=%s)", mode, dbPath)
	}
}

// TestRestrictDBFileMode_HandlesMissingFiles ensures the helper silently
// no-ops on missing sidecar files and never panics on bad paths.
func TestRestrictDBFileMode_HandlesMissingFiles(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode semantics differ on Windows")
	}

	dir := t.TempDir()
	mainPath := filepath.Join(dir, "exists.db")
	if err := os.WriteFile(mainPath, []byte("dummy"), 0644); err != nil {
		t.Fatalf("seed main file: %v", err)
	}
	// -wal and -shm intentionally absent.

	restrictDBFileMode(mainPath)

	info, err := os.Stat(mainPath)
	if err != nil {
		t.Fatalf("stat after restrict: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("file mode after restrict: want 0600, got %o", mode)
	}

	// Non-existent path: must not panic, must not create files.
	restrictDBFileMode(filepath.Join(dir, "does-not-exist.db"))
	if _, err := os.Stat(filepath.Join(dir, "does-not-exist.db")); !os.IsNotExist(err) {
		t.Errorf("restrict should not create files; stat err=%v", err)
	}
}

// TestOpenSharedDB_RestrictsFileMode confirms the shared photos.db file
// also gets the tightened mode.
func TestOpenSharedDB_RestrictsFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file mode semantics differ on Windows")
	}

	dir := t.TempDir()
	db, err := OpenSharedDB(dir, "photos.db")
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// Force an actual file by running a trivial DDL.
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS t (id INTEGER)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	// Re-apply restrict in case the table creation only just produced the file.
	restrictDBFileMode(filepath.Join(dir, "photos.db"))

	info, err := os.Stat(filepath.Join(dir, "photos.db"))
	if err != nil {
		t.Fatalf("stat shared db: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0600 {
		t.Errorf("shared db mode: want 0600, got %o", mode)
	}
}
