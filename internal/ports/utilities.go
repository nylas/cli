package ports

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// UtilityServices defines interfaces for non-Nylas utility features.
// These services provide offline-capable tools that don't require Nylas API access.
type UtilityServices interface {
	TimeZoneService
	WebhookService
	EmailUtilityService
	ContactUtilityService
}

// TimeZoneService provides time zone conversion and meeting finder utilities.
// Addresses the pain point where 83% of professionals struggle with time zone scheduling.
type TimeZoneService interface {
	// ConvertTime converts a time from one zone to another
	ConvertTime(ctx context.Context, fromZone, toZone string, t time.Time) (time.Time, error)

	// FindMeetingTime finds overlapping working hours across multiple time zones
	FindMeetingTime(ctx context.Context, req *domain.MeetingFinderRequest) (*domain.MeetingTimeSlots, error)

	// GetDSTTransitions returns DST transition dates for a zone in a given year
	GetDSTTransitions(ctx context.Context, zone string, year int) ([]domain.DSTTransition, error)

	// ListTimeZones returns all available IANA time zones
	ListTimeZones(ctx context.Context) ([]string, error)

	// GetTimeZoneInfo returns detailed information about a time zone
	GetTimeZoneInfo(ctx context.Context, zone string, at time.Time) (*domain.TimeZoneInfo, error)
}

// WebhookService provides local webhook server capabilities.
// Addresses developer frustration with ngrok's URL changes on every restart.
type WebhookService interface {
	// StartServer starts a local webhook server
	StartServer(ctx context.Context, config *domain.WebhookServerConfig) error

	// StopServer stops the running webhook server
	StopServer(ctx context.Context) error

	// GetReceivedWebhooks returns all captured webhook payloads
	GetReceivedWebhooks(ctx context.Context) ([]domain.WebhookPayload, error)

	// ValidateSignature validates a webhook signature
	ValidateSignature(payload []byte, signature string, secret string) bool

	// ReplayWebhook replays a captured webhook to a target URL
	ReplayWebhook(ctx context.Context, webhookID string, targetURL string) error

	// SaveWebhook saves a webhook payload to file for later replay
	SaveWebhook(ctx context.Context, payload *domain.WebhookPayload, filepath string) error

	// LoadWebhook loads a webhook payload from file
	LoadWebhook(ctx context.Context, filepath string) (*domain.WebhookPayload, error)
}

// EmailUtilityService provides email utilities (templates, validation, etc).
// Addresses pain points: 50%+ emails opened on mobile, deliverability issues, testing across clients.
type EmailUtilityService interface {
	// BuildTemplate creates an email template with variable support
	BuildTemplate(ctx context.Context, req *domain.TemplateRequest) (*domain.EmailTemplate, error)

	// PreviewTemplate renders a template with test data
	PreviewTemplate(ctx context.Context, template *domain.EmailTemplate, data map[string]any) (string, error)

	// CheckDeliverability analyzes an email for deliverability issues
	CheckDeliverability(ctx context.Context, emlFile string) (*domain.DeliverabilityReport, error)

	// SanitizeHTML cleans HTML for email compatibility
	SanitizeHTML(ctx context.Context, html string) (string, error)

	// InlineCSS inlines CSS styles for email client compatibility
	InlineCSS(ctx context.Context, html string) (string, error)

	// ParseEML parses an .eml file into a structured message
	ParseEML(ctx context.Context, emlFile string) (*domain.ParsedEmail, error)

	// GenerateEML generates an .eml file from message data
	GenerateEML(ctx context.Context, message *domain.EmailMessage) (string, error)

	// ValidateEmailAddress validates email address format and DNS MX records
	ValidateEmailAddress(ctx context.Context, email string) (*domain.EmailValidation, error)

	// AnalyzeSpamScore calculates spam score using local rules
	AnalyzeSpamScore(ctx context.Context, html string, headers map[string]string) (*domain.SpamAnalysis, error)
}

// ContactUtilityService provides contact management utilities.
// Addresses pain points: data duplication, vCard field transfer issues, cross-platform compatibility.
type ContactUtilityService interface {
	// DeduplicateContacts finds and merges duplicate contacts
	DeduplicateContacts(ctx context.Context, req *domain.DeduplicationRequest) (*domain.DeduplicationResult, error)

	// ParseVCard parses vCard (.vcf) data into contacts
	ParseVCard(ctx context.Context, vcfData string) ([]domain.Contact, error)

	// ExportVCard exports contacts to vCard format
	ExportVCard(ctx context.Context, contacts []domain.Contact) (string, error)

	// MapVCardFields maps vCard fields between different providers (Outlook, Google, etc)
	MapVCardFields(ctx context.Context, from, to string, contact *domain.Contact) (*domain.Contact, error)

	// MergeContacts merges multiple contact records into one
	MergeContacts(ctx context.Context, contacts []domain.Contact, strategy string) (*domain.Contact, error)

	// ImportCSV imports contacts from CSV file
	ImportCSV(ctx context.Context, csvFile string, mapping map[string]string) ([]domain.Contact, error)

	// ExportCSV exports contacts to CSV file
	ExportCSV(ctx context.Context, contacts []domain.Contact) (string, error)

	// EnrichContact enriches contact with additional data (e.g., Gravatar)
	EnrichContact(ctx context.Context, contact *domain.Contact) (*domain.Contact, error)
}
