package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// echoPath is compiled once per package test run.
var echoPath string

func init() {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	echoPath = filepath.Join(dir, "testdata", "mcp_echo")
	_ = exec.Command("go", "build", "-o", echoPath, filepath.Join(dir, "testdata", "mcp_echo.go")).Run()
}

func skipIfNoEcho(t *testing.T) {
	if _, err := os.Stat(echoPath); err != nil {
		t.Skip("echo helper not compiled")
	}
}

func TestStdioTransportSend(t *testing.T) {
	skipIfNoEcho(t)

	transport, err := NewStdioTransport(echoPath, nil, nil)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	defer transport.Close()

	req := JSONRPCRequest{Method: "initialize"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := transport.Send(ctx, req)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	var result InitializeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if result.ServerInfo.Name != "test-server" {
		t.Errorf("server name = %q, want test-server", result.ServerInfo.Name)
	}
}

func TestStdioTransportNotification(t *testing.T) {
	skipIfNoEcho(t)

	transport, err := NewStdioTransport(echoPath, nil, nil)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Notify(ctx, "notifications/initialized", nil); err != nil {
		t.Fatalf("notify: %v", err)
	}
}

func TestStdioTransportClose(t *testing.T) {
	skipIfNoEcho(t)

	transport, err := NewStdioTransport(echoPath, nil, nil)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}

	if err := transport.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestStdioTransportRequestTimeout(t *testing.T) {
	transport, err := NewStdioTransport("sh", []string{"-c", "cat > /dev/null"}, nil)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	defer transport.Close()

	req := JSONRPCRequest{Method: "initialize"}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = transport.Send(ctx, req)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}

func TestStdioTransportErrorResponse(t *testing.T) {
	errScript := `package main
import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req map[string]interface{}
		json.Unmarshal(scanner.Bytes(), &req)
		id, _ := req["id"].(float64)
		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
			},
		}
		data, _ := json.Marshal(resp)
		fmt.Println(string(data))
	}
}
`
	tmpDir := t.TempDir()
	errSrc := filepath.Join(tmpDir, "error_server.go")
	if err := os.WriteFile(errSrc, []byte(errScript), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}

	errBin := filepath.Join(tmpDir, "error_server")
	if out, err := exec.Command("go", "build", "-o", errBin, errSrc).CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, out)
	}

	transport, err := NewStdioTransport(errBin, nil, nil)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	defer transport.Close()

	req := JSONRPCRequest{Method: "unknown"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := transport.Send(ctx, req)
	if err == nil {
		t.Fatal("expected error")
	}
	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("code = %d, want -32601", resp.Error.Code)
	}
}

func TestParseSSE(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantEv *sseEvent
		wantOk bool
	}{
		{
			name:   "simple event",
			input:  "event: message\ndata: hello\n\n",
			wantEv: &sseEvent{event: "message", data: "hello"},
			wantOk: true,
		},
		{
			name:   "multiline data",
			input:  "event: message\ndata: line1\ndata: line2\n\n",
			wantEv: &sseEvent{event: "message", data: "line1\nline2"},
			wantOk: true,
		},
		{
			name:   "endpoint event",
			input:  "event: endpoint\ndata: /message?session=abc\n\n",
			wantEv: &sseEvent{event: "endpoint", data: "/message?session=abc"},
			wantOk: true,
		},
		{
			name:   "empty input",
			input:  "",
			wantEv: nil,
			wantOk: false,
		},
		{
			name:   "no data",
			input:  "event: ping\n\n",
			wantEv: nil,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := bufio.NewScanner(strings.NewReader(tt.input))
			got, ok := parseSSE(scanner)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if !tt.wantOk {
				return
			}
			if got.event != tt.wantEv.event {
				t.Errorf("event = %q, want %q", got.event, tt.wantEv.event)
			}
			if got.data != tt.wantEv.data {
				t.Errorf("data = %q, want %q", got.data, tt.wantEv.data)
			}
		})
	}
}

func TestSSETransportEndpointDiscovery(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher := w.(http.Flusher)
			fmt.Fprintf(w, "event: endpoint\ndata: %s/message\n\n", srv.URL)
			flusher.Flush()
			// Keep connection open briefly then return.
			time.Sleep(100 * time.Millisecond)
			return
		}
		if r.Method == "POST" {
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer srv.Close()

	transport := NewSSETransport(srv.URL)
	defer transport.Close()

	time.Sleep(300 * time.Millisecond)

	if transport.getPostURL() != srv.URL+"/message" {
		t.Fatalf("postURL = %q, want %q", transport.getPostURL(), srv.URL+"/message")
	}
}

