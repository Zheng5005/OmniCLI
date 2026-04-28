# Proposal: MCP Integration for OmniCLI

## Intent

Enable OmniCLI to act as an MCP Host, connecting to standardized external servers (databases, web services, third-party platforms) via the Model Context Protocol. This transcends local-only tool execution, giving the agent access to a growing ecosystem of MCP servers without requiring new code for every integration.

## Scope

### In Scope
- MCP Host implementation with Stdio (local child processes) and SSE (remote HTTP) transports
- Server lifecycle: auto-start on boot, health monitoring, graceful shutdown (SIGTERM)
- Config block `mcp_servers` in `omnisettings.json` with `trusted` flag per server
- CLI: `omni mcp add <name> <transport-details>` with handshake test
- Dynamic tool discovery (`tools/list`) and registration into the agent's tool registry
- Token-only billing: MCP data tokens counted in LLM cost, no external pass-through
- TUI: `/mcp` slash command to list connected servers and their tools
- TUI: `/attach` command for MCP resources, collapsible "Pinned Resources" side panel
- Tool approval: untrusted = visual confirmation box, trusted = silent with status bar notification

### Out of Scope
- MCP Prompts (`prompts/list`, `prompts/get`) — deferred to future iteration
- MCP sampling (host acting as client for server-initiated LLM calls)
- Streamable HTTP transport (SSE-only for remote; Streamable HTTP is a newer spec variant)
- External cost pass-through for remote SSE server usage fees
- MCP server development — OmniCLI is host only, not a server

## Capabilities

### New Capabilities
- `mcp-transport`: JSON-RPC protocol implementation over Stdio and SSE transports, message framing, request/response correlation
- `mcp-lifecycle`: Server process management — spawn, health check, crash recovery, graceful SIGTERM shutdown
- `mcp-tools`: Dynamic tool discovery from MCP servers, registration into agent tool registry, tool execution delegation
- `mcp-resources`: Resource listing, reading, and attachment to conversation context; TUI resource browser side panel
- `mcp-config`: `mcp_servers` config schema in omnisettings.json, `omni mcp` CLI subcommands for server management
- `mcp-approval`: Trust-based approval model — visual confirmation for untrusted servers, silent execution with status bar for trusted

### Modified Capabilities
- `omnicli-v1`: Config schema extended with `mcp_servers` block; TUI gains new states (`StateMcpApproval`, `StateResourceBrowser`); slash router gains `/mcp` and `/attach`; agent loop handles MCP tool execution delegation; status bar shows MCP server health

## Approach

**Phased delivery within a single change:**

**Phase 1 — MCP Tools (foundation):**
1. New `internal/mcp/` package: `transport.go` (Stdio + SSE), `client.go` (JSON-RPC), `server.go` (lifecycle), `types.go` (protocol types)
2. Config: extend `Config` struct with `McpServers` map; two-tier merge applies
3. Startup: wire MCP server manager in `cmd/omni/main.go`; auto-start configured servers; run `tools/list` handshake
4. Agent: dynamically register MCP tools as `MCPTool` wrappers in the tool registry; agent loop delegates execution to MCP client
5. TUI: add `/mcp` slash command; trusted/untrusted approval UX (new state or extend `StateAwaitingApproval`)
6. CLI: `omni mcp add` subcommand with handshake validation

**Phase 2 — MCP Resources (UX layer):**
1. MCP client: implement `resources/list` and `resources/read`
2. TUI: build collapsible "Pinned Resources" side panel (`resource_panel.go`)
3. Slash router: add `/attach` command with resource picker
4. Agent: prepend pinned resource content to system prompt / conversation context
5. Status bar: show MCP server connection status (healthy/error/disconnected)

**Key architectural decisions:**
- MCP tools integrate as dynamic entries in the existing tool registry — no separate execution path
- Server lifecycle runs as a goroutine pool managed at startup, not per-request
- SSE transport uses `gorilla/websocket` (already in go.mod) upgraded to SSE; Stdio uses `os/exec`
- No external MCP SDK — implement JSON-RPC 2.0 directly to minimize dependencies

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/mcp/` | New | MCP protocol package: transport, client, server lifecycle, types |
| `internal/config/config.go` | Modified | Add `McpServers` field to Config struct |
| `internal/tools/registry.go` | Modified | Support dynamic tool registration/deregistration |
| `internal/agent/loop.go` | Modified | Handle MCP tool execution (delegate to MCP client) |
| `internal/agent/omnigo.go` | Modified | Include MCP tool schemas in LLM session |
| `internal/tui/model.go` | Modified | New states, MCP messages, resource panel integration |
| `internal/tui/messages.go` | Modified | MCP message types (approval, server status, resource attach) |
| `internal/tui/slashrouter.go` | Modified | Register `/mcp`, `/attach` commands |
| `internal/tui/statusbar.go` | Modified | Show MCP server health status |
| `internal/tui/resource_panel.go` | New | Collapsible side panel for pinned resources |
| `cmd/omni/main.go` | Modified | Wire MCP manager, register slash commands |
| `cmd/omni/mcp_cmd.go` | New | `omni mcp add` CLI subcommand |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| No MCP Go SDK — must implement JSON-RPC from scratch | High | MCP spec is well-documented (JSON-RPC 2.0); start with stdio transport (simpler) |
| Dynamic tool registration breaks existing static registry | Medium | Make registry thread-safe with mutex; trigger session recreation on tool changes |
| Child process crashes during session | Medium | Health monitor goroutine with exponential backoff restart; TUI shows ERROR state |
| OmniGo session recreation on tool list change | Medium | Already has `RecreateSession()` pattern; trigger on MCP server connect/disconnect events |
| SSE transport complexity (connection drops, reconnection) | Medium | Phase 2 scope; leverage existing `gorilla/websocket` dependency; implement reconnect with backoff |
| MCP server availability for testing | High | Mock JSON-RPC server for unit tests; use `mcp-server-filesystem` for integration tests |

## Rollback Plan

1. MCP code lives in isolated `internal/mcp/` package — remove package and revert config/TUI changes
2. Config changes are additive (`McpServers` field) — existing configs without it work unchanged
3. Dynamic tool registration is opt-in — if no servers configured, registry behaves identically
4. Git revert of the entire change branch if integration issues are unresolvable

## Dependencies

- MCP specification 2024-11-05 (JSON-RPC 2.0 based)
- `gorilla/websocket` (already in go.mod) for SSE transport
- Test MCP server: `mcp-server-filesystem` or similar for integration testing

## Success Criteria

- [ ] Configured MCP servers start automatically on OmniCLI boot
- [ ] `tools/list` discovers and registers MCP tools into the agent's tool registry
- [ ] Agent can execute MCP tools and receive results in the conversation loop
- [ ] `omni mcp add` performs handshake and adds server to config
- [ ] Untrusted server tool calls show visual confirmation box in TUI
- [ ] Trusted server tool calls execute silently with status bar notification
- [ ] `/attach` lists and pins MCP resources to conversation context
- [ ] Pinned resources appear in collapsible side panel
- [ ] Graceful shutdown sends SIGTERM to all child processes
- [ ] Crashed server shows ERROR state in status bar
- [ ] All existing 62 tests continue to pass
- [ ] New tests cover MCP transport, lifecycle, and tool execution
