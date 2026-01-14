package nylas

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		wantErr   bool
	}{
		{
			name:      "valid value",
			fieldName: "grant ID",
			value:     "grant-123",
			wantErr:   false,
		},
		{
			name:      "empty value",
			fieldName: "grant ID",
			value:     "",
			wantErr:   true,
		},
		{
			name:      "whitespace only is valid",
			fieldName: "field",
			value:     "   ",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequired(tt.fieldName, tt.value)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if !errors.Is(err, domain.ErrInvalidInput) {
					t.Errorf("expected ErrInvalidInput, got %v", err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetRequestID(t *testing.T) {
	tests := []struct {
		name     string
		resp     *http.Response
		expected string
	}{
		{
			name:     "nil response",
			resp:     nil,
			expected: "",
		},
		{
			name: "X-Request-Id header",
			resp: &http.Response{
				Header: http.Header{
					"X-Request-Id": []string{"req-123"},
				},
			},
			expected: "req-123",
		},
		{
			name: "Request-Id header",
			resp: &http.Response{
				Header: http.Header{
					"Request-Id": []string{"req-456"},
				},
			},
			expected: "req-456",
		},
		{
			name: "X-Request-Id takes precedence",
			resp: &http.Response{
				Header: http.Header{
					"X-Request-Id": []string{"req-primary"},
					"Request-Id":   []string{"req-secondary"},
				},
			},
			expected: "req-primary",
		},
		{
			name: "no request ID header",
			resp: &http.Response{
				Header: http.Header{},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getRequestID(tt.resp)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHTTPClient_GetMessages_Validation(t *testing.T) {
	client := NewHTTPClient()
	ctx := context.Background()

	t.Run("rejects empty grant ID", func(t *testing.T) {
		_, err := client.GetMessages(ctx, "", 10)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestHTTPClient_GetMessage_Validation(t *testing.T) {
	client := NewHTTPClient()
	ctx := context.Background()

	t.Run("rejects empty grant ID", func(t *testing.T) {
		_, err := client.GetMessage(ctx, "", "msg-123")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("rejects empty message ID", func(t *testing.T) {
		_, err := client.GetMessage(ctx, "grant-123", "")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestHTTPClient_GetEvents_Validation(t *testing.T) {
	client := NewHTTPClient()
	ctx := context.Background()

	t.Run("rejects empty grant ID", func(t *testing.T) {
		_, err := client.GetEvents(ctx, "", "calendar-123", nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("rejects empty calendar ID", func(t *testing.T) {
		_, err := client.GetEvents(ctx, "grant-123", "", nil)
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}

func TestHTTPClient_GetEvent_Validation(t *testing.T) {
	client := NewHTTPClient()
	ctx := context.Background()

	t.Run("rejects empty grant ID", func(t *testing.T) {
		_, err := client.GetEvent(ctx, "", "calendar-123", "event-123")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("rejects empty calendar ID", func(t *testing.T) {
		_, err := client.GetEvent(ctx, "grant-123", "", "event-123")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("rejects empty event ID", func(t *testing.T) {
		_, err := client.GetEvent(ctx, "grant-123", "calendar-123", "")
		if err == nil {
			t.Error("expected error, got nil")
		}
		if !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}
