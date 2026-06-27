// Package mcp implements an MCP (Model Context Protocol) security proxy.
// Provides real-time traffic auditing, token burn-rate tracking, prompt injection
// detection, context window optimization, PII redaction, rate limiting,
// tool-use approval, and agent-scoped token budgets.
package mcp

import "encoding/json"

// JSONRPC is a generic JSON-RPC 2.0 message.
type JSONRPC struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

// JSONRPCError is a JSON-RPC error object.
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// MCPPolicyFunc checks whether a client request for the given method is allowed.
// Return (false, reason) to deny.
type MCPPolicyFunc func(clientAddr, method string) (allowed bool, reason string)

// MCPToolApprovalFunc checks whether a tools/call request is approved.
type MCPToolApprovalFunc func(clientAddr, toolName string, params json.RawMessage) (approved bool, reason string)

// MCP method constants.
const (
	MethodInitialize       = "initialize"
	MethodResourcesList    = "resources/list"
	MethodResourcesRead    = "resources/read"
	MethodPromptsGet       = "prompts/get"
	MethodToolsCall        = "tools/call"
	MethodToolsList        = "tools/list"
	MethodLoggingMessage   = "notifications/message"
	MethodInitialized      = "notifications/initialized"
	MethodResourceUpdated  = "notifications/resources/updated"
)

// Direction of a proxied message.
type Direction string

const (
	DirClientToServer Direction = "c2s"
	DirServerToClient Direction = "s2c"
)

// LogEntry is a single audited MCP message.
type LogEntry struct {
	ID          int64     `json:"id"`
	Direction   Direction `json:"direction"`
	Method      string    `json:"method"`
	RequestID   string    `json:"request_id,omitempty"`
	Content     string    `json:"content,omitempty"`
	ToksEst     int       `json:"toks_est"`
	Injection   bool      `json:"injection"`
	InjectionTy string    `json:"injection_type,omitempty"`
	Pruned      bool      `json:"pruned"`
	PrunedChars int       `json:"pruned_chars"`
	TimestampNS int64     `json:"timestamp_ns"`
}

// Stats holds proxy traffic statistics.
type Stats struct {
	TotalMessages    int64   `json:"total_messages"`
	TotalToks        int64   `json:"total_toks"`
	TokenBurnRate    float64 `json:"token_burn_rate"`    // tokens/sec over last window
	ContextDepthToks int     `json:"context_depth_toks"` // estimated current context size
	InjectionCount   int64   `json:"injection_count"`
	PrunedCount      int64   `json:"pruned_count"`
	PrunedBytes      int64   `json:"pruned_bytes"`
	UptimeSeconds    int64   `json:"uptime_seconds"`
}

// IsNotification returns true if this is a notification (no ID).
func (m *JSONRPC) IsNotification() bool {
	return len(m.ID) == 0 || string(m.ID) == "null"
}

// IsResponse returns true if this is a response (has ID but no method).
func (m *JSONRPC) IsResponse() bool {
	return len(m.ID) > 0 && string(m.ID) != "null" && m.Method == ""
}

// IsRequest returns true if this is a request (has method).
func (m *JSONRPC) IsRequest() bool {
	return m.Method != ""
}

// EstimateTokens estimates the number of tokens in a string (~4 chars per token).
func EstimateTokens(s string) int {
	return len(s) / 4
}

// TruncateString truncates a string to maxLen chars, adding "...".
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