func TestSSETransportSend(t *testing.T) {
	var mu sync.Mutex
	pending := make(map[int64]JSONRPCRequest)
	var endpoint string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher := w.(http.Flusher)
			fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint)
			flusher.Flush()

			// Poll for pending requests and send responses.
			for i := 0; i < 50; i++ {
				time.Sleep(50 * time.Millisecond)
				mu.Lock()
				for id := range pending {
					resp := JSONRPCResponse{
						JSONRPC: "2.0",
						ID:      id,
						Result:  json.RawMessage(`{"tools":[]}`),
					}
					data, _ := json.Marshal(resp)
					fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(data))
					flusher.Flush()
					delete(pending, id)
					break
				}
				mu.Unlock()
			}
			return
		}
		if r.Method == "POST" {
			var req JSONRPCRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			pending[req.ID] = req
			mu.Unlock()
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer server.Close()

	endpoint = server.URL + "/msg"

	transport := NewSSETransport(server.URL)
	defer transport.Close()

	time.Sleep(300 * time.Millisecond)

	req := JSONRPCRequest{Method: "tools/list"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := transport.Send(ctx, req)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}
}

func TestSSETransportReconnection(t *testing.T) {
	requests := atomic.Int32{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			requests.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher := w.(http.Flusher)
			flusher.Flush()
			return
		}
	}))
	defer srv.Close()

	transport := NewSSETransport(srv.URL)
	defer transport.Close()

	time.Sleep(2 * time.Second)

	if requests.Load() < 2 {
		t.Fatalf("expected at least 2 connection attempts, got %d", requests.Load())
	}
}

func TestSSETransportClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		time.Sleep(50 * time.Millisecond)
	}))
	defer server.Close()

	transport := NewSSETransport(server.URL)

	if err := transport.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestStdioTransportSendAfterClose(t *testing.T) {
	skipIfNoEcho(t)
	transport, _ := NewStdioTransport(echoPath, nil, nil)
	transport.Close()
	ctx := context.Background()
	_, err := transport.Send(ctx, JSONRPCRequest{Method: "test"})
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func TestStdioTransportNotifyAfterClose(t *testing.T) {
	skipIfNoEcho(t)
	transport, _ := NewStdioTransport(echoPath, nil, nil)
	transport.Close()
	ctx := context.Background()
	err := transport.Notify(ctx, "test", nil)
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func TestStdioTransportReceiveInvalidJSON(t *testing.T) {
	script := `package main
import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req map[string]interface{}
		json.Unmarshal(scanner.Bytes(), &req)
		id, _ := req["id"].(float64)
		fmt.Println("not json")
		resp := map[string]interface{}{"jsonrpc":"2.0","id":id,"result":true}
		data, _ := json.Marshal(resp)
		fmt.Println(string(data))
	}
}`
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "srv.go")
	if err := os.WriteFile(src, []byte(script), 0o644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	bin := filepath.Join(tmpDir, "srv")
	if out, err := exec.Command("go", "build", "-o", bin, src).CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, out)
	}
	transport, err := NewStdioTransport(bin, nil, nil)
	if err != nil {
		t.Fatalf("create transport: %v", err)
	}
	defer transport.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := transport.Send(ctx, JSONRPCRequest{Method: "test"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.ID == 0 {
		t.Error("expected valid response after ignoring garbage")
	}
}

func TestSSETransportConnectFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	transport := NewSSETransport(srv.URL)
	defer transport.Close()
	// Allow a couple of reconnect attempts.
	time.Sleep(1500 * time.Millisecond)
}

func TestSSETransportSendAfterClose(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	transport := NewSSETransport(srv.URL)
	transport.Close()
	ctx := context.Background()
	_, err := transport.Send(ctx, JSONRPCRequest{Method: "test"})
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func TestSSETransportNotifyAfterClose(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	transport := NewSSETransport(srv.URL)
	transport.Close()
	ctx := context.Background()
	err := transport.Notify(ctx, "test", nil)
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func TestParseSSEComments(t *testing.T) {
	input := ":comment\nevent: msg\ndata: hello\n\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	ev, ok := parseSSE(scanner)
	if !ok {
		t.Fatal("expected event")
	}
	if ev.event != "msg" {
		t.Errorf("event = %q, want msg", ev.event)
	}
	if ev.data != "hello" {
		t.Errorf("data = %q, want hello", ev.data)
	}
}

func TestBaseTransportResolveUnknownID(t *testing.T) {
	bt := &baseTransport{pending: make(map[int64]*pendingRequest)}
	resolved := bt.resolveResponse(JSONRPCResponse{ID: 999})
	if resolved {
		t.Error("expected false for unknown id")
	}
}
