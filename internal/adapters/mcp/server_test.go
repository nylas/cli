package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// newTestServer creates a Server with nil client for tests that don't invoke the Nylas API.
func newTestServer(defaultGrant string) *Server {
	return &Server{defaultGrant: defaultGrant}
}

// unmarshalResponse unmarshals a JSON-RPC response byte slice into a generic map.
func unmarshalResponse(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return got
}

// TestDispatch_Methods tests dispatch routing for methods that don't require the Nylas client.
func TestDispatch_Methods(t *testing.T) {
	t.Parallel()

	s := newTestServer("test-grant")
	ctx := context.Background()

	tests := []struct {
		name           string
		req            *Request
		wantNil        bool   // true if we expect a nil response (notifications)
		wantErrorCode  *int   // if non-nil, expect an error response with this code
		wantResultKeys []string
	}{
		{
			name:           "initialize returns capabilities",
			req:            &Request{Method: "initialize", ID: float64(1)},
			wantResultKeys: []string{"protocolVersion", "capabilities", "serverInfo", "instructions"},
		},
		{
			name:           "tools/list returns tools array",
			req:            &Request{Method: "tools/list", ID: float64(2)},
			wantResultKeys: []string{"tools"},
		},
		{
			name:           "ping returns empty object",
			req:            &Request{Method: "ping", ID: float64(3)},
			wantResultKeys: []string{},
		},
		{
			name:    "notifications/initialized returns nil",
			req:     &Request{Method: "notifications/initialized"},
			wantNil: true,
		},
		{
			name:    "notifications/cancelled returns nil",
			req:     &Request{Method: "notifications/cancelled"},
			wantNil: true,
		},
		{
			name: "unknown method returns error -32601",
			req:  &Request{Method: "unknown/method", ID: float64(99)},
			wantErrorCode: func() *int { c := codeMethodNotFound; return &c }(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := s.dispatch(ctx, tt.req)

			if tt.wantNil {
				if resp != nil {
					t.Errorf("dispatch() = %s, want nil", resp)
				}
				return
			}

			if resp == nil {
				t.Fatal("dispatch() = nil, want non-nil response")
			}

			got := unmarshalResponse(t, resp)

			if got["jsonrpc"] != "2.0" {
				t.Errorf("jsonrpc = %v, want 2.0", got["jsonrpc"])
			}

			if tt.wantErrorCode != nil {
				errObj, ok := got["error"].(map[string]any)
				if !ok {
					t.Fatalf("error field missing or wrong type; got = %v", got)
				}
				gotCode := int(errObj["code"].(float64))
				if gotCode != *tt.wantErrorCode {
					t.Errorf("error.code = %d, want %d", gotCode, *tt.wantErrorCode)
				}
				return
			}

			result, ok := got["result"].(map[string]any)
			if !ok {
				t.Fatalf("result field missing or wrong type; got = %v", got)
			}
			for _, key := range tt.wantResultKeys {
				if _, exists := result[key]; !exists {
					t.Errorf("result missing key %q", key)
				}
			}
		})
	}
}

// TestHandleInitialize_Fields verifies the initialize response structure and content.
func TestHandleInitialize_Fields(t *testing.T) {
	t.Parallel()

	s := newTestServer("grant-abc")
	req := &Request{Method: "initialize", ID: float64(1)}
	data := s.handleInitialize(req)

	got := unmarshalResponse(t, data)

	result, ok := got["result"].(map[string]any)
	if !ok {
		t.Fatalf("result field missing or wrong type")
	}

	if result["protocolVersion"] != protocolVersion {
		t.Errorf("protocolVersion = %v, want %v", result["protocolVersion"], protocolVersion)
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("serverInfo missing or wrong type")
	}
	if serverInfo["name"] != serverName {
		t.Errorf("serverInfo.name = %v, want %v", serverInfo["name"], serverName)
	}
	if serverInfo["version"] != serverVersion {
		t.Errorf("serverInfo.version = %v, want %v", serverInfo["version"], serverVersion)
	}

	if _, ok := result["capabilities"]; !ok {
		t.Error("capabilities field missing")
	}

	instructions, ok := result["instructions"].(string)
	if !ok || instructions == "" {
		t.Error("instructions field missing or empty")
	}
}

