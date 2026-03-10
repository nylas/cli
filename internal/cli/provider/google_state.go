package provider

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nylas/cli/internal/domain"
)

const stateFileName = "provider-setup-state.json"

// loadState loads the setup state from the config directory.
func loadState(configDir string) (*domain.SetupState, error) {
	path := filepath.Join(configDir, stateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state domain.SetupState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	if state.IsExpired() {
		_ = clearState(configDir)
		return nil, nil
	}

	return &state, nil
}

// saveState saves the setup state to the config directory.
func saveState(configDir string, state *domain.SetupState) error {
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	path := filepath.Join(configDir, stateFileName)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	return nil
}

// clearState removes the setup state file.
func clearState(configDir string) error {
	path := filepath.Join(configDir, stateFileName)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// promptResume asks the user whether to resume from saved state.
func promptResume(reader lineReader, state *domain.SetupState) (bool, error) {
	pendingDesc := state.PendingStep
	if pendingDesc == "" && len(state.CompletedSteps) > 0 {
		pendingDesc = "next step"
	}

	fmt.Printf("\n  Previous setup detected for project '%s'.\n", state.ProjectID)
	fmt.Printf("  Completed %d steps. Resume from [%s]? (Y/n): ", len(state.CompletedSteps), pendingDesc)

	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input = trimInput(input)
	return input == "" || input == "y" || input == "yes", nil
}
