package mcp

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// fallbackErrorResponse is a pre-built error response for when JSON marshaling fails.
// This should never happen with well-formed maps, but provides a safety net.
var fallbackErrorResponse = []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"internal marshaling error"}}`)

// handleLocalToolCall checks if a tool call can be handled locally.
// Returns the response and true if handled locally, nil and false otherwise.
// Uses the pre-parsed request to avoid re-parsing.
func (p *Proxy) handleLocalToolCall(req *rpcRequest) ([]byte, bool) {
	p.mu.RLock()
	grantStore := p.grantStore
	defaultGrant := p.defaultGrant
	p.mu.RUnlock()

	// Need grant store for local handling
	if grantStore == nil {
		return nil, false
	}

	// Only handle tools/call for get_grant
	if req.Method != "tools/call" || req.Params.Name != "get_grant" {
		return nil, false
	}

	// Check if email is provided - if so, let cloud handle it
	if req.Params.Arguments != nil {
		if email, ok := req.Params.Arguments["email"].(string); ok && email != "" {
			return nil, false
		}
	}

	// No email provided - return the default grant from local storage
	var grantInfo *domain.GrantInfo
	var err error

	if defaultGrant != "" {
		grantInfo, err = grantStore.GetGrant(defaultGrant)
	}

	// If no default grant or not found, try to get the first available grant
	if grantInfo == nil || err != nil {
		grants, listErr := grantStore.ListGrants()
		if listErr == nil && len(grants) > 0 {
			grantInfo = &grants[0]
		}
	}

	if grantInfo == nil {
		// Return error response
		return p.createToolErrorResponse(req.ID, "No authenticated grants found. Please run 'nylas auth login' first."), true
	}

	// Build successful response
	return p.createToolSuccessResponse(req.ID, map[string]any{
		"grant_id": grantInfo.ID,
		"email":    grantInfo.Email,
		"provider": string(grantInfo.Provider),
	}), true
}

// createToolSuccessResponse creates a successful MCP tool call response.
// Returns an MCP-formatted JSON-RPC response with the result embedded as text content.
func (p *Proxy) createToolSuccessResponse(id any, result map[string]any) []byte {
	// Format result as text content (MCP tool response format).
	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("mcp: failed to marshal tool result: %v", err)
		return fallbackErrorResponse
	}

	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": string(resultJSON),
				},
			},
		},
	}

	resp, err := json.Marshal(response)
	if err != nil {
		log.Printf("mcp: failed to marshal success response: %v", err)
		return fallbackErrorResponse
	}
	return resp
}

// createToolErrorResponse creates an error response for a tool call.
func (p *Proxy) createToolErrorResponse(id any, message string) []byte {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result": map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": message,
				},
			},
			"isError": true,
		},
	}

	resp, err := json.Marshal(response)
	if err != nil {
		log.Printf("mcp: failed to marshal error response: %v", err)
		return fallbackErrorResponse
	}
	return resp
}

// createErrorResponse creates a JSON-RPC error response.
// Uses the pre-parsed request if available to get the ID.
func (p *Proxy) createErrorResponse(req *rpcRequest, originalErr error) []byte {
	var id any
	if req != nil {
		id = req.ID
	}

	errorResp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    -32603,
			"message": originalErr.Error(),
		},
	}

	respBytes, err := json.Marshal(errorResp)
	if err != nil {
		log.Printf("mcp: failed to marshal JSON-RPC error response: %v", err)
		return fallbackErrorResponse
	}
	return respBytes
}

// modifyToolsListResponse modifies the tools/list response to make get_grant email optional.
// This allows AI assistants to call get_grant without providing an email,
// which triggers the local grant lookup in handleLocalToolCall.
func (p *Proxy) modifyToolsListResponse(response []byte) []byte {
	// Parse the JSON-RPC response
	var rpcResp map[string]any
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return response
	}

	// Navigate to result.tools
	result, ok := rpcResp["result"].(map[string]any)
	if !ok {
		return response
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		return response
	}

	// Find and modify the get_grant tool
	for _, tool := range tools {
		toolMap, ok := tool.(map[string]any)
		if !ok {
			continue
		}

		name, ok := toolMap["name"].(string)
		if !ok || name != "get_grant" {
			continue
		}

		// Found get_grant - modify its inputSchema to make email optional
		inputSchema, ok := toolMap["inputSchema"].(map[string]any)
		if !ok {
			continue
		}

		// Remove "email" from required array
		required, ok := inputSchema["required"].([]any)
		if ok {
			newRequired := make([]any, 0, len(required))
			for _, r := range required {
				if r != "email" {
					newRequired = append(newRequired, r)
				}
			}
			inputSchema["required"] = newRequired
		}

		// Update the description to indicate email is optional
		if desc, ok := toolMap["description"].(string); ok {
			toolMap["description"] = desc + " If email is not provided, returns the default authenticated grant."
		}

		break
	}

	// Re-marshal the modified response
	modified, err := json.Marshal(rpcResp)
	if err != nil {
		return response
	}

	return modified
}

// modifyInitializeResponse enhances the initialize response with timezone guidance.
// This ensures AI assistants display all timestamps consistently in the user's timezone.
func (p *Proxy) modifyInitializeResponse(response []byte) []byte {
	// Parse the JSON-RPC response
	var rpcResp map[string]any
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return response
	}

	// Navigate to result
	result, ok := rpcResp["result"].(map[string]any)
	if !ok {
		return response
	}

	// Get existing instructions
	instructions, _ := result["instructions"].(string)

	// Detect system timezone
	localZone, _ := time.Now().Zone()
	tzName := time.Local.String()
	if tzName == "Local" {
		tzName = localZone // Fallback to abbreviation if no IANA name
	}

	// Append timezone guidance with detected timezone
	timezoneGuidance := fmt.Sprintf(`

IMPORTANT - Timezone Consistency:
The user's local timezone is: %s (%s)
When displaying ANY timestamps to users (from emails, events, availability, etc.):
1. Always use epoch_to_datetime tool with timezone "%s" to convert Unix timestamps
2. Display ALL times in %s, never in UTC or the event's original timezone
3. Format times clearly (e.g., "2:00 PM %s")`, tzName, localZone, tzName, localZone, localZone)

	result["instructions"] = instructions + timezoneGuidance
	rpcResp["result"] = result

	// Re-marshal the modified response
	modified, err := json.Marshal(rpcResp)
	if err != nil {
		return response
	}

	return modified
}
