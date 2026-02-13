package chat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupServerWithMemory creates a test server with memory store.
func setupServerWithMemory(t *testing.T) *Server {
	t.Helper()
	memory := setupMemoryStore(t)
	agent := &Agent{Type: AgentClaude, Version: "1.0"}
	return &Server{
		agent:  agent,
		memory: memory,
	}
}

func TestHandleConversations(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupConvs     int // number of conversations to create before request
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "GET returns empty array when no conversations",
			method:         http.MethodGet,
			setupConvs:     0,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var convs []ConversationSummary
				err := json.Unmarshal([]byte(body), &convs)
				require.NoError(t, err)
				assert.Empty(t, convs)
			},
		},
		{
			name:           "GET lists all conversations",
			method:         http.MethodGet,
			setupConvs:     3,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var convs []ConversationSummary
				err := json.Unmarshal([]byte(body), &convs)
				require.NoError(t, err)
				assert.Len(t, convs, 3)
				for _, conv := range convs {
					assert.NotEmpty(t, conv.ID)
					assert.NotEmpty(t, conv.Title)
					assert.NotEmpty(t, conv.Agent)
				}
			},
		},
		{
			name:           "POST creates new conversation",
			method:         http.MethodPost,
			setupConvs:     0,
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body string) {
				var resp map[string]string
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.NotEmpty(t, resp["id"])
				assert.Equal(t, "New conversation", resp["title"])
			},
		},
		{
			name:           "PUT returns method not allowed",
			method:         http.MethodPut,
			setupConvs:     0,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Method not allowed")
			},
		},
		{
			name:           "DELETE returns method not allowed",
			method:         http.MethodDelete,
			setupConvs:     0,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Method not allowed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupServerWithMemory(t)

			// Setup: create conversations if needed
			for i := 0; i < tt.setupConvs; i++ {
				_, err := server.memory.Create("test-agent")
				require.NoError(t, err)
			}

			// Create request
			req := httptest.NewRequest(tt.method, "/api/conversations", nil)
			w := httptest.NewRecorder()

			// Execute
			server.handleConversations(w, req)

			// Verify status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify response body
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

func TestHandleConversationByID(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		setupConv      bool   // whether to create a conversation first
		urlPath        string // URL path to use (empty = use created conv ID)
		expectedStatus int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "GET retrieves existing conversation",
			method:         http.MethodGet,
			setupConv:      true,
			urlPath:        "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var conv Conversation
				err := json.Unmarshal([]byte(body), &conv)
				require.NoError(t, err)
				assert.NotEmpty(t, conv.ID)
				assert.Equal(t, "New conversation", conv.Title)
				assert.NotNil(t, conv.Messages)
			},
		},
		{
			name:           "GET returns 404 for non-existent conversation",
			method:         http.MethodGet,
			setupConv:      false,
			urlPath:        "/api/conversations/nonexistent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Conversation not found")
			},
		},
		{
			name:           "GET returns 400 for empty ID",
			method:         http.MethodGet,
			setupConv:      false,
			urlPath:        "/api/conversations/",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "conversation ID required")
			},
		},
		{
			name:           "DELETE removes existing conversation",
			method:         http.MethodDelete,
			setupConv:      true,
			urlPath:        "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var resp map[string]string
				err := json.Unmarshal([]byte(body), &resp)
				require.NoError(t, err)
				assert.Equal(t, "deleted", resp["status"])
			},
		},
		{
			name:           "DELETE returns 404 for non-existent conversation",
			method:         http.MethodDelete,
			setupConv:      false,
			urlPath:        "/api/conversations/nonexistent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Failed to delete conversation")
			},
		},
		{
			name:           "DELETE returns 400 for empty ID",
			method:         http.MethodDelete,
			setupConv:      false,
			urlPath:        "/api/conversations/",
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "conversation ID required")
			},
		},
		{
			name:           "POST returns method not allowed",
			method:         http.MethodPost,
			setupConv:      true,
			urlPath:        "",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Method not allowed")
			},
		},
		{
			name:           "PUT returns method not allowed",
			method:         http.MethodPut,
			setupConv:      true,
			urlPath:        "",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Method not allowed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupServerWithMemory(t)

			// Setup: create conversation if needed
			var convID string
			if tt.setupConv {
				conv, err := server.memory.Create("test-agent")
				require.NoError(t, err)
				convID = conv.ID
			}

			// Build URL path
			urlPath := tt.urlPath
			if urlPath == "" && tt.setupConv {
				urlPath = "/api/conversations/" + convID
			}

			// Create request
			req := httptest.NewRequest(tt.method, urlPath, nil)
			w := httptest.NewRecorder()

			// Execute
			server.handleConversationByID(w, req)

			// Verify status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify response body
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.String())
			}
		})
	}
}

