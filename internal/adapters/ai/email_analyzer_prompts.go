package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (a *EmailAnalyzer) buildAnalysisPrompt(threadContext string, req *domain.EmailAnalysisRequest) string {
	var builder strings.Builder

	builder.WriteString("Analyze the following email thread and provide:\n\n")
	builder.WriteString("1. The primary purpose of the discussion (1 sentence)\n")
	builder.WriteString("2. Key topics discussed (list 2-5 topics)\n")
	builder.WriteString("3. Priority level (low, medium, high, or urgent) with reasoning\n")
	builder.WriteString("4. Suggested meeting duration in minutes\n")

	if req.IncludeAgenda {
		builder.WriteString("5. A structured meeting agenda with items and estimated durations\n")
	}

	if req.IncludeTime {
		builder.WriteString("6. Best time for the meeting considering participant timezones\n")
	}

	builder.WriteString("\nFormat your response as follows:\n")
	builder.WriteString("PURPOSE: [purpose]\n")
	builder.WriteString("TOPICS:\n- [topic 1]\n- [topic 2]\n")
	builder.WriteString("PRIORITY: [level] - [reasoning]\n")
	builder.WriteString("DURATION: [minutes] minutes - [reasoning]\n")

	if req.IncludeAgenda {
		builder.WriteString("AGENDA:\n## [Agenda Title]\n### Item 1: [title] ([duration] min)\n[description]\n")
	}

	builder.WriteString("\n---\n\n")
	builder.WriteString(threadContext)

	return builder.String()
}

// parseAnalysisResponse parses the LLM response into an EmailThreadAnalysis.
func (a *EmailAnalyzer) parseAnalysisResponse(response string, req *domain.EmailAnalysisRequest) (*domain.EmailThreadAnalysis, error) {
	analysis := &domain.EmailThreadAnalysis{
		ThreadID: req.ThreadID,
	}

	lines := strings.Split(response, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Parse PURPOSE
		if strings.HasPrefix(line, "PURPOSE:") {
			analysis.Purpose = strings.TrimSpace(strings.TrimPrefix(line, "PURPOSE:"))
		}

		// Parse TOPICS
		if strings.HasPrefix(line, "TOPICS:") {
			// Next lines starting with "- " are topics
			for j := i + 1; j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "- "); j++ {
				topic := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[j]), "- "))
				analysis.Topics = append(analysis.Topics, topic)
			}
		}

		// Parse PRIORITY
		if strings.HasPrefix(line, "PRIORITY:") {
			priorityLine := strings.TrimPrefix(line, "PRIORITY:")
			parts := strings.SplitN(priorityLine, "-", 2)
			if len(parts) > 0 {
				priorityStr := strings.TrimSpace(strings.ToLower(parts[0]))
				analysis.Priority = domain.MeetingPriority(priorityStr)
			}
		}

		// Parse DURATION
		if strings.HasPrefix(line, "DURATION:") {
			durationLine := strings.TrimPrefix(line, "DURATION:")
			// Extract number from "60 minutes - reasoning"
			parts := strings.Fields(durationLine)
			if len(parts) > 0 {
				var duration int
				_, _ = fmt.Sscanf(parts[0], "%d", &duration)
				analysis.SuggestedDuration = duration
			}
		}

		// Parse AGENDA (if requested)
		if req.IncludeAgenda && strings.HasPrefix(line, "AGENDA:") {
			agenda := a.parseAgenda(lines[i+1:])
			analysis.Agenda = agenda
		}
	}

	// Default values if parsing failed
	if analysis.SuggestedDuration == 0 {
		analysis.SuggestedDuration = 30 // Default 30 minutes
	}
	if analysis.Priority == "" {
		analysis.Priority = domain.PriorityMedium
	}

	return analysis, nil
}

// parseAgenda parses the agenda section from LLM response.
func (a *EmailAnalyzer) parseAgenda(lines []string) *domain.MeetingAgenda {
	agenda := &domain.MeetingAgenda{
		Items: []domain.AgendaItem{},
	}

	var currentItem *domain.AgendaItem

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Agenda title (##)
		if strings.HasPrefix(line, "## ") {
			agenda.Title = strings.TrimPrefix(line, "## ")
		}

		// Agenda item (###)
		if strings.HasPrefix(line, "### ") {
			if currentItem != nil {
				agenda.Items = append(agenda.Items, *currentItem)
			}

			// Parse "Item 1: Title (30 min)"
			itemLine := strings.TrimPrefix(line, "### ")
			currentItem = &domain.AgendaItem{}

			// Extract duration from parentheses
			if idx := strings.Index(itemLine, "("); idx != -1 {
				if endIdx := strings.Index(itemLine, ")"); endIdx > idx {
					durationStr := itemLine[idx+1 : endIdx]
					var duration int
					_, _ = fmt.Sscanf(durationStr, "%d", &duration)
					currentItem.Duration = duration
					itemLine = strings.TrimSpace(itemLine[:idx])
				}
			}

			currentItem.Title = itemLine
		}

		// Description (regular text after item)
		if currentItem != nil && !strings.HasPrefix(line, "#") && line != "" {
			if currentItem.Description != "" {
				currentItem.Description += " "
			}
			currentItem.Description += line
		}
	}

	// Add the last item
	if currentItem != nil {
		agenda.Items = append(agenda.Items, *currentItem)
	}

	return agenda
}

