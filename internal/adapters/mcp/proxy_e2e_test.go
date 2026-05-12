package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// mockMCPServer simulates the upstream Nylas MCP server for E2E proxy tests.
// It returns realistic tools/list, initialize, and tools/call responses.
type mockMCPServer struct {
	t              *testing.T
	lastToolCall   string
	lastArgs       map[string]any
	receivedGrant  string
	receivedMethod string
}

func (m *mockMCPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	var req rpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"parse error"}}`))
		return
	}

	m.receivedMethod = req.Method

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Mcp-Session-Id", "e2e-session-001")

	switch req.Method {
	case "tools/list":
		_, _ = w.Write([]byte(mockToolsListResponse))
	case "initialize":
		_, _ = w.Write([]byte(mockInitializeResponse))
	case "tools/call":
		m.lastToolCall = req.Params.Name
		m.lastArgs = req.Params.Arguments
		if grantID, ok := req.Params.Arguments["grant_id"].(string); ok {
			m.receivedGrant = grantID
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + idToJSON(req.ID) + `,"result":{"content":[{"type":"text","text":"{\"status\":\"ok\",\"tool\":\"` + req.Params.Name + `\"}"}]}}`))
	default:
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + idToJSON(req.ID) + `,"result":{}}`))
	}
}

func idToJSON(id any) string {
	b, _ := json.Marshal(id)
	return string(b)
}

// TestE2E_ProxyLifecycle tests the full proxy lifecycle:
//  1. initialize — timezone guidance is appended
//  2. tools/list — dynamic discovery populates grantTools
//  3. tools/call with grant tools — grant_id is auto-injected
//  4. tools/call with utility tools — grant_id is NOT injected
//  5. tools/call with explicit grant_id — NOT overridden
//  6. tools/call with new upstream tool — dynamically discovered
//  7. get_grant without email — handled locally
func TestE2E_ProxyLifecycle(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("e2e-grant-abc")
	proxy.SetGrantStore(&mockGrantStore{
		grants: []domain.GrantInfo{
			{ID: "e2e-grant-abc", Email: "user@example.com", Provider: "google"},
		},
		defaultGrant: "e2e-grant-abc",
	})

	ctx := t.Context()

	// === Step 1: initialize — timezone guidance appended ===
	t.Run("initialize_appends_timezone_guidance", func(t *testing.T) {
		req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
		resp, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}

		var result map[string]any
		mustUnmarshal(t, resp, &result)
		instructions := dig[string](result, "result", "instructions")
		if !strings.Contains(instructions, "epoch_to_datetime") {
			t.Error("expected timezone guidance mentioning epoch_to_datetime in instructions")
		}
		if !strings.Contains(instructions, "Nylas MCP server") {
			t.Error("expected original instructions to be preserved")
		}
	})

	// === Step 2: tools/list — discovery + get_grant modification ===
	t.Run("tools_list_discovers_grant_tools_and_modifies_get_grant", func(t *testing.T) {
		req := parseRPC(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
		resp, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}

		var result map[string]any
		mustUnmarshal(t, resp, &result)

		tools := dig[[]any](result, "result", "tools")
		if len(tools) == 0 {
			t.Fatal("expected tools in response")
		}

		// Verify get_grant was modified (email no longer required)
		for _, tool := range tools {
			toolMap := tool.(map[string]any)
			if toolMap["name"] == "get_grant" {
				schema := toolMap["inputSchema"].(map[string]any)
				required, _ := schema["required"].([]any)
				for _, r := range required {
					if r == "email" {
						t.Error("expected email removed from get_grant required")
					}
				}
				desc := toolMap["description"].(string)
				if !strings.Contains(desc, "default authenticated grant") {
					t.Error("expected get_grant description to be appended")
				}
			}
		}

		// Verify dynamic discovery populated grantTools
		if proxy.grantTools == nil {
			t.Fatal("expected grantTools to be populated after tools/list")
		}

		// Grant tools should be discovered
		expectedGrant := []string{
			"list_messages", "list_calendars", "list_contacts", "get_contact",
			"create_event", "delete_event", "confirm_send_draft", "send_message",
			"brand_new_tool",
		}
		for _, name := range expectedGrant {
			if !proxy.grantTools[name] {
				t.Errorf("expected %q to be discovered as grant tool", name)
			}
		}

		// Utility/non-grant tools should NOT be discovered
		expectedNoGrant := []string{
			"current_time", "epoch_to_datetime", "availability",
			"confirm_send_message", "get_grant",
		}
		for _, name := range expectedNoGrant {
			if proxy.grantTools[name] {
				t.Errorf("expected %q to NOT be in grantTools", name)
			}
		}
	})

	// === Step 3: tools/call with grant tools — grant_id auto-injected ===
	t.Run("grant_tools_get_grant_id_injected", func(t *testing.T) {
		grantTools := []struct {
			name  string
			input string
		}{
			{"list_messages", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_messages","arguments":{}}}`},
			{"list_calendars", `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_calendars","arguments":{}}}`},
			{"list_contacts", `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"list_contacts","arguments":{}}}`},
			{"create_event", `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"create_event","arguments":{"calendar_id":"primary","event_request":{}}}}`},
			{"delete_event", `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"delete_event","arguments":{"calendar_id":"primary","event_id":"evt-1"}}}`},
			{"confirm_send_draft", `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"confirm_send_draft","arguments":{"draft_id":"d-1"}}}`},
			{"brand_new_tool", `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"brand_new_tool","arguments":{"payload":{}}}}`},
		}

		for _, tt := range grantTools {
			t.Run(tt.name, func(t *testing.T) {
				mock.receivedGrant = ""
				req := parseRPC(tt.input)
				_, err := proxy.forward(ctx, req.raw, req.parsed)
				if err != nil {
					t.Fatalf("forward failed: %v", err)
				}
				if mock.receivedGrant != "e2e-grant-abc" {
					t.Errorf("expected grant_id 'e2e-grant-abc' injected for %s, got %q", tt.name, mock.receivedGrant)
				}
			})
		}
	})

	// === Step 4: utility tools — grant_id NOT injected ===
	t.Run("utility_tools_no_grant_id_injected", func(t *testing.T) {
		utilityTools := []struct {
			name  string
			input string
		}{
			{"current_time", `{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"current_time","arguments":{"timezone":"UTC"}}}`},
			{"epoch_to_datetime", `{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"epoch_to_datetime","arguments":{"batch":[{"epoch_time":1700000000}]}}}`},
			{"availability", `{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{}}}}`},
			{"confirm_send_message", `{"jsonrpc":"2.0","id":13,"method":"tools/call","params":{"name":"confirm_send_message","arguments":{"message_request":{}}}}`},
		}

		for _, tt := range utilityTools {
			t.Run(tt.name, func(t *testing.T) {
				mock.receivedGrant = ""
				req := parseRPC(tt.input)
				_, err := proxy.forward(ctx, req.raw, req.parsed)
				if err != nil {
					t.Fatalf("forward failed: %v", err)
				}
				if mock.receivedGrant != "" {
					t.Errorf("expected NO grant_id for %s, but got %q", tt.name, mock.receivedGrant)
				}
			})
		}
	})

	// === Step 5: explicit grant_id — NOT overridden ===
	t.Run("explicit_grant_id_not_overridden", func(t *testing.T) {
		mock.receivedGrant = ""
		req := parseRPC(`{"jsonrpc":"2.0","id":14,"method":"tools/call","params":{"name":"list_messages","arguments":{"grant_id":"user-provided-grant"}}}`)
		_, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}
		if mock.receivedGrant != "user-provided-grant" {
			t.Errorf("expected user-provided grant_id preserved, got %q", mock.receivedGrant)
		}
	})

	// === Step 6: explicit identifier — grant_id NOT injected ===
	t.Run("identifier_prevents_grant_injection", func(t *testing.T) {
		mock.receivedGrant = ""
		req := parseRPC(`{"jsonrpc":"2.0","id":15,"method":"tools/call","params":{"name":"list_messages","arguments":{"identifier":"user@example.com"}}}`)
		_, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}
		if mock.receivedGrant != "" {
			t.Errorf("expected no grant_id when identifier is set, got %q", mock.receivedGrant)
		}
	})

	// === Step 7: get_grant without email — local response ===
	t.Run("get_grant_without_email_handled_locally", func(t *testing.T) {
		var req rpcRequest
		raw := []byte(`{"jsonrpc":"2.0","id":16,"method":"tools/call","params":{"name":"get_grant","arguments":{}}}`)
		_ = json.Unmarshal(raw, &req)

		resp, handled := proxy.handleLocalToolCall(&req)
		if !handled {
			t.Fatal("expected get_grant without email to be handled locally")
		}

		var result struct {
			Result struct {
				Content []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"result"`
		}
		mustUnmarshal(t, resp, &result)

		if len(result.Result.Content) == 0 {
			t.Fatal("expected content in local get_grant response")
		}

		var grant struct {
			GrantID  string `json:"grant_id"`
			Email    string `json:"email"`
			Provider string `json:"provider"`
		}
		mustUnmarshal(t, []byte(result.Result.Content[0].Text), &grant)

		if grant.GrantID != "e2e-grant-abc" {
			t.Errorf("expected grant_id 'e2e-grant-abc', got %q", grant.GrantID)
		}
		if grant.Email != "user@example.com" {
			t.Errorf("expected email 'user@example.com', got %q", grant.Email)
		}
	})

	// === Step 8: get_grant with email — passes through to server ===
	t.Run("get_grant_with_email_passes_through", func(t *testing.T) {
		var req rpcRequest
		raw := []byte(`{"jsonrpc":"2.0","id":17,"method":"tools/call","params":{"name":"get_grant","arguments":{"email":"other@example.com"}}}`)
		_ = json.Unmarshal(raw, &req)

		_, handled := proxy.handleLocalToolCall(&req)
		if handled {
			t.Error("expected get_grant with email to pass through to server")
		}
	})

	// === Step 9: session ID stored from server response ===
	t.Run("session_id_stored", func(t *testing.T) {
		if proxy.sessionID != "e2e-session-001" {
			t.Errorf("expected session ID 'e2e-session-001', got %q", proxy.sessionID)
		}
	})

	// === Step 10: non-tools/call method passes through ===
	t.Run("non_tools_method_passes_through", func(t *testing.T) {
		mock.receivedMethod = ""
		req := parseRPC(`{"jsonrpc":"2.0","id":18,"method":"notifications/initialized","params":{}}`)
		_, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}
		if mock.receivedMethod != "notifications/initialized" {
			t.Errorf("expected method forwarded as-is, got %q", mock.receivedMethod)
		}
	})
}

// TestE2E_DiscoveryOverridesFallback verifies that after tools/list,
// a tool NOT in the static fallback but present upstream gets grant injection.
func TestE2E_DiscoveryOverridesFallback(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("grant-xyz")

	ctx := t.Context()

	// Before discovery: brand_new_tool is NOT in fallback
	if proxy.toolRequiresGrant("brand_new_tool") {
		t.Fatal("brand_new_tool should NOT be in fallback")
	}

	// Trigger discovery via tools/list
	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	_, err := proxy.forward(ctx, req.raw, req.parsed)
	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	// After discovery: brand_new_tool IS now known
	if !proxy.toolRequiresGrant("brand_new_tool") {
		t.Fatal("brand_new_tool should be discovered after tools/list")
	}

	// Verify grant injection works for the newly discovered tool
	mock.receivedGrant = ""
	callReq := parseRPC(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"brand_new_tool","arguments":{"payload":{}}}}`)
	_, err = proxy.forward(ctx, callReq.raw, callReq.parsed)
	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}
	if mock.receivedGrant != "grant-xyz" {
		t.Errorf("expected grant_id injected for brand_new_tool, got %q", mock.receivedGrant)
	}
}

// TestE2E_NoGrantSetSkipsInjection verifies that when no default grant
// is configured, tools/call passes through without grant_id.
func TestE2E_NoGrantSetSkipsInjection(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	// No SetDefaultGrant called

	ctx := t.Context()

	mock.receivedGrant = ""
	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{}}}`)
	_, err := proxy.forward(ctx, req.raw, req.parsed)
	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}
	if mock.receivedGrant != "" {
		t.Errorf("expected no grant_id when none configured, got %q", mock.receivedGrant)
	}
}

// TestE2E_ServerError tests proxy behavior when upstream returns errors.
func TestE2E_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL

	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	_, err := proxy.forward(t.Context(), req.raw, req.parsed)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

// TestE2E_NormalizeListEventsArgs verifies the proxy coerces integer start/end
// to strings before forwarding list_events to the upstream server.
func TestE2E_NormalizeListEventsArgs(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("grant-norm")

	ctx := t.Context()

	t.Run("integer_start_end_coerced_to_strings", func(t *testing.T) {
		req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000,"limit":5}}}}`)
		_, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}

		params, ok := mock.lastArgs["get_all_query_parameters"].(map[string]any)
		if !ok {
			t.Fatal("expected get_all_query_parameters in forwarded args")
		}

		start, ok := params["start"].(string)
		if !ok {
			t.Errorf("expected start to be string, got %T (%v)", params["start"], params["start"])
		} else if start != "1747065600" {
			t.Errorf("start = %q, want %q", start, "1747065600")
		}

		end, ok := params["end"].(string)
		if !ok {
			t.Errorf("expected end to be string, got %T (%v)", params["end"], params["end"])
		} else if end != "1747152000" {
			t.Errorf("end = %q, want %q", end, "1747152000")
		}
	})

	t.Run("string_start_end_preserved", func(t *testing.T) {
		req := parseRPC(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":"1747065600","end":"1747152000","limit":5}}}}`)
		_, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}

		params, ok := mock.lastArgs["get_all_query_parameters"].(map[string]any)
		if !ok {
			t.Fatal("expected get_all_query_parameters in forwarded args")
		}

		if start, ok := params["start"].(string); !ok || start != "1747065600" {
			t.Errorf("expected start string preserved, got %T %v", params["start"], params["start"])
		}
	})
}

