package config

import (
	"github.com/nylas/cli/internal/domain"
)

// MockConfigStore is a mock implementation of ConfigStore for testing.
type MockConfigStore struct {
	config *domain.Config
	path   string
	exists bool
}

// NewMockConfigStore creates a new MockConfigStore.
func NewMockConfigStore() *MockConfigStore {
	return &MockConfigStore{
		config: domain.DefaultConfig(),
		path:   "/mock/config.yaml",
		exists: true,
	}
}

// Load loads the configuration.
func (m *MockConfigStore) Load() (*domain.Config, error) {
	if m.config == nil {
		return domain.DefaultConfig(), nil
	}
	return m.config, nil
}

// Save saves the configuration.
func (m *MockConfigStore) Save(config *domain.Config) error {
	m.config = config
	m.exists = true
	return nil
}

// Path returns the mock path.
func (m *MockConfigStore) Path() string {
	return m.path
}

// Exists returns whether the config exists.
func (m *MockConfigStore) Exists() bool {
	return m.exists
}

// SetConfig sets the mock config.
func (m *MockConfigStore) SetConfig(config *domain.Config) {
	m.config = config
	m.exists = true
}

// SetExists sets whether the config file exists.
func (m *MockConfigStore) SetExists(exists bool) {
	m.exists = exists
}
