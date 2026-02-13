// Package chat provides an AI chat interface using locally installed CLI agents.
package chat

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// AgentType represents a supported AI agent.
type AgentType string

const (
	AgentClaude AgentType = "claude"
	AgentCodex  AgentType = "codex"
	AgentOllama AgentType = "ollama"
)

// Agent represents a detected AI agent on the system.
type Agent struct {
	Type    AgentType `json:"type"`
	Path    string    `json:"path"`
	Model   string    `json:"model,omitempty"`   // for ollama: model name
	Version string    `json:"version,omitempty"` // detected version
}

// DetectAgents scans the system for installed AI agents.
// It checks for claude, codex, and ollama in $PATH.
func DetectAgents() []Agent {
	var agents []Agent

	checks := []struct {
		name        AgentType
		binary      string
		versionArgs []string
	}{
		{AgentClaude, "claude", []string{"--version"}},
		{AgentCodex, "codex", []string{"--version"}},
		{AgentOllama, "ollama", []string{"--version"}},
	}

	for _, check := range checks {
		path, err := exec.LookPath(check.binary)
		if err != nil {
			continue
		}

		agent := Agent{
			Type: check.name,
			Path: path,
		}

		// Try to get version
		if len(check.versionArgs) > 0 {
			out, err := exec.Command(path, check.versionArgs...).Output()
			if err == nil {
				agent.Version = strings.TrimSpace(string(out))
			}
		}

		// Default model for ollama
		if check.name == AgentOllama {
			agent.Model = "mistral"
		}

		agents = append(agents, agent)
	}

	return agents
}

// FindAgent returns the first agent matching the given type, or nil.
func FindAgent(agents []Agent, agentType AgentType) *Agent {
	for i := range agents {
		if agents[i].Type == agentType {
			return &agents[i]
		}
	}
	return nil
}

// Run executes the agent with the given prompt and returns the response.
// Each agent type has a different invocation pattern:
//   - claude: claude -p --output-format text "prompt"
//   - codex: codex exec "prompt"
//   - ollama: echo "prompt" | ollama run <model>
func (a *Agent) Run(ctx context.Context, prompt string) (string, error) {
	switch a.Type {
	case AgentClaude:
		return a.runClaude(ctx, prompt)
	case AgentCodex:
		return a.runCodex(ctx, prompt)
	case AgentOllama:
		return a.runOllama(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported agent type: %s", a.Type)
	}
}

// cleanEnv returns the current environment with nesting-detection vars removed
// so agent subprocesses don't refuse to start inside our server process.
func cleanEnv() []string {
	skip := map[string]bool{
		"CLAUDECODE":    true,
		"CLAUDE_CODE":   true,
		"CODEX_SANDBOX": true,
		"CODEX_ENV":     true,
		"INSIDE_CODEX":  true,
	}

	var env []string
	for _, e := range os.Environ() {
		key, _, _ := strings.Cut(e, "=")
		if !skip[key] {
			env = append(env, e)
		}
	}
	return env
}

func (a *Agent) runClaude(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, a.Path, "-p", "--output-format", "text", prompt)
	cmd.Env = cleanEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude error: %w: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (a *Agent) runCodex(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, a.Path, "exec", prompt)
	cmd.Env = cleanEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("codex error: %w: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (a *Agent) runOllama(ctx context.Context, prompt string) (string, error) {
	model := a.Model
	if model == "" {
		model = "mistral"
	}

	cmd := exec.CommandContext(ctx, a.Path, "run", model)
	cmd.Env = cleanEnv()
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ollama error: %w: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// String returns a human-readable description of the agent.
func (a *Agent) String() string {
	s := string(a.Type)
	if a.Model != "" {
		s += " (" + a.Model + ")"
	}
	if a.Version != "" {
		s += " " + a.Version
	}
	return s
}