// analyzeParticipants analyzes participant involvement in the thread.
func (a *EmailAnalyzer) analyzeParticipants(thread *domain.Thread, messages []domain.Message) []domain.ParticipantInfo {
	participantMap := make(map[string]*domain.ParticipantInfo)

	// Initialize participants from thread
	for _, p := range thread.Participants {
		participantMap[p.Email] = &domain.ParticipantInfo{
			Email:        p.Email,
			Name:         p.Name,
			MentionCount: 0,
			MessageCount: 0,
			Required:     false,
			Involvement:  domain.InvolvementLow,
		}
	}

	// Count messages per participant
	for _, msg := range messages {
		for _, from := range msg.From {
			if p, exists := participantMap[from.Email]; exists {
				p.MessageCount++
				p.LastMessageAt = msg.Date.Format(time.RFC3339)
			}
		}

		// Count mentions in message body
		body := strings.ToLower(msg.Body)
		for email, p := range participantMap {
			if strings.Contains(body, strings.ToLower(email)) {
				p.MentionCount++
			}
		}
	}

	// Determine involvement level and required status
	totalMessages := len(messages)
	participants := make([]domain.ParticipantInfo, 0, len(participantMap))

	for _, p := range participantMap {
		// High involvement: sent >30% of messages or mentioned >3 times
		if totalMessages > 0 && (float64(p.MessageCount)/float64(totalMessages) > 0.3 || p.MentionCount > 3) {
			p.Involvement = domain.InvolvementHigh
			p.Required = true
		} else if p.MessageCount > 1 || p.MentionCount > 0 {
			p.Involvement = domain.InvolvementMedium
			p.Required = true
		} else {
			p.Involvement = domain.InvolvementLow
			p.Required = false
		}

		participants = append(participants, *p)
	}

	return participants
}

// InboxSummaryRequest represents a request to summarize recent emails.
type InboxSummaryRequest struct {
	Messages     []domain.Message
	ProviderName string // Optional: specific provider to use
}

// InboxSummaryResponse represents the AI summary of emails.
type InboxSummaryResponse struct {
	Summary      string          `json:"summary"`
	Categories   []EmailCategory `json:"categories"`
	ActionItems  []ActionItem    `json:"action_items"`
	Highlights   []string        `json:"highlights"`
	ProviderUsed string          `json:"provider_used"`
	TokensUsed   int             `json:"tokens_used"`
}

// EmailCategory groups emails by type.
type EmailCategory struct {
	Name     string   `json:"name"`
	Count    int      `json:"count"`
	Subjects []string `json:"subjects"`
}

// ActionItem represents an email that needs attention.
type ActionItem struct {
	Subject string `json:"subject"`
	From    string `json:"from"`
	Urgency string `json:"urgency"` // high, medium, low
	Reason  string `json:"reason"`
}

// AnalyzeInbox analyzes recent emails and returns a summary.
func (a *EmailAnalyzer) AnalyzeInbox(ctx context.Context, req *InboxSummaryRequest) (*InboxSummaryResponse, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("no messages to analyze")
	}

	// Build the prompt with email data
	prompt := a.buildInboxPrompt(req.Messages)

	// Create chat request
	chatReq := &domain.ChatRequest{
		Messages: []domain.ChatMessage{
			{
				Role:    "system",
				Content: inboxAnalysisSystemPrompt,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3, // Lower temperature for more consistent output
	}

	// Send to AI
	var resp *domain.ChatResponse
	var err error

	if req.ProviderName != "" {
		resp, err = a.llmRouter.ChatWithProvider(ctx, req.ProviderName, chatReq)
	} else {
		resp, err = a.llmRouter.Chat(ctx, chatReq)
	}

	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}

	// Parse response
	result, err := a.parseInboxResponse(resp.Content)
	if err != nil {
		// If parsing fails, return a basic response with the raw content
		return &InboxSummaryResponse{
			Summary:      resp.Content,
			Categories:   []EmailCategory{},
			ActionItems:  []ActionItem{},
			Highlights:   []string{},
			ProviderUsed: resp.Provider,
			TokensUsed:   resp.Usage.TotalTokens,
		}, nil
	}

	result.ProviderUsed = resp.Provider
	result.TokensUsed = resp.Usage.TotalTokens

	return result, nil
}

