package utilities

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// MockUtilityServices implements ports.UtilityServices for testing.
type MockUtilityServices struct {
	// TimeZoneService
	ConvertTimeFunc       func(ctx context.Context, fromZone, toZone string, t time.Time) (time.Time, error)
	FindMeetingTimeFunc   func(ctx context.Context, req *domain.MeetingFinderRequest) (*domain.MeetingTimeSlots, error)
	GetDSTTransitionsFunc func(ctx context.Context, zone string, year int) ([]domain.DSTTransition, error)
	ListTimeZonesFunc     func(ctx context.Context) ([]string, error)
	GetTimeZoneInfoFunc   func(ctx context.Context, zone string, at time.Time) (*domain.TimeZoneInfo, error)

	// WebhookService
	StartServerFunc         func(ctx context.Context, config *domain.WebhookServerConfig) error
	StopServerFunc          func(ctx context.Context) error
	GetReceivedWebhooksFunc func(ctx context.Context) ([]domain.WebhookPayload, error)
	ValidateSignatureFunc   func(payload []byte, signature string, secret string) bool
	ReplayWebhookFunc       func(ctx context.Context, webhookID string, targetURL string) error
	SaveWebhookFunc         func(ctx context.Context, payload *domain.WebhookPayload, filepath string) error
	LoadWebhookFunc         func(ctx context.Context, filepath string) (*domain.WebhookPayload, error)

	// EmailUtilityService
	BuildTemplateFunc        func(ctx context.Context, req *domain.TemplateRequest) (*domain.EmailTemplate, error)
	PreviewTemplateFunc      func(ctx context.Context, template *domain.EmailTemplate, data map[string]any) (string, error)
	CheckDeliverabilityFunc  func(ctx context.Context, emlFile string) (*domain.DeliverabilityReport, error)
	SanitizeHTMLFunc         func(ctx context.Context, html string) (string, error)
	InlineCSSFunc            func(ctx context.Context, html string) (string, error)
	ParseEMLFunc             func(ctx context.Context, emlFile string) (*domain.ParsedEmail, error)
	GenerateEMLFunc          func(ctx context.Context, message *domain.EmailMessage) (string, error)
	ValidateEmailAddressFunc func(ctx context.Context, email string) (*domain.EmailValidation, error)
	AnalyzeSpamScoreFunc     func(ctx context.Context, html string, headers map[string]string) (*domain.SpamAnalysis, error)

	// ContactUtilityService
	DeduplicateContactsFunc func(ctx context.Context, req *domain.DeduplicationRequest) (*domain.DeduplicationResult, error)
	ParseVCardFunc          func(ctx context.Context, vcfData string) ([]domain.Contact, error)
	ExportVCardFunc         func(ctx context.Context, contacts []domain.Contact) (string, error)
	MapVCardFieldsFunc      func(ctx context.Context, from, to string, contact *domain.Contact) (*domain.Contact, error)
	MergeContactsFunc       func(ctx context.Context, contacts []domain.Contact, strategy string) (*domain.Contact, error)
	ImportCSVFunc           func(ctx context.Context, csvFile string, mapping map[string]string) ([]domain.Contact, error)
	ExportCSVFunc           func(ctx context.Context, contacts []domain.Contact) (string, error)
	EnrichContactFunc       func(ctx context.Context, contact *domain.Contact) (*domain.Contact, error)
}

