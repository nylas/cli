package rpcserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nylas/cli/internal/domain"
)

type otpGetParams struct {
	Email string `json:"email"`
}

type otpService interface {
	GetOTP(ctx context.Context, email string) (*domain.OTPResult, error)
	GetOTPDefault(ctx context.Context) (*domain.OTPResult, error)
}

func RegisterOTPHandlers(d *Dispatcher, svc otpService) {
	d.Register("otp.get", func(ctx context.Context, params json.RawMessage) (any, error) {
		var p otpGetParams
		if err := decodeParams(params, &p); err != nil {
			return nil, err
		}

		var (
			result *domain.OTPResult
			err    error
		)
		if p.Email != "" {
			result, err = svc.GetOTP(ctx, p.Email)
		} else {
			result, err = svc.GetOTPDefault(ctx)
		}
		if err != nil {
			return nil, fmt.Errorf("otp.get: %w", err)
		}
		return result, nil
	})
}
