// Package calendar provides calendar-related CLI commands.
package calendar

import (
	"fmt"

	"github.com/nylas/cli/internal/adapters/ai"
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/ports"
	"github.com/spf13/cobra"
)

var llmRouter ports.LLMRouter

// NewCalendarCmd creates the calendar command group.
func NewCalendarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "calendar",
		Aliases: []string{"cal"},
		Short:   "Manage calendars and events",
		Long: `Manage calendars and events from your connected accounts.

View calendars, list events, create new events, and more.

API reference: https://developer.nylas.com/docs/v3/calendar/`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newShowCmd())
	cmd.AddCommand(newCreateCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newEventsCmd())
	cmd.AddCommand(newAvailabilityCmd())
	cmd.AddCommand(newVirtualCmd())
	cmd.AddCommand(newRecurringCmd())
	cmd.AddCommand(newFindTimeCmd())
	cmd.AddCommand(newScheduleCmd())
	cmd.AddCommand(newAICmd()) // AI command group includes: analyze, conflicts, reschedule, focus-time, adapt

	return cmd
}

func getLLMRouter() (ports.LLMRouter, error) {
	if llmRouter != nil {
		return llmRouter, nil
	}

	// Load configuration
	configStore := config.NewDefaultFileStore()
	cfg, err := configStore.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w\n\nTo configure AI, run:\n  nylas ai config set default_provider ollama\n  nylas ai config set ollama.host http://localhost:11434\n  nylas ai config set ollama.model mistral:latest", err)
	}

	// Check if AI is configured
	if cfg.AI == nil || !cfg.AI.IsConfigured() {
		return nil, fmt.Errorf("AI not configured in %s\n\nTo configure AI, run:\n  nylas ai config set default_provider ollama\n  nylas ai config set ollama.host http://localhost:11434\n  nylas ai config set ollama.model mistral:latest", configStore.Path())
	}

	// Validate the default provider is configured
	provider := cfg.AI.DefaultProvider
	if provider == "" {
		return nil, fmt.Errorf("no default AI provider set\n\nTo set a default provider, run:\n  nylas ai config set default_provider ollama")
	}

	if err := cfg.AI.ValidateForProvider(provider); err != nil {
		return nil, fmt.Errorf("AI configuration error: %w\n\nRun 'nylas ai config show' to see current configuration", err)
	}

	// Create and cache router
	llmRouter = ai.NewRouter(cfg.AI)
	return llmRouter, nil
}
