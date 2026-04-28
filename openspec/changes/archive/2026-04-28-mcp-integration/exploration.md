# Exploration: MCP Integration for OmniCLI

## Current State

### Package Structure (8 internal packages)

| Package | Responsibility | Key Files |
|---------|---------------|-----------|
| `internal/tui/` | Bubble Tea TUI layer — model, input, viewport, statusbar, wizard, slash router | `model.go`, `input.go`, `viewport.go`, `statusbar.go`, `wizard.go`, `slashrouter.go`, `messages.go` |
| `internal/agent/` | LLM orchestration loop, OmniGo adapter, model resolution | `agent.go`, `loop.go`, `omnigo.go`, `resolve.go` |
| `internal/tools/` | Tool interface and built-in tools (list_files, grep_search, read_file, run_command) | `registry.go`, `list_files.go`, `grep_search.go`, `read_file.go`, `run_command.go`, `ignore.go` |
| `internal/skills/` | Skill/plugin system — JSON loading, variable injection, manager | `skill.go`, `manager.go`, `generator.go` |
| `internal/config/` | Two-tier JSON config loading (project + global omnisettings.json) | `config.go` |
| `internal/exec/` | Shell command execution with safety classification | `exec.go`, `safelist.go` |
| `internal/history/` | Session persistence as JSON files | `history.go` |
| `internal/security/` | Credential redaction in log output | `redact.go` |
| `cmd/omni/` | Entry point: wires all dependencies, creates Bubble Tea program | `main.go`, `skill_cmd.go` |

### TUI Model — Bubble Tea State Machine

The TUI uses **Bubble Tea v1.3.10** (Elm Architecture). The root `Model` composes:

```
Model
├── InputModel       ← textarea.Textarea (user input box)
├── ViewportModel    ← viewport.Model + glamour.TermRenderer (scrollable chat, markdown)
├── StatusBarModel   ← lipgloss-rendered footer (model name, activity, cost, skill)
├── SlashRouter      ← map[string]Handler (slash command dispatch)
├── VariableWizard   ← textinput-based form for skill variable input
├── *agent.Agent     ← LLM orchestration
└── *skills.Manager  ← skill loading/lookup
```

**4 TUI States:**
- `StateNormal` (0): Idle, accepting input
- `StateStreaming` (1): Agent producing output
- `StateAwaitingApproval` (2): Blocking on y/n for a command
- `StateWizard` (3): Skill variable input form overlay

**Message Flow:**
1. User types → Enter → `SubmitMsg` emitted
2. If starts with `/` → `SlashRouter.Dispatch()` → returns `tea.Msg` + `tea.Cmd`
3. If not slash → `agent.Run(ctx, prompt)` launched in goroutine
4. Agent sends back: `StreamChunkMsg`, `ToolCallMsg`, `AgentDoneMsg`, `ErrorMsg`, `CostUpdateMsg`
5. `run_command` tool sends `ApprovalRequestMsg` via channel → TUI enters `StateAwaitingApproval`

### Config Loading (`internal/config/config.go`)

Two-tier JSON merge:
1. **Built-in defaults**: `ModelPriority: ["gpt-4o"]`, empty `AllowedCommands`, empty `Theme`
2. **Global**: `~/.config/omni/omnisettings.json`
3. **Project**: `./omnisettings.json`

Merge is shallow — non-zero top-level fields in higher-priority config overwrite lower ones. EXCEPTION: `AllowedCommands` concatenates (project commands appended to global commands).

Config struct:
```go
type Config struct {
    ModelPriority   []string
    AllowedCommands []string
    Theme           ThemeConfig
}
```

### Agent Loop (`internal/agent/loop.go`)

```
1. Add user message to session history
2. Build messages from session + Build ToolDefs from registry
3. Loop (max 20 iterations):
   a. ChatStream(ctx, messages, toolDefs, onChunk) → finalMsg, usage
   b. If no ToolCalls → save assistant message, send AgentDoneMsg, DONE
   c. For each ToolCall: execute → append tool result to messages
   d. Repeat from (a)
```

