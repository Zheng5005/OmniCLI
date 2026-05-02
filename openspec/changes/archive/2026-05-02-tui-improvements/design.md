# Design: TUI Improvements — Command Palette & Loading Indicators

## Technical Approach

Two independent features sharing no code paths, shipped as one change:

1. **Command Palette**: A new `CommandPalette` sub-model in `internal/tui/command_palette.go` following the existing overlay pattern (centered box via `lipgloss.Place`, rounded border, keyboard navigation). Triggered by `Ctrl+P` in `StateNormal`. Populates from `SlashRouter.List()` + a new `SlashRouter.Descriptions()` map for descriptions. Uses `bubbles/textinput` for filtering and manual cursor list for selection.

2. **Spinner**: Embed `spinner.Model` from `bubbles/spinner` into the main `Model`. Start spinner on entry to streaming/approval states, stop on return to `StateNormal`. Rendered as a prefix to the activity text in `StatusBarModel.View()`.

Both features map directly to the proposal's approach — no deviation.

## Architecture Decisions

### Decision: Command Palette Sub-Model

**Choice**: Standalone `CommandPalette` struct with `Init/Update/View` in a new file.
**Alternatives**: Integrate palette logic inline into `model.go`.
**Rationale**: Mirrors existing pattern (`VariableWizard`, `ApprovalDialog`, `ResourceBrowser`). Each overlay is a self-contained sub-model. Keeps `model.go` from growing further.

### Decision: Palette Filtering via textinput

**Choice**: Use `bubbles/textinput.Model` for the filter input, manual filtered slice for the list.
**Alternatives**: Use `bubbles/list.Model` for full list with built-in filtering.
**Rationale**: The palette shows ≤10 items. `bubbles/list` pulls in delegation and pagination complexity we don't need. A simple filtered slice + cursor index is lighter and follows how `ResourceBrowser` works (cursor + manual rendering).

### Decision: Command Descriptions from SlashRouter

**Choice**: Add a `Descriptions() map[string]string` method to `SlashRouter` that harcodes built-in descriptions alongside existing handler registrations.
**Alternatives**: Separate config file; descriptions embedded in handler signatures.
**Rationale**: Descriptions are static strings that pair 1:1 with registered commands. Embedding them in `SlashRouter` keeps everything in one place, no new file or config format needed.

### Decision: Spinner in StatusBar

**Choice**: Add a `spinner.Model` field to `Model` (not `StatusBarModel`). Pass spinner view output to `StatusBarModel.View()` via a new `spinnerView string` parameter.
**Alternatives**: Embed spinner directly in `StatusBarModel`.
**Rationale**: Spinner needs `tea.Cmd` for its tick — `StatusBarModel` has no `Update` method and isn't a `tea.Model`. The main `Model.Update()` already manages state transitions and can start/stop the spinner. Keeping `StatusBarModel` a pure renderer avoids making it a mini state machine.

### Decision: Spinner Only in Long-Running States

**Choice**: Start spinner on `StateStreaming`/`StateAwaitingApproval`/`StateMcpApproval`/`StateWizard`/`StateResourceBrowser` entries; stop on `StateNormal`.
**Alternatives**: Start spinner on every state change; add a minimum duration threshold.
**Rationale**: These are the states where the user is waiting. No threshold needed — the spinner's 100ms tick rate provides natural visual delay; if a state resolves in <100ms, the spinner only renders one frame, which is harmless.

## Data Flow

### Command Palette

```
User presses Ctrl+P (StateNormal)
    │
    ├─► Model.Update() sets state=StateCommandPalette, clears filter
    │
    ▼
CommandPalette.Update() handles all keys:
    ├── Esc → CommandPaletteDismissMsg → Model restores StateNormal
    ├── typing → filter input updates, filteredItems recomputed
    ├── ↑/↓ → moves cursor
    └── Enter → SlashRouter.Dispatch("/" + selected) → returns Msg
```

### Spinner

```
State transition to Streaming/Approval/etc:
    │
    ├─► Model.Update() calls spinner.Tick → returns tea.Cmd
    │
    ▼
Each Update cycle:
    │
    ├─► spinner.Update(tick) → Model stores view string
    │
    ├─► StatusBarModel.View(spinnerView) renders "⠋ Thinking"
    │
    State transition back to Normal:
    │
    └─► spinner stops, spinnerView = ""
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/command_palette.go` | Create | CommandPalette sub-model: Init/Update/View, filter + filtered list, Enter dispatches command |
| `internal/tui/model.go` | Modify | Add `commandPalette` field, `showCommandPalette` bool, `ctrl+p` key binding, `StateCommandPalette` state, spinner.Model field, spinner start/stop on state transitions, pass spinnerView to StatusBar |
| `internal/tui/statusbar.go` | Modify | `View()` accepts `spinnerView string` param rendered before activity text; activity labels get `.Bold(true)` on non-Ready states |
| `internal/tui/slashrouter.go` | Modify | Add `Descriptions() map[string]string` method returning built-in command descriptions |
| `internal/tui/messages.go` | Modify | Add `CommandPaletteDismissMsg` and `CommandPaletteExecuteMsg` message types |

## Interfaces / Contracts

```go
// CommandPalette — new sub-model in command_palette.go
type CommandPalette struct {
    input     textinput.Model
    items     []paletteItem  // all commands (name + description)
    filtered  []paletteItem  // filtered subset
    cursor   int
    width    int
    height   int
}

type paletteItem struct {
    name        string
    description string
}

func NewCommandPalette(router *SlashRouter) CommandPalette
func (m CommandPalette) Init() tea.Cmd
func (m CommandPalette) Update(msg tea.Msg) (CommandPalette, tea.Cmd)
func (m CommandPalette) View() string
func (m *CommandPalette) SetSize(width, height int)
```

```go
// SlashRouter — new method
func (r *SlashRouter) Descriptions() map[string]string
```

```go
// StatusBarModel — modified signature
func (m StatusBarModel) View(spinnerView string) string
```

```go
// New message types in messages.go
type CommandPaletteDismissMsg struct{}
type CommandPaletteExecuteMsg struct{ Command string }
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `CommandPalette` filter logic | Create palette with known items, type filter text, assert `filtered` subset matches |
| Unit | `SlashRouter.Descriptions()` | Assert all built-in commands have descriptions |
| Unit | `CommandPalette` key navigation | Assert ↑/↓ moves cursor, Esc emits `CommandPaletteDismissMsg`, Enter emits correct command |
| Unit | Spinner start/stop in `Model.Update` | Assert spinner.View is non-empty during streaming, empty in StateNormal |
| Unit | `StatusBarModel.View()` with spinner | Assert rendered string contains spinner prefix when spinnerView is non-empty |
| Integration | Full palette flow in `Model.Update` | Ctrl+P opens palette, Esc closes, Enter dispatches command via router |

## Migration / Rollback

No migration required. Both features are additive — no existing data structures or configs change. Rollback is a single revert commit. Ctrl+P becomes a no-op, status bar falls back to static activity text (spinnerView="" is the default).

## Open Questions

- None — both features are well-scoped with clear existing patterns to follow.