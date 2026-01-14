package contacts

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

// Service implements ports.ContactUtilityService.
// Provides contact deduplication, vCard parsing, and import/export utilities.
type Service struct{}

// NewService creates a new contact utility service.
func NewService() *Service {
	return &Service{}
}

// DeduplicateContacts finds and merges duplicate contacts.
func (s *Service) DeduplicateContacts(ctx context.Context, req *domain.DeduplicationRequest) (*domain.DeduplicationResult, error) {
	if len(req.Contacts) == 0 {
		return &domain.DeduplicationResult{
			OriginalCount:     0,
			DeduplicatedCount: 0,
			DuplicateGroups:   []domain.DuplicateGroup{},
		}, nil
	}

	// Find duplicate groups
	groups := s.findDuplicates(req.Contacts, req.FuzzyThreshold, req.MatchFields)

	result := &domain.DeduplicationResult{
		OriginalCount:     len(req.Contacts),
		DeduplicatedCount: len(req.Contacts) - s.countDuplicates(groups),
		DuplicateGroups:   groups,
	}

	// Auto-merge if requested
	if req.AutoMerge {
		merged := make([]domain.Contact, 0)
		processed := make(map[string]bool)

		// Merge each duplicate group
		for _, group := range groups {
			if len(group.Contacts) > 0 {
				mergedContact, err := s.MergeContacts(ctx, group.Contacts, req.MergeStrategy)
				if err != nil {
					return nil, fmt.Errorf("merge contacts: %w", err)
				}
				merged = append(merged, *mergedContact)

				// Mark contacts as processed
				for _, c := range group.Contacts {
					processed[c.ID] = true
				}
			}
		}

		// Add non-duplicate contacts
		for _, contact := range req.Contacts {
			if !processed[contact.ID] {
				merged = append(merged, contact)
			}
		}

		result.MergedContacts = merged
	}

	return result, nil
}

// ParseVCard parses vCard (.vcf) data into contacts.
func (s *Service) ParseVCard(ctx context.Context, vcfData string) ([]domain.Contact, error) {
	// TODO: Implement proper vCard parsing
	// This would parse RFC 6350 vCard format
	// For now, return empty slice
	return []domain.Contact{}, fmt.Errorf("vCard parsing not implemented yet")
}

// ExportVCard exports contacts to vCard format.
func (s *Service) ExportVCard(ctx context.Context, contacts []domain.Contact) (string, error) {
	// TODO: Implement vCard generation
	// This would generate RFC 6350 vCard format
	return "", fmt.Errorf("vCard export not implemented yet")
}

// MapVCardFields maps vCard fields between different providers.
func (s *Service) MapVCardFields(ctx context.Context, from, to string, contact *domain.Contact) (*domain.Contact, error) {
	// TODO: Implement field mapping for different providers
	// This would handle Outlook -> Google, Google -> Nylas, etc.
	return contact, fmt.Errorf("vCard field mapping not implemented yet")
}

// MergeContacts merges multiple contact records into one.
func (s *Service) MergeContacts(ctx context.Context, contacts []domain.Contact, strategy string) (*domain.Contact, error) {
	if len(contacts) == 0 {
		return nil, fmt.Errorf("no contacts to merge")
	}

	if len(contacts) == 1 {
		return &contacts[0], nil
	}

	// Select base contact based on strategy
	var base domain.Contact
	switch strategy {
	case "newest":
		base = s.selectNewest(contacts)
	case "oldest":
		base = s.selectOldest(contacts)
	case "most_complete":
		base = s.selectMostComplete(contacts)
	default:
		base = contacts[0]
	}

	// Merge fields from other contacts
	merged := base

	for _, contact := range contacts {
		if contact.ID == base.ID {
			continue
		}

		// Merge emails
		merged.Emails = mergeEmails(merged.Emails, contact.Emails)

		// Merge phone numbers
		merged.PhoneNumbers = mergePhoneNumbers(merged.PhoneNumbers, contact.PhoneNumbers)

		// Merge notes (concatenate)
		if contact.Notes != "" && !strings.Contains(merged.Notes, contact.Notes) {
			if merged.Notes != "" {
				merged.Notes += "\n\n"
			}
			merged.Notes += contact.Notes
		}

		// Fill in missing fields
		if merged.GivenName == "" && contact.GivenName != "" {
			merged.GivenName = contact.GivenName
		}
		if merged.Surname == "" && contact.Surname != "" {
			merged.Surname = contact.Surname
		}
		if merged.CompanyName == "" && contact.CompanyName != "" {
			merged.CompanyName = contact.CompanyName
		}
		if merged.JobTitle == "" && contact.JobTitle != "" {
			merged.JobTitle = contact.JobTitle
		}
	}

	return &merged, nil
}

