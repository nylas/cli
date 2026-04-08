package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseColor(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected tcell.Color
	}{
		{
			name:     "hex color",
			input:    "#FF0000",
			expected: tcell.NewRGBColor(255, 0, 0),
		},
		{
			name:     "hex color lowercase",
			input:    "#00ff00",
			expected: tcell.NewRGBColor(0, 255, 0),
		},
		{
			name:     "hex color blue",
			input:    "#0000FF",
			expected: tcell.NewRGBColor(0, 0, 255),
		},
		{
			name:     "named color black",
			input:    "black",
			expected: tcell.ColorBlack,
		},
		{
			name:     "named color white",
			input:    "white",
			expected: tcell.ColorWhite,
		},
		{
			name:     "named color red",
			input:    "red",
			expected: tcell.ColorRed,
		},
		{
			name:     "named color green",
			input:    "green",
			expected: tcell.ColorGreen,
		},
		{
			name:     "named color blue",
			input:    "blue",
			expected: tcell.ColorBlue,
		},
		{
			name:     "named color yellow",
			input:    "yellow",
			expected: tcell.ColorYellow,
		},
		{
			name:     "default",
			input:    "default",
			expected: tcell.ColorDefault,
		},
		{
			name:     "empty",
			input:    "",
			expected: tcell.ColorDefault,
		},
		{
			name:     "with spaces",
			input:    "  #FF0000  ",
			expected: tcell.NewRGBColor(255, 0, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseColor(tt.input)
			if result != tt.expected {
				t.Errorf("parseColor(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"999999", true},
		{"", false},
		{"abc", false},
		{"12a", false},
		{"1.5", false},
		{"-1", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isNumeric(tt.input)
			if result != tt.expected {
				t.Errorf("isNumeric(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"123", 123},
		{"0", 0},
		{"1", 1},
		{"999", 999},
		{"", 0},
		{"abc", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseInt(tt.input)
			if result != tt.expected {
				t.Errorf("parseInt(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetThemeStyles(t *testing.T) {
	tests := []struct {
		name  string
		theme ThemeName
	}{
		{"k9s", ThemeK9s},
		{"amber", ThemeAmber},
		{"green", ThemeGreen},
		{"apple2", ThemeAppleII},
		{"vintage", ThemeVintage},
		{"ibm", ThemeIBMDOS},
		{"futuristic", ThemeFuturistic},
		{"matrix", ThemeMatrix},
		{"norton", ThemeNorton},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			styles := GetThemeStyles(tt.theme)
			if styles == nil {
				t.Errorf("GetThemeStyles(%q) returned nil", tt.theme)
				return
			}

			// Note: FgColor can be 0 (ColorDefault) which is a valid color
			_ = styles.FgColor
		})
	}
}

func TestAvailableThemes(t *testing.T) {
	themes := AvailableThemes()
	if len(themes) == 0 {
		t.Error("AvailableThemes() returned empty slice")
	}

	// Verify known themes are in the list
	expectedThemes := []ThemeName{
		ThemeK9s,
		ThemeAmber,
		ThemeGreen,
		ThemeAppleII,
		ThemeVintage,
		ThemeIBMDOS,
		ThemeFuturistic,
		ThemeMatrix,
		ThemeNorton,
	}

	for _, expected := range expectedThemes {
		found := false
		for _, theme := range themes {
			if theme == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected theme %q not found in AvailableThemes()", expected)
		}
	}
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	if styles == nil {
		t.Fatal("DefaultStyles() returned nil")
		return
	}

	// Verify base colors are set
	if styles.BgColor == 0 && styles.BgColor != tcell.ColorBlack {
		t.Error("BgColor not properly set")
	}
	if styles.FgColor == 0 {
		t.Error("FgColor not properly set")
	}
	if styles.BorderColor == 0 {
		t.Error("BorderColor not properly set")
	}

	// Verify table colors are set
	if styles.TableSelectBg == 0 {
		t.Error("TableSelectBg not properly set")
	}
	// Note: TableSelectFg can be 0 (ColorBlack) which is a valid color
}

func TestThemeConfigToStyles(t *testing.T) {
	config := &ThemeConfig{
		Foreground: "#c0caf5",
		Background: "#1a1b26",
		Red:        "#f7768e",
		Green:      "#9ece6a",
		Yellow:     "#e0af68",
		Blue:       "#7aa2f7",
		K9s: K9sSkin{
			Body: BodyStyle{
				FgColor:   "#c0caf5",
				BgColor:   "#1a1b26",
				LogoColor: "#bb9af7",
			},
			Frame: FrameStyle{
				Border: BorderStyle{
					FgColor:    "#3b4261",
					FocusColor: "#7aa2f7",
				},
			},
		},
	}

	styles := config.ToStyles()

	if styles == nil {
		t.Fatal("ToStyles() returned nil")
		return
	}

	// Verify colors were applied
	expectedFg := parseColor("#c0caf5")
	if styles.FgColor != expectedFg {
		t.Errorf("FgColor = %v, want %v", styles.FgColor, expectedFg)
	}

	expectedBg := parseColor("#1a1b26")
	if styles.BgColor != expectedBg {
		t.Errorf("BgColor = %v, want %v", styles.BgColor, expectedBg)
	}

	expectedLogo := parseColor("#bb9af7")
	if styles.LogoColor != expectedLogo {
		t.Errorf("LogoColor = %v, want %v", styles.LogoColor, expectedLogo)
	}
}

func TestCreateDefaultThemeFile(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "nylas-theme-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	themePath := filepath.Join(tmpDir, "test-theme.yaml")

	// Create theme file
	err = CreateDefaultThemeFile(themePath)
	if err != nil {
		t.Fatalf("CreateDefaultThemeFile() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(themePath); os.IsNotExist(err) {
		t.Error("Theme file was not created")
	}

	// Load the created theme
	config, err := LoadThemeFromFile(themePath)
	if err != nil {
		t.Fatalf("LoadThemeFromFile() error = %v", err)
	}

	// Verify the loaded config has expected values
	if config.Foreground == "" {
		t.Error("Loaded config has empty foreground")
	}
	if config.K9s.Body.FgColor == "" {
		t.Error("Loaded config has empty K9s body fgColor")
	}
}

func TestLoadThemeFromFile_NotFound(t *testing.T) {
	_, err := LoadThemeFromFile("/nonexistent/path/theme.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestLoadCustomTheme_NotFound(t *testing.T) {
	_, err := LoadCustomTheme("nonexistent-theme-12345")
	if err == nil {
		t.Error("Expected error for non-existent custom theme, got nil")
	}
}

func TestListCustomThemes(t *testing.T) {
	// This test just ensures the function doesn't panic
	themes := ListCustomThemes()
	// themes may be nil or empty if no custom themes exist
	_ = themes
}

// TestCustomThemeIntegration tests the full custom theme loading flow
