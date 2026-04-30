package air

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// handleAISmartReplies handles POST /api/ai/smart-replies requests.
func (s *Server) handleAISmartReplies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SmartReplyRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SmartReplyResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Body == "" {
		writeJSON(w, http.StatusBadRequest, SmartReplyResponse{
			Success: false,
			Error:   "Email body is required",
		})
		return
	}

	// Truncate body for prompt
	body := req.Body
	if len(body) > 2000 {
		body = body[:2000] + "..."
	}

	// Build prompt for smart replies
	prompt := fmt.Sprintf(`Generate exactly 3 short, professional email reply suggestions for this email. Each reply should be 1-2 sentences max. Return ONLY a JSON array of 3 strings, nothing else.

From: %s
Subject: %s

%s

Return format: ["Reply 1", "Reply 2", "Reply 3"]`, req.From, req.Subject, body)

	result, err := runClaudeCommand(r.Context(), prompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, SmartReplyResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Parse the JSON array from the result
	var replies []string
	// Try to extract JSON array from response
	start := strings.Index(result, "[")
	end := strings.LastIndex(result, "]")
	if start != -1 && end != -1 && end > start {
		jsonStr := result[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &replies); err != nil {
			// Fallback: split by newlines if JSON parsing fails
			replies = parseRepliesFromText(result)
		}
	} else {
		replies = parseRepliesFromText(result)
	}

	// Ensure we have exactly 3 replies
	for len(replies) < 3 {
		replies = append(replies, "Thanks for your email!")
	}
	if len(replies) > 3 {
		replies = replies[:3]
	}

	writeJSON(w, http.StatusOK, SmartReplyResponse{
		Success: true,
		Replies: replies,
	})
}

// parseRepliesFromText extracts reply suggestions from plain text.
func parseRepliesFromText(text string) []string {
	var replies []string
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove numbering like "1.", "2.", etc.
		if len(line) > 2 && line[0] >= '1' && line[0] <= '9' && line[1] == '.' {
			line = strings.TrimSpace(line[2:])
		}
		// Remove quotes
		line = strings.Trim(line, `"'`)
		if len(line) > 10 && len(line) < 200 {
			replies = append(replies, line)
		}
	}
	return replies
}

// handleAIAutoLabel handles POST /api/ai/auto-label requests.
func (s *Server) handleAIAutoLabel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AutoLabelRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, AutoLabelResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Body == "" && req.Subject == "" {
		writeJSON(w, http.StatusBadRequest, AutoLabelResponse{
			Success: false,
			Error:   "Email subject or body is required",
		})
		return
	}

	// Truncate body for prompt
	body := req.Body
	if len(body) > 2000 {
		body = body[:2000] + "..."
	}

	// Build prompt for auto-labeling
	prompt := fmt.Sprintf(`Analyze this email and suggest appropriate labels. Return ONLY valid JSON.

From: %s
Subject: %s

%s

Return format:
{
  "labels": ["label1", "label2"],
  "category": "one of: meeting|task|fyi|question|social|newsletter|promotion|urgent|personal|work",
  "priority": "one of: high|normal|low"
}

Rules:
- labels: 1-4 relevant labels (e.g., "finance", "project-x", "team", "client")
- category: Choose the PRIMARY purpose
- priority: Based on urgency and sender importance`, req.From, req.Subject, body)

	result, err := runClaudeCommand(r.Context(), prompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, AutoLabelResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Parse JSON response
	var parsed struct {
		Labels   []string `json:"labels"`
		Category string   `json:"category"`
		Priority string   `json:"priority"`
	}

	// Try to extract JSON from response
	start := strings.Index(result, "{")
	end := strings.LastIndex(result, "}")
	if start != -1 && end != -1 && end > start {
		jsonStr := result[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			// Fallback to defaults
			writeJSON(w, http.StatusOK, AutoLabelResponse{
				Success:  true,
				Labels:   []string{"inbox"},
				Category: "fyi",
				Priority: "normal",
			})
			return
		}
	} else {
		writeJSON(w, http.StatusOK, AutoLabelResponse{
			Success:  true,
			Labels:   []string{"inbox"},
			Category: "fyi",
			Priority: "normal",
		})
		return
	}

	// Validate category
	validCategories := map[string]bool{
		"meeting": true, "task": true, "fyi": true, "question": true,
		"social": true, "newsletter": true, "promotion": true,
		"urgent": true, "personal": true, "work": true,
	}
	if !validCategories[parsed.Category] {
		parsed.Category = "fyi"
	}

	// Validate priority
	validPriorities := map[string]bool{"high": true, "normal": true, "low": true}
	if !validPriorities[parsed.Priority] {
		parsed.Priority = "normal"
	}

	// Ensure at least one label
	if len(parsed.Labels) == 0 {
		parsed.Labels = []string{"inbox"}
	}

	writeJSON(w, http.StatusOK, AutoLabelResponse{
		Success:  true,
		Labels:   parsed.Labels,
		Category: parsed.Category,
		Priority: parsed.Priority,
	})
}
