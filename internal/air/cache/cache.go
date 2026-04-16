// Package cache provides local SQLite caching for Nylas Air.
// Each email account has its own database file at ~/.config/nylas/air/{email}.db
package cache

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// Manager handles per-email cache databases.
type Manager struct {
	basePath string
	dbs      map[string]*sql.DB
	mu       sync.RWMutex
}

// Config holds cache configuration options.
type Config struct {
	// BasePath is the directory for cache files. Defaults to ~/.config/nylas/air/
	BasePath string
	// MaxSizeMB is the maximum cache size in MB (default: 500)
	MaxSizeMB int
	// TTLDays is how long to keep cached items (default: 30)
	TTLDays int
	// SyncIntervalMinutes is background sync frequency (default: 5)
	SyncIntervalMinutes int
}

// DefaultConfig returns default cache configuration.
func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	return Config{
		BasePath:            filepath.Join(homeDir, ".config", "nylas", "air"),
		MaxSizeMB:           500,
		TTLDays:             30,
		SyncIntervalMinutes: 5,
	}
}

// NewManager creates a new cache manager.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.BasePath == "" {
		cfg = DefaultConfig()
	}

	// Ensure base directory exists
	if err := os.MkdirAll(cfg.BasePath, 0700); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	return &Manager{
		basePath: cfg.BasePath,
		dbs:      make(map[string]*sql.DB, 4), // Pre-allocate for typical 1-4 accounts
	}, nil
}

// OpenSharedDB opens a shared SQLite database for cross-account data (like photos).
func OpenSharedDB(basePath, filename string) (*sql.DB, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("create cache directory: %w", err)
	}

	dbPath := filepath.Join(basePath, filename)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open shared database: %w", err)
	}

	// Enable WAL mode for better concurrency
	_, _ = db.Exec("PRAGMA journal_mode=WAL")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL")

	return db, nil
}

// sanitizeEmail converts email to a safe filename.
// Example: user@example.com -> user@example.com.db
func sanitizeEmail(email string) string {
	// Email addresses are generally safe, but clean up any path separators
	safe := strings.ReplaceAll(email, "/", "_")
	safe = strings.ReplaceAll(safe, "\\", "_")
	safe = strings.ReplaceAll(safe, ":", "_")
	return safe + ".db"
}

func isAccountDBFile(name string) bool {
	if !strings.HasSuffix(name, ".db") || strings.HasSuffix(name, "-wal") || strings.HasSuffix(name, "-shm") {
		return false
	}

	// Shared databases are not per-account caches.
	return name != "photos.db"
}

// DBPath returns the database path for an email.
func (m *Manager) DBPath(email string) string {
	return filepath.Join(m.basePath, sanitizeEmail(email))
}

// GetDB returns or creates a database for the given email.
func (m *Manager) GetDB(email string) (*sql.DB, error) {
	m.mu.RLock()
	db, exists := m.dbs[email]
	m.mu.RUnlock()

	if exists && db != nil {
		return db, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if db, exists = m.dbs[email]; exists && db != nil {
		return db, nil
	}

	// Create new database
	dbPath := m.DBPath(email)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database for %s: %w", email, err)
	}

	// Configure SQLite for better performance
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA temp_store=MEMORY",
		"PRAGMA mmap_size=268435456", // 256MB mmap
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("set pragma: %w", err)
		}
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema for %s: %w", email, err)
	}

	m.dbs[email] = db
	return db, nil
}

// Close closes all open database connections.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for email, db := range m.dbs {
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close %s: %w", email, err))
		}
	}
	m.dbs = make(map[string]*sql.DB)

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// CloseDB closes the database for a specific email.
func (m *Manager) CloseDB(email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, exists := m.dbs[email]
	if !exists {
		return nil
	}

	delete(m.dbs, email)
	return db.Close()
}

// ClearCache removes the cache database for an email.
func (m *Manager) ClearCache(email string) error {
	// Close the database first
	if err := m.CloseDB(email); err != nil {
		return fmt.Errorf("close db: %w", err)
	}

	// Remove the file
	dbPath := m.DBPath(email)
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove cache file: %w", err)
	}

	// Also remove WAL and SHM files
	_ = os.Remove(dbPath + "-wal")
	_ = os.Remove(dbPath + "-shm")

	return nil
}

// ClearAllCaches removes all cache databases.
func (m *Manager) ClearAllCaches() error {
	// Close all databases
	if err := m.Close(); err != nil {
		return err
	}

	// Find and remove all .db files
	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		return fmt.Errorf("read cache dir: %w", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".db") ||
			strings.HasSuffix(entry.Name(), ".db-wal") ||
			strings.HasSuffix(entry.Name(), ".db-shm") {
			path := filepath.Join(m.basePath, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// ListCachedAccounts returns emails that have cache databases.
func (m *Manager) ListCachedAccounts() ([]string, error) {
	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var emails []string
	for _, entry := range entries {
		name := entry.Name()
		if isAccountDBFile(name) {
			email := strings.TrimSuffix(name, ".db")
			emails = append(emails, email)
		}
	}
	return emails, nil
}

// CacheStats contains statistics about a cache database.
type CacheStats struct {
	Email        string
	SizeBytes    int64
	EmailCount   int
	EventCount   int
	ContactCount int
	LastSync     time.Time
}

// GetStats returns statistics for a cache database.
func (m *Manager) GetStats(email string) (*CacheStats, error) {
	db, err := m.GetDB(email)
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{Email: email}

	// Get file size
	info, err := os.Stat(m.DBPath(email))
	if err == nil {
		stats.SizeBytes = info.Size()
	}

	// Count emails
	row := db.QueryRow("SELECT COUNT(*) FROM emails")
	_ = row.Scan(&stats.EmailCount)

	// Count events
	row = db.QueryRow("SELECT COUNT(*) FROM events")
	_ = row.Scan(&stats.EventCount)

	// Count contacts
	row = db.QueryRow("SELECT COUNT(*) FROM contacts")
	_ = row.Scan(&stats.ContactCount)

	// Get last sync time
	var lastSync int64
	row = db.QueryRow("SELECT MAX(last_sync) FROM sync_state")
	if err := row.Scan(&lastSync); err == nil && lastSync > 0 {
		stats.LastSync = time.Unix(lastSync, 0)
	}

	return stats, nil
}
