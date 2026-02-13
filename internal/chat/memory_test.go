package chat

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemoryStore(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) string
		wantError bool
	}{
		{
			name: "creates directory if not exists",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "conversations")
			},
			wantError: false,
		},
		{
			name: "works with existing directory",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "existing")
				require.NoError(t, os.MkdirAll(dir, 0750))
				return dir
			},
			wantError: false,
		},
		{
			name: "creates nested directories",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "level1", "level2", "conversations")
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			store, err := NewMemoryStore(path)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, store)
			} else {
				require.NoError(t, err)
				require.NotNil(t, store)
				assert.Equal(t, path, store.basePath)

				// Verify directory was created
				info, err := os.Stat(path)
				require.NoError(t, err)
				assert.True(t, info.IsDir())
			}
		})
	}
}

func TestMemoryStore_Create(t *testing.T) {
	store := setupMemoryStore(t)

	tests := []struct {
		name      string
		agent     string
		wantError bool
	}{
		{
			name:      "creates conversation with claude agent",
			agent:     "claude",
			wantError: false,
		},
		{
			name:      "creates conversation with ollama agent",
			agent:     "ollama",
			wantError: false,
		},
		{
			name:      "creates conversation with empty agent",
			agent:     "",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := store.Create(tt.agent)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, conv)

				// Verify fields
				assert.NotEmpty(t, conv.ID, "ID should be generated")
				assert.True(t, len(conv.ID) > len("conv_"), "ID should have conv_ prefix and random suffix")
				assert.Equal(t, "New conversation", conv.Title)
				assert.Equal(t, tt.agent, conv.Agent)
				assert.False(t, conv.CreatedAt.IsZero(), "CreatedAt should be set")
				assert.False(t, conv.UpdatedAt.IsZero(), "UpdatedAt should be set")
				assert.Empty(t, conv.Messages, "Messages should be empty")
				assert.Equal(t, 0, conv.MsgCount)
				assert.Empty(t, conv.Summary)

				// Verify file was created
				filePath := filepath.Join(store.basePath, conv.ID+".json")
				_, err := os.Stat(filePath)
				require.NoError(t, err, "conversation file should exist")
			}
		})
	}
}

func TestMemoryStore_Get(t *testing.T) {
	store := setupMemoryStore(t)

	// Create a conversation
	created, err := store.Create("claude")
	require.NoError(t, err)

	tests := []struct {
		name      string
		id        string
		wantError bool
	}{
		{
			name:      "retrieves existing conversation",
			id:        created.ID,
			wantError: false,
		},
		{
			name:      "returns error for non-existent conversation",
			id:        "conv_nonexistent",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv, err := store.Get(tt.id)

			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, conv)
			} else {
				require.NoError(t, err)
				require.NotNil(t, conv)
				assert.Equal(t, created.ID, conv.ID)
				assert.Equal(t, created.Title, conv.Title)
				assert.Equal(t, created.Agent, conv.Agent)
			}
		})
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := setupMemoryStore(t)

	t.Run("returns empty list when no conversations", func(t *testing.T) {
		summaries, err := store.List()
		require.NoError(t, err)
		assert.Empty(t, summaries)
	})

	t.Run("lists conversations sorted by updated_at", func(t *testing.T) {
		// Create multiple conversations with different update times
		conv1, err := store.Create("claude")
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)

		conv2, err := store.Create("ollama")
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)

		conv3, err := store.Create("codex")
		require.NoError(t, err)

		// Update conv1 to make it most recent
		time.Sleep(10 * time.Millisecond)
		err = store.AddMessage(conv1.ID, Message{
			Role:    "user",
			Content: "Latest message",
		})
		require.NoError(t, err)

		// List and verify order (most recent first)
		summaries, err := store.List()
		require.NoError(t, err)
		require.Len(t, summaries, 3)

		// conv1 should be first (most recently updated)
		assert.Equal(t, conv1.ID, summaries[0].ID)
		assert.Equal(t, "claude", summaries[0].Agent)

		// conv3 should be second
		assert.Equal(t, conv3.ID, summaries[1].ID)

		// conv2 should be last
		assert.Equal(t, conv2.ID, summaries[2].ID)
	})

	t.Run("includes preview from last user or assistant message", func(t *testing.T) {
		store := setupMemoryStore(t)
		conv, err := store.Create("claude")
		require.NoError(t, err)

		// Add messages
		err = store.AddMessage(conv.ID, Message{Role: "user", Content: "Hello"})
		require.NoError(t, err)
		err = store.AddMessage(conv.ID, Message{Role: "assistant", Content: "Hi there! How can I help you today?"})
		require.NoError(t, err)

		summaries, err := store.List()
		require.NoError(t, err)
		require.Len(t, summaries, 1)

		assert.Equal(t, "Hi there! How can I help you today?", summaries[0].Preview)
		assert.Equal(t, 2, summaries[0].MsgCount)
	})

	t.Run("truncates long preview to 100 characters", func(t *testing.T) {
		store := setupMemoryStore(t)
		conv, err := store.Create("claude")
		require.NoError(t, err)

		longMessage := "This is a very long message that should be truncated because it exceeds the maximum preview length of one hundred characters"
		err = store.AddMessage(conv.ID, Message{Role: "user", Content: longMessage})
		require.NoError(t, err)

		summaries, err := store.List()
		require.NoError(t, err)
		require.Len(t, summaries, 1)

		assert.Equal(t, 103, len(summaries[0].Preview)) // 100 chars + "..."
		assert.True(t, len(summaries[0].Preview) <= 103)
		assert.Contains(t, summaries[0].Preview, "...")
	})
}

