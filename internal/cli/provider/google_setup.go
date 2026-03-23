package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

// runGoogleSetup orchestrates the full Google provider setup wizard.
func runGoogleSetup(ctx context.Context, gcpClient ports.GCPClient, nylasClient ports.NylasClient, opts *googleSetupOpts) error {
	configDir := config.DefaultConfigDir()
	reader := newStdinReader()
	bro := browser.NewDefaultBrowser()

	// Check for saved state
	var state *domain.SetupState
	cfg := &domain.GoogleSetupConfig{
		SkipConfirmations: opts.yes,
	}

	if !opts.fresh {
		savedState, err := loadState(configDir)
		if err != nil {
			common.PrintWarning("Could not load saved state: %v", err)
		}
		if savedState != nil {
			resume, err := promptResume(reader, savedState)
			if err != nil {
				return err
			}
			if resume {
				state = savedState
				cfg.ProjectID = state.ProjectID
				cfg.DisplayName = state.DisplayName
				cfg.Region = state.Region
				cfg.Features = state.Features
				cfg.IsNewProject = state.IsNewProject
			}
		}
	}

	if state == nil {
		state = &domain.SetupState{
			StartedAt: time.Now(),
		}
	}

	// Phase 0: Prerequisites
	fmt.Println("\n  Checking prerequisites...")
	_, err := checkPrerequisites(ctx, gcpClient)
	if err != nil {
		return err
	}

	// Phase 0: Gather config (skip if resuming)
	if cfg.ProjectID == "" {
		if err := gatherConfig(ctx, gcpClient, reader, cfg, opts); err != nil {
			return err
		}
		state.ProjectID = cfg.ProjectID
		state.DisplayName = cfg.DisplayName
		state.Region = cfg.Region
		state.Features = cfg.Features
		state.IsNewProject = cfg.IsNewProject
		_ = saveState(configDir, state)
	}

	// Phase 1: Automated GCP setup
	fmt.Printf("\n━━━ Phase 1: Automated GCP setup ━━━━━━━━━━━━━━━━━━━━\n\n")

	if err := runPhase1(ctx, gcpClient, cfg, state, configDir, reader); err != nil {
		return err
	}

	// Phase 2: Browser configuration
	fmt.Printf("\n━━━ Phase 2: Browser configuration ━━━━━━━━━━━━━━━━━━\n")

	if err := runPhase2(bro, reader, cfg, state, configDir); err != nil {
		return err
	}

	// Phase 3: Connect to Nylas
	fmt.Printf("\n━━━ Phase 3: Connecting to Nylas ━━━━━━━━━━━━━━━━━━━━\n\n")

	if err := runPhase3(ctx, nylasClient, cfg, state, configDir); err != nil {
		return err
	}

	return nil
}

// gatherConfig collects project, features, and region from flags or interactive prompts.
func gatherConfig(ctx context.Context, gcpClient ports.GCPClient, reader lineReader, cfg *domain.GoogleSetupConfig, opts *googleSetupOpts) error {
	// Project selection
	projectID, displayName, isNew, err := promptProjectSelection(ctx, gcpClient, reader, opts.projectID)
	if err != nil {
		return err
	}
	cfg.ProjectID = projectID
	cfg.DisplayName = displayName
	cfg.IsNewProject = isNew

	// Feature selection from flags or interactive
	if opts.hasFeatureFlags() {
		cfg.Features = opts.selectedFeatures()
	} else {
		features, err := promptFeatureSelection(reader)
		if err != nil {
			return err
		}
		cfg.Features = features
	}

	// Region from flag or interactive
	if opts.region != "" {
		cfg.Region = opts.region
	} else {
		region, err := promptRegion(reader)
		if err != nil {
			return err
		}
		cfg.Region = region
	}

	return nil
}

func runPhase1(ctx context.Context, gcpClient ports.GCPClient, cfg *domain.GoogleSetupConfig, state *domain.SetupState, configDir string, reader lineReader) error {
	// Create project
	if !state.IsStepCompleted(domain.StepCreateProject) {
		state.PendingStep = domain.StepCreateProject
		_ = saveState(configDir, state)

		if cfg.IsNewProject {
			if err := createGCPProject(ctx, gcpClient, cfg); err != nil {
				return err
			}
		}
		state.CompleteStep(domain.StepCreateProject)
		_ = saveState(configDir, state)
	}

	// Enable APIs
	if !state.IsStepCompleted(domain.StepEnableAPIs) {
		state.PendingStep = domain.StepEnableAPIs
		_ = saveState(configDir, state)

		if err := enableAPIs(ctx, gcpClient, cfg); err != nil {
			return err
		}
		state.CompleteStep(domain.StepEnableAPIs)
		_ = saveState(configDir, state)
	}

	// Add IAM owner
	if !state.IsStepCompleted(domain.StepIAMOwner) {
		state.PendingStep = domain.StepIAMOwner
		_ = saveState(configDir, state)

		if err := addIAMOwner(ctx, gcpClient, cfg, reader); err != nil {
			return err
		}
		state.CompleteStep(domain.StepIAMOwner)
		_ = saveState(configDir, state)
	}

	// Setup Pub/Sub (topic + service account + publisher role)
	if err := setupPubSub(ctx, gcpClient, cfg, state, configDir); err != nil {
		return err
	}

	return nil
}

func runPhase2(bro ports.Browser, reader lineReader, cfg *domain.GoogleSetupConfig, state *domain.SetupState, configDir string) error {
	if !state.IsStepCompleted(domain.StepConsentScreen) {
		state.PendingStep = domain.StepConsentScreen
		_ = saveState(configDir, state)

		if err := guideBrowserSteps(bro, reader, cfg); err != nil {
			return err
		}
		state.CompleteStep(domain.StepConsentScreen)
		_ = saveState(configDir, state)
	}

	if !state.IsStepCompleted(domain.StepCredentials) || cfg.ClientID == "" || cfg.ClientSecret == "" {
		state.PendingStep = domain.StepCredentials
		_ = saveState(configDir, state)

		clientID, clientSecret, err := promptOAuthCredentials(reader)
		if err != nil {
			return err
		}
		cfg.ClientID = clientID
		cfg.ClientSecret = clientSecret

		state.CompleteStep(domain.StepCredentials)
		_ = saveState(configDir, state)
	}

	return nil
}

func runPhase3(ctx context.Context, nylasClient ports.NylasClient, cfg *domain.GoogleSetupConfig, state *domain.SetupState, configDir string) error {
	if !state.IsStepCompleted(domain.StepConnector) {
		state.PendingStep = domain.StepConnector
		_ = saveState(configDir, state)

		connector, err := createNylasConnector(ctx, nylasClient, cfg)
		if err != nil {
			return err
		}

		validateSetup(ctx, nylasClient, connector.ID)

		state.CompleteStep(domain.StepConnector)
	}

	// Clean up state file on success
	_ = clearState(configDir)

	printSummary(cfg)
	return nil
}