func TestListConversations(t *testing.T) {
	tests := []struct {
		name          string
		setupConvs    int
		expectedCount int
	}{
		{
			name:          "returns empty array when no conversations",
			setupConvs:    0,
			expectedCount: 0,
		},
		{
			name:          "lists single conversation",
			setupConvs:    1,
			expectedCount: 1,
		},
		{
			name:          "lists multiple conversations",
			setupConvs:    5,
			expectedCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupServerWithMemory(t)

			// Setup: create conversations
			for i := 0; i < tt.setupConvs; i++ {
				_, err := server.memory.Create("test-agent")
				require.NoError(t, err)
			}

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
			w := httptest.NewRecorder()

			// Execute
			server.listConversations(w, req)

			// Verify response
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

			var convs []ConversationSummary
			err := json.Unmarshal(w.Body.Bytes(), &convs)
			require.NoError(t, err)
			assert.Len(t, convs, tt.expectedCount)
		})
	}
}

func TestCreateConversation(t *testing.T) {
	tests := []struct {
		name      string
		agentType AgentType
	}{
		{
			name:      "creates conversation with claude agent",
			agentType: AgentClaude,
		},
		{
			name:      "creates conversation with ollama agent",
			agentType: AgentOllama,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := setupMemoryStore(t)
			server := &Server{
				agent:  &Agent{Type: tt.agentType},
				memory: memory,
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/conversations", nil)
			w := httptest.NewRecorder()

			// Execute
			server.createConversation(w, req)

			// Verify response
			assert.Equal(t, http.StatusCreated, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.NotEmpty(t, resp["id"])
			assert.Equal(t, "New conversation", resp["title"])

			// Verify conversation was actually created
			conv, err := memory.Get(resp["id"])
			require.NoError(t, err)
			assert.Equal(t, resp["id"], conv.ID)
			assert.Equal(t, string(tt.agentType), conv.Agent)
		})
	}
}

func TestGetConversation(t *testing.T) {
	tests := []struct {
		name           string
		setupConv      bool
		convID         string
		expectedStatus int
	}{
		{
			name:           "retrieves existing conversation",
			setupConv:      true,
			convID:         "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "returns 404 for non-existent conversation",
			setupConv:      false,
			convID:         "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupServerWithMemory(t)

			// Setup: create conversation if needed
			convID := tt.convID
			if tt.setupConv {
				conv, err := server.memory.Create("test-agent")
				require.NoError(t, err)
				convID = conv.ID
			}

			// Create request
			w := httptest.NewRecorder()

			// Execute
			server.getConversation(w, convID)

			// Verify status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify response body for successful retrieval
			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
				var conv Conversation
				err := json.Unmarshal(w.Body.Bytes(), &conv)
				require.NoError(t, err)
				assert.Equal(t, convID, conv.ID)
				assert.NotNil(t, conv.Messages)
			}
		})
	}
}

