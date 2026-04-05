package cli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSecretStore_WarnsWhenFileStoreIsForced(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempDir, "xdg"))
	t.Setenv("HOME", tempDir)
	t.Setenv("NYLAS_DISABLE_KEYRING", "true")
	t.Setenv("NYLAS_FILE_STORE_PASSPHRASE", "doctor-test-passphrase")

	result := checkSecretStore()

	if result.Status != CheckStatusWarning {
		t.Fatalf("Status = %v, want %v", result.Status, CheckStatusWarning)
	}
	if result.Message != "encrypted file" {
		t.Fatalf("Message = %q, want %q", result.Message, "encrypted file")
	}
	if !strings.Contains(result.Detail, "NYLAS_FILE_STORE_PASSPHRASE") {
		t.Fatalf("Detail %q does not mention NYLAS_FILE_STORE_PASSPHRASE", result.Detail)
	}
	if !strings.Contains(result.Detail, "unset NYLAS_DISABLE_KEYRING") {
		t.Fatalf("Detail %q does not mention unsetting NYLAS_DISABLE_KEYRING", result.Detail)
	}
}