// ImportCSV imports contacts from CSV file.
func (s *Service) ImportCSV(ctx context.Context, csvFile string, mapping map[string]string) ([]domain.Contact, error) {
	// #nosec G304 -- csvFile comes from validated CLI argument, user controls their own file system
	file, err := os.Open(csvFile)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}

	if len(records) == 0 {
		return []domain.Contact{}, nil
	}

	// First row is header
	header := records[0]
	contacts := make([]domain.Contact, 0, len(records)-1)

	// Parse each record
	for i := 1; i < len(records); i++ {
		record := records[i]
		contact := s.parseCSVRecord(header, record, mapping)
		contacts = append(contacts, contact)
	}

	return contacts, nil
}

// ExportCSV exports contacts to CSV file.
func (s *Service) ExportCSV(ctx context.Context, contacts []domain.Contact) (string, error) {
	// TODO: Implement CSV export
	// This would generate CSV with contact fields
	return "", fmt.Errorf("CSV export not implemented yet")
}

// EnrichContact enriches contact with additional data.
func (s *Service) EnrichContact(ctx context.Context, contact *domain.Contact) (*domain.Contact, error) {
	// TODO: Implement contact enrichment
	// This could:
	// - Fetch Gravatar photo by email
	// - Look up company info
	// - Validate phone numbers
	// - Geocode addresses
	return contact, nil
}

// ============================================================================
// Helper functions
// ============================================================================

// findDuplicates finds groups of duplicate contacts.
func (s *Service) findDuplicates(contacts []domain.Contact, threshold float64, matchFields []string) []domain.DuplicateGroup {
	groups := make([]domain.DuplicateGroup, 0)
	processed := make(map[string]bool)

	for i := 0; i < len(contacts); i++ {
		if processed[contacts[i].ID] {
			continue
		}

		group := domain.DuplicateGroup{
			Contacts:      []domain.Contact{contacts[i]},
			MatchedFields: []string{},
		}

		// Find similar contacts
		for j := i + 1; j < len(contacts); j++ {
			if processed[contacts[j].ID] {
				continue
			}

			score, fields := s.calculateSimilarity(contacts[i], contacts[j], matchFields)
			if score >= threshold {
				group.Contacts = append(group.Contacts, contacts[j])
				group.MatchedFields = fields
				group.MatchScore = score
				processed[contacts[j].ID] = true
			}
		}

		// Only add groups with duplicates
		if len(group.Contacts) > 1 {
			groups = append(groups, group)
			processed[contacts[i].ID] = true
		}
	}

	return groups
}

// calculateSimilarity calculates similarity between two contacts.
func (s *Service) calculateSimilarity(c1, c2 domain.Contact, matchFields []string) (float64, []string) {
	var score float64
	matched := []string{}

	for _, field := range matchFields {
		switch field {
		case "email":
			if s.hasCommonEmail(c1, c2) {
				score += 0.5
				matched = append(matched, "email")
			}
		case "phone":
			if s.hasCommonPhone(c1, c2) {
				score += 0.3
				matched = append(matched, "phone")
			}
		case "name":
			if s.hasSimilarName(c1, c2) {
				score += 0.2
				matched = append(matched, "name")
			}
		}
	}

	return score, matched
}

// hasCommonEmail checks if two contacts share an email address.
func (s *Service) hasCommonEmail(c1, c2 domain.Contact) bool {
	for _, e1 := range c1.Emails {
		for _, e2 := range c2.Emails {
			if strings.EqualFold(e1.Email, e2.Email) {
				return true
			}
		}
	}
	return false
}

// hasCommonPhone checks if two contacts share a phone number.
func (s *Service) hasCommonPhone(c1, c2 domain.Contact) bool {
	for _, p1 := range c1.PhoneNumbers {
		for _, p2 := range c2.PhoneNumbers {
			if normalizePhone(p1.Number) == normalizePhone(p2.Number) {
				return true
			}
		}
	}
	return false
}

// hasSimilarName checks if two contacts have similar names.
func (s *Service) hasSimilarName(c1, c2 domain.Contact) bool {
	name1 := strings.ToLower(c1.GivenName + " " + c1.Surname)
	name2 := strings.ToLower(c2.GivenName + " " + c2.Surname)
	return name1 == name2
}

