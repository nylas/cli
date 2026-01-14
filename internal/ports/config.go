package ports

import "github.com/nylas/cli/internal/domain"

// ConfigStore defines the interface for configuration storage.
type ConfigStore interface {
	// Load loads the configuration from storage.
	Load() (*domain.Config, error)

	// Save saves the configuration to storage.
	Save(config *domain.Config) error

	// Path returns the path to the config file.
	Path() string

	// Exists returns true if the config file exists.
	Exists() bool
}
