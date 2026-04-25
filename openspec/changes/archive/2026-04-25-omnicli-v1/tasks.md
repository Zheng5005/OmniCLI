# Tasks: OmniCLI v1 — Full Implementation

## Phase 1: Project Scaffolding [S]

- [x] 1.1 Create `go.mod` with module `github.com/omnicli/omnicli`, Go 1.23, add dependencies: bubbletea, lipgloss, glamour, omnigo | Files: `go.mod`, `go.sum` | Size: S
- [x] 1.2 Create directory structure: `cmd/omni/`, `internal/tui/`, `internal/agent/`, `internal/tools/`, `internal/exec/`, `internal/config/`, `internal/history/`, `internal/security/` | Size: S
- [x] 1.3 Create `cmd/omni/main.go` — minimal entrypoint: parse `--resume` flag with `flag` stdlib, stub `tea.Program` init | Files: `cmd/omni/main.go` | Size: S
- [x] 1.4 Create `.omniignore` with defaults: `.git/`, `node_modules/`, `vendor/`, `*.exe`, `*.bin`, `*.o` | Files: `.omniignore` | Size: S

## Phase 2: Core Infrastructure [M]

- [x] 2.1 Create `internal/config/config.go` — `Config` struct (`ModelPriority []string`, `AllowedCommands []string`, `Theme ThemeConfig`), `Load()` with project→global fallback, JSON unmarshal, shallow merge logic. Defaults for missing config. | Files: `internal/config/config.go` | Deps: 1.2 | Size: M
- [x] 2.2 Create `internal/security/redact.go` — `RedactingWriter` wrapping `io.Writer`, pattern registry for `sk-`, `AIza`, `ghp_`, env var `KEY=value` patterns. `Redact(string) string` standalone func. Pre-compile regexes. | Files: `internal/security/redact.go` | Deps: 1.2 | Size: M
- [x] 2.3 Create `internal/history/history.go` — `Session` struct holding messages slice, `Save()` with atomic write (temp file + rename), `Load(path)`, `Latest(dir)` for `--resume`. Directory creation with `os.MkdirAll`. | Files: `internal/history/history.go` | Deps: 1.2 | Size: M

## Phase 3: Tool System [M]

- [x] 3.1 Create `internal/tools/registry.go` — `Tool` interface, `Registry` struct with `map[string]Tool`, `Register()`, `Get()`, `List()`. | Files: `internal/tools/registry.go` | Deps: 1.2 | Size: S
- [x] 3.2 Create `internal/tools/list_files.go` — `ListFilesTool`. Uses `filepath.WalkDir`, loads `.omniignore`, supports `max_depth` param, outputs tree format. | Files: `internal/tools/list_files.go`, `internal/tools/ignore.go` | Deps: 3.1 | Size: M
- [x] 3.3 Create `internal/tools/grep_search.go` — `GrepSearchTool`. Walks files, compiles regex from `pattern` arg, returns matches. Skips binary files. | Files: `internal/tools/grep_search.go` | Deps: 3.1 | Size: M
- [x] 3.4 Create `internal/tools/read_file.go` — `ReadFileTool`. Reads file with optional `start_line`/`end_line` args. Returns error message (not panic) for missing files. | Files: `internal/tools/read_file.go` | Deps: 3.1 | Size: S

## Phase 4: Command Execution [M]

- [x] 4.1 Create `internal/exec/safelist.go` — Default safe patterns (10), `CompilePatterns`, `Match`. | Files: `internal/exec/safelist.go` | Deps: 2.1 | Size: M
- [x] 4.2 Create `internal/exec/exec.go` — `Classification` (Safe/Risky), `Classify`, `Run` using `os/exec` with timeout support, captures stdout/stderr/exitCode. | Files: `internal/exec/exec.go` | Deps: 4.1 | Size: M
- [x] 4.3 Create `internal/tools/run_command.go` — `RunCommandTool` with approvalFn callback for TUI. | Files: `internal/tools/run_command.go` | Deps: 3.1, 4.2 | Size: M

## Phase 5: Agent Loop [L]

- [x] 5.1 Create `internal/agent/agent.go` — `Agent` struct, `LLMClient` interface, `ChatMessage`/`ToolCall`/`StreamChunk`/`Usage`/`ToolDefinition` types, `SendFunc` callback. | Files: `internal/agent/agent.go` | Deps: 3.1, 2.3 | Size: M
- [x] 5.2 Create `internal/agent/loop.go` — `Run(ctx, prompt)` loop: ChatStream → tool dispatch → repeat. 20-iteration guard. TUI message types. | Files: `internal/agent/loop.go` | Deps: 5.1, 4.3 | Size: L

