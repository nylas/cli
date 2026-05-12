package mcp

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestProxy_normalizeToolArguments_ListEvents(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name      string
		input     string
		wantStart string
		wantEnd   string
	}{
		{
			name:      "coerces integer start and end to strings",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000,"limit":5}}}}`,
			wantStart: "1747065600",
			wantEnd:   "1747152000",
		},
		{
			name:      "coerces float start and end to strings",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600.0,"end":1747152000.0,"limit":5}}}}`,
			wantStart: "1747065600",
			wantEnd:   "1747152000",
		},
		{
			name:      "preserves string start and end unchanged",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":"1747065600","end":"1747152000","limit":5}}}}`,
			wantStart: "1747065600",
			wantEnd:   "1747152000",
		},
		{
			name:      "handles only start present (no end)",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"limit":5}}}}`,
			wantStart: "1747065600",
			wantEnd:   "",
		},
		{
			name:      "handles only end present (no start)",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","end":1747152000,"limit":5}}}}`,
			wantStart: "",
			wantEnd:   "1747152000",
		},
		{
			name:      "mixed types: integer start with string end",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":"1747152000"}}}}`,
			wantStart: "1747065600",
			wantEnd:   "1747152000",
		},
		{
			name:      "mixed types: string start with integer end",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":"1747065600","end":1747152000}}}}`,
			wantStart: "1747065600",
			wantEnd:   "1747152000",
		},
		{
			name:      "zero epoch value coerced to string",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":0,"end":86400}}}}`,
			wantStart: "0",
			wantEnd:   "86400",
		},
		{
			name:      "far future timestamps (year 2050)",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":2524608000,"end":2524694400}}}}`,
			wantStart: "2524608000",
			wantEnd:   "2524694400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.normalizeToolArguments([]byte(tt.input), nil)

			var parsed rpcRequest
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("failed to parse result: %v", err)
			}

			params, ok := parsed.Params.Arguments["get_all_query_parameters"].(map[string]any)
			if !ok {
				t.Fatal("expected get_all_query_parameters to be a map")
			}

			if tt.wantStart != "" {
				start, ok := params["start"].(string)
				if !ok {
					t.Errorf("expected start to be string, got %T (%v)", params["start"], params["start"])
				} else if start != tt.wantStart {
					t.Errorf("start = %q, want %q", start, tt.wantStart)
				}
			}

			if tt.wantEnd != "" {
				end, ok := params["end"].(string)
				if !ok {
					t.Errorf("expected end to be string, got %T (%v)", params["end"], params["end"])
				} else if end != tt.wantEnd {
					t.Errorf("end = %q, want %q", end, tt.wantEnd)
				}
			}
		})
	}
}

func TestProxy_normalizeToolArguments_ListEvents_OtherFieldsUntouched(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"grant_id":"abc","get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000,"limit":5}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)

	var parsed rpcRequest
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse result: %v", err)
	}

	params := parsed.Params.Arguments["get_all_query_parameters"].(map[string]any)

	// limit should still be a number, not coerced to string
	limit, ok := params["limit"].(float64)
	if !ok {
		t.Errorf("expected limit to remain numeric, got %T (%v)", params["limit"], params["limit"])
	} else if limit != 5 {
		t.Errorf("limit = %v, want 5", limit)
	}

	// calendar_id should still be a string
	calID, ok := params["calendar_id"].(string)
	if !ok || calID != "primary" {
		t.Errorf("calendar_id = %v, want 'primary'", params["calendar_id"])
	}

	// grant_id should be untouched at the top level
	grantID, ok := parsed.Params.Arguments["grant_id"].(string)
	if !ok || grantID != "abc" {
		t.Errorf("grant_id = %v, want 'abc'", parsed.Params.Arguments["grant_id"])
	}
}

