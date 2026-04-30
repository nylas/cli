package air

import (
	"encoding/json"
	"maps"
	"net/http"
	"strings"
	"sync"

	"github.com/nylas/cli/internal/httputil"
)

// AIConfig represents AI provider configuration
type AIConfig struct {
	Provider    string            `json:"provider"` // claude, openai, ollama, groq
	Model       string            `json:"model"`    // claude-3-opus, gpt-4, etc.
	APIKey      string            `json:"apiKey,omitempty"`
	BaseURL     string            `json:"baseUrl,omitempty"`
	MaxTokens   int               `json:"maxTokens"`
	Temperature float64           `json:"temperature"`
	TaskModels  map[string]string `json:"taskModels"`  // task -> model mapping
	UsageBudget float64           `json:"usageBudget"` // Monthly budget in USD
	UsageSpent  float64           `json:"usageSpent"`  // Current month spend
	Enabled     bool              `json:"enabled"`
}

// AIUsageStats represents AI usage statistics
type AIUsageStats struct {
	TotalRequests  int            `json:"totalRequests"`
	TotalTokens    int            `json:"totalTokens"`
	TotalCost      float64        `json:"totalCost"`
	RequestsByTask map[string]int `json:"requestsByTask"`
	TokensByTask   map[string]int `json:"tokensByTask"`
}

// aiConfigStore holds AI configuration
type aiConfigStore struct {
	config *AIConfig
	stats  *AIUsageStats
	mu     sync.RWMutex
}

var aiStore = &aiConfigStore{
	config: &AIConfig{
		Provider:    "claude",
		Model:       "claude-3-haiku-20240307",
		MaxTokens:   1024,
		Temperature: 0.7,
		TaskModels: map[string]string{
			"summarize":     "claude-3-haiku-20240307",
			"smart_reply":   "claude-3-haiku-20240307",
			"smart_compose": "claude-3-haiku-20240307",
			"categorize":    "claude-3-haiku-20240307",
		},
		UsageBudget: 50.0,
		UsageSpent:  0.0,
		Enabled:     true,
	},
	stats: &AIUsageStats{
		RequestsByTask: make(map[string]int),
		TokensByTask:   make(map[string]int),
	},
}

// handleAIConfigRoute dispatches AI config requests by method
func (s *Server) handleAIConfigRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetAIConfig(w, r)
	case http.MethodPut, http.MethodPost:
		s.handleUpdateAIConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetAIConfig returns current AI configuration
func (s *Server) handleGetAIConfig(w http.ResponseWriter, r *http.Request) {
	aiStore.mu.RLock()
	defer aiStore.mu.RUnlock()

	// Mask API key for security
	config := *aiStore.config
	if config.APIKey != "" {
		config.APIKey = "***" + config.APIKey[max(0, len(config.APIKey)-4):]
	}

	httputil.WriteJSON(w, http.StatusOK, config)
}

// handleUpdateAIConfig updates AI configuration
func (s *Server) handleUpdateAIConfig(w http.ResponseWriter, r *http.Request) {
	var req AIConfig
	if err := json.NewDecoder(limitedBody(w, r)).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	aiStore.mu.Lock()
	defer aiStore.mu.Unlock()

	// Update fields
	if req.Provider != "" {
		aiStore.config.Provider = req.Provider
	}
	if req.Model != "" {
		aiStore.config.Model = req.Model
	}
	// Saved value is masked for the read path as `***` + last 4 chars; never
	// overwrite the real key with a masked value that came back via PUT.
	if req.APIKey != "" && !strings.HasPrefix(req.APIKey, "***") {
		aiStore.config.APIKey = req.APIKey
	}
	if req.BaseURL != "" {
		aiStore.config.BaseURL = req.BaseURL
	}
	if req.MaxTokens > 0 {
		aiStore.config.MaxTokens = req.MaxTokens
	}
	if req.Temperature >= 0 {
		aiStore.config.Temperature = req.Temperature
	}
	if req.TaskModels != nil {
		maps.Copy(aiStore.config.TaskModels, req.TaskModels)
	}
	if req.UsageBudget > 0 {
		aiStore.config.UsageBudget = req.UsageBudget
	}
	aiStore.config.Enabled = req.Enabled

	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleTestAIConnection tests the AI provider connection
func (s *Server) handleTestAIConnection(w http.ResponseWriter, r *http.Request) {
	aiStore.mu.RLock()
	config := aiStore.config
	aiStore.mu.RUnlock()

	// Simulate connection test
	result := map[string]any{
		"success":  true,
		"provider": config.Provider,
		"model":    config.Model,
		"message":  "Connection successful",
	}

	if config.APIKey == "" && config.Provider != "ollama" {
		result["success"] = false
		result["message"] = "API key required"
	}

	httputil.WriteJSON(w, http.StatusOK, result)
}

// handleGetAIUsage returns AI usage statistics
func (s *Server) handleGetAIUsage(w http.ResponseWriter, r *http.Request) {
	aiStore.mu.RLock()
	defer aiStore.mu.RUnlock()

	// Guard against zero budget: division would yield +Inf, which encoding/json
	// refuses to marshal and would surface as a 500.
	var percentUsed float64
	if aiStore.config.UsageBudget > 0 {
		percentUsed = (aiStore.config.UsageSpent / aiStore.config.UsageBudget) * 100
	}

	response := map[string]any{
		"stats":       aiStore.stats,
		"budget":      aiStore.config.UsageBudget,
		"spent":       aiStore.config.UsageSpent,
		"remaining":   aiStore.config.UsageBudget - aiStore.config.UsageSpent,
		"percentUsed": percentUsed,
	}

	httputil.WriteJSON(w, http.StatusOK, response)
}

// GetAIProviders returns available AI providers
func (s *Server) handleGetAIProviders(w http.ResponseWriter, r *http.Request) {
	providers := []map[string]any{
		{
			"id":          "claude",
			"name":        "Anthropic Claude",
			"models":      []string{"claude-3-opus-20240229", "claude-3-sonnet-20240229", "claude-3-haiku-20240307"},
			"requiresKey": true,
		},
		{
			"id":          "openai",
			"name":        "OpenAI",
			"models":      []string{"gpt-4-turbo", "gpt-4", "gpt-3.5-turbo"},
			"requiresKey": true,
		},
		{
			"id":          "ollama",
			"name":        "Ollama (Local)",
			"models":      []string{"llama2", "mistral", "codellama"},
			"requiresKey": false,
		},
		{
			"id":          "groq",
			"name":        "Groq",
			"models":      []string{"mixtral-8x7b-32768", "llama2-70b-4096"},
			"requiresKey": true,
		},
	}

	httputil.WriteJSON(w, http.StatusOK, providers)
}

// RecordAIUsage records AI usage for a task
func RecordAIUsage(task string, tokens int, cost float64) {
	aiStore.mu.Lock()
	defer aiStore.mu.Unlock()

	aiStore.stats.TotalRequests++
	aiStore.stats.TotalTokens += tokens
	aiStore.stats.TotalCost += cost
	aiStore.stats.RequestsByTask[task]++
	aiStore.stats.TokensByTask[task] += tokens
	aiStore.config.UsageSpent += cost
}
