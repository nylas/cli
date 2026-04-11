package nylas

import (
	"context"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func (d *DemoClient) GetSignatures(ctx context.Context, grantID string) ([]domain.Signature, error) {
	now := time.Now()
	return []domain.Signature{
		{
			ID:        "sig-demo-work",
			Name:      "Work",
			Body:      "<div><strong>Demo User</strong><br/>Developer Advocate</div>",
			CreatedAt: now.Add(-24 * time.Hour),
			UpdatedAt: now.Add(-2 * time.Hour),
		},
		{
			ID:        "sig-demo-mobile",
			Name:      "Mobile",
			Body:      "<div>Sent from my phone</div>",
			CreatedAt: now.Add(-48 * time.Hour),
			UpdatedAt: now.Add(-6 * time.Hour),
		},
	}, nil
}

func (d *DemoClient) GetSignature(ctx context.Context, grantID, signatureID string) (*domain.Signature, error) {
	signatures, err := d.GetSignatures(ctx, grantID)
	if err != nil {
		return nil, err
	}
	for _, signature := range signatures {
		if signature.ID == signatureID {
			return &signature, nil
		}
	}
	return nil, domain.ErrSignatureNotFound
}

func (d *DemoClient) CreateSignature(ctx context.Context, grantID string, req *domain.CreateSignatureRequest) (*domain.Signature, error) {
	now := time.Now()
	return &domain.Signature{
		ID:        "sig-demo-new",
		Name:      req.Name,
		Body:      req.Body,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (d *DemoClient) UpdateSignature(ctx context.Context, grantID, signatureID string, req *domain.UpdateSignatureRequest) (*domain.Signature, error) {
	signature := &domain.Signature{
		ID:        signatureID,
		Name:      "Work",
		Body:      "<div><strong>Demo User</strong><br/>Developer Advocate</div>",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
	}
	if req.Name != nil {
		signature.Name = *req.Name
	}
	if req.Body != nil {
		signature.Body = *req.Body
	}
	return signature, nil
}

func (d *DemoClient) DeleteSignature(ctx context.Context, grantID, signatureID string) error {
	return nil
}
