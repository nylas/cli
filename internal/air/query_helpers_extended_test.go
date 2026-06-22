//go:build !integration
// +build !integration

package air

import (
	"net/url"
	"testing"
)

// =============================================================================
// Extended Query Helpers Tests
// =============================================================================

// TestQueryParams_EdgeCases tests edge cases for query parameter parsing.
func TestQueryParams_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		checkFunc func(q *QueryParams) bool
		desc      string
	}{
		{
			name:  "multiple values same key",
			query: "tag=a&tag=b&tag=c",
			checkFunc: func(q *QueryParams) bool {
				// Get returns first value
				return q.Get("tag") == "a"
			},
			desc: "should return first value",
		},
		{
			name:  "url encoded value",
			query: "q=hello%20world",
			checkFunc: func(q *QueryParams) bool {
				return q.Get("q") == "hello world"
			},
			desc: "should decode URL encoding",
		},
		{
			name:  "special characters",
			query: "q=test%2B1%3D2",
			checkFunc: func(q *QueryParams) bool {
				return q.Get("q") == "test+1=2"
			},
			desc: "should handle special characters",
		},
		{
			name:  "unicode value",
			query: "name=%E4%B8%AD%E6%96%87",
			checkFunc: func(q *QueryParams) bool {
				return q.Get("name") != ""
			},
			desc: "should handle unicode",
		},
		{
			name:  "ampersand in value",
			query: "q=a%26b",
			checkFunc: func(q *QueryParams) bool {
				return q.Get("q") == "a&b"
			},
			desc: "should handle escaped ampersand",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			q := NewQueryParams(values)

			if !tt.checkFunc(q) {
				t.Errorf("test %s failed: %s", tt.name, tt.desc)
			}
		})
	}
}

