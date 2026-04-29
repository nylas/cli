package cache

import (
	"os"
	"path/filepath"
	"testing"
)

// Tests for settings.go functions

// ================================
// SETTINGS.GO TESTS
// ================================

func TestLoadSettings_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	if settings == nil {
		t.Fatal("LoadSettings() returned nil")
		return
	}

	// Should have default values
	if !settings.Enabled {
		t.Error("Default Enabled should be true")
	}
	if settings.MaxSizeMB != 500 {
		t.Errorf("Default MaxSizeMB = %d, want 500", settings.MaxSizeMB)
	}

	// File should exist now
	settingsPath := filepath.Join(tmpDir, "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		t.Error("Settings file should have been created")
	}
}

func TestDefaultSettingsBasePathIsEmptyUntilLoaded(t *testing.T) {
	settings := DefaultSettings()

	if got := settings.BasePath(); got != "" {
		t.Fatalf("DefaultSettings().BasePath() = %q, want empty path", got)
	}
}

func TestLoadSettings_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a custom settings file
	settingsPath := filepath.Join(tmpDir, "settings.json")
	customSettings := `{
		"cache_enabled": false,
		"cache_max_size_mb": 1000,
		"cache_ttl_days": 60,
		"theme": "light"
	}`
	if err := os.WriteFile(settingsPath, []byte(customSettings), 0600); err != nil {
		t.Fatalf("Failed to create settings file: %v", err)
	}

	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	if settings.Enabled {
		t.Error("Enabled should be false from custom settings")
	}
	if settings.MaxSizeMB != 1000 {
		t.Errorf("MaxSizeMB = %d, want 1000", settings.MaxSizeMB)
	}
	if settings.TTLDays != 60 {
		t.Errorf("TTLDays = %d, want 60", settings.TTLDays)
	}
	if settings.Theme != "light" {
		t.Errorf("Theme = %q, want 'light'", settings.Theme)
	}
}

func TestLoadSettings_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid JSON file
	settingsPath := filepath.Join(tmpDir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("not valid json{"), 0600); err != nil {
		t.Fatalf("Failed to create settings file: %v", err)
	}

	_, err := LoadSettings(tmpDir)
	if err == nil {
		t.Error("LoadSettings() should error on invalid JSON")
	}
}

func TestSettings_SetMaxSize_Bounds(t *testing.T) {
	tmpDir := t.TempDir()

	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	// Test minimum bound
	if err := settings.SetMaxSize(10); err != nil {
		t.Fatalf("SetMaxSize(10) error: %v", err)
	}
	if settings.MaxSizeMB != 50 {
		t.Errorf("MaxSizeMB should be clamped to minimum 50, got %d", settings.MaxSizeMB)
	}

	// Test maximum bound
	if err := settings.SetMaxSize(20000); err != nil {
		t.Fatalf("SetMaxSize(20000) error: %v", err)
	}
	if settings.MaxSizeMB != 10000 {
		t.Errorf("MaxSizeMB should be clamped to maximum 10000, got %d", settings.MaxSizeMB)
	}
}

func TestSettings_SetTheme_Invalid(t *testing.T) {
	tmpDir := t.TempDir()

	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	// Invalid theme should default to "dark"
	if err := settings.SetTheme("invalid_theme"); err != nil {
		t.Fatalf("SetTheme() error: %v", err)
	}
	if settings.Theme != "dark" {
		t.Errorf("Theme should default to 'dark', got %q", settings.Theme)
	}

	// Valid themes should work
	for _, theme := range []string{"light", "dark", "system"} {
		if err := settings.SetTheme(theme); err != nil {
			t.Fatalf("SetTheme(%q) error: %v", theme, err)
		}
		if settings.Theme != theme {
			t.Errorf("Theme = %q, want %q", settings.Theme, theme)
		}
	}
}

func TestSettings_Validate_AllErrors(t *testing.T) {
	tests := []struct {
		name      string
		modify    func(*Settings)
		wantError string
	}{
		{
			name:      "MaxSizeMB too small",
			modify:    func(s *Settings) { s.MaxSizeMB = 10 },
			wantError: "cache_max_size_mb",
		},
		{
			name:      "TTLDays too small",
			modify:    func(s *Settings) { s.TTLDays = 0 },
			wantError: "cache_ttl_days",
		},
		{
			name:      "SyncIntervalMinutes too small",
			modify:    func(s *Settings) { s.SyncIntervalMinutes = 0 },
			wantError: "sync_interval_minutes",
		},
		{
			name:      "InitialSyncDays too small",
			modify:    func(s *Settings) { s.InitialSyncDays = 0 },
			wantError: "initial_sync_days",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := DefaultSettings()
			tt.modify(settings)

			err := settings.Validate()
			if err == nil {
				t.Error("Validate() should return an error")
			}
		})
	}
}