func TestMemoryStore_Delete(t *testing.T) {
	store := setupMemoryStore(t)

	// Create a conversation
	conv, err := store.Create("claude")
	require.NoError(t, err)

	t.Run("deletes existing conversation", func(t *testing.T) {
		err := store.Delete(conv.ID)
		require.NoError(t, err)

		// Verify file was deleted
		filePath := filepath.Join(store.basePath, conv.ID+".json")
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err), "file should not exist after deletion")

		// Verify Get returns error
		_, err = store.Get(conv.ID)
		assert.Error(t, err)
	})

	t.Run("returns error for non-existent conversation", func(t *testing.T) {
		err := store.Delete("conv_nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conversation not found")
	})
}

func TestMemoryStore_AddMessage(t *testing.T) {
	store := setupMemoryStore(t)
	conv, err := store.Create("claude")
	require.NoError(t, err)

	tests := []struct {
		name    string
		message Message
	}{
		{
			name: "adds user message",
			message: Message{
				Role:    "user",
				Content: "Hello, assistant!",
			},
		},
		{
			name: "adds assistant message",
			message: Message{
				Role:    "assistant",
				Content: "Hello! How can I help?",
			},
		},
		{
			name: "adds tool_call message",
			message: Message{
				Role:    "tool_call",
				Content: `{"name":"list_emails","args":{}}`,
				Name:    "list_emails",
			},
		},
		{
			name: "adds tool_result message",
			message: Message{
				Role:    "tool_result",
				Content: `{"name":"list_emails","data":[]}`,
				Name:    "list_emails",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeUpdate := conv.UpdatedAt
			time.Sleep(10 * time.Millisecond) // Ensure time difference

			err := store.AddMessage(conv.ID, tt.message)
			require.NoError(t, err)

			// Retrieve and verify
			updated, err := store.Get(conv.ID)
			require.NoError(t, err)

			assert.Greater(t, len(updated.Messages), 0)
			lastMsg := updated.Messages[len(updated.Messages)-1]
			assert.Equal(t, tt.message.Role, lastMsg.Role)
			assert.Equal(t, tt.message.Content, lastMsg.Content)
			assert.Equal(t, tt.message.Name, lastMsg.Name)
			assert.False(t, lastMsg.Timestamp.IsZero(), "timestamp should be set")
			assert.Equal(t, len(updated.Messages), updated.MsgCount)
			assert.True(t, updated.UpdatedAt.After(beforeUpdate), "UpdatedAt should be updated")
		})
	}

	t.Run("returns error for non-existent conversation", func(t *testing.T) {
		err := store.AddMessage("conv_nonexistent", Message{
			Role:    "user",
			Content: "Test",
		})
		assert.Error(t, err)
	})
}

