package cache

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	// Import for side effects - registers sqlite3 driver and adiantum VFS
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	_ "github.com/ncruces/go-sqlite3/vfs/adiantum"

	"github.com/zalando/go-keyring"
)

const (
	// keyringService is the service name for storing encryption keys.
	keyringService = "nylas-air-cache"
	// keySize is the size of the encryption key in bytes (256-bit).
	keySize = 32
	// encryptedVFSName is the name of the encrypted VFS.
	encryptedVFSName = "adiantum"
	// driverName is the name of the registered SQL driver.
	driverName = "sqlite3"
)

// allowedTables is a whitelist of table names that can be used in SQL queries.
// This prevents SQL injection by ensuring only known table names are used.
var allowedTables = map[string]bool{
	"emails":        true,
	"events":        true,
	"contacts":      true,
	"folders":       true,
	"calendars":     true,
	"sync_state":    true,
	"attachments":   true,
	"offline_queue": true,
}

var (
	getOrCreateKeyFunc = getOrCreateKey
	deleteKeyFunc      = deleteKey
)

// tableNames returns the list of allowed table names for migration operations.
func tableNames() []string {
	names := make([]string, 0, len(allowedTables))
	for name := range allowedTables {
		names = append(names, name)
	}
	return names
}

// EncryptionConfig holds encryption configuration.
type EncryptionConfig struct {
	Enabled bool
	KeyID   string // Identifier for the key in keyring (usually email)
}

// generateKey generates a new random 256-bit encryption key.
func generateKey() ([]byte, error) {
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate random key: %w", err)
	}
	return key, nil
}

// getOrCreateKey retrieves the encryption key from keyring, or creates a new one.
func getOrCreateKey(keyID string) ([]byte, error) {
	// Try to get existing key
	hexKey, err := keyring.Get(keyringService, keyID)
	if err == nil && hexKey != "" {
		key, decodeErr := hex.DecodeString(hexKey)
		if decodeErr == nil && len(key) == keySize {
			return key, nil
		}
		// Invalid key format, regenerate
	}

	// Generate new key
	key, err := generateKey()
	if err != nil {
		return nil, err
	}

	// Store in keyring
	hexKey = hex.EncodeToString(key)
	if err := keyring.Set(keyringService, keyID, hexKey); err != nil {
		return nil, fmt.Errorf("store key in keyring: %w", err)
	}

	return key, nil
}

// deleteKey removes the encryption key from keyring.
func deleteKey(keyID string) error {
	err := keyring.Delete(keyringService, keyID)
	if err == keyring.ErrNotFound {
		return nil // Already deleted
	}
	return err
}

// openEncryptedDB opens an encrypted SQLite database.
func openEncryptedDB(dbPath string, key []byte) (*sql.DB, error) {
	// Format: file:path?vfs=adiantum&_pragma=hexkey('hexkey')
	hexKey := hex.EncodeToString(key)
	dsn := fmt.Sprintf("file:%s?vfs=%s&_pragma=hexkey('%s')", dbPath, encryptedVFSName, hexKey)

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open encrypted database: %w", err)
	}

	// Verify the key works by reading the schema, which fails with the wrong key.
	var schemaObjects int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&schemaObjects); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("verify encryption key: %w", err)
	}

	return db, nil
}

// EncryptedManager extends Manager with encryption support.
type EncryptedManager struct {
	*Manager
	encryption EncryptionConfig
	keys       map[string][]byte // Cache of loaded keys
}

// NewEncryptedManager creates a new cache manager with encryption support.
func NewEncryptedManager(cfg Config, encCfg EncryptionConfig) (*EncryptedManager, error) {
	mgr, err := NewManager(cfg)
	if err != nil {
		return nil, err
	}

	return &EncryptedManager{
		Manager:    mgr,
		encryption: encCfg,
		keys:       make(map[string][]byte),
	}, nil
}

