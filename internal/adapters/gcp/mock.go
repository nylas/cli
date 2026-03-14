package gcp

import (
	"context"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// MockClient implements ports.GCPClient for testing.
type MockClient struct {
	CheckAuthFunc            func(ctx context.Context) (string, error)
	ListProjectsFunc         func(ctx context.Context) ([]domain.GCPProject, error)
	CreateProjectFunc        func(ctx context.Context, projectID, displayName string) error
	GetProjectFunc           func(ctx context.Context, projectID string) error
	BatchEnableAPIsFunc      func(ctx context.Context, projectID string, apis []string) error
	GetIAMPolicyFunc         func(ctx context.Context, projectID string) (*ports.IAMPolicy, error)
	SetIAMPolicyFunc         func(ctx context.Context, projectID string, policy *ports.IAMPolicy) error
	CreateTopicFunc          func(ctx context.Context, projectID, topicName string) error
	TopicExistsFunc          func(ctx context.Context, projectID, topicName string) bool
	SetTopicIAMPolicyFunc    func(ctx context.Context, projectID, topicName, member, role string) error
	CreateServiceAccountFunc func(ctx context.Context, projectID, accountID, displayName string) (string, error)
	ServiceAccountExistsFunc func(ctx context.Context, projectID, email string) bool
}

func (m *MockClient) CheckAuth(ctx context.Context) (string, error) {
	if m.CheckAuthFunc != nil {
		return m.CheckAuthFunc(ctx)
	}
	return "user@example.com", nil
}

func (m *MockClient) ListProjects(ctx context.Context) ([]domain.GCPProject, error) {
	if m.ListProjectsFunc != nil {
		return m.ListProjectsFunc(ctx)
	}
	return []domain.GCPProject{
		{ProjectID: "test-project", DisplayName: "Test Project", State: "ACTIVE"},
	}, nil
}

func (m *MockClient) CreateProject(ctx context.Context, projectID, displayName string) error {
	if m.CreateProjectFunc != nil {
		return m.CreateProjectFunc(ctx, projectID, displayName)
	}
	return nil
}

func (m *MockClient) GetProject(ctx context.Context, projectID string) error {
	if m.GetProjectFunc != nil {
		return m.GetProjectFunc(ctx, projectID)
	}
	return nil
}

func (m *MockClient) BatchEnableAPIs(ctx context.Context, projectID string, apis []string) error {
	if m.BatchEnableAPIsFunc != nil {
		return m.BatchEnableAPIsFunc(ctx, projectID, apis)
	}
	return nil
}

func (m *MockClient) GetIAMPolicy(ctx context.Context, projectID string) (*ports.IAMPolicy, error) {
	if m.GetIAMPolicyFunc != nil {
		return m.GetIAMPolicyFunc(ctx, projectID)
	}
	return &ports.IAMPolicy{}, nil
}

func (m *MockClient) SetIAMPolicy(ctx context.Context, projectID string, policy *ports.IAMPolicy) error {
	if m.SetIAMPolicyFunc != nil {
		return m.SetIAMPolicyFunc(ctx, projectID, policy)
	}
	return nil
}

func (m *MockClient) CreateTopic(ctx context.Context, projectID, topicName string) error {
	if m.CreateTopicFunc != nil {
		return m.CreateTopicFunc(ctx, projectID, topicName)
	}
	return nil
}

func (m *MockClient) TopicExists(ctx context.Context, projectID, topicName string) bool {
	if m.TopicExistsFunc != nil {
		return m.TopicExistsFunc(ctx, projectID, topicName)
	}
	return false
}

func (m *MockClient) SetTopicIAMPolicy(ctx context.Context, projectID, topicName, member, role string) error {
	if m.SetTopicIAMPolicyFunc != nil {
		return m.SetTopicIAMPolicyFunc(ctx, projectID, topicName, member, role)
	}
	return nil
}

func (m *MockClient) CreateServiceAccount(ctx context.Context, projectID, accountID, displayName string) (string, error) {
	if m.CreateServiceAccountFunc != nil {
		return m.CreateServiceAccountFunc(ctx, projectID, accountID, displayName)
	}
	return accountID + "@" + projectID + ".iam.gserviceaccount.com", nil
}

func (m *MockClient) ServiceAccountExists(ctx context.Context, projectID, email string) bool {
	if m.ServiceAccountExistsFunc != nil {
		return m.ServiceAccountExistsFunc(ctx, projectID, email)
	}
	return false
}