// TestE2E_NormalizeAvailabilityArgs verifies the proxy rounds availability
// timestamps to 5-minute boundaries before forwarding.
func TestE2E_NormalizeAvailabilityArgs(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL

	ctx := t.Context()

	t.Run("unaligned_timestamps_rounded", func(t *testing.T) {
		req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065601,"end_time":1747065899,"duration_minutes":30,"interval_minutes":30,"participants":[{"email":"user@example.com"}]}}}}`)
		_, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}

		avail, ok := mock.lastArgs["availability_request"].(map[string]any)
		if !ok {
			t.Fatal("expected availability_request in forwarded args")
		}

		startTime, ok := avail["start_time"].(float64)
		if !ok {
			t.Fatalf("expected start_time to be float64, got %T", avail["start_time"])
		}
		if startTime != 1747065600 {
			t.Errorf("start_time = %v, want %v (rounded down)", startTime, 1747065600)
		}

		endTime, ok := avail["end_time"].(float64)
		if !ok {
			t.Fatalf("expected end_time to be float64, got %T", avail["end_time"])
		}
		if endTime != 1747065900 {
			t.Errorf("end_time = %v, want %v (rounded up)", endTime, 1747065900)
		}
	})

	t.Run("aligned_timestamps_unchanged", func(t *testing.T) {
		req := parseRPC(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065600,"end_time":1747066200,"duration_minutes":30,"interval_minutes":30,"participants":[{"email":"user@example.com"}]}}}}`)
		_, err := proxy.forward(ctx, req.raw, req.parsed)
		if err != nil {
			t.Fatalf("forward failed: %v", err)
		}

		avail, ok := mock.lastArgs["availability_request"].(map[string]any)
		if !ok {
			t.Fatal("expected availability_request in forwarded args")
		}

		if startTime, ok := avail["start_time"].(float64); !ok || startTime != 1747065600 {
			t.Errorf("start_time = %v, want %v (should be unchanged)", avail["start_time"], 1747065600)
		}
		if endTime, ok := avail["end_time"].(float64); !ok || endTime != 1747066200 {
			t.Errorf("end_time = %v, want %v (should be unchanged)", avail["end_time"], 1747066200)
		}
	})
}

