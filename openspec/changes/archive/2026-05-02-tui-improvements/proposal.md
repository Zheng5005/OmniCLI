# Proposal: TUI Improvements — Command Palette & Loading Indicators

## Intent

Two UX gaps in the OmniCLI TUI reduce discoverability and feedback clarity:
1. Users must memorize or type `/help` to discover slash commands — no quick reference overlay exists.
2. The status bar shows static text ("Thinking", "Searching", "Executing") with no animated spinner, making it easy to miss that the AI is actively working.

## Scope

### In Scope
- **Command Palette** (Ctrl+P): centered overlay listing all registered slash commands with descriptions, type-to-filter, Up/Down navigation, Enter to execute, Esc to dismiss
- **Spinner Animation**: add `bubbletea/spinner.Model` to the status bar alongside existing activity text labels; animate during Streaming, AwaitingApproval, Wizard, McpApproval, ResourceBrowser states

### Out of Scope
- New slash commands or changes to command routing logic
- Changes to existing overlay dialogs (help, skill list, approval views)
- Visual redesign of the status bar layout beyond adding the spinner
- Mouse support for the command palette

## Capabilities

### New Capabilities
- `command-palette`: Ctrl+P overlay for discovering and executing slash commands with filtering and keyboard navigation

### Modified Capabilities
- `omnicli-v1`: Status bar Activity Indicators requirement — spinner animation added to "Thinking", "Searching", "Executing" states

## Approach

**Command Palette**: Reuse the existing overlay pattern (centered box via `lipgloss.Place`, rounded border, keyboard navigation) used by the 3 existing dialogs. New `command_palette.go` model with its own `Init/Update/View`. Triggered by a new `tea.KeyMsg` case for `ctrl+p` in the main `Update` function. Reads registered commands from the existing `SlashRouter` to populate the list.

**Spinner**: Import `github.com/charmbracelet/bubbles/spinner`. Embed `spinner.Model` in the main model. Start/stop spinner in response to state transitions (start on streaming begin, stop on streaming end). Render spinner before the activity text in `statusbar.go`.

Both features are independent and share no code paths — safe to ship together.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tui/model.go` | Modified | Add `ctrl+p` key binding, command palette state field, spinner model |
| `internal/tui/command_palette.go` | New | Command palette overlay model (Init/Update/View) |
| `internal/tui/statusbar.go` | Modified | Render spinner alongside activity text |
| `internal/tui/slashrouter.go` | Modified | Expose registered commands list for palette population |
| `go.mod` | Modified | Add `bubbles/spinner` dependency (v1.0.0 already available) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Key binding conflict with terminal/Ctrl+P passthrough | Low | Ctrl+P is not used by common terminals in fullscreen mode; document in /help |
| Spinner flicker on fast responses | Medium | Use spinner only for states lasting >100ms; skip for instant transitions |
| Command palette overlaps with other overlays | Low | Guard: only open palette when no other overlay is active |

## Rollback Plan

Revert the commit. Both features are additive — no existing behavior is removed. The status bar falls back to static text, and Ctrl+P becomes a no-op key press.

## Dependencies

- `github.com/charmbracelet/bubbles/spinner` (v1.0.0, already in go.mod)
- Existing `SlashRouter` must expose a `Commands() []Command` accessor (minor addition)

## Success Criteria

- [ ] Ctrl+P opens a centered overlay listing all 7+ slash commands with descriptions
- [ ] Typing in the palette filters the command list in real-time
- [ ] Up/Down arrows navigate the filtered list; Enter executes the selected command
- [ ] Esc or Enter dismisses the palette and returns to normal state
- [ ] Status bar shows an animated spinner during AI processing states
- [ ] Spinner stops when streaming completes or user returns to idle
- [ ] No regression in existing overlay dialogs or TUI states
