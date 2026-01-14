package mcp

import (
	"runtime"
	"testing"
)

func TestGetAssistantByID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id       string
		wantName string
		wantNil  bool
	}{
		{"claude-desktop", "Claude Desktop", false},
		{"claude-code", "Claude Code", false},
		{"cursor", "Cursor", false},
		{"windsurf", "Windsurf", false},
		{"vscode", "VS Code", false},
		{"unknown", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			a := GetAssistantByID(tt.id)
			if tt.wantNil {
				if a != nil {
					t.Errorf("expected nil, got %v", a)
				}
				return
			}
			if a == nil {
				t.Fatal("expected assistant, got nil")
			}
			if a.Name != tt.wantName {
				t.Errorf("expected name %s, got %s", tt.wantName, a.Name)
			}
		})
	}
}

func TestAssistant_GetConfigPath(t *testing.T) {
	t.Parallel()

	for _, a := range Assistants {
		t.Run(a.ID, func(t *testing.T) {
			path := a.GetConfigPath()
			if path == "" {
				t.Error("expected non-empty config path")
			}

			// Verify path doesn't contain unexpanded ~
			if len(path) > 0 && path[0] == '~' {
				t.Error("path should have ~ expanded")
			}

			// Verify path doesn't contain unexpanded env vars
			if runtime.GOOS == "windows" {
				// On Windows, check for %VAR% pattern
				for i := 0; i < len(path); i++ {
					if path[i] == '%' {
						t.Error("path should have environment variables expanded")
						break
					}
				}
			}
		})
	}
}

func TestAssistant_IsProjectConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id     string
		isProj bool
	}{
		{"claude-desktop", false},
		{"claude-code", false},
		{"cursor", false},
		{"windsurf", false},
		{"vscode", true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			a := GetAssistantByID(tt.id)
			if a == nil {
				t.Fatalf("assistant %s not found", tt.id)
			}
			if a.IsProjectConfig() != tt.isProj {
				t.Errorf("IsProjectConfig() = %v, want %v", a.IsProjectConfig(), tt.isProj)
			}
		})
	}
}

func TestExpandPath(t *testing.T) {
	t.Parallel()

	// Test that ~ is expanded
	path := expandPath("~/test")
	if len(path) > 0 && path[0] == '~' {
		t.Error("~ should be expanded")
	}

	// Test that regular paths are unchanged
	path = expandPath("/usr/local/bin")
	if path != "/usr/local/bin" {
		t.Errorf("expected /usr/local/bin, got %s", path)
	}
}

func TestAssistantsHaveRequiredFields(t *testing.T) {
	t.Parallel()

	for _, a := range Assistants {
		t.Run(a.ID, func(t *testing.T) {
			if a.Name == "" {
				t.Error("Name should not be empty")
			}
			if a.ID == "" {
				t.Error("ID should not be empty")
			}
			if len(a.ConfigPaths) == 0 {
				t.Error("ConfigPaths should not be empty")
			}

			// Check that darwin config path exists
			if _, ok := a.ConfigPaths["darwin"]; !ok {
				t.Error("should have darwin config path")
			}
		})
	}
}