func TestProxy_normalizeToolArguments_ListEvents_PreParsed(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000}}}}`)
	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}

	result := proxy.normalizeToolArguments(raw, &req)

	var parsed rpcRequest
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	params := parsed.Params.Arguments["get_all_query_parameters"].(map[string]any)
	if _, ok := params["start"].(string); !ok {
		t.Errorf("expected start to be string when using pre-parsed request, got %T", params["start"])
	}
}

func TestProxy_normalizeToolArguments_Availability(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name      string
		input     string
		wantStart float64
		wantEnd   float64
	}{
		{
			name:      "rounds unaligned timestamps to 5-min boundaries",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065601,"end_time":1747065899,"duration_minutes":30,"participants":[]}}}}`,
			wantStart: 1747065600, // floor(1747065601/300)*300
			wantEnd:   1747065900, // ceil(1747065899/300)*300
		},
		{
			name:      "preserves already-aligned timestamps",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065600,"end_time":1747066200,"duration_minutes":30,"participants":[]}}}}`,
			wantStart: 1747065600,
			wantEnd:   1747066200,
		},
		{
			name:      "rounds start down and end up for same unaligned value",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065700,"end_time":1747065700,"duration_minutes":30,"participants":[]}}}}`,
			wantStart: 1747065600,
			wantEnd:   1747065900,
		},
		{
			name:      "handles timestamps at exact 5-min boundary",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065900,"end_time":1747065900,"duration_minutes":30,"participants":[]}}}}`,
			wantStart: 1747065900,
			wantEnd:   1747065900,
		},
		{
			name:      "rounds 1 second past boundary",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":301,"end_time":301,"duration_minutes":30,"participants":[]}}}}`,
			wantStart: 300,
			wantEnd:   600,
		},
		{
			name:      "rounds 1 second before boundary",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":299,"end_time":299,"duration_minutes":30,"participants":[]}}}}`,
			wantStart: 0,
			wantEnd:   300,
		},
		{
			name:      "zero timestamps stay at zero",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":0,"end_time":0,"duration_minutes":30,"participants":[]}}}}`,
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "large realistic timestamps rounded",
			input:     `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1778588147,"end_time":1778674547,"duration_minutes":30,"participants":[]}}}}`,
			wantStart: 1778588100,
			wantEnd:   1778674800,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.normalizeToolArguments([]byte(tt.input), nil)

			var parsed rpcRequest
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("failed to parse result: %v", err)
			}

			req, ok := parsed.Params.Arguments["availability_request"].(map[string]any)
			if !ok {
				t.Fatal("expected availability_request to be a map")
			}

			gotStart, ok := req["start_time"].(float64)
			if !ok {
				t.Fatalf("expected start_time to be float64, got %T", req["start_time"])
			}
			if gotStart != tt.wantStart {
				t.Errorf("start_time = %v, want %v", gotStart, tt.wantStart)
			}

			gotEnd, ok := req["end_time"].(float64)
			if !ok {
				t.Fatalf("expected end_time to be float64, got %T", req["end_time"])
			}
			if gotEnd != tt.wantEnd {
				t.Errorf("end_time = %v, want %v", gotEnd, tt.wantEnd)
			}
		})
	}
}

func TestProxy_normalizeToolArguments_Availability_PartialFields(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name       string
		input      string
		checkStart bool
		checkEnd   bool
		wantStart  float64
		wantEnd    float64
	}{
		{
			name:       "only start_time present",
			input:      `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065601,"duration_minutes":30,"participants":[]}}}}`,
			checkStart: true,
			wantStart:  1747065600,
		},
		{
			name:     "only end_time present",
			input:    `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"end_time":1747065601,"duration_minutes":30,"participants":[]}}}}`,
			checkEnd: true,
			wantEnd:  1747065900,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.normalizeToolArguments([]byte(tt.input), nil)

			var parsed rpcRequest
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Fatalf("failed to parse result: %v", err)
			}

			req := parsed.Params.Arguments["availability_request"].(map[string]any)

			if tt.checkStart {
				gotStart, ok := req["start_time"].(float64)
				if !ok {
					t.Fatalf("expected start_time float64, got %T", req["start_time"])
				}
				if gotStart != tt.wantStart {
					t.Errorf("start_time = %v, want %v", gotStart, tt.wantStart)
				}
			}
			if tt.checkEnd {
				gotEnd, ok := req["end_time"].(float64)
				if !ok {
					t.Fatalf("expected end_time float64, got %T", req["end_time"])
				}
				if gotEnd != tt.wantEnd {
					t.Errorf("end_time = %v, want %v", gotEnd, tt.wantEnd)
				}
			}
		})
	}
}

