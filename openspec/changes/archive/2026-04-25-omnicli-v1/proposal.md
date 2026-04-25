# Proposal: OmniCLI v1 — Full Implementation

## Intent

Build OmniCLI from scratch: a terminal-based AI agent providing a project-aware REPL for developers. It combines local code search, safe command execution, and AI-powered conversation in a Bubble Tea TUI. This is the complete v1 greenfield implementation covering all PRD requirements.

## Scope

### In Scope
- **REPL TUI**: Bubble Tea app with streaming responses, Glamour markdown rendering, Lip Gloss styling
- **Status Bar**: Active model, live session cost (OmniGo pricing), activity indicators (Searching/Thinking/Executing)
- **Project Awareness Tools**: `list_files` (respects `.omniignore`), `grep_search` (keyword + regex), `read_file` (line-range support)
- **Command Execution**: `os/exec` with safe list system, approval gates (Enter for safe, y/n for risky), user-customizable patterns
- **State & Persistence**: JSON history in `.omni/history/` (project) and `~/.config/omni/history/` (global), `--resume` flag for session reload
- **Configuration**: `omnisettings.json` with project → global fallback; fields: ModelPriority, AllowedCommands, Theme
- **Security**: API key / env var redaction in all logs, strictly local data

### Out of Scope
- Cloud sync or remote storage
- Plugin/extension system
- Multi-model simultaneous sessions
- GUI or web interface
- Package distribution (Homebrew, apt) — manual `go install` only for v1

## Approach

Layered architecture with clear separation:

1. **`cmd/omni/`** — CLI entrypoint (cobra or minimal flag parsing), `--resume` flag
2. **`internal/tui/`** — Bubble Tea Model (Init/Update/View), sub-models for input area, chat viewport, status bar
3. **`internal/agent/`** — Agent loop: receives user input → builds messages → calls OmniGo → processes tool calls → streams response
4. **`internal/tools/`** — Tool registry with implementations: `list_files`, `grep_search`, `read_file`, `run_command`
5. **`internal/exec/`** — Command execution engine: safe list matching (regex), approval gate logic, `os/exec` wrapper
6. **`internal/config/`** — Config loader: project `.omnisettings.json` → global `~/.config/omni/omnisettings.json` merge
7. **`internal/history/`** — JSON session read/write, session resumption logic
8. **`internal/security/`** — Redaction middleware for API keys and env vars

Data flow: User Input → TUI (Bubble Tea Msg) → Agent Loop → OmniGo API → Stream chunks back → TUI Update → Glamour render → View

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/omni/main.go` | New | CLI entrypoint, flag parsing, app bootstrap |
| `internal/tui/` | New | Bubble Tea model, input, viewport, status bar components |
| `internal/agent/` | New | Agent loop, OmniGo integration, tool dispatch |
| `internal/tools/` | New | list_files, grep_search, read_file implementations |
| `internal/exec/` | New | Safe list engine, approval gates, os/exec wrapper |
| `internal/config/` | New | Config loading with project→global fallback |
| `internal/history/` | New | JSON session persistence, --resume support |
| `internal/security/` | New | API key redaction middleware |
| `go.mod` | New | Module definition, dependencies |
| `.omniignore` | New | Default ignore patterns (reference/docs) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| OmniGo API surface unknown — may need adapters | Med | Explore OmniGo early; define internal interfaces to decouple |
| Bubble Tea streaming complexity (channel → Msg pipeline) | Med | Use proven tea.Program Sub pattern; prototype streaming first |
| Safe list regex injection | Low | Compile patterns at config load time; reject invalid regex |
| Large monorepo performance with filepath.WalkDir | Low | Respect .omniignore aggressively; set max depth/file limits |
| Session JSON corruption on crash | Low | Write to temp file + atomic rename |

## Rollback Plan

Greenfield project — rollback is `git revert` to empty repo. No existing functionality at risk. Each feature area is a separate package, so partial rollback is possible by reverting specific packages.

## Dependencies

- **OmniGo**: Custom Go AI library (must be available as Go module)
- **Bubble Tea** / **Lip Gloss** / **Glamour**: Charm ecosystem (well-maintained, stable)
- **Go 1.22+**: For range-over-func and other modern features

## Success Criteria

- [x] `go build ./cmd/omni/` produces working binary
- [x] REPL accepts input, sends to OmniGo, streams response with markdown rendering
- [x] Status bar shows active model, session cost, activity state
- [x] `list_files` respects `.omniignore` and returns project tree
- [x] `grep_search` finds matches by keyword and regex pattern
- [x] `read_file` loads specific line ranges into context
- [x] Safe commands execute with Enter confirmation
- [x] Risky commands require explicit y/n approval
- [x] AllowedCommands in omnisettings.json adds to safe list
- [x] Config merges project-level over global-level settings
- [x] Sessions persist as JSON in correct hierarchy
- [x] `--resume` flag restores last session state
- [x] API keys are redacted from all log output
- [x] No project data leaves the local machine
