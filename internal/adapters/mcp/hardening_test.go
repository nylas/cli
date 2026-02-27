package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// ============================================================================
// TestRunWithIO_InvalidRequest_MissingJSONRPC
// ============================================================================

func TestRunWithIO_InvalidRequest_MissingJSONRPC(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Valid JSON but missing "jsonrpc" field → -32600 Invalid Request.
	input := `{"id":1,"method":"ping","params":{}}` + "\n"
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeInvalidRequest)) {
		t.Errorf("output missing invalid request code %d; got: %s", codeInvalidRequest, outStr)
	}
	if !strings.Contains(outStr, "invalid request") {
		t.Errorf("output missing 'invalid request' message; got: %s", outStr)
	}
}

// ============================================================================
// TestRunWithIO_InvalidRequest_WrongJSONRPCVersion
// ============================================================================

func TestRunWithIO_InvalidRequest_WrongJSONRPCVersion(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// jsonrpc is "1.0" instead of "2.0" → -32600 Invalid Request.
	input := `{"jsonrpc":"1.0","id":1,"method":"ping","params":{}}` + "\n"
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeInvalidRequest)) {
		t.Errorf("output missing invalid request code %d; got: %s", codeInvalidRequest, outStr)
	}
}

// ============================================================================
// TestRunWithIO_InvalidRequest_EmptyMethod
// ============================================================================

func TestRunWithIO_InvalidRequest_EmptyMethod(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Valid jsonrpc but empty method → -32600 Invalid Request.
	input := `{"jsonrpc":"2.0","id":2,"method":"","params":{}}` + "\n"
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeInvalidRequest)) {
		t.Errorf("output missing invalid request code %d; got: %s", codeInvalidRequest, outStr)
	}
}

// ============================================================================
// TestRunWithIO_InvalidRequest_NoMethod
// ============================================================================

func TestRunWithIO_InvalidRequest_NoMethod(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Valid jsonrpc but method field completely absent → empty string → -32600.
	input := `{"jsonrpc":"2.0","id":3}` + "\n"
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeInvalidRequest)) {
		t.Errorf("output missing invalid request code %d; got: %s", codeInvalidRequest, outStr)
	}
}

// ============================================================================
// TestRunWithIO_InvalidRequest_InvalidIDType
// ============================================================================

func TestRunWithIO_InvalidRequest_InvalidIDType(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// JSON-RPC id must be string, number, or null. Object id should be rejected.
	input := `{"jsonrpc":"2.0","id":{"bad":1},"method":"ping","params":{}}` + "\n"
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeInvalidRequest)) {
		t.Errorf("output missing invalid request code %d; got: %s", codeInvalidRequest, outStr)
	}
}

// ============================================================================
// TestRunWithIO_BatchRequests
// ============================================================================

func TestRunWithIO_BatchRequests(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Batch with two requests should return a JSON array of two responses.
	input := `[` +
		`{"jsonrpc":"2.0","id":1,"method":"ping","params":{}},` +
		`{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}` +
		`]` + "\n"
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	line := strings.TrimSpace(out.String())
	var resp []map[string]any
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("batch response unmarshal: %v (raw=%s)", err, line)
	}
	if len(resp) != 2 {
		t.Fatalf("batch response count = %d, want 2", len(resp))
	}
}

// ============================================================================
// TestRunWithIO_InvalidRequest_ContentLength
// ============================================================================

func TestRunWithIO_InvalidRequest_ContentLength(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Content-Length framed request with bad jsonrpc version → -32600 in CL mode.
	body := `{"jsonrpc":"1.0","id":5,"method":"ping","params":{}}`
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	if !strings.Contains(outStr, "Content-Length:") {
		t.Errorf("response missing Content-Length header for CL-framed request; got: %s", outStr)
	}
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeInvalidRequest)) {
		t.Errorf("output missing invalid request code %d; got: %s", codeInvalidRequest, outStr)
	}
}

// ============================================================================
// TestRunWithIO_InvalidRequest_WriteError
// ============================================================================

func TestRunWithIO_InvalidRequest_WriteError(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// Invalid request + failing writer → RunWithIO returns write error.
	input := `{"id":1,"method":"ping","params":{}}` + "\n"
	err := s.RunWithIO(ctx, strings.NewReader(input), errWriter{})
	if err == nil {
		t.Fatal("RunWithIO() with failing writer on invalid request should return error")
	}
}

// ============================================================================
// TestRunWithIO_InvalidRequest_ThenValidRequest
// ============================================================================

func TestRunWithIO_InvalidRequest_ThenValidRequest(t *testing.T) {
	t.Parallel()

	s := NewServer(&mockNylasClient{}, "")
	ctx := context.Background()

	// First: invalid request (missing jsonrpc) → error response, server continues.
	// Second: valid ping → success response.
	input := `{"id":1,"method":"ping","params":{}}` + "\n" + pingNewlineJSON(2)
	var out bytes.Buffer
	err := s.RunWithIO(ctx, strings.NewReader(input), &out)
	if err != nil {
		t.Fatalf("RunWithIO() = %v, want nil", err)
	}

	outStr := out.String()
	// Should contain both the invalid request error and the valid ping response.
	if !strings.Contains(outStr, fmt.Sprintf("%d", codeInvalidRequest)) {
		t.Errorf("output missing invalid request error; got: %s", outStr)
	}
	if !strings.Contains(outStr, `"id":2`) {
		t.Errorf("output missing valid ping response (id=2); got: %s", outStr)
	}
}

// ============================================================================
// TestToolSchemas_AdditionalPropertiesFalse
// ============================================================================

func TestToolSchemas_AdditionalPropertiesFalse(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	if len(tools) == 0 {
		t.Fatal("registeredTools() returned empty list")
	}

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()

			schema := tool.InputSchema
			if schema.Type != "object" {
				t.Fatalf("InputSchema.Type = %q, want object", schema.Type)
			}

			if schema.AdditionalProperties == nil {
				t.Fatalf("InputSchema.AdditionalProperties is nil, want *false")
			}
			if *schema.AdditionalProperties {
				t.Fatalf("InputSchema.AdditionalProperties = true, want false")
			}
		})
	}
}

// ============================================================================
// TestToolSchemas_AdditionalProperties_SerializedAsJSON
// ============================================================================

func TestToolSchemas_AdditionalProperties_SerializedAsJSON(t *testing.T) {
	t.Parallel()

	// Verify that when serialized to JSON, the schema includes "additionalProperties": false.
	tools := registeredTools()
	if len(tools) == 0 {
		t.Fatal("registeredTools() returned empty list")
	}

	// Check the first tool as a representative sample.
	data, err := json.Marshal(tools[0].InputSchema)
	if err != nil {
		t.Fatalf("json.Marshal(InputSchema) = %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal = %v", err)
	}

	val, ok := raw["additionalProperties"]
	if !ok {
		t.Fatal("JSON output missing additionalProperties field")
	}
	if val != false {
		t.Errorf("additionalProperties = %v (%T), want false", val, val)
	}
}
