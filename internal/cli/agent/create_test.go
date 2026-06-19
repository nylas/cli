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
	createFn func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error)
	updateFn func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error)
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

func (s stubAgentClient) CreateAgentAccount(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
	if s.createFn == nil {
		return nil, nil
	}
	return s.createFn(ctx, email, name, appPassword, workspaceID)
}

func (s stubAgentClient) UpdateAgentAccount(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
	if s.updateFn == nil {
		return nil, nil
	}
	return s.updateFn(ctx, grantID, email, name, appPassword)
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
	retryErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "domain is not registered",
		RequestID:  "req-123",
	}
	createCalls := 0

	client := stubAgentClient{
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
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
		"",
		"ValidAgentPass123ABC!",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, retryErr)
	assert.ErrorContains(t, err, "retrying without app password")
	assert.ErrorContains(t, err, "Cannot create agent account because domain")
	assert.NotErrorIs(t, err, initialErr)
	assert.Equal(t, 2, createCalls)

	var cliErr *common.CLIError
	require.ErrorAs(t, err, &cliErr)
	assert.Equal(t, "req-123", cliErr.RequestID)
	assert.Equal(t, `Cannot create agent account because domain "example.com" is not registered`, cliErr.Message)
	assert.Contains(t, cliErr.Suggestions, `Create or register "example.com" as an agent domain in the Nylas Dashboard: `+agentDomainDashboardURL)
	assert.Contains(t, cliErr.Suggestions, "Or use an email address on an agent domain already registered in the Dashboard")
	assert.Contains(t, cliErr.Suggestions, "After registering the domain, retry: nylas agent account create agent@example.com")
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
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			createCalls++
			if appPassword != "" {
				return nil, initialErr
			}
			t.Fatalf("unexpected create retry for existing grant")
			return nil, nil
		},
		updateFn: func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
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
		"",
		"ValidAgentPass123ABC!",
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
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			createCalls++
			assert.Equal(t, "ValidAgentPass123ABC!", appPassword)
			return nil, initialErr
		},
		updateFn: func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
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
		"",
		"ValidAgentPass123ABC!",
	)

	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, "agent-existing", account.ID)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 1, updateCalls)
}

func TestCreateAgentAccountWithFallback_UpdatesExistingGrantWithoutCheckingPolicy(t *testing.T) {
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
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			createCalls++
			return nil, initialErr
		},
		updateFn: func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
			updateCalls++
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
		"",
		"ValidAgentPass123ABC!",
	)

	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, "agent-existing", account.ID)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 1, updateCalls)
}

func TestCreateAgentAccountWithFallback_UpdatesExistingGrantOnDifferentPolicy(t *testing.T) {
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
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			createCalls++
			return nil, initialErr
		},
		updateFn: func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
			updateCalls++
			return &domain.AgentAccount{
				ID:       grantID,
				Email:    email,
				Provider: domain.ProviderNylas,
				Settings: domain.AgentAccountSettings{PolicyID: "policy-other"},
			}, nil
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"",
		"ValidAgentPass123ABC!",
	)

	require.NoError(t, err)
	require.NotNil(t, account)
	assert.Equal(t, "agent-existing", account.ID)
	assert.Equal(t, 1, createCalls)
	assert.Equal(t, 1, updateCalls)
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
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			if appPassword != "" {
				return nil, initialErr
			}
			return &domain.AgentAccount{
				ID:       "agent-new",
				Email:    email,
				Provider: domain.ProviderNylas,
			}, nil
		},
		updateFn: func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
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
		"",
		"ValidAgentPass123ABC!",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, updateErr)
	assert.ErrorContains(t, err, "created agent account agent-new but failed to set app password")
	assert.ErrorContains(t, err, "nylas agent account update agent-new --app-password <password>")
	assert.Equal(t, 0, deleteCalls)
}