func TestProxy_normalizeToolArguments_Availability_OtherFieldsUntouched(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065601,"end_time":1747065899,"duration_minutes":30,"interval_minutes":15,"participants":[{"email":"user@example.com"}]}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)

	var parsed rpcRequest
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	req := parsed.Params.Arguments["availability_request"].(map[string]any)

	// duration_minutes should be untouched
	if dur, ok := req["duration_minutes"].(float64); !ok || dur != 30 {
		t.Errorf("duration_minutes = %v, want 30", req["duration_minutes"])
	}

	// interval_minutes should be untouched
	if iv, ok := req["interval_minutes"].(float64); !ok || iv != 15 {
		t.Errorf("interval_minutes = %v, want 15", req["interval_minutes"])
	}

	// participants should be untouched
	participants, ok := req["participants"].([]any)
	if !ok || len(participants) != 1 {
		t.Errorf("participants should have 1 entry, got %v", req["participants"])
	}
}

func TestProxy_normalizeToolArguments_Availability_StringTimestampsIgnored(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	// String timestamps should not be modified (only numeric ones are rounded)
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":"not_a_number","end_time":"also_not","duration_minutes":30,"participants":[]}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)

	// Should be unmodified since toInt64 returns false for strings
	if string(result) != input {
		t.Errorf("expected string timestamps to pass through unchanged")
	}
}

func TestProxy_normalizeToolArguments_Availability_NullTimestamps(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":null,"end_time":null,"duration_minutes":30,"participants":[]}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)

	// null values are not numeric, toInt64 returns false, so no modification
	if string(result) != input {
		t.Errorf("expected null timestamps to pass through unchanged")
	}
}

func TestProxy_normalizeToolArguments_NoOp(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "does not modify list_messages",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{"grant_id":"abc","get_all_query_parameters":{"limit":5}}}}`,
		},
		{
			name:  "does not modify current_time",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"current_time","arguments":{"timezone":"UTC"}}}`,
		},
		{
			name:  "does not modify epoch_to_datetime",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"epoch_to_datetime","arguments":{"batch":[{"epoch_time":1747065600,"timezone":"UTC"}]}}}`,
		},
		{
			name:  "does not modify datetime_to_epoch",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"datetime_to_epoch","arguments":{"batch":[{"date":"2025-05-12","time":"14:00:00","timezone":"UTC"}]}}}`,
		},
		{
			name:  "does not modify create_event",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"create_event","arguments":{"calendar_id":"primary","event_request":{"title":"Test"}}}}`,
		},
		{
			name:  "does not modify send_message",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"send_message","arguments":{"confirmation_hash":"abc","message_request":{"to":[{"email":"t@t.com"}]}}}}`,
		},
		{
			name:  "does not modify create_draft",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"create_draft","arguments":{"draft_request":{"to":[{"email":"t@t.com"}],"subject":"test"}}}}`,
		},
		{
			name:  "does not modify list_contacts",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_contacts","arguments":{"query_parameters":{"limit":10}}}}`,
		},
		{
			name:  "does not modify get_grant",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_grant","arguments":{"email":"user@example.com"}}}`,
		},
		{
			name:  "does not modify get_search_syntax",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_search_syntax","arguments":{"provider":"google"}}}`,
		},
		{
			name:  "does not modify non-tools/call methods",
			input: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		},
		{
			name:  "does not modify tools/list",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		},
		{
			name:  "does not modify notifications",
			input: `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`,
		},
		{
			name:  "handles nil arguments",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events"}}`,
		},
		{
			name:  "handles empty arguments",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{}}}`,
		},
		{
			name:  "handles list_events without get_all_query_parameters",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"grant_id":"abc"}}}`,
		},
		{
			name:  "handles availability without availability_request",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"something":"else"}}}`,
		},
		{
			name:  "handles availability with empty availability_request",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{}}}}`,
		},
		{
			name:  "handles invalid JSON gracefully",
			input: `not json at all`,
		},
		{
			name:  "handles empty string",
			input: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.normalizeToolArguments([]byte(tt.input), nil)
			if string(result) != tt.input {
				t.Errorf("expected no modification\n  input:  %s\n  output: %s", tt.input, string(result))
			}
		})
	}
}

