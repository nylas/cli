package otp

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
)

func TestNewService(t *testing.T) {
	client := nylas.NewMockClient()
	grantStore := &mockGrantStore{}
	configStore := &mockConfigStore{}

	service := NewService(client, grantStore, configStore)

	if service == nil {
		t.Error("NewService() returned nil")
		return
	}
	if service.client == nil {
		t.Error("client is nil")
	}
	if service.grantStore == nil {
		t.Error("grantStore is nil")
	}
	if service.config == nil {
		t.Error("config is nil")
	}
}

func TestService_GetOTP(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		setupMocks  func(*nylas.MockClient, *mockGrantStore)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful OTP retrieval",
			email: "test@example.com",
			setupMocks: func(client *nylas.MockClient, grants *mockGrantStore) {
				grants.grants = []domain.GrantInfo{
					{ID: "grant-1", Email: "test@example.com"},
				}
				client.GetMessagesFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
					return []domain.Message{
						{
							ID:      "msg-1",
							Subject: "Your verification code is 123456",
							Body:    "Your code: 123456",
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name:  "grant not found",
			email: "unknown@example.com",
			setupMocks: func(client *nylas.MockClient, grants *mockGrantStore) {
				grants.grants = []domain.GrantInfo{}
				grants.getByEmailErr = domain.ErrGrantNotFound
			},
			wantErr:     true,
			errContains: "grant not found",
		},
		{
			name:  "no messages",
			email: "test@example.com",
			setupMocks: func(client *nylas.MockClient, grants *mockGrantStore) {
				grants.grants = []domain.GrantInfo{
					{ID: "grant-1", Email: "test@example.com"},
				}
				client.GetMessagesFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
					return []domain.Message{}, nil
				}
			},
			wantErr:     true,
			errContains: "no messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			grantStore := &mockGrantStore{}
			configStore := &mockConfigStore{}

			tt.setupMocks(client, grantStore)

			service := NewService(client, grantStore, configStore)
			ctx := context.Background()

			result, err := service.GetOTP(ctx, tt.email)

			if tt.wantErr {
				if err == nil {
					t.Error("GetOTP() error = nil, want error")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetOTP() error = %v, want nil", err)
				}
				if result == nil {
					t.Error("result is nil")
				}
			}
		})
	}
}

func TestService_GetOTPByGrantID(t *testing.T) {
	tests := []struct {
		name        string
		grantID     string
		messages    []domain.Message
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful OTP extraction",
			grantID: "grant-1",
			messages: []domain.Message{
				{
					ID:      "msg-1",
					Subject: "Your verification code",
					Body:    "Your OTP is: 987654",
				},
			},
			wantErr: false,
		},
		{
			name:        "no messages",
			grantID:     "grant-1",
			messages:    []domain.Message{},
			wantErr:     true,
			errContains: "no messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			client.GetMessagesFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
				return tt.messages, nil
			}

			grantStore := &mockGrantStore{}
			configStore := &mockConfigStore{}

			service := NewService(client, grantStore, configStore)
			ctx := context.Background()

			result, err := service.GetOTPByGrantID(ctx, tt.grantID)

			if tt.wantErr {
				if err == nil {
					t.Error("GetOTPByGrantID() error = nil, want error")
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, should contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetOTPByGrantID() error = %v, want nil", err)
				}
				if result == nil {
					t.Error("result is nil")
				}
			}
		})
	}
}

func TestService_GetOTPDefault(t *testing.T) {
	tests := []struct {
		name           string
		defaultGrantID string
		getDefaultErr  error
		messages       []domain.Message
		wantErr        bool
	}{
		{
			name:           "successful default OTP",
			defaultGrantID: "grant-1",
			getDefaultErr:  nil,
			messages: []domain.Message{
				{ID: "msg-1", Body: "Code: 111222"},
			},
			wantErr: false,
		},
		{
			name:           "no default grant",
			defaultGrantID: "",
			getDefaultErr:  domain.ErrNoDefaultGrant,
			messages:       []domain.Message{},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			client.GetMessagesFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
				return tt.messages, nil
			}

			grantStore := &mockGrantStore{
				defaultGrant:  tt.defaultGrantID,
				getDefaultErr: tt.getDefaultErr,
			}
			configStore := &mockConfigStore{}

			service := NewService(client, grantStore, configStore)
			ctx := context.Background()

			_, err := service.GetOTPDefault(ctx)

			if tt.wantErr && err == nil {
				t.Error("GetOTPDefault() error = nil, want error")
			} else if !tt.wantErr && err != nil {
				t.Errorf("GetOTPDefault() error = %v, want nil", err)
			}
		})
	}
}

