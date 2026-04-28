package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// failingTransport always returns an error after a configurable number of successes.
type failingTransport struct {
	mu        sync.Mutex
	successes int
	calls     int
	closed    bool
}

func (f *failingTransport) Send(ctx context.Context, req JSONRPCRequest) (JSONRPCResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.calls <= f.successes {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  json.RawMessage(`{"tools":[]}`),
		}, nil
	}
	return JSONRPCResponse{}, fmt.Errorf("transport failed")
}

func (f *failingTransport) Notify(ctx context.Context, method string, params json.RawMessage) error {
	return nil
}

func (f *failingTransport) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}

func TestManagerStartAll(t *testing.T) {
	mt := newMockTransport()
	mt.addResponse("initialize", JSONRPCResponse{
		Result: mustMarshal(t, InitializeResult{
			ProtocolVersion: "2024-11-05",
			ServerInfo:      ServerInfo{Name: "test", Version: "1.0"},
		}),
	})
	mt.addResponse("tools/list", JSONRPCResponse{
		Result: json.RawMessage(`{"tools":[]}`),
	})

	// Override transport factory for testing.
	// We test the Manager using a mock by directly injecting clients after StartAll.
	// Since StartAll uses NewStdioTransport/NewSSETransport, we'll test with
	// real transports via the echo helper and also test the Manager's methods
	// by creating a Manager and manually adding clients.

	mgr := NewManager(map[string]ServerConfig{
		"test": {Type: "stdio", Command: echoPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("start all: %v", err)
	}
	defer mgr.StopAll()

	status := mgr.Status()
	if len(status) != 1 {
		t.Fatalf("status len = %d, want 1", len(status))
	}
	if status["test"].State != "ready" {
		t.Errorf("state = %q, want ready", status["test"].State)
	}

	client := mgr.Client("test")
	if client == nil {
		t.Fatal("client is nil")
	}
}

func TestManagerStatus(t *testing.T) {
	mgr := NewManager(nil)

	status := mgr.Status()
	if len(status) != 0 {
		t.Errorf("len = %d, want 0", len(status))
	}
}

func TestManagerStopAll(t *testing.T) {
	mgr := NewManager(map[string]ServerConfig{
		"test": {Type: "stdio", Command: echoPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("start all: %v", err)
	}

	if err := mgr.StopAll(); err != nil {
		t.Fatalf("stop all: %v", err)
	}

	status := mgr.Status()
	if status["test"].State != "disconnected" {
		t.Errorf("state = %q, want disconnected", status["test"].State)
	}
}

func TestManagerReconnectServer(t *testing.T) {
	mgr := NewManager(map[string]ServerConfig{
		"test": {Type: "stdio", Command: echoPath},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("start all: %v", err)
	}
	defer mgr.StopAll()

	if err := mgr.ReconnectServer(ctx, "test"); err != nil {
		t.Fatalf("reconnect: %v", err)
	}

	status := mgr.Status()
	if status["test"].State != "ready" {
		t.Errorf("state = %q, want ready", status["test"].State)
	}
}

func TestManagerReconnectUnknownServer(t *testing.T) {
	mgr := NewManager(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mgr.ReconnectServer(ctx, "unknown"); err == nil {
		t.Fatal("expected error for unknown server")
	}
}

func TestManagerStartAllUnknownType(t *testing.T) {
	mgr := NewManager(map[string]ServerConfig{
		"bad": {Type: "unknown"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("start all: %v", err)
	}

	status := mgr.Status()
	if status["bad"].State != "error" {
		t.Errorf("state = %q, want error", status["bad"].State)
	}
}

func TestManagerHealthMonitorRecovers(t *testing.T) {
	oldInterval := healthCheckInterval
	healthCheckInterval = 100 * time.Millisecond
	defer func() { healthCheckInterval = oldInterval }()

	mgr := NewManager(map[string]ServerConfig{
		"bad": {Type: "stdio", Command: "/nonexistent/command"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("start all: %v", err)
	}

	// Wait for health monitor to run a few cycles.
	time.Sleep(800 * time.Millisecond)

	status := mgr.Status()
	if status["bad"].State != "failed" {
		t.Logf("status: %+v", status["bad"])
		// The server might still be in error state depending on timing.
		// Either error or failed is acceptable.
		if status["bad"].State != "error" {
			t.Errorf("state = %q, want error or failed", status["bad"].State)
		}
	}

	_ = mgr.StopAll()
}

func TestManagerStartAllEmpty(t *testing.T) {
	mgr := NewManager(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("start all: %v", err)
	}
}

func TestManagerClient(t *testing.T) {
	mgr := NewManager(nil)
	if mgr.Client("anything") != nil {
		t.Error("expected nil client for empty manager")
	}
}

func TestManagerHealthMonitorMockFailure(t *testing.T) {
	oldInterval := healthCheckInterval
	healthCheckInterval = 100 * time.Millisecond
	defer func() { healthCheckInterval = oldInterval }()

	ft := &failingTransport{successes: 0} // always fails
	client := NewClient("mock", ServerConfig{Type: "stdio", Command: "echo"}, ft)
	mgr := NewManager(nil)
	mgr.clients["mock"] = client
	mgr.status["mock"] = ServerStatus{State: "ready", Healthy: true}
	mgr.wg.Add(1)
	go mgr.healthMonitor("mock", client)

	time.Sleep(350 * time.Millisecond)

	mgr.mu.RLock()
	st := mgr.status["mock"]
	mgr.mu.RUnlock()

	if st.State != "failed" && st.State != "error" {
		t.Errorf("state = %q, want failed or error", st.State)
	}

	close(mgr.done)
	mgr.wg.Wait()
}

func TestManagerStatusSnapshot(t *testing.T) {
	mgr := NewManager(nil)
	mgr.status["s"] = ServerStatus{State: "ready"}
	snapshot := mgr.Status()
	snapshot["s"] = ServerStatus{State: "hacked"}
	if mgr.status["s"].State != "ready" {
		t.Error("modifying snapshot affected internal state")
	}
}

func TestManagerStopAllTwice(t *testing.T) {
	mgr := NewManager(map[string]ServerConfig{
		"test": {Type: "stdio", Command: echoPath},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mgr.StartAll(ctx); err != nil {
		t.Fatalf("start all: %v", err)
	}
	if err := mgr.StopAll(); err != nil {
		t.Fatalf("first stop: %v", err)
	}
	if err := mgr.StopAll(); err != nil {
		t.Fatalf("second stop: %v", err)
	}
}