// GetDB returns or creates an encrypted database for the given email.
func (m *EncryptedManager) GetDB(email string) (*sql.DB, error) {
	if !m.encryption.Enabled {
		return m.Manager.GetDB(email)
	}

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

	// Get or create encryption key
	key, err := getOrCreateKeyFunc(email)
	if err != nil {
		return nil, fmt.Errorf("get encryption key for %s: %w", email, err)
	}
	m.keys[email] = key

	// Open encrypted database
	dbPath := m.DBPath(email)
	db, err = openEncryptedDB(dbPath, key)
	if err != nil {
		return nil, fmt.Errorf("open encrypted database for %s: %w", email, err)
	}

	// Configure SQLite for better performance
	pragmas := []string{
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA temp_store=MEMORY",
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

// ClearCache removes the cache database and encryption key for an email.
func (m *EncryptedManager) ClearCache(email string) error {
	// Close and remove database
	if err := m.Manager.ClearCache(email); err != nil {
		return err
	}

	// Remove encryption key if encryption is enabled
	if m.encryption.Enabled {
		delete(m.keys, email)
		if err := deleteKeyFunc(email); err != nil {
			// Log but don't fail - key might not exist
			fmt.Fprintf(os.Stderr, "warning: failed to delete encryption key: %v\n", err)
		}
	}

	return nil
}

// MigrateToEncrypted migrates an unencrypted database to encrypted.
func (m *EncryptedManager) MigrateToEncrypted(email string) error {
	dbPath := m.DBPath(email)

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil // Nothing to migrate
	}

	// Open unencrypted database
	unencryptedDB, err := sql.Open(driverName, dbPath)
	if err != nil {
		return fmt.Errorf("open unencrypted database: %w", err)
	}
	defer func() { _ = unencryptedDB.Close() }()

	// Get or create encryption key
	key, err := getOrCreateKeyFunc(email)
	if err != nil {
		return fmt.Errorf("get encryption key: %w", err)
	}

	// Create encrypted database with .encrypted suffix
	encryptedPath := dbPath + ".encrypted"
	encryptedDB, err := openEncryptedDB(encryptedPath, key)
	if err != nil {
		return fmt.Errorf("create encrypted database: %w", err)
	}
	defer func() { _ = encryptedDB.Close() }()

	// Initialize schema in encrypted database
	if err := initSchema(encryptedDB); err != nil {
		return fmt.Errorf("init encrypted schema: %w", err)
	}

	// Copy data (for each table)
	tables := tableNames()
	for _, table := range tables {
		if err := copyTable(unencryptedDB, encryptedDB, table); err != nil {
			_ = os.Remove(encryptedPath)
			return fmt.Errorf("copy table %s: %w", table, err)
		}
	}

	// Close databases before file operations
	_ = unencryptedDB.Close()
	_ = encryptedDB.Close()

	// Backup unencrypted database
	backupPath := dbPath + ".unencrypted.bak"
	if err := os.Rename(dbPath, backupPath); err != nil {
		return fmt.Errorf("backup unencrypted database: %w", err)
	}

	// Move encrypted database to original path
	if err := os.Rename(encryptedPath, dbPath); err != nil {
		// Restore backup
		_ = os.Rename(backupPath, dbPath)
		return fmt.Errorf("replace with encrypted database: %w", err)
	}

	// Remove backup
	_ = os.Remove(backupPath)
	_ = os.Remove(backupPath + "-wal")
	_ = os.Remove(backupPath + "-shm")

	return nil
}

// MigrateToUnencrypted migrates an encrypted database to unencrypted.
func (m *EncryptedManager) MigrateToUnencrypted(email string) error {
	dbPath := m.DBPath(email)

	// Check if database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return nil // Nothing to migrate
	}

	// Get encryption key
	key, ok := m.keys[email]
	if !ok {
		var err error
		key, err = getOrCreateKeyFunc(email)
		if err != nil {
			return fmt.Errorf("get encryption key: %w", err)
		}
	}

	// Open encrypted database
	encryptedDB, err := openEncryptedDB(dbPath, key)
	if err != nil {
		return fmt.Errorf("open encrypted database: %w", err)
	}
	defer func() { _ = encryptedDB.Close() }()

	// Create unencrypted database
	unencryptedPath := dbPath + ".unencrypted"
	unencryptedDB, err := sql.Open(driverName, unencryptedPath)
	if err != nil {
		return fmt.Errorf("create unencrypted database: %w", err)
	}
	defer func() { _ = unencryptedDB.Close() }()

	// Initialize schema
	if err := initSchema(unencryptedDB); err != nil {
		return fmt.Errorf("init unencrypted schema: %w", err)
	}

	// Copy data
	tables := tableNames()
	for _, table := range tables {
		if err := copyTable(encryptedDB, unencryptedDB, table); err != nil {
			_ = os.Remove(unencryptedPath)
			return fmt.Errorf("copy table %s: %w", table, err)
		}
	}

	// Close databases
	_ = encryptedDB.Close()
	_ = unencryptedDB.Close()

	// Replace encrypted with unencrypted
	backupPath := dbPath + ".encrypted.bak"
	if err := os.Rename(dbPath, backupPath); err != nil {
		return fmt.Errorf("backup encrypted database: %w", err)
	}

	if err := os.Rename(unencryptedPath, dbPath); err != nil {
		_ = os.Rename(backupPath, dbPath)
		return fmt.Errorf("replace with unencrypted database: %w", err)
	}

	// Remove backup and encryption key
	_ = os.Remove(backupPath)
	_ = os.Remove(backupPath + "-wal")
	_ = os.Remove(backupPath + "-shm")
	_ = deleteKeyFunc(email)
	delete(m.keys, email)

	return nil
}

