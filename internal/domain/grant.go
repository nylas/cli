package domain

import (
	"encoding/json"
	"time"
)

// UnixTime wraps time.Time to handle Unix timestamp JSON unmarshaling
type UnixTime struct {
	time.Time
}

// UnmarshalJSON handles both Unix timestamps (integers) and RFC3339 strings
func (ut *UnixTime) UnmarshalJSON(data []byte) error {
	// Try as integer (Unix timestamp)
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err == nil {
		ut.Time = time.Unix(timestamp, 0)
		return nil
	}

	// Try as string (RFC3339)
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		t, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return err
		}
		ut.Time = t
		return nil
	}

	return nil // Ignore if neither format
}

// Grant represents a Nylas grant (authenticated account).
type Grant struct {
	ID           string   `json:"id"`
	Provider     Provider `json:"provider"`
	Email        string   `json:"email"`
	GrantStatus  string   `json:"grant_status"`
	Scope        []string `json:"scope,omitempty"`
	CreatedAt    UnixTime `json:"created_at,omitempty"`
	UpdatedAt    UnixTime `json:"updated_at,omitempty"`
	AccessToken  string   `json:"access_token,omitempty"`
	RefreshToken string   `json:"refresh_token,omitempty"`
}

// IsValid returns true if the grant is in a valid state.
func (g *Grant) IsValid() bool {
	return g.GrantStatus == "valid"
}

// GrantInfo is a lightweight representation of a grant for storage.
type GrantInfo struct {
	ID       string   `yaml:"id" json:"id"`
	Email    string   `yaml:"email" json:"email"`
	Provider Provider `yaml:"provider" json:"provider"`
}

// GrantStatus represents the status information for a grant.
type GrantStatus struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Provider  Provider `json:"provider"`
	Status    string   `json:"status"`
	IsDefault bool     `json:"is_default"`
	Error     string   `json:"error,omitempty"`
}
