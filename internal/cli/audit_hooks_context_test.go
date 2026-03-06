package cli

import (
	"testing"
)

func TestSetAuditRequestInfo(t *testing.T) {
	tests := []struct {
		name       string
		setup      func()
		requestID  string
		httpStatus int
		wantID     string
		wantStatus int
	}{
		{
			name: "sets request info when audit context exists",
			setup: func() {
				auditMu.Lock()
				currentAudit = &AuditContext{}
				auditMu.Unlock()
			},
			requestID:  "req-123",
			httpStatus: 200,
			wantID:     "req-123",
			wantStatus: 200,
		},
		{
			name: "does nothing when audit context is nil",
			setup: func() {
				auditMu.Lock()
				currentAudit = nil
				auditMu.Unlock()
			},
			requestID:  "req-456",
			httpStatus: 500,
			wantID:     "",
			wantStatus: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			SetAuditRequestInfo(tt.requestID, tt.httpStatus)

			auditMu.Lock()
			defer auditMu.Unlock()

			if currentAudit != nil {
				if currentAudit.RequestID != tt.wantID {
					t.Errorf("RequestID = %q, want %q", currentAudit.RequestID, tt.wantID)
				}
				if currentAudit.HTTPStatus != tt.wantStatus {
					t.Errorf("HTTPStatus = %d, want %d", currentAudit.HTTPStatus, tt.wantStatus)
				}
			}
		})
	}

	// Cleanup
	auditMu.Lock()
	currentAudit = nil
	auditMu.Unlock()
}

func TestSetAuditGrantInfo(t *testing.T) {
	tests := []struct {
		name       string
		setup      func()
		grantID    string
		grantEmail string
		wantID     string
		wantEmail  string
	}{
		{
			name: "sets grant info when audit context exists",
			setup: func() {
				auditMu.Lock()
				currentAudit = &AuditContext{}
				auditMu.Unlock()
			},
			grantID:    "grant-123",
			grantEmail: "alice@example.com",
			wantID:     "grant-123",
			wantEmail:  "alice@example.com",
		},
		{
			name: "does nothing when audit context is nil",
			setup: func() {
				auditMu.Lock()
				currentAudit = nil
				auditMu.Unlock()
			},
			grantID:    "grant-456",
			grantEmail: "bob@example.com",
			wantID:     "",
			wantEmail:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			SetAuditGrantInfo(tt.grantID, tt.grantEmail)

			auditMu.Lock()
			defer auditMu.Unlock()

			if currentAudit != nil {
				if currentAudit.GrantID != tt.wantID {
					t.Errorf("GrantID = %q, want %q", currentAudit.GrantID, tt.wantID)
				}
				if currentAudit.GrantEmail != tt.wantEmail {
					t.Errorf("GrantEmail = %q, want %q", currentAudit.GrantEmail, tt.wantEmail)
				}
			}
		})
	}

	// Cleanup
	auditMu.Lock()
	currentAudit = nil
	auditMu.Unlock()
}
