package common

import (
	"testing"

	"github.com/fatih/color"
)

func TestStatusColor(t *testing.T) {
	// Disable colors for consistent test output
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name      string
		status    string
		wantColor *color.Color
	}{
		{"active is green", "active", Green},
		{"confirmed is green", "confirmed", Green},
		{"complete is green", "complete", Green},
		{"attending is green", "attending", Green},
		{"pending is yellow", "pending", Yellow},
		{"scheduled is yellow", "scheduled", Yellow},
		{"inactive is yellow", "inactive", Yellow},
		{"failed is red", "failed", Red},
		{"failing is red", "failing", Red},
		{"error is red", "error", Red},
		{"connecting is cyan", "connecting", Cyan},
		{"waiting is cyan", "waiting", Cyan},
		{"processing is cyan", "processing", Cyan},
		{"rescheduled is cyan", "rescheduled", Cyan},
		{"waiting_for_entry is cyan", "waiting_for_entry", Cyan},
		{"media_processing is cyan", "media_processing", Cyan},
		{"cancelled is dim", "cancelled", Dim},
		{"deleted is dim", "deleted", Dim},
		{"archived is dim", "archived", Dim},
		{"unknown is reset", "unknown_status", Reset},
		{"empty is reset", "", Reset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusColor(tt.status)
			if got != tt.wantColor {
				t.Errorf("StatusColor(%q) returned wrong color", tt.status)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name   string
		status string
		want   string
	}{
		{"active shows filled dot", "active", "●"},
		{"pending shows filled dot", "pending", "●"},
		{"failed shows filled dot", "failed", "●"},
		{"unknown shows empty dot", "unknown", "○"},
		{"empty shows empty dot", "", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusIcon(tt.status)
			if got != tt.want {
				t.Errorf("StatusIcon(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestColorSprint(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	tests := []struct {
		name   string
		status string
		want   string
	}{
		{"renders status text", "active", "active"},
		{"renders pending text", "pending", "pending"},
		{"renders unknown text", "unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ColorSprint(tt.status)
			if got != tt.want {
				t.Errorf("ColorSprint(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
