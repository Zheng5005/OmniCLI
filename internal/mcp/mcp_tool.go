package mcp

import (
	"context"
	"encoding/json"
	"fmt"
)

// MCPApprovalFunc is called for untrusted MCP servers to request user
// approval before executing a tool. It returns true if approved, false if
// denied, and any error encountered during approval.
type MCPApprovalFunc func(serverName, toolName, description, args string) (bool, error)

// MCPTool wraps an MCP server tool as a tools.Tool implementation.
// The tool name is namespace-prefixed with the server name to avoid
// collisions across servers (e.g., "postgres__query").
type MCPTool struct {
	qualifiedName string
	description   string
	parameters    json.RawMessage
	serverName    string
	toolName      string
	client        *Client
	trusted       bool
	approvalFn    MCPApprovalFunc
}

// NewMCPTool creates a new MCPTool wrapping the given MCP tool definition.
func NewMCPTool(serverName string, def McpToolDef, client *Client, trusted bool, approvalFn MCPApprovalFunc) *MCPTool {
	return &MCPTool{
		qualifiedName: fmt.Sprintf("%s__%s", serverName, def.Name),
		description:   def.Description,
		parameters:    def.InputSchema,
		serverName:    serverName,
		toolName:      def.Name,
		client:        client,
		trusted:       trusted,
		approvalFn:    approvalFn,
	}
}

// Name returns the namespace-prefixed tool name.
func (t *MCPTool) Name() string { return t.qualifiedName }

// Description returns the tool description from the MCP server.
func (t *MCPTool) Description() string { return t.description }

// Parameters returns the JSON Schema for the tool's input parameters.
// Returns an empty object schema if no parameters were provided by the server.
func (t *MCPTool) Parameters() json.RawMessage {
	if len(t.parameters) == 0 {
		return json.RawMessage(`{}`)
	}
	return t.parameters
}

// ValidateSchema checks whether a JSON Schema object contains unsupported
// features. It returns true if the schema is supported, and false with the
// name of the first unsupported feature found ($ref, oneOf, anyOf, allOf).
func ValidateSchema(schema map[string]any) (bool, string) {
	return validateSchemaRecursive(schema)
}

func validateSchemaRecursive(v interface{}) (bool, string) {
	switch m := v.(type) {
	case map[string]any:
		for key, val := range m {
			if key == "$ref" || key == "oneOf" || key == "anyOf" || key == "allOf" {
				return false, key
			}
			if ok, feature := validateSchemaRecursive(val); !ok {
				return false, feature
			}
		}
	case []any:
		for _, item := range m {
			if ok, feature := validateSchemaRecursive(item); !ok {
				return false, feature
			}
		}
	}
	return true, ""
}

// Execute delegates the tool call to the MCP server via the client.
// For untrusted servers, it blocks on the approval callback before
// executing. MCP errors are returned as tool execution errors.
func (t *MCPTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if !t.trusted {
		if t.approvalFn == nil {
			return fmt.Sprintf("MCP tool call denied (no approval handler): %s", t.qualifiedName), nil
		}
		approved, err := t.approvalFn(t.serverName, t.toolName, t.description, string(args))
		if err != nil {
			return "", fmt.Errorf("approval error: %w", err)
		}
		if !approved {
			return fmt.Sprintf("MCP tool call denied by user: %s", t.qualifiedName), nil
		}
	}

	result, err := t.client.CallTool(ctx, t.toolName, args)
	if err != nil {
		return "", fmt.Errorf("MCP tool error: %w", err)
	}
	return result, nil
}