// NewMockUtilityServices creates a new mock utility services with sensible defaults.
func NewMockUtilityServices() *MockUtilityServices {
	return &MockUtilityServices{
		// TimeZoneService defaults
		ConvertTimeFunc: func(ctx context.Context, fromZone, toZone string, t time.Time) (time.Time, error) {
			return t, nil
		},
		FindMeetingTimeFunc: func(ctx context.Context, req *domain.MeetingFinderRequest) (*domain.MeetingTimeSlots, error) {
			return &domain.MeetingTimeSlots{Slots: []domain.MeetingSlot{}, TimeZones: req.TimeZones}, nil
		},
		GetDSTTransitionsFunc: func(ctx context.Context, zone string, year int) ([]domain.DSTTransition, error) {
			return []domain.DSTTransition{}, nil
		},
		ListTimeZonesFunc: func(ctx context.Context) ([]string, error) {
			return []string{"UTC", "America/New_York", "Europe/London"}, nil
		},
		GetTimeZoneInfoFunc: func(ctx context.Context, zone string, at time.Time) (*domain.TimeZoneInfo, error) {
			return &domain.TimeZoneInfo{Name: zone, Abbreviation: "UTC", Offset: 0}, nil
		},

		// WebhookService defaults
		StartServerFunc: func(ctx context.Context, config *domain.WebhookServerConfig) error {
			return nil
		},
		StopServerFunc: func(ctx context.Context) error {
			return nil
		},
		GetReceivedWebhooksFunc: func(ctx context.Context) ([]domain.WebhookPayload, error) {
			return []domain.WebhookPayload{}, nil
		},
		ValidateSignatureFunc: func(payload []byte, signature string, secret string) bool {
			return true
		},
		ReplayWebhookFunc: func(ctx context.Context, webhookID string, targetURL string) error {
			return nil
		},
		SaveWebhookFunc: func(ctx context.Context, payload *domain.WebhookPayload, filepath string) error {
			return nil
		},
		LoadWebhookFunc: func(ctx context.Context, filepath string) (*domain.WebhookPayload, error) {
			return &domain.WebhookPayload{ID: "mock-webhook"}, nil
		},

		// EmailUtilityService defaults
		BuildTemplateFunc: func(ctx context.Context, req *domain.TemplateRequest) (*domain.EmailTemplate, error) {
			return &domain.EmailTemplate{ID: "tpl_1", Name: req.Name, Subject: req.Subject}, nil
		},
		PreviewTemplateFunc: func(ctx context.Context, template *domain.EmailTemplate, data map[string]any) (string, error) {
			return template.HTMLBody, nil
		},
		CheckDeliverabilityFunc: func(ctx context.Context, emlFile string) (*domain.DeliverabilityReport, error) {
			return &domain.DeliverabilityReport{Score: 100, Issues: []domain.DeliverabilityIssue{}}, nil
		},
		SanitizeHTMLFunc: func(ctx context.Context, html string) (string, error) {
			return html, nil
		},
		InlineCSSFunc: func(ctx context.Context, html string) (string, error) {
			return html, nil
		},
		ParseEMLFunc: func(ctx context.Context, emlFile string) (*domain.ParsedEmail, error) {
			return &domain.ParsedEmail{Subject: "Test Email"}, nil
		},
		GenerateEMLFunc: func(ctx context.Context, message *domain.EmailMessage) (string, error) {
			return "mock eml data", nil
		},
		ValidateEmailAddressFunc: func(ctx context.Context, email string) (*domain.EmailValidation, error) {
			return &domain.EmailValidation{Email: email, Valid: true, FormatValid: true}, nil
		},
		AnalyzeSpamScoreFunc: func(ctx context.Context, html string, headers map[string]string) (*domain.SpamAnalysis, error) {
			return &domain.SpamAnalysis{Score: 0, IsSpam: false}, nil
		},

		// ContactUtilityService defaults
		DeduplicateContactsFunc: func(ctx context.Context, req *domain.DeduplicationRequest) (*domain.DeduplicationResult, error) {
			return &domain.DeduplicationResult{OriginalCount: len(req.Contacts), DeduplicatedCount: len(req.Contacts)}, nil
		},
		ParseVCardFunc: func(ctx context.Context, vcfData string) ([]domain.Contact, error) {
			return []domain.Contact{}, nil
		},
		ExportVCardFunc: func(ctx context.Context, contacts []domain.Contact) (string, error) {
			return "mock vcard", nil
		},
		MapVCardFieldsFunc: func(ctx context.Context, from, to string, contact *domain.Contact) (*domain.Contact, error) {
			return contact, nil
		},
		MergeContactsFunc: func(ctx context.Context, contacts []domain.Contact, strategy string) (*domain.Contact, error) {
			if len(contacts) > 0 {
				return &contacts[0], nil
			}
			return &domain.Contact{}, nil
		},
		ImportCSVFunc: func(ctx context.Context, csvFile string, mapping map[string]string) ([]domain.Contact, error) {
			return []domain.Contact{}, nil
		},
		ExportCSVFunc: func(ctx context.Context, contacts []domain.Contact) (string, error) {
			return "mock csv", nil
		},
		EnrichContactFunc: func(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
			return contact, nil
		},
	}
}

