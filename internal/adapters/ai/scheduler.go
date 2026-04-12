package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// AIScheduler provides AI-powered scheduling functionality.
type AIScheduler struct {
	router       ports.LLMRouter
	nylasClient  ports.NylasClient
	providerName string
}

// NewAIScheduler creates a new AI scheduler.
func NewAIScheduler(router ports.LLMRouter, nylasClient ports.NylasClient, providerName string) *AIScheduler {
	return &AIScheduler{
		router:       router,
		nylasClient:  nylasClient,
		providerName: providerName,
	}
}

// ScheduleRequest represents a natural language scheduling request.
type ScheduleRequest struct {
	Query        string // Natural language request
	GrantID      string // User's grant ID
	UserTimezone string // User's timezone
	MaxOptions   int    // Maximum number of options to return
}

// ScheduleOption represents a suggested meeting time option.
type ScheduleOption struct {
	Rank         int                        `json:"rank"`
	Score        int                        `json:"score"`      // 0-100
	StartTime    time.Time                  `json:"start_time"` // Proposed start time
	EndTime      time.Time                  `json:"end_time"`   // Proposed end time
	Timezone     string                     `json:"timezone"`
	Reasoning    string                     `json:"reasoning"` // AI explanation
	Warnings     []string                   `json:"warnings"`
	Participants map[string]ParticipantTime `json:"participants"` // Participant views
}

// ParticipantTime shows the meeting time from a participant's perspective.
type ParticipantTime struct {
	Email     string    `json:"email"`
	Timezone  string    `json:"timezone"`
	LocalTime time.Time `json:"local_time"`
	TimeDesc  string    `json:"time_desc"` // e.g., "9:00 AM - 10:00 AM EST"
	Notes     string    `json:"notes"`     // e.g., "Morning", "End of day"
}

// ScheduleResponse contains AI-suggested meeting options.
type ScheduleResponse struct {
	Options      []ScheduleOption `json:"options"`
	Analysis     string           `json:"analysis"`      // AI's overall analysis
	ProviderUsed string           `json:"provider_used"` // Which LLM was used
	TokensUsed   int              `json:"tokens_used"`
}

// Schedule processes a natural language scheduling request and returns suggested options.
func (s *AIScheduler) Schedule(ctx context.Context, req *ScheduleRequest) (*ScheduleResponse, error) {
	// Build the system prompt
	systemPrompt := s.buildSystemPrompt(req)

	// Build the user query
	userQuery := s.buildUserQuery(req)

	// Prepare chat request with tools
	chatReq := &domain.ChatRequest{
		Messages: []domain.ChatMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userQuery,
			},
		},
		Temperature: 0.7, // Balance between creativity and consistency
		MaxTokens:   2000,
	}

	// Get scheduling tools
	tools := GetSchedulingTools()

	// Call LLM with provider selection and tools
	var chatResp *domain.ChatResponse
	var err error

	// Get the provider to use
	var provider ports.LLMProvider
	if s.providerName != "" {
		provider, err = s.router.GetProvider(s.providerName)
	} else {
		provider, err = s.router.GetProvider("")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get provider: %w", err)
	}

	// Call with tools for function calling
	chatResp, err = provider.ChatWithTools(ctx, chatReq, tools)

	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Process function calls if any
	if len(chatResp.ToolCalls) > 0 {
		// Execute tool calls and get results
		toolResults, err := s.executeToolCalls(ctx, chatResp.ToolCalls, req)
		if err != nil {
			return nil, fmt.Errorf("tool execution failed: %w", err)
		}

		// Build follow-up request with tool results
		chatReq.Messages = append(chatReq.Messages,
			domain.ChatMessage{
				Role:    "assistant",
				Content: chatResp.Content,
			},
		)

		// Add tool results
		for _, result := range toolResults {
			chatReq.Messages = append(chatReq.Messages, domain.ChatMessage{
				Role:    "tool",
				Content: result,
			})
		}

		// Get final response with tool results
		chatResp, err = provider.Chat(ctx, chatReq)
		if err != nil {
			return nil, fmt.Errorf("follow-up LLM request failed: %w", err)
		}
	}

	// Parse AI response into structured options
	options, err := s.parseScheduleOptions(chatResp.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	response := &ScheduleResponse{
		Options:      options,
		Analysis:     chatResp.Content,
		ProviderUsed: chatResp.Provider,
		TokensUsed:   chatResp.Usage.TotalTokens,
	}

	return response, nil
}

