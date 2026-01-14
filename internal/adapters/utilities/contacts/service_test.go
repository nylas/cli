package contacts

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestDeduplicateContacts_EmptyList(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	req := &domain.DeduplicationRequest{
		Contacts: []domain.Contact{},
	}

	result, err := service.DeduplicateContacts(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OriginalCount != 0 {
		t.Errorf("expected OriginalCount=0, got %d", result.OriginalCount)
	}
	if result.DeduplicatedCount != 0 {
		t.Errorf("expected DeduplicatedCount=0, got %d", result.DeduplicatedCount)
	}
	if len(result.DuplicateGroups) != 0 {
		t.Errorf("expected no duplicate groups, got %d", len(result.DuplicateGroups))
	}
}

func TestDeduplicateContacts_NoDuplicates(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	contacts := []domain.Contact{
		{ID: "1", GivenName: "Alice", Surname: "Smith", Emails: []domain.ContactEmail{{Email: "alice@example.com"}}},
		{ID: "2", GivenName: "Bob", Surname: "Jones", Emails: []domain.ContactEmail{{Email: "bob@example.com"}}},
	}

	req := &domain.DeduplicationRequest{
		Contacts: contacts,
	}

	result, err := service.DeduplicateContacts(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OriginalCount != 2 {
		t.Errorf("expected OriginalCount=2, got %d", result.OriginalCount)
	}
}

func TestMergeContacts_EmptyList(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	_, err := service.MergeContacts(ctx, []domain.Contact{}, "prefer_first")
	if err == nil {
		t.Error("expected error for empty contact list")
	}
	if err != nil && err.Error() != "no contacts to merge" {
		t.Errorf("expected 'no contacts to merge' error, got: %v", err)
	}
}

func TestMergeContacts_SingleContact(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	contact := domain.Contact{
		ID:        "1",
		GivenName: "Alice",
		Surname:   "Smith",
		Emails:    []domain.ContactEmail{{Email: "alice@example.com"}},
	}

	result, err := service.MergeContacts(ctx, []domain.Contact{contact}, "prefer_first")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != "1" {
		t.Errorf("expected ID=1, got %s", result.ID)
	}
	if result.GivenName != "Alice" {
		t.Errorf("expected GivenName=Alice, got %s", result.GivenName)
	}
}

func TestParseVCard_NotImplemented(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	_, err := service.ParseVCard(ctx, "BEGIN:VCARD\nEND:VCARD")
	if err == nil {
		t.Error("expected error for unimplemented vCard parsing")
	}
}

func TestExportVCard_NotImplemented(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	contact := domain.Contact{
		ID:        "1",
		GivenName: "Alice",
		Surname:   "Smith",
	}

	_, err := service.ExportVCard(ctx, []domain.Contact{contact})
	if err == nil {
		t.Error("expected error for unimplemented vCard export")
	}
}

func TestMapVCardFields_NotImplemented(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	contact := domain.Contact{
		ID:        "1",
		GivenName: "Alice",
	}

	_, err := service.MapVCardFields(ctx, "outlook", "google", &contact)
	if err == nil {
		t.Error("expected error for unimplemented field mapping")
	}
}
