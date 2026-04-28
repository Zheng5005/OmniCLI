# Tasks: MCP Integration for OmniCLI

## Phase 1: Foundation & MCP Protocol (Tasks 1.1–1.8)

- [x] 1.1 Create `internal/mcp/types.go` with JSON-RPC 2.0 envelope types (`JSONRPCRequest`, `JSONRPCResponse`, `JSONRPCError`), MCP protocol types (`InitializeParams`, `InitializeResult`, `Tool`, `Resource`, `CallToolParams`, `CallToolResult`)
- [x] 1.2 Create `internal/mcp/transport.go` with `Transport` interface and `StdioTransport` struct implementing `Send()` and `Close()` using `os/exec` pipes with newline-delimited JSON framing
- [x] 1.3 Add `SSETransport` struct to `internal/mcp/transport.go` using `net/http` for HTTP SSE with reconnection logic (exponential backoff, max 5 retries)
- [x] 1.4 Create `internal/mcp/client.go` with `Client` struct implementing `Initialize()`, `ListTools()`, `CallTool()`, `ListResources()`, `ReadResource()`, `Close()` methods
- [x] 1.5 Create `internal/mcp/server.go` with `Manager` struct: `StartAll()`, `StopAll()`, `Status()`, `ReconnectServer()`; per-server goroutines with health monitor and crash recovery
- [x] 1.6 Create `internal/mcp/mcp_tool.go` with `MCPTool` struct implementing `tools.Tool` interface; namespace-prefixed names (`server__tool`); `Execute()` delegates to client with approval gating
- [x] 1.7 Add `McpServerConfig` struct to `internal/config/config.go`: `Type`, `Command`, `Args`, `URL`, `Env`, `Trusted`, `Timeout`; add `McpServers` map field to `Config` struct
- [x] 1.8 Add config merge logic in `internal/config/config.go` for `McpServers`: project config merges with global (no override), validation for required fields per transport type

## Phase 2: Tool Registry & Agent Integration (Tasks 2.1–2.6)

- [x] 2.1 Add `sync.Mutex` to `internal/tools/registry.go`; modify `Register()` to be thread-safe; add `Deregister(serverPrefix string)` method to remove namespaced tools
- [x] 2.2 Modify `internal/agent/agent.go`: add `mcpmgr *mcp.Manager` field; add `SetMCPManager()` method; wire session recreation on tool changes
- [x] 2.3 Modify `cmd/omni/main.go`: instantiate `mcp.Manager` after config load; call `StartAll()` before REPL; defer `StopAll()` on shutdown
- [x] 2.4 Wire MCP tool registration in `cmd/omni/main.go`: iterate `manager.Status()`, call `ListTools()` per server, register each as `MCPTool` in tool registry
- [x] 2.5 Modify `internal/agent/loop.go`: include MCP tool schemas in `buildToolDefs()`; ensure `RecreateSession()` is called on server connect/disconnect events
- [x] 2.6 Implement approval callback in `cmd/omni/main.go`: `MCPApprovalFunc` that sends `MCPApprovalRequestMsg` via channel for untrusted servers, auto-approves trusted

## Phase 3: TUI Integration — Approval & Status (Tasks 3.1–3.7)

- [x] 3.1 Add `MCPApprovalRequestMsg`, `McpServerStatusMsg` to `internal/tui/messages.go` with appropriate channels for synchronous approval response
- [x] 3.2 Modify `internal/tui/model.go`: add `mcpmgr *mcp.Manager`, `pinnedResources` slice; handle `MCPApprovalRequestMsg` in `Update()` setting `StateAwaitingApproval`
- [x] 3.3 Modify `internal/tui/statusbar.go`: add `mcpStatus` field; render server indicators (green=connected, red=error, gray=disconnected) alongside existing status
- [x] 3.4 Modify `internal/tui/slashrouter.go`: register `/mcp` command dispatching to `McpServerListMsg` showing server name, status, tool count, trust level
- [x] 3.5 Create `internal/tui/approval_view.go`: render MCP-specific approval dialog with server name, tool name, description, formatted JSON args, Approve/Deny options
- [x] 3.6 Wire approval flow in `internal/tui/update.go`: on user approval, send `true` on response channel; on deny, send `false` with error message to agent
- [x] 3.7 Add trusted tool notification in `internal/tui/statusbar.go`: brief `[✓] server:tool` flash on successful trusted tool execution

## Phase 4: CLI Commands (Tasks 4.1–4.4)

- [x] 4.1 Create `cmd/omni/mcp_cmd.go`: implement `omni mcp add <name> --type stdio|sse --command ... --args ... --url ... --trusted` with flag parsing
- [x] 4.2 Implement handshake validation in `cmd/omni/mcp_cmd.go`: spawn/connect to server, call `initialize`, verify response; exit non-zero on failure
- [x] 4.3 Implement config write in `cmd/omni/mcp_cmd.go`: append server to `omnisettings.json` `McpServers` block; handle duplicate name with confirmation prompt
- [x] 4.4 Implement `omni mcp list` and `omni mcp remove <name>` commands in `cmd/omni/mcp_cmd.go`: list shows status; remove deletes from config and shuts down if running

## Phase 5: MCP Resources (Phase 2) (Tasks 5.1–5.7)

