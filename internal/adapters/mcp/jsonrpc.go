// Package mcp provides a native MCP server that calls the Nylas API directly.
package mcp

import (
	"encoding/json"
	"log"
	"strings"
)

// fallbackErrorResponse is a pre-built error response for when JSON marshaling fails.
var fallbackErrorResponse = []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"internal marshaling error"}}`)

// JSON-RPC error codes.
const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
)

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// ToolCallParams holds parsed parameters for a tools/call request.
type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
	Cursor    string         `json:"cursor,omitempty"`
}

// InitializeParams holds parsed parameters for an initialize request.
type InitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

// parseToolCallParams parses ToolCallParams from raw JSON params.
func parseToolCallParams(raw json.RawMessage) ToolCallParams {
	var p ToolCallParams
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	return p
}

// parseInitializeParams parses InitializeParams from raw JSON params.
func parseInitializeParams(raw json.RawMessage) InitializeParams {
	var p InitializeParams
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	return p
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ToolResponse represents an MCP tool call result.
type ToolResponse struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in an MCP tool response.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// successResponse builds a JSON-RPC success response.
func successResponse(id any, result any) []byte {
	resp := Response{JSONRPC: "2.0", ID: id, Result: result}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("mcp: failed to marshal success response: %v", err)
		return fallbackErrorResponse
	}
	return data
}

// errorResponse builds a JSON-RPC error response.
func errorResponse(id any, code int, message string) []byte {
	resp := Response{JSONRPC: "2.0", ID: id, Error: RPCError{Code: code, Message: message}}
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("mcp: failed to marshal error response: %v", err)
		return fallbackErrorResponse
	}
	return data
}

// toolSuccess builds a successful MCP tool response with JSON-encoded data.
func toolSuccess(data any) *ToolResponse {
	text, err := json.Marshal(data)
	if err != nil {
		return toolError("failed to marshal result: " + err.Error())
	}
	return &ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: string(text)}},
	}
}

// toolSuccessText builds a successful MCP tool response with plain text (no JSON encoding).
func toolSuccessText(text string) *ToolResponse {
	return &ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

// sanitizeError wraps API errors to prevent leaking internal details.
// Preserves the high-level message while stripping URLs and auth headers.
func sanitizeError(err error) string {
	msg := err.Error()
	// Strip full API URLs that may contain grant IDs or tokens.
	if idx := strings.Index(msg, "https://"); idx >= 0 {
		// Keep the error prefix, replace URL with "[API]"
		msg = msg[:idx] + "[API]"
	}
	return msg
}

// toolError builds an error MCP tool response.
func toolError(message string) *ToolResponse {
	return &ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: message}},
		IsError: true,
	}
}

// getString extracts a string argument with a default value.
func getString(args map[string]any, key, defaultVal string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

// getInt extracts an integer argument with a default value.
func getInt(args map[string]any, key string, defaultVal int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

// getInt64 extracts an int64 argument with a default value.
func getInt64(args map[string]any, key string, defaultVal int64) int64 {
	if v, ok := args[key].(float64); ok {
		return int64(v)
	}
	return defaultVal
}

// getBool extracts a boolean argument. Returns nil if not present.
func getBool(args map[string]any, key string) *bool {
	if v, ok := args[key].(bool); ok {
		return &v
	}
	return nil
}

// getStringSlice extracts a string slice from an interface slice argument.
func getStringSlice(args map[string]any, key string) []string {
	val, ok := args[key]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
