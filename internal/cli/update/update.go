package update

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nylas/cli/internal/cli"
	"github.com/nylas/cli/internal/cli/common"
	"github.com/nylas/cli/internal/domain"
)

// NewUpdateCmd creates the update command.
func NewUpdateCmd() *cobra.Command {
	var checkOnly bool
	var force bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Nylas CLI to the latest version",
		Long: `Check for and install the latest version of Nylas CLI from GitHub releases.

This command will:
1. Check GitHub for the latest release
2. Compare with your current version
3. Download and verify the new binary
4. Replace the current binary with the new one`,
		Example: `  # Check for updates and install if available
  nylas update

  # Only check for updates without installing
  nylas update --check

  # Force update even if already on latest version
  nylas update --force

  # Skip confirmation prompt
  nylas update --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd.Context(), checkOnly, force, yes)
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates without installing")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force update even if already on latest version")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runUpdate(ctx context.Context, checkOnly, force, yes bool) error {
	currentVersion := cli.Version

	fmt.Printf("Current version: %s\n", currentVersion)

	// Check if installed via Homebrew
	if isHomebrewInstall() {
		fmt.Println("\nNylas CLI was installed via Homebrew.")
		fmt.Println("To update, run: brew upgrade nylas")
		return nil
	}

	// Fetch latest release
	fmt.Println("Checking for updates...")

	release, err := getLatestRelease(ctx)
	if err != nil {
		return common.WrapGetError("updates", err)
	}

	latestVersion := parseVersion(release.TagName)
	fmt.Printf("Latest version:  %s\n", latestVersion)

	// Compare versions
	if !force && !isUpdateAvailable(currentVersion, latestVersion) {
		fmt.Println("\nYou are already running the latest version.")
		return nil
	}

	if checkOnly {
		if isUpdateAvailable(currentVersion, latestVersion) {
			fmt.Println("\nUpdate available! Run 'nylas update' to install.")
			fmt.Printf("Release notes: %s\n", release.HTMLURL)
		}
		return nil
	}

	// Show update info
	fmt.Println("\nA new version is available!")
	fmt.Printf("Release notes: %s\n", release.HTMLURL)

	// Confirm update
	if !yes {
		fmt.Print("\nDo you want to update? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read input: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Update cancelled.")
			return nil
		}
	}

	// Find the right asset for this platform
	assetName := getAssetName(latestVersion)
	asset := findAsset(release, assetName)
	if asset == nil {
		return fmt.Errorf("no download available for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download the asset
	fmt.Printf("\nDownloading %s...\n", assetName)
	archivePath, err := downloadFile(ctx, asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = os.Remove(archivePath) }()

	// Download and verify checksum
	checksumAsset := findChecksumAsset(release)
	if checksumAsset != nil {
		fmt.Println("Verifying checksum...")
		checksums, err := downloadChecksums(ctx, checksumAsset.BrowserDownloadURL)
		if err != nil {
			return common.WrapDownloadError("checksums", err)
		}

		expectedChecksum, ok := checksums[assetName]
		if !ok {
			return fmt.Errorf("checksum not found for %s", assetName)
		}

		valid, err := verifyChecksum(archivePath, expectedChecksum)
		if err != nil {
			return fmt.Errorf("checksum verification error: %w", err)
		}
		if !valid {
			return fmt.Errorf("checksum verification failed - download may be corrupted")
		}
		fmt.Println("Checksum verified.")
	}

	// Extract binary
	fmt.Println("Extracting...")
	binaryPath, err := extractBinary(archivePath, runtime.GOOS)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}
	defer func() { _ = os.Remove(binaryPath) }()

	// Get current binary path
	currentBinaryPath, err := getCurrentBinaryPath()
	if err != nil {
		return common.WrapGetError("current binary", err)
	}

	// Install new binary
	fmt.Println("Installing...")
	if err := installBinary(binaryPath, currentBinaryPath); err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Printf("\nSuccessfully updated to %s!\n", latestVersion)
	fmt.Println("Run 'nylas version' to verify.")

	return nil
}

// CheckForUpdateAsync checks for updates in the background and prints a message if available.
// This can be called during CLI startup for non-blocking update notifications.
func CheckForUpdateAsync(currentVersion string) {
	go func() {
		ctx, cancel := common.CreateContextWithTimeout(domain.TimeoutQuickCheck)
		defer cancel()

		release, err := getLatestRelease(ctx)
		if err != nil {
			return // Silently ignore errors in async check
		}

		latestVersion := parseVersion(release.TagName)
		if isUpdateAvailable(currentVersion, latestVersion) {
			fmt.Printf("\nA new version of Nylas CLI is available: %s (current: %s)\n", latestVersion, currentVersion)
			fmt.Println("Run 'nylas update' to upgrade.")
		}
	}()
}
