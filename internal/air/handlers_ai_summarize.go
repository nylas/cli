package air

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// handleAISummarize handles POST /api/ai/summarize requests.
func (s *Server) handleAISummarize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AIRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, AIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, AIResponse{
			Success: false,
			Error:   "Prompt is required",
		})
		return
	}

	// Run claude -p with the prompt
	summary, err := runClaudeCommand(r.Context(), req.Prompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, AIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, AIResponse{
		Success: true,
		Summary: summary,
	})
}

// handleAIEnhancedSummary handles POST /api/ai/enhanced-summary requests.
func (s *Server) handleAIEnhancedSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req EnhancedSummaryRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, EnhancedSummaryResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	if req.Body == "" {
		writeJSON(w, http.StatusBadRequest, EnhancedSummaryResponse{
			Success: false,
			Error:   "Email body is required",
		})
		return
	}

	// Truncate body for prompt
	body := req.Body
	if len(body) > 3000 {
		body = body[:3000] + "..."
	}

	// Build prompt for enhanced summary
	prompt := fmt.Sprintf(`Analyze this email and provide a structured response in JSON format.

From: %s
Subject: %s

%s

Return ONLY valid JSON in this exact format:
{
  "summary": "2-3 sentence summary of the email",
  "action_items": ["action 1", "action 2"],
  "sentiment": "positive|neutral|negative|urgent",
  "category": "meeting|task|fyi|question|social"
}

Rules:
- action_items: List specific tasks or requests. Empty array if none.
- sentiment: Choose ONE based on tone and urgency
- category: Choose the PRIMARY purpose of the email`, req.From, req.Subject, body)

	result, err := runClaudeCommand(r.Context(), prompt)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, EnhancedSummaryResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// Parse JSON response
	var parsed struct {
		Summary     string   `json:"summary"`
		ActionItems []string `json:"action_items"`
		Sentiment   string   `json:"sentiment"`
		Category    string   `json:"category"`
	}

	// Try to extract JSON from response
	start := strings.Index(result, "{")
	end := strings.LastIndex(result, "}")
	if start != -1 && end != -1 && end > start {
		jsonStr := result[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
			// Fallback to basic summary
			writeJSON(w, http.StatusOK, EnhancedSummaryResponse{
				Success:     true,
				Summary:     result,
				ActionItems: []string{},
				Sentiment:   "neutral",
				Category:    "fyi",
			})
			return
		}
	} else {
		// No JSON found, use raw result as summary
		writeJSON(w, http.StatusOK, EnhancedSummaryResponse{
			Success:     true,
			Summary:     result,
			ActionItems: []string{},
			Sentiment:   "neutral",
			Category:    "fyi",
		})
		return
	}

	// Validate sentiment
	validSentiments := map[string]bool{"positive": true, "neutral": true, "negative": true, "urgent": true}
	if !validSentiments[parsed.Sentiment] {
		parsed.Sentiment = "neutral"
	}

	// Validate category
	validCategories := map[string]bool{"meeting": true, "task": true, "fyi": true, "question": true, "social": true}
	if !validCategories[parsed.Category] {
		parsed.Category = "fyi"
	}

	writeJSON(w, http.StatusOK, EnhancedSummaryResponse{
		Success:     true,
		Summary:     parsed.Summary,
		ActionItems: parsed.ActionItems,
		Sentiment:   parsed.Sentiment,
		Category:    parsed.Category,
	})
}
