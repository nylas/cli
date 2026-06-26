package rpcserver

import (
	"context"
	"errors"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

type fakeOTPService struct {
	calledGetOTP        bool
	calledGetOTPDefault bool
	email               string
	result              *domain.OTPResult
	err                 error
}

func (f *fakeOTPService) GetOTP(_ context.Context, email string) (*domain.OTPResult, error) {
	f.calledGetOTP = true
	f.email = email
	return f.result, f.err
}

func (f *fakeOTPService) GetOTPDefault(_ context.Context) (*domain.OTPResult, error) {
	f.calledGetOTPDefault = true
	return f.result, f.err
}

func TestRegisterOTPHandlers(t *testing.T) {
	tests := []struct {
		name   string
		params string
		svc    *fakeOTPService
		assert func(*testing.T, *fakeOTPService, rpcTestResponse)
	}{
		{
			name:   "otp.get with email routes to GetOTP",
			params: `{"email":"user@example.com"}`,
			svc: &fakeOTPService{
				result: &domain.OTPResult{Code: "123456", From: "sender@example.com", Subject: "Your code", MessageID: "msg-1"},
			},
			assert: func(t *testing.T, svc *fakeOTPService, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if !svc.calledGetOTP || svc.calledGetOTPDefault {
					t.Fatalf("calledGetOTP = %t, calledGetOTPDefault = %t; want GetOTP only", svc.calledGetOTP, svc.calledGetOTPDefault)
				}
				if svc.email != "user@example.com" {
					t.Fatalf("email = %q, want user@example.com", svc.email)
				}

				var result domain.OTPResult
				unmarshalResult(t, resp, &result)
				if result.Code != "123456" || result.From != "sender@example.com" || result.Subject != "Your code" || result.MessageID != "msg-1" {
					t.Fatalf("result = %#v, want returned OTP result", result)
				}
			},
		},
		{
			name:   "otp.get without email routes to GetOTPDefault",
			params: `{}`,
			svc: &fakeOTPService{
				result: &domain.OTPResult{Code: "654321", From: "default@example.com", Subject: "Default code", MessageID: "msg-2"},
			},
			assert: func(t *testing.T, svc *fakeOTPService, resp rpcTestResponse) {
				requireNoRPCError(t, resp)
				if svc.calledGetOTP || !svc.calledGetOTPDefault {
					t.Fatalf("calledGetOTP = %t, calledGetOTPDefault = %t; want GetOTPDefault only", svc.calledGetOTP, svc.calledGetOTPDefault)
				}

				var result domain.OTPResult
				unmarshalResult(t, resp, &result)
				if result.Code != "654321" || result.From != "default@example.com" || result.Subject != "Default code" || result.MessageID != "msg-2" {
					t.Fatalf("result = %#v, want returned OTP result", result)
				}
			},
		},
		{
			name:   "otp.get service error returns internal error",
			params: `{"email":"user@example.com"}`,
			svc: &fakeOTPService{
				err: errors.New("otp unavailable"),
			},
			assert: func(t *testing.T, svc *fakeOTPService, resp rpcTestResponse) {
				if !svc.calledGetOTP || svc.email != "user@example.com" {
					t.Fatalf("calledGetOTP = %t, email = %q; want GetOTP with email", svc.calledGetOTP, svc.email)
				}
				requireRPCErrorCode(t, resp, InternalError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDispatcher()
			RegisterOTPHandlers(d, tt.svc)

			resp := dispatchLocalRequest(t, d, "otp.get", tt.params)
			tt.assert(t, tt.svc, resp)
		})
	}
}
