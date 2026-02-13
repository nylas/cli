package chat

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
					decision, ok := pa.Wait()
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
					decision, ok := pa.Wait()
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
