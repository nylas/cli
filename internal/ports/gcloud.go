package ports

import (
	"context"

	"github.com/nylas/cli/internal/domain"
)

// GCPClient defines the interface for interacting with Google Cloud Platform.
type GCPClient interface {
	// CheckAuth verifies ADC works and returns the authenticated email.
	CheckAuth(ctx context.Context) (string, error)

	// ListProjects lists the user's accessible GCP projects.
	ListProjects(ctx context.Context) ([]domain.GCPProject, error)

	// CreateProject creates a new GCP project.
	CreateProject(ctx context.Context, projectID, displayName string) error

	// GetProject checks if a project exists. Returns nil if it exists.
	GetProject(ctx context.Context, projectID string) error

	// BatchEnableAPIs enables multiple APIs for a project.
	BatchEnableAPIs(ctx context.Context, projectID string, apis []string) error

	// GetIAMPolicy retrieves the IAM policy for a project.
	GetIAMPolicy(ctx context.Context, projectID string) (*IAMPolicy, error)

	// SetIAMPolicy sets the IAM policy for a project.
	SetIAMPolicy(ctx context.Context, projectID string, policy *IAMPolicy) error

	// CreateTopic creates a Pub/Sub topic.
	CreateTopic(ctx context.Context, projectID, topicName string) error

	// TopicExists checks if a Pub/Sub topic exists.
	TopicExists(ctx context.Context, projectID, topicName string) bool

	// SetTopicIAMPolicy sets IAM policy on a Pub/Sub topic.
	SetTopicIAMPolicy(ctx context.Context, projectID, topicName, member, role string) error

	// CreateServiceAccount creates a service account.
	CreateServiceAccount(ctx context.Context, projectID, accountID, displayName string) (string, error)

	// ServiceAccountExists checks if a service account exists.
	ServiceAccountExists(ctx context.Context, projectID, email string) bool
}

// IAMPolicy represents a simplified IAM policy for GCP projects.
type IAMPolicy struct {
	Bindings []*IAMBinding
	Etag     string
}

// IAMBinding represents a single IAM policy binding.
type IAMBinding struct {
	Role    string
	Members []string
}

// HasMemberInRole checks if a member exists in a specific role.
func (p *IAMPolicy) HasMemberInRole(role, member string) bool {
	for _, b := range p.Bindings {
		if b.Role == role {
			for _, m := range b.Members {
				if m == member {
					return true
				}
			}
		}
	}
	return false
}

// AddBinding adds a member to a role, creating the binding if needed.
func (p *IAMPolicy) AddBinding(role, member string) {
	for _, b := range p.Bindings {
		if b.Role == role {
			for _, m := range b.Members {
				if m == member {
					return
				}
			}
			b.Members = append(b.Members, member)
			return
		}
	}
	p.Bindings = append(p.Bindings, &IAMBinding{
		Role:    role,
		Members: []string{member},
	})
}