func TestService_GetMessages(t *testing.T) {
	client := nylas.NewMockClient()
	client.GetMessagesFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
		return []domain.Message{
			{ID: "msg-1", Subject: "Test 1"},
			{ID: "msg-2", Subject: "Test 2"},
		}, nil
	}

	grantStore := &mockGrantStore{
		grants: []domain.GrantInfo{
			{ID: "grant-1", Email: "test@example.com"},
		},
	}
	configStore := &mockConfigStore{}

	service := NewService(client, grantStore, configStore)
	ctx := context.Background()

	messages, err := service.GetMessages(ctx, "test@example.com", 10)

	if err != nil {
		t.Errorf("GetMessages() error = %v, want nil", err)
	}
	if len(messages) != 2 {
		t.Errorf("len(messages) = %d, want 2", len(messages))
	}
}

func TestService_GetMessagesDefault(t *testing.T) {
	client := nylas.NewMockClient()
	client.GetMessagesFunc = func(ctx context.Context, grantID string, limit int) ([]domain.Message, error) {
		return []domain.Message{
			{ID: "msg-1", Subject: "Test"},
		}, nil
	}

	grantStore := &mockGrantStore{
		defaultGrant: "grant-1",
	}
	configStore := &mockConfigStore{}

	service := NewService(client, grantStore, configStore)
	ctx := context.Background()

	messages, err := service.GetMessagesDefault(ctx, 5)

	if err != nil {
		t.Errorf("GetMessagesDefault() error = %v, want nil", err)
	}
	if len(messages) != 1 {
		t.Errorf("len(messages) = %d, want 1", len(messages))
	}
}

func TestService_ListAccounts(t *testing.T) {
	grantStore := &mockGrantStore{
		grants: []domain.GrantInfo{
			{ID: "grant-1", Email: "user1@example.com"},
			{ID: "grant-2", Email: "user2@example.com"},
		},
	}

	client := nylas.NewMockClient()
	configStore := &mockConfigStore{}

	service := NewService(client, grantStore, configStore)

	accounts, err := service.ListAccounts()

	if err != nil {
		t.Errorf("ListAccounts() error = %v, want nil", err)
	}
	if len(accounts) != 2 {
		t.Errorf("len(accounts) = %d, want 2", len(accounts))
	}
}

// Mock implementations

type mockGrantStore struct {
	grants        []domain.GrantInfo
	defaultGrant  string
	getByEmailErr error
	getDefaultErr error
}

func (m *mockGrantStore) GetGrant(grantID string) (*domain.GrantInfo, error) {
	for _, grant := range m.grants {
		if grant.ID == grantID {
			return &grant, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

func (m *mockGrantStore) GetGrantByEmail(email string) (*domain.GrantInfo, error) {
	if m.getByEmailErr != nil {
		return nil, m.getByEmailErr
	}
	for _, grant := range m.grants {
		if grant.Email == email {
			return &grant, nil
		}
	}
	return nil, domain.ErrGrantNotFound
}

func (m *mockGrantStore) GetDefaultGrant() (string, error) {
	if m.getDefaultErr != nil {
		return "", m.getDefaultErr
	}
	if m.defaultGrant == "" {
		return "", domain.ErrNoDefaultGrant
	}
	return m.defaultGrant, nil
}

func (m *mockGrantStore) ListGrants() ([]domain.GrantInfo, error) {
	return m.grants, nil
}

func (m *mockGrantStore) SaveGrant(grant domain.GrantInfo) error {
	return nil
}

func (m *mockGrantStore) ReplaceGrants(grants []domain.GrantInfo) error {
	m.grants = append([]domain.GrantInfo(nil), grants...)
	return nil
}

func (m *mockGrantStore) DeleteGrant(grantID string) error {
	return nil
}

func (m *mockGrantStore) SetDefaultGrant(grantID string) error {
	m.defaultGrant = grantID
	return nil
}

func (m *mockGrantStore) ClearGrants() error {
	m.grants = []domain.GrantInfo{}
	return nil
}

type mockConfigStore struct {
	config *domain.Config
	err    error
}

func (m *mockConfigStore) Load() (*domain.Config, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.config == nil {
		return domain.DefaultConfig(), nil
	}
	return m.config, nil
}

func (m *mockConfigStore) Save(config *domain.Config) error {
	m.config = config
	return m.err
}

func (m *mockConfigStore) Path() string {
	return "/tmp/test-config.yaml"
}

func (m *mockConfigStore) Exists() bool {
	return m.config != nil
}

// Helper function
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
