package common

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestIsDeprecatedConnectorProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		want     bool
	}{
		{name: "matches inbox", provider: "inbox", want: true},
		{name: "matches inbox case insensitive", provider: "Inbox", want: true},
		{name: "ignores whitespace", provider: " inbox ", want: true},
		{name: "allows google", provider: "google", want: false},
		{name: "allows empty", provider: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDeprecatedConnectorProvider(tt.provider); got != tt.want {
				t.Fatalf("IsDeprecatedConnectorProvider(%q) = %t, want %t", tt.provider, got, tt.want)
			}
		})
	}
}

func TestFilterVisibleConnectors(t *testing.T) {
	connectors := []domain.Connector{
		{Provider: "google", Name: "Google"},
		{Provider: "inbox", Name: "Inbox"},
		{Provider: "microsoft", Name: "Microsoft"},
	}

	filtered := FilterVisibleConnectors(connectors)
	if len(filtered) != 2 {
		t.Fatalf("FilterVisibleConnectors() returned %d connectors, want 2", len(filtered))
	}
	if filtered[0].Provider != "google" || filtered[1].Provider != "microsoft" {
		t.Fatalf("FilterVisibleConnectors() returned %#v", filtered)
	}
}

func TestValidateSupportedConnectorProvider(t *testing.T) {
	if err := ValidateSupportedConnectorProvider("google"); err != nil {
		t.Fatalf("ValidateSupportedConnectorProvider(google) returned error: %v", err)
	}

	err := ValidateSupportedConnectorProvider("inbox")
	if err == nil {
		t.Fatal("ValidateSupportedConnectorProvider(inbox) returned nil, want error")
	}
	if err.Error() != "invalid provider: inbox" {
		t.Fatalf("ValidateSupportedConnectorProvider(inbox) error = %q, want %q", err.Error(), "invalid provider: inbox")
	}
}
