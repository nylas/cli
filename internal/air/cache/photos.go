package cache

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultPhotoTTL is the default time-to-live for cached photos (30 days).
const DefaultPhotoTTL = 30 * 24 * time.Hour

// CachedPhoto represents a contact photo stored in the cache.
type CachedPhoto struct {
	ContactID   string    `json:"contact_id"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	LocalPath   string    `json:"local_path"`
	CachedAt    time.Time `json:"cached_at"`
	AccessedAt  time.Time `json:"accessed_at"`
}

// PhotoStore provides contact photo caching operations.
type PhotoStore struct {
	db       *sql.DB
	basePath string
	ttl      time.Duration
}

// NewPhotoStore creates a photo store.
func NewPhotoStore(db *sql.DB, basePath string, ttl time.Duration) (*PhotoStore, error) {
	if ttl == 0 {
		ttl = DefaultPhotoTTL
	}

	// Create photos table if not exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS photos (
			contact_id TEXT PRIMARY KEY,
			content_type TEXT NOT NULL,
			size INTEGER NOT NULL,
			local_path TEXT NOT NULL,
			cached_at INTEGER NOT NULL,
			accessed_at INTEGER NOT NULL
		)
	`)
	if err != nil {
		return nil, fmt.Errorf("create photos table: %w", err)
	}

	// Create index for cleanup queries
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_photos_cached_at ON photos(cached_at)")

	// Ensure photos directory exists
	photosDir := filepath.Join(basePath, "photos")
	if err := os.MkdirAll(photosDir, 0700); err != nil {
		return nil, fmt.Errorf("create photos directory: %w", err)
	}

	return &PhotoStore{
		db:       db,
		basePath: photosDir,
		ttl:      ttl,
	}, nil
}

// Put stores a contact photo.
func (s *PhotoStore) Put(contactID, contentType string, data []byte) error {
	// Write photo to file
	localPath := filepath.Join(s.basePath, contactID)
	if err := os.WriteFile(localPath, data, 0600); err != nil {
		return fmt.Errorf("write photo file: %w", err)
	}

	now := time.Now()

	// Save metadata to database
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO photos (
			contact_id, content_type, size, local_path, cached_at, accessed_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`,
		contactID, contentType, len(data), localPath, now.Unix(), now.Unix(),
	)
	if err != nil {
		// Clean up file on database error
		_ = os.Remove(localPath)
		return fmt.Errorf("save photo metadata: %w", err)
	}

	return nil
}

// Get retrieves a cached photo if it exists and is not expired.
// Returns nil, nil if the photo is not cached or expired.
func (s *PhotoStore) Get(contactID string) ([]byte, string, error) {
	row := s.db.QueryRow(`
		SELECT content_type, size, local_path, cached_at
		FROM photos WHERE contact_id = ?
	`, contactID)

	var contentType, localPath string
	var size, cachedAtUnix int64

	err := row.Scan(&contentType, &size, &localPath, &cachedAtUnix)
	if err == sql.ErrNoRows {
		return nil, "", nil // Not cached
	}
	if err != nil {
		return nil, "", fmt.Errorf("query photo: %w", err)
	}

	// Check if expired
	cachedAt := time.Unix(cachedAtUnix, 0)
	if time.Since(cachedAt) > s.ttl {
		// Expired - delete and return nil
		_ = s.Delete(contactID)
		return nil, "", nil
	}

	// Read photo from file
	// #nosec G304 -- localPath constructed from validated cache directory and contact ID
	data, err := os.ReadFile(localPath)
	if err != nil {
		// File missing - delete metadata and return nil
		_ = s.Delete(contactID)
		return nil, "", nil
	}

	// Update accessed time
	_, _ = s.db.Exec("UPDATE photos SET accessed_at = ? WHERE contact_id = ?", time.Now().Unix(), contactID)

	return data, contentType, nil
}

// IsValid checks if a cached photo exists and is not expired.
func (s *PhotoStore) IsValid(contactID string) bool {
	var cachedAtUnix int64
	err := s.db.QueryRow("SELECT cached_at FROM photos WHERE contact_id = ?", contactID).Scan(&cachedAtUnix)
	if err != nil {
		return false
	}

	cachedAt := time.Unix(cachedAtUnix, 0)
	return time.Since(cachedAt) <= s.ttl
}

// Delete removes a cached photo.
func (s *PhotoStore) Delete(contactID string) error {
	// Get local path first
	var localPath string
	err := s.db.QueryRow("SELECT local_path FROM photos WHERE contact_id = ?", contactID).Scan(&localPath)
	if err == nil && localPath != "" {
		_ = os.Remove(localPath)
	}

	// Delete from database
	_, err = s.db.Exec("DELETE FROM photos WHERE contact_id = ?", contactID)
	return err
}

// Prune removes expired photos.
func (s *PhotoStore) Prune() (int, error) {
	cutoff := time.Now().Add(-s.ttl).Unix()

	// Get expired photos
	rows, err := s.db.Query("SELECT contact_id, local_path FROM photos WHERE cached_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("query expired photos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var toDelete []string
	for rows.Next() {
		var contactID, localPath string
		if err := rows.Scan(&contactID, &localPath); err == nil {
			toDelete = append(toDelete, contactID)
			_ = os.Remove(localPath)
		}
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	// Delete from database
	_, err = s.db.Exec("DELETE FROM photos WHERE cached_at < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete expired photos: %w", err)
	}

	return len(toDelete), nil
}

// Count returns the number of cached photos.
func (s *PhotoStore) Count() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM photos").Scan(&count)
	return count, err
}

// TotalSize returns the total size of cached photos in bytes.
func (s *PhotoStore) TotalSize() (int64, error) {
	var size int64
	err := s.db.QueryRow("SELECT COALESCE(SUM(size), 0) FROM photos").Scan(&size)
	return size, err
}

// Close releases the underlying photo metadata database handle.
func (s *PhotoStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// RemoveOrphaned removes photo files not referenced in database.
func (s *PhotoStore) RemoveOrphaned() (int, error) {
	// Get all known contact IDs
	rows, err := s.db.Query("SELECT contact_id FROM photos")
	if err != nil {
		return 0, err
	}

	knownIDs := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			knownIDs[id] = true
		}
	}
	_ = rows.Close()

	// Walk the photos directory and remove unknown files
	count := 0
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !knownIDs[entry.Name()] {
			_ = os.Remove(filepath.Join(s.basePath, entry.Name()))
			count++
		}
	}

	return count, nil
}

// PhotoCacheStats contains statistics about the photo cache.
type PhotoCacheStats struct {
	Count     int
	TotalSize int64
	TTLDays   int
	Oldest    time.Time
	Newest    time.Time
}

// GetStats returns photo cache statistics.
func (s *PhotoStore) GetStats() (*PhotoCacheStats, error) {
	stats := &PhotoCacheStats{
		TTLDays: int(s.ttl.Hours() / 24),
	}

	count, err := s.Count()
	if err != nil {
		return nil, err
	}
	stats.Count = count

	size, err := s.TotalSize()
	if err != nil {
		return nil, err
	}
	stats.TotalSize = size

	var oldestUnix, newestUnix int64
	_ = s.db.QueryRow("SELECT MIN(cached_at) FROM photos").Scan(&oldestUnix)
	_ = s.db.QueryRow("SELECT MAX(cached_at) FROM photos").Scan(&newestUnix)

	if oldestUnix > 0 {
		stats.Oldest = time.Unix(oldestUnix, 0)
	}
	if newestUnix > 0 {
		stats.Newest = time.Unix(newestUnix, 0)
	}

	return stats, nil
}
