package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

func TestHandleApprove(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           map[string]any
		setupApproval  bool
		approvalID     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "wrong method returns 405",
			method:         http.MethodGet,
			body:           map[string]any{"approval_id": "approval_1"},
			setupApproval:  false,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "missing approval_id returns 400",
			method:         http.MethodPost,
			body:           map[string]any{},
			setupApproval:  false,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "approval_id is required",
		},
		{
			name:           "empty approval_id returns 400",
			method:         http.MethodPost,
			body:           map[string]any{"approval_id": ""},
			setupApproval:  false,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "approval_id is required",
		},
		{
			name:           "unknown approval_id returns 404",
			method:         http.MethodPost,
			body:           map[string]any{"approval_id": "nonexistent"},
			setupApproval:  false,
			expectedStatus: http.StatusNotFound,
			expectedBody:   "approval not found or already resolved",
		},
		{
			name:           "valid approval returns 200",
			method:         http.MethodPost,
			body:           map[string]any{"approval_id": "approval_1"},
			setupApproval:  true,
			approvalID:     "approval_1",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"approved"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create server with approval store
			s := &Server{
				approvals: NewApprovalStore(),
			}

			// Setup pending approval if needed
			if tt.setupApproval {
				pa := s.approvals.Create(
					ToolCall{Name: "send_email", Args: map[string]any{"to": "test@example.com"}},
					map[string]any{"to": "test@example.com"},
				)
				require.Equal(t, tt.approvalID, pa.ID)

				// Start goroutine to receive decision
				go func() {
					decision, ok := pa.Wait(context.Background())
					assert.True(t, ok)
					assert.True(t, decision.Approved)
					assert.Empty(t, decision.Reason)
				}()
			}

			// Create request
			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			req := httptest.NewRequest(tt.method, "/api/chat/approve", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Record response
			w := httptest.NewRecorder()

			// Call handler
			s.handleApprove(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				// JSON response
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else {
				// Plain text error
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// TestHandleApprove_AfterTimeoutDiscard verifies that approving an action
// whose Wait already gave up (timeout / cancellation discarded it) returns a
// non-200 error instead of silently "succeeding" against a dead approval.
func TestHandleApprove_AfterTimeoutDiscard(t *testing.T) {
	s := &Server{
		approvals: NewApprovalStore(),
	}

	pa := s.approvals.Create(
		ToolCall{Name: "send_email", Args: map[string]any{"to": "test@example.com"}},
		map[string]any{"to": "test@example.com"},
	)

	// Simulate the handler flow when Wait does not get a decision.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, resolved := pa.Wait(ctx); resolved {
		t.Fatal("Wait with cancelled context returned resolved=true, want false")
	}
	s.approvals.Discard(pa.ID)

	body, err := json.Marshal(map[string]any{"approval_id": pa.ID})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/chat/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleApprove(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "approval not found or already resolved")
}

func TestHandleReject(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           map[string]any
		setupApproval  bool
		approvalID     string
		expectedStatus int
		expectedBody   string
		expectedReason string
	}{
		{
			name:           "wrong method returns 405",
			method:         http.MethodGet,
			body:           map[string]any{"approval_id": "approval_1"},
			setupApproval:  false,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "missing approval_id returns 400",
			method:         http.MethodPost,
			body:           map[string]any{},
			setupApproval:  false,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "approval_id is required",
		},
		{
			name:           "empty approval_id returns 400",
			method:         http.MethodPost,
			body:           map[string]any{"approval_id": ""},
			setupApproval:  false,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "approval_id is required",
		},
		{
			name:           "unknown approval_id returns 404",
			method:         http.MethodPost,
			body:           map[string]any{"approval_id": "nonexistent"},
			setupApproval:  false,
			expectedStatus: http.StatusNotFound,
			expectedBody:   "approval not found or already resolved",
		},
		{
			name:           "valid reject with custom reason",
			method:         http.MethodPost,
			body:           map[string]any{"approval_id": "approval_1", "reason": "Not authorized"},
			setupApproval:  true,
			approvalID:     "approval_1",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"rejected"}`,
			expectedReason: "Not authorized",
		},
		{
			name:           "valid reject with default reason",
			method:         http.MethodPost,
			body:           map[string]any{"approval_id": "approval_1"},
			setupApproval:  true,
			approvalID:     "approval_1",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"rejected"}`,
			expectedReason: "rejected by user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create server with approval store
			s := &Server{
				approvals: NewApprovalStore(),
			}

			// Setup pending approval if needed
			if tt.setupApproval {
				pa := s.approvals.Create(
					ToolCall{Name: "create_event", Args: map[string]any{"title": "Meeting"}},
					map[string]any{"title": "Meeting"},
				)
				require.Equal(t, tt.approvalID, pa.ID)

				// Start goroutine to receive decision
				go func() {
					decision, ok := pa.Wait(context.Background())
					assert.True(t, ok)
					assert.False(t, decision.Approved)
					assert.Equal(t, tt.expectedReason, decision.Reason)
				}()
			}

			// Create request
			bodyBytes, err := json.Marshal(tt.body)
			require.NoError(t, err)

			req := httptest.NewRequest(tt.method, "/api/chat/reject", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// Record response
			w := httptest.NewRecorder()

			// Call handler
			s.handleReject(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				// JSON response
				assert.JSONEq(t, tt.expectedBody, w.Body.String())
			} else {
				// Plain text error
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

// newApprovalChatServer builds a Server whose agent is a fake codex script:
// the first invocation emits a gated send_email tool call, later invocations
// return plain text. runTimeout simulates the agent-run deadline.
func newApprovalChatServer(t *testing.T, runTimeout time.Duration, client *nylas.MockClient) (*Server, string) {
	t.Helper()

	dir := t.TempDir()
	state := filepath.Join(dir, "called")
	script := filepath.Join(dir, "fake-codex")
	body := "#!/bin/sh\n" +
		"if [ ! -f " + state + " ]; then\n" +
		"  touch " + state + "\n" +
		"  echo 'TOOL_CALL: {\"name\":\"send_email\",\"args\":{\"to\":\"a@example.com\",\"subject\":\"Hi\",\"body\":\"Hello\"}}'\n" +
		"else\n" +
		"  echo 'Email sent.'\n" +
		"fi\n"
	require.NoError(t, os.WriteFile(script, []byte(body), 0o700))

	agent := &Agent{Type: AgentCodex, Path: script}
	mem, err := NewMemoryStore(t.TempDir())
	require.NoError(t, err)

	// Pre-create the conversation with a title so handleChat does not spawn
	// the async generateTitle goroutine (which would outlive the test).
	conv, err := mem.Create(string(AgentCodex))
	require.NoError(t, err)
	require.NoError(t, mem.UpdateTitle(conv.ID, "approval test"))

	s := &Server{
		agent:      agent,
		agents:     []Agent{*agent},
		grantID:    "test-grant",
		memory:     mem,
		executor:   NewToolExecutor(client, "test-grant"),
		context:    NewContextBuilder(agent, mem, "test-grant"),
		session:    NewActiveSession(),
		approvals:  NewApprovalStore(),
		runTimeout: runTimeout,
	}
	return s, conv.ID
}

// nextSSEEvent reads the next SSE event from the stream.
func nextSSEEvent(t *testing.T, sc *bufio.Scanner) (string, map[string]any) {
	t.Helper()

	var event string
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			raw := strings.TrimPrefix(line, "data: ")
			var data map[string]any
			if raw != "null" {
				require.NoError(t, json.Unmarshal([]byte(raw), &data))
			}
			return event, data
		}
	}
	t.Fatalf("SSE stream ended unexpectedly (last event: %q, err: %v)", event, sc.Err())
	return "", nil
}

// readUntilSSEEvent reads SSE events until one with the given name arrives.
func readUntilSSEEvent(t *testing.T, sc *bufio.Scanner, name string) map[string]any {
	t.Helper()
	for {
		event, data := nextSSEEvent(t, sc)
		if event == name {
			return data
		}
	}
}

func (s *Server) approveViaHandler(t *testing.T, approvalID string) int {
	t.Helper()

	body, err := json.Marshal(map[string]any{"approval_id": approvalID})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/chat/approve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.handleApprove(w, req)
	return w.Code
}

// TestHandleChat_ApprovalSurvivesAgentRunDeadline reproduces the UX
// regression where the approval wait was bounded by the agent-run deadline
// instead of the approval window: a user approving AFTER the agent-run
// deadline must still win, and the approved tool call must then execute with
// a fresh, non-expired context.
func TestHandleChat_ApprovalSurvivesAgentRunDeadline(t *testing.T) {
	client := nylas.NewMockClient()
	execCtxErr := make(chan error, 1)
	client.SendMessageFunc = func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
		execCtxErr <- ctx.Err()
		return &domain.Message{ID: "msg-1"}, nil
	}

	// Must outlast the fake agent script's exec (cold start ~150ms, but
	// 1s proved flaky when the full suite runs in parallel and starves the
	// process start), while still expiring during the approval wait below.
	runTimeout := 3 * time.Second
	s, convID := newApprovalChatServer(t, runTimeout, client)

	ts := httptest.NewServer(http.HandlerFunc(s.handleChat))
	defer ts.Close()

	reqBody := `{"message":"send the email","conversation_id":"` + convID + `"}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(reqBody))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	sc := bufio.NewScanner(resp.Body)
	required := readUntilSSEEvent(t, sc, "approval_required")
	approvalID, _ := required["approval_id"].(string)
	require.NotEmpty(t, approvalID)

	// Let the agent-run deadline expire while the user is deciding.
	time.Sleep(runTimeout + 500*time.Millisecond)

	// The late approval must still land (404 here means the wait was
	// killed by the agent-run deadline and the approval discarded).
	require.Equal(t, http.StatusOK, s.approveViaHandler(t, approvalID),
		"approval after agent-run deadline must succeed")

	resolved := readUntilSSEEvent(t, sc, "approval_resolved")
	assert.Equal(t, true, resolved["approved"], "late approval must be honored")

	// The approved action must execute on a fresh, non-expired context.
	select {
	case ctxErr := <-execCtxErr:
		assert.NoError(t, ctxErr, "approved tool call ran with an expired context")
	case <-time.After(5 * time.Second):
		t.Fatal("approved tool call was never executed")
	}

	result := readUntilSSEEvent(t, sc, "tool_result")
	errVal, _ := result["error"].(string)
	assert.Empty(t, errVal, "approved tool call must not fail")

	readUntilSSEEvent(t, sc, "done")
}

// TestHandleChat_ClientDisconnectDiscardsApproval verifies that closing the
// SSE connection while an approval is pending still cancels the wait and
// discards the pending approval so a late approve returns 404.
func TestHandleChat_ClientDisconnectDiscardsApproval(t *testing.T) {
	client := nylas.NewMockClient()
	client.SendMessageFunc = func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
		t.Error("tool must not execute after client disconnect")
		return &domain.Message{ID: "msg-1"}, nil
	}

	s, convID := newApprovalChatServer(t, 5*time.Second, client)

	ts := httptest.NewServer(http.HandlerFunc(s.handleChat))
	defer ts.Close()

	reqBody := `{"message":"send the email","conversation_id":"` + convID + `"}`
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(reqBody))
	require.NoError(t, err)

	sc := bufio.NewScanner(resp.Body)
	required := readUntilSSEEvent(t, sc, "approval_required")
	approvalID, _ := required["approval_id"].(string)
	require.NotEmpty(t, approvalID)

	// Disconnect the client mid-wait.
	require.NoError(t, resp.Body.Close())

	// The wait must observe the disconnect and discard the approval.
	require.Eventually(t, func() bool {
		_, pending := s.approvals.pending.Load(approvalID)
		return !pending
	}, 5*time.Second, 20*time.Millisecond, "approval not discarded after client disconnect")

	assert.Equal(t, http.StatusNotFound, s.approveViaHandler(t, approvalID),
		"approve after disconnect must report the approval as gone")
}
