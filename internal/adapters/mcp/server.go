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

var errRecoverableRead = errors.New("recoverable request read error")

// Server is a native MCP server that calls the Nylas API directly via NylasClient.
type Server struct {
	client       ports.NylasClient
	grantStore   ports.GrantStore
	defaultGrant string
	inFlight     map[string]context.CancelFunc
	mu           sync.RWMutex
}

// NewServer creates a new native MCP server.
func NewServer(client ports.NylasClient, grantID string) *Server {
	return &Server{
		client:       client,
		defaultGrant: grantID,
		inFlight:     make(map[string]context.CancelFunc),
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
	var writeMu sync.Mutex
	var workers sync.WaitGroup
	writeErrCh := make(chan error, 1)

	writeResponse := func(payload []byte, contentLengthMode bool) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		return writePayload(writer, payload, contentLengthMode)
	}

	for {
		select {
		case err := <-writeErrCh:
			return err
		default:
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		payload, usedContentLength, err := readRequestPayload(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				workers.Wait()
				select {
				case writeErr := <-writeErrCh:
					return writeErr
				default:
				}
				return nil
			}
			if !errors.Is(err, errRecoverableRead) {
				return fmt.Errorf("reading stdin: %w", err)
			}
			resp := errorResponse(nil, codeParseError, "parse error: "+err.Error())
			if err := writeResponse(resp, usedContentLength); err != nil {
				return err
			}
			continue
		}

		trimmed := bytes.TrimSpace(payload)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			resp, err := s.handleBatch(ctx, trimmed)
			if err != nil {
				resp = errorResponse(nil, codeParseError, "parse error: "+err.Error())
			}
			if resp != nil {
				if err := writeResponse(resp, usedContentLength); err != nil {
					return err
				}
			}
			continue
		}

		var req Request
		if err := json.Unmarshal(trimmed, &req); err != nil {
			resp := errorResponse(nil, codeParseError, "parse error: "+err.Error())
			if err := writeResponse(resp, usedContentLength); err != nil {
				return err
			}
			continue
		}

		if err := validateRequest(&req); err != nil {
			resp := errorResponse(nil, codeInvalidRequest, "invalid request: "+err.Error())
			if err := writeResponse(resp, usedContentLength); err != nil {
				return err
			}
			continue
		}

		reqCopy := req
		workers.Add(1)
		go func(contentLengthMode bool) {
			defer workers.Done()
			resp := s.dispatch(ctx, &reqCopy)
			if resp == nil {
				return
			}
			if err := writeResponse(resp, contentLengthMode); err != nil {
				select {
				case writeErrCh <- err:
				default:
				}
			}
		}(usedContentLength)
	}
}

func handleBatchItemError(out *[]json.RawMessage, code int, msg string) {
	*out = append(*out, json.RawMessage(errorResponse(nil, code, msg)))
}

