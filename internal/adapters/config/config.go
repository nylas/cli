// Package config provides configuration file management.
package config

import (
	"os"
	"path/filepath"

	"github.com/nylas/cli/internal/domain"
	"gopkg.in/yaml.v3"
)

// FileStore implements ConfigStore using a YAML file.
type FileStore struct {
	path string
}

// NewFileStore creates a new FileStore.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// NewDefaultFileStore creates a FileStore at the default location.
func NewDefaultFileStore() *FileStore {
	return NewFileStore(DefaultConfigPath())
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "nylas", "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nylas", "config.yaml")
}

// DefaultConfigDir returns the default config directory.
func DefaultConfigDir() string {
	return filepath.Dir(DefaultConfigPath())
}

// Load loads the configuration from the file.
func (f *FileStore) Load() (*domain.Config, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.DefaultConfig(), nil
		}
		return nil, err
	}

	var config domain.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Apply defaults for missing fields
	if config.Region == "" {
		config.Region = "us"
	}
	if config.CallbackPort == 0 {
		config.CallbackPort = 8080
	}

	return &config, nil
}

// Save saves the configuration to the file.
func (f *FileStore) Save(config *domain.Config) error {
	// Ensure directory exists
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(f.path, data, 0600)
}

// Path returns the path to the config file.
func (f *FileStore) Path() string {
	return f.path
}

// Exists returns true if the config file exists.
func (f *FileStore) Exists() bool {
	_, err := os.Stat(f.path)
	return err == nil
}