## Phase 6: TUI [L]

- [x] 6.1 Create `internal/tui/messages.go` — `SubmitMsg`, `ApprovalRequestMsg` (with `ResponseCh chan bool`). | Files: `internal/tui/messages.go` | Deps: 1.2 | Size: S
- [x] 6.2 Create `internal/tui/input.go` — `InputModel` wrapping textarea, Enter submits, Shift+Enter newline, enable/disable. | Files: `internal/tui/input.go` | Deps: 6.1 | Size: M
- [x] 6.3 Create `internal/tui/viewport.go` — `ViewportModel`: raw text during streaming, glamour render on FinalizeResponse, AppendSystem for system messages. | Files: `internal/tui/viewport.go` | Deps: 6.1 | Size: M
- [x] 6.4 Create `internal/tui/statusbar.go` — `StatusBarModel`: Activity enum (Ready/Thinking/Searching/Executing), color-coded, full-width lipgloss bar with model name + cost. | Files: `internal/tui/statusbar.go` | Deps: 6.1 | Size: M
- [x] 6.5 Create `internal/tui/model.go` — Root `Model` composing all sub-models. State machine (Normal → Streaming → AwaitingApproval). Handles all agent + TUI messages. | Files: `internal/tui/model.go` | Deps: 6.2, 6.3, 6.4, 5.2 | Size: L

## Phase 7: Integration & Polish [L]

- [x] 7.1 Wire `cmd/omni/main.go` — Full bootstrap: config → security → history → safelist → OmniGo → tools → agent → TUI → tea.Program. Added `internal/agent/omnigo.go` adapter. | Files: `cmd/omni/main.go`, `internal/agent/omnigo.go`, `internal/agent/agent.go` (added SetSend) | Deps: 6.5, 2.1, 2.2, 2.3 | Size: L
- [x] 7.2 Implement approval flow end-to-end — TUI receives `ApprovalRequestMsg`, enters AwaitingApproval, renders prompt. UX fix: risky commands show `[y]` not `[Enter/y]`. | Files: `internal/tui/model.go`, `internal/agent/loop.go`, `internal/tools/run_command.go` | Deps: 7.1 | Size: M
- [x] 7.3 Implement `--resume` full flow — `history.Latest()`, load messages into agent context, render previous messages in viewport via `LoadHistory()`. Graceful no-session-found handling. | Files: `cmd/omni/main.go`, `internal/history/history.go`, `internal/tui/model.go` | Deps: 7.1 | Size: M
- [x] 7.4 Error handling pass — Improved exec timeout/command-not-found error messages. All tools return user-friendly errors. | Files: `internal/exec/exec.go` | Deps: 7.1 | Size: M

## Phase 8: Testing [L]

- [x] 8.1 Unit tests for config — Table-driven: TestLoadFile (3), TestMerge (4), TestDefaults. | Files: `internal/config/config_test.go` | Deps: 2.1 | Size: S
- [x] 8.2 Unit tests for security — Table-driven: TestRedact (8 patterns), TestRedactingWriter (3). | Files: `internal/security/redact_test.go` | Deps: 2.2 | Size: S
- [x] 8.3 Unit tests for history — Save/load roundtrip, atomic write, Latest() (3 cases), dir helpers. | Files: `internal/history/history_test.go` | Deps: 2.3 | Size: S
- [x] 8.4 Unit tests for tools — list_files (5), grep_search (6), read_file (5) with t.TempDir(). | Files: `internal/tools/list_files_test.go`, `grep_search_test.go`, `read_file_test.go` | Deps: 3.2, 3.3, 3.4 | Size: M
- [x] 8.5 Unit tests for exec — TestClassify (2), TestRun (4), TestMatch (8 commands), TestCompilePatterns. | Files: `internal/exec/exec_test.go`, `internal/exec/safelist_test.go` | Deps: 4.1, 4.2 | Size: M
- [x] 8.6 Integration test for agent loop — mockLLMClient + mockTool, 4 cases (text response, tool loop, unknown tool, LLM error). | Files: `internal/agent/agent_test.go` | Deps: 5.2 | Size: L
- [x] 8.7 TUI tests for Model state machine — 11 cases for Update() transitions, approval flow, Ctrl+C, LoadHistory. | Files: `internal/tui/model_test.go` | Deps: 6.5 | Size: L
