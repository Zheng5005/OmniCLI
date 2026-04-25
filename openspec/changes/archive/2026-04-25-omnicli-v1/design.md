# Design: OmniCLI v1 — Full Implementation

## Technical Approach

Layered architecture in Go with strict package boundaries. Bubble Tea drives the UI event loop; the agent loop runs asynchronously via `tea.Cmd`, communicating results back as `tea.Msg`. OmniGo handles LLM interaction; tool calls are dispatched through a registry pattern. Every layer depends only on the layer below it — no circular imports.

```
cmd/omni/main.go
    ↓
internal/tui/          ← Bubble Tea Model (orchestrates UI)
    ↓
internal/agent/        ← Agent loop (LLM ↔ tools cycle)
    ↓
internal/tools/        ← Tool registry + implementations
internal/exec/         ← Command execution + approval
internal/config/       ← Configuration loading
internal/history/      ← Session persistence
internal/security/     ← Redaction middleware
```

## Architecture Decisions

| # | Decision | Choice | Alternatives | Rationale |
|---|----------|--------|-------------|-----------|
| 1 | CLI framework | `flag` stdlib | Cobra | Single binary, one command, `--resume` flag only. Cobra is overkill. |
| 2 | Agent ↔ TUI communication | `tea.Cmd` + custom `tea.Msg` types | Channels directly | Bubble Tea's Elm architecture expects Cmds. Fighting it with raw channels creates race conditions. Use `tea.Program.Send()` for streaming chunks. |
| 3 | Streaming delivery | Agent goroutine pushes `StreamChunkMsg` via `p.Send()` | Blocking in Update | Non-blocking streaming. The agent loop runs in a goroutine; each chunk from OmniGo's stream callback fires `p.Send(StreamChunkMsg{...})`. |
| 4 | Tool dispatch | Interface + registry map | Switch statement | Registry map (`map[string]Tool`) is extensible without modifying dispatch code. Each tool is a self-contained implementation. |
| 5 | Config merge strategy | Shallow merge (project overrides global per top-level key) | Deep merge | Simple and predictable. `AllowedCommands` arrays concatenate (project + global). Other fields: project wins if set. |
| 6 | History format | Raw JSON matching OmniGo message types | SQLite, custom binary | OmniGo already uses JSON message arrays. Zero serialization overhead. Direct `json.Marshal`/`Unmarshal`. |
| 7 | Safe list matching | Pre-compiled `regexp.Regexp` slice | String prefix matching | PRD specifies regex patterns in config. Compile once at load time; reject invalid patterns early. |
| 8 | Security redaction | `io.Writer` wrapper | Post-processing | Wrapping the log writer catches ALL output including third-party libs. Pattern: `security.NewRedactingWriter(os.Stderr, patterns)`. |
| 9 | Sub-models | Separate structs for InputModel, ViewportModel, StatusBarModel | Single monolithic model | Separation of concerns. Each sub-model handles its own Update/View. Parent TUI model composes them. |
| 10 | Approval flow | TUI state machine (Normal → AwaitingApproval → Executing) | Separate approval prompt | Keeps everything in the Bubble Tea event loop. No blocking prompts. |

## Data Flow

### User Prompt Flow
```
User types → InputModel captures → Enter pressed
    ↓
TUI.Update receives SubmitMsg
    ↓
tea.Cmd: launches agent.Run(prompt, history) in goroutine
    ↓
Agent builds messages → calls OmniGo.ChatStream()
    ↓
OmniGo streams chunks → agent calls p.Send(StreamChunkMsg{text})
    ↓
TUI.Update receives StreamChunkMsg → appends to ViewportModel
    ↓
OmniGo returns tool_call → agent calls p.Send(ToolCallMsg{...})
    ↓
Agent dispatches tool → gets result → appends to messages → re-calls OmniGo
    ↓
Loop until no more tool calls → p.Send(AgentDoneMsg{})
    ↓
TUI.Update → re-enables input, updates status bar
```

