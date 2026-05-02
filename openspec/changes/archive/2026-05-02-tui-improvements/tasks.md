# Tasks: TUI Improvements — Command Palette & Loading Indicators

## Phase 1: Foundation — Messages and SlashRouter Extensions

- [x] 1.1 Add `CommandPaletteDismissMsg struct{}` and `CommandPaletteExecuteMsg struct{ Command string }` to `internal/tui/messages.go`
- [x] 1.2 Add `Descriptions() []CommandDesc` method to `SlashRouter` in `internal/tui/slashrouter.go` returning built-in command descriptions
- [x] 1.3 Update `/help` command handler in `internal/tui/slashrouter.go` to include `/help`, `/exit`, `/mcp`, `/attach`, `/detach`, `/skills`, `/skill`, `/palette` (Ctrl+P)

## Phase 2: Command Palette Implementation

- [x] 2.1 Create `internal/tui/command_palette.go` with `paletteItem` struct and `CommandPalette` struct (input, items, filtered, cursor, width, height)
- [x] 2.2 Implement `NewCommandPalette(commands []CommandDesc) *CommandPalette` constructor populating items from `router.Descriptions()`
- [x] 2.3 Implement `CommandPalette.Init()` returning `textinput.Blink` cmd
- [x] 2.4 Implement `CommandPalette.Update()` handling: Esc→dismiss msg, typing→filter, ↑/↓→cursor, Enter→execute msg
- [x] 2.5 Implement `CommandPalette.View()` rendering centered overlay with rounded border via `lipgloss.Place`
- [x] 2.6 Implement `CommandPalette.SetSize(width, height int)` for resize handling
- [x] 2.7 Add `commandPalette CommandPalette` field and `showCommandPalette bool` to `Model` in `internal/tui/model.go`
- [x] 2.8 Add `StateCommandPalette` state constant to `State` enum in `internal/tui/model.go`
- [x] 2.9 Add `ctrl+p` key binding case in `Model.Update()` to open palette (guard: only in `StateNormal`)
- [x] 2.10 Add `CommandPaletteDismissMsg` and `CommandPaletteExecuteMsg` handlers in `Model.Update()` to close palette and dispatch commands

## Phase 3: Spinner Integration

- [x] 3.1 Add `spinner spinner.Model` field to `Model` in `internal/tui/model.go`
- [x] 3.2 Import `github.com/charmbracelet/bubbles/spinner` in `internal/tui/model.go`
- [x] 3.3 Initialize spinner in `NewModel()` with `spinner.New()` and set spinner type (e.g., `spinner.Dot`)
- [x] 3.4 Add spinner start logic in `Model.Update()` on entry to `StateStreaming`/`StateAwaitingApproval`/`StateMcpApproval`/`StateWizard`/`StateResourceBrowser`
- [x] 3.5 Add spinner stop logic in `Model.Update()` on return to `StateNormal`
- [x] 3.6 Handle `spinner.TickMsg` in `Model.Update()` to advance spinner animation
- [x] 3.7 Modify `StatusBarModel.View()` signature to accept `spinnerView string` parameter in `internal/tui/statusbar.go`
- [x] 3.8 Render `spinnerView` prefix before activity text in `StatusBarModel.View()`
- [x] 3.9 Pass `m.spinner.View()` to `m.statusBar.View()` in `Model.View()`
- [x] 3.10 Add `.Bold(true)` to activity text style in `StatusBarModel.View()` for non-Ready states

## Phase 4: Testing

- [x] 4.1 Write unit test for `SlashRouter.Descriptions()` asserting all built-in commands have non-empty descriptions
- [x] 4.2 Write unit tests for `CommandPalette` filter logic in `internal/tui/command_palette_test.go`
- [x] 4.3 Write unit tests for `CommandPalette` key navigation (Esc, ↑/↓, Enter) in `internal/tui/command_palette_test.go`
- [x] 4.4 Write unit test for spinner start/stop in `Model.Update()` asserting spinner.View non-empty during streaming
- [x] 4.5 Write unit test for `StatusBarModel.View()` with spinner prefix in `internal/tui/statusbar_test.go`
- [x] 4.6 Write integration test for full palette flow (Ctrl+P→open, Esc→close, Enter→dispatch) in `internal/tui/model_test.go`

## Phase 5: Cleanup and Verification

- [x] 5.1 Run `go mod tidy` to ensure `bubbles/spinner` dependency is recorded
- [x] 5.2 Verify no regressions in existing overlay dialogs (approval, wizard, resource browser)
- [x] 5.3 Verify Ctrl+P is documented in `/help` output
- [x] 5.4 Manual test: confirm spinner animates during AI processing states
- [x] 5.5 Manual test: confirm palette filters commands in real-time with type-to-search
