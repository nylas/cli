package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// EmailAnalyzer analyzes email threads to extract meeting context.
type EmailAnalyzer struct {
	nylasClient ports.NylasClient
	llmRouter   ports.LLMRouter
}

// NewEmailAnalyzer creates a new email analyzer.
func NewEmailAnalyzer(nylasClient ports.NylasClient, llmRouter ports.LLMRouter) *EmailAnalyzer {
	return &EmailAnalyzer{
		nylasClient: nylasClient,
		llmRouter:   llmRouter,
	}
}

// AnalyzeThread analyzes an email thread and extracts meeting context.
func (a *EmailAnalyzer) AnalyzeThread(ctx context.Context, grantID, threadID string, req *domain.EmailAnalysisRequest) (*domain.EmailThreadAnalysis, error) {
	// 1. Fetch the thread from Nylas API
	thread, err := a.nylasClient.GetThread(ctx, grantID, threadID)
	if err != nil {
		return nil, fmt.Errorf("fetch thread: %w", err)
	}

	// 2. Fetch all messages in the thread
	messages, err := a.fetchThreadMessages(ctx, grantID, threadID)
	if err != nil {
		return nil, fmt.Errorf("fetch thread messages: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("thread has no messages")
	}

	// 3. Build analysis context
	threadContext := a.buildThreadContext(thread, messages)

	// 4. Use LLM to analyze the thread
	analysis, err := a.analyzeWithLLM(ctx, threadContext, req)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis: %w", err)
	}

	// 5. Analyze participants
	participants := a.analyzeParticipants(thread, messages)
	analysis.Participants = participants

	// 6. Detect urgency indicators
	urgencyIndicators := a.detectUrgencyIndicators(messages)
	analysis.UrgencyIndicators = urgencyIndicators

	return analysis, nil
}

// fetchThreadMessages fetches all messages in a thread.
func (a *EmailAnalyzer) fetchThreadMessages(ctx context.Context, grantID, threadID string) ([]domain.Message, error) {
	params := &domain.MessageQueryParams{
		ThreadID: threadID,
		Limit:    100, // Fetch up to 100 messages in the thread
	}

	messages, err := a.nylasClient.GetMessagesWithParams(ctx, grantID, params)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

// buildThreadContext builds a string representation of the thread for LLM analysis.
func (a *EmailAnalyzer) buildThreadContext(thread *domain.Thread, messages []domain.Message) string {
	var builder strings.Builder

	_, _ = fmt.Fprintf(&builder, "Email Thread: %s\n", thread.Subject)
	_, _ = fmt.Fprintf(&builder, "Participants: %d\n", len(thread.Participants))
	_, _ = fmt.Fprintf(&builder, "Messages: %d\n\n", len(messages))

	// Add participants
	builder.WriteString("Participants:\n")
	for _, p := range thread.Participants {
		if p.Name != "" {
			_, _ = fmt.Fprintf(&builder, "- %s <%s>\n", p.Name, p.Email)
		} else {
			_, _ = fmt.Fprintf(&builder, "- %s\n", p.Email)
		}
	}
	builder.WriteString("\n")

	// Add message summaries (most recent first)
	builder.WriteString("Message Thread:\n")
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		sender := "Unknown"
		if len(msg.From) > 0 {
			if msg.From[0].Name != "" {
				sender = msg.From[0].Name
			} else {
				sender = msg.From[0].Email
			}
		}

		// Format timestamp
		timestamp := msg.Date.Format("Jan 2, 2006 3:04 PM")

		_, _ = fmt.Fprintf(&builder, "\n[%s] %s:\n", timestamp, sender)

		// Add message body (truncate if too long)
		body := msg.Body
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		builder.WriteString(body)
		builder.WriteString("\n")
	}

	return builder.String()
}

// analyzeWithLLM uses the LLM to analyze the thread and generate insights.
func (a *EmailAnalyzer) analyzeWithLLM(ctx context.Context, threadContext string, req *domain.EmailAnalysisRequest) (*domain.EmailThreadAnalysis, error) {
	// Build the analysis prompt
	prompt := a.buildAnalysisPrompt(threadContext, req)

	// Create chat request
	chatReq := &domain.ChatRequest{
		Messages: []domain.ChatMessage{
			{
				Role:    "system",
				Content: "You are an expert meeting scheduler and email analyst. Analyze email threads to extract meeting context, topics, priority, and participant involvement.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3, // Lower temperature for more factual analysis
		MaxTokens:   2000,
	}

	// Call LLM
	response, err := a.llmRouter.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("LLM chat: %w", err)
	}

	// Parse LLM response into analysis
	analysis, err := a.parseAnalysisResponse(response.Content, req)
	if err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	return analysis, nil
}

// buildAnalysisPrompt builds the prompt for LLM analysis.