// TestHandleInitialize_TimezoneGuidance verifies the instructions contain timezone information.
func TestHandleInitialize_TimezoneGuidance(t *testing.T) {
	t.Parallel()

	s := newTestServer("")
	req := &Request{Method: "initialize", ID: float64(1)}
	data := s.handleInitialize(req)

	got := unmarshalResponse(t, data)
	result := got["result"].(map[string]any)
	instructions := result["instructions"].(string)

	tzKeywords := []string{
		"Timezone",
		"epoch_to_datetime",
		"UTC",
	}
	for _, kw := range tzKeywords {
		if !strings.Contains(instructions, kw) {
			t.Errorf("instructions missing expected timezone keyword %q", kw)
		}
	}
}

// TestHandleToolsList_ToolCount verifies exactly 37 tools are returned with required fields.
func TestHandleToolsList_ToolCount(t *testing.T) {
	t.Parallel()

	s := newTestServer("")
	req := &Request{Method: "tools/list", ID: float64(2)}
	data := s.handleToolsList(req)

	got := unmarshalResponse(t, data)
	result, ok := got["result"].(map[string]any)
	if !ok {
		t.Fatal("result field missing or wrong type")
	}

	toolsRaw, ok := result["tools"].([]any)
	if !ok {
		t.Fatal("tools field missing or wrong type")
	}

	const wantCount = 37
	if len(toolsRaw) != wantCount {
		t.Errorf("tool count = %d, want %d", len(toolsRaw), wantCount)
	}

	for i, raw := range toolsRaw {
		tool, ok := raw.(map[string]any)
		if !ok {
			t.Errorf("tools[%d] is not a map", i)
			continue
		}
		for _, field := range []string{"name", "description", "inputSchema"} {
			if _, exists := tool[field]; !exists {
				t.Errorf("tools[%d] missing field %q", i, field)
			}
		}
		name, _ := tool["name"].(string)
		if name == "" {
			t.Errorf("tools[%d] has empty name", i)
		}
	}
}

// TestRegisteredTools_UniqueNames verifies all tool names are unique.
func TestRegisteredTools_UniqueNames(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	seen := make(map[string]int)
	for i, tool := range tools {
		if prev, exists := seen[tool.Name]; exists {
			t.Errorf("duplicate tool name %q at index %d (first seen at index %d)", tool.Name, i, prev)
		}
		seen[tool.Name] = i
	}
}

// TestRegisteredTools_RequiredFields verifies all tools have non-empty name, description, and schema type.
func TestRegisteredTools_RequiredFields(t *testing.T) {
	t.Parallel()

	for _, tool := range registeredTools() {
		tool := tool
		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()

			if tool.Name == "" {
				t.Error("name is empty")
			}
			if tool.Description == "" {
				t.Errorf("tool %q has empty description", tool.Name)
			}
			if tool.InputSchema.Type == "" {
				t.Errorf("tool %q has empty inputSchema.type", tool.Name)
			}
		})
	}
}

// TestResolveGrantID verifies grant resolution priority.
func TestResolveGrantID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		defaultGrant string
		args         map[string]any
		want         string
	}{
		{
			name:         "args grant_id takes priority over default",
			defaultGrant: "default-grant",
			args:         map[string]any{"grant_id": "args-grant"},
			want:         "args-grant",
		},
		{
			name:         "missing args grant_id falls back to default",
			defaultGrant: "default-grant",
			args:         map[string]any{"other_key": "value"},
			want:         "default-grant",
		},
		{
			name:         "empty args grant_id falls back to default",
			defaultGrant: "default-grant",
			args:         map[string]any{"grant_id": ""},
			want:         "default-grant",
		},
		{
			name:         "nil args falls back to default",
			defaultGrant: "default-grant",
			args:         nil,
			want:         "default-grant",
		},
		{
			name:         "both empty returns empty string",
			defaultGrant: "",
			args:         map[string]any{},
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := newTestServer(tt.defaultGrant)
			got := s.resolveGrantID(tt.args)
			if got != tt.want {
				t.Errorf("resolveGrantID() = %q, want %q", got, tt.want)
			}
		})
	}
}
