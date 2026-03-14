package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/gcp"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockReader simulates user input for testing.
type mockReader struct {
	inputs []string
	idx    int
}

func newMockReader(inputs ...string) *mockReader {
	return &mockReader{inputs: inputs}
}

func (m *mockReader) ReadString(_ byte) (string, error) {
	if m.idx >= len(m.inputs) {
		return "\n", nil
	}
	s := m.inputs[m.idx] + "\n"
	m.idx++
	return s, nil
}

func TestPromptProjectSelection_FlagOverride(t *testing.T) {
	ctx := context.Background()
	mock := &gcp.MockClient{}

	projectID, _, isNew, err := promptProjectSelection(ctx, mock, newMockReader(), "my-project")
	require.NoError(t, err)
	assert.Equal(t, "my-project", projectID)
	assert.False(t, isNew)
}

func TestPromptProjectSelection_ExistingProject(t *testing.T) {
	ctx := context.Background()
	mock := &gcp.MockClient{
		ListProjectsFunc: func(_ context.Context) ([]domain.GCPProject, error) {
			return []domain.GCPProject{
				{ProjectID: "proj-1", DisplayName: "Project One"},
				{ProjectID: "proj-2", DisplayName: "Project Two"},
			}, nil
		},
	}

	reader := newMockReader("1") // Select first project
	projectID, _, isNew, err := promptProjectSelection(ctx, mock, reader, "")
	require.NoError(t, err)
	assert.Equal(t, "proj-1", projectID)
	assert.False(t, isNew)
}

func TestPromptProjectSelection_NewProject(t *testing.T) {
	ctx := context.Background()
	mock := &gcp.MockClient{
		ListProjectsFunc: func(_ context.Context) ([]domain.GCPProject, error) {
			return []domain.GCPProject{
				{ProjectID: "proj-1", DisplayName: "Project One"},
			}, nil
		},
	}

	reader := newMockReader("2", "My New App", "y") // Select "Create new", enter name, accept generated ID
	projectID, displayName, isNew, err := promptProjectSelection(ctx, mock, reader, "")
	require.NoError(t, err)
	assert.Equal(t, "my-new-app-nylas", projectID)
	assert.Equal(t, "My New App", displayName)
	assert.True(t, isNew)
}

func TestPromptProjectSelection_NewProject_CustomID(t *testing.T) {
	ctx := context.Background()
	mock := &gcp.MockClient{
		ListProjectsFunc: func(_ context.Context) ([]domain.GCPProject, error) {
			return []domain.GCPProject{
				{ProjectID: "proj-1", DisplayName: "Project One"},
			}, nil
		},
	}

	// Select "Create new", enter name, reject generated ID, enter custom ID
	reader := newMockReader("2", "My App", "custom", "my-custom-id")
	projectID, displayName, isNew, err := promptProjectSelection(ctx, mock, reader, "")
	require.NoError(t, err)
	assert.Equal(t, "my-custom-id", projectID)
	assert.Equal(t, "My App", displayName)
	assert.True(t, isNew)
}

func TestPromptProjectSelection_NewProject_InlineCustomID(t *testing.T) {
	ctx := context.Background()
	mock := &gcp.MockClient{
		ListProjectsFunc: func(_ context.Context) ([]domain.GCPProject, error) {
			return []domain.GCPProject{}, nil
		},
	}

	// Select "Create new" (only option), enter name, type custom ID directly
	reader := newMockReader("1", "My App", "inline-id")
	projectID, _, isNew, err := promptProjectSelection(ctx, mock, reader, "")
	require.NoError(t, err)
	assert.Equal(t, "inline-id", projectID)
	assert.True(t, isNew)
}

func TestPromptProjectSelection_InvalidSelection(t *testing.T) {
	ctx := context.Background()
	mock := &gcp.MockClient{
		ListProjectsFunc: func(_ context.Context) ([]domain.GCPProject, error) {
			return []domain.GCPProject{
				{ProjectID: "proj-1", DisplayName: "Project One"},
			}, nil
		},
	}

	reader := newMockReader("99")
	_, _, _, err := promptProjectSelection(ctx, mock, reader, "")
	assert.Error(t, err)
}

func TestPromptFeatureSelection_Empty(t *testing.T) {
	reader := newMockReader("")
	features, err := promptFeatureSelection(reader)
	require.NoError(t, err)
	assert.Len(t, features, 4) // empty = "all"
}

func TestPromptFeatureSelection_Invalid(t *testing.T) {
	reader := newMockReader("99")
	_, err := promptFeatureSelection(reader)
	assert.Error(t, err)
}

func TestPromptFeatureSelection_All(t *testing.T) {
	reader := newMockReader("all")
	features, err := promptFeatureSelection(reader)
	require.NoError(t, err)
	assert.Len(t, features, 4)
}

func TestPromptFeatureSelection_Specific(t *testing.T) {
	reader := newMockReader("1,3")
	features, err := promptFeatureSelection(reader)
	require.NoError(t, err)
	assert.Equal(t, []string{domain.FeatureEmail, domain.FeatureContacts}, features)
}

func TestPromptRegion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{"us region", "us", "us", false},
		{"eu region", "eu", "eu", false},
		{"default", "", "us", false},
		{"invalid", "invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newMockReader(tt.input)
			region, err := promptRegion(reader)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, region)
			}
		})
	}
}

