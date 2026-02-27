// Package mcp provides a native MCP server that calls the Nylas API directly.
package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/nylas/cli/internal/domain"
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
func parseToolCallParams(raw json.RawMessage) (ToolCallParams, error) {
	var p ToolCallParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return p, fmt.Errorf("invalid tool call params: %w", err)
		}
	}
	return p, nil
}

// parseInitializeParams parses InitializeParams from raw JSON params.
func parseInitializeParams(raw json.RawMessage) (InitializeParams, error) {
	var p InitializeParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return p, fmt.Errorf("invalid initialize params: %w", err)
		}
	}
	return p, nil
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
	Content           []ContentBlock `json:"content"`
	StructuredContent any            `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
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
		Content:           []ContentBlock{{Type: "text", Text: string(text)}},
		StructuredContent: data,
	}
}

// toolSuccessText builds a successful MCP tool response with plain text (no JSON encoding).
func toolSuccessText(text string) *ToolResponse {
	return &ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: text}},
	}
}

// reURL matches http:// and https:// URLs for sanitization.
var reURL = regexp.MustCompile(`https?://\S+`)

// sanitizeError wraps API errors to prevent leaking internal details.
// Maps errors to stable categories and strips URLs from residual messages.
func sanitizeError(err error) string {
	if err == nil {
		return "request failed"
	}

	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		return "invalid input"
	case errors.Is(err, domain.ErrInvalidGrant),
		errors.Is(err, domain.ErrGrantNotFound),
		errors.Is(err, domain.ErrNoDefaultGrant),
		errors.Is(err, domain.ErrTokenExpired),
		errors.Is(err, domain.ErrAuthFailed):
		return "authentication failed"
	case errors.Is(err, domain.ErrNetworkError):
		return "network error"
	case errors.Is(err, domain.ErrAPIError):
		return "nylas API error"
	case errors.Is(err, domain.ErrMessageNotFound),
		errors.Is(err, domain.ErrThreadNotFound),
		errors.Is(err, domain.ErrDraftNotFound),
		errors.Is(err, domain.ErrFolderNotFound),
		errors.Is(err, domain.ErrAttachmentNotFound),
		errors.Is(err, domain.ErrContactNotFound),
		errors.Is(err, domain.ErrCalendarNotFound),
		errors.Is(err, domain.ErrEventNotFound):
		return "resource not found"
	}

	msg := strings.ToLower(err.Error())
	msg = reURL.ReplaceAllString(msg, "[api]")
	switch {
	case strings.Contains(msg, "rate limit"), strings.Contains(msg, "429"):
		return "rate limit exceeded"
	case strings.Contains(msg, "timeout"), strings.Contains(msg, "deadline exceeded"):
		return "request timed out"
	case strings.Contains(msg, "unauthorized"), strings.Contains(msg, "forbidden"), strings.Contains(msg, "401"), strings.Contains(msg, "403"):
		return "authentication failed"
	case strings.Contains(msg, "not found"), strings.Contains(msg, "404"):
		return "resource not found"
	}
	return "request failed"
}

// toolError builds an error MCP tool response.
func toolError(message string) *ToolResponse {
	return &ToolResponse{
		Content: []ContentBlock{{Type: "text", Text: message}},
		IsError: true,
	}
}

// maxLimit caps the maximum number of results for list operations.
const maxLimit = 200

// clampLimit returns the limit value clamped to [1, maxLimit].
func clampLimit(args map[string]any, key string, defaultVal int) int {
	v := getInt(args, key, defaultVal)
	if v <= 0 {
		return defaultVal
	}
	if v > maxLimit {
		return maxLimit
	}
	return v
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