func (s *Server) handleBatch(ctx context.Context, payload []byte) ([]byte, error) {
	var raws []json.RawMessage
	if err := json.Unmarshal(payload, &raws); err != nil {
		return nil, err
	}
	if len(raws) == 0 {
		return errorResponse(nil, codeInvalidRequest, "invalid request: empty batch"), nil
	}

	responses := make([]json.RawMessage, 0, len(raws))
	for _, raw := range raws {
		var req Request
		if err := json.Unmarshal(raw, &req); err != nil {
			handleBatchItemError(&responses, codeInvalidRequest, "invalid request: malformed batch item")
			continue
		}
		if err := validateRequest(&req); err != nil {
			handleBatchItemError(&responses, codeInvalidRequest, "invalid request: "+err.Error())
			continue
		}
		resp := s.dispatch(ctx, &req)
		if resp == nil {
			continue
		}
		responses = append(responses, json.RawMessage(resp))
	}

	if len(responses) == 0 {
		return nil, nil
	}

	data, err := json.Marshal(responses)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// readRequestPayload reads a single MCP request payload from stdin.
// Supports standard MCP stdio framing (Content-Length) and newline-delimited
// JSON (used by the official MCP Go SDK and others).
// Returns the payload, whether Content-Length framing was used, and any error.
func readRequestPayload(reader *bufio.Reader) ([]byte, bool, error) {
	for {
		// Skip leading ASCII whitespace.
		peek, err := reader.Peek(1)
		if err != nil {
			return nil, false, err
		}
		if peek[0] == '\n' || peek[0] == '\r' || peek[0] == ' ' || peek[0] == '\t' {
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
						return nil, false, fmt.Errorf("%w: payload size %d exceeds maximum %d", errRecoverableRead, len(line), maxContentLength)
					}
					return line, false, nil
				}
				return nil, false, err
			}

			// SEC-001: enforce size limit on newline-delimited payloads.
			if len(line) > maxContentLength {
				return nil, false, fmt.Errorf("%w: payload size %d exceeds maximum %d", errRecoverableRead, len(line), maxContentLength)
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
			return nil, true, fmt.Errorf("%w: invalid header line %q", errRecoverableRead, line)
		}

		if strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
			n, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || n < 0 {
				return nil, true, fmt.Errorf("%w: invalid Content-Length %q", errRecoverableRead, strings.TrimSpace(value))
			}
			contentLength = n
		}
	}

	if contentLength <= 0 {
		return nil, true, fmt.Errorf("%w: missing or invalid Content-Length header", errRecoverableRead)
	}

	if contentLength > maxContentLength {
		return nil, true, fmt.Errorf("%w: Content-Length %d exceeds maximum %d", errRecoverableRead, contentLength, maxContentLength)
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

func isValidID(id any) bool {
	switch id.(type) {
	case nil, float64, string:
		return true
	default:
		return false
	}
}

func validateRequest(req *Request) error {
	if req.JSONRPC != "2.0" {
		return errors.New("missing jsonrpc version")
	}
	if req.Method == "" {
		return errors.New("missing method")
	}
	if !isValidID(req.ID) {
		return errors.New("invalid id type")
	}
	return nil
}

func idKey(id any) (string, bool) {
	if id == nil {
		return "", false
	}
	b, err := json.Marshal(id)
	if err != nil {
		return "", false
	}
	return string(b), true
}

func (s *Server) registerInFlight(id any, cancel context.CancelFunc) {
	key, ok := idKey(id)
	if !ok {
		return
	}
	s.mu.Lock()
	if s.inFlight == nil {
		s.inFlight = make(map[string]context.CancelFunc)
	}
	s.inFlight[key] = cancel
	s.mu.Unlock()
}

func (s *Server) unregisterInFlight(id any) {
	key, ok := idKey(id)
	if !ok {
		return
	}
	s.mu.Lock()
	delete(s.inFlight, key)
	s.mu.Unlock()
}

func (s *Server) cancelInFlight(id any) {
	key, ok := idKey(id)
	if !ok {
		return
	}
	s.mu.Lock()
	cancel, ok := s.inFlight[key]
	if ok {
		delete(s.inFlight, key)
	}
	s.mu.Unlock()
	if ok {
		cancel()
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
		var params struct {
			RequestID any `json:"requestId"`
		}
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &params)
		}
		s.cancelInFlight(params.RequestID)
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		callCtx, cancel := context.WithCancel(ctx)
		s.registerInFlight(req.ID, cancel)
		defer s.unregisterInFlight(req.ID)
		defer cancel()
		return s.handleToolCall(callCtx, req)
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
	defaultGrant := s.defaultGrant
	store := s.grantStore
	s.mu.RUnlock()
	if defaultGrant != "" {
		return defaultGrant
	}

	if store != nil {
		if grantID, err := store.GetDefaultGrant(); err == nil && grantID != "" {
			s.mu.Lock()
			if s.defaultGrant == "" {
				s.defaultGrant = grantID
			}
			s.mu.Unlock()
			return grantID
		}
	}

	return ""
}
