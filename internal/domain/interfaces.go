package domain

import (
	"fmt"
	"time"
)

// =============================================================================
// Shared Interfaces
// =============================================================================

// Validator is implemented by types that can validate themselves.
type Validator interface {
	Validate() error
}

// Paginated is implemented by all paginated response types.
type Paginated interface {
	GetPagination() Pagination
	HasMore() bool
}

// Resource is implemented by all domain resources with ID.
type Resource interface {
	GetID() string
}

// Timestamped is implemented by resources with creation/update timestamps.
type Timestamped interface {
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

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

// =============================================================================
// List Response Helpers
// =============================================================================

// FilterFunc is a predicate function for filtering list items.
type FilterFunc[T any] func(T) bool

// MapFunc transforms an item of type T to type R.
type MapFunc[T, R any] func(T) R

// Filter returns items matching the predicate.
func Filter[T any](items []T, predicate FilterFunc[T]) []T {
	var result []T
	for _, item := range items {
		if predicate(item) {
			result = append(result, item)
		}
	}
	return result
}

// Map transforms a slice using the provided function.
func Map[T, R any](items []T, fn MapFunc[T, R]) []R {
	result := make([]R, len(items))
	for i, item := range items {
		result[i] = fn(item)
	}
	return result
}

// Each iterates over items and calls the function for each.
func Each[T any](items []T, fn func(T)) {
	for _, item := range items {
		fn(item)
	}
}