func TestSettings_GetSyncInterval(t *testing.T) {
	settings := DefaultSettings()
	settings.SyncIntervalMinutes = 10

	interval := settings.GetSyncInterval()
	expected := 10 * 60 * 1000000000 // 10 minutes in nanoseconds

	if interval.Nanoseconds() != int64(expected) {
		t.Errorf("GetSyncInterval() = %v, want %v nanoseconds", interval, expected)
	}
}

func TestSettings_GetTTL(t *testing.T) {
	settings := DefaultSettings()
	settings.TTLDays = 30

	ttl := settings.GetTTL()
	expected := 30 * 24 * 60 * 60 * 1000000000 // 30 days in nanoseconds

	if ttl.Nanoseconds() != int64(expected) {
		t.Errorf("GetTTL() = %v, want %v nanoseconds", ttl, expected)
	}
}

func TestSettings_GetMaxSizeBytes(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxSizeMB = 500

	bytes := settings.GetMaxSizeBytes()
	expected := int64(500 * 1024 * 1024)

	if bytes != expected {
		t.Errorf("GetMaxSizeBytes() = %d, want %d", bytes, expected)
	}
}

func TestSettings_IsEncryptionEnabled(t *testing.T) {
	settings := DefaultSettings()

	// Default is false
	if settings.IsEncryptionEnabled() {
		t.Error("IsEncryptionEnabled() should be false by default")
	}

	settings.EncryptionEnabled = true
	if !settings.IsEncryptionEnabled() {
		t.Error("IsEncryptionEnabled() should be true after setting")
	}
}

func TestSettings_IsCacheEnabled(t *testing.T) {
	settings := DefaultSettings()

	// Default is true
	if !settings.IsCacheEnabled() {
		t.Error("IsCacheEnabled() should be true by default")
	}

	settings.Enabled = false
	if settings.IsCacheEnabled() {
		t.Error("IsCacheEnabled() should be false after disabling")
	}
}

func TestSettings_SetEnabled(t *testing.T) {
	tmpDir := t.TempDir()

	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	// Disable caching
	if err := settings.SetEnabled(false); err != nil {
		t.Fatalf("SetEnabled(false) error: %v", err)
	}

	if settings.Enabled {
		t.Error("Enabled should be false after SetEnabled(false)")
	}

	// Re-enable
	if err := settings.SetEnabled(true); err != nil {
		t.Fatalf("SetEnabled(true) error: %v", err)
	}

	if !settings.Enabled {
		t.Error("Enabled should be true after SetEnabled(true)")
	}
}

func TestSettings_SetEncryption(t *testing.T) {
	tmpDir := t.TempDir()

	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	// Enable encryption
	if err := settings.SetEncryption(true); err != nil {
		t.Fatalf("SetEncryption(true) error: %v", err)
	}

	if !settings.EncryptionEnabled {
		t.Error("EncryptionEnabled should be true after SetEncryption(true)")
	}

	// Disable encryption
	if err := settings.SetEncryption(false); err != nil {
		t.Fatalf("SetEncryption(false) error: %v", err)
	}

	if settings.EncryptionEnabled {
		t.Error("EncryptionEnabled should be false after SetEncryption(false)")
	}
}

func TestSettings_Get_ThreadSafe(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxSizeMB = 750

	copy := settings.Get()

	// Verify copy has same values
	if copy.MaxSizeMB != 750 {
		t.Errorf("Get().MaxSizeMB = %d, want 750", copy.MaxSizeMB)
	}

	// Modify copy shouldn't affect original
	copy.MaxSizeMB = 1000
	if settings.MaxSizeMB != 750 {
		t.Error("Modifying copy should not affect original")
	}
}

