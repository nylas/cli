package mcp

import (
	"encoding/json"
	"testing"
)

func TestSuccessResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		id     any
		result any
	}{
		{
			name:   "map result with string id",
			id:     "1",
			result: map[string]any{"key": "value", "count": float64(3)},
		},
		{
			name:   "nil id",
			id:     nil,
			result: map[string]any{"status": "ok"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := successResponse(tt.id, tt.result)

			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if got["jsonrpc"] != "2.0" {
				t.Errorf("jsonrpc = %v, want 2.0", got["jsonrpc"])
			}
			if got["id"] != tt.id {
				t.Errorf("id = %v, want %v", got["id"], tt.id)
			}
			if _, ok := got["result"]; !ok {
				t.Error("result field missing")
			}
			if _, ok := got["error"]; ok {
				t.Error("error field should not be present in success response")
			}
		})
	}
}

func TestErrorResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      any
		code    int
		message string
	}{
		{
			name:    "specific code and message",
			id:      "42",
			code:    codeMethodNotFound,
			message: "method not found",
		},
		{
			name:    "nil id",
			id:      nil,
			code:    codeInternalError,
			message: "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := errorResponse(tt.id, tt.code, tt.message)

			var got map[string]any
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}

			if got["jsonrpc"] != "2.0" {
				t.Errorf("jsonrpc = %v, want 2.0", got["jsonrpc"])
			}
			if got["id"] != tt.id {
				t.Errorf("id = %v, want %v", got["id"], tt.id)
			}
			if _, ok := got["result"]; ok {
				t.Error("result field should not be present in error response")
			}

			errObj, ok := got["error"].(map[string]any)
			if !ok {
				t.Fatal("error field missing or wrong type")
			}
			if int(errObj["code"].(float64)) != tt.code {
				t.Errorf("error.code = %v, want %v", errObj["code"], tt.code)
			}
			if errObj["message"] != tt.message {
				t.Errorf("error.message = %v, want %v", errObj["message"], tt.message)
			}
		})
	}
}

func TestToolSuccess(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data any
	}{
		{
			name: "map result",
			data: map[string]any{"id": "abc", "subject": "Hello"},
		},
		{
			name: "array result",
			data: []any{"item1", "item2", "item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := toolSuccess(tt.data)

			if resp == nil {
				t.Fatal("expected non-nil response")
			}
			if resp.IsError {
				t.Error("IsError should be false for successful response")
			}
			if len(resp.Content) != 1 {
				t.Fatalf("content length = %d, want 1", len(resp.Content))
			}

			block := resp.Content[0]
			if block.Type != "text" {
				t.Errorf("content[0].type = %q, want \"text\"", block.Type)
			}
			if block.Text == "" {
				t.Error("content[0].text should not be empty")
			}

			// Verify Text is valid JSON representation of the original data.
			expectedJSON, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("marshal test data: %v", err)
			}
			if block.Text != string(expectedJSON) {
				t.Errorf("content[0].text = %q, want %q", block.Text, string(expectedJSON))
			}
		})
	}
}

func TestToolError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "non-empty message",
			message: "something went wrong",
		},
		{
			name:    "empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := toolError(tt.message)

			if resp == nil {
				t.Fatal("expected non-nil response")
			}
			if !resp.IsError {
				t.Error("IsError should be true")
			}
			if len(resp.Content) != 1 {
				t.Fatalf("content length = %d, want 1", len(resp.Content))
			}

			block := resp.Content[0]
			if block.Type != "text" {
				t.Errorf("content[0].type = %q, want \"text\"", block.Type)
			}
			if block.Text != tt.message {
				t.Errorf("content[0].text = %q, want %q", block.Text, tt.message)
			}
		})
	}
}

func TestGetString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       map[string]any
		key        string
		defaultVal string
		want       string
	}{
		{
			name:       "key exists with non-empty value",
			args:       map[string]any{"name": "Alice"},
			key:        "name",
			defaultVal: "default",
			want:       "Alice",
		},
		{
			name:       "key exists with empty value returns default",
			args:       map[string]any{"name": ""},
			key:        "name",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "key does not exist returns default",
			args:       map[string]any{"other": "value"},
			key:        "name",
			defaultVal: "default",
			want:       "default",
		},
		{
			name:       "key exists but is non-string returns default",
			args:       map[string]any{"name": 42},
			key:        "name",
			defaultVal: "default",
			want:       "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getString(tt.args, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       map[string]any
		key        string
		defaultVal int
		want       int
	}{
		{
			name:       "key exists as float64",
			args:       map[string]any{"limit": float64(50)},
			key:        "limit",
			defaultVal: 10,
			want:       50,
		},
		{
			name:       "key does not exist returns default",
			args:       map[string]any{"other": float64(5)},
			key:        "limit",
			defaultVal: 10,
			want:       10,
		},
		{
			name:       "key is string wrong type returns default",
			args:       map[string]any{"limit": "50"},
			key:        "limit",
			defaultVal: 10,
			want:       10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getInt(tt.args, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       map[string]any
		key        string
		defaultVal int64
		want       int64
	}{
		{
			name:       "key exists as float64",
			args:       map[string]any{"timestamp": float64(1700000000)},
			key:        "timestamp",
			defaultVal: 0,
			want:       1700000000,
		},
		{
			name:       "key does not exist returns default",
			args:       map[string]any{"other": float64(1)},
			key:        "timestamp",
			defaultVal: int64(99),
			want:       99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getInt64(tt.args, tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("getInt64() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestGetBool(t *testing.T) {
	t.Parallel()

	trueVal := true
	falseVal := false

	tests := []struct {
		name string
		args map[string]any
		key  string
		want *bool
	}{
		{
			name: "key exists as true",
			args: map[string]any{"unread": true},
			key:  "unread",
			want: &trueVal,
		},
		{
			name: "key exists as false",
			args: map[string]any{"unread": false},
			key:  "unread",
			want: &falseVal,
		},
		{
			name: "key does not exist returns nil",
			args: map[string]any{"other": true},
			key:  "unread",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getBool(tt.args, tt.key)
			if tt.want == nil {
				if got != nil {
					t.Errorf("getBool() = %v, want nil", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("getBool() = nil, want non-nil")
			}
			if *got != *tt.want {
				t.Errorf("getBool() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestGetStringSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args map[string]any
		key  string
		want []string
	}{
		{
			name: "key exists as slice of strings",
			args: map[string]any{"tags": []any{"inbox", "work", "important"}},
			key:  "tags",
			want: []string{"inbox", "work", "important"},
		},
		{
			name: "key does not exist returns nil",
			args: map[string]any{"other": []any{"a"}},
			key:  "tags",
			want: nil,
		},
		{
			name: "key is wrong type returns nil",
			args: map[string]any{"tags": "inbox"},
			key:  "tags",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := getStringSlice(tt.args, tt.key)
			if tt.want == nil {
				if got != nil {
					t.Errorf("getStringSlice() = %v, want nil", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("getStringSlice() len = %d, want %d", len(got), len(tt.want))
			}
			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("getStringSlice()[%d] = %q, want %q", i, got[i], v)
				}
			}
		})
	}
}
