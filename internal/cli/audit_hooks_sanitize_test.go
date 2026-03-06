package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestGetCommandPath(t *testing.T) {
	tests := []struct {
		name     string
		setupCmd func() *cobra.Command
		want     string
	}{
		{
			name: "single command",
			setupCmd: func() *cobra.Command {
				return &cobra.Command{Use: "list"}
			},
			want: "list",
		},
		{
			name: "nested command under nylas",
			setupCmd: func() *cobra.Command {
				root := &cobra.Command{Use: "nylas"}
				email := &cobra.Command{Use: "email"}
				list := &cobra.Command{Use: "list"}
				root.AddCommand(email)
				email.AddCommand(list)
				return list
			},
			want: "email list",
		},
		{
			name: "deeply nested command",
			setupCmd: func() *cobra.Command {
				root := &cobra.Command{Use: "nylas"}
				email := &cobra.Command{Use: "email"}
				attachments := &cobra.Command{Use: "attachments"}
				download := &cobra.Command{Use: "download"}
				root.AddCommand(email)
				email.AddCommand(attachments)
				attachments.AddCommand(download)
				return download
			},
			want: "email attachments download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.setupCmd()
			got := getCommandPath(cmd)
			if got != tt.want {
				t.Errorf("getCommandPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsExcludedCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmdName string
		want    bool
	}{
		{"help command excluded", "help", true},
		{"version command excluded", "version", true},
		{"completion command excluded", "completion", true},
		{"__complete excluded", "__complete", true},
		{"__completeNoDesc excluded", "__completeNoDesc", true},
		{"email command not excluded", "email", false},
		{"list command not excluded", "list", false},
		{"audit command not excluded", "audit", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: tt.cmdName}
			got := isExcludedCommand(cmd)
			if got != tt.want {
				t.Errorf("isExcludedCommand(%q) = %v, want %v", tt.cmdName, got, tt.want)
			}
		})
	}
}

func TestSanitizeArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "empty args",
			args: []string{},
			want: []string{},
		},
		{
			name: "no sensitive args",
			args: []string{"--limit", "10", "--format", "json"},
			want: []string{"--limit", "10", "--format", "json"},
		},
		{
			name: "redacts --api-key value",
			args: []string{"--api-key", "secret123"},
			want: []string{"--api-key", "[REDACTED]"},
		},
		{
			name: "redacts --password value",
			args: []string{"--password", "mypassword"},
			want: []string{"--password", "[REDACTED]"},
		},
		{
			name: "redacts --token value",
			args: []string{"--token", "tok_abc123"},
			want: []string{"--token", "[REDACTED]"},
		},
		{
			name: "redacts --secret value",
			args: []string{"--secret", "supersecret"},
			want: []string{"--secret", "[REDACTED]"},
		},
		{
			name: "redacts --client-secret value",
			args: []string{"--client-secret", "clientsecret123"},
			want: []string{"--client-secret", "[REDACTED]"},
		},
		{
			name: "redacts --access-token value",
			args: []string{"--access-token", "access123"},
			want: []string{"--access-token", "[REDACTED]"},
		},
		{
			name: "redacts --refresh-token value",
			args: []string{"--refresh-token", "refresh456"},
			want: []string{"--refresh-token", "[REDACTED]"},
		},
		{
			name: "redacts --body value",
			args: []string{"--body", "sensitive content"},
			want: []string{"--body", "[REDACTED]"},
		},
		{
			name: "redacts --subject value",
			args: []string{"--subject", "Private email subject"},
			want: []string{"--subject", "[REDACTED]"},
		},
		{
			name: "redacts --html value",
			args: []string{"--html", "<html>content</html>"},
			want: []string{"--html", "[REDACTED]"},
		},
		{
			name: "redacts -p short flag",
			args: []string{"-p", "password123"},
			want: []string{"-p", "[REDACTED]"},
		},
		{
			name: "redacts --flag=value format",
			args: []string{"--api-key=secret123"},
			want: []string{"--api-key=[REDACTED]"},
		},
		{
			name: "redacts nyk_ prefixed tokens",
			args: []string{"nyk_abcdef123456789012345678901234567890"},
			want: []string{"[REDACTED]"},
		},
		{
			name: "redacts long base64 strings",
			args: []string{"YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkw"},
			want: []string{"[REDACTED]"},
		},
		{
			name: "mixed args with sensitive and non-sensitive",
			args: []string{"--limit", "10", "--api-key", "secret", "--format", "json"},
			want: []string{"--limit", "10", "--api-key", "[REDACTED]", "--format", "json"},
		},
		{
			name: "multiple sensitive flags",
			args: []string{"--password", "pass1", "--token", "tok1"},
			want: []string{"--password", "[REDACTED]", "--token", "[REDACTED]"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeArgs(tt.args)
			if len(got) != len(tt.want) {
				t.Errorf("sanitizeArgs() length = %d, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("sanitizeArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIsLongBase64(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"short string", "abc", false},
		{"exactly 39 chars", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklm", false},
		{"40 char base64", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmn", true},
		{"long base64 with numbers", "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkw", true},
		{"base64 with plus", "ABCDEFGHIJKLMNOPQRSTUVWXYZ+abcdefghijklmn", true},
		{"base64 with slash", "ABCDEFGHIJKLMNOPQRSTUVWXYZ/abcdefghijklmn", true},
		{"base64 with equals", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijk===", true},
		{"base64 with dash (URL safe)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ-abcdefghijklmn", true},
		{"base64 with underscore (URL safe)", "ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmn", true},
		{"contains space", "ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmn", false},
		{"contains special char", "ABCDEFGHIJKLMNOPQRSTUVWXYZ!abcdefghijklmn", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLongBase64(tt.input)
			if got != tt.want {
				t.Errorf("isLongBase64(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
