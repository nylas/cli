package ui

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// =============================================================================
// Template Loading Tests
// =============================================================================

func TestLoadTemplates(t *testing.T) {
	t.Parallel()

	tmpl, err := loadTemplates()
	if err != nil {
		t.Fatalf("loadTemplates() failed: %v", err)
	}

	if tmpl == nil {
		t.Fatal("loadTemplates() returned nil template")
	}

	// Verify templates are loaded (ParseFS uses base filename as name)
	templates := tmpl.Templates()
	if len(templates) == 0 {
		t.Error("No templates loaded")
	}

	// Log loaded template names for debugging
	var names []string
	for _, tpl := range templates {
		if tpl.Name() != "" {
			names = append(names, tpl.Name())
		}
	}

	// Verify we have some templates
	if len(names) < 3 {
		t.Errorf("Expected at least 3 templates, got %d: %v", len(names), names)
	}
}

func TestLoadTemplates_FunctionsAvailable(t *testing.T) {
	t.Parallel()

	tmpl, err := loadTemplates()
	if err != nil {
		t.Fatalf("loadTemplates() failed: %v", err)
	}

	// Test that template functions work by executing a simple template
	testTmpl, err := tmpl.New("test").Parse(`{{ upper "hello" }}`)
	if err != nil {
		t.Fatalf("Failed to parse test template: %v", err)
	}

	var buf bytes.Buffer
	if err := testTmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}

	if buf.String() != "HELLO" {
		t.Errorf("Expected 'HELLO', got %q", buf.String())
	}
}

// TestBaseTemplate_InitialStateIsParseableJSON verifies the CSP-safe
// <script type="application/json"> data block survives html/template's
// contextual escaping as valid JSON, including breakout payloads in the data.
// The strict CSP (script-src 'self') blocks inline executable scripts, so the
// UI depends on this block being parseable by JSON.parse in app.js.
func TestBaseTemplate_InitialStateIsParseableJSON(t *testing.T) {
	t.Parallel()

	tmpl, err := loadTemplates()
	if err != nil {
		t.Fatalf("loadTemplates() failed: %v", err)
	}

	data := PageData{
		Configured:   true,
		ClientID:     `client-"</script><script>alert(1)</script>`,
		Region:       "us",
		HasAPIKey:    true,
		DefaultGrant: "grant-1",
		Grants: []Grant{
			{ID: "grant-1", Email: `evil"@example.com`, Provider: "google"},
		},
		Commands: GetDefaultCommands(),
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		t.Fatalf("ExecuteTemplate(base) failed: %v", err)
	}
	html := buf.String()

	if strings.Contains(html, "window.__INITIAL_STATE__") {
		t.Error("inline executable initial-state script still present (blocked by CSP)")
	}

	re := regexp.MustCompile(`(?s)<script type="application/json" id="initial-state">(.*?)</script>`)
	m := re.FindStringSubmatch(html)
	if m == nil {
		t.Fatal("initial-state JSON data block not found in rendered page")
	}

	var state struct {
		Configured   bool    `json:"configured"`
		ClientID     string  `json:"clientID"`
		Region       string  `json:"region"`
		DefaultGrant string  `json:"defaultGrant"`
		Grants       []Grant `json:"grants"`
	}
	if err := json.Unmarshal([]byte(m[1]), &state); err != nil {
		t.Fatalf("initial-state block is not valid JSON: %v\nblock: %s", err, m[1])
	}

	if !state.Configured || state.Region != "us" || state.DefaultGrant != "grant-1" {
		t.Errorf("initial state round-trip mismatch: %+v", state)
	}
	if state.ClientID != data.ClientID {
		t.Errorf("ClientID round-trip mismatch: got %q want %q", state.ClientID, data.ClientID)
	}
}

// =============================================================================
// Template Function Tests
// =============================================================================

func TestTemplateFuncs_Upper(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "HELLO"},
		{"HELLO", "HELLO"},
		{"Hello World", "HELLO WORLD"},
		{"", ""},
		{"123abc", "123ABC"},
	}

	upperFn := templateFuncs["upper"].(func(string) string)

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := upperFn(tt.input)
			if result != tt.expected {
				t.Errorf("upper(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTemplateFuncs_Lower(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"HELLO", "hello"},
		{"hello", "hello"},
		{"Hello World", "hello world"},
		{"", ""},
		{"123ABC", "123abc"},
	}

	lowerFn := templateFuncs["lower"].(func(string) string)

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := lowerFn(tt.input)
			if result != tt.expected {
				t.Errorf("lower(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTemplateFuncs_Slice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		start    int
		end      int
		expected string
	}{
		{"normal slice", "hello", 0, 3, "hel"},
		{"full string", "hello", 0, 5, "hello"},
		{"middle slice", "hello", 1, 4, "ell"},
		{"empty result", "hello", 2, 2, ""},
		{"start beyond length", "hello", 10, 15, ""},
		{"end beyond length", "hello", 0, 100, "hello"},
		{"empty string", "", 0, 0, ""},
		{"unicode string bytes", "héllo", 0, 3, "hé"}, // slice works on bytes, not runes (é = 2 bytes)
		{"single char", "a", 0, 1, "a"},
	}

	sliceFn := templateFuncs["slice"].(func(string, int, int) string)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sliceFn(tt.input, tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("slice(%q, %d, %d) = %q, want %q",
					tt.input, tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

// =============================================================================
