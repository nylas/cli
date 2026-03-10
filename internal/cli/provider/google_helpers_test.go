package provider

import (
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestGenerateProjectID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "My App",
			expected: "my-app-nylas",
		},
		{
			name:     "with special chars",
			input:    "My App! @#$",
			expected: "my-app-nylas",
		},
		{
			name:     "long name gets truncated",
			input:    "this is a very long project name that exceeds limits",
			expected: "this-is-a-very-long-proj-nylas",
		},
		{
			name:     "empty name gets padded",
			input:    "",
			expected: "-nylas",
		},
		{
			name:     "already lowercase",
			input:    "test-project",
			expected: "test-project-nylas",
		},
		{
			name:     "multiple dashes collapsed",
			input:    "my---app",
			expected: "my-app-nylas",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateProjectID(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.GreaterOrEqual(t, len(result), 6, "project ID must be at least 6 chars")
			assert.LessOrEqual(t, len(result), 30, "project ID must be at most 30 chars")
		})
	}
}

func TestFeatureToAPIs(t *testing.T) {
	tests := []struct {
		name     string
		features []string
		expected []string
	}{
		{
			name:     "all features",
			features: []string{domain.FeatureEmail, domain.FeatureCalendar, domain.FeatureContacts, domain.FeaturePubSub},
			expected: []string{"gmail.googleapis.com", "calendar-json.googleapis.com", "people.googleapis.com", "pubsub.googleapis.com"},
		},
		{
			name:     "email only",
			features: []string{domain.FeatureEmail},
			expected: []string{"gmail.googleapis.com"},
		},
		{
			name:     "empty features",
			features: []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := featureToAPIs(tt.features)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFeatureToScopes(t *testing.T) {
	t.Run("email has gmail scopes", func(t *testing.T) {
		scopes := featureToScopes([]string{domain.FeatureEmail})
		assert.Contains(t, scopes, "https://www.googleapis.com/auth/gmail.modify")
		assert.Contains(t, scopes, "https://www.googleapis.com/auth/gmail.send")
	})

	t.Run("calendar has calendar scope", func(t *testing.T) {
		scopes := featureToScopes([]string{domain.FeatureCalendar})
		assert.Contains(t, scopes, "https://www.googleapis.com/auth/calendar")
	})

	t.Run("no duplicate scopes", func(t *testing.T) {
		scopes := featureToScopes([]string{domain.FeatureEmail, domain.FeatureEmail})
		seen := map[string]bool{}
		for _, s := range scopes {
			assert.False(t, seen[s], "duplicate scope: %s", s)
			seen[s] = true
		}
	})
}

func TestRedirectURI(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		expected string
	}{
		{"us region", "us", "https://api.us.nylas.com/v3/connect/callback"},
		{"eu region", "eu", "https://api.eu.nylas.com/v3/connect/callback"},
		{"default region", "", "https://api.us.nylas.com/v3/connect/callback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, redirectURI(tt.region))
		})
	}
}

func TestConsentScreenURL(t *testing.T) {
	url := consentScreenURL("my-project")
	assert.Contains(t, url, "my-project")
	assert.Contains(t, url, "consent")
}

func TestCredentialsURL(t *testing.T) {
	url := credentialsURL("my-project")
	assert.Contains(t, url, "my-project")
	assert.Contains(t, url, "oauthclient")
}

func TestFeatureLabel(t *testing.T) {
	assert.Contains(t, featureLabel(domain.FeatureEmail), "Gmail")
	assert.Contains(t, featureLabel(domain.FeatureCalendar), "Calendar")
	assert.Equal(t, "unknown", featureLabel("unknown"))
}
