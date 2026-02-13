package chat

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	browserpkg "github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
)

// NewChatCmd creates the chat command.
func NewChatCmd() *cobra.Command {
	var (
		port      int
		noBrowser bool
		agentName string
		model     string
	)

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Chat with AI using your email and calendar",
		Long: `Launch an AI chat interface that can access your Nylas email, calendar, and contacts.

Uses locally installed AI agents (Claude, Codex, or Ollama) to answer questions
and perform actions on your behalf through a web-based chat interface.`,
		Example: `  # Launch chat (auto-detects best agent)
  nylas chat

  # Use a specific agent
  nylas chat --agent claude

  # Use Ollama with a specific model
  nylas chat --agent ollama --model llama2

  # Launch on custom port without browser
  nylas chat --port 8080 --no-browser`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Detect available agents
			agents := DetectAgents()
			if len(agents) == 0 {
				return fmt.Errorf("no AI agents found; install claude, codex, or ollama")
			}

			// Select agent
			var agent *Agent
			if agentName != "" {
				agent = FindAgent(agents, AgentType(agentName))
				if agent == nil {
					return fmt.Errorf("agent %q not found; available: %v", agentName, agentNames(agents))
				}
			} else {
				agent = &agents[0] // use first detected
			}

			// Override model for ollama
			if model != "" && agent.Type == AgentOllama {
				agent.Model = model
			}

			// Set up Nylas client
			nylasClient, err := common.GetNylasClient()
			if err != nil {
				return err
			}

			grantID, err := common.GetGrantID(args)
			if err != nil {
				return err
			}

			// Set up conversation storage
			chatDir := filepath.Join(config.DefaultConfigDir(), "chat", "conversations")
			memory, err := NewMemoryStore(chatDir)
			if err != nil {
				return fmt.Errorf("initialize chat storage: %w", err)
			}

			addr := fmt.Sprintf("localhost:%d", port)
			url := fmt.Sprintf("http://%s", addr)

			fmt.Printf("Starting Nylas Chat at %s\n", url)
			fmt.Printf("Agent: %s\n", agent)
			fmt.Println("Press Ctrl+C to stop")
			fmt.Println()

			if !noBrowser {
				b := browserpkg.NewDefaultBrowser()
				if err := b.Open(url); err != nil {
					fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
					fmt.Printf("Open %s manually\n", url)
				}
			}

			server := NewServer(addr, agent, agents, nylasClient, grantID, memory)
			return server.Start()
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 7367, "Port to run the server on")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")
	cmd.Flags().StringVar(&agentName, "agent", "", "AI agent to use (claude, codex, ollama)")
	cmd.Flags().StringVar(&model, "model", "", "Model name for ollama agent")

	return cmd
}

func agentNames(agents []Agent) []string {
	names := make([]string, len(agents))
	for i, a := range agents {
		names[i] = string(a.Type)
	}
	return names
}
