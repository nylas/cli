package agent

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAgentClient struct {
	listFn   func(ctx context.Context) ([]domain.AgentAccount, error)
	createFn func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error)
	updateFn func(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error)
	deleteFn func(ctx context.Context, grantID string) error
}

func (s stubAgentClient) ListAgentAccounts(ctx context.Context) ([]domain.AgentAccount, error) {
	if s.listFn == nil {
		return nil, nil
	}
	return s.listFn(ctx)
}

func (s stubAgentClient) GetAgentAccount(ctx context.Context, grantID string) (*domain.AgentAccount, error) {
	return nil, nil
}

func (s stubAgentClient) CreateAgentAccount(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
	if s.createFn == nil {
		return nil, nil
	}
	return s.createFn(ctx, email, appPassword, policyID)
}

func (s stubAgentClient) UpdateAgentAccount(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error) {
	if s.updateFn == nil {
		return nil, nil
	}
	return s.updateFn(ctx, grantID, email, appPassword)
}

func (s stubAgentClient) DeleteAgentAccount(ctx context.Context, grantID string) error {
	if s.deleteFn == nil {
		return nil
	}
	return s.deleteFn(ctx, grantID)
}

func TestCreateAgentAccountWithFallback_ReturnsRetryError(t *testing.T) {
	t.Helper()

	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "settings.app_password is an unknown field",
	}
	retryErr := errors.New("domain is not registered")
	createCalls := 0

	client := stubAgentClient{
		createFn: func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
			createCalls++
			switch createCalls {
			case 1:
				assert.Equal(t, "ValidAgentPass123ABC!", appPassword)
				return nil, initialErr
			case 2:
				assert.Empty(t, appPassword)
				return nil, retryErr
			default:
				t.Fatalf("unexpected CreateAgentAccount call %d", createCalls)
				return nil, nil
			}
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"ValidAgentPass123ABC!",
		"policy-123",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, retryErr)
	assert.ErrorContains(t, err, "retrying without app password")
	assert.NotErrorIs(t, err, initialErr)
	assert.Equal(t, 2, createCalls)
}

func TestCreateAgentAccountWithFallback_SkipsCleanupForExistingGrant(t *testing.T) {
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "settings.app_password is an unknown field",
	}
	updateErr := errors.New("grant update failed")
	deleteCalls := 0
	createCalls := 0

	client := stubAgentClient{
		listFn: func(ctx context.Context) ([]domain.AgentAccount, error) {
			return []domain.AgentAccount{{
				ID:       "agent-existing",
				Email:    "agent@example.com",
				Provider: domain.ProviderNylas,
				Settings: domain.AgentAccountSettings{PolicyID: "policy-123"},
			}}, nil
		},
		createFn: func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
			createCalls++
			if appPassword != "" {
				return nil, initialErr
			}
			t.Fatalf("unexpected create retry for existing grant")
			return nil, nil
		},
		updateFn: func(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error) {
			assert.Equal(t, "agent-existing", grantID)
			return nil, updateErr
		},
		deleteFn: func(ctx context.Context, grantID string) error {
			deleteCalls++
			return nil
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"ValidAgentPass123ABC!",
		"policy-123",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, updateErr)
	assert.ErrorContains(t, err, "failed to set app password on existing agent account")
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 0, deleteCalls)
}

func TestCreateAgentAccountWithFallback_UpdatesExistingGrantWithoutRetryCreate(t *testing.T) {
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "settings.app_password is an unknown field",
	}
	createCalls := 0
	updateCalls := 0

	client := stubAgentClient{
		listFn: func(ctx context.Context) ([]domain.AgentAccount, error) {
			return []domain.AgentAccount{{
				ID:       "agent-existing",
				Email:    "agent@example.com",
				Provider: domain.ProviderNylas,
				Settings: domain.AgentAccountSettings{PolicyID: "policy-123"},
			}}, nil
		},
		createFn: func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
			createCalls++
			assert.Equal(t, "ValidAgentPass123ABC!", appPassword)
			return nil, initialErr
		},
		updateFn: func(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error) {
			updateCalls++
			assert.Equal(t, "agent-existing", grantID)
			assert.Equal(t, "agent@example.com", email)
			assert.Equal(t, "ValidAgentPass123ABC!", appPassword)
			return &domain.AgentAccount{
				ID:       grantID,
				Email:    email,
				Provider: domain.ProviderNylas,
			}, nil
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"ValidAgentPass123ABC!",
		"policy-123",
	)

	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, "agent-existing", account.ID)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 1, updateCalls)
}