func TestCreateGCPProject(t *testing.T) {
	t.Run("new project", func(t *testing.T) {
		ctx := context.Background()
		created := false
		mock := &gcp.MockClient{
			CreateProjectFunc: func(_ context.Context, projectID, displayName string) error {
				created = true
				assert.Equal(t, "my-project", projectID)
				assert.Equal(t, "My Project", displayName)
				return nil
			},
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID:    "my-project",
			DisplayName:  "My Project",
			IsNewProject: true,
		}

		err := createGCPProject(ctx, mock, cfg)
		require.NoError(t, err)
		assert.True(t, created)
	})

	t.Run("existing project skips creation", func(t *testing.T) {
		ctx := context.Background()
		cfg := &domain.GoogleSetupConfig{IsNewProject: false}

		err := createGCPProject(ctx, &gcp.MockClient{}, cfg)
		require.NoError(t, err)
	})
}

func TestEnableAPIs(t *testing.T) {
	ctx := context.Background()
	var enabledAPIs []string
	mock := &gcp.MockClient{
		BatchEnableAPIsFunc: func(_ context.Context, _ string, apis []string) error {
			enabledAPIs = apis
			return nil
		},
	}

	cfg := &domain.GoogleSetupConfig{
		ProjectID: "my-project",
		Features:  []string{domain.FeatureEmail, domain.FeaturePubSub},
	}

	err := enableAPIs(ctx, mock, cfg)
	require.NoError(t, err)
	assert.Contains(t, enabledAPIs, "gmail.googleapis.com")
	assert.Contains(t, enabledAPIs, "pubsub.googleapis.com")
}

func TestAddIAMOwner(t *testing.T) {
	t.Run("adds owner when not present", func(t *testing.T) {
		ctx := context.Background()
		var savedPolicy *ports.IAMPolicy
		mock := &gcp.MockClient{
			GetIAMPolicyFunc: func(_ context.Context, _ string) (*ports.IAMPolicy, error) {
				return &ports.IAMPolicy{}, nil
			},
			SetIAMPolicyFunc: func(_ context.Context, _ string, policy *ports.IAMPolicy) error {
				savedPolicy = policy
				return nil
			},
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID:         "my-project",
			SkipConfirmations: true,
		}

		err := addIAMOwner(ctx, mock, cfg, newMockReader())
		require.NoError(t, err)
		require.NotNil(t, savedPolicy)
		assert.True(t, savedPolicy.HasMemberInRole("roles/owner", "user:support@nylas.com"))
	})

	t.Run("skips when already present", func(t *testing.T) {
		ctx := context.Background()
		mock := &gcp.MockClient{
			GetIAMPolicyFunc: func(_ context.Context, _ string) (*ports.IAMPolicy, error) {
				policy := &ports.IAMPolicy{}
				policy.AddBinding("roles/owner", "user:support@nylas.com")
				return policy, nil
			},
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID:         "my-project",
			SkipConfirmations: true,
		}

		err := addIAMOwner(ctx, mock, cfg, newMockReader())
		require.NoError(t, err)
	})

	t.Run("user can decline", func(t *testing.T) {
		ctx := context.Background()
		mock := &gcp.MockClient{}

		cfg := &domain.GoogleSetupConfig{
			ProjectID: "my-project",
		}

		reader := newMockReader("n")
		err := addIAMOwner(ctx, mock, cfg, reader)
		require.NoError(t, err)
	})
}

func TestSetupPubSub(t *testing.T) {
	t.Run("skips when feature not selected", func(t *testing.T) {
		ctx := context.Background()
		cfg := &domain.GoogleSetupConfig{Features: []string{domain.FeatureEmail}}
		state := &domain.SetupState{}

		err := setupPubSub(ctx, &gcp.MockClient{}, cfg, state, t.TempDir())
		require.NoError(t, err)
	})

	t.Run("creates all resources", func(t *testing.T) {
		ctx := context.Background()
		topicCreated := false
		saCreated := false
		policySet := false

		mock := &gcp.MockClient{
			CreateTopicFunc: func(_ context.Context, _, _ string) error {
				topicCreated = true
				return nil
			},
			CreateServiceAccountFunc: func(_ context.Context, _, _, _ string) (string, error) {
				saCreated = true
				return "sa@test.iam.gserviceaccount.com", nil
			},
			SetTopicIAMPolicyFunc: func(_ context.Context, _, _, _, _ string) error {
				policySet = true
				return nil
			},
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID: "my-project",
			Features:  []string{domain.FeaturePubSub},
		}
		state := &domain.SetupState{}

		err := setupPubSub(ctx, mock, cfg, state, t.TempDir())
		require.NoError(t, err)
		assert.True(t, topicCreated)
		assert.True(t, saCreated)
		assert.True(t, policySet)
	})
}

func TestPromptOAuthCredentials(t *testing.T) {
	// Test with regular reader (non-terminal fallback)
	reader := newMockReader("my-client-id.apps.googleusercontent.com", "my-secret")
	clientID, clientSecret, err := promptOAuthCredentials(reader)
	require.NoError(t, err)
	assert.Equal(t, "my-client-id.apps.googleusercontent.com", clientID)
	assert.True(t, strings.Contains(clientSecret, "my-secret") || clientSecret != "")
}

func TestGoogleSetupConfig_HasFeature(t *testing.T) {
	cfg := &domain.GoogleSetupConfig{
		Features: []string{domain.FeatureEmail, domain.FeatureCalendar, domain.FeaturePubSub},
	}

	assert.True(t, cfg.HasFeature(domain.FeatureEmail))
	assert.True(t, cfg.HasFeature(domain.FeaturePubSub))
	assert.False(t, cfg.HasFeature(domain.FeatureContacts))
}