func TestProxy_normalizeToolArguments_CombinedWithGrantInjection(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")
	proxy.SetDefaultGrant("my-grant-123")

	// Simulate the full pipeline: injectDefaultGrant + normalizeToolArguments
	raw := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000,"limit":5}}}}`)

	var req rpcRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatal(err)
	}

	// Apply both transformations like forward() does
	result := proxy.injectDefaultGrant(raw, &req)
	result = proxy.normalizeToolArguments(result, nil) // nil because injectDefaultGrant may have changed the bytes

	var parsed rpcRequest
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Verify grant was injected
	grantID, ok := parsed.Params.Arguments["grant_id"].(string)
	if !ok || grantID != "my-grant-123" {
		t.Errorf("grant_id = %v, want 'my-grant-123'", parsed.Params.Arguments["grant_id"])
	}

	// Verify start/end were coerced to strings
	params := parsed.Params.Arguments["get_all_query_parameters"].(map[string]any)
	if start, ok := params["start"].(string); !ok || start != "1747065600" {
		t.Errorf("start = %v (%T), want string '1747065600'", params["start"], params["start"])
	}
	if end, ok := params["end"].(string); !ok || end != "1747152000" {
		t.Errorf("end = %v (%T), want string '1747152000'", params["end"], params["end"])
	}
}

func TestRoundDown5Min(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input int64
		want  int64
	}{
		{"zero", 0, 0},
		{"exact boundary 300", 300, 300},
		{"just below boundary", 299, 0},
		{"just above boundary", 301, 300},
		{"just below 600", 599, 300},
		{"exact 600", 600, 600},
		{"mid-block 150", 150, 0},
		{"mid-block 450", 450, 300},
		{"realistic timestamp +1s", 1747065601, 1747065600},
		{"realistic timestamp +299s", 1747065899, 1747065600},
		{"realistic timestamp exact", 1747065600, 1747065600},
		{"very large timestamp", 2524608001, 2524608000},
		{"one", 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundDown5Min(tt.input)
			if got != tt.want {
				t.Errorf("roundDown5Min(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestRoundUp5Min(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input int64
		want  int64
	}{
		{"zero", 0, 0},
		{"exact boundary 300", 300, 300},
		{"just below boundary", 299, 300},
		{"just above boundary", 301, 600},
		{"just below 600", 599, 600},
		{"exact 600", 600, 600},
		{"mid-block 150", 150, 300},
		{"mid-block 450", 450, 600},
		{"realistic timestamp +1s", 1747065601, 1747065900},
		{"realistic timestamp +299s", 1747065899, 1747065900},
		{"realistic timestamp exact", 1747065900, 1747065900},
		{"very large timestamp", 2524608001, 2524608300},
		{"one", 1, 300},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := roundUp5Min(tt.input)
			if got != tt.want {
				t.Errorf("roundUp5Min(%d) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		input  any
		want   int64
		wantOK bool
	}{
		{"float64", float64(1747065600), 1747065600, true},
		{"float64 zero", float64(0), 0, true},
		{"float64 negative", float64(-1), -1, true},
		{"float64 large", float64(2524608000), 2524608000, true},
		{"int", int(42), 42, true},
		{"int zero", int(0), 0, true},
		{"int negative", int(-100), -100, true},
		{"int64", int64(999), 999, true},
		{"int64 zero", int64(0), 0, true},
		{"json.Number valid", json.Number("1747065600"), 1747065600, true},
		{"json.Number zero", json.Number("0"), 0, true},
		{"json.Number float", json.Number("1.5"), 0, false},
		{"json.Number empty", json.Number(""), 0, false},
		{"json.Number invalid", json.Number("abc"), 0, false},
		{"string returns false", "not a number", 0, false},
		{"string numeric returns false", "12345", 0, false},
		{"nil returns false", nil, 0, false},
		{"bool true returns false", true, 0, false},
		{"bool false returns false", false, 0, false},
		{"slice returns false", []int{1, 2}, 0, false},
		{"map returns false", map[string]int{"a": 1}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toInt64(tt.input)
			if ok != tt.wantOK {
				t.Errorf("toInt64(%v) ok = %v, want %v", tt.input, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("toInt64(%v) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeListEventsArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     map[string]any
		wantMod  bool
		wantArgs map[string]any
	}{
		{
			name:    "nil get_all_query_parameters",
			args:    map[string]any{"grant_id": "abc"},
			wantMod: false,
		},
		{
			name:    "wrong type get_all_query_parameters",
			args:    map[string]any{"get_all_query_parameters": "not a map"},
			wantMod: false,
		},
		{
			name: "start is already string",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": "12345",
			}},
			wantMod: false,
		},
		{
			name: "start is float64",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": float64(12345),
			}},
			wantMod: true,
			wantArgs: map[string]any{"get_all_query_parameters": map[string]any{
				"start": "12345",
			}},
		},
		{
			name:    "empty params map",
			args:    map[string]any{"get_all_query_parameters": map[string]any{}},
			wantMod: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeListEventsArgs(tt.args)
			if got != tt.wantMod {
				t.Errorf("normalizeListEventsArgs() modified = %v, want %v", got, tt.wantMod)
			}
		})
	}
}

func TestNormalizeAvailabilityArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    map[string]any
		wantMod bool
	}{
		{
			name:    "nil availability_request",
			args:    map[string]any{"other": "stuff"},
			wantMod: false,
		},
		{
			name:    "wrong type availability_request",
			args:    map[string]any{"availability_request": "not a map"},
			wantMod: false,
		},
		{
			name: "already aligned",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": float64(300),
				"end_time":   float64(600),
			}},
			wantMod: false,
		},
		{
			name: "needs rounding",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": float64(301),
				"end_time":   float64(599),
			}},
			wantMod: true,
		},
		{
			name:    "empty availability_request",
			args:    map[string]any{"availability_request": map[string]any{}},
			wantMod: false,
		},
		{
			name: "string timestamps ignored",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": "not_numeric",
				"end_time":   "also_not",
			}},
			wantMod: false,
		},
		{
			name: "boolean timestamps ignored",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": true,
				"end_time":   false,
			}},
			wantMod: false,
		},
		{
			name: "nil timestamps ignored",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": nil,
				"end_time":   nil,
			}},
			wantMod: false,
		},
		{
			name: "array timestamps ignored",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": []any{1, 2, 3},
				"end_time":   []any{4, 5, 6},
			}},
			wantMod: false,
		},
		{
			name: "object timestamps ignored",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": map[string]any{"nested": true},
				"end_time":   map[string]any{"nested": true},
			}},
			wantMod: false,
		},
		{
			name: "only start_time needs rounding",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": float64(301),
				"end_time":   float64(600),
			}},
			wantMod: true,
		},
		{
			name: "only end_time needs rounding",
			args: map[string]any{"availability_request": map[string]any{
				"start_time": float64(300),
				"end_time":   float64(599),
			}},
			wantMod: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeAvailabilityArgs(tt.args)
			if got != tt.wantMod {
				t.Errorf("normalizeAvailabilityArgs() modified = %v, want %v", got, tt.wantMod)
			}
		})
	}
}

func TestNormalizeListEventsArgs_ExoticTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    map[string]any
		wantMod bool
	}{
		{
			name: "boolean start ignored",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": true,
			}},
			wantMod: false,
		},
		{
			name: "null start ignored",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": nil,
			}},
			wantMod: false,
		},
		{
			name: "array start ignored",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": []any{1, 2},
			}},
			wantMod: false,
		},
		{
			name: "object start ignored",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": map[string]any{"nested": "value"},
			}},
			wantMod: false,
		},
		{
			name: "boolean end ignored",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"end": false,
			}},
			wantMod: false,
		},
		{
			name: "null end ignored",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"end": nil,
			}},
			wantMod: false,
		},
		{
			name: "only start is numeric (end is string)",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": float64(100),
				"end":   "200",
			}},
			wantMod: true,
		},
		{
			name: "only end is numeric (start is string)",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": "100",
				"end":   float64(200),
			}},
			wantMod: true,
		},
		{
			name: "negative start coerced",
			args: map[string]any{"get_all_query_parameters": map[string]any{
				"start": float64(-1),
			}},
			wantMod: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeListEventsArgs(tt.args)
			if got != tt.wantMod {
				t.Errorf("normalizeListEventsArgs() modified = %v, want %v", got, tt.wantMod)
			}
		})
	}
}

func TestProxy_normalizeToolArguments_ListEvents_ExoticValues(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name        string
		input       string
		expectNoMod bool
	}{
		{
			name:        "null start and end pass through unchanged",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":null,"end":null}}}}`,
			expectNoMod: true,
		},
		{
			name:        "boolean start and end pass through unchanged",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":true,"end":false}}}}`,
			expectNoMod: true,
		},
		{
			name:        "array start and end pass through unchanged",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":[1,2],"end":[3,4]}}}}`,
			expectNoMod: true,
		},
		{
			name:        "object start and end pass through unchanged",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":{"v":1},"end":{"v":2}}}}}`,
			expectNoMod: true,
		},
		{
			name:        "get_all_query_parameters is a string",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":"not_a_map"}}}`,
			expectNoMod: true,
		},
		{
			name:        "get_all_query_parameters is null",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":null}}}`,
			expectNoMod: true,
		},
		{
			name:        "get_all_query_parameters is an array",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":[1,2,3]}}}`,
			expectNoMod: true,
		},
		{
			name:        "get_all_query_parameters is a number",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":42}}}`,
			expectNoMod: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.normalizeToolArguments([]byte(tt.input), nil)
			if tt.expectNoMod && string(result) != tt.input {
				t.Errorf("expected no modification\n  input:  %s\n  output: %s", tt.input, string(result))
			}
		})
	}
}

