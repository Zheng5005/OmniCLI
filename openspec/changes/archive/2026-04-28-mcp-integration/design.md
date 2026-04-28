# Design: MCP Integration for OmniCLI

## Technical Approach

Add `internal/mcp/` as a self-contained package implementing MCP Host functionality: JSON-RPC 2.0 over Stdio/SSE transports, server lifecycle management, and tool/resource discovery. MCP tools wrap as `tools.Tool` interface instances and register into the existing registry — the agent loop remains unchanged. Trust gating happens inside the `MCPTool.Execute()` wrapper via the same channel-based approval pattern used by `RunCommandTool`. On server connect/disconnect, the agent session is recreated via the existing `RecreateSession()` pattern. Resources prepend to conversation context (system prompt injection). TUI gains two new slash commands (`/mcp`, `/attach`) and reuses `StateAwaitingApproval` for untrusted tool approval.

## Architecture Decisions

### Decision: MCP Tool Registration Strategy

**Choice**: `MCPTool` struct implementing `tools.Tool` interface with namespace-prefixed names (`server__tool`)
**Alternatives**: (A) Separate MCP registry with own dispatch; (B) Lazy registration on first call
**Rationale**: The agent loop only knows `tools.Tool`. Option A duplicates dispatch logic. Option B requires session rebuild mid-conversation anyway. Wrapping as `Tool` reuses the entire execution path — the agent loop, `buildToolDefs()`, and `executeTool()` all work without changes. Namespace prefixes (`postgres__query`) prevent name collisions across servers.

### Decision: Server Lifecycle Model

**Choice**: One goroutine per server with health monitor and exponential-backoff restart
**Alternatives**: (A) Single goroutine polling all servers; (B) Start/stop per-request
**Rationale**: Each server is independent (child process or HTTP). Per-server goroutines enable isolated crash recovery without blocking others. Per-request spawning would cause cold-start latency. The manager exposes a channel-based API (`Connect()`, `Disconnect()`, `Status()`) consumed by TUI status bar.

### Decision: Untrusted Tool Approval Flow

**Choice**: Reuse `ApprovalRequestMsg` channel pattern — `MCPTool.Execute()` blocks on approval channel for untrusted servers, auto-approves for trusted
**Alternatives**: (A) New `StateMcpApproval` TUI state; (B) Prompt inline in viewport text
**Rationale**: The existing approval pattern (`ApprovalRequestMsg` + response channel) is proven in `RunCommandTool`. Adding a new state duplicates state machine complexity. The TUI already handles `StateAwaitingApproval` — an MCP approval message just sets different text (`"MCP tool: postgres__query — args: {...}"`) and follows the same y/n flow.

### Decision: Dynamic Tool Registration & Session Rebuild

**Choice**: On every server connect/disconnect event, rebuild the full tool registry and call `Agent.RecreateSession()`
**Alternatives**: (A) Register tools once at startup, never update; (B) Lazy rebuild on next prompt
**Rationale**: OmniGo sessions require upfront tool schema registration. Option A fails if servers crash. Option B adds latency to the next user prompt. Proactive rebuild on state change is simple and leverages the existing `RecreateSession()` pattern (already used for skills).

### Decision: JSON-RPC Implementation

**Choice**: Implement JSON-RPC 2.0 directly in `internal/mcp/transport.go` with `encoding/json` + `bufio.Scanner`
**Alternatives**: (A) Import `modelcontextprotocol/go-sdk`; (B) Use a generic JSON-RPC library
**Rationale**: No official stable Go MCP SDK exists. The protocol is simple (JSON-RPC 2.0 over newline-delimited stdio). Direct implementation keeps dependencies minimal and gives full control over error handling and transport framing.

### Decision: Resource Attachment Strategy

**Choice**: Pinned resources prepended to system prompt, separated by `---` delimiters
**Alternatives**: (A) Inject as tool-call results; (B) Append after user message
**Rationale**: System prompt injection is the same pattern used by skills. Resources are context-providers (schemas, configs), not tool outputs. Tool-call injection would confuse the LLM. User message append would be invisible in the TUI.

## Data Flow

### Server Startup Sequence

```
main.go: start
     │
     ├── config.Load() → Config{McpServers: map[string]McpServerConfig}
     │
     ├── mcp.NewManager(cfg.McpServers)
     │      │
     │      ├── for each server config:
     │      │      ├── mcp.NewClient(serverConfig)
     │      │      │      ├── StdioTransport: os/exec.Cmd → stdin/stdout pipes
     │      │      │      └── SSETransport: HTTP client to remote URL
     │      │      │
     │      │      ├── client.Initialize() → capabilities negotiation
     │      │      ├── client.ListTools() → []McpToolDef
     │      │      │
     │      │      └── for each tool:
     │      │             registry.Register(&MCPTool{
     │      │                 name: "server__tool",
     │      │                 client: client,
     │      │                 trusted: config.Trusted,
     │      │                 approvalFn: approvalFn,
     │      │             })
     │      │
     │      └── go client.HealthMonitor() → restart on crash
     │
     └── agent.RecreateSession(systemPrompt, allToolNames, models)
```

