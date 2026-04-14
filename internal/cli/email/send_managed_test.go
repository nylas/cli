package email

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessageForGrant_UsesTransactionalSendForManagedProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider domain.Provider
		email    string
	}{
		{
			name:     "inbox grant uses transactional send",
			provider: domain.ProviderInbox,
			email:    "info@qasim.nylas.email",
		},
		{
			name:     "nylas grant uses transactional send",
			provider: domain.ProviderNylas,
			email:    "xyz@qasim.nylas.email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := nylas.NewMockClient()
			client.SendMessageFunc = func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
				t.Fatalf("SendMessage should not be called for managed provider %s", tt.provider)
				return nil, nil
			}

			var gotDomain string
			var gotFrom []domain.EmailParticipant
			nylas.SendTransactionalMessageFunc = func(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
				gotDomain = domainName
				gotFrom = append([]domain.EmailParticipant(nil), req.From...)
				return &domain.Message{ID: "txn-msg-id", Subject: req.Subject}, nil
			}
			t.Cleanup(func() {
				nylas.SendTransactionalMessageFunc = nil
			})

			req := &domain.SendMessageRequest{
				Subject: "Hello",
				Body:    "Body",
				To:      []domain.EmailParticipant{{Email: "to@example.com"}},
			}
			grant := &domain.Grant{
				ID:       "grant-123",
				Provider: tt.provider,
				Email:    tt.email,
			}

			msg, err := sendMessageForGrant(context.Background(), client, "grant-123", grant, req)

			require.NoError(t, err)
			require.NotNil(t, msg)
			assert.Equal(t, "txn-msg-id", msg.ID)
			assert.False(t, client.SendMessageCalled)
			assert.Equal(t, "qasim.nylas.email", gotDomain)
			require.Len(t, gotFrom, 1)
			assert.Equal(t, tt.email, gotFrom[0].Email)
			require.Len(t, req.From, 1)
			assert.Equal(t, tt.email, req.From[0].Email)
		})
	}
}

func TestSendMessageForGrant_UsesGrantSendForStandardProviders(t *testing.T) {
	client := nylas.NewMockClient()

	var gotGrantID string
	var gotFrom []domain.EmailParticipant
	client.SendMessageFunc = func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
		gotGrantID = grantID
		gotFrom = append([]domain.EmailParticipant(nil), req.From...)
		return &domain.Message{ID: "grant-msg-id", Subject: req.Subject}, nil
	}

	nylas.SendTransactionalMessageFunc = func(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
		t.Fatalf("SendTransactionalMessage should not be called for standard providers")
		return nil, nil
	}
	t.Cleanup(func() {
		nylas.SendTransactionalMessageFunc = nil
	})

	req := &domain.SendMessageRequest{
		Subject: "Hello",
		Body:    "Body",
		To:      []domain.EmailParticipant{{Email: "to@example.com"}},
	}
	grant := &domain.Grant{
		ID:       "grant-456",
		Provider: domain.ProviderGoogle,
		Email:    "user@example.com",
	}

	msg, err := sendMessageForGrant(context.Background(), client, "grant-456", grant, req)

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, "grant-msg-id", msg.ID)
	assert.Equal(t, "grant-456", gotGrantID)
	assert.Empty(t, gotFrom)
	assert.Empty(t, req.From)
}

func TestValidateManagedSecureSendSupport(t *testing.T) {
	tests := []struct {
		name      string
		sign      bool
		encrypt   bool
		grant     *domain.Grant
		wantError bool
	}{
		{
			name:      "managed nylas sign is rejected",
			sign:      true,
			grant:     &domain.Grant{Provider: domain.ProviderNylas},
			wantError: true,
		},
		{
			name:      "managed inbox encrypt is rejected",
			encrypt:   true,
			grant:     &domain.Grant{Provider: domain.ProviderInbox},
			wantError: true,
		},
		{
			name:      "standard provider sign is allowed",
			sign:      true,
			grant:     &domain.Grant{Provider: domain.ProviderGoogle},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateManagedSecureSendSupport(tt.sign, tt.encrypt, tt.grant)
			if tt.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetGrantForSend_PropagatesLookupFailure(t *testing.T) {
	client := nylas.NewMockClient()
	client.GetGrantFunc = func(ctx context.Context, grantID string) (*domain.Grant, error) {
		return nil, errors.New("temporary lookup failure")
	}

	grant, err := getGrantForSend(context.Background(), client, "grant-123")

	require.Error(t, err)
	assert.Nil(t, grant)
	assert.ErrorContains(t, err, "failed to get grant")
	assert.ErrorContains(t, err, "temporary lookup failure")
}