func TestProxy_normalizeToolArguments_Availability_ExoticValues(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	tests := []struct {
		name        string
		input       string
		expectNoMod bool
	}{
		{
			name:        "boolean start_time and end_time pass through",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":true,"end_time":false,"duration_minutes":30,"participants":[]}}}}`,
			expectNoMod: true,
		},
		{
			name:        "null start_time and end_time pass through",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":null,"end_time":null,"duration_minutes":30,"participants":[]}}}}`,
			expectNoMod: true,
		},
		{
			name:        "array start_time and end_time pass through",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":[1],"end_time":[2],"duration_minutes":30,"participants":[]}}}}`,
			expectNoMod: true,
		},
		{
			name:        "object start_time and end_time pass through",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":{"v":1},"end_time":{"v":2},"duration_minutes":30,"participants":[]}}}}`,
			expectNoMod: true,
		},
		{
			name:        "availability_request is a string",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":"not_a_map"}}}`,
			expectNoMod: true,
		},
		{
			name:        "availability_request is null",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":null}}}`,
			expectNoMod: true,
		},
		{
			name:        "availability_request is an array",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":[1,2]}}}`,
			expectNoMod: true,
		},
		{
			name:        "availability_request is a number",
			input:       `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":99}}}`,
			expectNoMod: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.normalizeToolArguments([]byte(tt.input), nil)
			if tt.expectNoMod && string(result) != tt.input {
				t.Errorf("expected no modification\n  input:  %s\n  output: %s", tt.input, string(result))
			}
		})
	}
}

