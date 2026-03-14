package provider

import (
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadState(t *testing.T) {
	dir := t.TempDir()

	state := &domain.SetupState{
		ProjectID:      "test-project",
		Region:         "us",
		Features:       []string{"email", "calendar"},
		CompletedSteps: []string{"create_project"},
		PendingStep:    "enable_apis",
		StartedAt:      time.Now(),
	}

	err := saveState(dir, state)
	require.NoError(t, err)

	loaded, err := loadState(dir)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, "test-project", loaded.ProjectID)
	assert.Equal(t, "us", loaded.Region)
	assert.Equal(t, []string{"email", "calendar"}, loaded.Features)
	assert.Equal(t, []string{"create_project"}, loaded.CompletedSteps)
	assert.Equal(t, "enable_apis", loaded.PendingStep)
}

func TestLoadState_NotExists(t *testing.T) {
	dir := t.TempDir()
	state, err := loadState(dir)
	assert.NoError(t, err)
	assert.Nil(t, state)
}

func TestLoadState_Expired(t *testing.T) {
	dir := t.TempDir()

	state := &domain.SetupState{
		ProjectID: "test-project",
		Region:    "us",
		StartedAt: time.Now().Add(-25 * time.Hour),
	}

	err := saveState(dir, state)
	require.NoError(t, err)

	loaded, err := loadState(dir)
	assert.NoError(t, err)
	assert.Nil(t, loaded, "expired state should return nil")
}

func TestClearState(t *testing.T) {
	dir := t.TempDir()

	state := &domain.SetupState{
		ProjectID: "test-project",
		StartedAt: time.Now(),
	}

	err := saveState(dir, state)
	require.NoError(t, err)

	err = clearState(dir)
	assert.NoError(t, err)

	loaded, err := loadState(dir)
	assert.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestClearState_NotExists(t *testing.T) {
	dir := t.TempDir()
	err := clearState(dir)
	assert.NoError(t, err)
}
