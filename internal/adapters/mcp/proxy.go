// Package mcp provides an MCP proxy that forwards requests to the Nylas MCP server.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/httputil"
	"github.com/nylas/cli/internal/ports"
)

const (
	// NylasMCPEndpointUS is the US regional MCP endpoint.
	NylasMCPEndpointUS = domain.MCPResourceUS
	// NylasMCPEndpointEU is the EU regional MCP endpoint.
	NylasMCPEndpointEU = domain.MCPResourceEU
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

	// notification is set when the message had no "id" member. ID cannot
	// say so (absent and null both decode to nil), and re-marshalling a
	// notification with "id": null would turn it into a request.
	notification bool
}

// MarshalJSON leaves "id" out of a notification; see rpcRequest.notification.
func (r rpcRequest) MarshalJSON() ([]byte, error) {
	type plain rpcRequest // drops this method, so Marshal does not recurse
	if !r.notification {
		return json.Marshal(plain(r))
	}
	return json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  any    `json:"params"`
	}{r.JSONRPC, r.Method, r.Params})
}

// mcpHTTPClient sends every request with its credential (an API key or an
// OAuth access token), so it follows no redirect.
var mcpHTTPClient = httputil.NewNoRedirectClient(httputil.DefaultClientTimeout)

// Proxy forwards MCP requests from STDIO to the Nylas MCP server.
type Proxy struct {
	// endpoint is the regional default, used when a credential names no
	// server of its own. An OAuth credential always names one.
	endpoint     string
	creds        ports.MCPCredentialSource
	oauth        bool
	defaultGrant string
	grantStore   ports.GrantStore
	httpClient   *http.Client
	// protocolVersion is what the server answered initialize with, sent back
	// as Mcp-Protocol-Version on every later request. There is no session:
	// the hosted server is stateless and issues no Mcp-Session-Id.
	protocolVersion string
	grantTools      map[string]bool // Dynamically discovered tools that accept grant_id
	mu              sync.RWMutex
}

// NewProxy creates a new MCP proxy with the given API key and region.
func NewProxy(apiKey, region string) *Proxy {
	return &Proxy{
		endpoint:   GetMCPEndpoint(region),
		creds:      apiKeyCredentials{apiKey: apiKey},
		httpClient: mcpHTTPClient,
	}
}

// NewOAuthProxy creates an MCP proxy that authenticates with an OAuth
// session. The source is asked for a credential before every request, so a
// fifteen-minute access token is refreshed as it ages rather than failing
// the first request after it expires, and the MCP host is whatever the
// credential names — the audience the token was issued for.
func NewOAuthProxy(creds ports.MCPCredentialSource) *Proxy {
	return &Proxy{
		creds:      creds,
		oauth:      true,
		httpClient: mcpHTTPClient,
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
	return p.serve(ctx, os.Stdin, os.Stdout)
}

// serve runs the proxy loop over the given streams.
func (p *Proxy) serve(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	writer := bufio.NewWriter(out)

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

		notification := isNotification(line)

		// Parse JSON once for all operations
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			// Not a single request (invalid JSON, a batch): forward as-is
			// and let the server handle it.
			response, fwdErr := p.forward(ctx, line, nil)
			if fwdErr != nil {
				if notification {
					log.Printf("mcp: forwarding a notification failed: %q", fwdErr.Error())
					continue
				}
				errorResp := p.createErrorResponse(nil, fwdErr)
				_, _ = writer.Write(append(errorResp, '\n'))
				_ = writer.Flush()
				continue
			}
			if len(response) > 0 && !notification {
				_, _ = writer.Write(append(response, '\n'))
				_ = writer.Flush()
			}
			continue
		}
		req.notification = notification

		// Try to handle locally first (for get_grant without email)
		if localResponse, handled := p.handleLocalToolCall(&req); handled {
			if len(localResponse) > 0 && !notification {
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
			// A notification must never be answered, not even with an error
			// (JSON-RPC 2.0 §4.1), so its failure goes to the log instead.
			// Both are quoted: they carry text from the client and the
			// server, which must not reach a terminal as escape sequences.
			if notification {
				log.Printf("mcp: forwarding %q notification failed: %q", req.Method, err.Error())
				continue
			}
			// Write error response
			errorResp := p.createErrorResponse(&req, err)
			if _, writeErr := writer.Write(append(errorResp, '\n')); writeErr != nil {
				return fmt.Errorf("writing error response: %w", writeErr)
			}
			_ = writer.Flush()
			continue
		}

		// Write response. Nothing answers a notification, so a body the
		// server sent for one is not relayed either.
		if len(response) > 0 && !notification {
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

	cred, err := p.credential(ctx)
	if err != nil {
		return nil, err
	}

	renewed := false
	for {
		resp, err := p.send(ctx, request, parsed, cred)
		if err != nil {
			return nil, err
		}

		// A 401 carrying a Bearer challenge means the token was refused, not
		// that the request was bad: renew once and resend. A second refusal
		// is reported rather than retried, so a revoked session cannot loop.
		if resp.StatusCode == http.StatusUnauthorized && !renewed &&
			domain.ParseBearerChallenge(resp.Header.Get("WWW-Authenticate")) != nil {
			next, renewErr := p.renew(ctx, cred)
			if renewErr == nil {
				drainAndClose(resp)
				cred = next
				renewed = true
				continue
			}
			if !errors.Is(renewErr, domain.ErrMCPCredentialNotRenewable) {
				drainAndClose(resp)
				return nil, renewErr
			}
		}

		body, err := p.readResponse(resp, cred)
		if err != nil {
			return nil, err
		}
		// Modify responses as needed
		if isToolsList {
			body = p.modifyToolsListResponse(body)
		}
		if isInitialize {
			p.rememberProtocolVersion(body)
			body = p.modifyInitializeResponse(body)
		}
		return body, nil
	}
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

// injectGrant injects defaultGrant as grant_id into a tool call that accepts
// one and names none. An empty defaultGrant injects nothing.
func (p *Proxy) injectGrant(request []byte, parsed *rpcRequest, defaultGrant string) []byte {
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
