package mcp

import (
	"encoding/json"
	"net/http"
	"regexp"
)

// Streamable HTTP request headers the hosted Nylas MCP server reads
// (api-v3 src/mcp/server.go). Mcp-Method and Mcp-Name let it route and
// meter a request without parsing the body, and it refuses a request whose
// Mcp-Name disagrees with the body. There is deliberately no Mcp-Session-Id:
// the server runs stateless — every request stands alone — so the proxy
// neither stores nor sends one.
const (
	headerMCPMethod          = "Mcp-Method"
	headerMCPName            = "Mcp-Name"
	headerMCPProtocolVersion = "Mcp-Protocol-Version"
)

// headerSafeName allow-lists what may be copied from a JSON-RPC body into a
// header: method and tool names are short identifiers. Anything else is left
// to the body, which the server still parses for clients that send no
// headers at all.
var headerSafeName = regexp.MustCompile(`^[A-Za-z0-9_./\-]{1,128}$`)

// protocolVersionPattern is the MCP date-stamped version shape.
var protocolVersionPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// setProtocolHeaders adds the MCP routing headers for a parsed request.
// Mcp-Name is sent for the methods whose params carry a name; if that name
// cannot go in a header, neither header is sent, because a method header
// without its name reads as a mismatch rather than as a legacy request.
func setProtocolHeaders(h http.Header, parsed *rpcRequest, protocolVersion string) {
	if protocolVersion != "" {
		h.Set(headerMCPProtocolVersion, protocolVersion)
	}
	if parsed == nil || !headerSafeName.MatchString(parsed.Method) {
		return
	}
	switch parsed.Method {
	case "tools/call", "prompts/get":
		if !headerSafeName.MatchString(parsed.Params.Name) {
			return
		}
		h.Set(headerMCPName, parsed.Params.Name)
	}
	h.Set(headerMCPMethod, parsed.Method)
}

// rememberProtocolVersion records the version the server negotiated in its
// initialize answer. Later requests carry it as Mcp-Protocol-Version, as the
// streamable HTTP transport requires once a version has been agreed.
func (p *Proxy) rememberProtocolVersion(initializeResponse []byte) {
	var resp struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
	}
	if err := json.Unmarshal(initializeResponse, &resp); err != nil {
		return
	}
	if !protocolVersionPattern.MatchString(resp.Result.ProtocolVersion) {
		return
	}
	p.mu.Lock()
	p.protocolVersion = resp.Result.ProtocolVersion
	p.mu.Unlock()
}
