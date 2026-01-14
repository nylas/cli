package email

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/nylas/cli/internal/domain"
)

// Service implements ports.EmailUtilityService.
// Provides email template building, validation, and deliverability checking.
type Service struct{}

// NewService creates a new email utility service.
func NewService() *Service {
	return &Service{}
}

// BuildTemplate creates an email template with variable support.
func (s *Service) BuildTemplate(ctx context.Context, req *domain.TemplateRequest) (*domain.EmailTemplate, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("template name is required")
	}

	if req.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	htmlBody := req.HTMLBody

	// Inline CSS if requested
	if req.InlineCSS && htmlBody != "" {
		// TODO: Implement CSS inlining using a library
		// For now, we skip inlining and use the original HTML
		_ = htmlBody // Prevent unused variable warning
	}

	if req.Sanitize && htmlBody != "" {
		var err error
		htmlBody, err = s.SanitizeHTML(ctx, htmlBody)
		if err != nil {
			return nil, fmt.Errorf("sanitize HTML: %w", err)
		}
	}

	template := &domain.EmailTemplate{
		ID:        generateID(),
		Name:      req.Name,
		Subject:   req.Subject,
		HTMLBody:  htmlBody,
		TextBody:  req.TextBody,
		Variables: req.Variables,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  req.Metadata,
	}

	return template, nil
}

// PreviewTemplate renders a template with test data.
func (s *Service) PreviewTemplate(ctx context.Context, template *domain.EmailTemplate, data map[string]any) (string, error) {
	html := template.HTMLBody

	// Simple variable substitution
	for key, value := range data {
		placeholder := fmt.Sprintf("{{%s}}", key)
		html = strings.ReplaceAll(html, placeholder, fmt.Sprintf("%v", value))
	}

	return html, nil
}

// CheckDeliverability analyzes an email for deliverability issues.
func (s *Service) CheckDeliverability(ctx context.Context, emlFile string) (*domain.DeliverabilityReport, error) {
	// Parse the EML file
	parsed, err := s.ParseEML(ctx, emlFile)
	if err != nil {
		return nil, fmt.Errorf("parse EML: %w", err)
	}

	report := &domain.DeliverabilityReport{
		Score:           100,
		Issues:          []domain.DeliverabilityIssue{},
		SPFStatus:       "not_checked",
		DKIMStatus:      "not_checked",
		DMARCStatus:     "not_checked",
		SpamScore:       0.0,
		MobileOptimized: true,
		Recommendations: []string{},
	}

	// Check for common issues
	if parsed.From == "" {
		report.Issues = append(report.Issues, domain.DeliverabilityIssue{
			Severity: "critical",
			Category: "headers",
			Message:  "Missing From header",
			Fix:      "Add a valid From address",
		})
		report.Score -= 20
	}

	if len(parsed.To) == 0 {
		report.Issues = append(report.Issues, domain.DeliverabilityIssue{
			Severity: "critical",
			Category: "headers",
			Message:  "Missing To header",
			Fix:      "Add at least one recipient",
		})
		report.Score -= 20
	}

	if parsed.Subject == "" {
		report.Issues = append(report.Issues, domain.DeliverabilityIssue{
			Severity: "warning",
			Category: "headers",
			Message:  "Missing Subject",
			Fix:      "Add a descriptive subject line",
		})
		report.Score -= 10
	}

	// Analyze spam score if HTML body exists
	if parsed.HTMLBody != "" {
		spamAnalysis, err := s.AnalyzeSpamScore(ctx, parsed.HTMLBody, parsed.Headers)
		if err == nil {
			report.SpamScore = spamAnalysis.Score
			if spamAnalysis.IsSpam {
				report.Score -= 30
				report.Issues = append(report.Issues, domain.DeliverabilityIssue{
					Severity: "critical",
					Category: "content",
					Message:  "High spam score detected",
					Fix:      "Review content for spam triggers",
				})
			}
		}
	}

	// Ensure score doesn't go negative
	if report.Score < 0 {
		report.Score = 0
	}

	return report, nil
}

// SanitizeHTML cleans HTML for email compatibility.
func (s *Service) SanitizeHTML(ctx context.Context, html string) (string, error) {
	// Basic sanitization - remove dangerous tags and attributes
	// In production, use a proper HTML sanitizer library

	dangerous := []string{
		"<script", "</script>",
		"<iframe", "</iframe>",
		"javascript:",
		"onerror=",
		"onclick=",
	}

	sanitized := html
	for _, pattern := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, pattern, "")
		sanitized = strings.ReplaceAll(sanitized, strings.ToUpper(pattern), "")
	}

	return sanitized, nil
}

// InlineCSS inlines CSS styles for email client compatibility.
func (s *Service) InlineCSS(ctx context.Context, html string) (string, error) {
	// TODO: Implement CSS inlining
	// This would parse <style> tags and apply them inline to elements
	// For now, just return the original HTML
	return html, nil
}

