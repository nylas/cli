package grantcache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nylas/cli/internal/domain"
)

const (
	fileVersion  = 1
	lockWait     = 10 * time.Second
	staleLockAge = 2 * time.Minute
)

// Store implements ports.GrantStore using a plaintext JSON file for
// non-secret grant metadata and local default-grant preference.
type Store struct {
	path string
	mu   sync.RWMutex
}

type fileShape struct {
	Version      int                `json:"version"`
	DefaultGrant string             `json:"default_grant,omitempty"`
	Grants       []domain.GrantInfo `json:"grants"`
}

// New creates a file-backed grant metadata store.
func New(path string) *Store {
	return &Store{path: path}
}

// SaveGrant saves or replaces one grant in local metadata.
func (s *Store) SaveGrant(info domain.GrantInfo) error {
	if info.ID == "" || info.Email == "" {
		return domain.ErrInvalidInput
	}
	return s.mutate(func(shape *fileShape) error {
		for i, grant := range shape.Grants {
			if grant.ID == info.ID {
				shape.Grants[i] = info
				return nil
			}
		}
		shape.Grants = append(shape.Grants, info)
		return nil
	})
}

// ReplaceGrants replaces locally cached grant metadata after a successful
// live API listing. The default grant is preserved only if it still exists.
func (s *Store) ReplaceGrants(grants []domain.GrantInfo) error {
	for _, grant := range grants {
		if grant.ID == "" || grant.Email == "" {
			return domain.ErrInvalidInput
		}
	}
	return s.mutate(func(shape *fileShape) error {
		defaultGrant := shape.DefaultGrant
		shape.Grants = append([]domain.GrantInfo(nil), grants...)
		if defaultGrant != "" && !containsGrantID(shape.Grants, defaultGrant) {
			shape.DefaultGrant = ""
		}
		return nil
	})
}

// GetGrant retrieves grant info by ID.
func (s *Store) GetGrant(grantID string) (*domain.GrantInfo, error) {
	grants, err := s.ListGrants()
	if err != nil {
		return nil, err
	}
	for _, grant := range grants {
		if grant.ID == grantID {
			out := grant
			return &out, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

// GetGrantByEmail retrieves grant info by email.
func (s *Store) GetGrantByEmail(email string) (*domain.GrantInfo, error) {
	grants, err := s.ListGrants()
	if err != nil {
		return nil, err
	}
	for _, grant := range grants {
		if grant.Email == email {
			out := grant
			return &out, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

// ListGrants returns locally cached grant metadata.
func (s *Store) ListGrants() ([]domain.GrantInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	shape, err := s.read()
	if err != nil {
		return nil, err
	}
	return append([]domain.GrantInfo(nil), shape.Grants...), nil
}

// DeleteGrant removes one grant from local metadata.
func (s *Store) DeleteGrant(grantID string) error {
	return s.mutate(func(shape *fileShape) error {
		grants := shape.Grants[:0]
		for _, grant := range shape.Grants {
			if grant.ID != grantID {
				grants = append(grants, grant)
			}
		}
		shape.Grants = grants
		if shape.DefaultGrant == grantID {
			shape.DefaultGrant = ""
		}
		return nil
	})
}

// SetDefaultGrant stores the local default-grant preference.
func (s *Store) SetDefaultGrant(grantID string) error {
	return s.mutate(func(shape *fileShape) error {
		shape.DefaultGrant = grantID
		return nil
	})
}

// GetDefaultGrant returns the local default-grant preference.
func (s *Store) GetDefaultGrant() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	shape, err := s.read()
	if err != nil {
		return "", err
	}
	if shape.DefaultGrant == "" {
		return "", domain.ErrNoDefaultGrant
	}
	return shape.DefaultGrant, nil
}

// ClearGrants removes all local grant metadata and default preference.
func (s *Store) ClearGrants() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.withFileLock(func() error {
		if err := os.Remove(s.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	})
}

func (s *Store) mutate(fn func(*fileShape) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.withFileLock(func() error {
		shape, err := s.read()
		if err != nil {
			return err
		}
		if err := fn(shape); err != nil {
			return err
		}
		return s.write(shape)
	})
}

func (s *Store) read() (*fileShape, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &fileShape{Version: fileVersion}, nil
		}
		return nil, err
	}

	var shape fileShape
	if err := json.Unmarshal(data, &shape); err != nil {
		return &fileShape{Version: fileVersion}, nil
	}
	if shape.Version == 0 {
		shape.Version = fileVersion
	}
	return &shape, nil
}

func (s *Store) write(shape *fileShape) error {
	shape.Version = fileVersion

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(shape, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".grants-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}

func (s *Store) withFileLock(fn func() error) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	lockPath := s.path + ".lock"
	deadline := time.Now().Add(lockWait)
	for {
		err := os.Mkdir(lockPath, 0o700)
		if err == nil {
			defer func() { _ = os.Remove(lockPath) }()
			return fn()
		}
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		if isStaleLock(lockPath) {
			_ = os.RemoveAll(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for grant cache lock: %s", lockPath)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func isStaleLock(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) > staleLockAge
}

func containsGrantID(grants []domain.GrantInfo, grantID string) bool {
	for _, grant := range grants {
		if grant.ID == grantID {
			return true
		}
	}
	return false
}