### Command Approval Flow
```
Agent receives tool_call: run_command("git push")
    ↓
exec.Classify(cmd, safeList) → "risky"
    ↓
p.Send(ApprovalRequestMsg{cmd, risk: "risky"})
    ↓
TUI enters AwaitingApproval state → renders "[y/n] Run: git push?"
    ↓
User presses 'y' → TUI sends ApprovalResponseMsg{approved: true}
    ↓
tea.Cmd: exec.Run(cmd) → captures stdout/stderr
    ↓
p.Send(CommandResultMsg{output, exitCode})
    ↓
Agent receives result → continues LLM loop
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/omni/main.go` | Create | Entrypoint: parse `--resume` flag, load config, init history, create `tea.Program`, run |
| `internal/tui/model.go` | Create | Root Bubble Tea model: composes InputModel, ViewportModel, StatusBarModel. Handles all Msg routing |
| `internal/tui/input.go` | Create | InputModel: multi-line text input with Enter to submit, Shift+Enter for newline |
| `internal/tui/viewport.go` | Create | ViewportModel: scrollable chat area, Glamour markdown rendering of accumulated content |
| `internal/tui/statusbar.go` | Create | StatusBarModel: model name, session cost, activity indicator. Lip Gloss styled |
| `internal/tui/messages.go` | Create | All custom `tea.Msg` types: StreamChunkMsg, ToolCallMsg, AgentDoneMsg, ApprovalRequestMsg, etc. |
| `internal/agent/agent.go` | Create | Agent struct: holds OmniGo client, tool registry, history ref. `Run()` method drives the LLM↔tool loop |
| `internal/agent/loop.go` | Create | Core loop logic: send messages → process response → dispatch tools → repeat until done |
| `internal/agent/omnigo.go` | Create (added during apply) | OmniGo adapter implementing LLMClient interface — bridges API gaps |
| `internal/tools/registry.go` | Create | Tool interface, registry map, registration helpers |
| `internal/tools/ignore.go` | Create (added during apply) | Shared `.omniignore` pattern loader for tools |
| `internal/tools/list_files.go` | Create | `list_files` tool: `filepath.WalkDir` respecting `.omniignore` patterns |
| `internal/tools/grep_search.go` | Create | `grep_search` tool: `filepath.WalkDir` + `regexp.MatchString` on file contents |
| `internal/tools/read_file.go` | Create | `read_file` tool: read file with optional line range (start, end) |
| `internal/tools/run_command.go` | Create | `run_command` tool: delegates to exec package, handles approval flow via channel/callback |
| `internal/exec/exec.go` | Create | `Classify()` (safe vs risky), `Run()` (os/exec wrapper with timeout), result capture |
| `internal/exec/safelist.go` | Create | Loads and compiles regex patterns from config. `Match(cmd) bool` |
| `internal/config/config.go` | Create | `Config` struct, `Load()` with project → global fallback, JSON unmarshal, merge logic |
| `internal/history/history.go` | Create | `Session` struct, `Save()`, `Load()`, `Latest()` for `--resume`. Atomic file writes |
| `internal/security/redact.go` | Create | `RedactingWriter`, pattern registry (API key formats, env var patterns), `Redact(string) string` |
| `go.mod` | Create | Module `github.com/omnicli/omnicli`, Go 1.23, dependencies |
| `.omniignore` | Create | Default ignore: `.git/`, `node_modules/`, `vendor/`, binary extensions |

## Interfaces / Contracts

```go
// internal/tools/registry.go
type Tool interface {
    Name() string
    Description() string
    Parameters() json.RawMessage // JSON Schema for OmniGo function calling
    Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type Registry struct {
    tools map[string]Tool
}

func (r *Registry) Register(t Tool)
func (r *Registry) Get(name string) (Tool, bool)
func (r *Registry) List() []Tool
```

```go
// internal/agent/agent.go
type LLMClient interface {
    ChatStream(ctx context.Context, messages []ChatMessage, toolDefs []ToolDefinition, onChunk func(StreamChunk)) (*ChatMessage, *Usage, error)
    ModelName() string
}

type Agent struct {
    client   LLMClient
    registry *tools.Registry
    session  *history.Session
    send     SendFunc // p.Send reference for streaming back to TUI
}

func (a *Agent) Run(ctx context.Context, prompt string) // Runs in goroutine
```

```go
// internal/exec/exec.go
type Classification int
const (
    Safe Classification = iota
    Risky
)

func Classify(cmd string, patterns []*regexp.Regexp) Classification
func Run(ctx context.Context, cmd string, timeout time.Duration) (Result, error)
```

```go
// internal/config/config.go
type Config struct {
    ModelPriority   []string          `json:"modelPriority"`
    AllowedCommands []string          `json:"allowedCommands"`
    Theme           ThemeConfig       `json:"theme"`
}

func Load() (*Config, error) // project → global fallback + merge
```

```go
// internal/security/redact.go
type RedactingWriter struct {
    inner    io.Writer
    patterns []*regexp.Regexp
}

func NewRedactingWriter(w io.Writer, extraPatterns ...string) *RedactingWriter
func Redact(s string) string // Standalone for non-writer use
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Tool implementations (list_files, grep_search, read_file) | Temp directories with known file structures; table-driven tests |
| Unit | Config merge logic | Multiple JSON fixtures; verify project overrides global |
| Unit | Safe list classification | Table-driven: command string → expected Classification |
| Unit | Redaction patterns | Known API key formats → verify masked output |
| Unit | History save/load round-trip | Write session JSON → read back → compare |
| Integration | Agent loop with mock LLM client | Mock client returning canned tool_calls; verify tool dispatch and loop termination |
| Integration | Exec pipeline | Run real commands (`echo hello`); verify capture. Test timeout behavior |
| Unit | TUI Model state machine | Direct Update() calls with synthetic Msgs |

## Migration / Rollout

No migration required. Greenfield project — first commit creates everything.

## Resolved Open Questions

- **OmniGo API surface**: Resolved during implementation. Created `internal/agent/omnigo.go` adapter that uses OmniGo's synchronous `Chat()` always (since `ChatStream()` does NOT return tool calls). Tool results injected as formatted user text since OmniGo's session has no `AddToolResult` API.
- **OmniGo message types**: Resolved — direct mapping via the adapter. OmniGo session manages its own history; the adapter syncs only the current turn.