// buildSystemPrompt creates the system prompt for the AI scheduler.
func (s *AIScheduler) buildSystemPrompt(req *ScheduleRequest) string {
	return fmt.Sprintf(`You are an expert AI scheduling assistant with deep knowledge of timezone management, working hours across cultures, and meeting optimization.

Your task is to help schedule meetings by:
1. Understanding natural language scheduling requests
2. Considering participant timezones and working hours
3. Avoiding DST transition issues
4. Providing 3 ranked time options with clear explanations
5. Using available tools to check calendars and validate times

Current context:
- User timezone: %s
- Current time: %s

Guidelines:
- ALWAYS use IANA timezone IDs (e.g., "America/Los_Angeles"), NEVER abbreviations (PST, EST)
- Check for DST transitions when scheduling near spring forward / fall back dates
- Prioritize working hours (9 AM - 5 PM) unless explicitly told otherwise
- Consider timezone fairness for international teams
- Provide clear reasoning for each suggestion
- Use the available tools to gather information before making suggestions

Respond with structured JSON containing your top 3 meeting time options, each with:
- rank (1-3)
- score (0-100)
- start_time (ISO 8601)
- end_time (ISO 8601)
- timezone (IANA ID)
- reasoning (why this time is good)
- warnings (any concerns about this time)
- participants (how this time looks for each participant)`,
		req.UserTimezone,
		time.Now().Format(time.RFC3339),
	)
}

// buildUserQuery creates the user query from the request.
func (s *AIScheduler) buildUserQuery(req *ScheduleRequest) string {
	return fmt.Sprintf(`Please help me schedule: %s

Provide your top 3 recommended meeting times with detailed explanations.`, req.Query)
}

// executeToolCalls executes the function calls requested by the LLM.
func (s *AIScheduler) executeToolCalls(ctx context.Context, toolCalls []domain.ToolCall, req *ScheduleRequest) ([]string, error) {
	results := make([]string, len(toolCalls))

	for i, call := range toolCalls {
		result, err := s.executeTool(ctx, call.Function, call.Arguments, req)
		if err != nil {
			results[i] = fmt.Sprintf("Error executing %s: %v", call.Function, err)
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// executeTool executes a single tool call.
func (s *AIScheduler) executeTool(ctx context.Context, function string, args map[string]any, req *ScheduleRequest) (string, error) {
	switch function {
	case "findMeetingTime":
		return s.findMeetingTime(ctx, args, req)
	case "checkDST":
		return s.checkDST(ctx, args)
	case "validateWorkingHours":
		return s.validateWorkingHours(ctx, args)
	case "createEvent":
		return s.createEvent(ctx, args, req)
	case "getAvailability":
		return s.getAvailability(ctx, args, req)
	case "getTimezoneInfo":
		return s.getTimezoneInfo(ctx, args, req)
	default:
		return "", fmt.Errorf("unknown function: %s", function)
	}
}

// Tool implementation methods

func (s *AIScheduler) findMeetingTime(ctx context.Context, args map[string]any, req *ScheduleRequest) (string, error) {
	return s.runFindMeetingTime(ctx, args, req)
}

func (s *AIScheduler) checkDST(ctx context.Context, args map[string]any) (string, error) {
	return s.runCheckDST(ctx, args)
}

func (s *AIScheduler) validateWorkingHours(ctx context.Context, args map[string]any) (string, error) {
	return s.runValidateWorkingHours(ctx, args)
}

func (s *AIScheduler) createEvent(ctx context.Context, args map[string]any, req *ScheduleRequest) (string, error) {
	return s.runCreateEvent(ctx, args, req)
}

func (s *AIScheduler) getAvailability(ctx context.Context, args map[string]any, req *ScheduleRequest) (string, error) {
	return s.runGetAvailability(ctx, args, req)
}

func (s *AIScheduler) getTimezoneInfo(ctx context.Context, args map[string]any, req *ScheduleRequest) (string, error) {
	return s.runGetTimezoneInfo(ctx, args, req)
}

// parseScheduleOptions parses the AI response into structured options.
func (s *AIScheduler) parseScheduleOptions(content string) ([]ScheduleOption, error) {
	// Try to extract JSON from the response
	var options []ScheduleOption

	// Look for JSON array in the response
	start := strings.Index(content, "[")
	end := strings.LastIndex(content, "]")

	if start >= 0 && end > start {
		jsonStr := content[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &options); err != nil {
			// If JSON parsing fails, create a simple fallback option
			return s.createFallbackOptions(), nil
		}
		return options, nil
	}

	// If no JSON found, create fallback options
	return s.createFallbackOptions(), nil
}

// createFallbackOptions creates simple fallback options when AI response can't be parsed.
func (s *AIScheduler) createFallbackOptions() []ScheduleOption {
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	// Tomorrow at 2 PM
	option1Start := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 14, 0, 0, 0, tomorrow.Location())

	return []ScheduleOption{
		{
			Rank:      1,
			Score:     85,
			StartTime: option1Start,
			EndTime:   option1Start.Add(30 * time.Minute),
			Timezone:  tomorrow.Location().String(),
			Reasoning: "Tomorrow afternoon - good for most timezones",
			Warnings:  []string{},
		},
	}
}
