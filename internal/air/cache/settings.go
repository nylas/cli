package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Settings holds all cache configuration.
type Settings struct {
	// Cache behavior
	Enabled             bool `json:"cache_enabled"`
	MaxSizeMB           int  `json:"cache_max_size_mb"`
	TTLDays             int  `json:"cache_ttl_days"`
	SyncIntervalMinutes int  `json:"sync_interval_minutes"`
	OfflineQueueEnabled bool `json:"offline_queue_enabled"`
	EncryptionEnabled   bool `json:"encryption_enabled"`

	// Attachment settings
	AttachmentCacheEnabled bool `json:"attachment_cache_enabled"`
	AttachmentMaxSizeMB    int  `json:"attachment_max_size_mb"`

	// Sync settings
	InitialSyncDays       int  `json:"initial_sync_days"`
	BackgroundSyncEnabled bool `json:"background_sync_enabled"`

	// UI preferences (also stored here for convenience)
	Theme           string `json:"theme,omitempty"`        // "dark", "light", "system"
	DefaultView     string `json:"default_view,omitempty"` // "email", "calendar", "contacts"
	CompactMode     bool   `json:"compact_mode,omitempty"`
	PreviewPosition string `json:"preview_position,omitempty"` // "right", "bottom", "off"

	mu       sync.RWMutex `json:"-"`
	filePath string       `json:"-"`
}

// DefaultSettings returns default cache settings.
func DefaultSettings() *Settings {
	return &Settings{
		Enabled:                true,
		MaxSizeMB:              500,
		TTLDays:                30,
		SyncIntervalMinutes:    5,
		OfflineQueueEnabled:    true,
		EncryptionEnabled:      false,
		AttachmentCacheEnabled: true,
		AttachmentMaxSizeMB:    100,
		InitialSyncDays:        30,
		BackgroundSyncEnabled:  true,
		Theme:                  "dark",
		DefaultView:            "email",
		CompactMode:            false,
		PreviewPosition:        "right",
	}
}

// LoadSettings loads settings from file, or creates default if not exists.
func LoadSettings(basePath string) (*Settings, error) {
	filePath := filepath.Join(basePath, "settings.json")

	settings := DefaultSettings()
	settings.filePath = filePath

	// #nosec G304 -- filePath constructed from validated cache directory
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		// Create default settings file
		if err := settings.Save(); err != nil {
			return nil, fmt.Errorf("create default settings: %w", err)
		}
		return settings, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read settings file: %w", err)
	}

	if err := json.Unmarshal(data, settings); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	return settings, nil
}

// Save writes settings to file.
func (s *Settings) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}

	return nil
}

// Get returns a copy of settings (thread-safe).
func (s *Settings) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	return Settings{
		Enabled:                s.Enabled,
		MaxSizeMB:              s.MaxSizeMB,
		TTLDays:                s.TTLDays,
		SyncIntervalMinutes:    s.SyncIntervalMinutes,
		OfflineQueueEnabled:    s.OfflineQueueEnabled,
		EncryptionEnabled:      s.EncryptionEnabled,
		AttachmentCacheEnabled: s.AttachmentCacheEnabled,
		AttachmentMaxSizeMB:    s.AttachmentMaxSizeMB,
		InitialSyncDays:        s.InitialSyncDays,
		BackgroundSyncEnabled:  s.BackgroundSyncEnabled,
		Theme:                  s.Theme,
		DefaultView:            s.DefaultView,
		CompactMode:            s.CompactMode,
		PreviewPosition:        s.PreviewPosition,
	}
}

// Update updates settings with the provided function.
func (s *Settings) Update(fn func(*Settings)) error {
	s.mu.Lock()
	fn(s)
	s.mu.Unlock()

	return s.Save()
}

// SetEnabled enables or disables caching.
func (s *Settings) SetEnabled(enabled bool) error {
	return s.Update(func(s *Settings) {
		s.Enabled = enabled
	})
}

