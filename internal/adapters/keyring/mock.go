package keyring

import (
	"sync"

	"github.com/nylas/cli/internal/domain"
)

// MockSecretStore is a mock implementation of SecretStore for testing.
type MockSecretStore struct {
	mu        sync.RWMutex
	secrets   map[string]string
	available bool

	// Custom functions for testing specific behaviors
	SetFunc    func(key, value string) error
	GetFunc    func(key string) (string, error)
	DeleteFunc func(key string) error
}

// NewMockSecretStore creates a new MockSecretStore.
func NewMockSecretStore() *MockSecretStore {
	return &MockSecretStore{
		secrets:   make(map[string]string),
		available: true,
	}
}

// Set stores a secret value for the given key.
func (m *MockSecretStore) Set(key, value string) error {
	if m.SetFunc != nil {
		return m.SetFunc(key, value)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secrets[key] = value
	return nil
}

// Get retrieves a secret value for the given key.
func (m *MockSecretStore) Get(key string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.secrets[key]
	if !ok {
		return "", domain.ErrSecretNotFound
	}
	return value, nil
}

// Delete removes a secret for the given key.
func (m *MockSecretStore) Delete(key string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(key)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.secrets, key)
	return nil
}

// IsAvailable returns whether the mock is available.
func (m *MockSecretStore) IsAvailable() bool {
	return m.available
}

// Name returns the name of the secret store backend.
func (m *MockSecretStore) Name() string {
	return "mock"
}

// SetAvailable sets whether the mock is available.
func (m *MockSecretStore) SetAvailable(available bool) {
	m.available = available
}

// Reset clears all secrets.
func (m *MockSecretStore) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	clear(m.secrets)
}

// GetAll returns all stored secrets.
func (m *MockSecretStore) GetAll() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range m.secrets {
		result[k] = v
	}
	return result
}
