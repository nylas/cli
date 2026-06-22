// Package templates provides email template storage functionality.
package templates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// ErrTemplateNotFound is returned when a template is not found.
var ErrTemplateNotFound = errors.New("template not found")

// FileStore stores email templates in a JSON file.
type FileStore struct {
	path string
	mu   sync.RWMutex
}

// templateFile represents the JSON file structure.
type templateFile struct {
	Templates []domain.EmailTemplate `json:"templates"`
}

// NewFileStore creates a new FileStore at the specified path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// NewDefaultFileStore creates a FileStore at the default location.
// The default location is ~/.config/nylas/templates.json
func NewDefaultFileStore() *FileStore {
	return NewFileStore(DefaultPath())
}

// DefaultPath returns the default templates file path.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "nylas", "templates.json")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nylas", "templates.json")
}

// List returns all templates, optionally filtered by category.
func (f *FileStore) List(_ context.Context, category string) ([]domain.EmailTemplate, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			return []domain.EmailTemplate{}, nil
		}
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	if category == "" {
		return data.Templates, nil
	}

	// Filter by category (case-insensitive)
	category = strings.ToLower(category)
	var filtered []domain.EmailTemplate
	for _, t := range data.Templates {
		if strings.ToLower(t.Category) == category {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

// Get retrieves a template by its ID.
func (f *FileStore) Get(_ context.Context, id string) (*domain.EmailTemplate, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	data, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	for i := range data.Templates {
		if data.Templates[i].ID == id {
			return &data.Templates[i], nil
		}
	}
	return nil, ErrTemplateNotFound
}

// Create creates a new template and returns it with generated ID.
func (f *FileStore) Create(_ context.Context, t *domain.EmailTemplate) (*domain.EmailTemplate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			data = &templateFile{Templates: []domain.EmailTemplate{}}
		} else {
			return nil, fmt.Errorf("failed to load templates: %w", err)
		}
	}

	// Generate ID and set timestamps
	now := time.Now()
	t.ID = fmt.Sprintf("tpl_%d", now.UnixNano()) // Nanosecond precision for unique IDs
	t.CreatedAt = now
	t.UpdatedAt = now

	// Extract variables from subject and body
	t.Variables = extractVariables(t.Subject, t.HTMLBody)

	data.Templates = append(data.Templates, *t)

	if err := f.save(data); err != nil {
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	return t, nil
}

// Update updates an existing template.
func (f *FileStore) Update(_ context.Context, t *domain.EmailTemplate) (*domain.EmailTemplate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	for i := range data.Templates {
		if data.Templates[i].ID == t.ID {
			// Preserve creation time and ID
			t.CreatedAt = data.Templates[i].CreatedAt
			t.UpdatedAt = time.Now()

			// Re-extract variables from subject and body
			t.Variables = extractVariables(t.Subject, t.HTMLBody)

			data.Templates[i] = *t

			if err := f.save(data); err != nil {
				return nil, fmt.Errorf("failed to save template: %w", err)
			}
			return t, nil
		}
	}
	return nil, ErrTemplateNotFound
}

// Delete removes a template by its ID.
func (f *FileStore) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			return ErrTemplateNotFound
		}
		return fmt.Errorf("failed to load templates: %w", err)
	}

	for i := range data.Templates {
		if data.Templates[i].ID == id {
			// Remove template at index i
			data.Templates = append(data.Templates[:i], data.Templates[i+1:]...)

			if err := f.save(data); err != nil {
				return fmt.Errorf("failed to save templates: %w", err)
			}
			return nil
		}
	}
	return ErrTemplateNotFound
}

// IncrementUsage increments the usage count for a template.
func (f *FileStore) IncrementUsage(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := f.load()
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	for i := range data.Templates {
		if data.Templates[i].ID == id {
			data.Templates[i].UsageCount++
			data.Templates[i].UpdatedAt = time.Now()

			if err := f.save(data); err != nil {
				return fmt.Errorf("failed to save templates: %w", err)
			}
			return nil
		}
	}
	return ErrTemplateNotFound
}

// Path returns the path to the templates file.
func (f *FileStore) Path() string {
	return f.path
}

// load reads the templates file.
func (f *FileStore) load() (*templateFile, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return nil, err
	}

	var tf templateFile
	if err := json.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("failed to parse templates file: %w", err)
	}
	return &tf, nil
}

// save writes the templates file.
func (f *FileStore) save(data *templateFile) error {
	// Ensure directory exists
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal templates: %w", err)
	}

	return os.WriteFile(f.path, jsonData, 0600)
}

// variableRegex matches {{variable}} patterns.
var variableRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// extractVariables extracts unique variable names from template text.
func extractVariables(texts ...string) []string {
	seen := make(map[string]bool)
	var variables []string

	for _, text := range texts {
		matches := variableRegex.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) > 1 {
				name := strings.TrimSpace(match[1])
				if !seen[name] {
					seen[name] = true
					variables = append(variables, name)
				}
			}
		}
	}
	return variables
}

// ExpandVariables replaces {{var}} placeholders with provided values.
// Returns the expanded text and a list of any missing variables.
func ExpandVariables(text string, vars map[string]string) (string, []string) {
	var missing []string

	result := variableRegex.ReplaceAllStringFunc(text, func(match string) string {
		// Extract variable name (without {{ }})
		name := strings.TrimSpace(match[2 : len(match)-2])
		if val, ok := vars[name]; ok {
			return val
		}
		missing = append(missing, name)
		return match // Keep original if not found
	})

	return result, missing
}