// TestE2E_NormalizeListEvents_GrantInjectionCombined verifies grant injection and
// list_events normalization work together through the full forward path.
func TestE2E_NormalizeListEvents_GrantInjectionCombined(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("combined-grant-xyz")

	ctx := t.Context()

	// No grant_id in request + integer start/end — both should be fixed
	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000,"limit":3}}}}`)
	_, err := proxy.forward(ctx, req.raw, req.parsed)
	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	// Verify grant was injected
	if mock.receivedGrant != "combined-grant-xyz" {
		t.Errorf("grant_id = %q, want 'combined-grant-xyz'", mock.receivedGrant)
	}

	// Verify start/end were coerced to strings
	params, ok := mock.lastArgs["get_all_query_parameters"].(map[string]any)
	if !ok {
		t.Fatal("expected get_all_query_parameters")
	}
	if start, ok := params["start"].(string); !ok || start != "1747065600" {
		t.Errorf("start = %v (%T), want string '1747065600'", params["start"], params["start"])
	}
	if end, ok := params["end"].(string); !ok || end != "1747152000" {
		t.Errorf("end = %v (%T), want string '1747152000'", params["end"], params["end"])
	}

	// Verify limit was NOT converted to string
	if limit, ok := params["limit"].(float64); !ok || limit != 3 {
		t.Errorf("limit = %v (%T), want float64 3", params["limit"], params["limit"])
	}
}

// TestE2E_NormalizeListEvents_ExplicitGrantPreserved verifies that explicit
// grant_id is preserved while start/end are still normalized.
func TestE2E_NormalizeListEvents_ExplicitGrantPreserved(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("default-grant")

	ctx := t.Context()

	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"grant_id":"user-explicit-grant","get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000}}}}`)
	_, err := proxy.forward(ctx, req.raw, req.parsed)
	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	// User's explicit grant should be preserved, not overridden
	if mock.receivedGrant != "user-explicit-grant" {
		t.Errorf("grant_id = %q, want 'user-explicit-grant'", mock.receivedGrant)
	}

	// start/end should still be normalized
	params := mock.lastArgs["get_all_query_parameters"].(map[string]any)
	if _, ok := params["start"].(string); !ok {
		t.Errorf("start should be string even with explicit grant, got %T", params["start"])
	}
}

