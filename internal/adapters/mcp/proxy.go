// Package mcp provides an MCP proxy that forwards requests to the Nylas MCP server.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/nylas/cli/internal/httputil"
	"github.com/nylas/cli/internal/ports"
)

const (
	// NylasMCPEndpointUS is the US regional MCP endpoint.
	NylasMCPEndpointUS = "https://mcp.us.nylas.com"
	// NylasMCPEndpointEU is the EU regional MCP endpoint.
	NylasMCPEndpointEU = "https://mcp.eu.nylas.com"
)

// GetMCPEndpoint returns the appropriate MCP endpoint for the given region.
func GetMCPEndpoint(region string) string {
	switch strings.ToLower(region) {
	case "eu":
		return NylasMCPEndpointEU
	default:
		return NylasMCPEndpointUS
	}
}

// rpcRequest represents a JSON-RPC request structure.
// Defined once to avoid duplicate parsing.
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Method  string `json:"method"`
	Params  struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"params"`
}

// Proxy forwards MCP requests from STDIO to the Nylas MCP server.
type Proxy struct {
	endpoint     string
	apiKey       string
	authHeader   string // Cached "Bearer <apiKey>" value
	defaultGrant string
	grantStore   ports.GrantStore
	httpClient   *http.Client
	sessionID    string
	grantTools   map[string]bool // Dynamically discovered tools that accept grant_id
	mu           sync.RWMutex
}

// NewProxy creates a new MCP proxy with the given API key and region.
func NewProxy(apiKey, region string) *Proxy {
	return &Proxy{
		endpoint:   GetMCPEndpoint(region),
		apiKey:     apiKey,
		authHeader: "Bearer " + apiKey, // Cache auth header
		httpClient: httputil.DefaultClient,
	}
}

// SetDefaultGrant sets the default grant ID to use for requests.
// This helps the MCP server know which account to use by default.
func (p *Proxy) SetDefaultGrant(grantID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultGrant = grantID
}

// SetGrantStore sets the grant store for local grant operations.
// This allows the proxy to respond to grant queries locally without
// requiring the AI to provide an email address.
func (p *Proxy) SetGrantStore(store ports.GrantStore) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.grantStore = store
}

// Run starts the proxy, reading from stdin and writing to stdout.
func (p *Proxy) Run(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read a line (JSON-RPC message)
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading stdin: %w", err)
		}

		// Skip empty lines
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Parse JSON once for all operations
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			// Not valid JSON - forward as-is, let server handle error
			response, fwdErr := p.forward(ctx, line, nil)
			if fwdErr != nil {
				errorResp := p.createErrorResponse(nil, fwdErr)
				_, _ = writer.Write(append(errorResp, '\n'))
				_ = writer.Flush()
				continue
			}
			if len(response) > 0 {
				_, _ = writer.Write(append(response, '\n'))
				_ = writer.Flush()
			}
			continue
		}

		// Try to handle locally first (for get_grant without email)
		if localResponse, handled := p.handleLocalToolCall(&req); handled {
			if len(localResponse) > 0 {
				if _, err := writer.Write(append(localResponse, '\n')); err != nil {
					return fmt.Errorf("writing local response: %w", err)
				}
				_ = writer.Flush()
			}
			continue
		}

		// Forward to Nylas MCP server
		response, err := p.forward(ctx, line, &req)
		if err != nil {
			// Write error response
			errorResp := p.createErrorResponse(&req, err)
			if _, writeErr := writer.Write(append(errorResp, '\n')); writeErr != nil {
				return fmt.Errorf("writing error response: %w", writeErr)
			}
			_ = writer.Flush()
			continue
		}

		// Write response
		if len(response) > 0 {
			if _, err := writer.Write(append(response, '\n')); err != nil {
				return fmt.Errorf("writing response: %w", err)
			}
			_ = writer.Flush()
		}
	}
}

// forward sends a request to the Nylas MCP server and returns the response.
// The parsed rpcRequest is optional - if nil, request is forwarded as-is.
func (p *Proxy) forward(ctx context.Context, request []byte, parsed *rpcRequest) ([]byte, error) {
	// Check request types that need response modification
	isToolsList := parsed != nil && parsed.Method == "tools/list"
	isInitialize := parsed != nil && parsed.Method == "initialize"

	// Inject default grant into tool calls if not specified
	request = p.injectDefaultGrant(request, parsed)

	// Normalize tool arguments (type coercion, timestamp rounding)
	request = p.normalizeToolArguments(request, parsed)

	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewReader(request))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set required headers (use cached auth header)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", p.authHeader)

	// Include session ID and default grant if we have them (read lock)
	p.mu.RLock()
	if p.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", p.sessionID)
	}
	if p.defaultGrant != "" {
		req.Header.Set("X-Nylas-Grant-Id", p.defaultGrant)
	}
	p.mu.RUnlock()

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Store session ID if provided
	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		p.mu.Lock()
		p.sessionID = sessionID
		p.mu.Unlock()
	}

	// Handle response based on content type
	contentType := resp.Header.Get("Content-Type")

	// Handle 202 Accepted (no body)
	if resp.StatusCode == http.StatusAccepted {
		return nil, nil
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	// Handle SSE stream
	if strings.HasPrefix(contentType, "text/event-stream") {
		body, err := p.readSSE(resp.Body)
		if err != nil {
			return nil, err
		}
		// Modify responses as needed
		if isToolsList {
			body = p.modifyToolsListResponse(body)
		}
		if isInitialize {
			body = p.modifyInitializeResponse(body)
		}
		return body, nil
	}

	// Handle JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	// Modify responses as needed
	if isToolsList {
		body = p.modifyToolsListResponse(body)
	}
	if isInitialize {
		body = p.modifyInitializeResponse(body)
	}

	return body, nil
}

// readSSE reads Server-Sent Events and extracts JSON-RPC messages.
func (p *Proxy) readSSE(reader io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	var responses []json.RawMessage

	for scanner.Scan() {
		line := scanner.Text()

		// SSE data lines start with "data: "
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data != "" {
				responses = append(responses, json.RawMessage(data))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading SSE: %w", err)
	}

	// Return single response or batch
	if len(responses) == 0 {
		return nil, nil
	}
	if len(responses) == 1 {
		return responses[0], nil
	}

	// Batch multiple responses
	batch, err := json.Marshal(responses)
	if err != nil {
		return nil, fmt.Errorf("marshaling batch: %w", err)
	}
	return batch, nil
}

// fallbackGrantTools is a static fallback used before the first tools/list response
// is received. Once tools/list is processed, the dynamically discovered set is used instead.
// Keep this list comprehensive to minimize the race window before dynamic discovery.
var fallbackGrantTools = map[string]bool{
	// Grants
	"get_grant": true,
	// Calendars
	"list_calendars": true,
	"get_calendar":   true,
	// Events
	"list_events":  true,
	"get_event":    true,
	"create_event": true,
	"update_event": true,
	"delete_event": true,
	// Messages
	"list_messages":  true,
	"get_message":    true,
	"update_message": true,
	"delete_message": true,
	// Threads
	"list_threads": true,
	"get_thread":   true,
	// Folders
	"list_folders":     true,
	"get_folder_by_id": true,
	"create_folder":    true,
	"update_folder":    true,
	"delete_folder":    true,
	// Drafts
	"list_drafts":        true,
	"get_draft":          true,
	"create_draft":       true,
	"update_draft":       true,
	"delete_draft":       true,
	"send_draft":         true,
	"confirm_send_draft": true,
	// Send
	"send_message": true,
	// Contacts
	"list_contacts":  true,
	"get_contact":    true,
	"create_contact": true,
	"update_contact": true,
	"delete_contact": true,
}

// toolRequiresGrant checks whether a tool accepts grant_id, using the dynamically
// discovered set from tools/list if available, falling back to the static list.
func (p *Proxy) toolRequiresGrant(toolName string) bool {
	p.mu.RLock()
	gt := p.grantTools
	p.mu.RUnlock()

	if gt != nil {
		return gt[toolName]
	}
	return fallbackGrantTools[toolName]
}

// injectDefaultGrant injects the default grant_id into tool call requests if not already specified.
// Uses the pre-parsed request if available to avoid re-parsing.
func (p *Proxy) injectDefaultGrant(request []byte, parsed *rpcRequest) []byte {
	p.mu.RLock()
	defaultGrant := p.defaultGrant
	p.mu.RUnlock()

	if defaultGrant == "" {
		return request
	}

	// Use parsed request if available, otherwise parse
	var req *rpcRequest
	if parsed != nil {
		req = parsed
	} else {
		var r rpcRequest
		if err := json.Unmarshal(request, &r); err != nil {
			return request // Not valid JSON, pass through
		}
		req = &r
	}

	// Only process tools/call requests
	if req.Method != "tools/call" {
		return request
	}

	// Only inject grant_id for tools that accept it (dynamically discovered)
	if !p.toolRequiresGrant(req.Params.Name) {
		return request
	}

	// Check if grant_id or identifier is already specified
	if req.Params.Arguments == nil {
		req.Params.Arguments = make(map[string]any)
	}

	// Don't override if already set
	if _, hasGrantID := req.Params.Arguments["grant_id"]; hasGrantID {
		return request
	}
	if _, hasIdentifier := req.Params.Arguments["identifier"]; hasIdentifier {
		return request
	}

	// Inject the default grant_id
	req.Params.Arguments["grant_id"] = defaultGrant

	// Re-marshal the request
	modified, err := json.Marshal(req)
	if err != nil {
		return request // Marshal failed, use original
	}

	return modified
}

// normalizeToolArguments fixes type mismatches and rounds timestamps before
// forwarding to the upstream server. LLMs frequently send integers where the
// schema expects strings, or send unaligned timestamps that the API rejects.
func (p *Proxy) normalizeToolArguments(request []byte, parsed *rpcRequest) []byte {
	var req *rpcRequest
	if parsed != nil {
		req = parsed
	} else {
		var r rpcRequest
		if err := json.Unmarshal(request, &r); err != nil {
			return request
		}
		req = &r
	}

	if req.Method != "tools/call" || req.Params.Arguments == nil {
		return request
	}

	var modified bool

	switch req.Params.Name {
	case "list_events":
		modified = normalizeListEventsArgs(req.Params.Arguments)
	case "availability":
		modified = normalizeAvailabilityArgs(req.Params.Arguments)
	}

	if !modified {
		return request
	}

	out, err := json.Marshal(req)
	if err != nil {
		return request
	}
	return out
}

// normalizeListEventsArgs coerces numeric start/end fields to strings inside
// get_all_query_parameters. The upstream schema expects string timestamps but
// LLMs naturally produce integers.
func normalizeListEventsArgs(args map[string]any) bool {
	params, ok := args["get_all_query_parameters"].(map[string]any)
	if !ok {
		return false
	}

	modified := false
	for _, key := range []string{"start", "end"} {
		if v, exists := params[key]; exists {
			if num, ok := toInt64(v); ok {
				params[key] = fmt.Sprintf("%d", num)
				modified = true
			}
		}
	}
	return modified
}

// normalizeAvailabilityArgs rounds start_time down and end_time up to the
// nearest 5-minute boundary. The Nylas API requires these to be multiples of
// 300 seconds.
func normalizeAvailabilityArgs(args map[string]any) bool {
	req, ok := args["availability_request"].(map[string]any)
	if !ok {
		return false
	}

	modified := false
	if v, exists := req["start_time"]; exists {
		if num, ok := toInt64(v); ok {
			rounded := roundDown5Min(num)
			if rounded != num {
				req["start_time"] = rounded
				modified = true
			}
		}
	}
	if v, exists := req["end_time"]; exists {
		if num, ok := toInt64(v); ok {
			rounded := roundUp5Min(num)
			if rounded != num {
				req["end_time"] = rounded
				modified = true
			}
		}
	}
	return modified
}

func roundDown5Min(epoch int64) int64 {
	return (epoch / 300) * 300
}

func roundUp5Min(epoch int64) int64 {
	return int64(math.Ceil(float64(epoch)/300)) * 300
}

// toInt64 extracts an integer from a JSON-decoded value. JSON numbers decode
// as float64 in map[string]any; this also handles explicit int/int64 values.
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}
