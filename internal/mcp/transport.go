package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Transport defines the interface for MCP transports.
type Transport interface {
	// Send sends a JSON-RPC request and waits for the correlated response.
	Send(ctx context.Context, req JSONRPCRequest) (JSONRPCResponse, error)
	// Notify sends a JSON-RPC notification (no response expected).
	Notify(ctx context.Context, method string, params json.RawMessage) error
	// Close terminates the transport and cleans up resources.
	Close() error
}

// pendingRequest tracks an in-flight request.
type pendingRequest struct {
	respCh chan JSONRPCResponse
}

// baseTransport contains shared request/response correlation logic.
type baseTransport struct {
	pending map[int64]*pendingRequest
	mu      sync.Mutex
	idGen   idGenerator
}

func (t *baseTransport) nextID() int64 {
	return t.idGen.Next()
}

func (t *baseTransport) registerRequest(id int64) *pendingRequest {
	pr := &pendingRequest{
		respCh: make(chan JSONRPCResponse, 1),
	}
	t.mu.Lock()
	t.pending[id] = pr
	t.mu.Unlock()
	return pr
}

func (t *baseTransport) resolveResponse(resp JSONRPCResponse) bool {
	t.mu.Lock()
	pr, ok := t.pending[resp.ID]
	if ok {
		delete(t.pending, resp.ID)
	}
	t.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case pr.respCh <- resp:
	default:
	}
	return true
}

func (t *baseTransport) cancelRequest(id int64) {
	t.mu.Lock()
	delete(t.pending, id)
	t.mu.Unlock()
}

// StdioTransport implements Transport using a child process's stdin/stdout.
type StdioTransport struct {
	baseTransport
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	done   chan struct{}
	wg     sync.WaitGroup
	closed atomic.Bool
}

// NewStdioTransport spawns a child process and returns a transport connected
// to its stdin/stdout. Stderr is captured for diagnostics.
func NewStdioTransport(command string, args []string, env map[string]string) (*StdioTransport, error) {
	cmd := exec.Command(command, args...)

	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command: %w", err)
	}

	t := &StdioTransport{
		baseTransport: baseTransport{
			pending: make(map[int64]*pendingRequest),
		},
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		done:   make(chan struct{}),
	}

	t.wg.Add(2)
	go t.readStdout()
	go t.readStderr()

	return t, nil
}

func (t *StdioTransport) readStdout() {
	defer t.wg.Done()
	scanner := bufio.NewScanner(t.stdout)
	for scanner.Scan() {
		select {
		case <-t.done:
			return
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}

		t.resolveResponse(resp)
	}
}

func (t *StdioTransport) readStderr() {
	defer t.wg.Done()
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		select {
		case <-t.done:
			return
		default:
		}
		// Stderr is captured for diagnostics but not treated as protocol messages.
		_ = scanner.Text()
	}
}

// Send sends a JSON-RPC request and waits for the correlated response.
func (t *StdioTransport) Send(ctx context.Context, request JSONRPCRequest) (JSONRPCResponse, error) {
	if t.closed.Load() {
		return JSONRPCResponse{}, fmt.Errorf("transport closed")
	}

	if request.ID == 0 {
		request.ID = t.nextID()
	}
	request.JSONRPC = "2.0"

	pr := t.registerRequest(request.ID)

	data, err := json.Marshal(request)
	if err != nil {
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, fmt.Errorf("write request: %w", err)
	}

	select {
	case resp := <-pr.respCh:
		if resp.Error != nil {
			return resp, resp.Error
		}
		return resp, nil
	case <-ctx.Done():
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, ctx.Err()
	case <-t.done:
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, fmt.Errorf("transport closed")
	}
}

// Notify sends a JSON-RPC notification without waiting for a response.
func (t *StdioTransport) Notify(ctx context.Context, method string, params json.RawMessage) error {
	if t.closed.Load() {
		return fmt.Errorf("transport closed")
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("write notification: %w", err)
	}

	return nil
}

// Close terminates the child process and cleans up resources.
// It sends SIGTERM, waits up to 5 seconds, then sends SIGKILL.
func (t *StdioTransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}

	close(t.done)
	_ = t.stdin.Close()

	if t.cmd.Process != nil {
		_ = t.cmd.Process.Signal(termSignal())

		done := make(chan error, 1)
		go func() {
			done <- t.cmd.Wait()
		}()

		select {
		case <-done:
			// exited cleanly
		case <-time.After(5 * time.Second):
			_ = t.cmd.Process.Kill()
			<-done
		}
	}

	t.wg.Wait()
	return nil
}

// sseEvent represents a parsed Server-Sent Event.
type sseEvent struct {
	event string
	data  string
}

