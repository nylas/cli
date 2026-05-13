package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *APIError
		want    []string
		wantNot []string
	}{
		{
			name: "message and type both surfaced",
			err:  &APIError{StatusCode: 403, Type: "insufficient_scopes", Message: "Grant lacks gmail.readonly"},
			want: []string{"Grant lacks gmail.readonly", "insufficient_scopes"},
		},
		{
			name: "message only",
			err:  &APIError{StatusCode: 403, Message: "Access denied"},
			want: []string{"Access denied"},
		},
		{
			name:    "type only (empty message)",
			err:     &APIError{StatusCode: 403, Type: "insufficient_scopes"},
			want:    []string{"insufficient_scopes"},
			wantNot: []string{"status 403"},
		},
		{
			name: "neither message nor type",
			err:  &APIError{StatusCode: 403},
			want: []string{"status 403"},
		},
		{
			name: "completely empty",
			err:  &APIError{},
			want: []string{ErrAPIError.Error()},
		},
		{
			name: "nil",
			err:  nil,
			want: []string{ErrAPIError.Error()},
		},
		{
			name: "request id appended when present",
			err:  &APIError{StatusCode: 401, Type: "token.unauthorized_access", Message: "Bearer token invalid", RequestID: "1120765200-c4c8e151-3414-4448-b884-1498872b0912"},
			want: []string{"Bearer token invalid", "token.unauthorized_access", "request_id: 1120765200-c4c8e151-3414-4448-b884-1498872b0912"},
		},
		{
			name:    "blank request id is omitted",
			err:     &APIError{StatusCode: 500, RequestID: "   "},
			want:    []string{"status 500"},
			wantNot: []string{"request_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("Error() = %q, missing substring %q", got, want)
				}
			}
			for _, notWant := range tt.wantNot {
				if strings.Contains(got, notWant) {
					t.Errorf("Error() = %q, should not contain %q", got, notWant)
				}
			}
		})
	}
}

func TestAPIError_UnwrapsToErrAPIError(t *testing.T) {
	err := &APIError{StatusCode: 403, Type: "insufficient_scopes"}
	if !errors.Is(err, ErrAPIError) {
		t.Errorf("APIError should unwrap to ErrAPIError")
	}
}
