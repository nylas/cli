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

func TestSendMessageForGrant_NylasGrantArchivesViaPerGrantSend(t *testing.T) {
	// For Nylas-managed grants, send must go through the per-grant endpoint
	// (the only one that archives to the Sent folder) with From auto-populated
	// from the grant. The domain-based transactional endpoint is a relay that
	// does not archive and must not be used here.
	client := nylas.NewMockClient()

	var gotGrantID string
	var gotFrom []domain.EmailParticipant
	client.SendMessageFunc = func(ctx context.Context, grantID string, req *domain.SendMessageRequest) (*domain.Message, error) {
		gotGrantID = grantID
		gotFrom = append([]domain.EmailParticipant(nil), req.From...)
		return &domain.Message{ID: "archived-msg-id", Subject: req.Subject}, nil
	}

	nylas.SendTransactionalMessageFunc = func(ctx context.Context, domainName string, req *domain.SendMessageRequest) (*domain.Message, error) {
		t.Fatalf("transactional endpoint must not be used; per-grant send is what archives to Sent")
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
		ID:       "grant-123",
		Provider: domain.ProviderNylas,
		Email:    "xyz@qasim.nylas.email",
	}

	msg, err := sendMessageForGrant(context.Background(), client, "grant-123", grant, req)

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, "archived-msg-id", msg.ID)
	assert.Equal(t, "grant-123", gotGrantID)
	require.Len(t, gotFrom, 1)
	assert.Equal(t, grant.Email, gotFrom[0].Email)
	require.Len(t, req.From, 1, "From must be populated for Nylas-managed grants since the API rejects empty From")
	assert.Equal(t, grant.Email, req.From[0].Email)
}

func TestSendMessageForGrant_NylasGrantPreservesCallerFrom(t *testing.T) {
	// If the caller already supplied a From, sendMessageForGrant must not overwrite it.
	client := nylas.NewMockClient()

	var gotFrom []domain.EmailParticipant
	client.SendMessageFunc = func(ctx context.Context, _ string, req *domain.SendMessageRequest) (*domain.Message, error) {
		gotFrom = append([]domain.EmailParticipant(nil), req.From...)
		return &domain.Message{ID: "msg"}, nil
	}

	explicit := []domain.EmailParticipant{{Email: "alias@qasim.nylas.email", Name: "Caller"}}
	req := &domain.SendMessageRequest{
		Subject: "Hello",
		Body:    "Body",
		To:      []domain.EmailParticipant{{Email: "to@example.com"}},
		From:    explicit,
	}
	grant := &domain.Grant{
		ID:       "grant-123",
		Provider: domain.ProviderNylas,
		Email:    "xyz@qasim.nylas.email",
	}

	_, err := sendMessageForGrant(context.Background(), client, "grant-123", grant, req)
	require.NoError(t, err)
	require.Len(t, gotFrom, 1)
	assert.Equal(t, "alias@qasim.nylas.email", gotFrom[0].Email,
		"caller-supplied From must be preserved (not rewritten to grant.Email)")
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
