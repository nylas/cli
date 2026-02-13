package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// TokenCallback is called for each token received during streaming.
type TokenCallback func(token string)

// SupportsStreaming returns true if the agent supports token-by-token streaming.
func (a *Agent) SupportsStreaming() bool {
	return a.Type == AgentClaude || a.Type == AgentOllama
}

// RunStreaming executes the agent with streaming output.
// The onToken callback is called for each text chunk received.
// Returns the complete response text and any error.
func (a *Agent) RunStreaming(ctx context.Context, prompt string, onToken TokenCallback) (string, error) {
	switch a.Type {
	case AgentClaude:
		return a.streamClaude(ctx, prompt, onToken)
	case AgentOllama:
		return a.streamOllama(ctx, prompt, onToken)
	default:
		// Fallback: run non-streaming, emit full response as single token
		return a.fallbackStream(ctx, prompt, onToken)
	}
}

// streamClaude streams tokens from Claude using stream-json output format.
// Each line is a JSON object; we extract text from content_block_delta events.
func (a *Agent) streamClaude(ctx context.Context, prompt string, onToken TokenCallback) (string, error) {
	cmd := exec.CommandContext(ctx, a.Path, "-p", "--verbose", "--output-format", "stream-json", prompt)
	cmd.Env = cleanEnv()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("claude stream pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("claude stream start: %w", err)
	}

	var full strings.Builder
	var resultText string // fallback from result event
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // allow large lines

	for scanner.Scan() {
		line := scanner.Bytes()

		// Try streaming token first
		token := parseClaudeStreamLine(line)
		if token != "" {
			full.WriteString(token)
			if onToken != nil {
				onToken(token)
			}
			continue
		}

		// Check for result event (contains full response as fallback)
		if r := parseClaudeResultLine(line); r != "" {
			resultText = r
		}
	}

	if err := cmd.Wait(); err != nil {
		// If we got output, return it despite the error
		if full.Len() > 0 {
			return full.String(), nil
		}
		if resultText != "" {
			return resultText, nil
		}
		return "", fmt.Errorf("claude stream error: %w: %s", err, stderr.String())
	}

	// Prefer streamed tokens; fall back to result event
	if full.Len() > 0 {
		return full.String(), nil
	}
	if resultText != "" {
		return resultText, nil
	}

	return "", nil
}

// claudeStreamEvent represents a Claude Code CLI stream-json line.
// The CLI wraps Anthropic API events in a stream_event envelope:
//
//	{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"..."}}}
//
// Result events contain the full response:
//
//	{"type":"result","result":"full text","subtype":"success"}
type claudeStreamEvent struct {
	Type string `json:"type"`
	// stream_event wraps the inner API event
	Event struct {
		Type  string `json:"type"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	} `json:"event"`
	// result event contains the full response text
	Result string `json:"result"`
}

// parseClaudeStreamLine extracts text tokens from a Claude stream-json line.
func parseClaudeStreamLine(line []byte) string {
	if len(line) == 0 {
		return ""
	}

	var event claudeStreamEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return ""
	}

	// stream_event envelope with nested content_block_delta
	if event.Type == "stream_event" && event.Event.Type == "content_block_delta" {
		return event.Event.Delta.Text
	}

	return ""
}

// parseClaudeResultLine extracts the full response from a result event.
func parseClaudeResultLine(line []byte) string {
	if len(line) == 0 {
		return ""
	}

	var event claudeStreamEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return ""
	}

	if event.Type == "result" && event.Result != "" {
		return event.Result
	}

	return ""
}

// streamOllama streams tokens from Ollama by reading stdout chunks.
func (a *Agent) streamOllama(ctx context.Context, prompt string, onToken TokenCallback) (string, error) {
	model := a.Model
	if model == "" {
		model = "mistral"
	}

	cmd := exec.CommandContext(ctx, a.Path, "run", model)
	cmd.Env = cleanEnv()
	cmd.Stdin = strings.NewReader(prompt)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("ollama stream pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("ollama stream start: %w", err)
	}

	var full strings.Builder
	buf := make([]byte, 256)

	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			token := string(buf[:n])
			full.WriteString(token)
			if onToken != nil {
				onToken(token)
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				// Non-EOF error, but if we have content, return it
				if full.Len() > 0 {
					break
				}
				return "", fmt.Errorf("ollama read: %w", readErr)
			}
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		if full.Len() > 0 {
			return strings.TrimSpace(full.String()), nil
		}
		return "", fmt.Errorf("ollama stream error: %w: %s", err, stderr.String())
	}

	return strings.TrimSpace(full.String()), nil
}

// fallbackStream runs the agent non-streaming and emits the full response as one token.
func (a *Agent) fallbackStream(ctx context.Context, prompt string, onToken TokenCallback) (string, error) {
	response, err := a.Run(ctx, prompt)
	if err != nil {
		return "", err
	}

	if onToken != nil && response != "" {
		onToken(response)
	}

	return response, nil
}