// ============================================================================
// TimeZoneService implementation
// ============================================================================

func (m *MockUtilityServices) ConvertTime(ctx context.Context, fromZone, toZone string, t time.Time) (time.Time, error) {
	if m.ConvertTimeFunc != nil {
		return m.ConvertTimeFunc(ctx, fromZone, toZone, t)
	}
	return t, nil
}

func (m *MockUtilityServices) FindMeetingTime(ctx context.Context, req *domain.MeetingFinderRequest) (*domain.MeetingTimeSlots, error) {
	if m.FindMeetingTimeFunc != nil {
		return m.FindMeetingTimeFunc(ctx, req)
	}
	return &domain.MeetingTimeSlots{}, nil
}

func (m *MockUtilityServices) GetDSTTransitions(ctx context.Context, zone string, year int) ([]domain.DSTTransition, error) {
	if m.GetDSTTransitionsFunc != nil {
		return m.GetDSTTransitionsFunc(ctx, zone, year)
	}
	return []domain.DSTTransition{}, nil
}

func (m *MockUtilityServices) ListTimeZones(ctx context.Context) ([]string, error) {
	if m.ListTimeZonesFunc != nil {
		return m.ListTimeZonesFunc(ctx)
	}
	return []string{}, nil
}

func (m *MockUtilityServices) GetTimeZoneInfo(ctx context.Context, zone string, at time.Time) (*domain.TimeZoneInfo, error) {
	if m.GetTimeZoneInfoFunc != nil {
		return m.GetTimeZoneInfoFunc(ctx, zone, at)
	}
	return &domain.TimeZoneInfo{}, nil
}

// ============================================================================
// WebhookService implementation
// ============================================================================

func (m *MockUtilityServices) StartServer(ctx context.Context, config *domain.WebhookServerConfig) error {
	if m.StartServerFunc != nil {
		return m.StartServerFunc(ctx, config)
	}
	return nil
}

func (m *MockUtilityServices) StopServer(ctx context.Context) error {
	if m.StopServerFunc != nil {
		return m.StopServerFunc(ctx)
	}
	return nil
}

func (m *MockUtilityServices) GetReceivedWebhooks(ctx context.Context) ([]domain.WebhookPayload, error) {
	if m.GetReceivedWebhooksFunc != nil {
		return m.GetReceivedWebhooksFunc(ctx)
	}
	return []domain.WebhookPayload{}, nil
}

func (m *MockUtilityServices) ValidateSignature(payload []byte, signature string, secret string) bool {
	if m.ValidateSignatureFunc != nil {
		return m.ValidateSignatureFunc(payload, signature, secret)
	}
	return true
}

func (m *MockUtilityServices) ReplayWebhook(ctx context.Context, webhookID string, targetURL string) error {
	if m.ReplayWebhookFunc != nil {
		return m.ReplayWebhookFunc(ctx, webhookID, targetURL)
	}
	return nil
}

func (m *MockUtilityServices) SaveWebhook(ctx context.Context, payload *domain.WebhookPayload, filepath string) error {
	if m.SaveWebhookFunc != nil {
		return m.SaveWebhookFunc(ctx, payload, filepath)
	}
	return nil
}

func (m *MockUtilityServices) LoadWebhook(ctx context.Context, filepath string) (*domain.WebhookPayload, error) {
	if m.LoadWebhookFunc != nil {
		return m.LoadWebhookFunc(ctx, filepath)
	}
	return &domain.WebhookPayload{}, nil
}

// ============================================================================
// EmailUtilityService implementation
// ============================================================================

func (m *MockUtilityServices) BuildTemplate(ctx context.Context, req *domain.TemplateRequest) (*domain.EmailTemplate, error) {
	if m.BuildTemplateFunc != nil {
		return m.BuildTemplateFunc(ctx, req)
	}
	return &domain.EmailTemplate{}, nil
}

func (m *MockUtilityServices) PreviewTemplate(ctx context.Context, template *domain.EmailTemplate, data map[string]any) (string, error) {
	if m.PreviewTemplateFunc != nil {
		return m.PreviewTemplateFunc(ctx, template, data)
	}
	return "", nil
}