func TestCreateAgentAccountWithFallback_DoesNotInventWorkspaceID(t *testing.T) {
	initialErr := &domain.APIError{
		StatusCode: http.StatusBadRequest,
		Message:    "extra fields not permitted: app_password",
	}

	client := stubAgentClient{
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			if appPassword != "" {
				return nil, initialErr
			}
			return &domain.AgentAccount{
				ID:       "agent-new",
				Email:    email,
				Provider: domain.ProviderNylas,
			}, nil
		},
		updateFn: func(ctx context.Context, grantID, email, name, appPassword string) (*domain.AgentAccount, error) {
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
		"",
		"ValidAgentPass123ABC!",
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
		createFn: func(ctx context.Context, email, name, appPassword, workspaceID string) (*domain.AgentAccount, error) {
			createCalls++
			return nil, initialErr
		},
	}

	account, err := createAgentAccountWithFallback(
		context.Background(),
		client,
		"agent@example.com",
		"",
		"ValidAgentPass123ABC!",
	)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.ErrorIs(t, err, initialErr)
	assert.Equal(t, 1, createCalls)
}

func TestNormalizeAgentAccountEmail(t *testing.T) {
	assert.Equal(t, "agent@nylas.email", normalizeAgentAccountEmail(" agent "))
	assert.Equal(t, "agent@example.com", normalizeAgentAccountEmail(" agent@example.com "))
	assert.Equal(t, "", normalizeAgentAccountEmail(" "))
}

func TestWrapAgentAccountCreateError_DomainFailures(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		email             string
		wantMessage       string
		wantSuggestion    string
		wantRetry         string
		wantOriginalError bool
	}{
		{
			name: "missing domain",
			err: &domain.APIError{
				StatusCode: http.StatusBadRequest,
				Message:    "Domain doesn't exist",
				RequestID:  "req-domain",
			},
			email:          "support@example.com",
			wantMessage:    `Cannot create agent account because domain "example.com" is not registered`,
			wantSuggestion: `Create or register "example.com" as an agent domain in the Nylas Dashboard: ` + agentDomainDashboardURL,
			wantRetry:      "After registering the domain, retry: nylas agent account create support@example.com",
		},
		{
			name: "live api missing domain wording",
			err: &domain.APIError{
				StatusCode: http.StatusNotFound,
				Type:       "api.not_found_error",
				Message:    "Provisioning the inbox failed: Domain not found",
				RequestID:  "req-domain",
			},
			email:          "agent@missing.nylas.email",
			wantMessage:    `Cannot create agent account because domain "missing.nylas.email" is not registered`,
			wantSuggestion: `Create or register "missing.nylas.email" as an agent domain in the Nylas Dashboard: ` + agentDomainDashboardURL,
			wantRetry:      "After registering the domain, retry: nylas agent account create agent@missing.nylas.email",
		},
		{
			name: "domain limit",
			err: &domain.APIError{
				StatusCode: http.StatusUnprocessableEntity,
				Message:    "maximum number of domains reached",
			},
			email:          "support@example.com",
			wantMessage:    "Maximum number of agent account domains reached",
			wantSuggestion: `Create or register "example.com" as an agent domain in the Nylas Dashboard: ` + agentDomainDashboardURL,
			wantRetry:      "Remove an unused domain or use an email address on one of your existing agent domains",
		},
		{
			name: "unrelated api error",
			err: &domain.APIError{
				StatusCode: http.StatusBadRequest,
				Message:    "invalid email address",
			},
			email:             "support@example.com",
			wantOriginalError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := wrapAgentAccountCreateError(tt.email, tt.err)
			require.Error(t, err)
			if tt.wantOriginalError {
				assert.Same(t, tt.err, err)
				return
			}

			var cliErr *common.CLIError
			require.ErrorAs(t, err, &cliErr)
			assert.Equal(t, tt.wantMessage, cliErr.Message)
			assert.Contains(t, cliErr.Suggestions, tt.wantSuggestion)
			assert.Contains(t, cliErr.Suggestions, tt.wantRetry)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}
