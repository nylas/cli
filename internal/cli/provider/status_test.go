package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStatusDots(t *testing.T) {
	tests := []struct {
		name    string
		label   string
		wantLen int
	}{
		{"short label", "API", 27},
		{"long label", "This is a very long label that exceeds thirty", 1},
		{"exact length", "123456789012345678901234567890", 1},
		{"empty label", "", 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dots := statusDots(tt.label)
			assert.Len(t, dots, tt.wantLen)
			for _, c := range dots {
				assert.Equal(t, '.', c)
			}
		})
	}
}

func TestCheckAndPrint(t *testing.T) {
	t.Run("prints check mark for true", func(t *testing.T) {
		// Should not panic
		checkAndPrint("Test Label", func() bool { return true })
	})

	t.Run("prints cross for false", func(t *testing.T) {
		// Should not panic
		checkAndPrint("Test Label", func() bool { return false })
	})
}

func TestPrintUnknown(t *testing.T) {
	// Should not panic
	printUnknown("OAuth Consent Screen")
	printUnknown("OAuth Credentials")
}

func TestNewStatusCmd(t *testing.T) {
	cmd := newStatusCmd()
	assert.Equal(t, "status google", cmd.Use)

	// project-id flag should exist
	f := cmd.Flags().Lookup("project-id")
	assert.NotNil(t, f)
}