**OmniGo Adapter Quirk** (`internal/agent/omnigo.go`):
- Uses synchronous `Chat()` because streaming mode doesn't return tool calls
- For tool-result iterations: creates a FRESH session with the original prompt + formatted tool results as a single message — workaround for Gemini's strict message format requirements
- Tool schemas defined as Go structs with `desc` tags matching OmniGo's naming convention (lowercase-first: `List_files` → `list_files`)

### Slash Router (`internal/tui/slashrouter.go`)

Simple `map[string]Handler` pattern:
- `Handler func(args string) (tea.Msg, tea.Cmd)`
- Built-in commands: `/exit`, `/help`, `/skills`, `/skill`
- In `main.go`, `/skill` and `/skills` are **overridden** with real skill manager handlers
- New commands: call `router.Register("name", handler)` — registration overwrites previous
- `/help` handler returns a static SystemMsg — NOT dynamic based on registered commands

### MCP-Related Code

**ZERO existing MCP code.** The only MCP artifact is `prd/MCP_PRD.md` (50 lines), which specifies:
- OmniCLI as MCP Host with Stdio (local) and SSE (remote) transports
- `mcp_servers` block in `omnisettings.json`
- `omni mcp add` CLI subcommand for onboarding
- `/attach` slash command to pin MCP resources to conversation
- Trusted vs untrusted server approval model
- Resource browser side panel in TUI

### Dependencies (`go.mod`)

- `github.com/Zheng5005/omnigo` — OmniGo multi-provider AI library (local fork)
- `github.com/charmbracelet/bubbletea v1.3.10` — TUI framework
- `github.com/charmbracelet/bubbles v1.0.0` — TUI components (textarea, viewport, textinput)
- `github.com/charmbracelet/glamour v1.0.0` — Markdown rendering
- `github.com/charmbracelet/lipgloss v1.1.1` — Styling

### Test Setup

- **17 test files**, **62 test functions**, all passing
- Tests in every internal package except no tests in `cmd/omni/`
- Uses standard `testing` package, table-driven tests
- TUI model tests: state transitions, slash routing, approval flow — 16 test cases
- No integration/E2E tests (all tests work with nil agent)
- Test coverage focused on unit logic, not rendering output

## Affected Areas

### Will Need Modification

- **`internal/config/config.go`** — Add `McpServers` config block for server registry
- **`internal/tui/model.go`** — Add new states (`StateMcpApproval`, `StateResourceBrowser`), MCP-related messages, resource panel
- **`internal/tui/messages.go`** — Add MCP-related message types (McpToolApprovalMsg, ResourceAttachMsg, McpServerStatusMsg)
- **`internal/tui/slashrouter.go`** — Add `/attach`, `/mcp` slash commands
- **`internal/agent/omnigo.go`** — Register MCP tools with OmniGo session; tool schema structs for MCP tools
- **`internal/agent/loop.go`** — Handle MCP tool execution (delegates to MCP server, not local exec)
- **`internal/tools/registry.go`** — May need dynamic tool registration (MCP tools are server-provided, not compile-time)
- **`cmd/omni/main.go`** — Wire MCP client, server lifecycle, register MCP slash commands, add `omni mcp add` subcommand

### Will Need Creation (new packages/files)

- **`internal/mcp/`** — New package for MCP protocol implementation:
  - `client.go` — MCP client (JSON-RPC over stdio/SSE)
  - `server.go` — Server lifecycle management (spawn, health check, shutdown)
  - `types.go` — MCP protocol types (tools/list, resources/list, prompts/list)
  - `transport.go` — Stdio and SSE transport implementations
- **`internal/tui/resource_panel.go`** — Collapsible side panel for pinned MCP resources
- **`cmd/omni/mcp_cmd.go`** — `omni mcp add` CLI subcommand

