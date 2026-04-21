// Package air provides a modern web-based email client interface for the Nylas CLI.
// Air is designed to be a lightweight, keyboard-driven email client that runs locally.
package air

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	browserpkg "github.com/nylas/cli/internal/adapters/browser"
	"github.com/nylas/cli/internal/air/cache"
)

// NewAirCmd creates the air command.
func NewAirCmd() *cobra.Command {
	var (
		port       int
		noBrowser  bool
		clearCache bool
		encrypted  bool
	)

	cmd := &cobra.Command{
		Use:   "air",
		Short: "Launch the modern web email client",
		Long: `Launch Nylas Air - a modern, keyboard-driven email client that runs in your browser.

Air provides:
  - Three-pane email interface (folders, list, preview)
  - Calendar and contacts views
  - Keyboard shortcuts (J/K navigate, C compose, E archive)
  - Command palette (Cmd+K)
  - Dark mode with customizable themes
  - AI-powered features (summaries, smart replies)
  - Local caching with full-text search
  - Offline support with action queuing
  - Optional encryption for cached data

The client runs locally on your machine for privacy and performance.`,
		Example: `  # Launch Air on default port (7365)
  nylas air

  # Launch on custom port
  nylas air --port 8080

  # Launch without opening browser
  nylas air --no-browser

  # Clear all cached data before starting
  nylas air --clear-cache

  # Enable encryption for cached data
  nylas air --encrypted`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get cache base path
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("get home directory: %w", err)
			}
			basePath := filepath.Join(homeDir, ".config", "nylas", "air")

			// Load or create settings
			settings, err := cache.LoadSettings(basePath)
			if err != nil {
				return fmt.Errorf("load settings: %w", err)
			}

			// Update encryption setting if flag provided
			if cmd.Flags().Changed("encrypted") {
				if err := settings.SetEncryption(encrypted); err != nil {
					return fmt.Errorf("update encryption setting: %w", err)
				}
			}

			// Clear cache if requested
			if clearCache {
				fmt.Println("Clearing cache...")
				mgr, err := cache.NewManager(settings.ToConfig(basePath))
				if err != nil {
					return fmt.Errorf("create cache manager: %w", err)
				}
				if err := mgr.ClearAllCaches(); err != nil {
					return fmt.Errorf("clear cache: %w", err)
				}
				fmt.Println("Cache cleared successfully")
			}

			addr := fmt.Sprintf("localhost:%d", port)
			url := fmt.Sprintf("http://%s", addr)

			// Add init flag when cache was just cleared (triggers loading overlay)
			browserURL := url
			if clearCache {
				browserURL = fmt.Sprintf("%s?init=1", url)
			}

			fmt.Printf("Starting Nylas Air at %s\n", url)
			if settings.IsEncryptionEnabled() {
				fmt.Println("Encryption: enabled (keys stored in system keyring)")
			}
			fmt.Println("Press Ctrl+C to stop")
			fmt.Println()

			// Open browser unless disabled
			if !noBrowser {
				b := browserpkg.NewDefaultBrowser()
				if err := b.Open(browserURL); err != nil {
					fmt.Printf("Could not open browser: %v\n", err)
					fmt.Printf("Please open %s manually\n", url)
				}
			}

			// Start the server (blocks until interrupted)
			server, err := NewServer(addr)
			if err != nil {
				return fmt.Errorf("create air server: %w", err)
			}
			return server.Start()
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 7365, "Port to run the server on")
	cmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Don't open browser automatically")
	cmd.Flags().BoolVar(&clearCache, "clear-cache", false, "Clear all cached data before starting")
	cmd.Flags().BoolVar(&encrypted, "encrypted", false, "Enable encryption for cached data (uses system keyring)")

	return cmd
}
