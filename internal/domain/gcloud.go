package domain

import (
	"slices"
	"time"
)

// Google provider setup feature constants.
const (
	FeatureEmail    = "email"
	FeatureCalendar = "calendar"
	FeatureContacts = "contacts"
	FeaturePubSub   = "pubsub"
)

// Google provider setup step constants.
const (
	StepCreateProject  = "create_project"
	StepEnableAPIs     = "enable_apis"
	StepIAMOwner       = "iam_owner"
	StepPubSubTopic    = "pubsub_topic"
	StepServiceAccount = "service_account"
	StepPubSubPublish  = "pubsub_publisher"
	StepConsentScreen  = "consent_screen"
	StepCredentials    = "credentials"
	StepConnector      = "connector"
)

// Google provider setup constants.
const (
	NylasSupportEmail         = "support@nylas.com"
	NylasPubSubTopicName      = "nylas-gmail-realtime"
	NylasPubSubServiceAccount = "nylas-gmail-realtime"
)

// GCPProject represents a Google Cloud Platform project.
type GCPProject struct {
	ProjectID   string `json:"project_id"`
	DisplayName string `json:"display_name"`
	State       string `json:"state"`
}

// GoogleSetupConfig holds configuration gathered during the setup wizard.
type GoogleSetupConfig struct {
	ProjectID         string
	DisplayName       string
	Region            string
	Features          []string
	SkipConfirmations bool
	IsNewProject      bool
	ClientID          string
	ClientSecret      string
}

// HasFeature checks if a feature is selected.
func (c *GoogleSetupConfig) HasFeature(feature string) bool {
	return slices.Contains(c.Features, feature)
}

// SetupState holds checkpoint state for resume support.
type SetupState struct {
	ProjectID      string    `json:"project_id"`
	DisplayName    string    `json:"display_name,omitempty"`
	Region         string    `json:"region"`
	Features       []string  `json:"features"`
	IsNewProject   bool      `json:"is_new_project,omitempty"`
	CompletedSteps []string  `json:"completed_steps"`
	PendingStep    string    `json:"pending_step,omitempty"`
	StartedAt      time.Time `json:"started_at"`
}

// IsStepCompleted checks if a step has been completed.
func (s *SetupState) IsStepCompleted(step string) bool {
	return slices.Contains(s.CompletedSteps, step)
}

// CompleteStep marks a step as completed.
func (s *SetupState) CompleteStep(step string) {
	if !s.IsStepCompleted(step) {
		s.CompletedSteps = append(s.CompletedSteps, step)
	}
	s.PendingStep = ""
}

// IsExpired returns true if the state is older than 24 hours.
func (s *SetupState) IsExpired() bool {
	return time.Since(s.StartedAt) > 24*time.Hour
}
