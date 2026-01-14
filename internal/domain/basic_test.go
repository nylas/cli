package domain

import (
	"testing"
)

// TestProvider tests the Provider type and its methods.
func TestProvider(t *testing.T) {
	t.Run("DisplayName", func(t *testing.T) {
		tests := []struct {
			provider Provider
			want     string
		}{
			{ProviderGoogle, "Google"},
			{ProviderMicrosoft, "Microsoft"},
			{ProviderIMAP, "IMAP"},
			{ProviderVirtual, "Virtual"},
			{Provider("unknown"), "unknown"},
		}

		for _, tt := range tests {
			got := tt.provider.DisplayName()
			if got != tt.want {
				t.Errorf("Provider(%q).DisplayName() = %q, want %q", tt.provider, got, tt.want)
			}
		}
	})

	t.Run("IsValid", func(t *testing.T) {
		tests := []struct {
			provider Provider
			want     bool
		}{
			{ProviderGoogle, true},
			{ProviderMicrosoft, true},
			{ProviderIMAP, true},
			{ProviderVirtual, true},
			{Provider("unknown"), false},
			{Provider(""), false},
			{Provider("GOOGLE"), false}, // Case sensitive
		}

		for _, tt := range tests {
			got := tt.provider.IsValid()
			if got != tt.want {
				t.Errorf("Provider(%q).IsValid() = %v, want %v", tt.provider, got, tt.want)
			}
		}
	})

	t.Run("ParseProvider", func(t *testing.T) {
		tests := []struct {
			input   string
			want    Provider
			wantErr bool
		}{
			{"google", ProviderGoogle, false},
			{"microsoft", ProviderMicrosoft, false},
			{"imap", ProviderIMAP, false},
			{"virtual", ProviderVirtual, false},
			{"unknown", "", true},
			{"", "", true},
			{"GOOGLE", "", true}, // Case sensitive
		}

		for _, tt := range tests {
			got, err := ParseProvider(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseProvider(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				continue
			}
			if got != tt.want {
				t.Errorf("ParseProvider(%q) = %q, want %q", tt.input, got, tt.want)
			}
		}
	})
}

// TestGrant tests the Grant struct.
func TestGrant(t *testing.T) {
	t.Run("grant_creation", func(t *testing.T) {
		grant := Grant{
			ID:           "test-grant-id",
			Email:        "test@example.com",
			Provider:     ProviderGoogle,
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
			GrantStatus:  "valid",
		}

		if grant.ID != "test-grant-id" {
			t.Errorf("Grant.ID = %q, want %q", grant.ID, "test-grant-id")
		}
		if grant.Email != "test@example.com" {
			t.Errorf("Grant.Email = %q, want %q", grant.Email, "test@example.com")
		}
		if grant.Provider != ProviderGoogle {
			t.Errorf("Grant.Provider = %q, want %q", grant.Provider, ProviderGoogle)
		}
	})

	t.Run("grant_is_valid", func(t *testing.T) {
		grant := Grant{GrantStatus: "valid"}
		if !grant.IsValid() {
			t.Error("Grant with 'valid' status should be valid")
		}

		invalidGrant := Grant{GrantStatus: "error"}
		if invalidGrant.IsValid() {
			t.Error("Grant with 'error' status should not be valid")
		}
	})
}

// TestGrantStatus tests the GrantStatus struct.
func TestGrantStatus(t *testing.T) {
	t.Run("grant_status_creation", func(t *testing.T) {
		status := GrantStatus{
			ID:        "test-grant-id",
			Email:     "test@example.com",
			Provider:  ProviderGoogle,
			Status:    "valid",
			IsDefault: true,
		}

		if status.ID != "test-grant-id" {
			t.Errorf("GrantStatus.ID = %q, want %q", status.ID, "test-grant-id")
		}
		if !status.IsDefault {
			t.Error("GrantStatus.IsDefault should be true")
		}
	})
}

// TestConfig tests the Config struct.
func TestConfig(t *testing.T) {
	t.Run("config_creation", func(t *testing.T) {
		cfg := Config{
			Region:       "us",
			CallbackPort: 8080,
		}

		if cfg.Region != "us" {
			t.Errorf("Config.Region = %q, want %q", cfg.Region, "us")
		}
		if cfg.CallbackPort != 8080 {
			t.Errorf("Config.CallbackPort = %d, want %d", cfg.CallbackPort, 8080)
		}
	})
}

// TestMessage tests the Message struct.
func TestMessage(t *testing.T) {
	t.Run("message_from_contacts", func(t *testing.T) {
		msg := Message{
			ID:      "msg-id",
			Subject: "Test Subject",
			From: []EmailParticipant{
				{Name: "Test User", Email: "test@example.com"},
			},
			Body:    "Test body content",
			Snippet: "Test snippet...",
		}

		if len(msg.From) != 1 {
			t.Fatalf("Expected 1 contact in From, got %d", len(msg.From))
		}
		if msg.From[0].Email != "test@example.com" {
			t.Errorf("From[0].Email = %q, want %q", msg.From[0].Email, "test@example.com")
		}
	})
}
