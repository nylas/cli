package tui

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestNewContactFormCreate(t *testing.T) {
	app := createTestApp(t)

	form := NewContactForm(app, nil, nil, nil)

	if form == nil {
		t.Fatal("NewContactForm returned nil")
		return
	}

	if form.mode != ContactFormCreate {
		t.Errorf("mode = %v, want ContactFormCreate", form.mode)
	}

	// Check defaults
	if form.emailType != "work" {
		t.Errorf("emailType = %q, want 'work'", form.emailType)
	}

	if form.phoneType != "mobile" {
		t.Errorf("phoneType = %q, want 'mobile'", form.phoneType)
	}
}

func TestNewContactFormEdit(t *testing.T) {
	app := createTestApp(t)

	contact := &domain.Contact{
		ID:          "contact-123",
		GivenName:   "John",
		Surname:     "Doe",
		CompanyName: "Acme Corp",
		JobTitle:    "Developer",
		Notes:       "Test notes",
		Emails: []domain.ContactEmail{
			{Email: "john@example.com", Type: "home"},
		},
		PhoneNumbers: []domain.ContactPhone{
			{Number: "555-1234", Type: "work"},
		},
	}

	form := NewContactForm(app, contact, nil, nil)

	if form == nil {
		t.Fatal("NewContactForm returned nil")
		return
	}

	if form.mode != ContactFormEdit {
		t.Errorf("mode = %v, want ContactFormEdit", form.mode)
	}

	if form.givenName != contact.GivenName {
		t.Errorf("givenName = %q, want %q", form.givenName, contact.GivenName)
	}

	if form.surname != contact.Surname {
		t.Errorf("surname = %q, want %q", form.surname, contact.Surname)
	}

	if form.email != contact.Emails[0].Email {
		t.Errorf("email = %q, want %q", form.email, contact.Emails[0].Email)
	}

	if form.emailType != contact.Emails[0].Type {
		t.Errorf("emailType = %q, want %q", form.emailType, contact.Emails[0].Type)
	}

	if form.phone != contact.PhoneNumbers[0].Number {
		t.Errorf("phone = %q, want %q", form.phone, contact.PhoneNumbers[0].Number)
	}

	if form.companyName != contact.CompanyName {
		t.Errorf("companyName = %q, want %q", form.companyName, contact.CompanyName)
	}
}

func TestContactFormValidation(t *testing.T) {
	app := createTestApp(t)

	tests := []struct {
		name      string
		setup     func(f *ContactForm)
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid with name only",
			setup: func(f *ContactForm) {
				f.givenName = "John"
			},
			wantError: false,
		},
		{
			name: "valid with surname only",
			setup: func(f *ContactForm) {
				f.surname = "Doe"
			},
			wantError: false,
		},
		{
			name: "valid with email only",
			setup: func(f *ContactForm) {
				f.email = "john@example.com"
			},
			wantError: false,
		},
		{
			name: "valid with full info",
			setup: func(f *ContactForm) {
				f.givenName = "John"
				f.surname = "Doe"
				f.email = "john@example.com"
				f.phone = "555-1234"
				f.companyName = "Acme"
			},
			wantError: false,
		},
		{
			name: "missing name and email",
			setup: func(f *ContactForm) {
				f.givenName = ""
				f.surname = ""
				f.email = ""
			},
			wantError: true,
			errorMsg:  "At least a name or email is required",
		},
		{
			name: "invalid email format",
			setup: func(f *ContactForm) {
				f.email = "not-an-email"
			},
			wantError: true,
			errorMsg:  "Email must be a valid email address",
		},
		{
			name: "whitespace only name",
			setup: func(f *ContactForm) {
				f.givenName = "   "
				f.surname = "   "
				f.email = ""
			},
			wantError: true,
			errorMsg:  "At least a name or email is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := NewContactForm(app, nil, nil, nil)
			// Reset default values that might affect validation
			form.email = ""
			form.givenName = ""
			form.surname = ""

			tt.setup(form)

			errors := form.validate()

			if tt.wantError {
				if len(errors) == 0 {
					t.Error("expected validation error, got none")
				} else {
					found := false
					for _, err := range errors {
						if err == tt.errorMsg {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected error %q, got %v", tt.errorMsg, errors)
					}
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("unexpected validation errors: %v", errors)
				}
			}
		})
	}
}

func TestContactFormMode(t *testing.T) {
	if ContactFormCreate != 0 {
		t.Errorf("ContactFormCreate = %d, want 0", ContactFormCreate)
	}
	if ContactFormEdit != 1 {
		t.Errorf("ContactFormEdit = %d, want 1", ContactFormEdit)
	}
}

func TestContactFormDefaultValues(t *testing.T) {
	app := createTestApp(t)

	// Test edit mode with contact that has no emails or phones
	contact := &domain.Contact{
		ID:        "contact-123",
		GivenName: "John",
	}

	form := NewContactForm(app, contact, nil, nil)

	// Email type should remain empty when there are no emails
	if form.email != "" {
		t.Errorf("email = %q, want empty", form.email)
	}

	// Phone type should remain empty when there are no phones
	if form.phone != "" {
		t.Errorf("phone = %q, want empty", form.phone)
	}
}