func (a *EmailAnalyzer) buildInboxPrompt(messages []domain.Message) string {
	var sb strings.Builder

	_, _ = fmt.Fprintf(&sb, "Analyze these %d emails and provide insights:\n\n", len(messages))

	for i, msg := range messages {
		_, _ = fmt.Fprintf(&sb, "--- Email %d ---\n", i+1)
		_, _ = fmt.Fprintf(&sb, "From: %s\n", formatInboxParticipants(msg.From))
		_, _ = fmt.Fprintf(&sb, "Subject: %s\n", msg.Subject)
		_, _ = fmt.Fprintf(&sb, "Date: %s\n", msg.Date.Format(time.RFC3339))

		// Use snippet for preview (cleaner than full body)
		if msg.Snippet != "" {
			_, _ = fmt.Fprintf(&sb, "Preview: %s\n", truncateStr(msg.Snippet, 200))
		}

		if msg.Unread {
			sb.WriteString("Status: UNREAD\n")
		}
		if msg.Starred {
			sb.WriteString("Status: STARRED\n")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func (a *EmailAnalyzer) parseInboxResponse(content string) (*InboxSummaryResponse, error) {
	// Try to extract JSON from the response
	content = strings.TrimSpace(content)

	// Find JSON block (may be wrapped in markdown code blocks)
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON found in response")
	}

	jsonStr := content[jsonStart : jsonEnd+1]

	var result InboxSummaryResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &result, nil
}

func formatInboxParticipants(participants []domain.EmailParticipant) string {
	if len(participants) == 0 {
		return "Unknown"
	}

	names := make([]string, 0, len(participants))
	for _, p := range participants {
		if p.Name != "" {
			names = append(names, fmt.Sprintf("%s <%s>", p.Name, p.Email))
		} else {
			names = append(names, p.Email)
		}
	}

	return strings.Join(names, ", ")
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

const inboxAnalysisSystemPrompt = `You are an email analyst. Analyze the provided emails and return a JSON response with the following structure:

{
  "summary": "A brief 2-3 sentence overview of the inbox",
  "categories": [
    {
      "name": "Category name (e.g., Work, Personal, Newsletters, Promotions)",
      "count": 3,
      "subjects": ["Subject 1", "Subject 2", "Subject 3"]
    }
  ],
  "action_items": [
    {
      "subject": "Email subject",
      "from": "sender@example.com",
      "urgency": "high|medium|low",
      "reason": "Why this needs attention"
    }
  ],
  "highlights": [
    "Key point or important information from the emails",
    "Another key insight"
  ]
}

Guidelines:
- Categories should group similar emails (Work, Personal, Newsletters, Social, Promotions, Updates)
- Action items are emails that likely need a response or action
- Urgency levels: high (time-sensitive, important), medium (should respond soon), low (informational)
- Highlights should capture 3-5 key points from across all emails
- Keep the summary concise and actionable
- Focus on what matters most to the user

Respond ONLY with valid JSON, no additional text.`

// detectUrgencyIndicators detects urgency signals in the email thread.
func (a *EmailAnalyzer) detectUrgencyIndicators(messages []domain.Message) []string {
	indicators := []string{}

	urgentKeywords := []string{
		"urgent", "asap", "immediately", "critical", "emergency",
		"deadline", "today", "tomorrow", "this week",
	}

	// Check for urgent keywords
	for _, msg := range messages {
		bodyLower := strings.ToLower(msg.Body)
		subjectLower := strings.ToLower(msg.Subject)

		for _, keyword := range urgentKeywords {
			if strings.Contains(bodyLower, keyword) || strings.Contains(subjectLower, keyword) {
				indicators = append(indicators, fmt.Sprintf("Contains urgent keyword: '%s'", keyword))
			}
		}
	}

	// Check for high message frequency
	if len(messages) > 5 {
		// Calculate time span
		if len(messages) > 1 {
			earliest := messages[0].Date
			latest := messages[len(messages)-1].Date
			duration := latest.Sub(earliest)

			if duration < 24*time.Hour && len(messages) > 3 {
				indicators = append(indicators, fmt.Sprintf("%d messages in %s (high activity)", len(messages), duration.Round(time.Hour)))
			}
		}
	}

	// Check for multiple participants (broad reach)
	participantEmails := make(map[string]bool)
	for _, msg := range messages {
		for _, from := range msg.From {
			participantEmails[from.Email] = true
		}
	}

	if len(participantEmails) > 5 {
		indicators = append(indicators, fmt.Sprintf("%d participants (broad reach)", len(participantEmails)))
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueIndicators := []string{}
	for _, indicator := range indicators {
		if !seen[indicator] {
			seen[indicator] = true
			uniqueIndicators = append(uniqueIndicators, indicator)
		}
	}

	return uniqueIndicators
}