// TestE2E_NormalizeAvailability_WithParticipants verifies availability rounding
// doesn't affect participant data.
func TestE2E_NormalizeAvailability_WithParticipants(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL

	ctx := t.Context()

	req := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065601,"end_time":1747065899,"duration_minutes":30,"interval_minutes":15,"participants":[{"email":"alice@example.com"},{"email":"bob@example.com","open_hours":[{"days":[1,2,3,4,5],"start":"09:00","end":"17:00","timezone":"America/New_York","exdates":[]}]}]}}}}`)
	_, err := proxy.forward(ctx, req.raw, req.parsed)
	if err != nil {
		t.Fatalf("forward failed: %v", err)
	}

	avail := mock.lastArgs["availability_request"].(map[string]any)

	// Timestamps should be rounded
	if st, ok := avail["start_time"].(float64); !ok || st != 1747065600 {
		t.Errorf("start_time = %v, want 1747065600", avail["start_time"])
	}
	if et, ok := avail["end_time"].(float64); !ok || et != 1747065900 {
		t.Errorf("end_time = %v, want 1747065900", avail["end_time"])
	}

	// Participants should be fully preserved
	participants, ok := avail["participants"].([]any)
	if !ok || len(participants) != 2 {
		t.Fatalf("expected 2 participants, got %v", avail["participants"])
	}

	p1 := participants[0].(map[string]any)
	if p1["email"] != "alice@example.com" {
		t.Errorf("first participant email = %v", p1["email"])
	}

	p2 := participants[1].(map[string]any)
	if p2["email"] != "bob@example.com" {
		t.Errorf("second participant email = %v", p2["email"])
	}

	// open_hours on participant 2 should be preserved
	openHours, ok := p2["open_hours"].([]any)
	if !ok || len(openHours) != 1 {
		t.Fatalf("expected 1 open_hours entry on p2, got %v", p2["open_hours"])
	}
	oh := openHours[0].(map[string]any)
	if oh["timezone"] != "America/New_York" {
		t.Errorf("open_hours timezone = %v, want America/New_York", oh["timezone"])
	}

	// duration and interval should be preserved
	if dur, ok := avail["duration_minutes"].(float64); !ok || dur != 30 {
		t.Errorf("duration_minutes = %v, want 30", avail["duration_minutes"])
	}
	if iv, ok := avail["interval_minutes"].(float64); !ok || iv != 15 {
		t.Errorf("interval_minutes = %v, want 15", avail["interval_minutes"])
	}
}

// TestE2E_NormalizationAfterDiscovery verifies that normalization works correctly
// after tools/list dynamic discovery has populated grantTools.
func TestE2E_NormalizationAfterDiscovery(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("disc-grant")

	ctx := t.Context()

	// Step 1: trigger discovery
	toolsReq := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	_, err := proxy.forward(ctx, toolsReq.raw, toolsReq.parsed)
	if err != nil {
		t.Fatalf("tools/list failed: %v", err)
	}

	// Step 2: list_events with integer start/end — should normalize + inject grant
	eventsReq := parseRPC(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000}}}}`)
	_, err = proxy.forward(ctx, eventsReq.raw, eventsReq.parsed)
	if err != nil {
		t.Fatalf("list_events failed: %v", err)
	}

	if mock.receivedGrant != "disc-grant" {
		t.Errorf("post-discovery grant = %q, want 'disc-grant'", mock.receivedGrant)
	}
	params := mock.lastArgs["get_all_query_parameters"].(map[string]any)
	if _, ok := params["start"].(string); !ok {
		t.Errorf("post-discovery start not coerced, got %T", params["start"])
	}
}

