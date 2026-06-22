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