func TestDeleteConversation(t *testing.T) {
	tests := []struct {
		name           string
		setupConv      bool
		convID         string
		expectedStatus int
	}{
		{
			name:           "deletes existing conversation",
			setupConv:      true,
			convID:         "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "returns 404 for non-existent conversation",
			setupConv:      false,
			convID:         "nonexistent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupServerWithMemory(t)

			// Setup: create conversation if needed
			convID := tt.convID
			if tt.setupConv {
				conv, err := server.memory.Create("test-agent")
				require.NoError(t, err)
				convID = conv.ID
			}

			// Create request
			w := httptest.NewRecorder()

			// Execute
			server.deleteConversation(w, convID)

			// Verify status
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify response body
			if tt.expectedStatus == http.StatusOK {
				assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
				var resp map[string]string
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				require.NoError(t, err)
				assert.Equal(t, "deleted", resp["status"])

				// Verify conversation is actually deleted
				_, err = server.memory.Get(convID)
				assert.Error(t, err)
			}
		})
	}
}

func TestConversationEndToEnd(t *testing.T) {
	server := setupServerWithMemory(t)

	// List conversations (should be empty)
	req := httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
	w := httptest.NewRecorder()
	server.handleConversations(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var convs []ConversationSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &convs))
	assert.Empty(t, convs)

	// Create a conversation
	req = httptest.NewRequest(http.MethodPost, "/api/conversations", nil)
	w = httptest.NewRecorder()
	server.handleConversations(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	var createResp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
	convID := createResp["id"]
	assert.NotEmpty(t, convID)

	// Get the conversation by ID
	req = httptest.NewRequest(http.MethodGet, "/api/conversations/"+convID, nil)
	w = httptest.NewRecorder()
	server.handleConversationByID(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var conv Conversation
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &conv))
	assert.Equal(t, convID, conv.ID)

	// List conversations (should have one)
	req = httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
	w = httptest.NewRecorder()
	server.handleConversations(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &convs))
	assert.Len(t, convs, 1)
	assert.Equal(t, convID, convs[0].ID)

	// Delete the conversation
	req = httptest.NewRequest(http.MethodDelete, "/api/conversations/"+convID, nil)
	w = httptest.NewRecorder()
	server.handleConversationByID(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify it's deleted
	req = httptest.NewRequest(http.MethodGet, "/api/conversations/"+convID, nil)
	w = httptest.NewRecorder()
	server.handleConversationByID(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestMemoryStoreFailure(t *testing.T) {
	t.Run("listConversations handles deleted directory", func(t *testing.T) {
		tempDir := t.TempDir()
		memory, err := NewMemoryStore(tempDir)
		require.NoError(t, err)
		server := &Server{agent: &Agent{Type: AgentClaude}, memory: memory}

		_ = os.RemoveAll(tempDir)
		req := httptest.NewRequest(http.MethodGet, "/api/conversations", nil)
		w := httptest.NewRecorder()
		server.listConversations(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("createConversation handles permission error", func(t *testing.T) {
		tempDir := t.TempDir()
		memory, err := NewMemoryStore(tempDir)
		require.NoError(t, err)
		require.NoError(t, os.Chmod(tempDir, 0444))
		t.Cleanup(func() { _ = os.Chmod(tempDir, 0755) })

		server := &Server{agent: &Agent{Type: AgentClaude}, memory: memory}
		req := httptest.NewRequest(http.MethodPost, "/api/conversations", nil)
		w := httptest.NewRecorder()
		server.createConversation(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to create conversation")
	})

	t.Run("getConversation handles corrupted file", func(t *testing.T) {
		tempDir := t.TempDir()
		memory, err := NewMemoryStore(tempDir)
		require.NoError(t, err)

		corruptedPath := filepath.Join(tempDir, "corrupt.json")
		require.NoError(t, os.WriteFile(corruptedPath, []byte("not valid json"), 0600))

		server := &Server{agent: &Agent{Type: AgentClaude}, memory: memory}
		w := httptest.NewRecorder()
		server.getConversation(w, "corrupt")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
