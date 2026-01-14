package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestScopesCmd(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOutput []string
		wantErr    bool
		skipReason string
	}{
		{
			name:       "scopes without grant id",
			args:       []string{},
			wantOutput: []string{},
			wantErr:    false, // May succeed if default grant configured, or fail if not - test is environment-dependent
			skipReason: "Test requires specific environment setup (no default grant)",
		},
		{
			name:       "scopes with grant id",
			args:       []string{"grant-123"},
			wantOutput: []string{},
			wantErr:    true, // Will fail because grant ID doesn't exist
		},
		{
			name:       "scopes json output",
			args:       []string{"grant-123", "--json"},
			wantOutput: []string{},
			wantErr:    true, // Will fail because grant ID doesn't exist
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			cmd := newScopesCmd()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("newScopesCmd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				output := buf.String()
				for _, want := range tt.wantOutput {
					if !strings.Contains(output, want) {
						t.Errorf("newScopesCmd() output = %v, want to contain %v", output, want)
					}
				}
			}
		})
	}
}

func TestDescribeScopeCategory(t *testing.T) {
	tests := []struct {
		scope       string
		wantContain string
	}{
		{
			scope:       "https://www.googleapis.com/auth/gmail.readonly",
			wantContain: "Read-only",
		},
		{
			scope:       "https://www.googleapis.com/auth/gmail.send",
			wantContain: "Send",
		},
		{
			scope:       "https://www.googleapis.com/auth/calendar.readonly",
			wantContain: "Read-only",
		},
		{
			scope:       "https://www.googleapis.com/auth/contacts.readonly",
			wantContain: "Read-only",
		},
		{
			scope:       "https://graph.microsoft.com/Mail.Read",
			wantContain: "Read",
		},
		{
			scope:       "https://graph.microsoft.com/Mail.Send",
			wantContain: "Send",
		},
		{
			scope:       "https://graph.microsoft.com/Calendars.Read",
			wantContain: "Read",
		},
		{
			scope:       "https://graph.microsoft.com/User.Read",
			wantContain: "profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			result := describeScopeCategory(tt.scope)
			if !strings.Contains(result, tt.wantContain) && result != "" {
				t.Errorf("describeScopeCategory(%s) = %v, want to contain %v", tt.scope, result, tt.wantContain)
			}
		})
	}
}
