package email

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/domain"
	"github.com/spf13/cobra"
)

func TestSendCmd_GPGFlags(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantSign bool
		wantKey  string
	}{
		{
			name:     "no sign flag",
			args:     []string{"--to", "test@example.com", "--subject", "Test", "--body", "Body"},
			wantSign: false,
			wantKey:  "",
		},
		{
			name:     "sign flag set",
			args:     []string{"--to", "test@example.com", "--subject", "Test", "--body", "Body", "--sign"},
			wantSign: true,
			wantKey:  "",
		},
		{
			name:     "sign flag with specific key",
			args:     []string{"--to", "test@example.com", "--subject", "Test", "--body", "Body", "--sign", "--gpg-key", "ABC123"},
			wantSign: true,
			wantKey:  "ABC123",
		},
		{
			name:     "gpg-key without sign flag",
			args:     []string{"--to", "test@example.com", "--subject", "Test", "--body", "Body", "--gpg-key", "ABC123"},
			wantSign: false,
			wantKey:  "ABC123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newSendCmd()

			// Add no-confirm flag to skip prompts
			tt.args = append(tt.args, "--yes")

			// Set args
			cmd.SetArgs(tt.args)

			// Parse flags
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Get flag values
			signFlag, err := cmd.Flags().GetBool("sign")
			if err != nil {
				t.Fatalf("Failed to get sign flag: %v", err)
			}

			gpgKeyFlag, err := cmd.Flags().GetString("gpg-key")
			if err != nil {
				t.Fatalf("Failed to get gpg-key flag: %v", err)
			}

			// Validate
			if signFlag != tt.wantSign {
				t.Errorf("sign flag = %v, want %v", signFlag, tt.wantSign)
			}
			if gpgKeyFlag != tt.wantKey {
				t.Errorf("gpg-key flag = %v, want %v", gpgKeyFlag, tt.wantKey)
			}
		})
	}
}

func TestSendCmd_ListGPGKeysFlag(t *testing.T) {
	cmd := newSendCmd()

	args := []string{"--list-gpg-keys"}
	cmd.SetArgs(args)

	// Parse flags
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	listGPGKeysFlag, err := cmd.Flags().GetBool("list-gpg-keys")
	if err != nil {
		t.Fatalf("Failed to get list-gpg-keys flag: %v", err)
	}

	if !listGPGKeysFlag {
		t.Error("Expected list-gpg-keys flag to be true")
	}
}

