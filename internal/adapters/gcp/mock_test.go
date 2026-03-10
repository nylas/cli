package gcp

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClient_Defaults(t *testing.T) {
	ctx := context.Background()
	mock := &MockClient{}

	t.Run("CheckAuth returns default email", func(t *testing.T) {
		email, err := mock.CheckAuth(ctx)
		require.NoError(t, err)
		assert.Equal(t, "user@example.com", email)
	})

	t.Run("ListProjects returns default project", func(t *testing.T) {
		projects, err := mock.ListProjects(ctx)
		require.NoError(t, err)
		assert.Len(t, projects, 1)
		assert.Equal(t, "test-project", projects[0].ProjectID)
	})

	t.Run("CreateProject succeeds", func(t *testing.T) {
		err := mock.CreateProject(ctx, "proj", "Proj")
		assert.NoError(t, err)
	})

	t.Run("GetProject succeeds", func(t *testing.T) {
		err := mock.GetProject(ctx, "proj")
		assert.NoError(t, err)
	})

	t.Run("BatchEnableAPIs succeeds", func(t *testing.T) {
		err := mock.BatchEnableAPIs(ctx, "proj", []string{"api1"})
		assert.NoError(t, err)
	})

	t.Run("GetIAMPolicy returns empty policy", func(t *testing.T) {
		policy, err := mock.GetIAMPolicy(ctx, "proj")
		require.NoError(t, err)
		assert.NotNil(t, policy)
		assert.Empty(t, policy.Bindings)
	})

	t.Run("SetIAMPolicy succeeds", func(t *testing.T) {
		err := mock.SetIAMPolicy(ctx, "proj", &ports.IAMPolicy{})
		assert.NoError(t, err)
	})

	t.Run("CreateTopic succeeds", func(t *testing.T) {
		err := mock.CreateTopic(ctx, "proj", "topic")
		assert.NoError(t, err)
	})

	t.Run("TopicExists returns false by default", func(t *testing.T) {
		assert.False(t, mock.TopicExists(ctx, "proj", "topic"))
	})

	t.Run("SetTopicIAMPolicy succeeds", func(t *testing.T) {
		err := mock.SetTopicIAMPolicy(ctx, "proj", "topic", "member", "role")
		assert.NoError(t, err)
	})

	t.Run("CreateServiceAccount returns formatted email", func(t *testing.T) {
		email, err := mock.CreateServiceAccount(ctx, "proj", "sa-id", "SA Name")
		require.NoError(t, err)
		assert.Equal(t, "sa-id@proj.iam.gserviceaccount.com", email)
	})

	t.Run("ServiceAccountExists returns false by default", func(t *testing.T) {
		assert.False(t, mock.ServiceAccountExists(ctx, "proj", "sa@proj.iam.gserviceaccount.com"))
	})
}

func TestMockClient_CustomFunctions(t *testing.T) {
	ctx := context.Background()
	errCustom := errors.New("custom error")

	t.Run("CheckAuth with custom func", func(t *testing.T) {
		mock := &MockClient{
			CheckAuthFunc: func(_ context.Context) (string, error) {
				return "custom@example.com", nil
			},
		}
		email, err := mock.CheckAuth(ctx)
		require.NoError(t, err)
		assert.Equal(t, "custom@example.com", email)
	})

	t.Run("ListProjects with error", func(t *testing.T) {
		mock := &MockClient{
			ListProjectsFunc: func(_ context.Context) ([]domain.GCPProject, error) {
				return nil, errCustom
			},
		}
		_, err := mock.ListProjects(ctx)
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("CreateProject with error", func(t *testing.T) {
		mock := &MockClient{
			CreateProjectFunc: func(_ context.Context, _, _ string) error {
				return errCustom
			},
		}
		err := mock.CreateProject(ctx, "p", "P")
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("GetProject with error", func(t *testing.T) {
		mock := &MockClient{
			GetProjectFunc: func(_ context.Context, _ string) error {
				return errCustom
			},
		}
		err := mock.GetProject(ctx, "p")
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("BatchEnableAPIs with custom func", func(t *testing.T) {
		mock := &MockClient{
			BatchEnableAPIsFunc: func(_ context.Context, _ string, _ []string) error {
				return errCustom
			},
		}
		err := mock.BatchEnableAPIs(ctx, "p", nil)
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("GetIAMPolicy with custom func", func(t *testing.T) {
		mock := &MockClient{
			GetIAMPolicyFunc: func(_ context.Context, _ string) (*ports.IAMPolicy, error) {
				return nil, errCustom
			},
		}
		_, err := mock.GetIAMPolicy(ctx, "p")
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("SetIAMPolicy with custom func", func(t *testing.T) {
		mock := &MockClient{
			SetIAMPolicyFunc: func(_ context.Context, _ string, _ *ports.IAMPolicy) error {
				return errCustom
			},
		}
		err := mock.SetIAMPolicy(ctx, "p", nil)
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("TopicExists with custom func", func(t *testing.T) {
		mock := &MockClient{
			TopicExistsFunc: func(_ context.Context, _, _ string) bool {
				return true
			},
		}
		assert.True(t, mock.TopicExists(ctx, "p", "t"))
	})

	t.Run("CreateTopic with custom func", func(t *testing.T) {
		mock := &MockClient{
			CreateTopicFunc: func(_ context.Context, _, _ string) error {
				return errCustom
			},
		}
		err := mock.CreateTopic(ctx, "p", "t")
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("SetTopicIAMPolicy with custom func", func(t *testing.T) {
		mock := &MockClient{
			SetTopicIAMPolicyFunc: func(_ context.Context, _, _, _, _ string) error {
				return errCustom
			},
		}
		err := mock.SetTopicIAMPolicy(ctx, "p", "t", "m", "r")
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("CreateServiceAccount with custom func", func(t *testing.T) {
		mock := &MockClient{
			CreateServiceAccountFunc: func(_ context.Context, _, _, _ string) (string, error) {
				return "", errCustom
			},
		}
		_, err := mock.CreateServiceAccount(ctx, "p", "a", "n")
		assert.ErrorIs(t, err, errCustom)
	})

	t.Run("ServiceAccountExists with custom func", func(t *testing.T) {
		mock := &MockClient{
			ServiceAccountExistsFunc: func(_ context.Context, _, _ string) bool {
				return true
			},
		}
		assert.True(t, mock.ServiceAccountExists(ctx, "p", "e"))
	})
}