// normalizePhone normalizes a phone number for comparison.
func normalizePhone(phone string) string {
	// Remove all non-digit characters
	normalized := ""
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			normalized += string(r)
		}
	}
	return normalized
}

// countDuplicates counts total duplicate contacts across all groups.
func (s *Service) countDuplicates(groups []domain.DuplicateGroup) int {
	count := 0
	for _, group := range groups {
		// Each group has N contacts, N-1 are duplicates
		if len(group.Contacts) > 1 {
			count += len(group.Contacts) - 1
		}
	}
	return count
}

// selectNewest selects the newest contact from a list.
func (s *Service) selectNewest(contacts []domain.Contact) domain.Contact {
	if len(contacts) == 0 {
		return domain.Contact{}
	}
	// TODO: Compare by UpdatedAt timestamp when available
	return contacts[len(contacts)-1]
}

// selectOldest selects the oldest contact from a list.
func (s *Service) selectOldest(contacts []domain.Contact) domain.Contact {
	if len(contacts) == 0 {
		return domain.Contact{}
	}
	return contacts[0]
}

// selectMostComplete selects the contact with most filled fields.
func (s *Service) selectMostComplete(contacts []domain.Contact) domain.Contact {
	if len(contacts) == 0 {
		return domain.Contact{}
	}

	maxScore := 0
	bestContact := contacts[0]

	for _, contact := range contacts {
		score := s.calculateCompleteness(contact)
		if score > maxScore {
			maxScore = score
			bestContact = contact
		}
	}

	return bestContact
}

// calculateCompleteness calculates how complete a contact is.
func (s *Service) calculateCompleteness(contact domain.Contact) int {
	score := 0

	if contact.GivenName != "" {
		score++
	}
	if contact.Surname != "" {
		score++
	}
	if len(contact.Emails) > 0 {
		score++
	}
	if len(contact.PhoneNumbers) > 0 {
		score++
	}
	if contact.CompanyName != "" {
		score++
	}
	if contact.JobTitle != "" {
		score++
	}
	if contact.Notes != "" {
		score++
	}

	return score
}

// parseCSVRecord parses a CSV record into a contact.
func (s *Service) parseCSVRecord(header, record []string, mapping map[string]string) domain.Contact {
	contact := domain.Contact{
		Emails:       []domain.ContactEmail{},
		PhoneNumbers: []domain.ContactPhone{},
	}

	for i, value := range record {
		if i >= len(header) {
			break
		}

		fieldName := header[i]
		if mappedName, ok := mapping[fieldName]; ok {
			fieldName = mappedName
		}

		switch strings.ToLower(fieldName) {
		case "given_name", "firstname", "first_name":
			contact.GivenName = value
		case "surname", "lastname", "last_name":
			contact.Surname = value
		case "email":
			if value != "" {
				contact.Emails = append(contact.Emails, domain.ContactEmail{
					Type:  "work",
					Email: value,
				})
			}
		case "phone", "phone_number":
			if value != "" {
				contact.PhoneNumbers = append(contact.PhoneNumbers, domain.ContactPhone{
					Type:   "work",
					Number: value,
				})
			}
		case "company", "company_name":
			contact.CompanyName = value
		case "job_title", "title":
			contact.JobTitle = value
		case "notes":
			contact.Notes = value
		}
	}

	return contact
}

// mergeEmails merges two email lists, removing duplicates.
func mergeEmails(emails1, emails2 []domain.ContactEmail) []domain.ContactEmail {
	merged := append([]domain.ContactEmail{}, emails1...)
	seen := make(map[string]bool)

	for _, e := range emails1 {
		seen[strings.ToLower(e.Email)] = true
	}

	for _, e := range emails2 {
		if !seen[strings.ToLower(e.Email)] {
			merged = append(merged, e)
			seen[strings.ToLower(e.Email)] = true
		}
	}

	return merged
}

// mergePhoneNumbers merges two phone lists, removing duplicates.
func mergePhoneNumbers(phones1, phones2 []domain.ContactPhone) []domain.ContactPhone {
	merged := append([]domain.ContactPhone{}, phones1...)
	seen := make(map[string]bool)

	for _, p := range phones1 {
		seen[normalizePhone(p.Number)] = true
	}

	for _, p := range phones2 {
		normalized := normalizePhone(p.Number)
		if !seen[normalized] {
			merged = append(merged, p)
			seen[normalized] = true
		}
	}

	return merged
}