func TestSendCmd_AutoSignConfig(t *testing.T) {
	// Create temp config directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Set up config with auto-sign enabled
	store := config.NewFileStore(configPath)
	cfg := domain.DefaultConfig()
	cfg.GPG = &domain.GPGConfig{
		AutoSign: true,
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Override config path for this test
	originalConfigPath := os.Getenv("NYLAS_CONFIG_PATH")
	_ = os.Setenv("NYLAS_CONFIG_PATH", configPath)
	defer func() {
		if originalConfigPath != "" {
			_ = os.Setenv("NYLAS_CONFIG_PATH", originalConfigPath)
		} else {
			_ = os.Unsetenv("NYLAS_CONFIG_PATH")
		}
	}()

	// Note: The actual auto-sign loading happens in RunE, which we can't easily test
	// without mocking the entire client and network stack. This test validates
	// that the config can be loaded and has the expected structure.

	loadedCfg, err := store.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.GPG == nil {
		t.Fatal("GPG config is nil")
	}
	if !loadedCfg.GPG.AutoSign {
		t.Error("Expected auto_sign to be true")
	}
}

func TestSendCmd_FlagDefinitions(t *testing.T) {
	cmd := newSendCmd()

	// Test that all expected flags are defined
	expectedFlags := []struct {
		name      string
		shorthand string
		flagType  string
	}{
		{name: "to", shorthand: "", flagType: "stringSlice"},
		{name: "cc", shorthand: "", flagType: "stringSlice"},
		{name: "bcc", shorthand: "", flagType: "stringSlice"},
		{name: "subject", shorthand: "s", flagType: "string"},
		{name: "body", shorthand: "b", flagType: "string"},
		{name: "sign", shorthand: "", flagType: "bool"},
		{name: "gpg-key", shorthand: "", flagType: "string"},
		{name: "list-gpg-keys", shorthand: "", flagType: "bool"},
		{name: "interactive", shorthand: "i", flagType: "bool"},
		{name: "yes", shorthand: "y", flagType: "bool"},
	}

	for _, expected := range expectedFlags {
		flag := cmd.Flags().Lookup(expected.name)
		if flag == nil {
			t.Errorf("Flag %s not found", expected.name)
			continue
		}

		if expected.shorthand != "" && flag.Shorthand != expected.shorthand {
			t.Errorf("Flag %s shorthand = %s, want %s", expected.name, flag.Shorthand, expected.shorthand)
		}

		// Validate flag type
		switch expected.flagType {
		case "bool":
			if flag.Value.Type() != "bool" {
				t.Errorf("Flag %s type = %s, want bool", expected.name, flag.Value.Type())
			}
		case "string":
			if flag.Value.Type() != "string" {
				t.Errorf("Flag %s type = %s, want string", expected.name, flag.Value.Type())
			}
		case "stringSlice":
			if flag.Value.Type() != "stringSlice" {
				t.Errorf("Flag %s type = %s, want stringSlice", expected.name, flag.Value.Type())
			}
		}
	}
}

func TestSendCmd_UsageText(t *testing.T) {
	cmd := newSendCmd()

	// Validate that usage includes GPG information
	usage := cmd.Long
	if !strings.Contains(usage, "GPG") && !strings.Contains(usage, "PGP") {
		t.Error("Usage text should mention GPG/PGP signing")
	}
	if !strings.Contains(usage, "--sign") {
		t.Error("Usage text should mention --sign flag")
	}
	if !strings.Contains(usage, "--gpg-key") {
		t.Error("Usage text should mention --gpg-key flag")
	}
	if !strings.Contains(usage, "--list-gpg-keys") {
		t.Error("Usage text should mention --list-gpg-keys flag")
	}

	// Validate examples include GPG
	example := cmd.Example
	if !strings.Contains(example, "gpg") && !strings.Contains(example, "sign") {
		t.Error("Examples should include GPG signing usage")
	}
}

func TestSendCmd_CommandStructure(t *testing.T) {
	cmd := newSendCmd()

	// Validate basic command structure
	if cmd.Use != "send [grant-id]" {
		t.Errorf("Use = %s, want 'send [grant-id]'", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.Example == "" {
		t.Error("Examples should not be empty")
	}

	// Validate RunE is set
	if cmd.RunE == nil {
		t.Error("RunE function should be set")
	}

	// Validate max args
	if cmd.Args == nil {
		t.Error("Args validator should be set")
	}
}

func TestSendCmd_GPGKeyFlagValidation(t *testing.T) {
	tests := []struct {
		name    string
		keyID   string
		wantErr bool
	}{
		{
			name:    "valid key ID",
			keyID:   "601FEE9B1D60185F",
			wantErr: false,
		},
		{
			name:    "short key ID",
			keyID:   "1D60185F",
			wantErr: false,
		},
		{
			name:    "full fingerprint",
			keyID:   "1234567890ABCDEF1234567890ABCDEF12345678",
			wantErr: false,
		},
		{
			name:    "email as identifier",
			keyID:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "empty key ID",
			keyID:   "",
			wantErr: false, // Empty is valid (uses default)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newSendCmd()

			args := []string{
				"--to", "test@example.com",
				"--subject", "Test",
				"--body", "Body",
				"--sign",
			}

			if tt.keyID != "" {
				args = append(args, "--gpg-key", tt.keyID)
			}

			cmd.SetArgs(args)

			// Parse flags (should not error on valid key IDs)
			err := cmd.ParseFlags(args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				gpgKey, _ := cmd.Flags().GetString("gpg-key")
				if tt.keyID != "" && gpgKey != tt.keyID {
					t.Errorf("gpg-key = %s, want %s", gpgKey, tt.keyID)
				}
			}
		})
	}
}

// TestSendCmd_Integration tests the command in a more integrated way
// This test validates that flags work together correctly
func TestSendCmd_Integration(t *testing.T) {
	tests := []struct {
		name string
		args []string
		test func(t *testing.T, cmd *cobra.Command)
	}{
		{
			name: "basic send with GPG signing",
			args: []string{
				"--to", "recipient@example.com",
				"--subject", "Secure Message",
				"--body", "This is a signed email",
				"--sign",
				"--yes",
			},
			test: func(t *testing.T, cmd *cobra.Command) {
				sign, _ := cmd.Flags().GetBool("sign")
				if !sign {
					t.Error("Expected sign flag to be true")
				}

				to, _ := cmd.Flags().GetStringSlice("to")
				if len(to) != 1 || to[0] != "recipient@example.com" {
					t.Errorf("to = %v, want [recipient@example.com]", to)
				}
			},
		},
		{
			name: "send with specific GPG key",
			args: []string{
				"--to", "recipient@example.com",
				"--subject", "Test",
				"--body", "Body",
				"--sign",
				"--gpg-key", "ABC123",
				"--yes",
			},
			test: func(t *testing.T, cmd *cobra.Command) {
				sign, _ := cmd.Flags().GetBool("sign")
				gpgKey, _ := cmd.Flags().GetString("gpg-key")

				if !sign {
					t.Error("Expected sign flag to be true")
				}
				if gpgKey != "ABC123" {
					t.Errorf("gpg-key = %s, want ABC123", gpgKey)
				}
			},
		},
		{
			name: "send with CC and BCC while signing",
			args: []string{
				"--to", "to@example.com",
				"--cc", "cc@example.com",
				"--bcc", "bcc@example.com",
				"--subject", "Multi-recipient",
				"--body", "Test",
				"--sign",
				"--yes",
			},
			test: func(t *testing.T, cmd *cobra.Command) {
				sign, _ := cmd.Flags().GetBool("sign")
				to, _ := cmd.Flags().GetStringSlice("to")
				cc, _ := cmd.Flags().GetStringSlice("cc")
				bcc, _ := cmd.Flags().GetStringSlice("bcc")

				if !sign {
					t.Error("Expected sign flag to be true")
				}
				if len(to) != 1 {
					t.Errorf("to length = %d, want 1", len(to))
				}
				if len(cc) != 1 {
					t.Errorf("cc length = %d, want 1", len(cc))
				}
				if len(bcc) != 1 {
					t.Errorf("bcc length = %d, want 1", len(bcc))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newSendCmd()
			cmd.SetArgs(tt.args)

			// Parse flags
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Run test-specific validation
			tt.test(t, cmd)
		})
	}
}

// TestSendCmd_OutputCapture tests that the command can capture output
func TestSendCmd_OutputCapture(t *testing.T) {
	cmd := newSendCmd()

	// Redirect stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	// Set help flag to get help output
	cmd.SetArgs([]string{"--help"})

	// Execute (should show help and exit)
	_ = cmd.Execute()

	// Close write end and read captured output
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	// Validate help output includes GPG information
	if !strings.Contains(output, "gpg") && !strings.Contains(output, "GPG") {
		t.Error("Help output should mention GPG")
	}
}
