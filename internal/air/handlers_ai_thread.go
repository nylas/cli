package air

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// handleAIThreadSummary handles POST /api/ai/thread-summary requests.
func (s *Server) handleAIThreadSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ThreadSummaryRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ThreadSummaryResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if len(req.Messages) == 0 {
		writeJSON(w, http.StatusBadRequest, ThreadSummaryResponse{
			Success: false,
			Error:   "At least one message is required",
		})
		return
	}

	// Build conversation text from messages
	var conversationBuilder strings.Builder
	participants := make(map[string]bool)

	for i, msg := range req.Messages {
		// Track participants
		if msg.From != "" {
			participants[msg.From] = true
		}

		// Truncate individual message bodies
		body := msg.Body
		if len(body) > 1000 {
			body = body[:1000] + "..."
		}

		_, _ = fmt.Fprintf(&conversationBuilder, "--- Message %d ---\n", i+1)
		_, _ = fmt.Fprintf(&conversationBuilder, "From: %s\n", msg.From)
		if msg.Subject != "" {
			_, _ = fmt.Fprintf(&conversationBuilder, "Subject: %s\n", msg.Subject)
		}
		conversationBuilder.WriteString(body)
		conversationBuilder.WriteString("\n\n")

		// Limit total conversation length
		if conversationBuilder.Len() > 6000 {
			conversationBuilder.WriteString("... (additional messages truncated)")
			break
		}
	}

	// Get participant list
	participantList := make([]string, 0, len(participants))
	for p := range participants {
		participantList = append(participantList, p)
	}

	// Build prompt for thread summary
	prompt := fmt.Sprintf(`Summarize this email thread conversation. Return ONLY valid JSON.

%s

Return format:
{
  "summary": "2-4 sentence overall summary of the thread",
  "key_points": ["point 1", "point 2", "point 3"],
  "action_items": ["action 1", "action 2"],
  "timeline": "Brief timeline description (e.g., 'Started Monday with request, followed up Wednesday, resolved Friday')",
  "next_steps": "What needs to happen next, if anything"
}

Rules:
- summary: Capture the main topic and outcome of the conversation
- key_points: 2-5 most important points discussed
- action_items: Specific tasks mentioned (empty array if none)
- timeline: Brief description of how the conversation evolved
- next_steps: Clear next action if any, or empty string`, conversationBuilder.String())

	result, err := runClaudeCommand(r.Context(), prompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, ThreadSummaryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Parse JSON response
	var parsed struct {
		Summary     string   `json:"summary"`
		KeyPoints   []string `json:"key_points"`
		ActionItems []string `json:"action_items"`
		Timeline    string   `json:"timeline"`
		NextSteps   string   `json:"next_steps"`
	}

	// Try to extract JSON from response
	start := strings.Index(result, "{")
	end := strings.LastIndex(result, "}")
	if start != -1 && end != -1 && end > start {
		jsonStr := result[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			// Fallback to raw result
			writeJSON(w, http.StatusOK, ThreadSummaryResponse{
				Success:      true,
				Summary:      result,
				KeyPoints:    []string{},
				ActionItems:  []string{},
				Participants: participantList,
				Timeline:     "",
				MessageCount: len(req.Messages),
			})
			return
		}
	} else {
		writeJSON(w, http.StatusOK, ThreadSummaryResponse{
			Success:      true,
			Summary:      result,
			KeyPoints:    []string{},
			ActionItems:  []string{},
			Participants: participantList,
			Timeline:     "",
			MessageCount: len(req.Messages),
		})
		return
	}

	writeJSON(w, http.StatusOK, ThreadSummaryResponse{
		Success:      true,
		Summary:      parsed.Summary,
		KeyPoints:    parsed.KeyPoints,
		ActionItems:  parsed.ActionItems,
		Participants: participantList,
		Timeline:     parsed.Timeline,
		NextSteps:    parsed.NextSteps,
		MessageCount: len(req.Messages),
	})
}

// runClaudeCommand runs the claude CLI with the given prompt.
//
// The provided ctx is honored: cancelling it (e.g. when the HTTP request is
// aborted by the client) terminates the subprocess. A 30-second timeout is
// also applied on top of the caller's context so a runaway claude process
// can't block forever.
func runClaudeCommand(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Find claude binary
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude code CLI not found: please install it from https://claude.ai/code")
	}

	// Create command: echo "prompt" | claude -p
	// #nosec G204 -- claudePath verified via exec.LookPath from system PATH, user prompt only in stdin (not in command path)
	cmd := exec.CommandContext(ctx, claudePath, "-p")
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		// Distinguish caller cancellation from our local 30s timeout.
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return "", fmt.Errorf("claude code timed out after 30 seconds")
		case context.Canceled:
			return "", ctx.Err()
		}
		// Return stderr if available
		if stderr.Len() > 0 {
			return "", fmt.Errorf("claude code error: %s", stderr.String())
		}
		return "", fmt.Errorf("claude code error: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
