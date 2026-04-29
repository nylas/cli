package air

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmailListNoMaxHeightConstraint is a regression test to ensure that the
// email list is not constrained by the accessibility-aria.css [role="listbox"] rule.
//
// Bug history: The email list had role="listbox" for accessibility, but the
// generic [role="listbox"] selector in accessibility.css had max-height: 300px,
// which prevented the email list from filling the viewport height.
//
// Fix: Changed the selector to [role="listbox"]:not(.email-list) to exclude
// the email list from this constraint.
func TestEmailListNoMaxHeightConstraint(t *testing.T) {
	t.Parallel()

	// Read the accessibility-aria.css file (ARIA roles are defined here)
	cssPath := filepath.Join("static", "css", "accessibility-aria.css")
	// #nosec G304 -- test reading project CSS file, path is hardcoded
	content, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read accessibility-aria.css: %v", err)
	}

	cssContent := string(content)

	// Verify that the [role="listbox"] selector excludes .email-list
	if !strings.Contains(cssContent, `[role="listbox"]:not(.email-list)`) {
		t.Error("accessibility-aria.css must use [role=\"listbox\"]:not(.email-list) to exclude email list from max-height constraint")
	}

	// Verify the old broken selector is not present
	if strings.Contains(cssContent, `[role="listbox"] {`) && !strings.Contains(cssContent, `:not(.email-list)`) {
		t.Error("Found [role=\"listbox\"] without :not(.email-list) - this will constrain email list to 300px!")
	}
}

// TestEmailListContainerUsesGrid verifies that email-list-container uses CSS Grid
// instead of flexbox for more reliable height calculation.
func TestEmailListContainerUsesGrid(t *testing.T) {
	t.Parallel()

	// Read the email-list.css file
	cssPath := filepath.Join("static", "css", "email-list.css")
	// #nosec G304 -- test reading project CSS file, path is hardcoded
	content, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read email-list.css: %v", err)
	}

	cssContent := string(content)

	// Verify .email-list-container uses CSS Grid
	if !strings.Contains(cssContent, "display: grid") {
		t.Error("email-list-container must use 'display: grid' for proper layout")
	}

	if !strings.Contains(cssContent, "grid-template-rows: auto 1fr") {
		t.Error("email-list-container must use 'grid-template-rows: auto 1fr' to size header and list")
	}
}

// TestEmailViewUsesGrid verifies that email-view uses CSS Grid for layout.
func TestEmailViewUsesGrid(t *testing.T) {
	t.Parallel()

	// Read the calendar-grid.css file (where email-view.active is defined)
	cssPath := filepath.Join("static", "css", "calendar-grid.css")
	// #nosec G304 -- test reading project CSS file, path is hardcoded
	content, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read calendar-grid.css: %v", err)
	}

	cssContent := string(content)

	// Verify .email-view.active uses CSS Grid
	if !strings.Contains(cssContent, ".email-view.active") {
		t.Error("calendar-grid.css must define .email-view.active")
	}

	// Check for grid layout (need to find the rule and verify it has display: grid)
	emailViewActiveIndex := strings.Index(cssContent, ".email-view.active")
	if emailViewActiveIndex == -1 {
		t.Fatal("Could not find .email-view.active in calendar-grid.css")
	}

	// Get the next 500 characters after .email-view.active to check for grid properties
	snippet := cssContent[emailViewActiveIndex : emailViewActiveIndex+500]

	if !strings.Contains(snippet, "display: grid") {
		t.Error("email-view.active must use 'display: grid'")
	}

	if !strings.Contains(snippet, "grid-template-columns") {
		t.Error("email-view.active must define grid-template-columns for sidebar|email-list|preview layout")
	}
}

// TestMainLayoutHasExplicitHeight verifies that main-layout has an explicit height
// calculation to ensure proper flexbox/grid sizing.
func TestMainLayoutHasExplicitHeight(t *testing.T) {
	t.Parallel()

	// Read the layout.css file
	cssPath := filepath.Join("static", "css", "layout.css")
	// #nosec G304 -- test reading project CSS file, path is hardcoded
	content, err := os.ReadFile(cssPath)
	if err != nil {
		t.Fatalf("Failed to read layout.css: %v", err)
	}

	cssContent := string(content)

	// Verify .main-layout has height calculation
	if !strings.Contains(cssContent, "calc(100vh") {
		t.Error("main-layout should use calc(100vh - ...) for explicit height calculation")
	}
}

func TestAccountDropdownIsViewportBoundedAndScrollable(t *testing.T) {
	t.Parallel()

	cssContent, err := staticFiles.ReadFile("static/css/components-account.css")
	if err != nil {
		t.Fatalf("failed to read components-account.css: %v", err)
	}

	rule := cssRule(t, string(cssContent), ".account-dropdown")
	required := []string{
		"max-height:",
		"overflow-y: auto",
		"overscroll-behavior: contain",
	}
	for _, declaration := range required {
		if !strings.Contains(rule, declaration) {
			t.Errorf(".account-dropdown must include %q so long account lists fit the viewport and scroll", declaration)
		}
	}
}

func cssRule(t *testing.T, css, selector string) string {
	t.Helper()

	start := strings.Index(css, selector+" {")
	if start == -1 {
		t.Fatalf("missing CSS rule for %s", selector)
	}
	end := strings.Index(css[start:], "}")
	if end == -1 {
		t.Fatalf("unterminated CSS rule for %s", selector)
	}
	return css[start : start+end]
}