### No Changes Needed

- `internal/history/` — Session persistence is already JSON-based, MCP context can be stored as part of session
- `internal/security/` — Redaction already handles credentials; MCP API keys (if any) would be environment variables
- `internal/exec/` — Only for local shell commands; MCP tool execution goes through MCP protocol, not shell
- `internal/skills/` — Skills are orthogonal to MCP (skills reconfigure the LLM session, MCP provides tools)
- `internal/tui/input.go` — Input model is generic textarea, no MCP-specific changes needed
- `internal/tui/viewport.go` — Viewport renders markdown, no MCP-specific changes needed
- `internal/tui/statusbar.go` — Minor: may want to show MCP server connection status
- `internal/tui/wizard.go` — No MCP changes needed

## Approaches

### 1. **Minimal Integration: MCP as Tool Provider**
Register MCP server tools alongside existing tools. The agent discovers MCP tools at startup via `tools/list`, registers them in the Tool Registry, and the existing agent loop calls them. Approval is handled per the MCP PRD (trusted vs untrusted servers).

- Pros: Minimal code changes, reuses existing tool interface, fast to implement
- Cons: No resource browsing, no `/attach`, limited to tool-only MCP features
- Effort: **Medium**

### 2. **Full Integration: MCP as First-Class Protocol**
New `internal/mcp/` package with full MCP client. Server lifecycle management (spawn/health/shutdown). TUI gets resource browser side panel, `/attach` command, trusted/untrusted approval UX. Tools are dynamically registered and removed. Config gets `mcp_servers` block.

- Pros: Complete MCP experience, matches PRD exactly, resource browsing is a differentiator
- Cons: More code, more test surface, needs MCP server for integration testing
- Effort: **High**

### 3. **Hybrid: Tools First, Resources Later**
Phase 1: MCP tools only — agent can use MCP server tools like databases, APIs. Phase 2: Resource browser and `/attach`.

- Pros: Delivers value faster, each phase independently testable, reduces risk
- Cons: Two SDD changes instead of one
- Effort: **Medium (Phase 1) + Medium (Phase 2)**

## Recommendation

**Approach 3 — Hybrid phased delivery.** 

Phase 1 (MCP Tools):
1. Add `mcp_servers` to config
2. Create `internal/mcp/` with client and server lifecycle
3. At startup, discover MCP tools via `tools/list` and register them in the Tool Registry as `MCPTool` wrappers
4. Add `/mcp` slash command to show connected servers and their tools
5. Implement trusted/untrusted approval in the TUI (new state or reuse `StateAwaitingApproval`)

Phase 2 (MCP Resources):
1. Add `resources/list` and `resources/read` MCP capabilities
2. Build resource browser side panel in TUI
3. Add `/attach` slash command
4. Allow pinning resources to conversation context (prepended to system prompt)

This splits the work into testable, reviewable chunks. Phase 1 gives immediate value (access to MCP tools). Phase 2 adds the rich UX.

## Risks

- **No existing MCP Go library in dependencies** — Will need to implement JSON-RPC transport from scratch or find a lightweight MCP SDK
- **Dynamic tool registration** — The Tool Registry is currently static (registered at startup in main.go). Dynamically adding/removing tools from MCP servers requires making the registry update the LLM session at runtime
- **Server lifecycle management** — Spawning and monitoring child processes adds complexity; crash recovery, restart, graceful shutdown need careful implementation
- **MCP server availability for testing** — Need a real MCP server (like mcp-server-postgres or mcp-server-filesystem) for integration tests; mock MCP server for unit tests
- **OmniGo session recreation** — When MCP tools change (server connects/disconnects), the OmniGo session must be recreated with updated tool list. This already works via `RecreateSession()` but needs triggering on MCP server events

## Ready for Proposal

**Yes.** The codebase is well-understood and all integration points are identified. The next step is a `sdd-propose` phase to define the scope, acceptance criteria, and phased plan.
