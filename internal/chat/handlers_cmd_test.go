package chat

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer creates a test server with mocked dependencies.
func setupTestServer(t *testing.T) *Server {
	t.Helper()
	memory := setupMemoryStore(t)
	mockClient := nylas.NewMockClient()
	agent := &Agent{Type: AgentClaude, Version: "1.0"}
	executor := NewToolExecutor(mockClient, "test-grant")

	return &Server{
		agent:    agent,
		grantID:  "test-grant",
		memory:   memory,
		executor: executor,
	}
}

func TestHandleCommand(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           commandRequest
		wantStatus     int
		wantErrMessage string
	}{
		{
			name:           "rejects non-POST method",
			method:         http.MethodGet,
			body:           commandRequest{},
			wantStatus:     http.StatusMethodNotAllowed,
			wantErrMessage: "Method not allowed",
		},
		{
			name:           "rejects missing command name",
			method:         http.MethodPost,
			body:           commandRequest{Name: ""},
			wantStatus:     http.StatusBadRequest,
			wantErrMessage: "command name is required",
		},
		{
			name:       "handles unknown command",
			method:     http.MethodPost,
			body:       commandRequest{Name: "invalid"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "handles status command",
			method:     http.MethodPost,
			body:       commandRequest{Name: "status"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "handles email command",
			method:     http.MethodPost,
			body:       commandRequest{Name: "email", Args: "test"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "handles calendar command",
			method:     http.MethodPost,
			body:       commandRequest{Name: "calendar", Args: "7"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "handles contacts command",
			method:     http.MethodPost,
			body:       commandRequest{Name: "contacts", Args: "john"},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := setupTestServer(t)

			// Create request
			var body []byte
			if tt.method == http.MethodPost {
				body, _ = json.Marshal(tt.body)
			}
			req := httptest.NewRequest(tt.method, "/api/command", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			// Execute
			server.handleCommand(w, req)

			// Verify status code
			assert.Equal(t, tt.wantStatus, w.Code)

			// Verify error message for non-200 responses
			if tt.wantErrMessage != "" {
				assert.Contains(t, w.Body.String(), tt.wantErrMessage)
			}

			// Verify successful responses have proper structure
			if tt.wantStatus == http.StatusOK {
				var resp commandResponse
				err := json.NewDecoder(w.Body).Decode(&resp)
				require.NoError(t, err)

				// Unknown command returns error
				if tt.body.Name == "invalid" {
					assert.Contains(t, resp.Error, "unknown command")
				} else {
					// Valid commands return content or data
					assert.True(t, resp.Content != "" || resp.Error == "")
				}
			}
		})
	}
}

func TestCmdStatus(t *testing.T) {
	tests := []struct {
		name      string
		agent     *Agent
		grantID   string
		convCount int
	}{
		{
			name:      "returns status with claude agent",
			agent:     &Agent{Type: AgentClaude, Version: "1.0"},
			grantID:   "grant-123",
			convCount: 0,
		},
		{
			name:      "returns status with ollama agent",
			agent:     &Agent{Type: AgentOllama, Model: "mistral"},
			grantID:   "grant-456",
			convCount: 3,
		},
		{
			name:      "handles empty grant ID",
			agent:     &Agent{Type: AgentClaude},
			grantID:   "",
			convCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memory := setupMemoryStore(t)
			// Create conversations
			for i := 0; i < tt.convCount; i++ {
				_, _ = memory.Create("test-agent")
			}

			server := &Server{
				agent:   tt.agent,
				grantID: tt.grantID,
				memory:  memory,
			}

			resp := server.cmdStatus()

			// Verify response structure
			require.NotEmpty(t, resp.Content)
			assert.Contains(t, resp.Content, "Status")
			assert.Contains(t, resp.Content, "Agent:")
			assert.Contains(t, resp.Content, "Grant ID:")
			assert.Contains(t, resp.Content, "Conversations:")
			assert.Empty(t, resp.Error)

			// Verify agent type is included
			assert.Contains(t, resp.Content, string(tt.agent.Type))

			// Verify grant ID is included
			if tt.grantID != "" {
				assert.Contains(t, resp.Content, tt.grantID)
			}
		})
	}
}

func TestToolResultToResponse(t *testing.T) {
	tests := []struct {
		name        string
		result      ToolResult
		label       string
		wantContent bool
		wantError   string
	}{
		{
			name: "converts result with data",
			result: ToolResult{
				Name: "test",
				Data: map[string]string{"key": "value"},
			},
			label:       "emails",
			wantContent: true,
		},
		{
			name: "converts result with array data",
			result: ToolResult{
				Name: "test",
				Data: []map[string]string{{"id": "1"}, {"id": "2"}},
			},
			label:       "events",
			wantContent: true,
		},
		{
			name: "converts result with error",
			result: ToolResult{
				Name:  "test",
				Error: "api error occurred",
			},
			label:     "contacts",
			wantError: "api error occurred",
		},
		{
			name: "handles nil data",
			result: ToolResult{
				Name: "test",
				Data: nil,
			},
			label:       "results",
			wantContent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := toolResultToResponse(tt.result, tt.label)

			if tt.wantError != "" {
				assert.Equal(t, tt.wantError, resp.Error)
				assert.Empty(t, resp.Content)
			} else if tt.wantContent {
				assert.NotEmpty(t, resp.Content)
				assert.Contains(t, resp.Content, tt.label)
				assert.Contains(t, resp.Content, "```json")
				assert.Empty(t, resp.Error)

				// Verify JSON formatting
				if tt.result.Data != nil {
					dataJSON, _ := json.MarshalIndent(tt.result.Data, "", "  ")
					assert.Contains(t, resp.Content, string(dataJSON))
				}
			}
		})
	}

	t.Run("handles marshal failure gracefully", func(t *testing.T) {
		// Create unmarshalable data (channels can't be marshaled)
		ch := make(chan int)
		result := ToolResult{
			Name: "test",
			Data: ch,
		}

		resp := toolResultToResponse(result, "test")

		assert.Empty(t, resp.Content)
		assert.Equal(t, "failed to format results", resp.Error)
	})
}