- [x] 5.1 Implement `ListResources()` and `ReadResource()` in `internal/mcp/client.go`: call `resources/list` and `resources/read` JSON-RPC methods; cache results
- [x] 5.2 Add `ResourceAttachMsg`, `McpResourceListMsg` to `internal/tui/messages.go` for resource picker communication
- [x] 5.3 Create `internal/tui/resource_panel.go`: Bubble Tea sub-model for collapsible side panel; display pinned resources with name, server, truncated preview
- [x] 5.4 Modify `internal/tui/model.go`: add `StateResourceBrowser`, `resourcePanel` field; wire panel toggle and resource selection handling
- [x] 5.5 Register `/attach` in `internal/tui/slashrouter.go`: dispatch to `McpResourceListMsg`; enter `StateResourceBrowser` with navigable resource list
- [x] 5.6 Implement resource attachment in `internal/agent/agent.go`: prepend pinned resource content to system prompt with `---` delimiters; trigger `RecreateSession()`
- [x] 5.7 Add unpin functionality in `internal/tui/resource_panel.go`: keyboard shortcut to remove resource from `pinnedResources` slice; update session

## Phase 6: Testing — Unit Tests (Tasks 6.1–6.8)

- [x] 6.1 Write unit tests for `internal/mcp/types.go`: table-driven tests for JSON-RPC message construction, parsing, error handling
- [x] 6.2 Write unit tests for `internal/mcp/transport.go`: mock server process for `StdioTransport.Send()`; verify newline-delimited framing and response correlation
- [x] 6.3 Write unit tests for `internal/mcp/client.go`: mock transport; test `Initialize()`, `ListTools()`, `CallTool()` with valid/invalid responses
- [x] 6.4 Write unit tests for `internal/mcp/server.go`: mock clients; verify `StartAll()` spawns goroutines, `StopAll()` sends SIGTERM, health monitor restarts on crash
- [x] 6.5 Write unit tests for `internal/mcp/mcp_tool.go`: verify trusted skips approval, untrusted sends `MCPApprovalRequestMsg`; test namespace prefixing
- [x] 6.6 Write unit tests for `internal/config/config.go`: table-driven tests for `McpServers` merge (global-only, project-only, both), validation errors
- [x] 6.7 Write unit tests for `internal/tools/registry.go`: verify thread-safe `Register()`/`Deregister()`, session recreation trigger on tool changes
- [x] 6.8 Write unit tests for `internal/tui/approval_view.go`: verify dialog renders server/tool name, args; keyboard navigation selects correct option

## Phase 7: Testing — Integration & E2E (Tasks 7.1–7.5)

- [x] 7.1 Create mock MCP server binary in `test/mcp/mock_server.go`: responds to `initialize`, `tools/list`, `tools/call`; use in integration tests
- [x] 7.2 Write integration test for full tool call round-trip: start mock server, `Initialize()` → `ListTools()` → `CallTool()` → verify result in agent loop
- [x] 7.3 Write integration test for crash recovery: kill child process mid-session; verify health monitor restarts and re-registers tools within 5s
- [x] 7.4 Write E2E test for `omni mcp add`: CLI command with mock server; verify config file updated, handshake validated
- [x] 7.5 Run full test suite: verify all existing 62 tests pass; add MCP tests to CI; target 85%+ coverage for `internal/mcp/`

## Phase 8: Documentation & Cleanup (Tasks 8.1–8.3)

- [x] 8.1 Add Go doc comments to all public types/functions in `internal/mcp/`; add README.md with MCP architecture overview and usage examples
- [x] 8.2 Update `CONTRIBUTING.md`: document how to test MCP integration locally with `mcp-server-filesystem`; add troubleshooting section for common issues
- [x] 8.3 Remove temporary debug logging from `internal/mcp/`; verify no hardcoded test paths; ensure graceful shutdown works with `go test -timeout 30s`

---

## Implementation Order Summary

| Phase | Tasks | Focus |
|-------|-------|-------|
| Phase 1 | 1.1–1.8 | MCP protocol foundation: types, transport, client, manager, config |
| Phase 2 | 2.1–2.6 | Tool registry thread-safety, agent wiring, MCP tool registration |
| Phase 3 | 3.1–3.7 | TUI: approval dialog, status bar indicators, `/mcp` command |
| Phase 4 | 4.1–4.4 | CLI: `omni mcp add/list/remove` commands with handshake validation |
| Phase 5 | 5.1–5.7 | MCP resources: `/attach`, resource panel, system prompt injection |
| Phase 6 | 6.1–6.8 | Unit tests for all MCP components |
| Phase 7 | 7.1–7.5 | Integration and E2E tests |
| Phase 8 | 8.1–8.3 | Documentation and cleanup |

**Total: 40 tasks**

### Critical Path
1.1 → 1.2 → 1.4 → 1.5 → 1.6 → 2.1 → 2.2 → 2.3 → 2.4 → 3.1 → 3.2 → 6.1–6.8 → 7.1–7.5

Phase 5 (Resources) can be deferred to a separate PR if needed — it's isolated behind feature flag (no resources configured = no UX changes).

---

## Change Status: ✅ COMPLETE

**Archived**: 2026-04-28  
**Archive Location**: `openspec/changes/archive/2026-04-28-mcp-integration/`  
**Verification**: PASS WITH WARNINGS (43/54 compliant, 8 partial, 3 untested)

All 40 tasks implemented. Build, vet, and tests pass. Two CRITICAL issues fixed during implementation:
1. Economic tracking for MCP data tokens
2. Tool schema compatibility validation

See verify-report.md for detailed compliance matrix and remaining warnings.

---

## Critical Fixes (Post-Implementation)

- [x] **Economic tracking for MCP data tokens** — `internal/agent/loop.go` now estimates MCP response tokens (chars/4) and includes them in `CostUpdateMsg`/`AgentDoneMsg` using the model's input rate derived from `Usage.InputCost`.
- [x] **Tool schema compatibility validation** — `internal/mcp/mcp_tool.go` adds `ValidateSchema()` to detect `$ref`, `oneOf`, `anyOf`, `allOf`. `cmd/omni/main.go` skips unsupported tools with a warning naming the server, tool, and feature.
