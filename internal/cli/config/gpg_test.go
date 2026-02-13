package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/domain"
)

func TestGPGConfig_SetAndGet(t *testing.T) {
	// Create temp config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	store := config.NewFileStore(configPath)

	tests := []struct {
		name     string
		key      string
		value    string
		wantErr  bool
		validate func(t *testing.T, cfg *domain.Config)
	}{
		{
			name:  "set gpg.default_key",
			key:   "gpg.default_key",
			value: "601FEE9B1D60185F",
			validate: func(t *testing.T, cfg *domain.Config) {
				if cfg.GPG == nil {
					t.Fatal("GPG config is nil")
				}
				if cfg.GPG.DefaultKey != "601FEE9B1D60185F" {
					t.Errorf("expected default_key=601FEE9B1D60185F, got %s", cfg.GPG.DefaultKey)
				}
			},
		},
		{
			name:  "set gpg.auto_sign to true",
			key:   "gpg.auto_sign",
			value: "true",
			validate: func(t *testing.T, cfg *domain.Config) {
				if cfg.GPG == nil {
					t.Fatal("GPG config is nil")
				}
				if !cfg.GPG.AutoSign {
					t.Error("expected auto_sign=true, got false")
				}
			},
		},
		{
			name:  "set gpg.auto_sign to false",
			key:   "gpg.auto_sign",
			value: "false",
			validate: func(t *testing.T, cfg *domain.Config) {
				if cfg.GPG == nil {
					t.Fatal("GPG config is nil")
				}
				if cfg.GPG.AutoSign {
					t.Error("expected auto_sign=false, got true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load or create config
			cfg, err := store.Load()
			if err != nil {
				t.Fatalf("failed to load config: %v", err)
			}

			// Set value
			err = setConfigValue(cfg, tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("setConfigValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Save config
			if err := store.Save(cfg); err != nil {
				t.Fatalf("failed to save config: %v", err)
			}

			// Reload config
			cfg, err = store.Load()
			if err != nil {
				t.Fatalf("failed to reload config: %v", err)
			}

			// Validate
			if tt.validate != nil {
				tt.validate(t, cfg)
			}

			// Test get
			value, err := getConfigValue(cfg, tt.key)
			if err != nil {
				t.Errorf("getConfigValue() error = %v", err)
				return
			}

			if value != tt.value {
				t.Errorf("getConfigValue() = %v, want %v", value, tt.value)
			}
		})
	}
}

func TestGPGConfig_Persistence(t *testing.T) {
	// Create temp config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	store := config.NewFileStore(configPath)

	// Initial config
	cfg := domain.DefaultConfig()
	cfg.GPG = &domain.GPGConfig{
		DefaultKey: "ABC123",
		AutoSign:   true,
	}

	// Save
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reload
	loadedCfg, err := store.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify
	if loadedCfg.GPG == nil {
		t.Fatal("GPG config is nil after reload")
	}
	if loadedCfg.GPG.DefaultKey != "ABC123" {
		t.Errorf("expected default_key=ABC123, got %s", loadedCfg.GPG.DefaultKey)
	}
	if !loadedCfg.GPG.AutoSign {
		t.Error("expected auto_sign=true, got false")
	}
}

func TestGPGConfig_FileFormat(t *testing.T) {
	// Create temp config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	store := config.NewFileStore(configPath)

	// Create config with GPG settings
	cfg := domain.DefaultConfig()
	cfg.GPG = &domain.GPGConfig{
		DefaultKey: "601FEE9B1D60185F",
		AutoSign:   true,
	}

	// Save
	if err := store.Save(cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Read raw file
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(data)

	// Verify YAML format
	expectedLines := []string{
		"gpg:",
		"  default_key: 601FEE9B1D60185F",
		"  auto_sign: true",
	}

	for _, line := range expectedLines {
		if !strings.Contains(content, line) {
			t.Errorf("config file missing expected line: %s\nGot:\n%s", line, content)
		}
	}
}