// TestQueryParams_GetInt_BoundaryConditions tests integer parsing boundaries.
func TestQueryParams_GetInt_BoundaryConditions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal int
		minVal     int
		maxVal     int
		want       int
	}{
		// Boundary at exact limits
		{"exact min", "val=0", "val", 50, 0, 100, 0},
		{"exact max", "val=100", "val", 50, 0, 100, 100},

		// Just inside limits
		{"just above min", "val=1", "val", 50, 0, 100, 1},
		{"just below max", "val=99", "val", 50, 0, 100, 99},

		// Just outside limits
		{"just below min", "val=-1", "val", 50, 0, 100, 50},
		{"just above max", "val=101", "val", 50, 0, 100, 50},

		// Large numbers
		{"large valid", "val=1000000", "val", 0, 0, 2000000, 1000000},
		{"large invalid", "val=9999999999999999999999", "val", 42, 0, 100, 42},

		// Negative ranges
		{"negative range valid", "val=-5", "val", 0, -10, 10, -5},
		{"negative range below", "val=-15", "val", 0, -10, 10, 0},

		// Single value range
		{"single value min=max valid", "val=5", "val", 0, 5, 5, 5},
		{"single value min=max invalid", "val=6", "val", 0, 5, 5, 0},

		// Float values (should fail)
		{"float value", "val=5.5", "val", 10, 0, 100, 10},

		// Whitespace
		{"leading whitespace", "val= 50", "val", 10, 0, 100, 10},
		{"trailing whitespace", "val=50 ", "val", 10, 0, 100, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			q := NewQueryParams(values)
			got := q.GetInt(tt.key, tt.defaultVal, tt.minVal, tt.maxVal)
			if got != tt.want {
				t.Errorf("GetInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestQueryParams_GetLimit_Comprehensive tests limit parsing comprehensively.
func TestQueryParams_GetLimit_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		defaultVal int
		want       int
	}{
		// Valid limits
		{"min limit", "limit=1", 50, 1},
		{"max limit", "limit=200", 50, 200},
		{"mid limit", "limit=100", 50, 100},

		// Invalid limits
		{"zero", "limit=0", 50, 50},
		{"negative", "limit=-10", 50, 50},
		{"over max", "limit=201", 50, 50},
		{"way over max", "limit=1000", 50, 50},

		// Edge cases
		{"empty", "limit=", 100, 100},
		{"missing", "", 75, 75},
		{"non-numeric", "limit=many", 50, 50},

		// Different defaults
		{"default 25", "limit=invalid", 25, 25},
		{"default 100", "limit=invalid", 100, 100},
		{"default 200", "limit=invalid", 200, 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			q := NewQueryParams(values)
			got := q.GetLimit(tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetLimit() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestQueryParams_GetInt64_Comprehensive tests int64 parsing.
func TestQueryParams_GetInt64_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal int64
		want       int64
	}{
		// Valid timestamps
		{"recent timestamp", "ts=1704067200", "ts", 0, 1704067200},
		{"zero timestamp", "ts=0", "ts", 100, 0},
		{"large timestamp", "ts=9999999999999", "ts", 0, 9999999999999},

		// Negative values
		{"negative", "ts=-1000", "ts", 0, -1000},

		// Invalid values
		{"non-numeric", "ts=abc", "ts", 500, 500},
		{"empty", "ts=", "ts", 500, 500},
		{"missing", "", "ts", 123, 123},
		{"float", "ts=123.456", "ts", 100, 100},

		// Overflow (should return default)
		{"overflow", "ts=99999999999999999999", "ts", 42, 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			q := NewQueryParams(values)
			got := q.GetInt64(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetInt64() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestQueryParams_GetBool_Comprehensive tests boolean parsing.
func TestQueryParams_GetBool_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
		key   string
		want  bool
	}{
		// True cases
		{"lowercase true", "flag=true", "flag", true},

		// False cases (anything not exactly "true")
		{"lowercase false", "flag=false", "flag", false},
		{"uppercase TRUE", "flag=TRUE", "flag", false},
		{"uppercase FALSE", "flag=FALSE", "flag", false},
		{"mixed case True", "flag=True", "flag", false},
		{"numeric 1", "flag=1", "flag", false},
		{"numeric 0", "flag=0", "flag", false},
		{"yes", "flag=yes", "flag", false},
		{"no", "flag=no", "flag", false},
		{"on", "flag=on", "flag", false},
		{"off", "flag=off", "flag", false},
		{"empty", "flag=", "flag", false},
		{"missing", "", "flag", false},
		{"random string", "flag=xyz", "flag", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			q := NewQueryParams(values)
			got := q.GetBool(tt.key)
			if got != tt.want {
				t.Errorf("GetBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestQueryParams_GetString_Comprehensive tests string parsing.
func TestQueryParams_GetString_Comprehensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		key        string
		defaultVal string
		want       string
	}{
		// Present values
		{"simple value", "name=john", "name", "default", "john"},
		{"with spaces", "name=john+doe", "name", "default", "john doe"},
		{"url encoded", "name=john%20doe", "name", "default", "john doe"},

		// Default cases
		{"empty value", "name=", "name", "default", "default"},
		{"missing key", "", "name", "default", "default"},
		{"different key", "other=value", "name", "default", "default"},

		// Special values
		{"equals sign", "expr=a%3Db", "expr", "", "a=b"},
		{"ampersand", "query=a%26b", "query", "", "a&b"},
		{"unicode", "text=%E4%B8%AD%E6%96%87", "text", "", "\u4e2d\u6587"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values, _ := url.ParseQuery(tt.query)
			q := NewQueryParams(values)
			got := q.GetString(tt.key, tt.defaultVal)
			if got != tt.want {
				t.Errorf("GetString() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestQueryParams_Get covers the raw Get accessor, including the missing-key
// case where it must return an empty string.
func TestQueryParams_Get(t *testing.T) {
	t.Parallel()

	values, _ := url.ParseQuery("foo=bar&empty=")
	q := NewQueryParams(values)

	if got := q.Get("foo"); got != "bar" {
		t.Errorf("Get(foo) = %q, want %q", got, "bar")
	}
	if got := q.Get("empty"); got != "" {
		t.Errorf("Get(empty) = %q, want empty", got)
	}
	if got := q.Get("missing"); got != "" {
		t.Errorf("Get(missing) = %q, want empty", got)
	}
}

// TestNewQueryParams tests QueryParams construction.
func TestNewQueryParams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values url.Values
	}{
		{"nil values", nil},
		{"empty values", url.Values{}},
		{"single value", url.Values{"key": {"value"}}},
		{"multiple values", url.Values{"a": {"1"}, "b": {"2"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewQueryParams(tt.values)
			if q == nil {
				t.Error("NewQueryParams() returned nil")
			}
		})
	}
}