func TestMemoryStore_UpdateTitle(t *testing.T) {
	store := setupMemoryStore(t)
	conv, err := store.Create("claude")
	require.NoError(t, err)

	tests := []struct {
		name     string
		newTitle string
	}{
		{
			name:     "updates title",
			newTitle: "Email Discussion",
		},
		{
			name:     "updates to empty title",
			newTitle: "",
		},
		{
			name:     "updates to long title",
			newTitle: "A very long title that should still be stored correctly without any truncation issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeUpdate := conv.UpdatedAt
			time.Sleep(10 * time.Millisecond)

			err := store.UpdateTitle(conv.ID, tt.newTitle)
			require.NoError(t, err)

			// Verify update
			updated, err := store.Get(conv.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.newTitle, updated.Title)
			assert.True(t, updated.UpdatedAt.After(beforeUpdate), "UpdatedAt should be updated")
		})
	}

	t.Run("returns error for non-existent conversation", func(t *testing.T) {
		err := store.UpdateTitle("conv_nonexistent", "New Title")
		assert.Error(t, err)
	})
}

func TestMemoryStore_UpdateSummary(t *testing.T) {
	store := setupMemoryStore(t)
	conv, err := store.Create("claude")
	require.NoError(t, err)

	// Add some messages
	for i := 0; i < 10; i++ {
		err := store.AddMessage(conv.ID, Message{
			Role:    "user",
			Content: "Message " + string(rune('A'+i)),
		})
		require.NoError(t, err)
	}

	tests := []struct {
		name     string
		summary  string
		keepFrom int
	}{
		{
			name:     "updates summary without trimming",
			summary:  "First summary",
			keepFrom: 0,
		},
		{
			name:     "updates summary and trims messages",
			summary:  "Summary after trimming",
			keepFrom: 5,
		},
		{
			name:     "updates summary and keeps all messages",
			summary:  "Keep all",
			keepFrom: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := setupMemoryStore(t)
			conv, err := store.Create("claude")
			require.NoError(t, err)

			// Add messages
			for i := 0; i < 10; i++ {
				err := store.AddMessage(conv.ID, Message{
					Role:    "user",
					Content: "Message " + string(rune('A'+i)),
				})
				require.NoError(t, err)
			}

			beforeUpdate, err := store.Get(conv.ID)
			require.NoError(t, err)
			time.Sleep(10 * time.Millisecond)

			err = store.UpdateSummary(conv.ID, tt.summary, tt.keepFrom)
			require.NoError(t, err)

			// Verify update
			updated, err := store.Get(conv.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.summary, updated.Summary)
			assert.False(t, updated.CompactedAt.IsZero(), "CompactedAt should be set")
			assert.True(t, updated.UpdatedAt.After(beforeUpdate.UpdatedAt), "UpdatedAt should be updated")

			// Verify message trimming
			if tt.keepFrom > 0 && tt.keepFrom < 10 {
				assert.Equal(t, 10-tt.keepFrom, len(updated.Messages), "messages should be trimmed")
				// First remaining message should be at keepFrom index
				assert.Equal(t, "Message "+string(rune('A'+tt.keepFrom)), updated.Messages[0].Content)
			} else if tt.keepFrom <= 0 {
				assert.Equal(t, 10, len(updated.Messages), "all messages should be kept")
			}
		})
	}

	t.Run("returns error for non-existent conversation", func(t *testing.T) {
		err := store.UpdateSummary("conv_nonexistent", "Summary", 0)
		assert.Error(t, err)
	})
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := setupMemoryStore(t)
	conv, err := store.Create("claude")
	require.NoError(t, err)

	// Test concurrent message additions
	t.Run("concurrent message additions", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 5; i++ {
			go func(index int) {
				err := store.AddMessage(conv.ID, Message{
					Role:    "user",
					Content: "Concurrent message",
				})
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 5; i++ {
			<-done
		}

		// Verify all messages were added
		updated, err := store.Get(conv.ID)
		require.NoError(t, err)
		assert.Equal(t, 5, len(updated.Messages))
	})
}

// Helper function to set up a memory store for testing
func setupMemoryStore(t *testing.T) *MemoryStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewMemoryStore(dir)
	require.NoError(t, err)
	return store
}
