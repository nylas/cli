package email

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/adapters/nylas"
	"github.com/nylas/cli/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSignaturesCmd(t *testing.T) {
	cmd := newSignaturesCmd()

	assert.Equal(t, "signatures", cmd.Use)

	subcommands := map[string]bool{}
	for _, sub := range cmd.Commands() {
		subcommands[sub.Name()] = true
	}

	for _, expected := range []string{"list", "show", "create", "update", "delete"} {
		assert.True(t, subcommands[expected], "missing signatures subcommand %q", expected)
	}
}

func TestSignaturesCommandFlags(t *testing.T) {
	createCmd := newSignaturesCreateCmd()
	require.NotNil(t, createCmd.Flags().Lookup("name"))
	require.NotNil(t, createCmd.Flags().Lookup("body"))
	require.NotNil(t, createCmd.Flags().Lookup("body-file"))

	updateCmd := newSignaturesUpdateCmd()
	require.NotNil(t, updateCmd.Flags().Lookup("name"))
	require.NotNil(t, updateCmd.Flags().Lookup("body"))
	require.NotNil(t, updateCmd.Flags().Lookup("body-file"))

	deleteCmd := newSignaturesDeleteCmd()
	require.NotNil(t, deleteCmd.Flags().Lookup("yes"))
}

func TestValidateSendSignatureSupport(t *testing.T) {
	tests := []struct {
		name        string
		signatureID string
		sign        bool
		encrypt     bool
		grant       *domain.Grant
		wantErr     bool
	}{
		{
			name:    "empty signature id is always allowed",
			wantErr: false,
		},
		{
			name:        "signing rejects stored signatures",
			signatureID: "sig-123",
			sign:        true,
			wantErr:     true,
		},
		{
			name:        "encrypting rejects stored signatures",
			signatureID: "sig-123",
			encrypt:     true,
			wantErr:     true,
		},
		{
			name:        "standard provider allows stored signatures",
			signatureID: "sig-123",
			grant:       &domain.Grant{Provider: domain.ProviderGoogle},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSendSignatureSupport(tt.signatureID, tt.sign, tt.encrypt, tt.grant)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidateSignatureSelection(t *testing.T) {
	ctx := context.Background()

	t.Run("empty signature id skips lookups", func(t *testing.T) {
		mock := nylas.NewMockClient()

		signatures, err := validateSignatureSelection(ctx, mock, "grant-123", "", nil)

		require.NoError(t, err)
		assert.Nil(t, signatures)
		assert.False(t, mock.GetGrantCalled)
		assert.False(t, mock.GetSignaturesCalled)
	})

	t.Run("uses provided grant and returns matching signatures", func(t *testing.T) {
		mock := nylas.NewMockClient()
		mock.GetSignaturesFunc = func(ctx context.Context, grantID string) ([]domain.Signature, error) {
			return []domain.Signature{
				{ID: "sig-123", Name: "Work", Body: "<p>Best regards</p>"},
				{ID: "sig-456", Name: "Other", Body: "<p>Thanks</p>"},
			}, nil
		}

		signatures, err := validateSignatureSelection(
			ctx,
			mock,
			"grant-123",
			"sig-123",
			&domain.Grant{Provider: domain.ProviderGoogle},
		)

		require.NoError(t, err)
		require.Len(t, signatures, 2)
		assert.False(t, mock.GetGrantCalled)
		assert.True(t, mock.GetSignaturesCalled)
	})

	t.Run("rejects nylas grants before listing signatures", func(t *testing.T) {
		mock := nylas.NewMockClient()

		signatures, err := validateSignatureSelection(
			ctx,
			mock,
			"grant-123",
			"sig-123",
			&domain.Grant{Provider: domain.ProviderNylas},
		)

		require.Error(t, err)
		assert.Nil(t, signatures)
		assert.ErrorContains(t, err, "`--signature-id` is not supported for managed transactional sends")
		assert.False(t, mock.GetGrantCalled)
		assert.False(t, mock.GetSignaturesCalled)
	})

	t.Run("rejects unknown signatures", func(t *testing.T) {
		mock := nylas.NewMockClient()
		mock.GetSignaturesFunc = func(ctx context.Context, grantID string) ([]domain.Signature, error) {
			return []domain.Signature{{ID: "sig-other", Name: "Other"}}, nil
		}

		signatures, err := validateSignatureSelection(ctx, mock, "grant-123", "sig-missing", nil)

		require.Error(t, err)
		assert.Nil(t, signatures)
		assert.True(t, mock.GetGrantCalled)
		assert.True(t, mock.GetSignaturesCalled)
		assert.ErrorContains(t, err, `signature "sig-missing" was not found for this grant`)
	})
}

func TestValidateDraftSendSignatureSelection(t *testing.T) {
	ctx := context.Background()

	t.Run("empty signature id skips validation", func(t *testing.T) {
		mock := nylas.NewMockClient()

		err := validateDraftSendSignatureSelection(ctx, mock, "grant-123", &domain.Draft{
			ID:   "draft-123",
			Body: "<p>Best regards</p>",
		}, "")

		require.NoError(t, err)
		assert.False(t, mock.GetGrantCalled)
		assert.False(t, mock.GetSignaturesCalled)
	})

	t.Run("rejects exact stored signature match in body", func(t *testing.T) {
		mock := nylas.NewMockClient()
		mock.GetSignaturesFunc = func(ctx context.Context, grantID string) ([]domain.Signature, error) {
			return []domain.Signature{{ID: "sig-123", Name: "Work", Body: "<p>Best regards</p>"}}, nil
		}

		err := validateDraftSendSignatureSelection(ctx, mock, "grant-123", &domain.Draft{
			ID:   "draft-123",
			Body: "<p>Hello</p><p>Best regards</p>",
		}, "sig-123")

		require.Error(t, err)
		assert.ErrorContains(t, err, "already contains a stored signature")
	})

	t.Run("rejects normalized stored signature match in body", func(t *testing.T) {
		mock := nylas.NewMockClient()
		mock.GetSignaturesFunc = func(ctx context.Context, grantID string) ([]domain.Signature, error) {
			return []domain.Signature{{ID: "sig-123", Name: "Work", Body: "<p>Best regards</p>"}}, nil
		}

		err := validateDraftSendSignatureSelection(ctx, mock, "grant-123", &domain.Draft{
			ID:   "draft-123",
			Body: "<div>Hello</div><div><strong>Best</strong> regards</div>",
		}, "sig-123")

		require.Error(t, err)
		assert.ErrorContains(t, err, "already contains a stored signature")
	})

	t.Run("allows send when body does not contain a stored signature", func(t *testing.T) {
		mock := nylas.NewMockClient()
		mock.GetSignaturesFunc = func(ctx context.Context, grantID string) ([]domain.Signature, error) {
			return []domain.Signature{{ID: "sig-123", Name: "Work", Body: "<p>Best regards</p>"}}, nil
		}

		err := validateDraftSendSignatureSelection(ctx, mock, "grant-123", &domain.Draft{
			ID:   "draft-123",
			Body: "<p>Hello</p><p>Talk soon</p>",
		}, "sig-123")

		require.NoError(t, err)
	})
}

func TestSendDraftRequest(t *testing.T) {
	assert.Nil(t, sendDraftRequest(""))

	req := sendDraftRequest("sig-123")
	require.NotNil(t, req)
	assert.Equal(t, "sig-123", req.SignatureID)
}

func TestSignaturePreview(t *testing.T) {
	assert.Equal(t, "", signaturePreview(""))
	assert.Contains(t, signaturePreview("<p>Hello <strong>world</strong></p>"), "Hello world")
}
