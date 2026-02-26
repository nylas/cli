package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
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

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading stdin: %w", err)
		}

		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp := errorResponse(nil, codeParseError, "parse error: "+err.Error())
			_, _ = writer.Write(append(resp, '\n'))
			_ = writer.Flush()
			continue
		}

		resp := s.dispatch(ctx, &req)
		if resp == nil {
			continue // Notification — no response needed
		}

		if _, err := writer.Write(append(resp, '\n')); err != nil {
			return fmt.Errorf("writing response: %w", err)
		}
		_ = writer.Flush()
	}
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