// SetMaxSize sets the maximum cache size in MB.
func (s *Settings) SetMaxSize(sizeMB int) error {
	if sizeMB < 50 {
		sizeMB = 50 // Minimum 50MB
	}
	if sizeMB > 10000 {
		sizeMB = 10000 // Maximum 10GB
	}
	return s.Update(func(s *Settings) {
		s.MaxSizeMB = sizeMB
	})
}

// SetEncryption enables or disables encryption.
func (s *Settings) SetEncryption(enabled bool) error {
	return s.Update(func(s *Settings) {
		s.EncryptionEnabled = enabled
	})
}

// SetTheme sets the UI theme.
func (s *Settings) SetTheme(theme string) error {
	if theme != "dark" && theme != "light" && theme != "system" {
		theme = "dark"
	}
	return s.Update(func(s *Settings) {
		s.Theme = theme
	})
}

// GetSyncInterval returns the sync interval as a duration.
func (s *Settings) GetSyncInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Duration(s.SyncIntervalMinutes) * time.Minute
}

// GetTTL returns the cache TTL as a duration.
func (s *Settings) GetTTL() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return time.Duration(s.TTLDays) * 24 * time.Hour
}

// GetMaxSizeBytes returns the maximum cache size in bytes.
func (s *Settings) GetMaxSizeBytes() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(s.MaxSizeMB) * 1024 * 1024
}

// IsEncryptionEnabled returns whether encryption is enabled.
func (s *Settings) IsEncryptionEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.EncryptionEnabled
}

// IsCacheEnabled returns whether caching is enabled.
func (s *Settings) IsCacheEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Enabled
}

// ToConfig converts settings to cache Config.
func (s *Settings) ToConfig(basePath string) Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return Config{
		BasePath:            basePath,
		MaxSizeMB:           s.MaxSizeMB,
		TTLDays:             s.TTLDays,
		SyncIntervalMinutes: s.SyncIntervalMinutes,
	}
}

// ToEncryptionConfig converts settings to EncryptionConfig.
func (s *Settings) ToEncryptionConfig() EncryptionConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return EncryptionConfig{
		Enabled: s.EncryptionEnabled,
	}
}

// BasePath returns the directory containing the settings file.
func (s *Settings) BasePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return filepath.Dir(s.filePath)
}

// Validate checks if settings are valid.
func (s *Settings) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.MaxSizeMB < 50 {
		return fmt.Errorf("cache_max_size_mb must be at least 50")
	}
	if s.TTLDays < 1 {
		return fmt.Errorf("cache_ttl_days must be at least 1")
	}
	if s.SyncIntervalMinutes < 1 {
		return fmt.Errorf("sync_interval_minutes must be at least 1")
	}
	if s.InitialSyncDays < 1 {
		return fmt.Errorf("initial_sync_days must be at least 1")
	}

	return nil
}

// Reset restores default settings.
func (s *Settings) Reset() error {
	defaults := DefaultSettings()

	s.mu.Lock()
	// Copy all fields except mu and filePath (don't copy mutex)
	s.Enabled = defaults.Enabled
	s.MaxSizeMB = defaults.MaxSizeMB
	s.TTLDays = defaults.TTLDays
	s.SyncIntervalMinutes = defaults.SyncIntervalMinutes
	s.OfflineQueueEnabled = defaults.OfflineQueueEnabled
	s.EncryptionEnabled = defaults.EncryptionEnabled
	s.AttachmentCacheEnabled = defaults.AttachmentCacheEnabled
	s.AttachmentMaxSizeMB = defaults.AttachmentMaxSizeMB
	s.InitialSyncDays = defaults.InitialSyncDays
	s.BackgroundSyncEnabled = defaults.BackgroundSyncEnabled
	s.Theme = defaults.Theme
	s.DefaultView = defaults.DefaultView
	s.CompactMode = defaults.CompactMode
	s.PreviewPosition = defaults.PreviewPosition
	s.mu.Unlock()

	return s.Save()
}
