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

func TestUpdateWorkspaceRequest_PolicyIDWireFormat(t *testing.T) {
	// The API accepts "policy_id" as a UUID or null, never "" — detaching a
	// policy requires an explicit JSON null. A pointer to the empty string is
	// the detach signal; nil still omits the field so rule-only updates don't
	// clobber the attached policy.
	policy := "policy-1"
	empty := ""

	tests := []struct {
		name     string
		policyID *string
		want     string // expected raw JSON for "policy_id"; "" means absent
	}{
		{name: "nil omits the field", policyID: nil, want: ""},
		{name: "empty string serializes as null (detach)", policyID: &empty, want: "null"},
		{name: "value serializes as string", policyID: &policy, want: `"policy-1"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(UpdateWorkspaceRequest{PolicyID: tt.policyID})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got, ok := raw["policy_id"]
			if tt.want == "" {
				if ok {
					t.Fatalf("policy_id must be omitted when PolicyID is nil; got %s", data)
				}
				return
			}
			if !ok || string(got) != tt.want {
				t.Fatalf("policy_id must serialize as %s; got %s", tt.want, data)
			}
		})
	}
}

func TestUpdateWorkspaceRequest_RulesAndPolicyTogether(t *testing.T) {
	// A pointer to an EMPTY rules slice must serialize as "rule_ids":[] — the
	// API replaces the whole list, so [] is how the last rule gets detached
	// (rule.go's detach flow depends on this). Only a nil pointer omits the
	// field. Empty-slice-to-null/omit would silently break rule detachment.
	policy := "policy-1"
	rules := []string{}

	data, err := json.Marshal(UpdateWorkspaceRequest{PolicyID: &policy, RulesIDs: &rules})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(raw["policy_id"]) != `"policy-1"` {
		t.Fatalf("policy_id must survive alongside rule_ids; got %s", data)
	}
	if string(raw["rule_ids"]) != "[]" {
		t.Fatalf("empty rules slice must serialize as \"rule_ids\":[] (clears the last rule); got %s", data)
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
