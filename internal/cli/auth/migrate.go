package auth

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/adapters/config"
	"github.com/nylas/cli/internal/adapters/keyring"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/ports"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate credentials from encrypted file to system keyring",
		Long: `Migrate credentials from the encrypted file store to the system keyring.

This command is useful when credentials were originally stored in the encrypted
file (e.g., when running in a sandboxed environment) but you now want to use
the more secure system keyring.

The system keyring provides better security as it's protected by your OS
authentication (macOS Keychain, Linux Secret Service, Windows Credential Manager).`,
		RunE: runMigrate,
	}

	return cmd
}

func runMigrate(_ *cobra.Command, _ []string) error {
	configDir := config.DefaultConfigDir()

	// Check if keyring is disabled
	if os.Getenv("NYLAS_DISABLE_KEYRING") == "true" {
		return fmt.Errorf("NYLAS_DISABLE_KEYRING is set. Unset it to use system keyring:\n  unset NYLAS_DISABLE_KEYRING")
	}

	// Check if system keyring is available
	kr := keyring.NewSystemKeyring()
	if !kr.IsAvailable() {
		return fmt.Errorf("system keyring is not available on this system")
	}

	// Check if already using keyring
	if _, err := kr.Get(ports.KeyAPIKey); err == nil {
		fmt.Println("✓ Already using system keyring")
		return nil
	}

	// Load credentials from encrypted file
	fileStore, err := keyring.NewEncryptedFileStore(configDir)
	if err != nil {
		return common.WrapGetError("encrypted file store", err)
	}

	// Get credentials from file store
	apiKey, err := fileStore.Get(ports.KeyAPIKey)
	if err != nil {
		return fmt.Errorf("no credentials found in encrypted file store")
	}

	// Migrate API key
	if err := kr.Set(ports.KeyAPIKey, apiKey); err != nil {
		return common.WrapSaveError("API key", err)
	}
	fmt.Println("✓ Migrated API key")

	// Migrate optional credentials
	if clientID, err := fileStore.Get(ports.KeyClientID); err == nil && clientID != "" {
		if err := kr.Set(ports.KeyClientID, clientID); err == nil {
			fmt.Println("✓ Migrated client ID")
		}
	}

	if clientSecret, err := fileStore.Get(ports.KeyClientSecret); err == nil && clientSecret != "" {
		if err := kr.Set(ports.KeyClientSecret, clientSecret); err == nil {
			fmt.Println("✓ Migrated client secret")
		}
	}

	// Migrate grants data
	if grants, err := fileStore.Get("grants"); err == nil && grants != "" {
		if err := kr.Set("grants", grants); err == nil {
			fmt.Println("✓ Migrated grants")
		}
	}

	if defaultGrant, err := fileStore.Get("default_grant"); err == nil && defaultGrant != "" {
		if err := kr.Set("default_grant", defaultGrant); err == nil {
			fmt.Println("✓ Migrated default grant")
		}
	}

	fmt.Println()
	fmt.Println("Migration complete! Your credentials are now stored in the system keyring.")
	fmt.Println("Run 'nylas doctor' to verify.")

	return nil
}
