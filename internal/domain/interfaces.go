package domain

import (
	"fmt"
)

// =============================================================================
// Person Type (Base for EmailParticipant and Participant)
// =============================================================================

// Person represents a person with name and email.
// This is the base type for EmailParticipant and embedded in Participant.
type Person struct {
	Name  string `json:"name,omitempty"`
	Email string `json:"email"`
}

// String returns a formatted display string for the person.
func (p Person) String() string {
	if p.Name != "" {
		return fmt.Sprintf("%s <%s>", p.Name, p.Email)
	}
	return p.Email
}

// DisplayName returns the name if available, otherwise the email.
func (p Person) DisplayName() string {
	if p.Name != "" {
		return p.Name
	}
	return p.Email
}