### Tool Call Flow (Trusted Server)

```
Agent loop → executeTool("postgres__query", args)
     │
     ├── registry.Get("postgres__query") → *MCPTool
     ├── MCPTool.Execute(ctx, args)
     │      ├── trusted → skip approval
     │      ├── client.CallTool("query", args) → JSON-RPC → McpServer → result
     │      └── return result
     │
     └── Agent loop continues with tool result
```

### Tool Call Flow (Untrusted Server)

```
Agent loop → executeTool("hack__query", args)
     │
     ├── registry.Get("hack__query") → *MCPTool
     ├── MCPTool.Execute(ctx, args)
     │      ├── untrusted → send MCPApprovalRequestMsg via channel
     │      │                   │
     │      │         ┌───────▼────────────┐
     │      │         │ StateAwaitingApproval │
     │      │         │  "MCP: hack__query"   │
     │      │         │  [y] approve / [n] deny│
     │      │         └───────┬───────────────┘
     │      │                  │
     │      ├── approve → client.CallTool("query", args)
     │      ├── deny    → return "tool call denied by user"
     │      └── return result
     │
     └── Agent loop continues with tool result
```

### Resource Attachment Flow

```
User types "/attach"
     │
     ├── SlashRouter dispatches → McpResourceListMsg
     ├── TUI enters StateResourceBrowser
     ├── User selects resource → ResourceAttachMsg{URI, Server}
     ├── MCPClient.ReadResource(uri) → content
     ├── Prepend to system prompt: "[Attached Resource: {uri}]\n{content}\n---"
     └── agent.RecreateSession(updatedPrompt, tools, models)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/mcp/types.go` | Create | MCP protocol types: `InitializeParams/Result`, `Tool`, `Resource`, `CallToolParams/Result`, JSON-RPC envelope types |
| `internal/mcp/transport.go` | Create | `Transport` interface + `StdioTransport` (child process pipes) + `SSETransport` (HTTP SSE to remote) |
| `internal/mcp/client.go` | Create | `Client` struct: `Initialize()`, `ListTools()`, `CallTool()`, `ListResources()`, `ReadResource()`, `Close()` |
| `internal/mcp/server.go` | Create | `Manager` struct: `StartAll()`, `StopAll()`, `Status()` map; per-server goroutines with health monitor and backoff restart |
| `internal/mcp/mcp_tool.go` | Create | `MCPTool` implementing `tools.Tool`: namespace-prefixed name, delegates Execute to MCP client, gates with approval for untrusted |
| `internal/config/config.go` | Modify | Add `McpServers` field (`map[string]McpServerConfig`) to Config; add merge logic for MCP servers |
| `internal/config/config.go` | Modify | Add `McpServerConfig` struct: `Type`, `Command`, `Args`, `URL`, `Trusted`, `Env` |
| `internal/tools/registry.go` | Modify | Add `sync.Mutex` for thread-safe `Register`/`Deregister`; add `Deregister(serverPrefix string)` to remove namespaced tools |
| `internal/agent/agent.go` | Modify | Add `mcpmgr *mcp.Manager` field; add `SetMCPManager()` method |
| `internal/tui/model.go` | Modify | Add `StateResourceBrowser` state; add `mcpmgr *mcp.Manager`, `resourcePanel`, `pinnedResources` fields; handle MCP messages |
| `internal/tui/messages.go` | Modify | Add `MCPApprovalRequestMsg`, `McpResourceListMsg`, `ResourceAttachMsg`, `McpServerStatusMsg` |
| `internal/tui/slashrouter.go` | Modify | Register `/mcp` (list servers+tools) and `/attach` (resource picker) commands |
| `internal/tui/statusbar.go` | Modify | Add `mcpStatus` field; render server connection indicators in status bar |
| `internal/tui/resource_panel.go` | Create | Collapsible side panel for pinned MCP resources; Bubble Tea sub-model |
| `cmd/omni/main.go` | Modify | Wire MCP manager startup, register MCP tools, register `/mcp` and `/attach` handlers |
| `cmd/omni/mcp_cmd.go` | Create | `omni mcp add` CLI subcommand: validate config, perform handshake, write to omnisettings.json |

## Interfaces / Contracts

### McpServerConfig (`internal/config/config.go`)

```go
type McpServerConfig struct {
    Type    string            `json:"type"`    // "stdio" or "sse"
    Command string            `json:"command"`  // stdio: executable
    Args    []string          `json:"args"`     // stdio: arguments
    URL     string            `json:"url"`      // sse: remote endpoint
    Env     map[string]string `json:"env"`      // environment variables
    Trusted bool              `json:"trusted"`  // skip approval gate
}
```

