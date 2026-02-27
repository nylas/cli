package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

// ============================================================================
// TestExecuteGetContact
// ============================================================================

func TestExecuteGetContact(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, contactID string) (*domain.Contact, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path returns full contact detail",
			args: map[string]any{"contact_id": "c1"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Contact, error) {
				return &domain.Contact{
					ID:           "c1",
					GivenName:    "Alice",
					Surname:      "Smith",
					MiddleName:   "M",
					Nickname:     "Al",
					Birthday:     "1990-01-01",
					CompanyName:  "Acme",
					JobTitle:     "Engineer",
					Notes:        "VIP",
					Emails:       []domain.ContactEmail{{Email: "alice@example.com", Type: "work"}},
					PhoneNumbers: []domain.ContactPhone{{Number: "+1234567890", Type: "mobile"}},
				}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "c1" {
					t.Errorf("id = %v, want c1", result["id"])
				}
				if result["given_name"] != "Alice" {
					t.Errorf("given_name = %v, want Alice", result["given_name"])
				}
				if result["surname"] != "Smith" {
					t.Errorf("surname = %v, want Smith", result["surname"])
				}
				if result["company_name"] != "Acme" {
					t.Errorf("company_name = %v, want Acme", result["company_name"])
				}
				if result["birthday"] != "1990-01-01" {
					t.Errorf("birthday = %v, want 1990-01-01", result["birthday"])
				}
			},
		},
		{
			name:      "missing contact_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates as toolError",
			args: map[string]any{"contact_id": "c2"},
			mockFn: func(_ context.Context, _, _ string) (*domain.Contact, error) {
				return nil, errors.New("contact not found")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{getContactFunc: tt.mockFn})
			resp := s.executeGetContact(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

// ============================================================================
// TestExecuteCreateContact
// ============================================================================

func TestExecuteCreateContact(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID string, req *domain.CreateContactRequest) (*domain.Contact, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path with name and email",
			args: map[string]any{
				"given_name": "Bob",
				"surname":    "Jones",
				"emails": []any{
					map[string]any{"email": "bob@example.com", "type": "work"},
				},
			},
			mockFn: func(_ context.Context, _ string, req *domain.CreateContactRequest) (*domain.Contact, error) {
				return &domain.Contact{
					ID:        "new-c1",
					GivenName: req.GivenName,
					Surname:   req.Surname,
				}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "new-c1" {
					t.Errorf("id = %v, want new-c1", result["id"])
				}
				if result["status"] != "created" {
					t.Errorf("status = %v, want created", result["status"])
				}
				if result["display_name"] != "Bob Jones" {
					t.Errorf("display_name = %v, want 'Bob Jones'", result["display_name"])
				}
			},
		},
		{
			name: "API error propagates as toolError",
			args: map[string]any{"given_name": "Error"},
			mockFn: func(_ context.Context, _ string, _ *domain.CreateContactRequest) (*domain.Contact, error) {
				return nil, errors.New("create failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{createContactFunc: tt.mockFn})
			resp := s.executeCreateContact(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

// ============================================================================
// TestExecuteUpdateContact
// ============================================================================

func TestExecuteUpdateContact(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, contactID string, req *domain.UpdateContactRequest) (*domain.Contact, error)
		wantError bool
		checkFn   func(t *testing.T, result map[string]any)
	}{
		{
			name: "happy path updates contact",
			args: map[string]any{
				"contact_id": "c3",
				"given_name": "Carol",
				"job_title":  "Manager",
			},
			mockFn: func(_ context.Context, _, _ string, req *domain.UpdateContactRequest) (*domain.Contact, error) {
				name := ""
				if req.GivenName != nil {
					name = *req.GivenName
				}
				return &domain.Contact{ID: "c3", GivenName: name}, nil
			},
			checkFn: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["id"] != "c3" {
					t.Errorf("id = %v, want c3", result["id"])
				}
				if result["status"] != "updated" {
					t.Errorf("status = %v, want updated", result["status"])
				}
			},
		},
		{
			name:      "missing contact_id returns error",
			args:      map[string]any{"given_name": "Carol"},
			wantError: true,
		},
		{
			name: "API error propagates as toolError",
			args: map[string]any{"contact_id": "c4", "surname": "Fail"},
			mockFn: func(_ context.Context, _, _ string, _ *domain.UpdateContactRequest) (*domain.Contact, error) {
				return nil, errors.New("update failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{updateContactFunc: tt.mockFn})
			resp := s.executeUpdateContact(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			var result map[string]any
			unmarshalText(t, resp, &result)
			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

// ============================================================================
// TestExecuteDeleteContact
// ============================================================================

func TestExecuteDeleteContact(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name      string
		args      map[string]any
		mockFn    func(ctx context.Context, grantID, contactID string) error
		wantError bool
		wantText  string
	}{
		{
			name:     "happy path deletes contact",
			args:     map[string]any{"contact_id": "c5"},
			mockFn:   func(_ context.Context, _, _ string) error { return nil },
			wantText: "c5",
		},
		{
			name:      "missing contact_id returns error",
			args:      map[string]any{},
			wantError: true,
		},
		{
			name: "API error propagates as toolError",
			args: map[string]any{"contact_id": "c6"},
			mockFn: func(_ context.Context, _, _ string) error {
				return errors.New("delete failed")
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := newMockServer(&mockNylasClient{deleteContactFunc: tt.mockFn})
			resp := s.executeDeleteContact(ctx, tt.args)
			if tt.wantError {
				if !resp.IsError {
					t.Errorf("expected error, got: %s", resp.Content[0].Text)
				}
				return
			}
			if resp.IsError {
				t.Fatalf("unexpected error: %s", resp.Content[0].Text)
			}
			text := resp.Content[0].Text
			if !strings.Contains(text, "Deleted") {
				t.Errorf("response text = %q, want to contain 'Deleted'", text)
			}
			if tt.wantText != "" && !strings.Contains(text, tt.wantText) {
				t.Errorf("response text = %q, want to contain %q", text, tt.wantText)
			}
		})
	}
}

// ============================================================================
// TestParseContactEmails
// ============================================================================

func TestParseContactEmails(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      map[string]any
		wantLen   int
		wantFirst *domain.ContactEmail
	}{
		{
			name: "valid array returns parsed emails",
			args: map[string]any{
				"emails": []any{
					map[string]any{"email": "a@b.com", "type": "work"},
				},
			},
			wantLen:   1,
			wantFirst: &domain.ContactEmail{Email: "a@b.com", Type: "work"},
		},
		{
			name: "multiple entries all parsed",
			args: map[string]any{
				"emails": []any{
					map[string]any{"email": "x@y.com", "type": "home"},
					map[string]any{"email": "z@w.com"},
				},
			},
			wantLen: 2,
		},
		{
			name:    "missing key returns nil",
			args:    map[string]any{},
			wantLen: 0,
		},
		{
			name: "items without email field are skipped",
			args: map[string]any{
				"emails": []any{
					map[string]any{"type": "work"},
					map[string]any{"email": "valid@example.com"},
				},
			},
			wantLen:   1,
			wantFirst: &domain.ContactEmail{Email: "valid@example.com"},
		},
		{
			name:    "non-array value returns nil",
			args:    map[string]any{"emails": "not-an-array"},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseContactEmails(tt.args)
			if len(result) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(result), tt.wantLen)
			}
			if tt.wantFirst != nil && len(result) > 0 {
				if result[0].Email != tt.wantFirst.Email {
					t.Errorf("result[0].Email = %q, want %q", result[0].Email, tt.wantFirst.Email)
				}
				if tt.wantFirst.Type != "" && result[0].Type != tt.wantFirst.Type {
					t.Errorf("result[0].Type = %q, want %q", result[0].Type, tt.wantFirst.Type)
				}
			}
		})
	}
}

// ============================================================================
// TestParseContactPhones
// ============================================================================

func TestParseContactPhones(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      map[string]any
		wantLen   int
		wantFirst *domain.ContactPhone
	}{
		{
			name: "valid array returns parsed phones",
			args: map[string]any{
				"phone_numbers": []any{
					map[string]any{"number": "+1234", "type": "mobile"},
				},
			},
			wantLen:   1,
			wantFirst: &domain.ContactPhone{Number: "+1234", Type: "mobile"},
		},
		{
			name: "multiple entries all parsed",
			args: map[string]any{
				"phone_numbers": []any{
					map[string]any{"number": "+111", "type": "work"},
					map[string]any{"number": "+222"},
				},
			},
			wantLen: 2,
		},
		{
			name:    "missing key returns nil",
			args:    map[string]any{},
			wantLen: 0,
		},
		{
			name: "items without number field are skipped",
			args: map[string]any{
				"phone_numbers": []any{
					map[string]any{"type": "mobile"},
					map[string]any{"number": "+555"},
				},
			},
			wantLen:   1,
			wantFirst: &domain.ContactPhone{Number: "+555"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := parseContactPhones(tt.args)
			if len(result) != tt.wantLen {
				t.Fatalf("len = %d, want %d", len(result), tt.wantLen)
			}
			if tt.wantFirst != nil && len(result) > 0 {
				if result[0].Number != tt.wantFirst.Number {
					t.Errorf("result[0].Number = %q, want %q", result[0].Number, tt.wantFirst.Number)
				}
				if tt.wantFirst.Type != "" && result[0].Type != tt.wantFirst.Type {
					t.Errorf("result[0].Type = %q, want %q", result[0].Type, tt.wantFirst.Type)
				}
			}
		})
	}
}