func TestSettings_ToConfig(t *testing.T) {
	settings := DefaultSettings()
	settings.MaxSizeMB = 600
	settings.TTLDays = 45
	settings.SyncIntervalMinutes = 10

	config := settings.ToConfig("/test/path")

	if config.BasePath != "/test/path" {
		t.Errorf("ToConfig().BasePath = %q, want %q", config.BasePath, "/test/path")
	}
	if config.MaxSizeMB != 600 {
		t.Errorf("ToConfig().MaxSizeMB = %d, want 600", config.MaxSizeMB)
	}
	if config.TTLDays != 45 {
		t.Errorf("ToConfig().TTLDays = %d, want 45", config.TTLDays)
	}
	if config.SyncIntervalMinutes != 10 {
		t.Errorf("ToConfig().SyncIntervalMinutes = %d, want 10", config.SyncIntervalMinutes)
	}
}

func TestSettings_ToEncryptionConfig(t *testing.T) {
	settings := DefaultSettings()
	settings.EncryptionEnabled = true

	encConfig := settings.ToEncryptionConfig()

	if !encConfig.Enabled {
		t.Error("ToEncryptionConfig().Enabled should be true")
	}

	settings.EncryptionEnabled = false
	encConfig = settings.ToEncryptionConfig()

	if encConfig.Enabled {
		t.Error("ToEncryptionConfig().Enabled should be false")
	}
}

func TestSettings_Reset(t *testing.T) {
	tmpDir := t.TempDir()

	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	// Modify settings
	settings.Enabled = false
	settings.MaxSizeMB = 1000
	settings.Theme = "light"

	// Reset to defaults
	if err := settings.Reset(); err != nil {
		t.Fatalf("Reset() error: %v", err)
	}

	defaults := DefaultSettings()
	if settings.Enabled != defaults.Enabled {
		t.Errorf("After Reset(), Enabled = %v, want %v", settings.Enabled, defaults.Enabled)
	}
	if settings.MaxSizeMB != defaults.MaxSizeMB {
		t.Errorf("After Reset(), MaxSizeMB = %d, want %d", settings.MaxSizeMB, defaults.MaxSizeMB)
	}
	if settings.Theme != defaults.Theme {
		t.Errorf("After Reset(), Theme = %q, want %q", settings.Theme, defaults.Theme)
	}
}

func TestSettings_Update(t *testing.T) {
	tmpDir := t.TempDir()

	settings, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() error: %v", err)
	}

	// Use Update to modify multiple fields
	err = settings.Update(func(s *Settings) {
		s.MaxSizeMB = 750
		s.TTLDays = 60
		s.Theme = "light"
	})

	if err != nil {
		t.Fatalf("Update() error: %v", err)
	}

	if settings.MaxSizeMB != 750 {
		t.Errorf("After Update(), MaxSizeMB = %d, want 750", settings.MaxSizeMB)
	}
	if settings.TTLDays != 60 {
		t.Errorf("After Update(), TTLDays = %d, want 60", settings.TTLDays)
	}
	if settings.Theme != "light" {
		t.Errorf("After Update(), Theme = %q, want 'light'", settings.Theme)
	}

	// Verify settings were saved to file
	reloaded, err := LoadSettings(tmpDir)
	if err != nil {
		t.Fatalf("LoadSettings() after Update() error: %v", err)
	}

	if reloaded.MaxSizeMB != 750 {
		t.Errorf("Reloaded MaxSizeMB = %d, want 750", reloaded.MaxSizeMB)
	}
}

func TestDefaultSettings_Values(t *testing.T) {
	defaults := DefaultSettings()

	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"Enabled", defaults.Enabled, true},
		{"MaxSizeMB", defaults.MaxSizeMB, 500},
		{"TTLDays", defaults.TTLDays, 30},
		{"SyncIntervalMinutes", defaults.SyncIntervalMinutes, 5},
		{"OfflineQueueEnabled", defaults.OfflineQueueEnabled, true},
		{"EncryptionEnabled", defaults.EncryptionEnabled, false},
		{"AttachmentCacheEnabled", defaults.AttachmentCacheEnabled, true},
		{"AttachmentMaxSizeMB", defaults.AttachmentMaxSizeMB, 100},
		{"InitialSyncDays", defaults.InitialSyncDays, 30},
		{"BackgroundSyncEnabled", defaults.BackgroundSyncEnabled, true},
		{"Theme", defaults.Theme, "dark"},
		{"DefaultView", defaults.DefaultView, "email"},
		{"CompactMode", defaults.CompactMode, false},
		{"PreviewPosition", defaults.PreviewPosition, "right"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}
