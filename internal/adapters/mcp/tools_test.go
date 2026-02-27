package mcp

import "testing"

// TestBoolPtr verifies the boolPtr helper returns a pointer to the correct value.
func TestBoolPtr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		val  bool
	}{
		{name: "true", val: true},
		{name: "false", val: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := boolPtr(tt.val)
			if got == nil {
				t.Fatal("boolPtr() returned nil")
			}
			if *got != tt.val {
				t.Errorf("boolPtr(%v) = %v, want %v", tt.val, *got, tt.val)
			}
		})
	}
}

// TestBoolPtr_UniquePointers verifies each call returns a distinct pointer.
func TestBoolPtr_UniquePointers(t *testing.T) {
	t.Parallel()

	a := boolPtr(true)
	b := boolPtr(true)
	if a == b {
		t.Error("boolPtr should return distinct pointers for separate calls")
	}
}

// TestProp verifies the prop helper creates a JSONSchema with type and description.
func TestProp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typ      string
		desc     string
		wantType string
		wantDesc string
	}{
		{name: "string property", typ: "string", desc: "A name", wantType: "string", wantDesc: "A name"},
		{name: "integer property", typ: "integer", desc: "Count", wantType: "integer", wantDesc: "Count"},
		{name: "boolean property", typ: "boolean", desc: "Is active", wantType: "boolean", wantDesc: "Is active"},
		{name: "empty description", typ: "string", desc: "", wantType: "string", wantDesc: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schema := prop(tt.typ, tt.desc)
			if schema.Type != tt.wantType {
				t.Errorf("prop().Type = %q, want %q", schema.Type, tt.wantType)
			}
			if schema.Desc != tt.wantDesc {
				t.Errorf("prop().Desc = %q, want %q", schema.Desc, tt.wantDesc)
			}
			if schema.Properties != nil {
				t.Error("prop() should not set Properties")
			}
			if schema.Required != nil {
				t.Error("prop() should not set Required")
			}
		})
	}
}

// TestObjectSchema verifies objectSchema creates a schema with the expected fields.
func TestObjectSchema(t *testing.T) {
	t.Parallel()

	props := map[string]JSONSchema{
		"name":  prop("string", "User name"),
		"email": prop("string", "Email address"),
	}
	required := []string{"email"}

	schema := objectSchema(props, required)

	if schema.Type != "object" {
		t.Errorf("Type = %q, want %q", schema.Type, "object")
	}
	if len(schema.Properties) != 3 {
		t.Errorf("Properties count = %d, want 3", len(schema.Properties))
	}
	if _, ok := schema.Properties["name"]; !ok {
		t.Error("Properties missing 'name'")
	}
	if _, ok := schema.Properties["email"]; !ok {
		t.Error("Properties missing 'email'")
	}
	if _, ok := schema.Properties["grant_id"]; !ok {
		t.Error("Properties missing 'grant_id'")
	}
	if len(schema.Required) != 1 || schema.Required[0] != "email" {
		t.Errorf("Required = %v, want [email]", schema.Required)
	}
	if schema.AdditionalProperties == nil {
		t.Fatal("AdditionalProperties should be set")
	}
	if *schema.AdditionalProperties {
		t.Error("AdditionalProperties = true, want false")
	}
}

// TestObjectSchema_EmptyFields verifies objectSchema works with no properties or required fields.
func TestObjectSchema_EmptyFields(t *testing.T) {
	t.Parallel()

	schema := objectSchema(nil, nil)

	if schema.Type != "object" {
		t.Errorf("Type = %q, want %q", schema.Type, "object")
	}
	if len(schema.Properties) != 1 {
		t.Errorf("Properties count = %d, want 1", len(schema.Properties))
	}
	if _, ok := schema.Properties["grant_id"]; !ok {
		t.Errorf("Properties missing grant_id: %v", schema.Properties)
	}
	if schema.Required != nil {
		t.Errorf("Required = %v, want nil", schema.Required)
	}
}

// TestParticipantArraySchema verifies the participant array schema structure.
func TestParticipantArraySchema(t *testing.T) {
	t.Parallel()

	desc := "List of attendees"
	schema := participantArraySchema(desc)

	if schema.Type != "array" {
		t.Errorf("Type = %q, want %q", schema.Type, "array")
	}
	if schema.Desc != desc {
		t.Errorf("Desc = %q, want %q", schema.Desc, desc)
	}
	if schema.Items == nil {
		t.Fatal("Items should not be nil")
	}
	if schema.Items.Type != "object" {
		t.Errorf("Items.Type = %q, want %q", schema.Items.Type, "object")
	}

	itemProps := schema.Items.Properties
	if len(itemProps) != 2 {
		t.Fatalf("Items.Properties count = %d, want 2", len(itemProps))
	}
	if itemProps["name"].Type != "string" {
		t.Errorf("Items.Properties[name].Type = %q, want string", itemProps["name"].Type)
	}
	if itemProps["email"].Type != "string" {
		t.Errorf("Items.Properties[email].Type = %q, want string", itemProps["email"].Type)
	}
}

func TestRegisteredTools_OutputSchemaPresent(t *testing.T) {
	t.Parallel()

	tools := registeredTools()
	if len(tools) == 0 {
		t.Fatal("registeredTools returned no tools")
	}
	for _, tool := range tools {
		if tool.OutputSchema == nil {
			t.Fatalf("tool %q missing outputSchema", tool.Name)
		}
	}
}
