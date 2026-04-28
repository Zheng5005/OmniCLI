package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// healthCheckInterval is the interval between health checks. It is a variable
// so tests can override it for faster execution.
var healthCheckInterval = 5 * time.Second

// Manager coordinates multiple MCP server clients, handling lifecycle,
// health monitoring, crash recovery, and graceful shutdown.
type Manager struct {
	configs map[string]ServerConfig
	clients map[string]*Client
	status  map[string]ServerStatus
	mu      sync.RWMutex
	wg      sync.WaitGroup
	done    chan struct{}
	stopOnce sync.Once
}

// NewManager creates a new manager with the given server configurations.
func NewManager(configs map[string]ServerConfig) *Manager {
	if configs == nil {
		configs = map[string]ServerConfig{}
	}
	return &Manager{
		configs: configs,
		clients: make(map[string]*Client),
		status:  make(map[string]ServerStatus),
		done:    make(chan struct{}),
	}
}

// StartAll initializes every configured server. Servers that fail handshake
// are marked ERROR but do not block the others.
func (m *Manager) StartAll(ctx context.Context) error {
	for name, cfg := range m.configs {
		if err := m.startServer(ctx, name, cfg); err != nil {
			m.setStatus(name, ServerStatus{
				State:   "error",
				Healthy: false,
				Error:   err.Error(),
			})
		}
	}
	return nil
}

func (m *Manager) startServer(ctx context.Context, name string, cfg ServerConfig) error {
	var transport Transport
	var err error

	switch cfg.Type {
	case "stdio":
		transport, err = NewStdioTransport(cfg.Command, cfg.Args, cfg.Env)
	case "sse":
		transport = NewSSETransport(cfg.URL)
	default:
		return fmt.Errorf("unknown transport type %q", cfg.Type)
	}
	if err != nil {
		return fmt.Errorf("create transport: %w", err)
	}

	client := NewClient(name, cfg, transport)

	timeout := time.Duration(cfg.RequestTimeout()) * time.Second
	initCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := client.Initialize(initCtx); err != nil {
		transport.Close()
		return fmt.Errorf("initialize: %w", err)
	}

	m.mu.Lock()
	m.clients[name] = client
	m.status[name] = ServerStatus{
		State:   "ready",
		Healthy: true,
	}
	m.mu.Unlock()

	// Start health monitor.
	m.wg.Add(1)
	go m.healthMonitor(name, client)

	return nil
}

func (m *Manager) setStatus(name string, st ServerStatus) {
	m.mu.Lock()
	m.status[name] = st
	m.mu.Unlock()
}

// healthMonitor watches a single server and restarts it on failure.
func (m *Manager) healthMonitor(name string, client *Client) {
	defer m.wg.Done()

	backoff := time.Second
	maxBackoff := 30 * time.Second
	failures := 0
	const maxFailures = 5

	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			if !m.isRunning(name) {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := m.healthCheck(ctx, client)
			cancel()

			if err != nil {
				failures++
				m.setStatus(name, ServerStatus{
					State:   "error",
					Healthy: false,
					Error:   err.Error(),
				})

				if failures >= maxFailures {
					m.setStatus(name, ServerStatus{
						State:   "failed",
						Healthy: false,
						Error:   "max failures reached",
					})
					return
				}

				// Attempt restart after backoff.
				select {
				case <-time.After(backoff):
				case <-m.done:
					return
				}

				m.stopClient(name)

				m.mu.RLock()
				cfg := m.configs[name]
				m.mu.RUnlock()

				ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
				if err := m.startServer(ctx2, name, cfg); err != nil {
					cancel2()
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
					continue
				}
				cancel2()

				// Restart succeeded; the new health monitor started by
				// startServer takes over. Old monitor exits.
				return
			} else {
				failures = 0
				backoff = time.Second
				m.mu.Lock()
				if st := m.status[name]; st.State != "ready" {
					m.status[name] = ServerStatus{
						State:   "ready",
						Healthy: true,
					}
				}
				m.mu.Unlock()
			}
		}
	}
}

func (m *Manager) healthCheck(ctx context.Context, client *Client) error {
	_, err := client.ListTools(ctx)
	return err
}

func (m *Manager) isRunning(name string) bool {
	m.mu.RLock()
	_, ok := m.clients[name]
	m.mu.RUnlock()
	return ok
}

// StopAll gracefully shuts down all servers. It sends SIGTERM and waits up
// to 5 seconds before forcefully killing each process.
func (m *Manager) StopAll() error {
	m.stopOnce.Do(func() {
		close(m.done)
	})

	m.mu.RLock()
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		m.stopClient(name)
	}

	m.wg.Wait()
	return nil
}

func (m *Manager) stopClient(name string) {
	m.mu.Lock()
	client, ok := m.clients[name]
	delete(m.clients, name)
	m.mu.Unlock()

	if !ok {
		return
	}

	client.Close()

	m.setStatus(name, ServerStatus{
		State:   "disconnected",
		Healthy: false,
	})
}

// Status returns a snapshot of each server's current state.
func (m *Manager) Status() map[string]ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]ServerStatus, len(m.status))
	for k, v := range m.status {
		result[k] = v
	}
	return result
}

// ReconnectServer stops and restarts a single server.
func (m *Manager) ReconnectServer(ctx context.Context, name string) error {
	m.mu.RLock()
	cfg, ok := m.configs[name]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("server not found: %s", name)
	}

	m.stopClient(name)
	return m.startServer(ctx, name, cfg)
}

// Client returns the named client or nil if not connected.
func (m *Manager) Client(name string) *Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[name]
}
