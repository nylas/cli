package util_test

import (
	"testing"

	"github.com/nylas/cli/internal/util"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		fn       func(int) int
		expected []int
	}{
		{
			name:     "double values",
			input:    []int{1, 2, 3, 4},
			fn:       func(x int) int { return x * 2 },
			expected: []int{2, 4, 6, 8},
		},
		{
			name:     "nil slice",
			input:    nil,
			fn:       func(x int) int { return x * 2 },
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    []int{},
			fn:       func(x int) int { return x * 2 },
			expected: []int{},
		},
		{
			name:     "transform type",
			input:    []int{1, 2, 3},
			fn:       func(x int) int { return x + 10 },
			expected: []int{11, 12, 13},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.Map(tt.input, tt.fn)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMap_DifferentTypes(t *testing.T) {
	// Test string to int transformation
	input := []string{"a", "bb", "ccc"}
	result := util.Map(input, func(s string) int { return len(s) })
	assert.Equal(t, []int{1, 2, 3}, result)
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name      string
		input     []int
		predicate func(int) bool
		expected  []int
	}{
		{
			name:      "filter even numbers",
			input:     []int{1, 2, 3, 4, 5, 6},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  []int{2, 4, 6},
		},
		{
			name:      "filter greater than 3",
			input:     []int{1, 2, 3, 4, 5},
			predicate: func(x int) bool { return x > 3 },
			expected:  []int{4, 5},
		},
		{
			name:      "nil slice",
			input:     nil,
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  nil,
		},
		{
			name:      "empty slice",
			input:     []int{},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  []int{},
		},
		{
			name:      "no matches",
			input:     []int{1, 3, 5},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  []int{},
		},
		{
			name:      "all match",
			input:     []int{2, 4, 6},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  []int{2, 4, 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.Filter(tt.input, tt.predicate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReduce(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		initial  int
		reduce   func(int, int) int
		expected int
	}{
		{
			name:     "sum",
			input:    []int{1, 2, 3, 4, 5},
			initial:  0,
			reduce:   func(acc, x int) int { return acc + x },
			expected: 15,
		},
		{
			name:     "product",
			input:    []int{1, 2, 3, 4},
			initial:  1,
			reduce:   func(acc, x int) int { return acc * x },
			expected: 24,
		},
		{
			name:     "empty slice",
			input:    []int{},
			initial:  10,
			reduce:   func(acc, x int) int { return acc + x },
			expected: 10,
		},
		{
			name:     "nil slice",
			input:    nil,
			initial:  5,
			reduce:   func(acc, x int) int { return acc + x },
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.Reduce(tt.input, tt.initial, tt.reduce)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		value    string
		expected bool
	}{
		{
			name:     "contains value",
			slice:    []string{"foo", "bar", "baz"},
			value:    "bar",
			expected: true,
		},
		{
			name:     "does not contain value",
			slice:    []string{"foo", "bar", "baz"},
			value:    "qux",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			value:    "foo",
			expected: false,
		},
		{
			name:     "nil slice",
			slice:    nil,
			value:    "foo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.Contains(tt.slice, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPartition(t *testing.T) {
	tests := []struct {
		name          string
		input         []int
		predicate     func(int) bool
		expectedTrue  []int
		expectedFalse []int
	}{
		{
			name:          "partition even and odd",
			input:         []int{1, 2, 3, 4, 5, 6},
			predicate:     func(x int) bool { return x%2 == 0 },
			expectedTrue:  []int{2, 4, 6},
			expectedFalse: []int{1, 3, 5},
		},
		{
			name:          "all true",
			input:         []int{2, 4, 6},
			predicate:     func(x int) bool { return x%2 == 0 },
			expectedTrue:  []int{2, 4, 6},
			expectedFalse: []int{},
		},
		{
			name:          "all false",
			input:         []int{1, 3, 5},
			predicate:     func(x int) bool { return x%2 == 0 },
			expectedTrue:  []int{},
			expectedFalse: []int{1, 3, 5},
		},
		{
			name:          "empty slice",
			input:         []int{},
			predicate:     func(x int) bool { return x%2 == 0 },
			expectedTrue:  []int{},
			expectedFalse: []int{},
		},
		{
			name:          "nil slice",
			input:         nil,
			predicate:     func(x int) bool { return x%2 == 0 },
			expectedTrue:  nil,
			expectedFalse: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trueSlice, falseSlice := util.Partition(tt.input, tt.predicate)
			assert.Equal(t, tt.expectedTrue, trueSlice)
			assert.Equal(t, tt.expectedFalse, falseSlice)
		})
	}
}

func TestFind(t *testing.T) {
	tests := []struct {
		name          string
		input         []int
		predicate     func(int) bool
		expectedValue int
		expectedFound bool
	}{
		{
			name:          "find first even",
			input:         []int{1, 3, 4, 5, 6},
			predicate:     func(x int) bool { return x%2 == 0 },
			expectedValue: 4,
			expectedFound: true,
		},
		{
			name:          "not found",
			input:         []int{1, 3, 5},
			predicate:     func(x int) bool { return x%2 == 0 },
			expectedValue: 0,
			expectedFound: false,
		},
		{
			name:          "empty slice",
			input:         []int{},
			predicate:     func(x int) bool { return x%2 == 0 },
			expectedValue: 0,
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, found := util.Find(tt.input, tt.predicate)
			assert.Equal(t, tt.expectedValue, value)
			assert.Equal(t, tt.expectedFound, found)
		})
	}
}

func TestAny(t *testing.T) {
	tests := []struct {
		name      string
		input     []int
		predicate func(int) bool
		expected  bool
	}{
		{
			name:      "has even number",
			input:     []int{1, 3, 4, 5},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  true,
		},
		{
			name:      "no even numbers",
			input:     []int{1, 3, 5},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  false,
		},
		{
			name:      "empty slice",
			input:     []int{},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.Any(tt.input, tt.predicate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAll(t *testing.T) {
	tests := []struct {
		name      string
		input     []int
		predicate func(int) bool
		expected  bool
	}{
		{
			name:      "all even",
			input:     []int{2, 4, 6},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  true,
		},
		{
			name:      "not all even",
			input:     []int{2, 3, 4},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  false,
		},
		{
			name:      "empty slice",
			input:     []int{},
			predicate: func(x int) bool { return x%2 == 0 },
			expected:  true, // vacuous truth
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := util.All(tt.input, tt.predicate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Benchmark tests
func BenchmarkMap(b *testing.B) {
	input := make([]int, 1000)
	for i := range input {
		input[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = util.Map(input, func(x int) int { return x * 2 })
	}
}

func BenchmarkFilter(b *testing.B) {
	input := make([]int, 1000)
	for i := range input {
		input[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = util.Filter(input, func(x int) bool { return x%2 == 0 })
	}
}