// ParseEML parses an .eml file into a structured message.
func (s *Service) ParseEML(ctx context.Context, emlFile string) (*domain.ParsedEmail, error) {
	// #nosec G304 -- emlFile comes from validated CLI argument, user controls their own file system
	data, err := os.ReadFile(emlFile)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Parse email message
	msg, err := mail.ReadMessage(strings.NewReader(string(data)))
	if err != nil {
		return nil, fmt.Errorf("parse message: %w", err)
	}

	parsed := &domain.ParsedEmail{
		Headers: make(map[string]string),
	}

	// Extract headers
	for k, v := range msg.Header {
		if len(v) > 0 {
			parsed.Headers[k] = v[0]
		}
	}

	// Extract common headers
	parsed.From = msg.Header.Get("From")
	parsed.Subject = msg.Header.Get("Subject")

	if to := msg.Header.Get("To"); to != "" {
		parsed.To = strings.Split(to, ",")
	}

	if cc := msg.Header.Get("Cc"); cc != "" {
		parsed.Cc = strings.Split(cc, ",")
	}

	// Parse date
	if dateStr := msg.Header.Get("Date"); dateStr != "" {
		parsed.Date, _ = mail.ParseDate(dateStr)
	}

	// TODO: Parse body (text/html)
	// This would need to handle MIME multipart messages

	return parsed, nil
}

// GenerateEML generates an .eml file from message data.
func (s *Service) GenerateEML(ctx context.Context, message *domain.EmailMessage) (string, error) {
	// TODO: Implement EML generation
	// This would format message as RFC 822 email
	return "", fmt.Errorf("not implemented")
}

// ValidateEmailAddress validates email address format and DNS MX records.
func (s *Service) ValidateEmailAddress(ctx context.Context, email string) (*domain.EmailValidation, error) {
	validation := &domain.EmailValidation{
		Email:       email,
		Valid:       false,
		FormatValid: false,
		MXExists:    false,
		Disposable:  false,
	}

	// Validate format
	addr, err := mail.ParseAddress(email)
	if err == nil && addr.Address == email {
		validation.FormatValid = true
	} else {
		return validation, nil
	}

	// Extract domain
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return validation, nil
	}
	domain := parts[1]

	// Check MX records
	mx, err := net.LookupMX(domain)
	if err == nil && len(mx) > 0 {
		validation.MXExists = true
		validation.Valid = true
	}

	// Check if disposable (basic check)
	disposableDomains := []string{
		"tempmail.com", "guerrillamail.com", "10minutemail.com",
		"mailinator.com", "throwaway.email",
	}
	for _, d := range disposableDomains {
		if strings.HasSuffix(domain, d) {
			validation.Disposable = true
			break
		}
	}

	return validation, nil
}

// AnalyzeSpamScore calculates spam score using local rules.
func (s *Service) AnalyzeSpamScore(ctx context.Context, html string, headers map[string]string) (*domain.SpamAnalysis, error) {
	analysis := &domain.SpamAnalysis{
		Score:       0.0,
		IsSpam:      false,
		Triggers:    []domain.SpamTrigger{},
		Passed:      []string{},
		Suggestions: []string{},
	}

	htmlLower := strings.ToLower(html)

	// Check for spam triggers
	spamWords := []string{
		"free money", "click here", "act now", "limited time",
		"congratulations", "you won", "viagra", "casino",
	}

	for _, word := range spamWords {
		if strings.Contains(htmlLower, word) {
			analysis.Triggers = append(analysis.Triggers, domain.SpamTrigger{
				Rule:        "spam_word",
				Description: fmt.Sprintf("Contains spam word: %s", word),
				Score:       1.0,
				Severity:    "medium",
			})
			analysis.Score += 1.0
		}
	}

	// Check for excessive capitalization
	if hasExcessiveCaps(html) {
		analysis.Triggers = append(analysis.Triggers, domain.SpamTrigger{
			Rule:        "excessive_caps",
			Description: "Excessive use of capital letters",
			Score:       0.5,
			Severity:    "low",
		})
		analysis.Score += 0.5
	}

	// Check for excessive exclamation marks
	if strings.Count(html, "!") > 3 {
		analysis.Triggers = append(analysis.Triggers, domain.SpamTrigger{
			Rule:        "excessive_exclamation",
			Description: "Too many exclamation marks",
			Score:       0.3,
			Severity:    "low",
		})
		analysis.Score += 0.3
	}

	// Determine if spam (threshold: 3.0)
	analysis.IsSpam = analysis.Score >= 3.0

	if !analysis.IsSpam {
		analysis.Passed = append(analysis.Passed, "No critical spam triggers found")
	}

	return analysis, nil
}

// ============================================================================
// Helper functions
// ============================================================================

// generateID generates a unique template ID.
func generateID() string {
	return fmt.Sprintf("tpl_%d", time.Now().UnixNano())
}

// hasExcessiveCaps checks if text has excessive capitalization.
func hasExcessiveCaps(text string) bool {
	if len(text) == 0 {
		return false
	}

	caps := 0
	letters := 0

	for _, r := range text {
		if r >= 'A' && r <= 'Z' {
			caps++
			letters++
		} else if r >= 'a' && r <= 'z' {
			letters++
		}
	}

	if letters == 0 {
		return false
	}

	ratio := float64(caps) / float64(letters)
	return ratio > 0.5 // More than 50% caps
}
