// Package audit provides audit log storage using JSON Lines files.
package audit

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nylas/cli/internal/domain"
)

const (
	configFileName = "config.json"
	logFileExt     = ".jsonl"
	dateFormat     = "2006-01-02"
)

// FileStore implements AuditStore using JSON Lines files.
type FileStore struct {
	basePath string
	mu       sync.RWMutex
	config   *domain.AuditConfig
}

// NewFileStore creates a new audit file store.
// If basePath is empty, uses default path (~/.config/nylas/audit).
func NewFileStore(basePath string) (*FileStore, error) {
	if basePath == "" {
		basePath = DefaultAuditPath()
	}

	store := &FileStore{
		basePath: basePath,
	}

	// Load or create config
	cfg, err := store.loadConfig()
	if err != nil {
		// If config doesn't exist, use defaults but don't save yet
		cfg = domain.DefaultAuditConfig()
		cfg.Path = basePath
	}
	store.config = cfg

	return store, nil
}

// DefaultAuditPath returns the default audit log directory.
func DefaultAuditPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "nylas", "audit")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nylas", "audit")
}

// GetConfig returns the current audit configuration.
func (s *FileStore) GetConfig() (*domain.AuditConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config, nil
}

// SaveConfig saves the audit configuration.
func (s *FileStore) SaveConfig(cfg *domain.AuditConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(s.basePath, 0700); err != nil {
		return fmt.Errorf("create audit directory: %w", err)
	}

	// Save config file
	configPath := filepath.Join(s.basePath, configFileName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	s.config = cfg
	return nil
}

// Log records an audit entry.
func (s *FileStore) Log(entry *domain.AuditEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if logging is enabled
	if s.config == nil || !s.config.Enabled {
		return nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(s.basePath, 0700); err != nil {
		return fmt.Errorf("create audit directory: %w", err)
	}

	// Generate ID if not set
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}

	// Set timestamp if not set
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	// Determine log file path
	var logPath string
	if s.config.RotateDaily {
		logPath = filepath.Join(s.basePath, entry.Timestamp.Format(dateFormat)+logFileExt)
	} else {
		logPath = filepath.Join(s.basePath, "audit"+logFileExt)
	}

	// Open file for appending
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Write entry as JSON line
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}

	return nil
}

// Path returns the audit log directory path.
func (s *FileStore) Path() string {
	return s.basePath
}

// loadConfig loads the config from file.
func (s *FileStore) loadConfig() (*domain.AuditConfig, error) {
	configPath := filepath.Join(s.basePath, configFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg domain.AuditConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// List returns recent audit entries with optional limit.
func (s *FileStore) List(ctx context.Context, limit int) ([]domain.AuditEntry, error) {
	return s.Query(ctx, &domain.AuditQueryOptions{Limit: limit})
}

// Query returns audit entries matching the given options.
func (s *FileStore) Query(ctx context.Context, opts *domain.AuditQueryOptions) ([]domain.AuditEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if opts == nil {
		opts = &domain.AuditQueryOptions{}
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}

	// Get all log files
	files, err := s.getLogFiles()
	if err != nil {
		return nil, err
	}

	// Sort files by date descending (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i] > files[j]
	})

	var entries []domain.AuditEntry

	// Read files until we have enough entries
	for _, file := range files {
		select {
		case <-ctx.Done():
			return entries, ctx.Err()
		default:
		}

		fileEntries, err := s.readLogFile(filepath.Join(s.basePath, file))
		if err != nil {
			continue // Skip files that can't be read
		}

		// Filter entries
		for _, entry := range fileEntries {
			if s.matchesQuery(&entry, opts) {
				entries = append(entries, entry)
			}
		}

		if len(entries) >= opts.Limit*2 { // Read extra for filtering
			break
		}
	}

	// Sort by timestamp descending (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	// Apply limit
	if len(entries) > opts.Limit {
		entries = entries[:opts.Limit]
	}

	return entries, nil
}

// matchesQuery checks if an entry matches the query options.
func (s *FileStore) matchesQuery(entry *domain.AuditEntry, opts *domain.AuditQueryOptions) bool {
	if !opts.Since.IsZero() && entry.Timestamp.Before(opts.Since) {
		return false
	}
	if !opts.Until.IsZero() && entry.Timestamp.After(opts.Until) {
		return false
	}
	if opts.Command != "" && !strings.HasPrefix(entry.Command, opts.Command) {
		return false
	}
	if opts.Status != "" && string(entry.Status) != opts.Status {
		return false
	}
	if opts.GrantID != "" && entry.GrantID != opts.GrantID {
		return false
	}
	if opts.RequestID != "" && entry.RequestID != opts.RequestID {
		return false
	}
	if opts.Invoker != "" && !strings.Contains(entry.Invoker, opts.Invoker) {
		return false
	}
	if opts.InvokerSource != "" && entry.InvokerSource != opts.InvokerSource {
		return false
	}
	return true
}

// getLogFiles returns all log file names in the audit directory.
func (s *FileStore) getLogFiles() ([]string, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), logFileExt) {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// readLogFile reads all entries from a log file.
func (s *FileStore) readLogFile(path string) ([]domain.AuditEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var entries []domain.AuditEntry
	scanner := bufio.NewScanner(f)
	// Increase buffer size for long lines
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		var entry domain.AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip invalid lines
		}
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}

// Clear removes all audit logs.
func (s *FileStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := s.getLogFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		path := filepath.Join(s.basePath, file)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", file, err)
		}
	}

	return nil
}

// Stats returns storage statistics.
func (s *FileStore) Stats() (fileCount int, totalSizeBytes int64, oldestEntry *domain.AuditEntry, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	files, err := s.getLogFiles()
	if err != nil {
		return 0, 0, nil, err
	}

	fileCount = len(files)

	for _, file := range files {
		info, err := os.Stat(filepath.Join(s.basePath, file))
		if err != nil {
			continue
		}
		totalSizeBytes += info.Size()
	}

	// Find oldest entry
	if len(files) > 0 {
		sort.Strings(files) // Sort by date ascending
		entries, err := s.readLogFile(filepath.Join(s.basePath, files[0]))
		if err == nil && len(entries) > 0 {
			oldestEntry = &entries[0]
		}
	}

	return fileCount, totalSizeBytes, oldestEntry, nil
}

// Cleanup removes old log files based on retention settings.
func (s *FileStore) Cleanup(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config == nil || s.config.RetentionDays <= 0 {
		return nil
	}

	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	files, err := s.getLogFiles()
	if err != nil {
		return err
	}

	for _, file := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Parse date from filename
		name := strings.TrimSuffix(file, logFileExt)
		fileDate, err := time.Parse(dateFormat, name)
		if err != nil {
			continue // Not a dated file, skip
		}

		if fileDate.Before(cutoff) {
			path := filepath.Join(s.basePath, file)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove %s: %w", file, err)
			}
		}
	}

	return nil
}