func TestProxy_normalizeToolArguments_CrossToolNoOp(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	// These tools also use get_all_query_parameters but should NOT have start/end coerced
	// (normalization only targets list_events)
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "list_messages with get_all_query_parameters is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_messages","arguments":{"get_all_query_parameters":{"limit":5,"received_after":1747065600}}}}`,
		},
		{
			name:  "list_threads with get_all_query_parameters is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_threads","arguments":{"get_all_query_parameters":{"limit":5}}}}`,
		},
		{
			name:  "list_contacts with query_parameters is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_contacts","arguments":{"query_parameters":{"limit":5}}}}`,
		},
		{
			name:  "list_folders is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_folders","arguments":{"query_parameters":{"limit":50}}}}`,
		},
		{
			name:  "get_event is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get_event","arguments":{"calendar_id":"primary","event_id":"evt-123"}}}`,
		},
		{
			name:  "create_event is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"create_event","arguments":{"calendar_id":"primary","event_request":{"title":"Test","when":{"start_time":1747065601,"end_time":1747065899}}}}}`,
		},
		{
			name:  "update_event is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"update_event","arguments":{"calendar_id":"primary","event_id":"evt-1","event_request":{"title":"Updated"}}}}`,
		},
		{
			name:  "delete_event is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"delete_event","arguments":{"calendar_id":"primary","event_id":"evt-1"}}}`,
		},
		{
			name:  "list_calendars is not modified",
			input: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_calendars","arguments":{}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := proxy.normalizeToolArguments([]byte(tt.input), nil)
			if string(result) != tt.input {
				t.Errorf("expected no modification for %s\n  input:  %s\n  output: %s", tt.name, tt.input, string(result))
			}
		})
	}
}

