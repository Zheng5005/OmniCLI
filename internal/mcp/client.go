package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Client handles MCP protocol communication with a single server.
type Client struct {
	name        string
	config      ServerConfig
	transport   Transport
	tools       []McpToolDef
	mu          sync.RWMutex
	initialized bool
}

// NewClient creates a new MCP client with the given configuration.
func NewClient(name string, config ServerConfig, transport Transport) *Client {
	return &Client{
		name:      name,
		config:    config,
		transport: transport,
	}
}

// Name returns the client's configured server name.
func (c *Client) Name() string {
	return c.name
}

// Config returns the client's server configuration.
func (c *Client) Config() ServerConfig {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

// Initialize performs the MCP handshake (initialize + initialized notification).
func (c *Client) Initialize(ctx context.Context) error {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		Capabilities:    map[string]interface{}{},
		ClientInfo: ClientInfo{
			Name:    "omnicli",
			Version: "1.0.0",
		},
	}

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("marshal initialize params: %w", err)
	}

	req := JSONRPCRequest{
		Method: "initialize",
		Params: paramsBytes,
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return fmt.Errorf("initialize request: %w", err)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parse initialize result: %w", err)
	}

	// Send initialized notification (best effort).
	ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = c.transport.Notify(ctx2, "notifications/initialized", nil)

	c.mu.Lock()
	c.initialized = true
	c.mu.Unlock()

	return nil
}

// ListTools calls tools/list on the server and returns discovered tools.
func (c *Client) ListTools(ctx context.Context) ([]McpToolDef, error) {
	req := JSONRPCRequest{
		Method: "tools/list",
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list tools: %w", err)
	}

	var result struct {
		Tools []McpToolDef `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse tools list: %w", err)
	}

	c.mu.Lock()
	c.tools = result.Tools
	c.mu.Unlock()

	return result.Tools, nil
}

// Tools returns the cached list of tools from the last ListTools call.
func (c *Client) Tools() []McpToolDef {
	c.mu.RLock()
	defer c.mu.RUnlock()
	tools := make([]McpToolDef, len(c.tools))
	copy(tools, c.tools)
	return tools
}

// CallTool invokes a tool on the server and returns its text result.
func (c *Client) CallTool(ctx context.Context, toolName string, args json.RawMessage) (string, error) {
	params := CallToolParams{
		Name:      toolName,
		Arguments: args,
	}

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("marshal call params: %w", err)
	}

	req := JSONRPCRequest{
		Method: "tools/call",
		Params: paramsBytes,
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return "", fmt.Errorf("call tool: %w", err)
	}

	var result CallToolResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("parse tool result: %w", err)
	}

	if result.IsError {
		var msgs []string
		for _, content := range result.Content {
			msgs = append(msgs, content.Text)
		}
		return "", fmt.Errorf("tool error: %s", strings.Join(msgs, "; "))
	}

	var out strings.Builder
	for _, content := range result.Content {
		out.WriteString(content.Text)
	}
	return out.String(), nil
}

// ListResources calls resources/list on the server.
func (c *Client) ListResources(ctx context.Context) ([]McpResource, error) {
	req := JSONRPCRequest{
		Method: "resources/list",
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}

	var result struct {
		Resources []McpResource `json:"resources"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parse resources list: %w", err)
	}

	return result.Resources, nil
}

// ReadResource calls resources/read for the given URI.
func (c *Client) ReadResource(ctx context.Context, uri string) (string, error) {
	params := ReadResourceParams{URI: uri}
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return "", fmt.Errorf("marshal read params: %w", err)
	}

	req := JSONRPCRequest{
		Method: "resources/read",
		Params: paramsBytes,
	}

	resp, err := c.transport.Send(ctx, req)
	if err != nil {
		return "", fmt.Errorf("read resource: %w", err)
	}

	var result ReadResourceResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("parse resource result: %w", err)
	}

	var out strings.Builder
	for _, content := range result.Contents {
		out.WriteString(content.Text)
	}
	return out.String(), nil
}

// Initialized reports whether the client has completed the handshake.
func (c *Client) Initialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// Close shuts down the underlying transport.
func (c *Client) Close() error {
	return c.transport.Close()
}
