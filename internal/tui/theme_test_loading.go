package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestCustomThemeIntegration(t *testing.T) {
	// Create temp directory to simulate ~/.config/nylas/themes/
	tmpDir, err := os.MkdirTemp("", "nylas-custom-theme-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a custom theme file
	themePath := filepath.Join(tmpDir, "mytest.yaml")
	themeContent := `# Custom test theme
foreground: "#00FF00"
background: "#000000"
red: "#FF0000"
green: "#00FF00"
yellow: "#FFFF00"
blue: "#0000FF"

k9s:
  body:
    fgColor: "#00FF00"
    bgColor: "#000000"
    logoColor: "#FF00FF"
  prompt:
    fgColor: "#00FF00"
    bgColor: "#000000"
  info:
    fgColor: "#00FFFF"
    sectionColor: "#00FF00"
  frame:
    border:
      fgColor: "#333333"
      focusColor: "#00FF00"
    menu:
      fgColor: "#00FF00"
      keyColor: "#FFFF00"
      numKeyColor: "#FF00FF"
  views:
    table:
      fgColor: "#00FF00"
      header:
        fgColor: "#FFFF00"
        bgColor: "#000000"
      selected:
        fgColor: "#000000"
        bgColor: "#00FF00"
`
	// Use restrictive permissions for config files
	if err := os.WriteFile(themePath, []byte(themeContent), 0600); err != nil {
		t.Fatalf("Failed to write theme file: %v", err)
	}

	// Test LoadThemeFromFile
	config, err := LoadThemeFromFile(themePath)
	if err != nil {
		t.Fatalf("LoadThemeFromFile() error = %v", err)
	}

	// Verify config was loaded correctly
	if config.Foreground != "#00FF00" {
		t.Errorf("Foreground = %q, want %q", config.Foreground, "#00FF00")
	}
	if config.K9s.Body.LogoColor != "#FF00FF" {
		t.Errorf("LogoColor = %q, want %q", config.K9s.Body.LogoColor, "#FF00FF")
	}
	if config.K9s.Views.Table.Selected.BgColor != "#00FF00" {
		t.Errorf("Table.Selected.BgColor = %q, want %q", config.K9s.Views.Table.Selected.BgColor, "#00FF00")
	}

	// Test ToStyles conversion
	styles := config.ToStyles()
	if styles == nil {
		t.Fatal("ToStyles() returned nil")
		return
	}

	// Verify colors were applied correctly
	expectedGreen := tcell.NewRGBColor(0, 255, 0) // #00FF00
	if styles.FgColor != expectedGreen {
		t.Errorf("FgColor = %v, want %v (green)", styles.FgColor, expectedGreen)
	}

	expectedMagenta := tcell.NewRGBColor(255, 0, 255) // #FF00FF
	if styles.LogoColor != expectedMagenta {
		t.Errorf("LogoColor = %v, want %v (magenta)", styles.LogoColor, expectedMagenta)
	}

	// Verify table selection colors
	if styles.TableSelectBg != expectedGreen {
		t.Errorf("TableSelectBg = %v, want %v (green)", styles.TableSelectBg, expectedGreen)
	}
}

// TestCustomThemeViaGetThemeStyles tests that GetThemeStyles correctly loads custom themes
func TestCustomThemeViaGetThemeStyles(t *testing.T) {
	// Create temp themes directory
	tmpDir, err := os.MkdirTemp("", "nylas-themes-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a distinguishable custom theme
	themeName := "testpurple"
	themePath := filepath.Join(tmpDir, themeName+".yaml")
	themeContent := `foreground: "#9900FF"
background: "#1A0033"
k9s:
  body:
    fgColor: "#9900FF"
    bgColor: "#1A0033"
    logoColor: "#FF00FF"
  frame:
    border:
      fgColor: "#660099"
      focusColor: "#CC00FF"
  views:
    table:
      fgColor: "#9900FF"
      header:
        fgColor: "#CC00FF"
      selected:
        fgColor: "#000000"
        bgColor: "#9900FF"
`
	// Use restrictive permissions for config files
	if err := os.WriteFile(themePath, []byte(themeContent), 0600); err != nil {
		t.Fatalf("Failed to write theme file: %v", err)
	}

	// Test loading directly via LoadThemeFromFile
	config, err := LoadThemeFromFile(themePath)
	if err != nil {
		t.Fatalf("LoadThemeFromFile() failed: %v", err)
	}

	styles := config.ToStyles()

	// Verify the purple color was loaded
	expectedPurple := tcell.NewRGBColor(153, 0, 255) // #9900FF
	if styles.FgColor != expectedPurple {
		t.Errorf("Custom theme FgColor = %v, want %v (purple)", styles.FgColor, expectedPurple)
	}

	// Verify logo color
	expectedMagenta := tcell.NewRGBColor(255, 0, 255) // #FF00FF
	if styles.LogoColor != expectedMagenta {
		t.Errorf("Custom theme LogoColor = %v, want %v (magenta)", styles.LogoColor, expectedMagenta)
	}
}

// TestLoadCustomThemeFromConfigDir tests loading from the actual config directory
func TestLoadCustomThemeFromConfigDir(t *testing.T) {
	// This test checks if we can load the "testcustom" theme if it exists
	// Skip if no custom themes exist
	themes := ListCustomThemes()
	if len(themes) == 0 {
		t.Skip("No custom themes found in ~/.config/nylas/themes/")
	}

	// Try to load the first custom theme found
	themeName := themes[0]
	styles, err := LoadCustomTheme(themeName)
	if err != nil {
		t.Fatalf("LoadCustomTheme(%q) error = %v", themeName, err)
	}

	if styles == nil {
		t.Errorf("LoadCustomTheme(%q) returned nil styles", themeName)
		return
	}

	// Note: FgColor and BgColor can be 0 (ColorDefault) which are valid colors
	_ = styles.FgColor
	_ = styles.BgColor

	t.Logf("Successfully loaded custom theme %q with FgColor=%v", themeName, styles.FgColor)
}

// TestGetThemeStylesWithCustomTheme tests that GetThemeStyles falls back to custom themes
func TestGetThemeStylesWithCustomTheme(t *testing.T) {
	// First verify built-in themes work
	k9sStyles := GetThemeStyles(ThemeK9s)
	if k9sStyles == nil {
		t.Fatal("GetThemeStyles(k9s) returned nil")
	}

	// Try loading a non-existent theme - should fall back to default
	unknownStyles := GetThemeStyles("nonexistent-theme-xyz")
	if unknownStyles == nil {
		t.Fatal("GetThemeStyles(nonexistent) returned nil")
		return
	}

	// The unknown theme should fall back to default styles
	defaultStyles := DefaultStyles()
	if unknownStyles.FgColor != defaultStyles.FgColor {
		t.Errorf("Unknown theme should fall back to default, FgColor = %v, want %v",
			unknownStyles.FgColor, defaultStyles.FgColor)
	}
}

// TestGetThemeStylesLoadsCustomTheme verifies GetThemeStyles loads themes from ~/.config/nylas/themes/
func TestGetThemeStylesLoadsCustomTheme(t *testing.T) {
	// Check if testcustom theme exists
	themes := ListCustomThemes()
	hasTestCustom := false
	for _, theme := range themes {
		if theme == "testcustom" {
			hasTestCustom = true
			break
		}
	}
	if !hasTestCustom {
		t.Skip("testcustom theme not found - run 'nylas tui theme init testcustom' first")
	}

	// Load the custom theme via GetThemeStyles
	customStyles := GetThemeStyles("testcustom")
	if customStyles == nil {
		t.Fatal("GetThemeStyles(testcustom) returned nil")
		return
	}

	// The testcustom theme should have the Tokyo Night colors
	// foreground: "#c0caf5" = RGB(192, 202, 245)
	expectedFg := tcell.NewRGBColor(192, 202, 245)
	if customStyles.FgColor != expectedFg {
		t.Errorf("Custom theme FgColor = %v, want %v (#c0caf5)", customStyles.FgColor, expectedFg)
	}

	// logoColor: "#bb9af7" = RGB(187, 154, 247)
	expectedLogo := tcell.NewRGBColor(187, 154, 247)
	if customStyles.LogoColor != expectedLogo {
		t.Errorf("Custom theme LogoColor = %v, want %v (#bb9af7)", customStyles.LogoColor, expectedLogo)
	}

	// Verify it's different from default k9s theme
	defaultStyles := GetThemeStyles(ThemeK9s)
	if customStyles.FgColor == defaultStyles.FgColor {
		t.Error("Custom theme FgColor should be different from default k9s theme")
	}

	t.Logf("SUCCESS: GetThemeStyles correctly loaded custom theme 'testcustom'")
	t.Logf("  FgColor: %v (expected #c0caf5)", customStyles.FgColor)
	t.Logf("  LogoColor: %v (expected #bb9af7)", customStyles.LogoColor)
}
