package demo

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestNew(t *testing.T) {
	client := New()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestClient_SetRegion(t *testing.T) {
	client := New()
	// Should not panic
	client.SetRegion("us")
	client.SetRegion("eu")
}

func TestClient_SetCredentials(t *testing.T) {
	client := New()
	// Should not panic
	client.SetCredentials("client-id", "client-secret", "api-key")
}

func TestClient_BuildAuthURL(t *testing.T) {
	client := New()

	tests := []struct {
		name        string
		provider    domain.Provider
		redirectURI string
		want        string
	}{
		{
			name:        "google provider",
			provider:    domain.ProviderGoogle,
			redirectURI: "http://localhost:8080/callback",
			want:        "https://demo.nylas.com/auth",
		},
		{
			name:        "microsoft provider",
			provider:    domain.ProviderMicrosoft,
			redirectURI: "http://localhost:8080/callback",
			want:        "https://demo.nylas.com/auth",
		},
		{
			name:        "imap provider",
			provider:    domain.ProviderIMAP,
			redirectURI: "http://localhost/callback",
			want:        "https://demo.nylas.com/auth",
		},
		{
			name:        "empty redirect URI",
			provider:    domain.ProviderGoogle,
			redirectURI: "",
			want:        "https://demo.nylas.com/auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.BuildAuthURL(tt.provider, tt.redirectURI)
			if got != tt.want {
				t.Errorf("BuildAuthURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClient_ExchangeCode(t *testing.T) {
	client := New()
	ctx := context.Background()

	tests := []struct {
		name        string
		code        string
		redirectURI string
		wantErr     bool
	}{
		{
			name:        "valid code exchange",
			code:        "valid-auth-code",
			redirectURI: "http://localhost/callback",
			wantErr:     false,
		},
		{
			name:        "empty code",
			code:        "",
			redirectURI: "http://localhost/callback",
			wantErr:     false, // Demo client doesn't validate
		},
		{
			name:        "empty redirect URI",
			code:        "valid-auth-code",
			redirectURI: "",
			wantErr:     false, // Demo client doesn't validate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grant, err := client.ExchangeCode(ctx, tt.code, tt.redirectURI)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExchangeCode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if grant == nil {
					t.Fatal("expected non-nil grant")
				}
				if grant.ID != "demo-grant-id" {
					t.Errorf("expected ID 'demo-grant-id', got %q", grant.ID)
				}
				if grant.Email != "demo@example.com" {
					t.Errorf("expected Email 'demo@example.com', got %q", grant.Email)
				}
				if grant.Provider != domain.ProviderGoogle {
					t.Errorf("expected Provider Google, got %v", grant.Provider)
				}
				if grant.GrantStatus != "valid" {
					t.Errorf("expected GrantStatus 'valid', got %q", grant.GrantStatus)
				}
			}
		})
	}
}

func TestClient_ExchangeCode_FieldValidation(t *testing.T) {
	client := New()
	ctx := context.Background()

	grant, err := client.ExchangeCode(ctx, "test-code", "http://localhost/callback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("has valid grant ID", func(t *testing.T) {
		if grant.ID == "" {
			t.Error("grant ID should not be empty")
		}
	})

	t.Run("has valid email", func(t *testing.T) {
		if grant.Email == "" {
			t.Error("email should not be empty")
		}
		// Basic email validation
		if !contains(grant.Email, "@") {
			t.Errorf("email should contain @, got %q", grant.Email)
		}
	})

	t.Run("has valid provider", func(t *testing.T) {
		if grant.Provider == "" {
			t.Error("provider should not be empty")
		}
	})

	t.Run("has valid grant status", func(t *testing.T) {
		if grant.GrantStatus == "" {
			t.Error("grant status should not be empty")
		}
		if grant.GrantStatus != "valid" {
			t.Errorf("expected grant status 'valid', got %q", grant.GrantStatus)
		}
	})
}

func TestClient_ExchangeCode_Consistency(t *testing.T) {
	client := New()
	ctx := context.Background()

	// Call multiple times to ensure consistency
	grant1, err1 := client.ExchangeCode(ctx, "code1", "uri1")
	grant2, err2 := client.ExchangeCode(ctx, "code2", "uri2")
	grant3, err3 := client.ExchangeCode(ctx, "code3", "uri3")

	if err1 != nil || err2 != nil || err3 != nil {
		t.Fatal("unexpected errors")
	}

	// All demo grants should have same consistent data
	if grant1.ID != grant2.ID || grant2.ID != grant3.ID {
		t.Error("demo grants should have consistent IDs")
	}
	if grant1.Email != grant2.Email || grant2.Email != grant3.Email {
		t.Error("demo grants should have consistent emails")
	}
	if grant1.Provider != grant2.Provider || grant2.Provider != grant3.Provider {
		t.Error("demo grants should have consistent providers")
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	client := New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Demo client should still work even with cancelled context
	// because it doesn't make real API calls
	grant, err := client.ExchangeCode(ctx, "code", "uri")
	if err != nil {
		t.Errorf("demo client should ignore context cancellation, got error: %v", err)
	}
	if grant == nil {
		t.Error("expected non-nil grant even with cancelled context")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