func TestProxy_normalizeToolArguments_EmptyToolName(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"","arguments":{"get_all_query_parameters":{"start":12345}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)
	if string(result) != input {
		t.Error("expected empty tool name to cause no modification")
	}
}

func TestProxy_normalizeToolArguments_MissingToolName(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"arguments":{"get_all_query_parameters":{"start":12345}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)
	if string(result) != input {
		t.Error("expected missing tool name to cause no modification")
	}
}

func TestProxy_normalizeToolArguments_ConcurrentSafety(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	listEventsInput := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"start":1747065600,"end":1747152000}}}}`
	availInput := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065601,"end_time":1747065899,"duration_minutes":30,"participants":[]}}}}`

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			result := proxy.normalizeToolArguments([]byte(listEventsInput), nil)
			var parsed rpcRequest
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Errorf("concurrent list_events parse failed: %v", err)
				return
			}
			params, ok := parsed.Params.Arguments["get_all_query_parameters"].(map[string]any)
			if !ok {
				t.Error("concurrent list_events: missing get_all_query_parameters")
				return
			}
			if _, ok := params["start"].(string); !ok {
				t.Errorf("concurrent list_events: start not coerced, got %T", params["start"])
			}
		}()
		go func() {
			defer wg.Done()
			result := proxy.normalizeToolArguments([]byte(availInput), nil)
			var parsed rpcRequest
			if err := json.Unmarshal(result, &parsed); err != nil {
				t.Errorf("concurrent availability parse failed: %v", err)
				return
			}
			req, ok := parsed.Params.Arguments["availability_request"].(map[string]any)
			if !ok {
				t.Error("concurrent availability: missing availability_request")
				return
			}
			st, ok := req["start_time"].(float64)
			if !ok {
				t.Errorf("concurrent availability: start_time not float64, got %T", req["start_time"])
				return
			}
			if st != 1747065600 {
				t.Errorf("concurrent availability: start_time = %v, want 1747065600", st)
			}
		}()
	}
	wg.Wait()
}

func TestProxy_normalizeToolArguments_Idempotent(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	// Applying normalization twice should produce the same result
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"calendar_id":"primary","start":1747065600,"end":1747152000}}}}`

	first := proxy.normalizeToolArguments([]byte(input), nil)
	second := proxy.normalizeToolArguments(first, nil)

	if string(first) != string(second) {
		t.Errorf("normalization is not idempotent\n  first:  %s\n  second: %s", first, second)
	}

	// Same for availability
	availInput := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"availability","arguments":{"availability_request":{"start_time":1747065601,"end_time":1747065899,"duration_minutes":30,"participants":[]}}}}`

	first = proxy.normalizeToolArguments([]byte(availInput), nil)
	second = proxy.normalizeToolArguments(first, nil)

	if string(first) != string(second) {
		t.Errorf("availability normalization is not idempotent\n  first:  %s\n  second: %s", first, second)
	}
}

func TestProxy_normalizeToolArguments_PreservesJSONRPCFields(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	input := `{"jsonrpc":"2.0","id":42,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"start":1747065600,"end":1747152000}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want '2.0'", parsed["jsonrpc"])
	}
	if parsed["id"] != float64(42) {
		t.Errorf("id = %v, want 42", parsed["id"])
	}
	if parsed["method"] != "tools/call" {
		t.Errorf("method = %v, want 'tools/call'", parsed["method"])
	}
}

func TestProxy_normalizeToolArguments_PreservesStringID(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	input := `{"jsonrpc":"2.0","id":"request-abc","method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"start":1747065600}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed["id"] != "request-abc" {
		t.Errorf("id = %v, want 'request-abc'", parsed["id"])
	}
}

func TestProxy_normalizeToolArguments_PreservesNullID(t *testing.T) {
	t.Parallel()

	proxy := NewProxy("test-api-key", "us")

	input := `{"jsonrpc":"2.0","id":null,"method":"tools/call","params":{"name":"list_events","arguments":{"get_all_query_parameters":{"start":1747065600}}}}`
	result := proxy.normalizeToolArguments([]byte(input), nil)

	var parsed map[string]any
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if parsed["id"] != nil {
		t.Errorf("id = %v, want nil", parsed["id"])
	}
}
