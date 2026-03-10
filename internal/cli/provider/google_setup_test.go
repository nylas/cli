package provider

import (
	"context"
	"testing"
	"time"

	"github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/adapters/gcp"
	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleSetupOpts_HasFeatureFlags(t *testing.T) {
	tests := []struct {
		name     string
		opts     googleSetupOpts
		expected bool
	}{
		{"no flags", googleSetupOpts{}, false},
		{"email only", googleSetupOpts{email: true}, true},
		{"calendar only", googleSetupOpts{calendar: true}, true},
		{"contacts only", googleSetupOpts{contacts: true}, true},
		{"pubsub only", googleSetupOpts{pubsub: true}, true},
		{"multiple flags", googleSetupOpts{email: true, calendar: true}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.opts.hasFeatureFlags())
		})
	}
}

func TestGoogleSetupOpts_SelectedFeatures(t *testing.T) {
	tests := []struct {
		name     string
		opts     googleSetupOpts
		expected []string
	}{
		{
			name:     "no flags",
			opts:     googleSetupOpts{},
			expected: nil,
		},
		{
			name:     "all flags",
			opts:     googleSetupOpts{email: true, calendar: true, contacts: true, pubsub: true},
			expected: []string{domain.FeatureEmail, domain.FeatureCalendar, domain.FeatureContacts, domain.FeaturePubSub},
		},
		{
			name:     "email and pubsub",
			opts:     googleSetupOpts{email: true, pubsub: true},
			expected: []string{domain.FeatureEmail, domain.FeaturePubSub},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.opts.selectedFeatures())
		})
	}
}

func TestGatherConfig(t *testing.T) {
	t.Run("with all flags set", func(t *testing.T) {
		cfg := &domain.GoogleSetupConfig{}
		opts := &googleSetupOpts{
			projectID: "my-project",
			region:    "eu",
			email:     true,
			calendar:  true,
		}

		err := gatherConfig(context.Background(), nil, newMockReader(), cfg, opts)
		assert.NoError(t, err)
		assert.Equal(t, "my-project", cfg.ProjectID)
		assert.Equal(t, "eu", cfg.Region)
		assert.Equal(t, []string{domain.FeatureEmail, domain.FeatureCalendar}, cfg.Features)
		assert.False(t, cfg.IsNewProject)
	})

	t.Run("with interactive prompts", func(t *testing.T) {
		ctx := context.Background()
		mock := &gcp.MockClient{
			ListProjectsFunc: func(_ context.Context) ([]domain.GCPProject, error) {
				return []domain.GCPProject{
					{ProjectID: "proj-1", DisplayName: "Project One"},
				}, nil
			},
		}

		cfg := &domain.GoogleSetupConfig{}
		opts := &googleSetupOpts{}

		// Select project 1, select all features, select us region
		reader := newMockReader("1", "all", "us")
		err := gatherConfig(ctx, mock, reader, cfg, opts)
		assert.NoError(t, err)
		assert.Equal(t, "proj-1", cfg.ProjectID)
		assert.Equal(t, "us", cfg.Region)
		assert.Len(t, cfg.Features, 4)
	})
}

func TestPromptResume(t *testing.T) {
	t.Run("accept resume", func(t *testing.T) {
		state := &domain.SetupState{
			ProjectID:      "test-project",
			CompletedSteps: []string{"create_project", "enable_apis"},
			PendingStep:    "iam_owner",
		}
		reader := newMockReader("y")
		resume, err := promptResume(reader, state)
		assert.NoError(t, err)
		assert.True(t, resume)
	})

	t.Run("decline resume", func(t *testing.T) {
		state := &domain.SetupState{
			ProjectID:      "test-project",
			CompletedSteps: []string{"create_project"},
		}
		reader := newMockReader("n")
		resume, err := promptResume(reader, state)
		assert.NoError(t, err)
		assert.False(t, resume)
	})

	t.Run("empty input accepts", func(t *testing.T) {
		state := &domain.SetupState{
			ProjectID:      "test-project",
			CompletedSteps: []string{"create_project"},
			PendingStep:    "enable_apis",
		}
		reader := newMockReader("")
		resume, err := promptResume(reader, state)
		assert.NoError(t, err)
		assert.True(t, resume)
	})
}

