package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// mockServerBin is compiled once per package test run.
var mockServerBin string

func init() {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	mockServerBin = filepath.Join(dir, "..", "..", "test", "mcp", "mock_server")
	_ = exec.Command("go", "build", "-o", mockServerBin, filepath.Join(dir, "..", "..", "test", "mcp", "mock_server.go")).Run()
}

func skipIfNoMockServer(t *testing.T) {
	if _, err := os.Stat(mockServerBin); err != nil {
		t.Skip("mock server binary not available")
	}
}

// TestIntegrationToolCallRoundTrip performs a full initialize → list → call
// cycle against the mock stdio server.
func TestIntegrationToolCallRoundTrip(t *testing.T) {
	skipIfNoMockServer(t)

	transport, err := NewStdioTransport(mockServerBin, nil, nil)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	defer transport.Close()

	client := NewClient("mock", ServerConfig{Type: "stdio", Command: mockServerBin}, transport)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("tools len = %d, want 2", len(tools))
	}

	// Call echo
	result, err := client.CallTool(ctx, "echo", json.RawMessage(`{"msg":"hello"}`))
	if err != nil {
		t.Fatalf("call echo: %v", err)
	}
	if result != "hello" {
		t.Errorf("echo result = %q, want hello", result)
	}

	// Call add
	result, err = client.CallTool(ctx, "add", json.RawMessage(`{"a":1,"b":2}`))
	if err != nil {
		t.Fatalf("call add: %v", err)
	}
	if result != "3" {
		t.Errorf("add result = %q, want 3", result)
	}
}

// TestIntegrationMCPToolExecute verifies that MCPTool.Execute delegates to
// the mock server correctly, exercising the same code path the agent loop uses.
func TestIntegrationMCPToolExecute(t *testing.T) {
	skipIfNoMockServer(t)

	transport, err := NewStdioTransport(mockServerBin, nil, nil)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	defer transport.Close()

	client := NewClient("mock", ServerConfig{Type: "stdio", Command: mockServerBin, Trusted: true}, transport)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}

	tool := NewMCPTool("mock", tools[0], client, true, nil)
	result, err := tool.Execute(ctx, json.RawMessage(`{"msg":"world"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result != "world" {
		t.Errorf("result = %q, want world", result)
	}
}

// TestIntegrationCrashRecovery kills the mock server process and verifies
// that the manager's health monitor detects the failure and restarts it.
func TestIntegrationCrashRecovery(t *testing.T) {
	skipIfNoMockServer(t)

	oldInterval := healthCheckInterval
	healthCheckInterval = 200 * time.Millisecond
	defer func() { healthCheckInterval = oldInterval }()

	mgr := NewManager(map[string]ServerConfig{
		"mock": {Type: "stdio", Command: mockServerBin},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("start all: %v", err)
	}
	defer mgr.StopAll()

	status := mgr.Status()
	if status["mock"].State != "ready" {
		t.Fatalf("expected ready, got %q", status["mock"].State)
	}

	// Obtain the underlying process and kill it to simulate a crash.
	oldClient := mgr.Client("mock")
	if oldClient == nil {
		t.Fatal("client is nil")
	}
	st, ok := oldClient.transport.(*StdioTransport)
	if !ok {
		t.Fatal("expected StdioTransport")
	}
	if st.cmd == nil || st.cmd.Process == nil {
		t.Fatal("process not available")
	}
	if err := st.cmd.Process.Kill(); err != nil {
		t.Fatalf("kill process: %v", err)
	}

	// Wait for the manager to remove the old client (detect the crash)
	// and then replace it with a new one.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if mgr.Client("mock") != oldClient {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Poll for the new server to be ready.
	recovered := false
	for time.Now().Before(deadline) {
		status := mgr.Status()["mock"]
		if status.State == "ready" && mgr.Client("mock") != nil && mgr.Client("mock") != oldClient {
			recovered = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !recovered {
		t.Fatalf("manager did not recover after crash; status: %+v", mgr.Status()["mock"])
	}

	// Verify the new process is functional.
	client := mgr.Client("mock")
	if client == nil {
		t.Fatal("client nil after recovery")
	}
	tools, err := client.ListTools(ctx)
	if err != nil {
		t.Fatalf("list tools after recovery: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("tools len = %d, want 2", len(tools))
	}
}
