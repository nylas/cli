package dashboard

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func reloginTo(orgPublicID string, calls *int) func(context.Context) (*domain.DashboardOAuthExchangeResponse, error) {
	return func(context.Context) (*domain.DashboardOAuthExchangeResponse, error) {
		*calls++
		return &domain.DashboardOAuthExchangeResponse{OrgPublicID: orgPublicID}, nil
	}
}

func TestSwitchOAuthSessionOrg_SignsInAgainAndReportsTheNewOrg(t *testing.T) {
	var out bytes.Buffer
	calls := 0

	err := switchOAuthSessionOrg(context.Background(), &out, "", reloginTo("org_2", &calls))

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
	assert.Contains(t, out.String(), "choose the organization there")
	assert.Contains(t, out.String(), "org_2")
}

func TestSwitchOAuthSessionOrg_FailsWhenAnotherOrgWasChosenThanTheOneNamed(t *testing.T) {
	calls := 0

	err := switchOAuthSessionOrg(context.Background(), &bytes.Buffer{}, "org_2", reloginTo("org_1", &calls))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "signed in to organization org_1, not org_2")
}

func TestSwitchOAuthSessionOrg_AcceptsTheOrgItWasToldToExpect(t *testing.T) {
	calls := 0

	err := switchOAuthSessionOrg(context.Background(), &bytes.Buffer{}, "org_2", reloginTo("org_2", &calls))

	require.NoError(t, err)
}

func TestSwitchOAuthSessionOrg_ReportsAFailedSignIn(t *testing.T) {
	failed := func(context.Context) (*domain.DashboardOAuthExchangeResponse, error) {
		return nil, errors.New("consent was denied")
	}

	err := switchOAuthSessionOrg(context.Background(), &bytes.Buffer{}, "", failed)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "consent was denied")
}

func TestSwitchOAuthSessionOrg_WithoutOAuthInTheBuildSaysHowToSwitch(t *testing.T) {
	err := switchOAuthSessionOrg(context.Background(), &bytes.Buffer{}, "", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nylas oauth login")
}