func TestRunPhase1(t *testing.T) {
	t.Run("all steps from scratch", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()

		var createdProject, enabledAPIs, setIAM, createdTopic, createdSA, grantedPublisher bool

		mock := &gcp.MockClient{
			CreateProjectFunc: func(_ context.Context, _, _ string) error {
				createdProject = true
				return nil
			},
			BatchEnableAPIsFunc: func(_ context.Context, _ string, _ []string) error {
				enabledAPIs = true
				return nil
			},
			GetIAMPolicyFunc: func(_ context.Context, _ string) (*ports.IAMPolicy, error) {
				return &ports.IAMPolicy{}, nil
			},
			SetIAMPolicyFunc: func(_ context.Context, _ string, _ *ports.IAMPolicy) error {
				setIAM = true
				return nil
			},
			CreateTopicFunc: func(_ context.Context, _, _ string) error {
				createdTopic = true
				return nil
			},
			CreateServiceAccountFunc: func(_ context.Context, _, _, _ string) (string, error) {
				createdSA = true
				return "sa@proj.iam.gserviceaccount.com", nil
			},
			SetTopicIAMPolicyFunc: func(_ context.Context, _, _, _, _ string) error {
				grantedPublisher = true
				return nil
			},
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID:         "my-project",
			DisplayName:       "My Project",
			IsNewProject:      true,
			Features:          []string{domain.FeatureEmail, domain.FeaturePubSub},
			SkipConfirmations: true,
		}

		state := &domain.SetupState{StartedAt: time.Now()}

		err := runPhase1(ctx, mock, cfg, state, dir, newMockReader("y"))
		require.NoError(t, err)

		assert.True(t, createdProject)
		assert.True(t, enabledAPIs)
		assert.True(t, setIAM)
		assert.True(t, createdTopic)
		assert.True(t, createdSA)
		assert.True(t, grantedPublisher)

		// Verify state was saved
		assert.True(t, state.IsStepCompleted(domain.StepCreateProject))
		assert.True(t, state.IsStepCompleted(domain.StepEnableAPIs))
		assert.True(t, state.IsStepCompleted(domain.StepIAMOwner))
		assert.True(t, state.IsStepCompleted(domain.StepPubSubTopic))
		assert.True(t, state.IsStepCompleted(domain.StepServiceAccount))
		assert.True(t, state.IsStepCompleted(domain.StepPubSubPublish))
	})

	t.Run("skips already completed steps", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()

		createCalled := false
		mock := &gcp.MockClient{
			CreateProjectFunc: func(_ context.Context, _, _ string) error {
				createCalled = true
				return nil
			},
			BatchEnableAPIsFunc: func(_ context.Context, _ string, _ []string) error {
				return nil
			},
			GetIAMPolicyFunc: func(_ context.Context, _ string) (*ports.IAMPolicy, error) {
				return &ports.IAMPolicy{}, nil
			},
			SetIAMPolicyFunc: func(_ context.Context, _ string, _ *ports.IAMPolicy) error {
				return nil
			},
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID:         "my-project",
			IsNewProject:      true,
			Features:          []string{domain.FeatureEmail},
			SkipConfirmations: true,
		}

		state := &domain.SetupState{
			StartedAt:      time.Now(),
			CompletedSteps: []string{domain.StepCreateProject}, // Already completed
		}

		err := runPhase1(ctx, mock, cfg, state, dir, newMockReader("y"))
		require.NoError(t, err)
		assert.False(t, createCalled, "should not recreate already completed project")
	})

	t.Run("existing project skips creation", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()

		mock := &gcp.MockClient{
			BatchEnableAPIsFunc: func(_ context.Context, _ string, _ []string) error {
				return nil
			},
			GetIAMPolicyFunc: func(_ context.Context, _ string) (*ports.IAMPolicy, error) {
				return &ports.IAMPolicy{}, nil
			},
			SetIAMPolicyFunc: func(_ context.Context, _ string, _ *ports.IAMPolicy) error {
				return nil
			},
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID:         "existing-project",
			IsNewProject:      false,
			Features:          []string{domain.FeatureCalendar},
			SkipConfirmations: true,
		}

		state := &domain.SetupState{StartedAt: time.Now()}

		err := runPhase1(ctx, mock, cfg, state, dir, newMockReader("y"))
		require.NoError(t, err)
		assert.True(t, state.IsStepCompleted(domain.StepCreateProject))
	})
}

func TestRunPhase2(t *testing.T) {
	t.Run("collects OAuth credentials", func(t *testing.T) {
		dir := t.TempDir()
		bro := browser.NewMockBrowser()

		cfg := &domain.GoogleSetupConfig{
			ProjectID: "my-project",
			Region:    "us",
		}

		state := &domain.SetupState{StartedAt: time.Now()}

		// Simulate: press Enter after consent screen, then enter client ID/secret
		reader := newMockReader("", "my-client-id.apps.googleusercontent.com", "my-secret")

		err := runPhase2(bro, reader, cfg, state, dir)
		require.NoError(t, err)

		assert.True(t, bro.OpenCalled)
		assert.Equal(t, "my-client-id.apps.googleusercontent.com", cfg.ClientID)
		assert.Equal(t, "my-secret", cfg.ClientSecret)
		assert.True(t, state.IsStepCompleted(domain.StepConsentScreen))
		assert.True(t, state.IsStepCompleted(domain.StepCredentials))
	})

	t.Run("skips already completed steps", func(t *testing.T) {
		dir := t.TempDir()
		bro := browser.NewMockBrowser()

		cfg := &domain.GoogleSetupConfig{
			ProjectID:    "my-project",
			Region:       "us",
			ClientID:     "already-set",
			ClientSecret: "already-set",
		}

		state := &domain.SetupState{
			StartedAt: time.Now(),
			CompletedSteps: []string{
				domain.StepConsentScreen,
				domain.StepCredentials,
			},
		}

		err := runPhase2(bro, newMockReader(), cfg, state, dir)
		require.NoError(t, err)
		assert.False(t, bro.OpenCalled, "should not open browser for completed steps")
	})
}

