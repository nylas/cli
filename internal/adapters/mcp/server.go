package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/nylas/cli/internal/ports"
)

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
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		payload, err := readRequestPayload(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading stdin: %w", err)
		}

		var req Request
		if err := json.Unmarshal(payload, &req); err != nil {
			resp := errorResponse(nil, codeParseError, "parse error: "+err.Error())
			if err := writeResponsePayload(writer, resp); err != nil {
				return err
			}
			continue
		}

		resp := s.dispatch(ctx, &req)
		if resp == nil {
			continue // Notification — no response needed
		}

		if err := writeResponsePayload(writer, resp); err != nil {
			return err
		}
	}
}

// readRequestPayload reads a single MCP request payload from stdin.
// Supports standard MCP stdio framing (Content-Length) and newline-delimited
// JSON as a compatibility fallback.
func readRequestPayload(reader *bufio.Reader) ([]byte, error) {
	for {
		peek, err := reader.Peek(1)
		if err != nil {
			return nil, err
		}
		if peek[0] == '\n' || peek[0] == '\r' {
			if _, err := reader.ReadByte(); err != nil {
				return nil, err
			}
			continue
		}
		break
	}

	peek, err := reader.Peek(1)
	if err != nil {
		return nil, err
	}

	// Backward-compatible newline-delimited JSON mode.
	if peek[0] == '{' || peek[0] == '[' {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				line = bytes.TrimSpace(line)
				if len(line) == 0 {
					return nil, io.EOF
				}
				return line, nil
			}
			return nil, err
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			return readRequestPayload(reader)
		}
		return line, nil
	}

	contentLength := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid header line %q", line)
		}

		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid Content-Length %q", strings.TrimSpace(value))
			}
			contentLength = n
		}
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("missing or invalid Content-Length header")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}

	return payload, nil
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
