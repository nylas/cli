//go:build !integration
// +build !integration

package nylas

import "testing"

func TestConvertContactIncludesUpdatedAt(t *testing.T) {
	contact := convertContact(contactResponse{
		ID:        "contact-1",
		UpdatedAt: 1700000000,
	})

	if contact.UpdatedAt != 1700000000 {
		t.Fatalf("UpdatedAt = %d, want 1700000000", contact.UpdatedAt)
	}
}
