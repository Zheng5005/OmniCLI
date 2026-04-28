package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMCPTool_Name(t *testing.T) {
	tool := NewMCPTool("postgres", McpToolDef{
		Name:        "query",
		Description: "Run SQL",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, nil, true, nil)

	if got, want := tool.Name(), "postgres__query"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

func TestMCPTool_TrustedExecution(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("tools/call", JSONRPCResponse{
		Result: mustMarshal(t, CallToolResult{
			Content: []ToolContent{{Type: "text", Text: `{"result":"ok"}`}},
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	tool := NewMCPTool("postgres", McpToolDef{
		Name:        "query",
		Description: "Run SQL",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, client, true, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, json.RawMessage(`{"sql":"SELECT 1"}`))
	if err != nil {
		t.Fatalf("trusted execution error: %v", err)
	}
	if result != `{"result":"ok"}` {
		t.Errorf("result = %q, want %q", result, `{"result":"ok"}`)
	}
}

func TestMCPTool_UntrustedDenied(t *testing.T) {
	approvalCalled := false
	approvalFn := func(serverName, toolName, description, args string) (bool, error) {
		approvalCalled = true
		return false, nil
	}

	tool := NewMCPTool("hack", McpToolDef{
		Name:        "exec",
		Description: "Run code",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, nil, false, approvalFn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, json.RawMessage(`{"cmd":"rm -rf /"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approvalCalled {
		t.Error("expected approval callback to be called")
	}
	want := "MCP tool call denied by user: hack__exec"
	if result != want {
		t.Errorf("result = %q, want %q", result, want)
	}
}

func TestMCPTool_UntrustedApproved(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("tools/call", JSONRPCResponse{
		Result: mustMarshal(t, CallToolResult{
			Content: []ToolContent{{Type: "text", Text: "executed"}},
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	approvalFn := func(serverName, toolName, description, args string) (bool, error) {
		return true, nil
	}

	tool := NewMCPTool("safe", McpToolDef{
		Name:        "run",
		Description: "Run",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, client, false, approvalFn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "executed" {
		t.Errorf("result = %q, want %q", result, "executed")
	}
}

func TestMCPTool_NoApprovalHandler(t *testing.T) {
	tool := NewMCPTool("untrusted", McpToolDef{
		Name:        "danger",
		Description: "Dangerous",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, nil, false, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "MCP tool call denied (no approval handler): untrusted__danger"
	if result != want {
		t.Errorf("result = %q, want %q", result, want)
	}
}

func TestMCPTool_ExecuteError(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("tools/call", JSONRPCResponse{
		Result: mustMarshal(t, CallToolResult{
			Content: []ToolContent{{Type: "text", Text: "something went wrong"}},
			IsError: true,
		}),
	})

	client := NewClient("test", ServerConfig{}, mt)
	tool := NewMCPTool("bad", McpToolDef{
		Name:        "fail",
		Description: "Fail",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, client, true, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := tool.Execute(ctx, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	want := "MCP tool error: tool error: something went wrong"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestMCPTool_EmptyParameters(t *testing.T) {
	tool := NewMCPTool("test", McpToolDef{
		Name:        "noop",
		Description: "No params",
		InputSchema: nil,
	}, nil, true, nil)

	params := tool.Parameters()
	if string(params) != "{}" {
		t.Errorf("Parameters() = %q, want %q", string(params), "{}")
	}
}

func TestMCPTool_ApprovalError(t *testing.T) {
	approvalFn := func(serverName, toolName, description, args string) (bool, error) {
		return false, fmt.Errorf("channel closed")
	}
	tool := NewMCPTool("untrusted", McpToolDef{
		Name:        "danger",
		Description: "Dangerous",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, nil, false, approvalFn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := tool.Execute(ctx, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "approval error") {
		t.Errorf("error = %q, want approval error", err.Error())
	}
}

func TestValidateSchema(t *testing.T) {
	tests := []struct {
		name        string
		schema      map[string]any
		wantOK      bool
		wantFeature string
	}{
		{"empty", map[string]any{}, true, ""},
		{"simple object", map[string]any{"type": "object"}, true, ""},
		{"nested properties", map[string]any{"type": "object", "properties": map[string]any{"name": map[string]any{"type": "string"}}}, true, ""},
		{"ref at root", map[string]any{"$ref": "#/$defs/Foo"}, false, "$ref"},
		{"oneOf at root", map[string]any{"oneOf": []any{map[string]any{"type": "string"}}}, false, "oneOf"},
		{"anyOf nested", map[string]any{"type": "object", "properties": map[string]any{"val": map[string]any{"anyOf": []any{map[string]any{"type": "string"}}}}}, false, "anyOf"},
		{"allOf in array items", map[string]any{"type": "array", "items": map[string]any{"allOf": []any{map[string]any{"type": "number"}}}}, false, "allOf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, feature := ValidateSchema(tt.schema)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if feature != tt.wantFeature {
				t.Errorf("feature = %q, want %q", feature, tt.wantFeature)
			}
		})
	}
}

func TestMCPTool_CallToolTransportError(t *testing.T) {
	mt := newMockTransport()
	mt.err = fmt.Errorf("transport failed")
	client := NewClient("test", ServerConfig{}, mt)
	tool := NewMCPTool("bad", McpToolDef{
		Name:        "fail",
		Description: "Fail",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, client, true, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := tool.Execute(ctx, json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "MCP tool error") {
		t.Errorf("error = %q, want MCP tool error", err.Error())
	}
}