func TestCreateAgentAccountWithFallback_RejectsExistingGrantWithoutRequestedPolicy(t *testing.T) {
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "settings.app_password is an unknown field",
	}
	createCalls := 0
	updateCalls := 0

	client := stubAgentClient{
		listFn: func(ctx context.Context) ([]domain.AgentAccount, error) {
			return []domain.AgentAccount{{
				ID:       "agent-existing",
				Email:    "agent@example.com",
				Provider: domain.ProviderNylas,
			}}, nil
		},
		createFn: func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
			createCalls++
			return nil, initialErr
		},
		updateFn: func(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error) {
			updateCalls++
			return nil, nil
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"ValidAgentPass123ABC!",
		"policy-123",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorContains(t, err, "existing agent account is not attached to the requested policy")
	var cliErr *common.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Contains(t, cliErr.Suggestion, "create fallback cannot attach it to policy policy-123")
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 0, updateCalls)
}

func TestCreateAgentAccountWithFallback_RejectsExistingGrantOnDifferentPolicy(t *testing.T) {
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "settings.app_password is an unknown field",
	}
	createCalls := 0
	updateCalls := 0

	client := stubAgentClient{
		listFn: func(ctx context.Context) ([]domain.AgentAccount, error) {
			return []domain.AgentAccount{{
				ID:       "agent-existing",
				Email:    "agent@example.com",
				Provider: domain.ProviderNylas,
				Settings: domain.AgentAccountSettings{PolicyID: "policy-other"},
			}}, nil
		},
		createFn: func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
			createCalls++
			return nil, initialErr
		},
		updateFn: func(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error) {
			updateCalls++
			return nil, nil
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"ValidAgentPass123ABC!",
		"policy-123",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorContains(t, err, "existing agent account is attached to a different policy")
	var cliErr *common.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Contains(t, cliErr.Suggestion, "policy-other")
	assert.Contains(t, cliErr.Suggestion, "policy-123")
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 0, updateCalls)
}

func TestCreateAgentAccountWithFallback_PreservesNewGrantOnUpdateFailure(t *testing.T) {
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "settings.app_password is an unknown field",
	}
	updateErr := errors.New("grant update failed")
	deleteCalls := 0

	client := stubAgentClient{
		listFn: func(ctx context.Context) ([]domain.AgentAccount, error) {
			return nil, nil
		},
		createFn: func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
			if appPassword != "" {
				return nil, initialErr
			}
			return &domain.AgentAccount{
				ID:       "agent-new",
				Email:    email,
				Provider: domain.ProviderNylas,
			}, nil
		},
		updateFn: func(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error) {
			return nil, updateErr
		},
		deleteFn: func(ctx context.Context, grantID string) error {
			deleteCalls++
			return nil
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"ValidAgentPass123ABC!",
		"policy-123",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, updateErr)
	assert.ErrorContains(t, err, "created agent account agent-new but failed to set app password")
	assert.ErrorContains(t, err, "nylas agent account update agent-new --app-password <password>")
	assert.Equal(t, 0, deleteCalls)
}

func TestCreateAgentAccountWithFallback_DoesNotInventPolicyID(t *testing.T) {
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "extra fields not permitted: app_password",
	}

	client := stubAgentClient{
		createFn: func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
			if appPassword != "" {
				return nil, initialErr
			}
			return &domain.AgentAccount{
				ID:       "agent-new",
				Email:    email,
				Provider: domain.ProviderNylas,
			}, nil
		},
		updateFn: func(ctx context.Context, grantID, email, appPassword string) (*domain.AgentAccount, error) {
			return &domain.AgentAccount{
				ID:       grantID,
				Email:    email,
				Provider: domain.ProviderNylas,
			}, nil
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"ValidAgentPass123ABC!",
		"policy-123",
	)

	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, "", account.Settings.PolicyID)
}

func TestCreateAgentAccountWithFallback_DoesNotRetryInvalidPasswordValue(t *testing.T) {
	createCalls := 0
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "invalid app_password length",
	}

	client := stubAgentClient{
		createFn: func(ctx context.Context, email, appPassword, policyID string) (*domain.AgentAccount, error) {
			createCalls++
			return nil, initialErr
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"ValidAgentPass123ABC!",
		"policy-123",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, initialErr)
	assert.Equal(t, 1, createCalls)
}
