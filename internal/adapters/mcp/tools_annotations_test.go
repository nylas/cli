package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestAnnotateTools_AllToolsHaveAnnotations verifies every tool has annotations after annotateTools().
func TestAnnotateTools_AllToolsHaveAnnotations(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()

			if tool.Annotations == nil {
				t.Errorf("tool %q has nil annotations", tool.Name)
			}
		})
	}
}

// TestAnnotateTools_AllToolsHaveTitles verifies every tool has a non-empty title.
func TestAnnotateTools_AllToolsHaveTitles(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()

			if tool.Title == "" {
				t.Errorf("tool %q has empty title", tool.Name)
			}
		})
	}
}

// TestAnnotateTools_ReadOnlyNoDestructive verifies read-only tools don't have destructiveHint.
func TestAnnotateTools_ReadOnlyNoDestructive(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			t.Parallel()

			if tool.Annotations == nil {
				return
			}
			if tool.Annotations.ReadOnlyHint != nil && *tool.Annotations.ReadOnlyHint {
				if tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint {
					t.Errorf("tool %q is both readOnly and destructive", tool.Name)
				}
			}
		})
	}
}

// TestAnnotateTools_ReadOnlyTools verifies list_*/get_* tools are read-only and idempotent.
func TestAnnotateTools_ReadOnlyTools(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		name := tool.Name
		if !strings.HasPrefix(name, "list_") && !strings.HasPrefix(name, "get_") {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			a := tool.Annotations
			if a == nil {
				t.Fatalf("tool %q has nil annotations", name)
			}
			if a.ReadOnlyHint == nil || !*a.ReadOnlyHint {
				t.Errorf("tool %q should have readOnlyHint=true", name)
			}
			if a.IdempotentHint == nil || !*a.IdempotentHint {
				t.Errorf("tool %q should have idempotentHint=true", name)
			}
		})
	}
}

// TestAnnotateTools_DestructiveTools verifies delete_* and cancel_* tools have destructiveHint.
func TestAnnotateTools_DestructiveTools(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		name := tool.Name
		if !strings.HasPrefix(name, "delete_") && name != "cancel_scheduled_message" {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			a := tool.Annotations
			if a == nil {
				t.Fatalf("tool %q has nil annotations", name)
			}
			if a.DestructiveHint == nil || !*a.DestructiveHint {
				t.Errorf("tool %q should have destructiveHint=true", name)
			}
		})
	}
}

// TestAnnotateTools_LocalUtilityTools verifies utility tools have no openWorldHint.
func TestAnnotateTools_LocalUtilityTools(t *testing.T) {
	t.Parallel()

	localTools := []string{"current_time", "epoch_to_datetime", "datetime_to_epoch"}
	tools := registeredTools()

	toolMap := make(map[string]MCPTool)
	for _, tool := range tools {
		toolMap[tool.Name] = tool
	}

	for _, name := range localTools {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tool, ok := toolMap[name]
			if !ok {
				t.Fatalf("tool %q not found", name)
			}
			a := tool.Annotations
			if a == nil {
				t.Fatalf("tool %q has nil annotations", name)
			}
			if a.OpenWorldHint != nil {
				t.Errorf("tool %q should not have openWorldHint (local tool)", name)
			}
			if a.ReadOnlyHint == nil || !*a.ReadOnlyHint {
				t.Errorf("tool %q should have readOnlyHint=true", name)
			}
		})
	}
}

// TestAnnotateTools_APIToolsHaveOpenWorld verifies all API-calling tools have openWorldHint.
func TestAnnotateTools_APIToolsHaveOpenWorld(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		name := tool.Name
		if localUtilityTools[name] {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			a := tool.Annotations
			if a == nil {
				t.Fatalf("tool %q has nil annotations", name)
			}
			if a.OpenWorldHint == nil || !*a.OpenWorldHint {
				t.Errorf("tool %q should have openWorldHint=true (API tool)", name)
			}
		})
	}
}

// TestAnnotateTools_JSONSerialization verifies annotations and title appear in JSON output.
func TestAnnotateTools_JSONSerialization(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	// Find list_messages as a representative tool.
	var listMessages MCPTool
	for _, tool := range tools {
		if tool.Name == "list_messages" {
			listMessages = tool
			break
		}
	}
	if listMessages.Name == "" {
		t.Fatal("list_messages not found")
	}

	data, err := json.Marshal(listMessages)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if _, ok := raw["title"]; !ok {
		t.Error("JSON output missing 'title' field")
	}
	if _, ok := raw["annotations"]; !ok {
		t.Error("JSON output missing 'annotations' field")
	}

	annotations, ok := raw["annotations"].(map[string]any)
	if !ok {
		t.Fatal("annotations is not a map")
	}
	if _, ok := annotations["readOnlyHint"]; !ok {
		t.Error("annotations missing 'readOnlyHint'")
	}
}

// TestAnnotateTools_MutatingToolsNotDestructive verifies send/create/update/smart_compose
// tools explicitly set destructiveHint=false.
func TestAnnotateTools_MutatingToolsNotDestructive(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		name := tool.Name
		isMutating := strings.HasPrefix(name, "send_") ||
			strings.HasPrefix(name, "create_") ||
			strings.HasPrefix(name, "update_") ||
			strings.HasPrefix(name, "smart_compose")
		if !isMutating {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			a := tool.Annotations
			if a == nil {
				t.Fatalf("tool %q has nil annotations", name)
			}
			if a.DestructiveHint == nil {
				t.Errorf("tool %q should explicitly set destructiveHint (spec default is true)", name)
			} else if *a.DestructiveHint {
				t.Errorf("tool %q should have destructiveHint=false", name)
			}
		})
	}
}

// TestAnnotateTools_UpdateToolsIdempotent verifies update_* tools have idempotentHint=true.
func TestAnnotateTools_UpdateToolsIdempotent(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		name := tool.Name
		if !strings.HasPrefix(name, "update_") {
			continue
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			a := tool.Annotations
			if a == nil {
				t.Fatalf("tool %q has nil annotations", name)
			}
			if a.IdempotentHint == nil || !*a.IdempotentHint {
				t.Errorf("tool %q should have idempotentHint=true", name)
			}
		})
	}
}

// TestToolTitles_CoverAllTools verifies the toolTitles map covers all registered tools.
func TestToolTitles_CoverAllTools(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	for _, tool := range tools {
		if _, ok := toolTitles[tool.Name]; !ok {
			t.Errorf("tool %q has no entry in toolTitles map", tool.Name)
		}
	}
}
