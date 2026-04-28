// Package mcp implements the Model Context Protocol (MCP) for OmniCLI.
//
// It provides JSON-RPC 2.0 communication over stdio and SSE transports,
// client lifecycle management, and server health monitoring.
package mcp

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request or notification.
// Notifications omit the ID field.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      int64         `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *JSONRPCError) Error() string {
	return fmt.Sprintf("JSON-RPC error %d: %s", e.Code, e.Message)
}

// ClientInfo describes the MCP client identity.
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ServerInfo describes the MCP server identity.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeParams holds parameters for the initialize method.
type InitializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ClientInfo      ClientInfo             `json:"clientInfo"`
}

// InitializeResult holds the result of the initialize method.
type InitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	ServerInfo      ServerInfo             `json:"serverInfo"`
}

// McpToolDef represents a tool definition discovered from an MCP server.
type McpToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// CallToolParams holds parameters for the tools/call method.
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// CallToolResult holds the result of a tool call.
type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent represents a single content item in a tool result.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// McpResource represents a resource exposed by an MCP server.
type McpResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// ReadResourceParams holds parameters for the resources/read method.
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ReadResourceResult holds the result of reading a resource.
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ResourceContent represents a single content item in a resource read result.
type ResourceContent struct {
	URI      string `json:"uri"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

// Notification represents a JSON-RPC notification (no id field).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// ServerConfig describes how to connect to an MCP server.
type ServerConfig struct {
	Type    string            `json:"type"`    // "stdio" or "sse"
	Command string            `json:"command"` // stdio: executable path
	Args    []string          `json:"args"`    // stdio: arguments
	URL     string            `json:"url"`     // sse: remote endpoint
	Env     map[string]string `json:"env"`     // environment variables
	Trusted bool              `json:"trusted"` // skip approval gate
	Timeout int               `json:"timeout"` // request timeout in seconds, default 30
}

// RequestTimeout returns the configured timeout or a default of 30s.
func (c ServerConfig) RequestTimeout() int {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return 30
}

// PinnedResource represents an MCP resource attached to a conversation.
type PinnedResource struct {
	ServerName  string
	URI         string
	Name        string
	Description string
	Content     string
}

// ServerStatus represents the current state of an MCP server.
type ServerStatus struct {
	State   string // "starting", "ready", "error", "failed", "disconnected"
	Tools   int
	Error   string
	Healthy bool
}

// idGenerator generates monotonically increasing request IDs.
type idGenerator struct {
	next atomic.Int64
}

// Next returns the next unique request ID.
func (g *idGenerator) Next() int64 {
	return g.next.Add(1)
}