// TestE2E_NormalizationDoesNotAffectOtherTools verifies that tools like
// list_messages, list_threads, etc. are forwarded without modification even
// when they contain get_all_query_parameters.
func TestE2E_NormalizationDoesNotAffectOtherTools(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("other-grant")

	ctx := t.Context()

	tests := []struct {
		name  string
		input string
		check func(t *testing.T)
	}{
		{
			name:  "list_messages numeric fields unchanged",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{"get_all_query_parameters":{"limit":10,"received_after":1747065600}}}}`,
			check: func(t *testing.T) {
				params := mock.lastArgs["get_all_query_parameters"].(map[string]any)
				// received_after should stay numeric, not converted to string
				if _, ok := params["received_after"].(float64); !ok {
					t.Errorf("received_after should remain numeric, got %T", params["received_after"])
				}
			},
		},
		{
			name:  "list_threads numeric fields unchanged",
			input: `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_threads","arguments":{"get_all_query_parameters":{"limit":5}}}}`,
			check: func(t *testing.T) {
				params := mock.lastArgs["get_all_query_parameters"].(map[string]any)
				if limit, ok := params["limit"].(float64); !ok || limit != 5 {
					t.Errorf("limit = %v, want 5", params["limit"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := parseRPC(tt.input)
			_, err := proxy.forward(ctx, req.raw, req.parsed)
			if err != nil {
				t.Fatalf("forward failed: %v", err)
			}
			tt.check(t)
		})
	}
}

// TestE2E_MultipleSequentialNormalizations verifies normalization works correctly
// across multiple sequential calls on the same proxy instance.
func TestE2E_MultipleSequentialNormalizations(t *testing.T) {
	t.Parallel()

	mock := &mockMCPServer{t: t}
	server := httptest.NewServer(mock)
	defer server.Close()

	proxy := NewProxy("test-api-key", "us")
	proxy.endpoint = server.URL
	proxy.SetDefaultGrant("seq-grant")

	ctx := t.Context()

	// Call 1: list_events
	req1 := parseRPC(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":100,"end":200}}}}`)
	_, err := proxy.forward(ctx, req1.raw, req1.parsed)
	if err != nil {
		t.Fatalf("call 1 failed: %v", err)
	}
	params1 := mock.lastArgs["get_all_query_parameters"].(map[string]any)
	if params1["start"] != "100" {
		t.Errorf("call 1 start = %v, want '100'", params1["start"])
	}

	// Call 2: availability
	req2 := parseRPC(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":301,"end_time":599,"duration_minutes":30,"participants":[]}}}}`)
	_, err = proxy.forward(ctx, req2.raw, req2.parsed)
	if err != nil {
		t.Fatalf("call 2 failed: %v", err)
	}
	avail := mock.lastArgs["availability_request"].(map[string]any)
	if st, ok := avail["start_time"].(float64); !ok || st != 300 {
		t.Errorf("call 2 start_time = %v, want 300", avail["start_time"])
	}
	if et, ok := avail["end_time"].(float64); !ok || et != 600 {
		t.Errorf("call 2 end_time = %v, want 600", avail["end_time"])
	}

	// Call 3: list_events again with different values
	req3 := parseRPC(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":999,"end":1000}}}}`)
	_, err = proxy.forward(ctx, req3.raw, req3.parsed)
	if err != nil {
		t.Fatalf("call 3 failed: %v", err)
	}
	params3 := mock.lastArgs["get_all_query_parameters"].(map[string]any)
	if params3["start"] != "999" {
		t.Errorf("call 3 start = %v, want '999'", params3["start"])
	}
	if params3["end"] != "1000" {
		t.Errorf("call 3 end = %v, want '1000'", params3["end"])
	}
}

// --- helpers ---

type parsedRPC struct {
	raw    []byte
	parsed *rpcRequest
}

func parseRPC(s string) parsedRPC {
	var req rpcRequest
	_ = json.Unmarshal([]byte(s), &req)
	return parsedRPC{raw: []byte(s), parsed: &req}
}

func mustUnmarshal(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("failed to unmarshal: %v\ndata: %s", err, string(data))
	}
}

// dig navigates nested maps to extract a typed value.
func dig[T any](m map[string]any, keys ...string) T {
	var zero T
	current := any(m)
	for _, k := range keys {
		cm, ok := current.(map[string]any)
		if !ok {
			return zero
		}
		current = cm[k]
	}
	v, _ := current.(T)
	return v
}
