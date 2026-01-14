package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestNewFileStore(t *testing.T) {
	path := "/tmp/test-config.yaml"
	store := NewFileStore(path)
	if store == nil {
		t.Error("NewFileStore() returned nil")
	}
	if store.Path() != path {
		t.Errorf("Path() = %q, want %q", store.Path(), path)
	}
}

func TestNewDefaultFileStore(t *testing.T) {
	store := NewDefaultFileStore()
	if store == nil {
		t.Error("NewDefaultFileStore() returned nil")
	}
	expectedPath := DefaultConfigPath()
	if store.Path() != expectedPath {
		t.Errorf("Path() = %q, want %q", store.Path(), expectedPath)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	// Save original XDG_CONFIG_HOME
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if originalXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	tests := []struct {
		name      string
		xdgConfig string
		wantPath  func() string
	}{
		{
			name:      "with XDG_CONFIG_HOME",
			xdgConfig: "/custom/config",
			wantPath: func() string {
				return filepath.Join("/custom/config", "nylas", "config.yaml")
			},
		},
		{
			name:      "without XDG_CONFIG_HOME",
			xdgConfig: "",
			wantPath: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, ".config", "nylas", "config.yaml")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.xdgConfig != "" {
				_ = os.Setenv("XDG_CONFIG_HOME", tt.xdgConfig)
			} else {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			}

			got := DefaultConfigPath()
			want := tt.wantPath()
			if got != want {
				t.Errorf("DefaultConfigPath() = %q, want %q", got, want)
			}
		})
	}
}

func TestDefaultConfigDir(t *testing.T) {
	dir := DefaultConfigDir()
	expectedDir := filepath.Dir(DefaultConfigPath())
	if dir != expectedDir {
		t.Errorf("DefaultConfigDir() = %q, want %q", dir, expectedDir)
	}
}

func TestFileStore_LoadSaveRoundTrip(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	store := NewFileStore(configPath)

	// Test config
	config := &domain.Config{
		Region:       "eu",
		CallbackPort: 9000,
	}

	// Save config
	if err := store.Save(config); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load config
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify
	if loaded.Region != config.Region {
		t.Errorf("Region = %q, want %q", loaded.Region, config.Region)
	}
	if loaded.CallbackPort != config.CallbackPort {
		t.Errorf("CallbackPort = %d, want %d", loaded.CallbackPort, config.CallbackPort)
	}
}

func TestFileStore_LoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yaml")

	store := NewFileStore(configPath)
	config, err := store.Load()

	if err != nil {
		t.Fatalf("Load() error = %v, want nil (should return default)", err)
	}

	// Should return default config
	defaultConfig := domain.DefaultConfig()
	if config.Region != defaultConfig.Region {
		t.Errorf("Region = %q, want %q (default)", config.Region, defaultConfig.Region)
	}
	if config.CallbackPort != defaultConfig.CallbackPort {
		t.Errorf("CallbackPort = %d, want %d (default)", config.CallbackPort, defaultConfig.CallbackPort)
	}
}

func TestFileStore_LoadAppliesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.yaml")

	// Write partial config (missing optional fields)
	partialYAML := []byte(`region: "eu"`)
	if err := os.WriteFile(configPath, partialYAML, 0600); err != nil {
		t.Fatalf("Failed to write partial config: %v", err)
	}

	store := NewFileStore(configPath)
	config, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify defaults are applied for missing fields
	if config.CallbackPort != 8080 {
		t.Errorf("CallbackPort = %d, want %d (default)", config.CallbackPort, 8080)
	}
}

func TestFileStore_LoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	invalidYAML := []byte(`invalid: yaml: content: [`)
	if err := os.WriteFile(configPath, invalidYAML, 0600); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	store := NewFileStore(configPath)
	_, err := store.Load()
	if err == nil {
		t.Error("Load() error = nil, want error for invalid YAML")
	}
}

func TestFileStore_SaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config.yaml")

	store := NewFileStore(configPath)
	config := domain.DefaultConfig()

	if err := store.Save(config); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if !store.Exists() {
		t.Error("Exists() = false, want true after Save()")
	}

	// Verify directory was created
	dir := filepath.Dir(configPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("Directory not created: %s", dir)
	}
}

func TestFileStore_Exists(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		createFile bool
		want       bool
	}{
		{
			name:       "file exists",
			createFile: true,
			want:       true,
		},
		{
			name:       "file does not exist",
			createFile: false,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.name+".yaml")
			store := NewFileStore(configPath)

			if tt.createFile {
				if err := store.Save(domain.DefaultConfig()); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
			}

			got := store.Exists()
			if got != tt.want {
				t.Errorf("Exists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileStore_Path(t *testing.T) {
	path := "/tmp/test-path.yaml"
	store := NewFileStore(path)
	if got := store.Path(); got != path {
		t.Errorf("Path() = %q, want %q", got, path)
	}
}

func TestNewMockConfigStore(t *testing.T) {
	store := NewMockConfigStore()
	if store == nil {
		t.Fatal("NewMockConfigStore() returned nil")
	}
	if store.Path() != "/mock/config.yaml" {
		t.Errorf("Path() = %q, want %q", store.Path(), "/mock/config.yaml")
	}
	if !store.Exists() {
		t.Error("Exists() = false, want true for new mock store")
	}
}

func TestMockConfigStore_LoadSave(t *testing.T) {
	store := NewMockConfigStore()

	// Load default config
	config, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config == nil {
		t.Fatal("Load() returned nil config")
	}

	// Modify and save
	config.Region = "eu"
	config.CallbackPort = 9000
	if err := store.Save(config); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load again and verify
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}
	if loaded.Region != "eu" {
		t.Errorf("Region = %q, want %q", loaded.Region, "eu")
	}
	if loaded.CallbackPort != 9000 {
		t.Errorf("CallbackPort = %d, want %d", loaded.CallbackPort, 9000)
	}
}

func TestMockConfigStore_LoadWithNilConfig(t *testing.T) {
	store := NewMockConfigStore()
	store.config = nil

	config, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if config == nil {
		t.Fatal("Load() returned nil, expected default config")
	}

	// Should return default config
	defaultConfig := domain.DefaultConfig()
	if config.Region != defaultConfig.Region {
		t.Errorf("Region = %q, want %q (default)", config.Region, defaultConfig.Region)
	}
}

func TestMockConfigStore_SetConfig(t *testing.T) {
	store := NewMockConfigStore()

	customConfig := &domain.Config{
		Region:       "eu",
		CallbackPort: 9000,
	}

	store.SetConfig(customConfig)

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Region != "eu" {
		t.Errorf("Region = %q, want %q", loaded.Region, "eu")
	}
	if loaded.CallbackPort != 9000 {
		t.Errorf("CallbackPort = %d, want %d", loaded.CallbackPort, 9000)
	}
	if !store.Exists() {
		t.Error("Exists() = false, want true after SetConfig")
	}
}

func TestMockConfigStore_SetExists(t *testing.T) {
	store := NewMockConfigStore()

	tests := []struct {
		name   string
		exists bool
	}{
		{"set to true", true},
		{"set to false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store.SetExists(tt.exists)
			if got := store.Exists(); got != tt.exists {
				t.Errorf("Exists() = %v, want %v", got, tt.exists)
			}
		})
	}
}

func TestMockConfigStore_Path(t *testing.T) {
	store := NewMockConfigStore()
	if got := store.Path(); got != "/mock/config.yaml" {
		t.Errorf("Path() = %q, want %q", got, "/mock/config.yaml")
	}
}