### Transport interface (`internal/mcp/transport.go`)

```go
type Transport interface {
    Send(ctx context.Context, request JSONRPCRequest) (JSONRPCResponse, error)
    Close() error
}
```

### Client (`internal/mcp/client.go`)

```go
type Client struct {
    name     string
    config   McpServerConfig
    transport Transport
    tools    []McpToolDef
}

func NewClient(name string, config McpServerConfig, approvalFn MCPApprovalFunc) (*Client, error)
func (c *Client) Initialize(ctx context.Context) error
func (c *Client) ListTools(ctx context.Context) ([]McpToolDef, error)
func (c *Client) CallTool(ctx context.Context, toolName string, args json.RawMessage) (string, error)
func (c *Client) ListResources(ctx context.Context) ([]McpResource, error)
func (c *Client) ReadResource(ctx context.Context, uri string) (string, error)
func (c *Client) Close() error
```

### Manager (`internal/mcp/server.go`)

```go
type Manager struct {
    servers  map[string]*Client
    registry *tools.Registry
    mu       sync.Mutex
}

func NewManager(configs map[string]McpServerConfig, registry *tools.Registry, approvalFn MCPApprovalFunc) *Manager
func (m *Manager) StartAll(ctx context.Context) error    // spawn + initialize all
func (m *Manager) StopAll() error                        // SIGTERM all + wait
func (m *Manager) Status() map[string]ServerStatus       // for TUI status bar
func (m *Manager) ReconnectServer(name string) error      // manual reconnect
```

### MCPTool (`internal/mcp/mcp_tool.go`)

```go
type MCPApprovalFunc func(serverName, toolName, args string) (bool, error)

type MCPTool struct {
    qualifiedName string       // "server__tool"
    description   string
    parameters    json.RawMessage
    client        *Client
    trusted       bool
    approvalFn    MCPApprovalFunc
}

func (t *MCPTool) Name() string            { return t.qualifiedName }
func (t *MCPTool) Description() string     { return t.description }
func (t *MCPTool) Parameters() json.RawMessage { return t.parameters }
func (t *MCPTool) Execute(ctx context.Context, args json.RawMessage) (string, error)
```

### TUI Messages (`internal/tui/messages.go` additions)

```go
type McpApprovalRequestMsg struct {
    ServerName string
    ToolName   string
    Args       string
    ResponseCh chan bool
}

type McpServerStatusMsg struct {
    Name   string
    Status string // "connected", "error", "disconnected"
}

type ResourceAttachMsg struct {
    ServerName string
    URI        string
    Content    string
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | JSON-RPC message framing (encode/decode) | Table-driven tests with valid/invalid JSON-RPC messages |
| Unit | `StdioTransport` message send/receive | Mock server process via `os/exec` test binary; stdin/stdout pipe |
| Unit | `MCPTool.Execute` trusted vs untrusted approval | Mock client; verify trusted skips approval, untrusted sends `McpApprovalRequestMsg` |
| Unit | `Manager.StartAll` / `StopAll` lifecycle | Mock clients; verify registry gets tools registered/deregistered |
| Unit | `Config` merge with `McpServers` | Table-driven: global-only, project-only, both (project overrides) |
| Unit | Tool namespace collision | Register two servers with same tool name; verify `server__` prefix disambiguates |
| Integration | Full tool call round-trip | Start mock MCP server (test binary), `Initialize` → `ListTools` → `CallTool` → verify result |
| Integration | Server crash recovery | Kill child process mid-session; verify health monitor restarts and re-registers tools |
| Integration | Agent loop with MCP tool | Agent with `MCPTool` in registry; verify `executeTool()` calls MCP client, returns result |
| E2E | `omni mcp add` handshake | CLI test: add server config, validate connection, write to config file |

## Migration / Rollout

No migration required. All changes are additive:
- `McpServers` config field: zero-value means "no MCP servers" — existing configs work unchanged
- `internal/mcp/` package: only imported when MCP servers are configured
- Tool registry gains thread-safe mutex — existing static registration (no concurrent access) is unaffected
- TUI adds `/mcp` and `/attach` commands — existing commands unmodified

## Open Questions

- [ ] Should MCP tool calls from trusted servers be completely silent or show a brief status bar notification (e.g., `[✓] postgres:query`)? PRD says status bar notification — needs TUI implementation.
- [ ] How to handle MCP server timeout? Currently no timeout on `CallTool`. Proposal: 30s default, configurable per-server in `McpServerConfig.Timeout`.
- [ ] Should `/attach` resources persist across sessions (resume) or be session-scoped only? Proposal: session-scoped for Phase 1.