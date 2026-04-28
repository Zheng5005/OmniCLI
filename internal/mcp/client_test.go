package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// mockTransport implements Transport for testing.
type mockTransport struct {
	responses map[string]JSONRPCResponse
	err       error
	closed    bool
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		responses: make(map[string]JSONRPCResponse),
	}
}

func (m *mockTransport) addResponse(method string, resp JSONRPCResponse) {
	m.responses[method] = resp
}

func (m *mockTransport) Send(ctx context.Context, req JSONRPCRequest) (JSONRPCResponse, error) {
	if m.err != nil {
		return JSONRPCResponse{}, m.err
	}
	resp, ok := m.responses[req.Method]
	if !ok {
		return JSONRPCResponse{}, fmt.Errorf("unexpected method: %s", req.Method)
	}
	resp.ID = req.ID
	if resp.Error != nil {
		return resp, resp.Error
	}
	return resp, nil
}

func (m *mockTransport) Notify(ctx context.Context, method string, params json.RawMessage) error {
	return nil
}

func (m *mockTransport) Close() error {
	m.closed = true
	return nil
}

func TestClientInitialize(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("initialize", JSONRPCResponse{
		Result: mustMarshal(t, InitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo:      ServerInfo{Name: "test", Version: "1.0"},
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if !client.Initialized() {
		t.Error("expected Initialized() = true")
	}
}

func TestClientInitializeError(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("initialize", JSONRPCResponse{
		Error: &JSONRPCError{Code: -32600, Message: "Invalid Request"},
	})

	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err == nil {
		t.Fatal("expected error")
	}
}

func TestClientListTools(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("initialize", JSONRPCResponse{
		Result: mustMarshal(t, InitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo:      ServerInfo{Name: "test", Version: "1.0"},
		}),
	})
	mt.addResponse("tools/list", JSONRPCResponse{
		Result: mustMarshal(t, struct {
			Tools []McpToolDef `json:"tools"`
		}{
			Tools: []McpToolDef{
				{Name: "echo", Description: "Echo input"},
			},
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}
	if tools[0].Name != "echo" {
		t.Errorf("name = %q, want echo", tools[0].Name)
	}

	cached := client.Tools()
	if len(cached) != 1 {
		t.Fatalf("cached len = %d, want 1", len(cached))
	}
}

func TestClientCallTool(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("tools/call", JSONRPCResponse{
		Result: mustMarshal(t, CallToolResult{
			Content: []ToolContent{{Type: "text", Text: "result"}},
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.CallTool(ctx, "echo", json.RawMessage(`{"msg":"hello"}`))
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result != "result" {
		t.Errorf("result = %q, want result", result)
	}
}

func TestClientCallToolError(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("tools/call", JSONRPCResponse{
		Result: mustMarshal(t, CallToolResult{
			Content: []ToolContent{{Type: "text", Text: "something went wrong"}},
			IsError: true,
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.CallTool(ctx, "echo", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientListResources(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("resources/list", JSONRPCResponse{
		Result: mustMarshal(t, struct {
			Resources []McpResource `json:"resources"`
		}{
			Resources: []McpResource{
				{URI: "file:///test.txt", Name: "test"},
			},
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resources, err := client.ListResources(ctx)
	if err != nil {
		t.Fatalf("list resources: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("len = %d, want 1", len(resources))
	}
	if resources[0].URI != "file:///test.txt" {
		t.Errorf("uri = %q", resources[0].URI)
	}
}

func TestClientReadResource(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("resources/read", JSONRPCResponse{
		Result: mustMarshal(t, ReadResourceResult{
			Contents: []ResourceContent{
				{URI: "file:///test.txt", Text: "hello world"},
			},
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	content, err := client.ReadResource(ctx, "file:///test.txt")
	if err != nil {
		t.Fatalf("read resource: %v", err)
	}
	if content != "hello world" {
		t.Errorf("content = %q, want hello world", content)
	}
}

func TestClientClose(t *testing.T) {
	mt := newMockTransport()
	client := NewClient("test", ServerConfig{}, mt)

	if err := client.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if !mt.closed {
		t.Error("expected transport to be closed")
	}
}

func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

func TestClientInitializeInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("initialize", JSONRPCResponse{Result: json.RawMessage(`{bad`)})
	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Initialize(ctx); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestClientListToolsInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("tools/list", JSONRPCResponse{Result: json.RawMessage(`{bad`)})
	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.ListTools(ctx)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestClientCallToolEmptyArgs(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("tools/call", JSONRPCResponse{
		Result: mustMarshal(t, CallToolResult{
			Content: []ToolContent{{Type: "text", Text: "done"}},
		}),
	})
	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := client.CallTool(ctx, "echo", nil)
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result != "done" {
		t.Errorf("result = %q, want done", result)
	}
}

func TestClientCallToolTransportError(t *testing.T) {
	mt := newMockTransport()
	mt.err = fmt.Errorf("network down")
	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.CallTool(ctx, "echo", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientReadResourceInvalidJSON(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("resources/read", JSONRPCResponse{Result: json.RawMessage(`{bad`)})
	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.ReadResource(ctx, "uri")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestClientListResourcesTransportError(t *testing.T) {
	mt := newMockTransport()
	mt.err = fmt.Errorf("network down")
	client := NewClient("test", ServerConfig{}, mt)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := client.ListResources(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientCloseIdempotent(t *testing.T) {
	mt := newMockTransport()
	client := NewClient("test", ServerConfig{}, mt)
	if err := client.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestClientToolsEmptyBeforeList(t *testing.T) {
	client := NewClient("test", ServerConfig{}, nil)
	tools := client.Tools()
	if len(tools) != 0 {
		t.Errorf("len = %d, want 0", len(tools))
	}
}

func TestClientInitializedBeforeInitialize(t *testing.T) {
	client := NewClient("test", ServerConfig{}, nil)
	if client.Initialized() {
		t.Error("expected false before Initialize")
	}
}
