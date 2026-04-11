package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (m *MockClient) GetSignatures(ctx context.Context, grantID string) ([]domain.Signature, error) {
	m.GetSignaturesCalled = true
	m.LastGrantID = grantID
	if m.GetSignaturesFunc != nil {
		return m.GetSignaturesFunc(ctx, grantID)
	}
	return []domain.Signature{
		{
			ID:        "sig-123",
			Name:      "Work Signature",
			Body:      "<p>Best regards</p>",
			CreatedAt: time.Unix(1704067200, 0),
			UpdatedAt: time.Unix(1704067200, 0),
		},
	}, nil
}

func (m *MockClient) GetSignature(ctx context.Context, grantID, signatureID string) (*domain.Signature, error) {
	m.GetSignatureCalled = true
	m.LastGrantID = grantID
	m.LastSignatureID = signatureID
	if m.GetSignatureFunc != nil {
		return m.GetSignatureFunc(ctx, grantID, signatureID)
	}
	return &domain.Signature{
		ID:        signatureID,
		Name:      "Work Signature",
		Body:      "<p>Best regards</p>",
		CreatedAt: time.Unix(1704067200, 0),
		UpdatedAt: time.Unix(1704067200, 0),
	}, nil
}

func (m *MockClient) CreateSignature(ctx context.Context, grantID string, req *domain.CreateSignatureRequest) (*domain.Signature, error) {
	m.CreateSignatureCalled = true
	m.LastGrantID = grantID
	if m.CreateSignatureFunc != nil {
		return m.CreateSignatureFunc(ctx, grantID, req)
	}
	return &domain.Signature{
		ID:        "sig-new",
		Name:      req.Name,
		Body:      req.Body,
		CreatedAt: time.Unix(1704067200, 0),
		UpdatedAt: time.Unix(1704067200, 0),
	}, nil
}

func (m *MockClient) UpdateSignature(ctx context.Context, grantID, signatureID string, req *domain.UpdateSignatureRequest) (*domain.Signature, error) {
	m.UpdateSignatureCalled = true
	m.LastGrantID = grantID
	m.LastSignatureID = signatureID
	if m.UpdateSignatureFunc != nil {
		return m.UpdateSignatureFunc(ctx, grantID, signatureID, req)
	}

	signature := &domain.Signature{
		ID:        signatureID,
		Name:      "Work Signature",
		Body:      "<p>Best regards</p>",
		CreatedAt: time.Unix(1704067200, 0),
		UpdatedAt: time.Unix(1704068200, 0),
	}
	if req.Name != nil {
		signature.Name = *req.Name
	}
	if req.Body != nil {
		signature.Body = *req.Body
	}
	return signature, nil
}

func (m *MockClient) DeleteSignature(ctx context.Context, grantID, signatureID string) error {
	m.DeleteSignatureCalled = true
	m.LastGrantID = grantID
	m.LastSignatureID = signatureID
	if m.DeleteSignatureFunc != nil {
		return m.DeleteSignatureFunc(ctx, grantID, signatureID)
	}
	return nil
}