// parseSSE reads a single SSE event from the scanner.
func parseSSE(scanner *bufio.Scanner) (*sseEvent, bool) {
	var ev sseEvent
	var data strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if data.Len() > 0 {
				ev.data = strings.TrimSuffix(data.String(), "\n")
				return &ev, true
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			ev.event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			text := strings.TrimPrefix(line, "data:")
			if len(text) > 0 && text[0] == ' ' {
				text = text[1:]
			}
			data.WriteString(text)
			data.WriteString("\n")
		}
	}

	if data.Len() > 0 {
		ev.data = strings.TrimSuffix(data.String(), "\n")
		return &ev, true
	}

	return nil, false
}

// SSETransport implements Transport over HTTP Server-Sent Events.
type SSETransport struct {
	baseTransport
	url        string
	postURL    string
	client     *http.Client
	done       chan struct{}
	wg         sync.WaitGroup
	closed     atomic.Bool
	mu         sync.RWMutex
	streamDone chan struct{}
}

// NewSSETransport creates a new SSE transport connected to the given URL.
func NewSSETransport(url string) *SSETransport {
	t := &SSETransport{
		baseTransport: baseTransport{
			pending: make(map[int64]*pendingRequest),
		},
		url:        url,
		client:     &http.Client{Timeout: 0},
		done:       make(chan struct{}),
		streamDone: make(chan struct{}),
	}
	t.wg.Add(1)
	go t.run()
	return t
}

func (t *SSETransport) setPostURL(u string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	u = strings.TrimSpace(u)
	if u == "" {
		return
	}
	// If relative, resolve against base URL.
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		base, err := url.Parse(t.url)
		if err == nil {
			ref, err := url.Parse(u)
			if err == nil {
				t.postURL = base.ResolveReference(ref).String()
				return
			}
		}
	}
	t.postURL = u
}

func (t *SSETransport) getPostURL() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.postURL
}

func (t *SSETransport) run() {
	defer t.wg.Done()

	backoff := time.Second
	maxBackoff := 30 * time.Second
	retries := 0
	const maxRetries = 5

	for {
		select {
		case <-t.done:
			return
		default:
		}

		if err := t.connect(); err != nil {
			retries++
			if retries > maxRetries {
				return
			}
			select {
			case <-time.After(backoff):
			case <-t.done:
				return
			}
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		retries = 0
		backoff = time.Second

		// Wait for the stream to close before reconnecting.
		select {
		case <-t.streamDone:
		case <-t.done:
			return
		}
		t.mu.Lock()
		t.streamDone = make(chan struct{})
		t.mu.Unlock()
	}
}

func (t *SSETransport) connect() error {
	req, err := http.NewRequestWithContext(context.Background(), "GET", t.url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	go t.readEvents(resp.Body)
	return nil
}

func (t *SSETransport) readEvents(body io.ReadCloser) {
	defer body.Close()
	defer func() {
		t.mu.RLock()
		ch := t.streamDone
		t.mu.RUnlock()
		if ch != nil {
			close(ch)
		}
	}()

	scanner := bufio.NewScanner(body)

	for {
		ev, ok := parseSSE(scanner)
		if !ok {
			return
		}

		if ev.event == "endpoint" {
			t.setPostURL(ev.data)
			continue
		}

		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(ev.data), &resp); err != nil {
			continue
		}

		t.resolveResponse(resp)
	}
}

// Send sends a JSON-RPC request via POST and waits for the correlated SSE response.
func (t *SSETransport) Send(ctx context.Context, request JSONRPCRequest) (JSONRPCResponse, error) {
	if t.closed.Load() {
		return JSONRPCResponse{}, fmt.Errorf("transport closed")
	}

	if request.ID == 0 {
		request.ID = t.nextID()
	}
	request.JSONRPC = "2.0"

	pr := t.registerRequest(request.ID)

	data, err := json.Marshal(request)
	if err != nil {
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	postURL := t.getPostURL()
	if postURL == "" {
		// Fallback: assume POST to the same URL.
		postURL = t.url
	}

	req, err := http.NewRequestWithContext(ctx, "POST", postURL, bytes.NewReader(data))
	if err != nil {
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, fmt.Errorf("post request: %w", err)
	}
	_ = resp.Body.Close()

	select {
	case result := <-pr.respCh:
		if result.Error != nil {
			return result, result.Error
		}
		return result, nil
	case <-ctx.Done():
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, ctx.Err()
	case <-t.done:
		t.cancelRequest(request.ID)
		return JSONRPCResponse{}, fmt.Errorf("transport closed")
	}
}

// Notify sends a JSON-RPC notification via POST without waiting for a response.
func (t *SSETransport) Notify(ctx context.Context, method string, params json.RawMessage) error {
	if t.closed.Load() {
		return fmt.Errorf("transport closed")
	}

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal notification: %w", err)
	}

	postURL := t.getPostURL()
	if postURL == "" {
		postURL = t.url
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", postURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("post notification: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}

// Close shuts down the SSE transport and stops reconnection attempts.
func (t *SSETransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}

	close(t.done)
	t.wg.Wait()
	return nil
}