func TestRunPhase3(t *testing.T) {
	t.Run("creates connector and validates", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()

		nylasClient := nylas.NewMockClient()

		cfg := &domain.GoogleSetupConfig{
			ProjectID:    "my-project",
			Region:       "us",
			Features:     []string{domain.FeatureEmail, domain.FeatureCalendar},
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		}

		state := &domain.SetupState{StartedAt: time.Now()}

		err := runPhase3(ctx, nylasClient, cfg, state, dir)
		require.NoError(t, err)
		assert.True(t, state.IsStepCompleted(domain.StepConnector))

		// Verify state file was cleaned up
		loaded, err := loadState(dir)
		assert.NoError(t, err)
		assert.Nil(t, loaded, "state file should be deleted on success")
	})

	t.Run("skips if connector already created", func(t *testing.T) {
		ctx := context.Background()
		dir := t.TempDir()

		nylasClient := nylas.NewMockClient()

		cfg := &domain.GoogleSetupConfig{
			ProjectID: "my-project",
			Features:  []string{domain.FeatureEmail},
		}

		state := &domain.SetupState{
			StartedAt:      time.Now(),
			CompletedSteps: []string{domain.StepConnector},
		}

		err := runPhase3(ctx, nylasClient, cfg, state, dir)
		require.NoError(t, err)
	})
}

func TestGuideBrowserSteps(t *testing.T) {
	t.Run("opens browser twice", func(t *testing.T) {
		bro := browser.NewMockBrowser()
		openCount := 0
		bro.OpenFunc = func(url string) error {
			openCount++
			return nil
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID: "my-project",
			Region:    "us",
		}

		// Press Enter for consent screen step
		reader := newMockReader("")

		err := guideBrowserSteps(bro, reader, cfg)
		require.NoError(t, err)
		assert.Equal(t, 2, openCount, "should open browser for consent screen and credentials")
	})

	t.Run("uses correct URLs", func(t *testing.T) {
		var urls []string
		bro := browser.NewMockBrowser()
		bro.OpenFunc = func(url string) error {
			urls = append(urls, url)
			return nil
		}

		cfg := &domain.GoogleSetupConfig{
			ProjectID: "test-proj",
			Region:    "eu",
		}

		reader := newMockReader("")

		err := guideBrowserSteps(bro, reader, cfg)
		require.NoError(t, err)
		assert.Len(t, urls, 2)
		assert.Contains(t, urls[0], "consent")
		assert.Contains(t, urls[0], "test-proj")
		assert.Contains(t, urls[1], "oauthclient")
		assert.Contains(t, urls[1], "test-proj")
	})
}

func TestCreateNylasConnector(t *testing.T) {
	t.Run("creates connector with correct scopes", func(t *testing.T) {
		ctx := context.Background()
		nylasClient := nylas.NewMockClient()

		cfg := &domain.GoogleSetupConfig{
			ProjectID:    "my-project",
			Region:       "us",
			Features:     []string{domain.FeatureEmail, domain.FeatureCalendar, domain.FeaturePubSub},
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		}

		connector, err := createNylasConnector(ctx, nylasClient, cfg)
		require.NoError(t, err)
		assert.NotNil(t, connector)
		assert.Equal(t, "Google", connector.Name)
		assert.Equal(t, "google", connector.Provider)
	})
}

func TestValidateSetup(t *testing.T) {
	t.Run("succeeds with mock", func(t *testing.T) {
		ctx := context.Background()
		nylasClient := nylas.NewMockClient()

		// Should not panic
		validateSetup(ctx, nylasClient)
	})
}

func TestPrintSummary(t *testing.T) {
	t.Run("with calendar feature", func(t *testing.T) {
		cfg := &domain.GoogleSetupConfig{
			Features: []string{domain.FeatureEmail, domain.FeatureCalendar},
		}
		// Should not panic
		printSummary(cfg)
	})

	t.Run("without calendar feature", func(t *testing.T) {
		cfg := &domain.GoogleSetupConfig{
			Features: []string{domain.FeatureEmail},
		}
		// Should not panic
		printSummary(cfg)
	})
}
