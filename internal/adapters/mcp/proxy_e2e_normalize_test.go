package mcp

import (
	"net/http/httptest"
	"testing"
)

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
