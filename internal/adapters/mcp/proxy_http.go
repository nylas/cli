package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

// send makes one HTTP request with cred. The caller closes the response.
func (p *Proxy) send(ctx context.Context, request []byte, parsed *rpcRequest, cred *domain.MCPCredential) (*http.Response, error) {
	endpoint := cred.Endpoint
	if endpoint == "" {
		endpoint = p.endpoint
	}
	if endpoint == "" {
		return nil, errors.New("no MCP server to send the request to")
	}

	p.mu.RLock()
	defaultGrant := p.defaultGrant
	protocolVersion := p.protocolVersion
	p.mu.RUnlock()

	// The default grant is only a hint, and an OAuth token only acts on the
	// grants it names: offering any other would be refused at best.
	grantHint := ""
	if cred.AllowsGrantHint(defaultGrant) {
		grantHint = defaultGrant
	}

	// Every attempt starts from the request as the assistant sent it: the
	// grant injection below writes into the parsed arguments, and a retry
	// after renewal must not inherit the previous credential's grant.
	parsed = cloneRPCRequest(parsed)

	// Inject default grant into tool calls if not specified
	request = p.injectGrant(request, parsed, grantHint)

	// Normalize tool arguments (type coercion, timestamp rounding)
	request = p.normalizeToolArguments(request, parsed)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(request))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+cred.Token)
	setProtocolHeaders(req.Header, parsed, protocolVersion)
	if grantHint != "" {
		req.Header.Set("X-Nylas-Grant-Id", grantHint)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	return resp, nil
}

// readResponse turns an HTTP response into the JSON-RPC payload to write
// back, closing it.
func (p *Proxy) readResponse(resp *http.Response, cred *domain.MCPCredential) ([]byte, error) {
	defer func() { _ = resp.Body.Close() }()

	// Handle 202 Accepted (no body)
	if resp.StatusCode == http.StatusAccepted {
		return nil, nil
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		return nil, p.statusError(resp, body, cred)
	}

	// Handle SSE stream
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		return p.readSSE(resp.Body)
	}

	// Handle JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
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

// isNotification reports whether a JSON-RPC message is a notification: one
// with no "id" member at all. An explicit "id": null is still a request, and
// rpcRequest.ID cannot tell the two apart. A batch is one when every member
// is: nothing in it expects an answer.
func isNotification(message []byte) bool {
	trimmed := bytes.TrimSpace(message)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(trimmed, &batch); err != nil || len(batch) == 0 {
			return false
		}
		for _, member := range batch {
			if !isNotification(member) {
				return false
			}
		}
		return true
	}
	var members map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &members); err != nil {
		return false
	}
	_, hasID := members["id"]
	return !hasID
}

// cloneRPCRequest copies req deeply, nested arguments included, so
// injectGrant and normalizeToolArguments can change the copy freely. A
// shallow copy let the first attempt's normalization write into the caller's
// nested maps, so a retry after renewal found nothing to normalize and sent
// the original bytes.
func cloneRPCRequest(req *rpcRequest) *rpcRequest {
	if req == nil {
		return nil
	}
	clone := *req
	if req.Params.Arguments != nil {
		clone.Params.Arguments = deepCopyJSON(req.Params.Arguments).(map[string]any)
	}
	return &clone
}

// deepCopyJSON copies a value decoded from JSON into any.
func deepCopyJSON(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			out[k] = deepCopyJSON(item)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = deepCopyJSON(item)
		}
		return out
	default:
		return v
	}
}
