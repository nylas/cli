package mcp

// MCPTool represents an MCP tool definition.
type MCPTool struct {
	Name        string           `json:"name"`
	Title       string           `json:"title,omitempty"`
	Description string           `json:"description"`
	InputSchema JSONSchema       `json:"inputSchema"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
}

// ToolAnnotations describes the behavior hints for an MCP tool (MCP spec 2025-06-18).
type ToolAnnotations struct {
	ReadOnlyHint    *bool `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

// JSONSchema represents a JSON Schema object.
type JSONSchema struct {
	Type                 string                `json:"type"`
	Properties           map[string]JSONSchema `json:"properties,omitempty"`
	Required             []string              `json:"required,omitempty"`
	Items                *JSONSchema           `json:"items,omitempty"`
	Enum                 []string              `json:"enum,omitempty"`
	Desc                 string                `json:"description,omitempty"`
	AdditionalProperties *bool                 `json:"additionalProperties,omitempty"`
}

// prop creates a simple typed property with a description.
func prop(typ, desc string) JSONSchema {
	return JSONSchema{Type: typ, Desc: desc}
}

// grantProp is the standard grant_id property description.
const grantDesc = "Nylas grant ID (omit for default)"

// epochDesc is a suffix for Unix epoch timestamp fields.
const epochDesc = " (unix epoch)"

// participantItems is the shared schema for {name, email} objects, allocated once.
var participantItems = &JSONSchema{
	Type: "object",
	Properties: map[string]JSONSchema{
		"name":  prop("string", "Name"),
		"email": prop("string", "Email"),
	},
}

// participantArraySchema returns a JSON Schema for an array of {name, email} objects.
func participantArraySchema(desc string) JSONSchema {
	return JSONSchema{
		Type:  "array",
		Desc:  desc,
		Items: participantItems,
	}
}

// objectSchema returns a schema with properties, required fields, and additionalProperties: false.
func objectSchema(props map[string]JSONSchema, required []string) JSONSchema {
	noExtra := false
	return JSONSchema{Type: "object", Properties: props, Required: required, AdditionalProperties: &noExtra}
}

// boolPtr returns a pointer to a bool value.
func boolPtr(v bool) *bool {
	return &v
}