// ClearAllCaches removes all encrypted cache databases and associated keys.
func (m *EncryptedManager) ClearAllCaches() error {
	accounts, err := m.ListCachedAccounts()
	if err != nil {
		return err
	}

	if err := m.Manager.ClearAllCaches(); err != nil {
		return err
	}

	for _, email := range accounts {
		delete(m.keys, email)
		if err := deleteKeyFunc(email); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to delete encryption key: %v\n", err)
		}
	}

	return nil
}

// copyTable copies all rows from one table to another.
func copyTable(src, dst *sql.DB, table string) error {
	// Validate table name against whitelist to prevent SQL injection
	if !allowedTables[table] {
		return fmt.Errorf("invalid table name: %s", table)
	}

	// Get column names
	rows, err := src.Query(fmt.Sprintf("SELECT * FROM %s LIMIT 0", table)) //nolint:gosec // table name validated above
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	_ = rows.Close()
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return nil // Empty table or doesn't exist
	}

	// Build INSERT statement
	placeholders := "?"
	for i := 1; i < len(columns); i++ {
		placeholders += ", ?"
	}
	insertSQL := fmt.Sprintf("INSERT INTO %s VALUES (%s)", table, placeholders) //nolint:gosec // table name validated above

	// Copy rows
	rows, err = src.Query(fmt.Sprintf("SELECT * FROM %s", table)) //nolint:gosec // table name validated above
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	tx, err := dst.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}
		if _, err := stmt.Exec(values...); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// IsEncrypted checks if the database for an email is encrypted.
func IsEncrypted(dbPath string) (bool, error) {
	// Try to open as unencrypted
	db, err := sql.Open(driverName, dbPath)
	if err != nil {
		return false, err
	}
	defer func() { _ = db.Close() }()

	// Read the schema - this fails when the database is encrypted and opened without a key.
	var schemaObjects int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&schemaObjects)
	if err != nil {
		// Database exists but can't be read - likely encrypted
		return true, nil
	}

	return false, nil
}

// GetStats returns statistics for an encrypted cache database.
func (m *EncryptedManager) GetStats(email string) (*CacheStats, error) {
	db, err := m.GetDB(email)
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{Email: email}

	info, err := os.Stat(m.DBPath(email))
	if err == nil {
		stats.SizeBytes = info.Size()
	}

	row := db.QueryRow("SELECT COUNT(*) FROM emails")
	_ = row.Scan(&stats.EmailCount)

	row = db.QueryRow("SELECT COUNT(*) FROM events")
	_ = row.Scan(&stats.EventCount)

	row = db.QueryRow("SELECT COUNT(*) FROM contacts")
	_ = row.Scan(&stats.ContactCount)

	var lastSync int64
	row = db.QueryRow("SELECT MAX(last_sync) FROM sync_state")
	if err := row.Scan(&lastSync); err == nil && lastSync > 0 {
		stats.LastSync = time.Unix(lastSync, 0)
	}

	return stats, nil
}
