package air

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/nylas/cli/internal/httputil"
)

const smartComposeTimeout = 5 * time.Second

var runSmartComposeCommand = func(ctx context.Context, prompt string) ([]byte, error) {
	//nolint:gosec // G204: Command is hardcoded "claude" binary, prompt is passed as a single argument.
	cmd := exec.Command("claude", "-p", prompt)
	return runCommandOutput(ctx, cmd)
}

// CompleteRequest represents a smart compose request
type CompleteRequest struct {
	Text      string `json:"text"`
	MaxLength int    `json:"maxLength"`
	Context   string `json:"context,omitempty"`
}

// CompleteResponse represents a smart compose response
type CompleteResponse struct {
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
}

// handleAIComplete handles smart compose autocomplete requests
func (s *Server) handleAIComplete(w http.ResponseWriter, r *http.Request) {
	var req CompleteRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Text == "" {
		httputil.WriteJSON(w, http.StatusOK, CompleteResponse{Suggestion: "", Confidence: 0})
		return
	}

	if req.MaxLength == 0 {
		req.MaxLength = 100
	}

	suggestion := getAICompletion(r.Context(), req.Text, req.MaxLength)

	httputil.WriteJSON(w, http.StatusOK, CompleteResponse{
		Suggestion: suggestion,
		Confidence: 0.8,
	})
}

// getAICompletion gets completion from Claude via CLI
func getAICompletion(ctx context.Context, text string, maxLen int) string {
	prompt := buildCompletionPrompt(text, maxLen)

	ctx, cancel := context.WithTimeout(ctx, smartComposeTimeout)
	defer cancel()

	output, err := runSmartComposeCommand(ctx, prompt)
	if err != nil {
		return ""
	}

	suggestion := strings.TrimSpace(string(output))

	// Limit length
	if len(suggestion) > maxLen {
		// Try to break at word boundary
		if idx := strings.LastIndex(suggestion[:maxLen], " "); idx > 0 {
			suggestion = suggestion[:idx]
		} else {
			suggestion = suggestion[:maxLen]
		}
	}

	return suggestion
}

// buildCompletionPrompt creates prompt for autocomplete
func buildCompletionPrompt(text string, maxLen int) string {
	return strings.Join([]string{
		"You are an email autocomplete assistant.",
		"Complete the following email text naturally.",
		"Only provide the completion, not the original text.",
		"Keep it concise and professional.",
		fmt.Sprintf("Maximum %s characters.", strconv.Itoa(maxLen)),
		"",
		"Text to complete:",
		text,
		"",
		"Completion:",
	}, "\n")
}

func runCommandOutput(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	configureChildProcessGroup(cmd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case err := <-waitCh:
		if err != nil {
			return nil, err
		}
		return stdout.Bytes(), nil
	case <-ctx.Done():
		_ = killCommandTree(cmd)
		<-waitCh
		return nil, ctx.Err()
	}
}

// NLSearchRequest represents a natural language search request
type NLSearchRequest struct {
	Query string `json:"query"`
}

// NLSearchResponse represents parsed search parameters
type NLSearchResponse struct {
	From       string `json:"from,omitempty"`
	To         string `json:"to,omitempty"`
	Subject    string `json:"subject,omitempty"`
	DateAfter  string `json:"dateAfter,omitempty"`
	DateBefore string `json:"dateBefore,omitempty"`
	HasAttach  bool   `json:"hasAttachment,omitempty"`
	IsUnread   bool   `json:"isUnread,omitempty"`
	Keywords   string `json:"keywords,omitempty"`
}

// handleNLSearch handles natural language search queries
func (s *Server) handleNLSearch(w http.ResponseWriter, r *http.Request) {
	var req NLSearchRequest
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Query == "" {
		http.Error(w, "Query required", http.StatusBadRequest)
		return
	}

	result := parseNaturalLanguageSearch(req.Query)

	httputil.WriteJSON(w, http.StatusOK, result)
}

// parseNaturalLanguageSearch converts NL query to search params
func parseNaturalLanguageSearch(query string) NLSearchResponse {
	result := NLSearchResponse{}
	queryLower := strings.ToLower(query)

	// Parse time-based patterns FIRST (before "from" to avoid conflicts)
	if strings.Contains(queryLower, "last week") {
		result.DateAfter = "7d"
	} else if strings.Contains(queryLower, "yesterday") {
		result.DateAfter = "1d"
	} else if strings.Contains(queryLower, "today") {
		result.DateAfter = "0d"
	} else if strings.Contains(queryLower, "this month") {
		result.DateAfter = "30d"
	}

	// Parse "from X" patterns (skip time words)
	timeWords := map[string]bool{"last": true, "yesterday": true, "today": true, "this": true}
	if strings.Contains(queryLower, "from ") && !strings.Contains(queryLower, "from last") &&
		!strings.Contains(queryLower, "from yesterday") && !strings.Contains(queryLower, "from today") {
		parts := strings.SplitN(queryLower, "from ", 2)
		if len(parts) > 1 {
			words := strings.Fields(parts[1])
			if len(words) > 0 && !timeWords[words[0]] {
				result.From = words[0]
			}
		}
	}

	// Parse "to X" patterns
	if strings.Contains(queryLower, "to ") {
		parts := strings.SplitN(queryLower, "to ", 2)
		if len(parts) > 1 {
			words := strings.Fields(parts[1])
			if len(words) > 0 {
				result.To = words[0]
			}
		}
	}

	// Parse attachment pattern
	if strings.Contains(queryLower, "attachment") ||
		strings.Contains(queryLower, "attached") {
		result.HasAttach = true
	}

	// Parse unread pattern
	if strings.Contains(queryLower, "unread") {
		result.IsUnread = true
	}

	// Extract remaining keywords
	keywords := extractKeywords(queryLower)
	if len(keywords) > 0 {
		result.Keywords = strings.Join(keywords, " ")
	}

	return result
}

// extractKeywords extracts search keywords from query
func extractKeywords(query string) []string {
	stopWords := map[string]bool{
		"from": true, "to": true, "about": true, "with": true,
		"the": true, "a": true, "an": true, "and": true,
		"or": true, "in": true, "on": true, "at": true,
		"last": true, "week": true, "month": true, "yesterday": true,
		"today": true, "emails": true, "email": true, "messages": true,
	}

	words := strings.Fields(query)
	keywords := []string{}

	for _, word := range words {
		word = strings.Trim(word, ".,!?")
		if !stopWords[word] && len(word) > 2 {
			keywords = append(keywords, word)
		}
	}

	return keywords
}
