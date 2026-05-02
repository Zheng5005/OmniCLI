package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommandPalette_FilterPrefixMatch(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "exit", Description: "Exit the app"},
		{Name: "help", Description: "Show help"},
		{Name: "mcp", Description: "List MCP servers"},
	}
	p := NewCommandPalette(cmds)

	// Simulate typing "ex"
	p.input.SetValue("ex")
	p.applyFilter()

	if len(p.filtered) != 1 {
		t.Fatalf("expected 1 filtered item, got %d", len(p.filtered))
	}
	if p.filtered[0].name != "exit" {
		t.Errorf("expected 'exit', got %q", p.filtered[0].name)
	}
}

func TestCommandPalette_FilterCaseInsensitive(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "Exit", Description: "Exit the app"},
		{Name: "HELP", Description: "Show help"},
	}
	p := NewCommandPalette(cmds)

	p.input.SetValue("ex")
	p.applyFilter()
	if len(p.filtered) != 1 || p.filtered[0].name != "Exit" {
		t.Errorf("expected 'Exit' for filter 'ex', got %+v", p.filtered)
	}

	p.input.SetValue("HELP")
	p.applyFilter()
	if len(p.filtered) != 1 || p.filtered[0].name != "HELP" {
		t.Errorf("expected 'HELP' for filter 'HELP', got %+v", p.filtered)
	}
}

func TestCommandPalette_FilterEmptyShowsAll(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "exit", Description: "Exit"},
		{Name: "help", Description: "Help"},
	}
	p := NewCommandPalette(cmds)

	p.input.SetValue("")
	p.applyFilter()

	if len(p.filtered) != len(cmds) {
		t.Errorf("expected %d items for empty filter, got %d", len(cmds), len(p.filtered))
	}
}

func TestCommandPalette_FilterNoMatches(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "exit", Description: "Exit"},
	}
	p := NewCommandPalette(cmds)

	p.input.SetValue("zzz")
	p.applyFilter()

	if len(p.filtered) != 0 {
		t.Errorf("expected 0 matches, got %d", len(p.filtered))
	}
}

func TestCommandPalette_NavigateDownAndWrap(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "a", Description: ""},
		{Name: "b", Description: ""},
		{Name: "c", Description: ""},
	}
	p := NewCommandPalette(cmds)
	p.cursor = 0

	p.moveDown()
	if p.cursor != 1 {
		t.Errorf("cursor = %d, want 1", p.cursor)
	}

	p.moveDown()
	p.moveDown()
	if p.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (wrap)", p.cursor)
	}
}

func TestCommandPalette_NavigateUpAndWrap(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "a", Description: ""},
		{Name: "b", Description: ""},
	}
	p := NewCommandPalette(cmds)
	p.cursor = 0

	p.moveUp()
	if p.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (wrap)", p.cursor)
	}

	p.moveUp()
	if p.cursor != 0 {
		t.Errorf("cursor = %d, want 0", p.cursor)
	}
}

func TestCommandPalette_KeyDown(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "a", Description: ""},
		{Name: "b", Description: ""},
	}
	p := NewCommandPalette(cmds)

	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyDown})
	up := updated.(CommandPalette)
	if up.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after KeyDown", up.cursor)
	}

	updated2, _ := up.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	up2 := updated2.(CommandPalette)
	if up2.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after j wrap", up2.cursor)
	}
}

func TestCommandPalette_KeyUp(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "a", Description: ""},
		{Name: "b", Description: ""},
	}
	p := NewCommandPalette(cmds)
	p.cursor = 1

	updated, _ := p.Update(tea.KeyMsg{Type: tea.KeyUp})
	up := updated.(CommandPalette)
	if up.cursor != 0 {
		t.Errorf("cursor = %d, want 0 after KeyUp", up.cursor)
	}

	updated2, _ := up.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	up2 := updated2.(CommandPalette)
	if up2.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after k wrap", up2.cursor)
	}
}

func TestCommandPalette_EnterSelection(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "exit", Description: "Exit"},
		{Name: "help", Description: "Help"},
	}
	p := NewCommandPalette(cmds)
	p.cursor = 1

	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = updated.(CommandPalette)

	if cmd == nil {
		t.Fatal("expected cmd after Enter, got nil")
	}
	msg := cmd()
	execMsg, ok := msg.(CommandPaletteExecuteMsg)
	if !ok {
		t.Fatalf("expected CommandPaletteExecuteMsg, got %T", msg)
	}
	if execMsg.Command != "help" {
		t.Errorf("command = %q, want 'help'", execMsg.Command)
	}
}

func TestCommandPalette_EnterEmptyFiltered(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "exit", Description: "Exit"},
	}
	p := NewCommandPalette(cmds)
	p.input.SetValue("zzz")
	p.applyFilter()

	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	_ = updated.(CommandPalette)
	if cmd != nil {
		t.Error("expected nil cmd when filtered is empty")
	}
}

func TestCommandPalette_EscDismiss(t *testing.T) {
	cmds := []CommandDesc{{Name: "exit", Description: "Exit"}}
	p := NewCommandPalette(cmds)

	updated, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEscape})
	_ = updated.(CommandPalette)

	if cmd == nil {
		t.Fatal("expected cmd after Esc, got nil")
	}
	msg := cmd()
	if _, ok := msg.(CommandPaletteDismissMsg); !ok {
		t.Fatalf("expected CommandPaletteDismissMsg, got %T", msg)
	}
}

func TestCommandPalette_ViewRenders(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "exit", Description: "Exit the app"},
	}
	p := NewCommandPalette(cmds)
	p.SetSize(80, 24)

	view := p.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
	if !strings.Contains(view, "exit") {
		t.Error("expected command name in view")
	}
	if !strings.Contains(view, "Exit the app") {
		t.Error("expected description in view")
	}
}

func TestCommandPalette_ViewNoMatches(t *testing.T) {
	cmds := []CommandDesc{
		{Name: "exit", Description: "Exit"},
	}
	p := NewCommandPalette(cmds)
	p.input.SetValue("zzz")
	p.applyFilter()
	p.SetSize(80, 24)

	view := p.View()
	if !strings.Contains(view, "No matching commands") {
		t.Error("expected 'No matching commands' in view")
	}
}

func TestCommandPalette_SetSize(t *testing.T) {
	p := NewCommandPalette(nil)
	p.SetSize(100, 50)
	if p.width != 100 || p.height != 50 {
		t.Errorf("size = %dx%d, want 100x50", p.width, p.height)
	}
}

func TestCommandPalette_Init(t *testing.T) {
	p := NewCommandPalette(nil)
	cmd := p.Init()
	if cmd == nil {
		t.Error("expected non-nil init cmd")
	}
}
