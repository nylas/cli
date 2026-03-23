package provider

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nylas/cli/internal/domain"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9-]`)
var multiDash = regexp.MustCompile(`-{2,}`)

// generateProjectID creates a valid GCP project ID from a display name.
// Project IDs must be 6-30 chars, lowercase, start with a letter.
func generateProjectID(name string) string {
	id := strings.ToLower(strings.TrimSpace(name))
	id = nonAlphaNum.ReplaceAllString(id, "-")
	id = multiDash.ReplaceAllString(id, "-")
	id = strings.Trim(id, "-")

	if id == "" {
		id = "project"
	}

	// GCP project IDs must start with a letter.
	if id[0] < 'a' || id[0] > 'z' {
		id = "p-" + id
	}

	// Append "-nylas" suffix
	if len(id) > 24 {
		id = id[:24]
	}
	id = strings.TrimRight(id, "-")
	id += "-nylas"

	// Ensure minimum length
	if len(id) < 6 {
		id = id + "-project"
	}

	return id
}

// featureToAPIs maps selected features to Google API service IDs.
func featureToAPIs(features []string) []string {
	apiMap := map[string]string{
		domain.FeatureEmail:    "gmail.googleapis.com",
		domain.FeatureCalendar: "calendar-json.googleapis.com",
		domain.FeatureContacts: "people.googleapis.com",
		domain.FeaturePubSub:   "pubsub.googleapis.com",
	}

	var apis []string
	for _, f := range features {
		if api, ok := apiMap[f]; ok {
			apis = append(apis, api)
		}
	}
	return apis
}

// featureToScopes maps selected features to Google OAuth scopes.
func featureToScopes(features []string) []string {
	scopeMap := map[string][]string{
		domain.FeatureEmail: {
			"https://www.googleapis.com/auth/gmail.modify",
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.compose",
			"https://www.googleapis.com/auth/gmail.send",
		},
		domain.FeatureCalendar: {
			"https://www.googleapis.com/auth/calendar",
		},
		domain.FeatureContacts: {
			"https://www.googleapis.com/auth/contacts",
		},
	}

	var scopes []string
	seen := map[string]bool{}
	for _, f := range features {
		for _, s := range scopeMap[f] {
			if !seen[s] {
				scopes = append(scopes, s)
				seen[s] = true
			}
		}
	}
	return scopes
}

// consentScreenURL returns the GCP console URL for configuring the OAuth consent screen.
func consentScreenURL(projectID string) string {
	return fmt.Sprintf("https://console.cloud.google.com/apis/credentials/consent?project=%s", projectID)
}

// credentialsURL returns the GCP console URL for creating OAuth credentials.
func credentialsURL(projectID string) string {
	return fmt.Sprintf("https://console.cloud.google.com/apis/credentials/oauthclient?project=%s", projectID)
}

// redirectURI returns the Nylas OAuth callback URI for the given region.
func redirectURI(region string) string {
	if region == "eu" {
		return "https://api.eu.nylas.com/v3/connect/callback"
	}
	return "https://api.us.nylas.com/v3/connect/callback"
}

// featureLabel returns a human-readable label for a feature.
func featureLabel(feature string) string {
	labels := map[string]string{
		domain.FeatureEmail:    "Email (Gmail API)",
		domain.FeatureCalendar: "Calendar (Google Calendar API)",
		domain.FeatureContacts: "Contacts (People API)",
		domain.FeaturePubSub:   "Real-time sync via Pub/Sub",
	}
	if label, ok := labels[feature]; ok {
		return label
	}
	return feature
}
