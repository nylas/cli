package domain

import (
	"encoding/json"
	"testing"
)

// The Nylas v3 API and the workspace service (uas) use the JSON field name
// "rule_ids" for workspace rule attachments. These tests pin the wire format so
// the CLI's rule-attach PATCH is actually read by the server — a mismatch
// silently no-ops the attach (the PATCH succeeds but the rule is never linked).

func TestUpdateWorkspaceRequest_RuleIDsWireName(t *testing.T) {
	ids := []string{"rule-1", "rule-2"}
	req := UpdateWorkspaceRequest{RulesIDs: &ids}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["rule_ids"]; !ok {
		t.Fatalf("UpdateWorkspaceRequest must serialize rule attachments as \"rule_ids\"; got %s", data)
	}
	if _, ok := raw["rules_ids"]; ok {
		t.Fatalf("UpdateWorkspaceRequest must not use \"rules_ids\" (server reads \"rule_ids\"); got %s", data)
	}
}

func TestWorkspace_DefaultWireField(t *testing.T) {
	// The server marks the connector's default workspace with "default": true.
	// Dropping the field hides which workspace new agent accounts attach to.
	const body = `{"workspace_id":"ws-1","auto_group":true,"default":true}`

	var ws Workspace
	if err := json.Unmarshal([]byte(body), &ws); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !ws.Default {
		t.Fatalf("Workspace must decode \"default\" into Default; got %#v", ws)
	}

	data, err := json.Marshal(ws)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal marshaled: %v", err)
	}
	if string(raw["default"]) != "true" {
		t.Fatalf("Workspace JSON output must include \"default\"; got %s", data)
	}
}

func TestWorkspace_RuleIDsWireName(t *testing.T) {
	// The server returns "rule_ids"; it must populate RulesIDs on read.
	const body = `{"workspace_id":"ws-1","rule_ids":["rule-a","rule-b"]}`

	var ws Workspace
	if err := json.Unmarshal([]byte(body), &ws); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(ws.RulesIDs) != 2 {
		t.Fatalf("Workspace must decode \"rule_ids\" into RulesIDs; got %#v", ws.RulesIDs)
	}
}