func (m *MockUtilityServices) CheckDeliverability(ctx context.Context, emlFile string) (*domain.DeliverabilityReport, error) {
	if m.CheckDeliverabilityFunc != nil {
		return m.CheckDeliverabilityFunc(ctx, emlFile)
	}
	return &domain.DeliverabilityReport{}, nil
}

func (m *MockUtilityServices) SanitizeHTML(ctx context.Context, html string) (string, error) {
	if m.SanitizeHTMLFunc != nil {
		return m.SanitizeHTMLFunc(ctx, html)
	}
	return html, nil
}

func (m *MockUtilityServices) InlineCSS(ctx context.Context, html string) (string, error) {
	if m.InlineCSSFunc != nil {
		return m.InlineCSSFunc(ctx, html)
	}
	return html, nil
}

func (m *MockUtilityServices) ParseEML(ctx context.Context, emlFile string) (*domain.ParsedEmail, error) {
	if m.ParseEMLFunc != nil {
		return m.ParseEMLFunc(ctx, emlFile)
	}
	return &domain.ParsedEmail{}, nil
}

func (m *MockUtilityServices) GenerateEML(ctx context.Context, message *domain.EmailMessage) (string, error) {
	if m.GenerateEMLFunc != nil {
		return m.GenerateEMLFunc(ctx, message)
	}
	return "", nil
}

func (m *MockUtilityServices) ValidateEmailAddress(ctx context.Context, email string) (*domain.EmailValidation, error) {
	if m.ValidateEmailAddressFunc != nil {
		return m.ValidateEmailAddressFunc(ctx, email)
	}
	return &domain.EmailValidation{}, nil
}

func (m *MockUtilityServices) AnalyzeSpamScore(ctx context.Context, html string, headers map[string]string) (*domain.SpamAnalysis, error) {
	if m.AnalyzeSpamScoreFunc != nil {
		return m.AnalyzeSpamScoreFunc(ctx, html, headers)
	}
	return &domain.SpamAnalysis{}, nil
}

// ============================================================================
// ContactUtilityService implementation
// ============================================================================

func (m *MockUtilityServices) DeduplicateContacts(ctx context.Context, req *domain.DeduplicationRequest) (*domain.DeduplicationResult, error) {
	if m.DeduplicateContactsFunc != nil {
		return m.DeduplicateContactsFunc(ctx, req)
	}
	return &domain.DeduplicationResult{}, nil
}

func (m *MockUtilityServices) ParseVCard(ctx context.Context, vcfData string) ([]domain.Contact, error) {
	if m.ParseVCardFunc != nil {
		return m.ParseVCardFunc(ctx, vcfData)
	}
	return []domain.Contact{}, nil
}

func (m *MockUtilityServices) ExportVCard(ctx context.Context, contacts []domain.Contact) (string, error) {
	if m.ExportVCardFunc != nil {
		return m.ExportVCardFunc(ctx, contacts)
	}
	return "", nil
}

func (m *MockUtilityServices) MapVCardFields(ctx context.Context, from, to string, contact *domain.Contact) (*domain.Contact, error) {
	if m.MapVCardFieldsFunc != nil {
		return m.MapVCardFieldsFunc(ctx, from, to, contact)
	}
	return contact, nil
}

func (m *MockUtilityServices) MergeContacts(ctx context.Context, contacts []domain.Contact, strategy string) (*domain.Contact, error) {
	if m.MergeContactsFunc != nil {
		return m.MergeContactsFunc(ctx, contacts, strategy)
	}
	return &domain.Contact{}, nil
}

func (m *MockUtilityServices) ImportCSV(ctx context.Context, csvFile string, mapping map[string]string) ([]domain.Contact, error) {
	if m.ImportCSVFunc != nil {
		return m.ImportCSVFunc(ctx, csvFile, mapping)
	}
	return []domain.Contact{}, nil
}

func (m *MockUtilityServices) ExportCSV(ctx context.Context, contacts []domain.Contact) (string, error) {
	if m.ExportCSVFunc != nil {
		return m.ExportCSVFunc(ctx, contacts)
	}
	return "", nil
}

func (m *MockUtilityServices) EnrichContact(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	if m.EnrichContactFunc != nil {
		return m.EnrichContactFunc(ctx, contact)
	}
	return contact, nil
}
