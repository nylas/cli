package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli/common"
)

// UsageStats represents AI usage statistics
type UsageStats struct {
	Month              string  `json:"month"`
	TotalRequests      int     `json:"total_requests"`
	OllamaRequests     int     `json:"ollama_requests"`
	ClaudeRequests     int     `json:"claude_requests"`
	OpenAIRequests     int     `json:"openai_requests"`
	GroqRequests       int     `json:"groq_requests"`
	OpenRouterRequests int     `json:"openrouter_requests"`
	TotalTokens        int     `json:"total_tokens"`
	EstimatedCost      float64 `json:"estimated_cost"`
}

func newUsageCmd() *cobra.Command {
	var month string

	cmd := &cobra.Command{
		Use:   "usage",
		Short: "Show AI usage statistics",
		Long: `Display AI usage statistics including request counts, token usage,
and estimated costs.

Examples:
  # Show current month usage
  nylas ai usage

  # Show specific month usage
  nylas ai usage --month 2025-01

  # Show usage as JSON
  nylas ai usage --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to current month if not specified
			if month == "" {
				month = time.Now().Format("2006-01")
			}

			stats, err := loadUsageStats(month)
			if err != nil {
				// No stats found - return empty stats
				stats = &UsageStats{
					Month: month,
				}
			}

			if common.IsJSON(cmd) {
				data, err := json.MarshalIndent(stats, "", "  ")
				if err != nil {
					return common.WrapMarshalError("usage stats", err)
				}
				fmt.Println(string(data))
				return nil
			}

			// Display formatted output
			fmt.Printf("AI Usage for %s\n", stats.Month)
			fmt.Println()
			fmt.Printf("  Total Requests:      %d\n", stats.TotalRequests)
			fmt.Println()
			fmt.Println("  Requests by Provider:")
			fmt.Printf("    Ollama:            %d\n", stats.OllamaRequests)
			fmt.Printf("    Claude:            %d\n", stats.ClaudeRequests)
			fmt.Printf("    OpenAI:            %d\n", stats.OpenAIRequests)
			fmt.Printf("    Groq:              %d\n", stats.GroqRequests)
			fmt.Printf("    OpenRouter:        %d\n", stats.OpenRouterRequests)
			fmt.Println()
			fmt.Printf("  Total Tokens:        %d\n", stats.TotalTokens)
			fmt.Printf("  Estimated Cost:      $%.2f\n", stats.EstimatedCost)

			if stats.TotalRequests == 0 {
				fmt.Println()
				fmt.Println("ℹ️  No AI requests recorded for this month")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&month, "month", "", "Month to show usage for (YYYY-MM format)")

	return cmd
}

// loadUsageStats loads usage statistics from disk
func loadUsageStats(month string) (*UsageStats, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, common.WrapGetError("config directory", err)
	}

	statsFile := filepath.Join(configDir, "nylas", "ai-data", "usage", fmt.Sprintf("%s.json", month))

	// #nosec G304 -- statsFile constructed from UserConfigDir + "nylas/ai-data/usage/<month>.json"
	data, err := os.ReadFile(statsFile)
	if err != nil {
		return nil, err
	}

	var stats UsageStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, common.WrapDecodeError("usage stats", err)
	}

	return &stats, nil
}
