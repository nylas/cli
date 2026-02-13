package chat

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Conversation represents a chat conversation with full message history.
type Conversation struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Agent       string    `json:"agent"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Messages    []Message `json:"messages"`
	Summary     string    `json:"summary,omitempty"`
	MsgCount    int       `json:"message_count"`
	CompactedAt time.Time `json:"compacted_at,omitempty"`
}

// Message represents a single message in a conversation.
type Message struct {
	Role      string    `json:"role"` // user, assistant, tool_call, tool_result
	Content   string    `json:"content"`
	Name      string    `json:"name,omitempty"` // tool name for tool_call/tool_result
	Timestamp time.Time `json:"timestamp"`
}

// ConversationSummary is a lightweight view for listing conversations.
type ConversationSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Agent     string    `json:"agent"`
	UpdatedAt time.Time `json:"updated_at"`
	Preview   string    `json:"preview"`
	MsgCount  int       `json:"message_count"`
}

// MemoryStore manages conversation persistence as JSON files on disk.
type MemoryStore struct {
	basePath string
	mu       sync.RWMutex
}

// NewMemoryStore creates a new MemoryStore at the given base path.
func NewMemoryStore(basePath string) (*MemoryStore, error) {
	if err := os.MkdirAll(basePath, 0750); err != nil {
		return nil, fmt.Errorf("create memory directory: %w", err)
	}
	return &MemoryStore{basePath: basePath}, nil
}

// List returns summaries of all conversations, sorted by most recent first.
func (m *MemoryStore) List() ([]ConversationSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read conversations dir: %w", err)
	}

	var summaries []ConversationSummary
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		conv, err := m.readFile(filepath.Join(m.basePath, entry.Name()))
		if err != nil {
			continue // skip corrupt files
		}

		preview := ""
		for i := len(conv.Messages) - 1; i >= 0; i-- {
			if conv.Messages[i].Role == "assistant" || conv.Messages[i].Role == "user" {
				preview = conv.Messages[i].Content
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				break
			}
		}

		summaries = append(summaries, ConversationSummary{
			ID:        conv.ID,
			Title:     conv.Title,
			Agent:     conv.Agent,
			UpdatedAt: conv.UpdatedAt,
			Preview:   preview,
			MsgCount:  conv.MsgCount,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].UpdatedAt.After(summaries[j].UpdatedAt)
	})

	return summaries, nil
}

// Get retrieves a conversation by ID.
func (m *MemoryStore) Get(id string) (*Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.readFile(m.filePath(id))
}

// Create creates a new conversation for the given agent.
func (m *MemoryStore) Create(agent string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, err := generateID()
	if err != nil {
		return nil, fmt.Errorf("generate conversation ID: %w", err)
	}

	conv := &Conversation{
		ID:        id,
		Title:     "New conversation",
		Agent:     agent,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Messages:  []Message{},
	}

	if err := m.writeFile(conv); err != nil {
		return nil, err
	}
	return conv, nil
}

// Delete removes a conversation by ID.
func (m *MemoryStore) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := m.filePath(id)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("conversation not found: %s", id)
		}
		return fmt.Errorf("delete conversation: %w", err)
	}
	return nil
}

// AddMessage appends a message to a conversation.
func (m *MemoryStore) AddMessage(id string, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, err := m.readFile(m.filePath(id))
	if err != nil {
		return err
	}

	msg.Timestamp = time.Now().UTC()
	conv.Messages = append(conv.Messages, msg)
	conv.MsgCount = len(conv.Messages)
	conv.UpdatedAt = time.Now().UTC()

	return m.writeFile(conv)
}

// UpdateTitle updates the title of a conversation.
func (m *MemoryStore) UpdateTitle(id string, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, err := m.readFile(m.filePath(id))
	if err != nil {
		return err
	}

	conv.Title = title
	conv.UpdatedAt = time.Now().UTC()
	return m.writeFile(conv)
}

// UpdateSummary updates the summary and trims compacted messages.
func (m *MemoryStore) UpdateSummary(id string, summary string, keepFrom int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conv, err := m.readFile(m.filePath(id))
	if err != nil {
		return err
	}

	conv.Summary = summary
	conv.CompactedAt = time.Now().UTC()
	if keepFrom > 0 && keepFrom < len(conv.Messages) {
		conv.Messages = conv.Messages[keepFrom:]
	}
	conv.UpdatedAt = time.Now().UTC()

	return m.writeFile(conv)
}

func (m *MemoryStore) filePath(id string) string {
	return filepath.Join(m.basePath, id+".json")
}

func (m *MemoryStore) readFile(path string) (*Conversation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read conversation: %w", err)
	}

	var conv Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, fmt.Errorf("parse conversation: %w", err)
	}
	return &conv, nil
}

func (m *MemoryStore) writeFile(conv *Conversation) error {
	data, err := json.MarshalIndent(conv, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal conversation: %w", err)
	}
	return os.WriteFile(m.filePath(conv.ID), data, 0600)
}

func generateID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "conv_" + hex.EncodeToString(b), nil
}
