package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/nylas/cli/internal/ports"
)

// maxContentLength caps the maximum payload size for Content-Length framing (10 MB).
const maxContentLength = 10 * 1024 * 1024

// Server is a native MCP server that calls the Nylas API directly via NylasClient.
type Server struct {
	client       ports.NylasClient
	grantStore   ports.GrantStore
	defaultGrant string
	mu           sync.RWMutex
}

// NewServer creates a new native MCP server.
func NewServer(client ports.NylasClient, grantID string) *Server {
	return &Server{
		client:       client,
		defaultGrant: grantID,
	}
}

// SetGrantStore sets the grant store for local grant operations.
func (s *Server) SetGrantStore(store ports.GrantStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grantStore = store
}

// Run starts the MCP server, reading JSON-RPC from stdin and writing responses to stdout.
func (s *Server) Run(ctx context.Context) error {
	return s.RunWithIO(ctx, os.Stdin, os.Stdout)
}

// RunWithIO starts the MCP server using the provided reader and writer.
// This is useful for testing without requiring os.Stdin/os.Stdout.
func (s *Server) RunWithIO(ctx context.Context, r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)
	writer := bufio.NewWriter(w)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		payload, usedContentLength, err := readRequestPayload(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("reading stdin: %w", err)
		}

		var req Request
		if err := json.Unmarshal(payload, &req); err != nil {
			resp := errorResponse(nil, codeParseError, "parse error: "+err.Error())
			if err := writePayload(writer, resp, usedContentLength); err != nil {
				return err
			}
			continue
		}

		// Validate JSON-RPC 2.0 required fields.
		if req.JSONRPC != "2.0" || req.Method == "" {
			resp := errorResponse(req.ID, codeInvalidRequest, "invalid request: missing jsonrpc version or method")
			if err := writePayload(writer, resp, usedContentLength); err != nil {
				return err
			}
			continue
		}

		resp := s.dispatch(ctx, &req)
		if resp == nil {
			continue // Notification — no response needed
		}

		if err := writePayload(writer, resp, usedContentLength); err != nil {
			return err
		}
	}
}

// readRequestPayload reads a single MCP request payload from stdin.
// Supports standard MCP stdio framing (Content-Length) and newline-delimited
// JSON (used by the official MCP Go SDK and others).
// Returns the payload, whether Content-Length framing was used, and any error.
func readRequestPayload(reader *bufio.Reader) ([]byte, bool, error) {
	for {
		// Skip leading whitespace (CR, LF).
		peek, err := reader.Peek(1)
		if err != nil {
			return nil, false, err
		}
		if peek[0] == '\n' || peek[0] == '\r' {
			if _, err := reader.ReadByte(); err != nil {
				return nil, false, err
			}
			continue
		}

		// Newline-delimited JSON mode (official MCP Go SDK, Codex, etc.).
		if peek[0] == '{' || peek[0] == '[' {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					line = bytes.TrimSpace(line)
					if len(line) == 0 {
						return nil, false, io.EOF
					}
					// SEC-001: enforce size limit on newline-delimited payloads.
					if len(line) > maxContentLength {
						return nil, false, fmt.Errorf("payload size %d exceeds maximum %d", len(line), maxContentLength)
					}
					return line, false, nil
				}
				return nil, false, err
			}

			// SEC-001: enforce size limit on newline-delimited payloads.
			if len(line) > maxContentLength {
				return nil, false, fmt.Errorf("payload size %d exceeds maximum %d", len(line), maxContentLength)
			}

			line = bytes.TrimSpace(line)
			if len(line) == 0 {
				continue // Empty line — retry (SEC-004: no recursion).
			}
			return line, false, nil
		}

		// Content-Length framing — break out to handle below.
		break
	}

	// Content-Length framing (TypeScript MCP SDK, Claude Code, etc.).
	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, true, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, true, fmt.Errorf("invalid header line %q", line)
		}

		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 {
				return nil, true, fmt.Errorf("invalid Content-Length %q", strings.TrimSpace(value))
			}
			contentLength = n
		}
	}

	if contentLength <= 0 {
		return nil, true, fmt.Errorf("missing or invalid Content-Length header")
	}

	if contentLength > maxContentLength {
		return nil, true, fmt.Errorf("Content-Length %d exceeds maximum %d", contentLength, maxContentLength)
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, true, err
	}

	return payload, true, nil
}

// writePayload writes a response using the framing that matches the request.
func writePayload(writer *bufio.Writer, payload []byte, contentLengthMode bool) error {
	if contentLengthMode {
		return writeResponsePayload(writer, payload)
	}
	return writeNewlinePayload(writer, payload)
}

func writeResponsePayload(writer *bufio.Writer, payload []byte) error {
	if _, err := fmt.Fprintf(writer, "Content-Length: %d\r\n\r\n", len(payload)); err != nil {
		return fmt.Errorf("writing response headers: %w", err)
	}
	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("writing response body: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flushing response: %w", err)
	}
	return nil
}

// writeNewlinePayload writes a newline-delimited JSON response (official MCP Go SDK format).
func writeNewlinePayload(writer *bufio.Writer, payload []byte) error {
	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("writing response body: %w", err)
	}
	if err := writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("writing response newline: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flushing response: %w", err)
	}
	return nil
}

// dispatch routes a JSON-RPC request to the appropriate handler.
func (s *Server) dispatch(ctx context.Context, req *Request) []byte {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		return nil // Notification — no response
	case "notifications/cancelled":
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolCall(ctx, req)
	case "ping":
		return successResponse(req.ID, map[string]any{})
	default:
		return errorResponse(req.ID, codeMethodNotFound, "method not found: "+req.Method)
	}
}

// resolveGrantID returns the grant_id from tool args, falling back to the default.
func (s *Server) resolveGrantID(args map[string]any) string {
	if id := getString(args, "grant_id", ""); id != "" {
		return id
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultGrant
}
